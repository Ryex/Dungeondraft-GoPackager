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

	// The watcher of the package that is loaded over is stopped right away,
	// so that its changes are not reported while the new package is loaded.
	// The package itself is only dropped when the new one is shown (see
	// setUnpackedContent), so that the interface never shows the panel of a
	// package that is not loaded anymore.
	a.teardownPackageWatcher()

	a.bgWork.Start()
	go func() {
		defer a.bgWork.Done()

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
//
// The scan runs on the goroutine of the caller, so that the interface stays
// usable while it runs: it only writes the resource index of the package,
// which is guarded by its own lock. The pack info, the tags and the
// resource metadata — which the interface reads without a lock — are read
// on that goroutine as well: they read files from disk, which must not
// block the Fyne goroutine, and the update holds packageUpdateLock while it
// runs, so the package cannot be replaced (and closed) while it is read.
// Only the parts of the update that touch the interface run on the Fyne
// goroutine as its last step. The dialog of the update is hidden with them,
// so that it disappears as soon as the package can be used again.
//
// The update never waits for the Fyne goroutine: the Fyne goroutine can
// wait for the update, when a package is replaced (see replacePackage).
func (a *App) updateFromPaths(pkg *ddpackage.Package, paths []string, full bool) {
	// the info of the package is not part of the file list, so it has to be
	// loaded separately, when the pack.json changed
	rebuild := a.reloadPackInfo(pkg, paths)

	// A scan of the package reports its progress for every file of it,
	// while the update of a few changed paths is over in a moment: the
	// dialog shows a progress bar for the scan and only a spinner for the
	// paths, because there is no progress to report for them.
	rescan := full || rebuild

	dlg := a.showPackageUpdateDialog(pkg, rescan)

	if a.packageUpdateHook != nil {
		a.packageUpdateHook(dlg)
	}

	var errs []error
	if rescan {
		log.Debugf("rescanning the package folder %s", pkg.UnpackedPath())
		errs = pkg.BuildFileListProgress(dlg.report)
	} else {
		errs = pkg.UpdateFromPathsProgress(paths, dlg.report)
	}
	for _, err := range errs {
		log.WithError(err).Error("could not update the package from the changed paths")
	}

	// The tags and the resource metadata are read here, on the goroutine
	// of the update, before the interface is touched: they read files from
	// disk, which must not block the Fyne goroutine. The update runs under
	// packageUpdateLock, so the package cannot be replaced (and closed)
	// while it is read.
	tagsErr := pkg.LoadTags()
	metaErr := pkg.LoadResourceMetadata()

	// showErrorDialog queues its own dialog on the Fyne goroutine, so it is
	// called here, outside of the fyne.Do below: the fyne.Do only hides the
	// dialog of the update and reports that the package was updated.
	if tagsErr != nil {
		a.showErrorDialog(
			dderrors.CausedBy(errors.New(lang.X("package.tags.error", "Failed to read tags")), tagsErr),
		)
	}
	if metaErr != nil {
		a.showErrorDialog(
			dderrors.CausedBy(errors.New(lang.X("package.metadata.error", "Failed to read metadata")), metaErr),
		)
	}

	fyne.Do(func() {
		defer dlg.hide()

		a.packageUpdated.Set(a.pkgUpdateCounter + 1)
	})
}

// packageUpdateDialog informs about an update of the package that runs in
// the background.
type packageUpdateDialog struct {
	dialog *dialog.CustomDialog
	// activity is the spinner of the dialog. It is nil when the dialog
	// shows a progress bar.
	activity *widget.Activity

	// progress is set when the dialog shows a progress bar, otherwise the
	// update does not report any progress and the dialog only shows that
	// the package is updated.
	progress bool

	// activityStr and activityProgress describe what the update does. They
	// are set from the goroutine of the update.
	activityStr      binding.String
	activityProgress binding.Float

	// lastPercent is the progress that was reported last. The progress of a
	// scan is reported for every file of the package, so it is throttled,
	// to not flood the queue of the Fyne goroutine.
	lastPercent int
}

// showPackageUpdateDialog shows the dialog that informs about an update of
// the package of the app.
//
// It must be called on the goroutine of the update; the dialog and its
// content are built and shown on the Fyne goroutine, because they are
// widgets.
func (a *App) showPackageUpdateDialog(
	pkg *ddpackage.Package,
	progress bool,
) *packageUpdateDialog {
	msg := lang.X(
		"pack.filesystemUpdate",
		"Updating resources from {{.Path}} (updating index) ...",
		map[string]any{
			"Path": utils.TruncatePathHumanFriendly(pkg.UnpackedPath(), 80),
		},
	)

	// The bindings are created here, outside of the fyne.Do, so that the
	// update can report what it does while the dialog is still built.
	activityStr := binding.NewString()
	activityProgress := binding.NewFloat()

	d := &packageUpdateDialog{
		progress:         progress,
		activityStr:      activityStr,
		activityProgress: activityProgress,
	}

	// The dialog and its content are widgets, so they are built on the Fyne
	// goroutine and not on the goroutine of the update. This runs before
	// the hide of the dialog (which also runs on the Fyne goroutine and is
	// queued after this), so the FIFO order of the queue keeps d.dialog
	// non-nil when hide reads it.
	fyne.Do(func() {
		content, activity := packageUpdateContent(msg, progress, activityStr, activityProgress)
		d.activity = activity
		d.dialog = dialog.NewCustomWithoutButtons(
			lang.X("task.updatePackage.text", "Updating pack resources ..."),
			content,
			a.window,
		)
		d.dialog.Show()
	})

	return d
}

// packageUpdateContent builds the content of the dialog of a package update.
//
// When progress is set, the content shows a progress bar, because the scan
// of the package reports its progress for every file of it. Otherwise it
// only shows a spinner: the update of a few changed paths does not report
// any progress.
//
// It returns the content and the spinner of it, which is nil when the
// content shows a progress bar.
func packageUpdateContent(
	msg string,
	progress bool,
	activityStr binding.String,
	activityProgress binding.Float,
) (fyne.CanvasObject, *widget.Activity) {
	msgText := widgets.NewThemedText(msg, theme.ColorNameForeground)
	msgText.Text.TextSize = 16
	msgText.Text.Alignment = fyne.TextAlignCenter
	activityText := widgets.NewThemedText("", theme.ColorNameForeground)
	activityText.Text.TextSize = 12
	activityText.Text.Alignment = fyne.TextAlignCenter
	bindings.Listen(activityStr, func(str string) {
		activityText.Text.Text = str
		activityText.Refresh()
	})

	var activity *widget.Activity
	objects := []fyne.CanvasObject{layout.NewSpacer()}
	if progress {
		progressBar := widget.NewProgressBar()
		progressBar.Bind(activityProgress)
		objects = append(objects, msgText, activityText, container.NewPadded(progressBar))
	} else {
		activity = widget.NewActivity()
		activity.Start()
		objects = append(objects, activity, msgText, activityText)
	}
	objects = append(objects, layout.NewSpacer())

	return container.NewVBox(objects...), activity
}

// report reports what the update does and, when it reports any progress,
// how far it got.
//
// It is called by the scan of the package, for every file of it, so the
// progress is throttled, to not flood the queue of the Fyne goroutine. Only
// the goroutine of the update reports, so no lock is needed.
func (d *packageUpdateDialog) report(p float64, curPath string, _ int64) {
	if !d.progress {
		return
	}

	percent := int(p * 100)
	if percent == d.lastPercent {
		return
	}
	d.lastPercent = percent

	d.activityProgress.Set(p)
	d.activityStr.Set(lang.X(
		"pack.buildList.activity",
		"Loading {{.Path}} ...",
		map[string]any{
			"Path": utils.TruncatePathHumanFriendly(curPath, 80),
		},
	))
}

// hide hides the dialog and stops its spinner. It runs on the Fyne
// goroutine, where the interface can be changed.
func (d *packageUpdateDialog) hide() {
	if d.activity != nil {
		d.activity.Stop()
	}
	d.dialog.Hide()
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
// without further events, on the goroutine of the watcher.
//
// The whole update — including the full rescan that a lost event of the
// watcher triggers — intentionally runs under packageUpdateLock. Updates
// only ever run against unpacked packages, where Close() is a no-op, so a
// long hold of the lock only makes a replacement of the package wait for
// the update, which is what it is for: the package must not be closed while
// it is updated. The critical section is deliberately not shrunk, because
// an update must never wait for the Fyne goroutine while it holds the lock:
// the Fyne goroutine can wait for the lock itself, when a package is
// replaced (see replacePackage), so waiting for it here would deadlock.
func (a *App) onPackageChanged(paths []string, full bool) {
	// the updates are serialized, so that the package is not updated while
	// it is read
	a.packageUpdateLock.Lock()
	defer a.packageUpdateLock.Unlock()

	pkg := a.currentPackage()
	if pkg == nil {
		return
	}

	log.Debugf("updating resource paths... %s", paths)

	// The update runs here, on the goroutine of the watcher, so that the
	// interface stays usable while it runs: it only queues the parts of it
	// that touch the interface. It must not wait for the Fyne goroutine,
	// because the Fyne goroutine can wait for this update, when a package
	// is replaced.
	a.updateFromPaths(pkg, paths, full)

	a.packageUpdateCount++
}

// watcher returns the filesystem watcher of the app, or nil when there is
// none. It loads the pointer atomically, because the watcher is replaced
// when a package is loaded, while it is read.
func (a *App) watcher() *packageWatcher {
	return a.packageWatcher.Load()
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
	a.packageWatcher.Store(watcher)

	pkg := a.currentPackage()
	if pkg == nil {
		return
	}
	toWatchPath := pkg.UnpackedPath()
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
	if w := a.watcher(); w != nil {
		w.Pause()
	}
}

// resumePackageWatcher lets the watcher report the changes that happened
// while it was paused.
func (a *App) resumePackageWatcher() {
	if w := a.watcher(); w != nil {
		w.Resume()
	}
}

// discardPackageWatcherChanges forgets the queued changes of the files under
// the given folder.
//
// It is used by the tasks that change the files of the package themselves
// and update the affected parts of the package themselves.
func (a *App) discardPackageWatcherChanges(dir string) {
	if w := a.watcher(); w != nil {
		w.DiscardUnder(dir)
	}
}

func (a *App) teardownPackageWatcher() {
	if w := a.packageWatcher.Swap(nil); w != nil {
		w.Close()
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
	// the package of the app is replaced only now, when the content that
	// shows it is built, so that the interface never refers to a package
	// that is not loaded anymore
	a.replacePackage(pkg)

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
			dlg := NewPackJSONDialogPkg(a.currentPackage(), true, func(options ddpackage.SavePackageJSONOptions) {
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
				a.currentPackage().UpdateFromPaths([]string{filepath.Join(options.Path, "pack.json")})
				err = a.currentPackage().LoadUnpackedPackJSON(options.Path)
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

	// The tasks of the package (the generation of the thumbnails, the
	// packing) disable the interface while they run, so the controls of the
	// panel are disabled with it.
	packPanelControls := []fyne.Disableable{
		outEntry,
		outBrowseBtn,
		thumbnailsBtn,
		editPackBtn,
		generateTagBtn,
		tagSetsBtn,
		overwriteCheck,
		packBtn,
	}

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
			for _, control := range packPanelControls {
				if disable {
					control.Disable()
				} else {
					control.Enable()
				}
			}
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
			fyne.Do(func() {
				progressDlg.Hide()
			})
			a.disableButtons.Set(false)
			return
		}

		// the progress is reported for every thumbnail of the package, so it
		// is throttled, to not flood the queue of the Fyne goroutine
		progress := newProgressReporter(func(p float64) {
			progressVal.Set(p)
		})
		errs := pkg.GenerateThumbnailsProgress(progress.Report)
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
	pkg := a.currentPackage()
	if pkg == nil {
		log.WithError(ddpackage.ErrPackageNotLoaded).Warn("cannot pack without a loaded package")
		return
	}
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

	targetPath := filepath.Join(path, pkg.Name()+".dungeondraft_pack")

	progressDlg := dialog.NewCustomWithoutButtons(
		lang.X("pack.packageProgressDlg.title", "Packing to {{.Path}}", map[string]any{"Path": targetPath}),
		container.NewVBox(taskLbl, progressBar),
		a.window,
	)
	progressDlg.Show()
	go func() {
		taskStr.Set(lang.X("task.package.text", "Packaging resources ..."))
		// the progress is reported for every file of the package, so it is
		// throttled, to not flood the queue of the Fyne goroutine
		progress := newProgressReporter(func(p float64) {
			progressVal.Set(p)
		})
		err := pkg.PackPackageProgress(path, options, progress.Report)
		fyne.Do(func() {
			progressDlg.Hide()
		})
		if err != nil {
			fyne.Do(func() {
				errDlg := dialog.NewError(
					dderrors.CausedBy(errors.New(lang.X(
						"pack.package.error.text",
						"Error packing {{.Path}} to {{.Pack}}",
						map[string]any{
							"Path": pkg.UnpackedPath(),
							"Pack": targetPath,
						},
					)), err),
					a.window,
				)
				errDlg.Show()
			})
			return
		}

		fyne.Do(func() {
			infoDlg := dialog.NewInformation(
				lang.X("pack.success.dlg.title", "Packaging successful"),
				lang.X(
					"pack.success.dlg.text",
					"{{.Path}} Packaged to {{.Pack}} successfully",
					map[string]any{
						"Path": pkg.UnpackedPath(),
						"Pack": targetPath,
					},
				),
				a.window,
			)
			infoDlg.Show()
		})
		a.disableButtons.Set(false)
	}()
}
