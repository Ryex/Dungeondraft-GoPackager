package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestReplacePackageFlushesPendingTagSave checks that a tag save that is
// still pending when the package is replaced is written to disk right away,
// instead of being left to a timer that fires after the package was closed.
func TestReplacePackageFlushesPendingTagSave(t *testing.T) {
	a := newTestApp(t)
	dir, pkg := loadFixturePackage(t, a)
	a.setUnpackedContent(pkg)

	tagsPath := filepath.Join(dir, "data", "default.dungeondraft_tags")

	a.pkg.Tags().Tag("pending", "textures/objects/Rock.png")
	a.saveUnpackedTags()
	a.resetPkg()

	// the flush happens while resetPkg runs, so the file must hold the new
	// tag long before the 500 ms debounce of the save timer would fire
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(tagsPath)
		return err == nil && strings.Contains(string(data), "pending")
	}, 300*time.Millisecond, 10*time.Millisecond,
		"the pending tag save must be flushed when the package is replaced")
}
