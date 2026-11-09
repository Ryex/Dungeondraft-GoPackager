package gui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
