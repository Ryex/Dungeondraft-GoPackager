// Package gui
package gui

import (
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	xdialog "fyne.io/x/fyne/dialog"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/assets"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/credits"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"

	log "github.com/sirupsen/logrus"
)

// The names of the parts of the main window.
//
// The parts of the interface are named where they are created, so that they
// can be looked up by name later, e.g. by the specification tests.
const (
	nameMainColumn       = "mainColumn"
	nameHeader           = "header"
	nameThemeToggle      = "themeToggle"
	nameLanguageSelect   = "languageSelect"
	nameTitleText        = "titleText"
	nameSubTitleText     = "subTitleText"
	nameAboutButton      = "aboutButton"
	nameGithubButton     = "githubButton"
	nameInfoButton       = "infoButton"
	namePathSection      = "pathSection"
	namePathEntry        = "pathEntry"
	nameOpenPackButton   = "openPackButton"
	nameOpenFolderButton = "openFolderButton"
	nameContent          = "content"
)

type App struct {
	app    fyne.App
	window fyne.Window

	knownLanguages []string
	prefLocale     binding.String

	operatingPath binding.String

	mainContentContainer *fyne.Container
	mainContentLock      sync.Mutex
	defaultMainContent   fyne.CanvasObject

	pkg *ddpackage.Package

	// pkgLock guards the package of the app, so that it can be accessed
	// from the goroutine of the filesystem watcher, too.
	pkgLock sync.RWMutex

	disableButtons          binding.Bool
	mainDisableListener     binding.DataListener
	currentDisableListeners []binding.DataListener

	// forceDarkMode is the preference that forces the dark theme.
	forceDarkMode binding.Bool
	// forceDarkModeListener is the listener registered on the force dark
	// mode preference. It is replaced whenever the UI is rebuilt and
	// detached when the app is cleaned up.
	forceDarkModeListener binding.DataListener

	// ui resolves the names of the parts of the user interface to the
	// objects they belong to.
	ui *named.Cache

	tagSaveTimer  *time.Timer
	resSaveTimers map[string]*time.Timer

	// packageWatcher watches the unpacked folder of the package for changes.
	packageWatcher *packageWatcher

	// packageWatcherDebounce is the time the watcher waits for further
	// events, before it reports the changed paths. It is zero in the app,
	// which makes the watcher use its default debounce time. The tests
	// lower it, to keep them fast.
	packageWatcherDebounce time.Duration

	// packageWatcherDisabled stops the filesystem watcher from being started.
	// Used by tests, which drive the package loading directly.
	packageWatcherDisabled bool

	// packageUpdateLock serializes the updates of the package that the
	// filesystem watcher triggers and the replacement of the package, so
	// that a package is not closed while it is updated. It guards
	// packageUpdateCount.
	//
	// An update must not wait for the Fyne goroutine while it holds the
	// lock: the Fyne goroutine can wait for the lock itself, when the
	// package is replaced (see replacePackage).
	packageUpdateLock sync.Mutex
	// packageUpdateCount counts the updates of the package that the
	// filesystem watcher triggered.
	packageUpdateCount int

	// loadUnpacked loads an unpacked package folder. It is a field so that
	// tests can control the timing of the load.
	loadUnpacked func(path string, progress func(p float64, curPath string)) (unpackedLoadResult, error)

	// loadPacked loads a packed resource pack file. It is a field so that
	// tests can control the timing of the load.
	loadPacked func(path string, progress func(p float64, curRes string)) (*ddpackage.Package, error)

	// packageUpdateHook is called while an update of the package runs in
	// the background, with the dialog of the update showing. It is a field
	// so that tests can look at the interface while the update runs.
	packageUpdateHook func(dlg *packageUpdateDialog)

	// bgWG tracks the background work that touches the interface: the
	// loads of a package and the computations of the md5 hashes.
	//
	// The interface must not be touched while the background work is
	// running, so the tests wait for it.
	bgWG sync.WaitGroup

	// prefLocaleListener is the listener registered on the locale preference.
	// It is removed and replaced whenever the UI is rebuilt and detached
	// when the app is cleaned up.
	prefLocaleListener binding.DataListener

	pkgUpdateCounter int
	packageUpdated   binding.Int

	lastSelectedTab string
}

//go:embed translation
var translations embed.FS

// translationsOnce guards loading the embedded translation catalog so that
// repeated calls to setup (e.g. from tests) do not register the same
// translations more than once in the global i18n bundle.
var translationsOnce sync.Once

func NewApp() *App {
	app := &App{
		operatingPath:  binding.NewString(),
		disableButtons: binding.NewBool(),
		resSaveTimers:  make(map[string]*time.Timer),
	}
	app.packageUpdated = binding.BindInt(&app.pkgUpdateCounter)
	app.loadUnpacked = app.loadUnpackedSync
	app.loadPacked = app.loadPackSync
	return app
}

// NewAppWithFyneApp creates an App that uses the given Fyne application
// instead of creating a desktop one. This is used by tests to inject a
// software rendered application from fyne.io/fyne/v2/test.
func NewAppWithFyneApp(fyneApp fyne.App) *App {
	a := NewApp()
	a.app = fyneApp
	return a
}

func (a *App) Main() {
	a.setup()
	a.app.Run()
	a.clean()
}

// setup prepares the application window and user interface without entering
// the event loop. It is separated from Main so that tests can build the UI
// without blocking on app.Run().
func (a *App) setup() {
	if a.app == nil {
		a.app = app.NewWithID("io.github.ryex.dungeondraft-gopackager")
	}
	locale := lang.SystemLocale()

	log.Infof("system local %s : %s", locale.LanguageString(), locale.String())
	translationsOnce.Do(func() {
		translationErr := lang.AddTranslationsFS(translations, "translation")
		if translationErr != nil {
			log.WithError(translationErr).Error("Failed to load translations")
		}
	})

	localeString := lang.ClosestSupportedLocale([]string{locale.LanguageString()}).LanguageString()
	prefLocaleBinding := binding.BindPreferenceString("locale", a.app.Preferences())
	prefLocale, err := prefLocaleBinding.Get()
	if err != nil || prefLocale == "" {
		prefLocaleBinding.Set(localeString)
		prefLocale = localeString
	}
	a.prefLocale = prefLocaleBinding
	lang.SetupLang(prefLocale)

	a.app.Settings().SetTheme(BetterDefaultTheme())
	a.window = a.app.NewWindow(lang.X("window.title", "Dungeondraft-GoPackager"))
	a.window.SetIcon(assets.Icon)
	a.window.Resize(fyne.NewSize(1200, 800))

	a.setupWindow()

	// The preference binding is shared globally by key, so the previous
	// listener has to be detached before a new one is attached.
	if a.prefLocaleListener != nil {
		a.prefLocale.RemoveListener(a.prefLocaleListener)
	}
	a.prefLocaleListener = binding.NewDataListener(func() {
		newLocale, err := prefLocaleBinding.Get()
		// ignore fetch error and if the local hasn't actually changed (first run)
		if err != nil || newLocale == prefLocale {
			return
		}
		prefLocale = newLocale
		lang.SetupLang(prefLocale)
		a.setupWindow()
	})
	a.prefLocale.AddListener(a.prefLocaleListener)
}

func (a *App) setupWindow() {
	a.window.SetTitle(lang.X("window.title", "Dungeondraft-GoPackager"))
	a.window.SetOnDropped(func(_ fyne.Position, u []fyne.URI) {
		if len(u) > 1 {
			dialog.ShowInformation(
				lang.X("tooManyDrops.title", "Too Many Dropped Items"),
				lang.X("tooManyDrop.message", "DungeonDraft GoPackager does not support dropping multiple items"),
				a.window,
			)
		} else if len(u) == 1 {
			path := filepath.Clean(u[0].Path())
			dlg := dialog.NewConfirm(
				lang.X("dropOpenDialog.title", "Confirm Open Pack"),
				lang.X(
					"dropOpenDialog.message",
					"Do you want to open '{{.Path}}' ?",
					map[string]string{
						"Path": path,
					},
				),
				func(confirmed bool) {
					if confirmed {
						a.operatingPath.Set(path)
					}
				},
				a.window,
			)
			dlg.Show()
		}
	})
	a.buildMainUI()
	a.setupPathHandler()
	a.window.Show()
}

// resetPkg stops the watcher of the package of the app, drops it and closes
// it.
func (a *App) resetPkg() {
	a.teardownPackageWatcher()
	a.replacePackage(nil)
}

// replacePackage sets the package of the app and closes the one it replaces.
//
// It waits for an update of the package that runs in the background, so
// that a package is not closed while it is updated.
func (a *App) replacePackage(pkg *ddpackage.Package) {
	a.packageUpdateLock.Lock()
	defer a.packageUpdateLock.Unlock()

	old := a.currentPackage()
	a.setCurrentPackage(pkg)
	if old != nil && old != pkg {
		old.Close()
	}
}

// currentPackage returns the package of the app.
//
// It is safe to call it from the goroutine of the filesystem watcher, which
// may run while another package is loaded.
func (a *App) currentPackage() *ddpackage.Package {
	a.pkgLock.RLock()
	defer a.pkgLock.RUnlock()
	return a.pkg
}

// setCurrentPackage sets the package of the app.
func (a *App) setCurrentPackage(pkg *ddpackage.Package) {
	a.pkgLock.Lock()
	defer a.pkgLock.Unlock()
	a.pkg = pkg
}

func (a *App) clean() {
	a.resetPkg()
	if a.tagSaveTimer != nil {
		a.tagSaveTimer.Stop()
		a.tagSaveTimer = nil
	}
	for _, timer := range a.resSaveTimers {
		timer.Stop()
	}
	a.resSaveTimers = map[string]*time.Timer{}
	if a.prefLocaleListener != nil && a.prefLocale != nil {
		a.prefLocale.RemoveListener(a.prefLocaleListener)
		a.prefLocaleListener = nil
	}
	if a.forceDarkModeListener != nil && a.forceDarkMode != nil {
		a.forceDarkMode.RemoveListener(a.forceDarkModeListener)
		a.forceDarkModeListener = nil
	}
	if a.window != nil {
		// release the resources held by the tool tip layer of the window
		fynetooltip.DestroyWindowToolTipLayer(a.window.Canvas())
	}
	fmt.Println("Exited")
}

func languageChoices() ([]string, map[string]string, map[string]string) {
	choices := structures.NewSet[string]()
	mapping := map[string]string{}
	reversed := map[string]string{}
	for _, tag := range lang.Translated {
		locale := lang.LocaleFromTag(tag)
		flag := lang.UnicodeFlagFromLocale(locale)
		choice := flag
		if choice != "" {
			choice += " "
		}
		choice += locale.LanguageString()
		mapping[choice] = locale.LanguageString()
		reversed[locale.LanguageString()] = choice
		choices.Add(choice)
	}
	return choices.AsSlice(), mapping, reversed
}

func (a *App) buildMainUI() {
	siteURL, _ := url.Parse("https://ryex.github.io/Dungeondraft-GoPackager/")
	githubURL, _ := url.Parse("https://github.com/Ryex/Dungeondraft-GoPackager")

	// the layout is rebuilt from scratch, so the cache of the named parts
	// has to be replaced as well
	a.ui = named.NewCache(func() fyne.CanvasObject {
		return a.window.Content()
	})

	darkThemeToggleLbl := widget.NewLabel(lang.X("app.darkTheme.toggle.label", "Dark Mode"))

	langChoices, choiceLangMap, langChoiceMap := languageChoices()
	langChoiceBinding := bindings.NewReversibleMapping(
		a.prefLocale,
		func(lang string) (string, error) {
			return langChoiceMap[lang], nil
		},
		func(choice string) (string, error) {
			return choiceLangMap[choice], nil
		},
	)
	langSelect := widget.NewSelectWithData(langChoices, langChoiceBinding)

	forceDarkMode := binding.BindPreferenceBool("forceDarkMode", a.app.Preferences())
	darkThemeToggle := widgets.NewToggleWithData(forceDarkMode)

	headerBg := widgets.NewThemedRect(theme.ColorNameHeaderBackground, 4)

	infoIcon := widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		crdWin := credits.CreditsWindow(a.app, fyne.NewSize(800, 400))
		crdWin.SetIcon(assets.Icon)
		crdWin.SetTitle(lang.X("creditsWindow.title", "Credits"))
		crdWin.Show()
	})

	ghIcon := widget.NewButtonWithIcon("", assets.GithubWhite, func() {
		a.app.OpenURL(githubURL)
	})

	titleText := widgets.NewThemedText(
		lang.X("greeting.title", "Dungeondraft-GoPackager"),
		theme.ColorNameForeground,
		18,
	)
	titleText.Text.Alignment = fyne.TextAlignCenter

	subTitleText := widgets.NewThemedText(
		lang.X("greeting.sub-title", "Package, edit, and prepare Dungeondraft resource packs"),
		theme.ColorNameForeground,
		14,
	)
	subTitleText.Text.Alignment = fyne.TextAlignCenter

	aboutBtn := widget.NewButtonWithIcon("", assets.Icon, func() {
		aboutDlg := xdialog.NewAbout(
			lang.X(
				"greeting.sub-title",
				"Package, edit, and prepare Dungeondraft resource packs",
			), []*widget.Hyperlink{
				widget.NewHyperlink(
					lang.X("website.link", "Website"),
					siteURL,
				),
			}, a.app, a.window,
		)
		aboutDlg.Resize(
			fyne.NewSize(
				fyne.Min(a.window.Canvas().Size().Width, 640),
				fyne.Min(a.window.Canvas().Size().Height, 480),
			),
		)
		aboutDlg.Show()
	})

	welcome := container.NewPadded(container.NewStack(
		headerBg,
		container.NewPadded(container.NewVBox(
			named.New(nameTitleText, titleText),
			named.New(nameSubTitleText, subTitleText),
		)),
		container.NewVBox(
			layout.NewSpacer(),
			container.NewPadded(container.NewHBox(
				darkThemeToggleLbl,
				named.New(nameThemeToggle, darkThemeToggle),
				named.New(nameLanguageSelect, langSelect),
				layout.NewSpacer(),
				named.New(nameAboutButton, aboutBtn),
				named.New(nameGithubButton, ghIcon),
				named.New(nameInfoButton, infoIcon),
			)),
			layout.NewSpacer(),
		),
	))

	pathInput := widget.NewEntryWithData(a.operatingPath)
	pathInput.SetPlaceHolder(lang.X("pathInput.placeholder", "Path to .dungeondraft_pack or folder"))
	var inputTimer *time.Timer
	pathInput.OnChanged = func(path string) {
		if inputTimer != nil {
			inputTimer.Stop()
		}
		inputTimer = time.AfterFunc(500*time.Millisecond, func() {
			inputTimer = nil
			path = strings.TrimSpace(path)
			_, err := os.Stat(path)
			if err == nil {
				a.app.Preferences().SetString("lastPack.path", path)
				a.operatingPath.Set(path)
			}
		})
	}
	pathInput.Validator = func(path string) error {
		_, err := os.Stat(strings.TrimSpace(path))
		return err
	}

	packBtn := widget.NewButtonWithIcon(lang.X("packBtn.text", "Open pack"), theme.FileIcon(), func() {
		dlg := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err == nil && uc != nil {
				log.Infof("open path %s", uc.URI().Path())
				a.app.Preferences().SetString("lastPack.path", uc.URI().Path())
				a.operatingPath.Set(uc.URI().Path())
			}
		}, a.window)
		dlg.SetFilter(storage.NewExtensionFileFilter([]string{".dungeondraft_pack"}))
		lastOpen := a.app.Preferences().String("lastPack.path")
		lastOpenURI := storage.NewFileURI(filepath.Dir(lastOpen))
		listableLastOpen, err := storage.ListerForURI(lastOpenURI)
		if err == nil {
			dlg.SetLocation(listableLastOpen)
		}
		dlg.Show()
		dlg.Resize(
			fyne.NewSize(
				fyne.Min(a.window.Canvas().Size().Width, 740),
				fyne.Min(a.window.Canvas().Size().Height, 580),
			),
		)
	})

	folderBtn := widget.NewButtonWithIcon(lang.X("folderBtn.text", "Open folder"), theme.FolderOpenIcon(), func() {
		dlg := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err == nil && lu != nil {
				log.Infof("open path %s", lu.Path())
				a.app.Preferences().SetString("lastFolder.path", lu.Path())
				a.operatingPath.Set(lu.Path())
			}
		}, a.window)
		lastOpen := a.app.Preferences().String("lastFolder.path")
		lastOpenURI := storage.NewFileURI(lastOpen)
		listableLastOpen, err := storage.ListerForURI(lastOpenURI)
		if err == nil {
			dlg.SetLocation(listableLastOpen)
		}
		dlg.Show()
		dlg.Resize(
			fyne.NewSize(
				fyne.Min(a.window.Canvas().Size().Width, 740),
				fyne.Min(a.window.Canvas().Size().Height, 580),
			),
		)
	})

	// the UI is rebuilt on every locale change, so the listener of the
	// previous build has to be detached before a new one is registered.
	// The binding belongs to the app, so it dies with it.
	if a.mainDisableListener != nil {
		a.disableButtons.RemoveListener(a.mainDisableListener)
		a.mainDisableListener = nil
	}
	a.mainDisableListener = bindings.Listen(a.disableButtons, func(disable bool) {
		if disable {
			packBtn.Disable()
			folderBtn.Disable()
			langSelect.Disable()
			darkThemeToggle.Disable()
		} else {
			packBtn.Enable()
			folderBtn.Enable()
			langSelect.Enable()
			darkThemeToggle.Enable()
		}
	})

	inputContainer := container.New(
		layouts.NewLeftExpandHBoxLayout(),
		named.New(namePathEntry, pathInput),
		layout.NewSpacer(),
		named.New(nameOpenPackButton, packBtn),
		layout.NewSpacer(),
		named.New(nameOpenFolderButton, folderBtn),
	)

	defaultMainText := multilineCanvasText(
		lang.X(
			"main.defaultText",
			"Enter a path to get started.\n"+
				"You can also drag and drop a resource pack file or unpacked folder.",
		),
		28,
		fyne.TextStyle{Italic: true},
		fyne.TextAlignCenter,
		theme.ColorNameForeground,
	)

	a.defaultMainContent = container.NewBorder(
		nil, nil, nil, nil,
		container.NewCenter(defaultMainText),
	)

	a.mainContentContainer = container.NewStack()
	a.setMainContent(a.defaultMainContent)

	mainColumn := container.New(
		layouts.NewBottomExpandVBoxLayout(),
		named.New(nameHeader, welcome),
		named.New(namePathSection, inputContainer),
		widget.NewSeparator(),
		container.NewPadded(
			named.New(nameContent, a.mainContentContainer),
		),
	)

	content := container.NewPadded(named.New(nameMainColumn, mainColumn))

	a.window.SetContent(fynetooltip.AddWindowToolTipLayer(content, a.window.Canvas()))

	refreshTheme := func() {
		// headerBg.Refresh()
		infoIcon.Icon = theme.InfoIcon()
		// infoIcon.Refresh()
		ghIcon.Icon = func() *fyne.StaticResource {
			forceDark, err := forceDarkMode.Get()
			if err != nil {
				forceDark = false
			}
			if forceDark || a.app.Settings().ThemeVariant() == theme.VariantDark {
				return assets.GithubWhite
			} else {
				return assets.Github
			}
		}()
		// ghIcon.Refresh()
		// welcome.Refresh()
		content.Refresh()
	}

	// the UI is rebuilt on every locale change, so the listener of the
	// previous build has to be detached before a new one is registered
	if a.forceDarkModeListener != nil && a.forceDarkMode != nil {
		a.forceDarkMode.RemoveListener(a.forceDarkModeListener)
		a.forceDarkModeListener = nil
	}
	a.forceDarkMode = forceDarkMode
	a.forceDarkModeListener = bindings.Listen(forceDarkMode, func(force bool) {
		log.Infof("dark mode toggled: %t", force)
		if force {
			a.app.Settings().SetTheme(&forcedVariantTheme{Theme: BetterDefaultTheme()})
		} else {
			a.app.Settings().SetTheme(BetterDefaultTheme())
		}
		refreshTheme()
	})
}

func (a *App) setMainContent(o fyne.CanvasObject, disableButtonsListeners ...func(bool)) {
	fyne.Do(func() {
		a.mainContentLock.Lock()
		defer a.mainContentLock.Unlock()

		if len(a.currentDisableListeners) > 0 {
			log.Infof("removing %v listeners", len(a.currentDisableListeners))
		}
		for _, listener := range a.currentDisableListeners {
			a.disableButtons.RemoveListener(listener)
		}
		a.currentDisableListeners = nil

		if len(disableButtonsListeners) > 0 {
			log.Infof("adding %v listeners", len(disableButtonsListeners))
		}
		for _, listenerFunc := range disableButtonsListeners {
			listener := bindings.Listen(a.disableButtons, listenerFunc)
			a.currentDisableListeners = append(a.currentDisableListeners, listener)
		}

		log.Info("clearing main content")
		a.mainContentContainer.RemoveAll()

		log.Info("setting main content")
		a.mainContentContainer.Add(o)

		// log.Info("refreshing main content")
		// winSize := a.window.Canvas().Size()
		// a.mainContentContainer.Refresh()
		// a.window.Resize(winSize)

		// the parts of the interface that are shown now are new objects, so
		// the cache of the named parts has to forget the old ones
		if a.ui != nil {
			a.ui.Reset()
		}
	})
}

func (a *App) setupPathHandler() {
	bindings.Listen(a.operatingPath, func(path string) {
		if path == "" {
			return
		}

		info, err := os.Stat(path)
		if err != nil {
			log.WithError(err).Errorf("can not stat path %s", path)
			a.setErrContent(
				lang.X(
					"err.badPath",
					"Can not open \"{{.Path}}\"",
					map[string]any{
						"Path": path,
					},
				),
				err,
			)
			return
		}

		if info.IsDir() {
			a.loadUnpackedPath(path)
		} else {
			a.loadPack(path)
		}
	})
}

func (a *App) setErrContent(msg string, errs ...error) {
	msgText := widgets.NewThemedText(msg, theme.ColorNameForeground, 16)
	msgText.Text.Alignment = fyne.TextAlignCenter

	errContainer := container.NewVBox()
	for i, err := range errs {
		errText := multilineCanvasText(
			fmt.Sprintf("%d) ", i+1)+err.Error(),
			14,
			fyne.TextStyle{Italic: true},
			fyne.TextAlignLeading,
			theme.ColorNameError,
		)
		errContainer.Add(errText)
	}

	msgContent := container.NewVBox(
		layout.NewSpacer(),
		msgText,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVScroll(errContainer),
			layout.NewSpacer(),
		),
		layout.NewSpacer(),
	)

	a.setMainContent(msgContent)
}

func multilineCanvasText(
	text string,
	size float32,
	style fyne.TextStyle,
	align fyne.TextAlign,
	color fyne.ThemeColorName,
) fyne.CanvasObject {
	lines := strings.Split(text, "\n")
	content := container.NewVBox(slices.Collect(utils.Map(
		slices.Values(lines),
		func(line string) fyne.CanvasObject {
			text := widgets.NewThemedText(line, color)
			text.Text.TextSize = size
			text.Text.TextStyle = style
			text.Text.Alignment = align
			return text
		},
	))...)
	return content
}

func (a *App) setWaitContent(msg string) (binding.Float, binding.String) {
	activity := widget.NewActivity()
	activity.Start()
	msgText := widgets.NewThemedText(msg, theme.ColorNameForeground)
	msgText.Text.TextSize = 16
	msgText.Text.Alignment = fyne.TextAlignCenter
	activityText := widgets.NewThemedText("", theme.ColorNameForeground)
	activityText.Text.TextSize = 12
	activityText.Text.Alignment = fyne.TextAlignCenter
	activityStr := binding.NewString()
	bindings.Listen(activityStr, func(str string) {
		activityText.Text.Text = str
		activityText.Refresh()
	})
	progressBar := widget.NewProgressBar()
	activityProgress := binding.NewFloat()
	progressBar.Bind(activityProgress)

	activityContent := container.NewVBox(
		layout.NewSpacer(),
		activity,
		msgText,
		activityText,
		container.NewPadded(progressBar),
		layout.NewSpacer(),
	)
	a.setMainContent(activityContent)
	return activityProgress, activityStr
}

// showErrorDialog shows the error in a dialog.
//
// It can be called from the background work that reports the error: the
// dialog is shown on the Fyne goroutine.
func (a *App) showErrorDialog(err error) {
	fyne.Do(func() {
		errDlg := dialog.NewError(err, a.window)
		errDlg.Show()
	})
}
