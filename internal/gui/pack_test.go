package gui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

// TestGenThumbnailsWithoutPackageHidesProgressDialog verifies that the
// progress dialog of the thumbnail generation is closed again when no
// package is loaded.
//
// The dialog is hidden on the goroutine of the generation, so the test
// must not poll the overlay stack while that goroutine removes the dialog
// from it: the poll and the removal would race inside Fyne's OverlayStack.
// Instead the test waits for the signal the hide path leaves behind — the
// buttons of the window are enabled again after the dialog is hidden (the
// last step of the hide path, see genThumbnails) — on the binding, which
// is synchronised internally, and reads the overlay stack once afterwards.
func TestGenThumbnailsWithoutPackageHidesProgressDialog(t *testing.T) {
	a := newTestApp(t)

	a.genThumbnails()

	require.Eventually(t, func() bool {
		disabled, err := a.disableButtons.Get()
		return err == nil && !disabled
	}, 10*time.Second, 20*time.Millisecond,
		"the thumbnail generation must finish without a package")

	require.Nil(t, a.window.Canvas().Overlays().Top(),
		"the progress dialog must be hidden when there is no package to generate thumbnails for")
}

// TestPackPackageFailureReenablesButtons verifies that the buttons of the
// window are enabled again when packing fails. They drive Open Pack and
// Open Folder — the controls a user needs to recover from the failure — so
// the error path of the task has to re-enable them like the success path
// does.
func TestPackPackageFailureReenablesButtons(t *testing.T) {
	a := newTestApp(t)
	_, pkg := loadFixturePackage(t, a)
	a.setUnpackedContent(pkg)

	// packing into a path that exists as a file fails
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a folder"), 0o644))

	a.packPackage(blocker, ddpackage.PackOptions{})

	require.Eventually(t, func() bool {
		disabled, err := a.disableButtons.Get()
		return err == nil && !disabled
	}, 10*time.Second, 20*time.Millisecond,
		"the buttons must be enabled again after a failed pack")
}
