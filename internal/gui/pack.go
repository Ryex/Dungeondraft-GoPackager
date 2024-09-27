package gui

import (
	"errors"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/custom_binding"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/custom_layout"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	log "github.com/sirupsen/logrus"
)

func (a *App) loadUnpackedPath(path string) {
	activity := a.setWaitContent(lang.X(
		"pack.wait",
		"Loading unpacked resources from {{.Path}} ...",
		map[string]any{
			"Path": utils.TruncatePathHumanFriendly(path, 80),
		},
	))
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

		err := pkg.LoadUnpackedFromFolder(path)
		if err != nil {
			l.WithError(err).Error("could not load directory")
			a.setErrContent(
				err,
				lang.X(
					"err.badResources",
					"Failed to load unpacked resources from {{.Path}}",
					map[string]any{"Path": path}),
			)
			return
		}

		err = pkg.BuildFileList(func(path string) {
			activity.Set(lang.X(
				"pack.buildList.activity",
				"Loading {{.Path}} ...",
				map[string]any{
					"Path": utils.TruncatePathHumanFriendly(path, 80),
				},
			))
		})
		if err != nil {
			l.WithError(err).Error("could not build file list")
			a.setErrContent(
				err,
				lang.X(
					"err.fileList",
					"Failed to build file list from {{.Path}}",
					map[string]any{
						"Path": path,
					},
				),
			)
			return
		}
		a.setUnpackedContent(pkg)
	}()
}

func (a *App) setUnpackedContent(pkg *ddpackage.Package) {
  a.pkg = pkg
	
	split := a.buildPackageTreeAndPreview()

	outputPath := binding.BindPreferenceString("pack.outPath", a.app.Preferences())

	outEntry := widget.NewEntryWithData(outputPath)
	outEntry.SetPlaceHolder(lang.X("pack.outPath.placeholder", "Where to save .dungeondraft_pack file"))
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
	thumbnailsOption := binding.NewBool()

	overwriteCheck := widget.NewCheckWithData(lang.X("pack.option.overwrite.text", "Overwrite existing files"), overwriteOption)
	thumbnailsCheck := widget.NewCheckWithData(lang.X("pack.option.thumbnails.text", "Generate thumbnails"), thumbnailsOption)

	packBtn := widget.NewButtonWithIcon(lang.X("pack.packBtn.text", "Package"), theme.DownloadIcon(), func() {
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
		thumbnails, err := thumbnailsOption.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound thumbnails value")
			return
		}
		a.packPackage(path, ddpackage.PackOptions{
			Overwrite: overwrite,
		}, thumbnails)
	})

	disableButtonsListener := binding.NewDataListener(func() {
		disable, _ := a.disableButtons.Get()
		if disable {
		} else {
		}
	})

	packForm := container.NewVBox(
		container.New(
			custom_layout.NewLeftExpandHBoxLayout(),
			outEntry,
			outBrowseBtn,
		),
		container.NewHBox(
			overwriteCheck,
			thumbnailsCheck,
		),
		packBtn,
	)

	a.setMainContent(
		container.New(
			custom_layout.NewBottomExpandVBoxLayout(),
			packForm,
			split,
		),
		disableButtonsListener,
	)

	a.disableButtons.Set(false)
}

func (a *App) packPackage(path string, options ddpackage.PackOptions, genThumbnails bool) {
	a.disableButtons.Set(true)

	thumbProgresVal := binding.NewFloat()
	packProgresVal := binding.NewFloat()
	progressVal := custom_binding.FloatMath(
		func(d ...float64) float64 {
			thumbP := d[0]
			packP := d[1]
			return thumbP*0.2 + packP*0.8
		},
		thumbProgresVal,
		packProgresVal,
	)
	progressBar := widget.NewProgressBarWithData(progressVal)

	targetPath := filepath.Join(path, a.pkg.Name()+".dungeondraft_pack")

	progressDlg := dialog.NewCustomWithoutButtons(
		lang.X("pack.packageProgressDlg.title", "Packing to {{.Path}}", map[string]any{"Path": targetPath}),
		progressBar,
		a.window,
	)
	progressDlg.Show()
	go func() {
		if genThumbnails {
			err := a.pkg.GenerateThumbnails(func(p float64) {
				thumbProgresVal.Set(p)
			})
			if err != nil {
				progressDlg.Hide()
				errDlg := dialog.NewError(
					errors.Join(err, errors.New(lang.X(
						"pack.thumbnails.error.text",
						"Error generating thumbnails for {{.Path}}",
						map[string]any{
							"Path": a.pkg.UnpackedPath,
						},
					))),
					a.window,
				)
				errDlg.Show()
				return
			}
		}
		thumbProgresVal.Set(1.0)

		err := a.pkg.PackPackage(path, options, func(p float64) {
			packProgresVal.Set(p)
		})
		progressDlg.Hide()
		if err != nil {
			errDlg := dialog.NewError(
				errors.Join(err, errors.New(lang.X(
					"pack.package.error.text",
					"Error packing {{.Path}} to {{.Pack}}",
					map[string]any{
						"Path": a.pkg.UnpackedPath,
						"Pack": targetPath,
					},
				))),
				a.window,
			)
			errDlg.Show()
			return
		}

		infoDlg := dialog.NewInformation(
			lang.X("pack.success.dlg.title", "Packaging successful"),
			lang.X(
				"pack.success.dlg.text",
				"{{.Path}} Packaged to {{.Pack}} successfully",
				map[string]any{
					"Path": a.pkg.UnpackedPath,
					"Pack": targetPath,
				}),
			a.window,
		)
		infoDlg.Show()
		a.disableButtons.Set(false)
	}()
}
