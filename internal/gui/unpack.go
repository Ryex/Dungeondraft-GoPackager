package gui

import (
	"errors"
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/custom_layout"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"

	log "github.com/sirupsen/logrus"
)

func (a *App) loadPack(path string) {
	a.setWaitContent(lang.X("unpack.wait", "Loading {{.Path}} ...", map[string]any{"Path": path}))
	a.disableButtons.Set(true)

	if a.pkg != nil {
		a.pkg.Close()
	}
	a.pkg = nil

	go func() {
		l := log.WithFields(log.Fields{
			"path": path,
		})

		pkg := ddpackage.NewPackage(l)

		err := pkg.LoadFromPackedPath(path)
		if err != nil {
			l.WithError(err).Error("could not load path")
			a.setErrContent(
				err,
				lang.X("err.badPack", "Failed to load {{.Path}}", map[string]any{"Path": path}),
			)
			return
		}

		err = pkg.LoadTags()
		if err != nil {
			a.showErrorDialog(errors.Join(err, fmt.Errorf(lang.X("unpack.tags.error", "Failed to read tags"))))
			err = nil
		}
		err = pkg.LoadResourceMetadata()
		if err != nil {
			a.showErrorDialog(errors.Join(err, fmt.Errorf(lang.X("unpack.metadata.error", "Failed to read metadata"))))
			err = nil
		}
		a.setPackContent(pkg)
	}()
}

func (a *App) setPackContent(pkg *ddpackage.Package) {
	a.pkg = pkg

	split := a.buildPackageTreeAndPreview()

	outputPath := binding.BindPreferenceString("unpack.outPath", a.app.Preferences())

	outEntry := widget.NewEntryWithData(outputPath)
	outEntry.SetPlaceHolder(lang.X("unpack.outPath.placeholder", "Where to extract resources"))
	outBrowseBtn := widget.NewButtonWithIcon(lang.X("browse", "Browse"), theme.FileIcon(), func() {
		dlg := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err == nil && lu != nil {
				log.Infof("open path %s", lu.Path())
				outputPath.Set(lu.Path())
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

	overwriteOption := binding.NewBool()
	ripTexOption := binding.NewBool()
	thumbnailsOption := binding.NewBool()

	overwriteCheck := widget.NewCheckWithData(lang.X("unpack.option.overwrite.text", "Overwrite existing files"), overwriteOption)
	ripTexCheck := widget.NewCheckWithData(lang.X("unpack.option.repTex.text", "Rip Textures to Png"), ripTexOption)
	thumbnailsCheck := widget.NewCheckWithData(lang.X("unpack.option.thumbnails.text", "Extract thumbnails"), thumbnailsOption)

	extrctBtn := widget.NewButtonWithIcon(lang.X("unpack.extractBtn.text", "Extract"), theme.UploadIcon(), func() {
		path, err := outputPath.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound output path value")
			return
		}
		overwrite, err := overwriteOption.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound overwrite value")
			return
		}
		ripTex, err := ripTexOption.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound ripTex value")
			return
		}
		thumbnails, err := thumbnailsOption.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound thumbnails value")
			return
		}
		a.extractPackage(path, ddpackage.UnpackOptions{
			Overwrite:   overwrite,
			RipTextures: ripTex,
			Thumbnails:  thumbnails,
		})
	})

	extractForm := container.NewVBox(
		container.New(
			custom_layout.NewLeftExpandHBoxLayout(),
			outEntry,
			outBrowseBtn,
		),
		container.NewHBox(
			overwriteCheck,
			ripTexCheck,
			thumbnailsCheck,
		),
		extrctBtn,
	)

	disableButtonsListener := binding.NewDataListener(func() {
		disable, _ := a.disableButtons.Get()
		log.Info("unpack buttons disable: ", disable)
		if disable {
			outBrowseBtn.Disable()
			extrctBtn.Disable()
		} else {
			outBrowseBtn.Enable()
			extrctBtn.Enable()
		}
	})

	a.setMainContent(
		container.New(
			custom_layout.NewBottomExpandVBoxLayout(),
			extractForm,
			split,
		),
		disableButtonsListener,
	)

	a.disableButtons.Set(false)
}

func (a *App) extractPackage(path string, options ddpackage.UnpackOptions) {
	a.disableButtons.Set(true)
	progressVal := binding.NewFloat()
	progressBar := widget.NewProgressBarWithData(progressVal)

	targetPath := filepath.Join(path, a.pkg.Name())
	progressDlg := dialog.NewCustomWithoutButtons(
		lang.X("unpack.extractProgressDlg.title", "Extracting to {{.Path}}", map[string]any{"Path": targetPath}),
		progressBar,
		a.window,
	)
	progressDlg.Show()
	go func() {
		err := a.pkg.ExtractPackage(path, options, func(p float64) {
			log.Info(p)
			progressVal.Set(p)
		})
		progressDlg.Hide()
		packPath, _ := a.operatingPath.Get()
		if err != nil {
			errDlg := dialog.NewError(
				errors.Join(err, errors.New(lang.X(
					"unpack.extract.error.text",
					"Error extracting {{.Pack}} to {{.Path}}",
					map[string]any{
						"Pack": packPath,
						"Path": targetPath,
					},
				))),
				a.window,
			)
			errDlg.Show()
			return
		}
		infoDlg := dialog.NewInformation(
			lang.X("unpack.success.dlg.title", "Extraction successful"),
			lang.X(
				"unpack.success.dlg.text",
				"{{.Pack}} extracted to {{.Path}} successfully",
				map[string]any{
					"Pack": packPath,
					"Path": targetPath,
				}),
			a.window,
		)
		infoDlg.Show()
		a.disableButtons.Set(false)
	}()
}
