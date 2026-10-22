package gui

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xlayout "fyne.io/x/fyne/layout"

	"github.com/davecgh/go-spew/spew"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"

	dderrors "github.com/ryex/dungeondraft-gopackager/internal/errors"

	log "github.com/sirupsen/logrus"
)

// The names of the parts of the pack content.
//
// The parts of the interface are named where they are created, so that they
// can be looked up by name later, e.g. by the specification tests.
const (
	namePackPanel = "packPanel"
)

var (
	// errUnpackedFolder is returned when the package folder itself could not be read.
	errUnpackedFolder = errors.New("could not load unpacked package folder")
	// errUnpackedFileList is returned when the resource index could not be built.
	errUnpackedFileList = errors.New("could not build unpacked package file list")
)

// unpackedLoadResult is the outcome of loading an unpacked package folder.
type unpackedLoadResult struct {
	// Pkg is the loaded package. It is nil when the load failed.
	Pkg *ddpackage.Package
	// TagsErr is set when the optional tags file could not be read.
	// The package is still usable.
	TagsErr error
	// MetadataErr is set when the optional resource metadata could not be read.
	// The package is still usable.
	MetadataErr error
}

// loadUnpackedSync loads an unpacked package folder, builds its resource index
// and reads its tags and metadata. It performs no user interface updates so it
// can be driven directly from tests.
func (a *App) loadUnpackedSync(
	path string,
	progress func(p float64, curPath string),
) (unpackedLoadResult, error) {
	l := log.WithFields(log.Fields{
		"path": path,
	})

	res := unpackedLoadResult{}

	pkg := ddpackage.NewPackage(l)

	err := pkg.LoadUnpackedFromFolder(path)
	if err != nil {
		l.WithError(err).Error("could not load directory")
		return res, errors.Join(errUnpackedFolder, err)
	}

	errs := pkg.BuildFileListProgress(func(p float64, curPath string, _ int64) {
		if progress != nil {
			progress(p, curPath)
		}
	})
	if len(errs) != 0 {
		for _, err := range errs {
			l.WithField("task", "build file list").Errorf("error: %s", err)
		}
		joinErrs := make([]error, 0, len(errs)+1)
		joinErrs = append(joinErrs, errUnpackedFileList)
		joinErrs = append(joinErrs, errs...)
		return res, errors.Join(joinErrs...)
	}

	res.Pkg = pkg
	res.TagsErr = pkg.LoadTags()
	res.MetadataErr = pkg.LoadResourceMetadata()

	return res, nil
}

func (a *App) loadUnpackedPath(path string) {
	packJSONPath := filepath.Join(path, "pack.json")
	if !utils.FileExists(packJSONPath) {
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

	a.bgWG.Add(1)
	go func() {
		defer a.bgWG.Done()

		loader := a.loadUnpacked
		if loader == nil {
			loader = a.loadUnpackedSync
		}
		result, err := loader(path, func(p float64, curPath string) {
			activityProgress.Set(p)
			activityStr.Set(lang.X(
				"pack.buildList.activity",
				"Loading {{.Path}} ...",
				map[string]any{
					"Path": utils.TruncatePathHumanFriendly(curPath, 80),
				},
			))
		})
		if err != nil {
			if errors.Is(err, errUnpackedFileList) {
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
			} else {
				a.setErrContent(
					lang.X(
						"err.badResources",
						"Failed to load unpacked resources from {{.Path}}",
						map[string]any{"Path": path},
					),
					err,
				)
			}
			return
		}

		if result.TagsErr != nil {
			a.showErrorDialog(
				dderrors.CausedBy(errors.New(lang.X("package.tags.error", "Failed to read tags")), result.TagsErr),
			)
		}
		if result.MetadataErr != nil {
			a.showErrorDialog(
				dderrors.CausedBy(errors.New(lang.X("package.metadata.error", "Failed to read metadata")), result.MetadataErr),
			)
		}
		a.setUnpackedContent(result.Pkg)
		a.setupPackageWatcher()
	}()
}

// updateFromPaths updates the package with the resources that changed on
// disk.
//
// If full is set, the package is scanned completely, instead of updating
// only the given paths. It is used when the events of the watcher were lost
// (e.g. because the buffer of the operating system overflowed), so that all
// the changes are found.
func (a *App) updateFromPaths(pkg *ddpackage.Package, paths []string, full bool) {
	activity := widget.NewActivity()
	activity.Start()
	msgText := widgets.NewThemedText(lang.X(
		"pack.filesystemUpdate",
		"Updating resources from {{.Path}} (updating index) ...",
		map[string]any{
			"Path": utils.TruncatePathHumanFriendly(pkg.UnpackedPath(), 80),
		},
	), theme.ColorNameForeground)
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

	progressDlg := dialog.NewCustomWithoutButtons(
		lang.X("task.updatePackage.text", "Updating pack resources ..."),
		activityContent,
		a.window,
	)
	progressDlg.Show()

	progress := func(p float64, path string, _ int64) {
		activityStr.Set(lang.X(
			"pack.buildList.activity",
			"Loading {{.Path}} ...",
			map[string]any{
				"Path": utils.TruncatePathHumanFriendly(path, 80),
			},
		))
	}

	// the info of the package is not part of the file list, so it has to be
	// loaded separately, when the pack.json changed
	rebuild := a.reloadPackInfo(pkg, paths)

	var errs []error
	if full || rebuild {
		log.Debugf("rescanning the package folder %s", pkg.UnpackedPath())
		errs = pkg.BuildFileListProgress(progress)
	} else {
		errs = pkg.UpdateFromPathsProgress(paths, progress)
	}
	for _, err := range errs {
		log.WithError(err).Error("could not update the package from the changed paths")
	}

	err := pkg.LoadTags()
	if err != nil {
		a.showErrorDialog(
			dderrors.CausedBy(errors.New(lang.X("package.tags.error", "Failed to read tags")), err),
		)
	}
	err = pkg.LoadResourceMetadata()
	if err != nil {
		a.showErrorDialog(dderrors.CausedBy(errors.New(lang.X("package.metadata.error", "Failed to read metadata")), err))
	}
	progressDlg.Hide()
	a.packageUpdated.Set(a.pkgUpdateCounter + 1)
}

// reloadPackInfo reloads the info of the package from its pack.json, when
// the pack.json changed.
//
// It reports whether the package has to be scanned completely, because the
// id of the pack is part of the resource paths.
func (a *App) reloadPackInfo(pkg *ddpackage.Package, paths []string) bool {
	packJSONPath := filepath.Join(pkg.UnpackedPath(), "pack.json")

	changed := false
	for _, path := range paths {
		if samePath(path, packJSONPath) {
			changed = true
			break
		}
	}
	if !changed {
		return false
	}

	oldID := pkg.ID()
	if err := pkg.LoadUnpackedPackJSON(pkg.UnpackedPath()); err != nil {
		log.WithError(err).Error("could not reload the pack.json")
		return false
	}

	return oldID != pkg.ID()
}

// samePath reports whether the two paths point to the same file.
func samePath(a string, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// onPackageChanged updates the package with the paths that changed on disk.
//
// It is called by the filesystem watcher, when the debounce time passed
// without further events.
func (a *App) onPackageChanged(paths []string, full bool) {
	pkg := a.currentPackage()
	if pkg == nil {
		return
	}

	// the updates are serialized, so that the package is not updated while
	// it is read
	a.packageUpdateLock.Lock()
	defer a.packageUpdateLock.Unlock()

	log.Debugf("updating resource paths... %s", paths)

	// the update touches the interface, so it has to run on the Fyne
	// goroutine
	fyne.DoAndWait(func() {
		a.updateFromPaths(pkg, paths, full)
	})

	a.packageUpdateCount++
}

// setupPackageWatcher starts the watcher that notices the changes of the
// files of the unpacked package.
func (a *App) setupPackageWatcher() {
	if a.packageWatcherDisabled {
		return
	}
	watcher, err := newPackageWatcher(
		a.packageWatcherDebounce,
		a.onPackageChanged,
		func(err error) {
			log.WithError(err).Warn("filesystem watcher error")
		},
	)
	if err != nil {
		log.WithError(err).Error("failed to setup filesystem watcher")
		return
	}
	a.packageWatcher = watcher

	toWatchPath := a.pkg.UnpackedPath()
	if toWatchPath == "" {
		return
	}
	if err := watcher.Watch(toWatchPath); err != nil {
		log.WithError(err).WithField("package", toWatchPath).Warnf("failed to watch %s", toWatchPath)
	}
}

// pausePackageWatcher stops the watcher from reporting the changes of the
// files of the package.
//
// The changes that happen while the watcher is paused are still queued, so
// that they are reported when it is resumed.
func (a *App) pausePackageWatcher() {
	if a.packageWatcher == nil {
		return
	}
	a.packageWatcher.Pause()
}

// resumePackageWatcher lets the watcher report the changes that happened
// while it was paused.
func (a *App) resumePackageWatcher() {
	if a.packageWatcher == nil {
		return
	}
	a.packageWatcher.Resume()
}

// discardPackageWatcherChanges forgets the queued changes of the files under
// the given folder.
//
// It is used by the tasks that change the files of the package themselves
// and update the affected parts of the package themselves.
func (a *App) discardPackageWatcherChanges(dir string) {
	if a.packageWatcher == nil {
		return
	}
	a.packageWatcher.DiscardUnder(dir)
}

func (a *App) teardownPackageWatcher() {
	if a.packageWatcher != nil {
		a.packageWatcher.Close()
		a.packageWatcher = nil
	}
}

func (a *App) setNotAPackageContent(path string) {
	msgText := multilineCanvasText(
		lang.X(
			"notAPackage.message",
			"No pack.json at {{.Path}}:\nThis does not appear to be a resource pack.",
			map[string]string{
				"Path": path,
			},
		),
		16,
		fyne.TextStyle{},
		fyne.TextAlignCenter,
		theme.ColorNameForeground,
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
								theme.ColorNameError,
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
	a.setCurrentPackage(pkg)

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
		listableOutPath, err := storage.ListerForURI(outPathURI)
		if err == nil {
			dlg.SetLocation(listableOutPath)
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
						dderrors.CausedBy(errors.New(lang.X(
							"pack.edit.error.text",
							"Error saving {{.Path}}",
							map[string]any{
								"Path": filepath.Join(options.Path, "pack.json"),
							},
						)), err),
						a.window,
					)
					errDlg.Show()
					return
				}
				a.pkg.UpdateFromPaths([]string{filepath.Join(options.Path, "pack.json")})
				err = a.pkg.LoadUnpackedPackJSON(options.Path)
				if err != nil {
					errDlg := dialog.NewError(
						dderrors.CausedBy(errors.New(lang.X(
							"pack.reload.error.text",
							"Error loading {{.Path}}",
							map[string]any{
								"Path": filepath.Join(options.Path, "pack.json"),
							},
						)), err),
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
			a.genThumbnails()
		},
	)
	tagSetsBtn := widget.NewButton(
		lang.X("pack.tagSetsBtn.text", "Edit Tag Sets"),
		func() {
			dlg := a.createTagSetsDialog(true)
			dlg.Show()
		},
	)

	generateTagBtn := widget.NewButton(
		lang.X("pack.generateTagsBtn.label", "Generate Tags"),
		func() {
			dlg := a.createTagGenDialog()
			dlg.Show()
		},
	)

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
					generateTagBtn,
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

	// the parts of the interface have to be named before the content is
	// shown, so that they are available as soon as the new content is visible
	a.setMainContent(
		container.New(
			layouts.NewBottomExpandVBoxLayout(),
			named.New(namePackPanel, packForm),
			widget.NewSeparator(),
			split,
		),
		func(disable bool) {
		},
	)

	a.disableButtons.Set(false)
}

func (a *App) genThumbnails() {
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
		// the generation writes a lot of files, so the watcher is paused,
		// to not be flooded with the events of the generated thumbnails
		a.pausePackageWatcher()
		defer a.resumePackageWatcher()

		pkg := a.currentPackage()
		if pkg == nil {
			a.disableButtons.Set(false)
			return
		}

		errs := pkg.GenerateThumbnailsProgress(func(p float64) {
			progressVal.Set(p)
		})
		fyne.Do(func() {
			progressDlg.Hide()
		})
		if len(errs) != 0 {
			fyne.Do(func() {
				errDlg := dialog.NewError(
					dderrors.CausedBy(errors.New(lang.X(
						"pack.thumbnails.error.text",
						"Error generating thumbnails for {{.Path}}",
						map[string]any{
							"Path": pkg.UnpackedPath(),
						},
					)), errs...),
					a.window,
				)
				errDlg.Show()
			})
			a.disableButtons.Set(false)
			return
		}
		thumbnailDir := filepath.Join(pkg.UnpackedPath(), "thumbnails")
		if errs := pkg.UpdateFromPaths([]string{thumbnailDir}); len(errs) > 0 {
			for _, err := range errs {
				log.WithError(err).Error("could not update the package with the generated thumbnails")
			}
		}
		// the changes of the generation are already applied, so they are
		// not reported by the watcher again
		a.discardPackageWatcherChanges(thumbnailDir)
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
				dderrors.CausedBy(errors.New(lang.X(
					"pack.package.error.text",
					"Error packing {{.Path}} to {{.Pack}}",
					map[string]any{
						"Path": a.pkg.UnpackedPath(),
						"Pack": targetPath,
					},
				)), err),
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
				},
			),
			a.window,
		)
		infoDlg.Show()
		a.disableButtons.Set(false)
	}()
}
