package gui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// TestSaveTimersAreGuardedUnderConcurrentSaves drives the save callbacks and
// the flush of stopSaveTimersFor from two goroutines at once. The timer
// fields are shared state: without a lock a write to the timer map while it
// is flushed is a runtime fatal (concurrent map writes), not just a data
// race, and a flush can stop a timer that was just created for another
// package.
func TestSaveTimersAreGuardedUnderConcurrentSaves(t *testing.T) {
	a := newTestApp(t)
	_, pkg := loadFixturePackage(t, a)
	a.setUnpackedContent(pkg)

	wallMetaPath := "res://packs/testpack/data/walls/Stone.dungeondraft_wall"
	tilesetMetaPath := "res://packs/testpack/data/tilesets/Grass.dungeondraft_tileset"
	require.Contains(t, a.pkg.Walls(), wallMetaPath)
	require.Contains(t, a.pkg.Tilesets(), tilesetMetaPath)

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	// In the app the save callbacks run on the Fyne goroutine; here they
	// hammer the shared timer fields from a goroutine of their own.
	go func() {
		defer wg.Done()
		defer close(done)
		for i := 0; i < 50000; i++ {
			a.saveUnpackedTags()
			a.saveWallMetadata(wallMetaPath)
			a.saveTilesetMetadata(tilesetMetaPath)
		}
	}()

	// ... while the flush of a package replace runs on the load goroutine.
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				a.stopSaveTimersFor(pkg)
			}
		}
	}()

	wg.Wait()

	// after the flush no save may be left pending, whatever the two
	// goroutines interleaved to
	a.stopSaveTimersFor(pkg)
	require.Nil(t, a.tagSaveTimer, "no tag save must be left pending")
	require.Empty(t, a.resSaveTimers, "no metadata save must be left pending")
}
