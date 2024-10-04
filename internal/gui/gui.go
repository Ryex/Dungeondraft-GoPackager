package gui

import (
	"embed"
	"fmt"
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xdialog "fyne.io/x/fyne/dialog"
	"github.com/fsnotify/fsnotify"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/assets"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/credits"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	log "github.com/sirupsen/logrus"
)

type App struct {
	app    fyne.App
	window fyne.Window

	operatingPath binding.String

	mainContentContainer *fyne.Container
	mainContentLock      sync.Mutex
	defaultMainContent   fyne.CanvasObject

	pkg *ddpackage.Package

	disableButtons          binding.Bool
	mainDisableListener     binding.DataListener
	currentDisableListeners []binding.DataListener

	tagSaveTimer  *time.Timer
	resSaveTimers map[string]*time.Timer

	packageWatcher *fsnotify.Watcher

	packageWatcherIgnoreThumbnails bool

	pkgUpdateCounter int
	packageUpdated   binding.Int
}

//go:embed translation
var translations embed.FS

func NewApp() *App {
	app := &App{
		operatingPath:  binding.NewString(),
		disableButtons: binding.NewBool(),
		resSaveTimers:  make(map[string]*time.Timer),
	}
	app.packageUpdated = binding.BindInt(&app.pkgUpdateCounter)
	return app
}

func (a *App) Main() {
	a.app = app.NewWithID("io.github.ryex.dungondraft-gopackager")
	local := lang.SystemLocale()
	log.Infof("system local %s : %s", local.LanguageString(), local.String())
	translationErr := lang.AddTranslationsFS(translations, "translation")
	if translationErr != nil {
		log.WithError(translationErr).Error("Failed to load translations")
	}
	a.window = a.app.NewWindow(lang.X("window.title", "Dungeondraft-GoPackager"))
	a.window.SetIcon(assets.Icon)
	a.window.Resize(fyne.NewSize(1200, 800))

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
	a.app.Run()
	a.clean()
}

func (a *App) resetPkg() {
	a.teardownPackageWatcher()
	if a.pkg != nil {
		a.pkg.Close()
	}
	a.pkg = nil
}

func (a *App) clean() {
	a.resetPkg()
	fmt.Println("Exited")
}

func (a *App) buildMainUI() {
	siteURL, _ := url.Parse("https://ryex.github.io/Dungeondraft-GoPackager/")
	githubURL, _ := url.Parse("https://github.com/Ryex/Dungeondraft-GoPackager")
	welcome := container.NewPadded(container.NewStack(
		&canvas.Rectangle{
			FillColor:    theme.Color(theme.ColorNameHeaderBackground),
			CornerRadius: 4,
		},
		container.NewPadded(container.NewVBox(
			&canvas.Text{
				Text:      lang.X("greeting.title", "Dungeondraft-GoPackager"),
				Color:     theme.Color(theme.ColorNameForeground),
				TextSize:  18,
				Alignment: fyne.TextAlignCenter,
			},
			&canvas.Text{
				Text:      lang.X("greeting.sub-title", "Package, edit, and prepare Dungeondraft resource packs"),
				Color:     theme.Color(theme.ColorNameForeground),
				TextSize:  14,
				Alignment: fyne.TextAlignCenter,
			},
		)),
		container.NewVBox(
			layout.NewSpacer(),
			container.NewHBox(
				layout.NewSpacer(),
				widget.NewButtonWithIcon("", assets.Icon, func() {
					aboutDlg := xdialog.NewAbout(
						lang.X(
							"greeting.sub-title",
							"Package, edit, and prepare Dungeondraft resource packs",
						), []*widget.Hyperlink{
							widget.NewHyperlink(
								lang.X("website.link", "Website"),
								siteURL,
							),
						}, a.app, a.window)
					aboutDlg.Resize(
						fyne.NewSize(
							fyne.Min(a.window.Canvas().Size().Width, 640),
							fyne.Min(a.window.Canvas().Size().Height, 480),
						),
					)
					aboutDlg.Show()
				}),
				widget.NewButtonWithIcon("", assets.GithubWhite, func() {
					a.app.OpenURL(githubURL)
				}),
				widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
					crdWin := credits.CreditsWindow(a.app, fyne.NewSize(800, 400))
					crdWin.SetIcon(assets.Icon)
					crdWin.SetTitle(lang.X("creditsWindow.title", "Credits"))
					crdWin.Show()
				}),
			),
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
			_, err := os.Stat(path)
			if err == nil {
				a.operatingPath.Set(path)
			}
		})
	}
	pathInput.Validator = func(path string) error {
		_, err := os.Stat(path)
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
		dlg.Resize(
			fyne.NewSize(
				fyne.Min(a.window.Canvas().Size().Width, 740),
				fyne.Min(a.window.Canvas().Size().Height, 580),
			),
		)
		lastOpen := a.app.Preferences().String("lastPack.path")
		lastOpenURI := storage.NewFileURI(filepath.Dir(lastOpen))
		lisableLastOpen, err := storage.ListerForURI(lastOpenURI)
		if err == nil {
			dlg.SetLocation(lisableLastOpen)
		}
		dlg.Show()
	})

	folderBtn := widget.NewButtonWithIcon(lang.X("folderBtn.text", "Open folder"), theme.FolderOpenIcon(), func() {
		dlg := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err == nil && lu != nil {
				log.Infof("open path %s", lu.Path())
				a.app.Preferences().SetString("lastFolder.path", lu.Path())
				a.operatingPath.Set(lu.Path())
			}
		}, a.window)
		dlg.Resize(
			fyne.NewSize(
				fyne.Min(a.window.Canvas().Size().Width, 740),
				fyne.Min(a.window.Canvas().Size().Height, 580),
			),
		)
		lastOpen := a.app.Preferences().String("lastFolder.path")
		lastOpenURI := storage.NewFileURI(lastOpen)
		lisableLastOpen, err := storage.ListerForURI(lastOpenURI)
		if err == nil {
			dlg.SetLocation(lisableLastOpen)
		}
		dlg.Show()
	})

	a.mainDisableListener = bindings.Listen(a.disableButtons, func(disable bool) {
		if disable {
			packBtn.Disable()
			folderBtn.Disable()
		} else {
			packBtn.Enable()
			folderBtn.Enable()
		}
	})

	inputContainer := container.New(
		layouts.NewLeftExpandHBoxLayout(),
		pathInput,
		layout.NewSpacer(),
		packBtn,
		layout.NewSpacer(),
		folderBtn,
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
		theme.Color(theme.ColorNameForeground),
	)

	a.defaultMainContent = container.NewBorder(
		nil, nil, nil, nil,
		container.NewCenter(defaultMainText),
	)

	a.mainContentContainer = container.NewStack()
	a.setMainContent(a.defaultMainContent)

	content := container.NewPadded(
		container.New(
			layouts.NewBottomExpandVBoxLayout(),
			welcome,
			inputContainer,
			widget.NewSeparator(),
			container.NewPadded(
				a.mainContentContainer,
			),
		),
	)
	a.window.SetContent(content)
}

func (a *App) setMainContent(o fyne.CanvasObject, disableButtonsListeners ...func(bool)) {
	go func() {
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
	}()
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
	msgText := canvas.NewText(msg, theme.Color(theme.ColorNameForeground))
	msgText.TextSize = 16
	msgText.Alignment = fyne.TextAlignCenter

	errContainer := container.NewVBox()
	for i, err := range errs {
		errText := multilineCanvasText(
			fmt.Sprintf("%d) ", i+1)+err.Error(),
			14,
			fyne.TextStyle{Italic: true},
			fyne.TextAlignLeading,
			theme.Color(theme.ColorNameError),
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
	color color.Color,
) fyne.CanvasObject {
	lines := strings.Split(text, "\n")
	content := container.NewVBox(utils.Map(
		lines,
		func(line string) fyne.CanvasObject {
			text := canvas.NewText(line, color)
			text.TextSize = size
			text.TextStyle = style
			text.Alignment = align
			return text
		},
	)...)
	return content
}

func (a *App) setWaitContent(msg string) (binding.Float, binding.String) {
	activity := widget.NewActivity()
	activity.Start()
	msgText := canvas.NewText(msg, theme.Color(theme.ColorNameForeground))
	msgText.TextSize = 16
	msgText.Alignment = fyne.TextAlignCenter
	activityText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	activityText.TextSize = 12
	activityText.Alignment = fyne.TextAlignCenter
	activityStr := binding.NewString()
	bindings.Listen(activityStr, func(str string) {
		activityText.Text = str
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

func (a *App) showErrorDialog(err error) {
	errDlg := dialog.NewError(err, a.window)
	errDlg.Show()
}
