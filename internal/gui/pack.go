package gui

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

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

	"github.com/davecgh/go-spew/spew"
	"github.com/fsnotify/fsnotify"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	log "github.com/sirupsen/logrus"
)

func (a *App) loadUnpackedPath(path string) {
	packjsonPath := filepath.Join(path, "pack.json")
	if !utils.FileExists(packjsonPath) {
		a.setNotAPackageContent(path)
		return
	}
	activityProgress, activityStr := a.setWaitContent(lang.X(
		"pack.wait",
		"Loading unpacked resources from {{.Path}} (building index) ...",
		map[string]any{
			"Path": utils.TruncatePathHumanFriendly(path, 80),
		},
	))
	a.disableButtons.Set(true)

	a.resetPkg()

	go func() {
		l := log.WithFields(log.Fields{
			"path": path,
		})

		pkg := ddpackage.NewPackage(l)

		err := pkg.LoadUnpackedFromFolder(path)
		if err != nil {
			l.WithError(err).Error("could not load directory")
			a.setErrContent(
				lang.X(
					"err.badResources",
					"Failed to load unpacked resources from {{.Path}}",
					map[string]any{"Path": path}),

				err,
			)
			return
		}

		errs := pkg.BuildFileListProgress(func(p float64, path string) {
			activityProgress.Set(p)
			activityStr.Set(lang.X(
				"pack.buildList.activity",
				"Loading {{.Path}} ...",
				map[string]any{
					"Path": utils.TruncatePathHumanFriendly(path, 80),
				},
			))
		})
		if len(errs) != 0 {
			for _, err := range errs {
				l.WithField("task", "build file list").Errorf("error: %s", err)
			}
			a.setErrContent(
				lang.X(
					"err.fileList",
					"Failed to build file list from {{.Path}}",
					map[string]any{
						"Path": path,
					},
				),
				err,
			)
			return
		}

		err = pkg.LoadTags()
		if err != nil {
			a.showErrorDialog(
				errors.Join(err, fmt.Errorf(lang.X("package.tags.error", "Failed to read tags"))))
			err = nil
		}
		err = pkg.LoadResourceMetadata()
		if err != nil {
			a.showErrorDialog(errors.Join(err, fmt.Errorf(lang.X("package.metadata.error", "Failed to read metadata"))))
			err = nil
		}
		a.setUnpackedContent(pkg)
		a.setupPackageWatcher()
	}()
}

func (a *App) setupPackageWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Error("failed to setup filesystem watcher")
		return
	}
	a.packageWatcher = watcher
	paths := structures.NewSet[string]()
	var eventTimer *time.Timer
	timerLock := &sync.RWMutex{}

	updatePackage := func() {
		timerLock.Lock()
		defer timerLock.Unlock()
		eventTimer = nil
		toUpdate := paths.AsSlice()
		paths = structures.NewSet[string]()
		if a.packageWatcherIgnoreThumbnails {
			thumbnailPrefix := filepath.Join(a.pkg.UnpackedPath(), "thumbnails")
			toUpdate = slices.Collect(utils.Filter(slices.Values(toUpdate), func(path string) bool {
				return !strings.HasPrefix(path, thumbnailPrefix)
			}))
		}
		if a.pkg != nil {
			a.pkg.UpdateFromPaths(toUpdate)
		}
		a.packageUpdated.Set(a.pkgUpdateCounter + 1)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Chmod) {
					path := event.Name
					func() {
						timerLock.Lock()
						defer timerLock.Unlock()
						if eventTimer != nil {
							eventTimer.Stop()
						}
						paths.Add(path)
						eventTimer = time.AfterFunc(
							2*time.Second,
							updatePackage,
						)
					}()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if errors.Is(err, fsnotify.ErrEventOverflow) {
					log.WithError(err).Warn("fsnotify overflow, force updating package")
					timerLock.Lock()
					defer timerLock.Unlock()
					if eventTimer != nil {
						eventTimer.Stop()
					}
					paths.Add(a.pkg.UnpackedPath())
					eventTimer = time.AfterFunc(
						2*time.Second,
						updatePackage,
					)
				}
				log.WithError(err).Warn("filesystem watcher error")
			}
		}
	}()

	toWatchPath := a.pkg.UnpackedPath()
	if toWatchPath != "" {
		_, dirs, _ := utils.ListDir(toWatchPath)
		for _, dir := range dirs {
			log.WithField("package", toWatchPath).Infof("watching %s", dir)
			err := watcher.Add(dir)
			if err != nil {
				log.WithError(err).WithField("package", toWatchPath).Warnf("failed to watch %s", dir)
			}
		}
	}
}

func (a *App) teardownPackageWatcher() {
	if a.packageWatcher != nil {
		a.packageWatcher.Close()
	}
}

func (a *App) setNotAPackageContent(path string) {
	msgText := multilineCanvasText(
		lang.X("notAPackage.message",
			"No pack.json at {{.Path}}:\nThis does not appear to be a resource pack.",
			map[string]string{
				"Path": path,
			},
		),
		16,
		fyne.TextStyle{},
		fyne.TextAlignCenter,
		theme.Color(theme.ColorNameForeground),
	)
	errTxtContainer := container.NewVBox()

	newPackBtn := widget.NewButton(
		lang.X("notAPackage.newPackBtn.text", "Create a pack.json"),
		func() {
			dlg := NewPackJSONDialog(
				path,
				func(options ddpackage.SavePackageJSONOptions) {
					log.Debug("pack json options: ", spew.Sdump(options))
					err := ddpackage.SavePackageJSON(
						log.WithField("path", path),
						options,
						true,
					)
					if err != nil {
						errTxtContainer.RemoveAll()
						errTxtContainer.Add(
							multilineCanvasText(
								err.Error(),
								12,
								fyne.TextStyle{Italic: true},
								fyne.TextAlignCenter,
								theme.Color(theme.ColorNameError),
							),
						)
						return
					}
					a.loadUnpackedPath(path)
				},
				a.window,
			)
			dlg.Show()
		},
	)

	a.setMainContent(
		container.NewVBox(
			layout.NewSpacer(),
			msgText,
			container.NewCenter(newPackBtn),
			errTxtContainer,
			layout.NewSpacer(),
		),
	)
}

func (a *App) setUnpackedContent(pkg *ddpackage.Package) {
	a.pkg = pkg

	split := a.buildPackageTreeAndInfoPane(true)

	outputPath := binding.BindPreferenceString("pack.outPath", a.app.Preferences())

	outLbl := widget.NewLabel(lang.X("outputPath.label", "Output Path"))
	outEntry := widget.NewEntryWithData(outputPath)
	outEntry.Validator = nil
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
		outPath := a.app.Preferences().String("pack.outPath")
		outPathURI := storage.NewFileURI(outPath)
		lisableOutPath, err := storage.ListerForURI(outPathURI)
		if err == nil {
			dlg.SetLocation(lisableOutPath)
		}
		dlg.Show()
	})

	overwriteOption := binding.NewBool()

	overwriteCheck := widget.NewCheckWithData(lang.X("pack.option.overwrite.text", "Overwrite existing files"), overwriteOption)

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
		a.packPackage(path, ddpackage.PackOptions{
			Overwrite: overwrite,
		})
	})
	editPackBtn := widget.NewButtonWithIcon(
		lang.X("pack.editPackBtn.text", "Edit settings"),
		theme.DocumentCreateIcon(),
		func() {
			dlg := NewPackJSONDialogPkg(a.pkg, true, func(options ddpackage.SavePackageJSONOptions) {
				err := ddpackage.SavePackageJSON(
					log.WithField("path", options.Path),
					options,
					true,
				)
				if err != nil {
					errDlg := dialog.NewError(
						errors.Join(err, errors.New(lang.X(
							"pack.edit.error.text",
							"Error saving {{.Path}}",
							map[string]any{
								"Path": filepath.Join(options.Path, "pack.json"),
							},
						))),
						a.window,
					)
					errDlg.Show()
					return
				}
				a.pkg.UpdateFromPaths([]string{filepath.Join(options.Path, "pack.json")})
				err = a.pkg.LoadUnpackedPackJSON(options.Path)
				if err != nil {
					errDlg := dialog.NewError(
						errors.Join(err, errors.New(lang.X(
							"pack.reload.error.text",
							"Error loading {{.Path}}",
							map[string]any{
								"Path": filepath.Join(options.Path, "pack.json"),
							},
						))),
						a.window,
					)
					errDlg.Show()
				}
			}, a.window)
			dlg.Show()
		},
	)
	thumbnailsBtn := widget.NewButtonWithIcon(
		lang.X("pack.option.thumbnails.text", "Generate thumbnails"),
		theme.MediaPhotoIcon(),
		func() {
			a.genthumbnails()
		},
	)
	tagSetsBtn := widget.NewButton(
		lang.X("pack.tagSetsBtn.text", "Edit Tag Sets"),
		func() {
			dlg := a.createTagSetsDialog(true)
			dlg.Show()
		},
	)

	generateTageBtn := widget.NewButton(
		lang.X("pack.generateTagsBtn.label", "Generate Tags"),
		func() {
			dlg := a.createTagGenDialog()
			dlg.Show()
		})

	packForm := container.NewVBox(
		layouts.NewLeftExpandHBox(
			container.New(layout.NewFormLayout(), outLbl, outEntry),
			outBrowseBtn,
		),
		container.New(
			xlayout.NewHPortion([]float64{40, 60}),
			container.New(
				xlayout.NewHPortion([]float64{50, 50}),

				container.NewVBox(
					thumbnailsBtn,
					editPackBtn,
				),
				container.NewVBox(
					generateTageBtn,
					tagSetsBtn,
				),
			),
			container.NewVBox(
				container.NewVBox(
					overwriteCheck,
				),
				packBtn,
			),
		),
	)

	a.setMainContent(
		container.New(
			layouts.NewBottomExpandVBoxLayout(),
			packForm,
			widget.NewSeparator(),
			split,
		),
		func(disable bool) {
			if disable {
			} else {
			}
		},
	)

	a.disableButtons.Set(false)
}

func (a *App) genthumbnails() {
	a.disableButtons.Set(true)

	progressVal := binding.NewFloat()
	progressBar := widget.NewProgressBarWithData(progressVal)
	progressDlg := dialog.NewCustomWithoutButtons(
		lang.X("task.genthumbnails.text", "Generating thumbnails ..."),
		container.NewPadded(progressBar),
		a.window,
	)
	progressDlg.Show()
	go func() {
		a.packageWatcherIgnoreThumbnails = true
		errs := a.pkg.GenerateThumbnailsProgress(func(p float64) {
			progressVal.Set(p)
		})
		progressDlg.Hide()
		if len(errs) != 0 {
			progressDlg.Hide()
			errDlg := dialog.NewError(
				errors.Join(append(errs, errors.New(lang.X(
					"pack.thumbnails.error.text",
					"Error generating thumbnails for {{.Path}}",
					map[string]any{
						"Path": a.pkg.UnpackedPath(),
					},
				)))...),
				a.window,
			)
			errDlg.Show()
			return
		}
		a.pkg.UpdateFromPaths([]string{filepath.Join(a.pkg.UnpackedPath(), "thumbnails")})
		a.packageWatcherIgnoreThumbnails = true
		a.disableButtons.Set(false)
	}()
}

func (a *App) packPackage(path string, options ddpackage.PackOptions) {
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
	taskStr := binding.NewString()
	progressBar := widget.NewProgressBarWithData(progressVal)
	taskLbl := widget.NewLabelWithData(taskStr)

	targetPath := filepath.Join(path, a.pkg.Name()+".dungeondraft_pack")

	progressDlg := dialog.NewCustomWithoutButtons(
		lang.X("pack.packageProgressDlg.title", "Packing to {{.Path}}", map[string]any{"Path": targetPath}),
		container.NewVBox(taskLbl, progressBar),
		a.window,
	)
	progressDlg.Show()
	go func() {
		taskStr.Set(lang.X("task.package.text", "Packaging resources ..."))
		err := a.pkg.PackPackageProgress(path, options, func(p float64) {
			progressVal.Set(p)
		})
		progressDlg.Hide()
		if err != nil {
			errDlg := dialog.NewError(
				errors.Join(err, errors.New(lang.X(
					"pack.package.error.text",
					"Error packing {{.Path}} to {{.Pack}}",
					map[string]any{
						"Path": a.pkg.UnpackedPath(),
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
					"Path": a.pkg.UnpackedPath(),
					"Pack": targetPath,
				}),
			a.window,
		)
		infoDlg.Show()
		a.disableButtons.Set(false)
	}()
}
