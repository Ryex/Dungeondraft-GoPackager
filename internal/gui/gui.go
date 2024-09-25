package gui

import (
	"embed"
	"fmt"
	"os"
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

	pkgFile *os.File
	pkg     *ddpackage.Package

	disableButtons          binding.Bool
	mainDisableListener     binding.DataListener
	currentDisableListeners []binding.DataListener
}

//go:embed translation
var translations embed.FS

func NewApp() *App {
	return &App{
		operatingPath:  binding.NewString(),
		disableButtons: binding.NewBool(),
	}
}

func (a *App) Main() {
	a.app = app.NewWithID("io.github.ryex.dungondraft-gopackager")
	translationErr := lang.AddTranslationsFS(translations, "translation")
	if translationErr != nil {
		log.WithError(translationErr).Error("Failed to load translations")
	}
	a.window = a.app.NewWindow(lang.X("window.title", "Dungeondraft-GoPackager"))
	a.window.Resize(fyne.NewSize(800, 600))

	a.buildMainUI()
	a.setupPathHandler()

	a.window.Show()
	a.app.Run()
	a.clean()
}

func (a *App) clean() {
	if a.pkgFile != nil {
		a.pkgFile.Close()
	}
	fmt.Println("Exited")
}

func (a *App) buildMainUI() {
	welcomeText := canvas.NewText(
		lang.X("greeting", "Dungeondraft-GoPackager - Package and edit Dungeondraft resource packs"),
		theme.Color(theme.ColorNameForeground),
	)
	welcomeText.TextSize = 18
	welcome := container.NewCenter(welcomeText)

	pathInput := widget.NewEntryWithData(a.operatingPath)
	pathInput.SetPlaceHolder(lang.X("pathInput.placeholder", "Path to .dungeondraft_pack or folder"))
	var inputTimer *time.Timer
	pathInput.OnChanged = func(path string) {
		if inputTimer != nil {
			inputTimer.Stop()
		}
		inputTimer = time.AfterFunc(500*time.Millisecond, func() {
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
		dlg.Show()
	})

	folderBtn := widget.NewButtonWithIcon(lang.X("folderBtn.text", "Open folder"), theme.FolderOpenIcon(), func() {
		dlg := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err == nil && lu != nil {
				log.Infof("open path %s", lu.Path())
				a.operatingPath.Set(lu.Path())
			}
		}, a.window)
		dlg.Resize(
			fyne.NewSize(
				fyne.Min(a.window.Canvas().Size().Width, 740),
				fyne.Min(a.window.Canvas().Size().Height, 580),
			),
		)
		dlg.Show()
	})

	a.mainDisableListener = binding.NewDataListener(func() {
		disable, _ := a.disableButtons.Get()
		if disable {
			packBtn.Disable()
			folderBtn.Disable()
		} else {
			packBtn.Enable()
			folderBtn.Enable()
		}
	})
	a.disableButtons.AddListener(a.mainDisableListener)

	inputContainer := container.New(
		NewLeftExpandHBoxLayout(),
		pathInput,
		layout.NewSpacer(),
		packBtn,
		layout.NewSpacer(),
		folderBtn,
	)

	defaultMainText := canvas.NewText(
		lang.X("main.defaultText", "Enter a path to get started."),
		theme.Color(theme.ColorNameForeground),
	)
	defaultMainText.Alignment = fyne.TextAlignCenter
	defaultMainText.TextSize = 28
	defaultMainText.TextStyle = fyne.TextStyle{Italic: true}

	a.defaultMainContent = container.NewBorder(
		nil, nil, nil, nil,
		container.NewCenter(defaultMainText),
	)

	a.mainContentContainer = container.NewStack()
	a.setMainContent(a.defaultMainContent)

	content := container.NewPadded(
		container.New(
			NewBottomExpandVBoxLayout(),
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

func (a *App) setMainContent(o fyne.CanvasObject, disableButtonsListeners ...binding.DataListener) {
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
		for _, listener := range disableButtonsListeners {
			a.currentDisableListeners = append(a.currentDisableListeners, listener)
			a.disableButtons.AddListener(listener)
		}

		log.Info("clearing main content")
		a.mainContentContainer.RemoveAll()

		log.Info("setting main content")
		a.mainContentContainer.Add(o)

		log.Info("refreshing main content")
		a.mainContentContainer.Refresh()
	}()
}

func (a *App) setupPathHandler() {
	a.operatingPath.AddListener(binding.NewDataListener(func() {
		path, err := a.operatingPath.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound path value")
			return
		}
		if path == "" {
			return
		}

		info, err := os.Stat(path)
		if err != nil {
			log.WithError(err).Errorf("can not stat path %s", path)
			a.setErrContent(err, lang.X("err.badPath", "Can not open \"{{.Path}}\"", map[string]any{"Path": path}))
			return
		}

		if info.IsDir() {
			a.loadUnpackedPath(path)
		} else {
			a.loadPack(path)
		}
	}))
}

func (a *App) setErrContent(err error, msg string) {
	msgText := canvas.NewText(msg, theme.Color(theme.ColorNameForeground))
	msgText.TextSize = 16
	msgText.Alignment = fyne.TextAlignCenter

	errText := canvas.NewText(err.Error(), theme.Color(theme.ColorNameForeground))
	errText.TextSize = 14
	errText.TextStyle = fyne.TextStyle{Italic: true}
	errText.Alignment = fyne.TextAlignCenter

	msgContent := container.NewVBox(
		layout.NewSpacer(),
		msgText,
		errText,
		layout.NewSpacer(),
	)

	a.setMainContent(msgContent)
}

func (a *App) setWaitContent(msg string) binding.String {
	activity := widget.NewActivity()
	activity.Start()
	msgText := canvas.NewText(msg, theme.Color(theme.ColorNameForeground))
	msgText.TextSize = 16
	msgText.Alignment = fyne.TextAlignCenter
	activityText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	activityText.TextSize = 12
	activityText.Alignment = fyne.TextAlignCenter
	activityStr := binding.NewString()
	activityStr.AddListener(binding.NewDataListener(func() {
		str, _ := activityStr.Get()
		activityText.Text = str
		activityText.Refresh()
	}))

	activityContent := container.NewVBox(
		layout.NewSpacer(),
		activity,
		msgText,
		activityText,
		layout.NewSpacer(),
	)
	a.setMainContent(activityContent)
	return activityStr
}

func (a *App) showErrorDialog(err error) {
	errDlg := dialog.NewError(err, a.window)
	errDlg.Show()
}
