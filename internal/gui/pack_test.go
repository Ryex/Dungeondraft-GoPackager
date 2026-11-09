package gui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGenThumbnailsWithoutPackageHidesProgressDialog verifies that the
// progress dialog of the thumbnail generation is closed again when no
// package is loaded.
func TestGenThumbnailsWithoutPackageHidesProgressDialog(t *testing.T) {
	a := newTestApp(t)

	a.genThumbnails()

	require.Eventually(t, func() bool {
		return a.window.Canvas().Overlays().Top() == nil
	}, 10*time.Second, 20*time.Millisecond,
		"the progress dialog must be hidden when there is no package to generate thumbnails for")
}
