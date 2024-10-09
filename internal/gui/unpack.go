package gui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xlayout "fyne.io/x/fyne/layout"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"

	log "github.com/sirupsen/logrus"
)

func (a *App) loadPack(path string) {
	activityProgress, activityStr := a.setWaitContent(
		lang.X(
			"unpack.wait",
			"Loading {{.Path}} ...",
			map[string]any{
				"Path": path,
			},
		),
	)
	a.disableButtons.Set(true)

	a.resetPkg()

	go func() {
		l := log.WithFields(log.Fields{
			"path": path,
		})

		pkg := ddpackage.NewPackage(l)

		err := pkg.LoadFromPackedPath(path, func(p float64, curRes string) {
			activityProgress.Set(p)
			activityStr.Set(
				"res://" + utils.TruncatePathHumanFriendly(
					strings.TrimPrefix(curRes, "res://"),
					80,
				),
			)
		})
		if err != nil {
			l.WithError(err).Error("could not load path")
			a.setErrContent(
				lang.X("err.badPack", "Failed to load {{.Path}}", map[string]any{"Path": path}),
				err,
			)
			return
		}

		err = pkg.LoadTags()
		if err != nil {
			a.showErrorDialog(errors.Join(err, fmt.Errorf(lang.X("package.tags.error", "Failed to read tags"))))
			err = nil
		}
		err = pkg.LoadResourceMetadata()
		if err != nil {
			a.showErrorDialog(errors.Join(err, fmt.Errorf(lang.X("package.metadata.error", "Failed to read metadata"))))
			err = nil
		}
		a.setPackContent(pkg)
	}()
}

func (a *App) setPackContent(pkg *ddpackage.Package) {
	a.pkg = pkg

	split := a.buildPackageTreeAndInfoPane(false)

	outputPath := binding.BindPreferenceString("unpack.outPath", a.app.Preferences())

	outLbl := widget.NewLabel(lang.X("outputPath.label", "Output Path"))
	outEntry := widget.NewEntryWithData(outputPath)
	outEntry.Validator = nil
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
		outPath := a.app.Preferences().String("unpack.outPath")
		outPathURI := storage.NewFileURI(outPath)
		lisableOutPath, err := storage.ListerForURI(outPathURI)
		if err == nil {
			dlg.SetLocation(lisableOutPath)
		}
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
	infoBtn := widget.NewButtonWithIcon(
		lang.X("unpack.infoBtn.text", "Package Information"),
		theme.DocumentIcon(),
		func() {
			dlg := NewPackJSONDialogPkg(a.pkg, false, nil, a.window)
			dlg.Show()
		},
	)
	tagSetsBtn := widget.NewButton(
		lang.X("unpack.tagSetsBtn.text", "View Tag Sets"),
		func() {
			dlg := a.createTagSetsDialog(false)
			dlg.Show()
		},
	)

	extractForm := container.NewVBox(
		layouts.NewLeftExpandHBox(
			container.New(layout.NewFormLayout(), outLbl, outEntry),
			outBrowseBtn,
		),
		container.New(
			xlayout.NewHPortion([]float64{20, 80}),

			container.NewVBox(
				infoBtn,
				tagSetsBtn,
			),
			container.NewVBox(
				container.NewHBox(
					overwriteCheck,
					ripTexCheck,
					thumbnailsCheck,
				),
				extrctBtn,
			),
		),
	)

	a.setMainContent(
		container.New(
			layouts.NewBottomExpandVBoxLayout(),
			extractForm,
			split,
		),
		func(disable bool) {
			log.Info("unpack buttons disable: ", disable)
			if disable {
				outBrowseBtn.Disable()
				extrctBtn.Disable()
			} else {
				outBrowseBtn.Enable()
				extrctBtn.Enable()
			}
		},
	)

	a.disableButtons.Set(false)
}

func (a *App) extractPackage(path string, options ddpackage.UnpackOptions) {
	if path == "" {
		dialog.ShowInformation(
			lang.X("needOutPathDialog.title", "Please Provide an Output Path"),
			lang.X("newOutPathDialog.message", "The output path can not be empty."),
			a.window,
		)
		return
	}
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
		err := a.pkg.ExtractPackageProgress(targetPath, options, func(p float64) {
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
