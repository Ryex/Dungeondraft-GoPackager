package gui

import (
	"image/color"
	"path/filepath"
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

// waitForUpdateDialogHidden waits until no dialog is open over the window
// anymore.
func waitForUpdateDialogHidden(t *testing.T, a *App) {
	t.Helper()

	require.Eventually(t, func() bool {
		return updateDialogOverlay(a) == nil
	}, 10*time.Second, 20*time.Millisecond,
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

	// the hook is set before the package is loaded, so that the watcher
	// does not read it while the test writes it
	a.packageUpdateHook = func(dlg *packageUpdateDialog) {
		assert.False(t, dlg.progress,
			"the update of a few changed paths does not report progress")
		close(running)
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

	reports := 0
	a.packageUpdateHook = func(dlg *packageUpdateDialog) {
		reports++
		assert.True(t, dlg.progress,
			"a scan of the whole package reports its progress")
	}

	_, _ = loadWatchedPackage(t, a)

	// the update is serialized the same way as the updates of the watcher
	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		a.updateFromPaths(pkg, nil, true)
	})

	assert.Equal(t, 1, reports, "the update must show its dialog once")
	waitForUpdateDialogHidden(t, a)
}
