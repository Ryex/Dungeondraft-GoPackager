package gui

import (
	"image/color"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

// updateDialogOverlay returns the dialog that is open over the window, or
// nil when no dialog is open.
func updateDialogOverlay(a *App) fyne.CanvasObject {
	return a.window.Canvas().Overlays().Top()
}

// waitForUpdateDialogHidden waits until the dialog of the update is hidden.
//
// The update holds packageUpdateLock while it runs and hides the dialog
// before it releases the lock, so acquiring the lock is a deterministic
// signal that the hide path has run. The overlay stack is read once
// afterwards: polling it while the update removes the dialog would race
// inside Fyne's OverlayStack (see TestGenThumbnailsWithoutPackageHidesProgressDialog).
func waitForUpdateDialogHidden(t *testing.T, a *App) {
	t.Helper()

	require.Eventually(t, a.packageUpdateLock.TryLock, 10*time.Second, 20*time.Millisecond,
		"the update must finish")
	defer a.packageUpdateLock.Unlock()

	require.Nil(t, updateDialogOverlay(a),
		"the dialog of the update must be hidden when the update is done")
}

// TestPackageUpdateContentShowsProgressBarOrSpinner verifies that the
// content of the dialog of an update shows a progress bar when the update
// reports its progress and only a spinner when it does not.
func TestPackageUpdateContentShowsProgressBarOrSpinner(t *testing.T) {
	activityStr := binding.NewString()
	activityProgress := binding.NewFloat()

	content, activity := packageUpdateContent("msg", false, activityStr, activityProgress)
	assert.NotNil(t, activity,
		"an update that does not report progress shows a spinner")
	assert.Empty(t, collectObjects[*widget.ProgressBar](content),
		"an update that does not report progress shows no progress bar")

	content, activity = packageUpdateContent("msg", true, activityStr, activityProgress)
	assert.Nil(t, activity,
		"an update that reports progress shows no spinner")
	assert.NotEmpty(t, collectObjects[*widget.ProgressBar](content),
		"an update that reports progress shows a progress bar")
}

// TestPackageUpdateShowsDialogAndHidesIt verifies that an update of the
// package that the filesystem watcher reports shows a dialog while it runs
// and hides it when it is done.
func TestPackageUpdateShowsDialogAndHidesIt(t *testing.T) {
	a := newWatcherApp(t)

	// the update runs in the background, so the test holds it up, to look
	// at the interface while it runs
	running := make(chan struct{})
	release := make(chan struct{})
	released := false
	releaseUpdate := func() {
		if released {
			return
		}
		released = true
		close(release)
	}
	t.Cleanup(releaseUpdate)

	// The hook is set before the package is loaded, so that the watcher
	// does not read it while the test writes it.
	//
	// The watcher can report a second, spurious update: NTFS defers the
	// directory-entry update of the fixture write and delivers it as a
	// Write event on a directory when the watch is armed, which the
	// watcher enqueues like any other change, so a second, non-rescanning
	// update of the package runs. Such a report is a paths
	// update too, so its dialog has no progress — but closing running must
	// happen exactly once, or the second report would panic. The wait for
	// release stays outside the Once, so that every invocation honours it;
	// it cannot deadlock on a later invocation, because the reports are
	// serialized by packageUpdateLock: a second report only runs after the
	// first update finished, and that update can only finish after release
	// was closed.
	var first sync.Once
	a.packageUpdateHook = func(dlg *packageUpdateDialog) {
		assert.False(t, dlg.progress,
			"the update of a few changed paths does not report progress")
		first.Do(func() {
			close(running)
		})
		<-release
	}

	dir, _ := loadWatchedPackage(t, a)

	writeTestTexture(t, filepath.Join(dir, "textures", "busy.png"), color.White)

	<-running
	assert.NotNil(t, updateDialogOverlay(a),
		"the update must show a dialog while it runs")

	releaseUpdate()

	waitForWatcherIdle(t, a)
	waitForUpdateDialogHidden(t, a)
}

// TestPackageUpdateRescanReportsProgress verifies that a scan of the whole
// package reports its progress, because it walks every file of it.
func TestPackageUpdateRescanReportsProgress(t *testing.T) {
	a := newWatcherApp(t)

	_, _ = loadWatchedPackage(t, a)

	// The hook is installed for the manual rescan only. It is read under
	// packageUpdateLock (pack.go), and this closure runs under that lock
	// — the same lock the updates of the filesystem watcher hold — so a
	// concurrent report cannot invoke it: it waits for the lock, and the
	// hook is cleared again before the lock is released. A spurious
	// non-rescan report of the watcher (the deferred NTFS directory-entry
	// update of the fixture write, which is delivered as a Write event on
	// a directory when the watch is armed and makes the watcher run a
	// second, non-rescanning update of the package) is therefore neither
	// counted nor asserted here, while the update under test still is.
	reports := 0
	var scanned *packageUpdateDialog
	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		a.packageUpdateHook = func(dlg *packageUpdateDialog) {
			reports++
			scanned = dlg
		}
		// cleared while the lock is still held, so that a report of the
		// watcher can never see the hook
		defer func() { a.packageUpdateHook = nil }()
		a.updateFromPaths(pkg, nil, true)
	})

	assert.Equal(t, 1, reports, "the update must show its dialog once")
	require.NotNil(t, scanned, "the update must show its dialog")
	assert.True(t, scanned.progress,
		"a scan of the whole package reports its progress")
	waitForUpdateDialogHidden(t, a)
}

// TestPackageUpdateDialogReportThrottles verifies that the dialog of an
// update only forwards a progress that changed by at least a percent.
//
// The scan of a package reports its progress for every file of it, and every
// report queues a function on the Fyne goroutine. Without the throttle, the
// queue would hold a function for every file of the package, which slows the
// interface down (and blocks it) for a package with many files.
func TestPackageUpdateDialogReportThrottles(t *testing.T) {
	d := packageUpdateDialog{
		progress:         true,
		activityStr:      binding.NewString(),
		activityProgress: binding.NewFloat(),
	}

	notifications := 0
	d.activityProgress.AddListener(binding.NewDataListener(func() {
		notifications++
	}))
	// a listener is notified when it is connected, to give it the value
	// the binding has; the notifications of the reports are counted from
	// the first one of them
	notifications = 0

	for range 100 {
		d.report(0.5, "x", 0)
	}
	assert.Equal(t, 1, notifications,
		"the progress must only be reported when it changed by at least a percent")

	d.report(0.51, "x", 0)
	assert.Equal(t, 2, notifications,
		"a progress that changed by a percent must be reported again")
}
