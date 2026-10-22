package gui

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	ddcolor "github.com/ryex/dungeondraft-gopackager/pkg/structures/color"
)

// watcherTestDebounce is the debounce time the watcher uses in the tests.
//
// It is a lot shorter than the debounce time of the app, to keep the tests
// fast.
const watcherTestDebounce = 50 * time.Millisecond

// newWatcherApp creates an app with the filesystem watcher enabled.
//
// The debounce time of the watcher is shortened, so that the tests don't
// have to wait for the debounce time of the app.
func newWatcherApp(t *testing.T) *App {
	t.Helper()

	a := NewAppWithFyneApp(test.NewApp())
	a.app.Preferences().SetString("locale", "en")
	a.packageWatcherDebounce = watcherTestDebounce
	a.setup()
	t.Cleanup(a.clean)

	return a
}

// loadWatchedPackage loads the fixture into the app and starts the watcher
// on the unpacked folder of the package.
func loadWatchedPackage(t *testing.T, a *App) (string, *ddpackage.Package) {
	t.Helper()

	dir := testutil.DefaultFixture().Write(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	require.NotNil(t, a.packageWatcher, "the package watcher must be running")

	return dir, a.currentPackage()
}

// packageUpdates returns the number of the updates of the package that the
// filesystem watcher did.
func packageUpdates(a *App) int {
	a.packageUpdateLock.Lock()
	defer a.packageUpdateLock.Unlock()
	return a.packageUpdateCount
}

// waitForPackageUpdate waits until the filesystem watcher updated the
// package after the given number of updates.
func waitForPackageUpdate(t *testing.T, a *App, previous int) {
	t.Helper()

	require.Eventually(t, func() bool {
		return packageUpdates(a) > previous
	}, 10*time.Second, 20*time.Millisecond,
		"the filesystem watcher must update the package")
}

// waitForWatcherIdle waits until the filesystem watcher has nothing to
// report anymore and returns the number of the updates it did.
//
// The watcher waits for its debounce time, before it reports the changes, so
// the test waits for a few of them, to be sure that all the events of the
// changes arrived.
func waitForWatcherIdle(t *testing.T, a *App) int {
	t.Helper()

	last := -1
	stable := 0
	count := 0
	require.Eventually(t, func() bool {
		current := packageUpdates(a)
		count = current
		if current == last && len(a.packageWatcher.Pending()) == 0 {
			stable++
			return stable >= 3
		}
		last = current
		stable = 0
		return false
	}, 10*time.Second, 50*time.Millisecond,
		"the filesystem watcher must settle")

	return count
}

// inspectPackage runs fn with the package of the app, while the filesystem
// watcher does not update it, so that its state can be read safely.
func inspectPackage(t *testing.T, a *App, fn func(pkg *ddpackage.Package)) {
	t.Helper()

	a.packageUpdateLock.Lock()
	defer a.packageUpdateLock.Unlock()
	fn(a.currentPackage())
}

// waitForResource waits until the package knows the resource with the given
// res path.
func waitForResource(t *testing.T, a *App, resPath string) {
	t.Helper()

	require.Eventually(t, func() bool {
		_, err := a.currentPackage().GetResourceInfo(resPath)
		return err == nil
	}, 10*time.Second, 20*time.Millisecond,
		"the package must know the resource %s", resPath)
}

// writeTestTexture writes a small PNG with the given color to the path.
func writeTestTexture(t *testing.T, path string, c color.Color) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	fill := color.NRGBAModel.Convert(c).(color.NRGBA)
	for x := range 4 {
		for y := range 4 {
			img.SetNRGBA(x, y, fill)
		}
	}

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	require.NoError(t, png.Encode(file, img))
}

// fileMd5 returns the md5 hash of the content of the file.
func fileMd5(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

// resourceMd5 returns the md5 hash the package knows for the resource with
// the given res path.
func resourceMd5(t *testing.T, pkg *ddpackage.Package, resPath string) string {
	t.Helper()

	info, err := pkg.GetResourceInfo(resPath)
	require.NoError(t, err)

	hash := ""
	var hashErr error
	done := make(chan struct{})
	pkg.GetOrUpdateResourceMd5(info, func(md5 string, err error) {
		hash = md5
		hashErr = err
		close(done)
	})
	<-done
	require.NoError(t, hashErr)

	return hash
}

// TestPackageWatcherDetectsEditedResourceInNestedFolder verifies that a
// change of a resource in a deeply nested folder is detected and that the
// package is updated to reflect it.
func TestPackageWatcherDetectsEditedResourceInNestedFolder(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	nestedPath := filepath.Join(dir, "textures", "objects", "deep", "nested", "Bush.png")
	resPath := "res://packs/testpack/textures/objects/deep/nested/Bush.png"

	count := waitForWatcherIdle(t, a)
	writeTestTexture(t, nestedPath, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	waitForPackageUpdate(t, a, count)
	waitForResource(t, a, resPath)

	// the hash of the resource is computed, so that it is known before the
	// resource is changed
	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		require.Equal(t, fileMd5(t, nestedPath), resourceMd5(t, pkg, resPath))
	})

	// change the content of the resource
	count = waitForWatcherIdle(t, a)
	writeTestTexture(t, nestedPath, color.NRGBA{R: 40, G: 50, B: 60, A: 255})
	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		assert.Equal(t, fileMd5(t, nestedPath), resourceMd5(t, pkg, resPath),
			"the hash of the changed resource must be updated")
	})
}

// TestPackageWatcherDetectsEditedPackJSONInPackageRoot verifies that a
// change of the pack.json in the root of the package is detected and that
// the info of the package is reloaded from it.
func TestPackageWatcherDetectsEditedPackJSONInPackageRoot(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	packJSONPath := filepath.Join(dir, "pack.json")

	// the hash of the pack.json is computed, so that it is known before the
	// file is changed
	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		require.Equal(t, fileMd5(t, packJSONPath), resourceMd5(t, pkg, "res://packs/testpack.json"))
	})

	data, err := os.ReadFile(packJSONPath)
	require.NoError(t, err)
	updated := strings.Replace(string(data), `"version": "1.0.0"`, `"version": "2.0.0"`, 1)
	require.NotEqual(t, string(data), updated, "the version of the pack must be changed")
	require.NoError(t, os.WriteFile(packJSONPath, []byte(updated), 0o644))

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		assert.Equal(t, "2.0.0", pkg.Info().Version,
			"the info of the package must be reloaded from the pack.json")
		assert.Equal(t, fileMd5(t, packJSONPath), resourceMd5(t, pkg, "res://packs/testpack.json"),
			"the hash of the changed pack.json must be updated")
	})
}

// TestPackageWatcherDetectsAddedFile verifies that a file that was added to
// the package is detected and added to the package.
func TestPackageWatcherDetectsAddedFile(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	addedPath := filepath.Join(dir, "textures", "objects", "Bush.png")
	writeTestTexture(t, addedPath, color.NRGBA{R: 10, G: 20, B: 30, A: 255})

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		info, err := pkg.GetResourceInfo("res://packs/testpack/textures/objects/Bush.png")
		require.NoError(t, err, "the added texture must be part of the package")
		assert.Equal(t, addedPath, info.Path)
	})
}

// TestPackageWatcherDetectsRemovedFile verifies that a file that was removed
// from the package is detected and removed from the package.
func TestPackageWatcherDetectsRemovedFile(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	require.NoError(t, os.Remove(filepath.Join(dir, "textures", "objects", "Rock.png")))

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		_, err := pkg.GetResourceInfo("res://packs/testpack/textures/objects/Rock.png")
		assert.ErrorIs(t, err, ddpackage.ErrResourceNotFound,
			"the removed texture must not be part of the package anymore")
	})
}

// TestPackageWatcherDetectsRenamedFile verifies that a file that was moved
// or renamed is detected as a change of the package.
func TestPackageWatcherDetectsRenamedFile(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	renamedPath := filepath.Join(dir, "textures", "objects", "Stone.png")
	require.NoError(t, os.Rename(
		filepath.Join(dir, "textures", "objects", "Rock.png"),
		renamedPath,
	))

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		_, err := pkg.GetResourceInfo("res://packs/testpack/textures/objects/Rock.png")
		assert.ErrorIs(t, err, ddpackage.ErrResourceNotFound,
			"the renamed texture must not be part of the package anymore")

		info, err := pkg.GetResourceInfo("res://packs/testpack/textures/objects/Stone.png")
		require.NoError(t, err, "the renamed texture must be part of the package")
		assert.Equal(t, renamedPath, info.Path)
	})
}

// TestPackageWatcherDetectsEditedTagsFile verifies that a change of the tags
// file is detected and that the tags of the package are reloaded from it.
func TestPackageWatcherDetectsEditedTagsFile(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	tagsPath := filepath.Join(dir, "data", "default.dungeondraft_tags")
	tags := structures.NewPackageTags()
	tags.Tag("newtag", "textures/objects/Rock.png")
	data, err := json.MarshalIndent(tags, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tagsPath, data, 0o644))

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		assert.True(t, pkg.Tags().TagExists("newtag"),
			"the new tag must be loaded")
		assert.False(t, pkg.Tags().TagExists("rock"),
			"the tag that was removed from the file must not be loaded anymore")
	})
}

// TestPackageWatcherDetectsEditedWallMetadata verifies that a change of the
// metadata of a wall is detected and that the metadata of the package is
// reloaded from it.
func TestPackageWatcherDetectsEditedWallMetadata(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	wallPath := filepath.Join(dir, "data", "walls", "Stone.dungeondraft_wall")
	wall := structures.NewPackageWall()
	wall.Path = "textures/walls/Stone.png"
	wall.Color = ddcolor.Color{R: 255, G: 0, B: 0, A: 255}
	data, err := json.MarshalIndent(wall, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(wallPath, data, 0o644))

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		loaded, ok := pkg.Walls()["res://packs/testpack/data/walls/Stone.dungeondraft_wall"]
		require.True(t, ok, "the metadata of the wall must be loaded")
		assert.Equal(t, ddcolor.Color{R: 255, G: 0, B: 0, A: 255}, loaded.Color,
			"the color of the wall must be reloaded")
	})
}

// TestPackageWatcherDetectsEditedTilesetMetadata verifies that a change of
// the metadata of a tileset is detected and that the metadata of the package
// is reloaded from it.
func TestPackageWatcherDetectsEditedTilesetMetadata(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	tilesetPath := filepath.Join(dir, "data", "tilesets", "Grass.dungeondraft_tileset")
	tileset := structures.NewPackageTileset()
	tileset.Path = "textures/tilesets/Grass.png"
	tileset.Name = "Lush Grass"
	tileset.Type = structures.TilesetCustomColor
	tileset.Color = ddcolor.Color{R: 0, G: 255, B: 0, A: 255}
	data, err := json.MarshalIndent(tileset, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tilesetPath, data, 0o644))

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		loaded, ok := pkg.Tilesets()["res://packs/testpack/data/tilesets/Grass.dungeondraft_tileset"]
		require.True(t, ok, "the metadata of the tileset must be loaded")
		assert.Equal(t, "Lush Grass", loaded.Name,
			"the name of the tileset must be reloaded")
		assert.Equal(t, structures.TilesetCustomColor, loaded.Type,
			"the type of the tileset must be reloaded")
		assert.Equal(t, ddcolor.Color{R: 0, G: 255, B: 0, A: 255}, loaded.Color,
			"the color of the tileset must be reloaded")
	})
}

// TestPackageWatcherDetectsNewThumbnail verifies that a thumbnail that was
// added to the package is detected and that it is linked to its texture.
func TestPackageWatcherDetectsNewThumbnail(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	// the thumbnail of a texture is named after the hash of its res path
	hash := md5.Sum([]byte("res://packs/testpack/textures/objects/Rock.png"))
	name := hex.EncodeToString(hash[:]) + ".png"
	writeTestTexture(t, filepath.Join(dir, "thumbnails", name), color.NRGBA{R: 1, G: 2, B: 3, A: 255})

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		info, err := pkg.GetResourceInfo("res://packs/testpack/thumbnails/" + name)
		require.NoError(t, err, "the new thumbnail must be part of the package")
		assert.True(t, info.IsThumbnail(), "the new resource must be a thumbnail")
		assert.Equal(t, "res://packs/testpack/textures/objects/Rock.png", info.ThumbnailFor,
			"the new thumbnail must be linked to its texture")
	})
}

// TestPackageWatcherDetectsFileInNewFolder verifies that a folder that was
// created in the package is watched, so that the files that are added to it
// later are noticed.
func TestPackageWatcherDetectsFileInNewFolder(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	// a new folder in the package
	newFolder := filepath.Join(dir, "textures", "objects", "stones")
	require.NoError(t, os.MkdirAll(newFolder, 0o755))
	waitForPackageUpdate(t, a, count)

	// a file that is added to the new folder afterwards
	count = waitForWatcherIdle(t, a)
	writeTestTexture(t, filepath.Join(newFolder, "Granite.png"), color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		_, err := pkg.GetResourceInfo("res://packs/testpack/textures/objects/stones/Granite.png")
		assert.NoError(t, err,
			"the texture in the new folder must be part of the package")
	})
}

// TestPackageWatcherRescansThePackageAfterAnOverflow verifies that the
// package is scanned completely, when the watcher lost its events.
//
// A lot of changes at once can overflow the buffer of the operating system,
// so the events of some of the changes are never reported. The package has
// to be scanned completely, to find all the changes.
func TestPackageWatcherRescansThePackageAfterAnOverflow(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	// the changes are made while the watcher is paused, so that they are
	// not noticed by the watcher
	a.pausePackageWatcher()

	writeTestTexture(t, filepath.Join(dir, "textures", "objects", "Bush.png"), color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	require.NoError(t, os.Remove(filepath.Join(dir, "textures", "objects", "Rock.png")))

	// the events of the watcher were lost, so the queued changes are dropped
	a.packageWatcher.DiscardUnder("")

	// the watcher tells the app that it lost its events
	a.packageWatcher.overflow()

	a.resumePackageWatcher()

	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		_, err := pkg.GetResourceInfo("res://packs/testpack/textures/objects/Bush.png")
		assert.NoError(t, err,
			"the added texture must be found by the rescan of the package")
		_, err = pkg.GetResourceInfo("res://packs/testpack/textures/objects/Rock.png")
		assert.ErrorIs(t, err, ddpackage.ErrResourceNotFound,
			"the removed texture must be gone after the rescan of the package")
	})
}

// TestPackageWatcherDoesNotReportWhilePaused verifies that the watcher does
// not report the changes while it is paused and that it picks them up after
// it is resumed.
func TestPackageWatcherDoesNotReportWhilePaused(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	a.pausePackageWatcher()
	writeTestTexture(t, filepath.Join(dir, "textures", "objects", "Bush.png"), color.NRGBA{R: 10, G: 20, B: 30, A: 255})

	// the watcher must not report the change while it is paused
	time.Sleep(4 * watcherTestDebounce)
	assert.Equal(t, count, packageUpdates(a),
		"the paused watcher must not report the changes")

	// the change must be reported after the watcher is resumed
	a.resumePackageWatcher()
	waitForPackageUpdate(t, a, count)

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		_, err := pkg.GetResourceInfo("res://packs/testpack/textures/objects/Bush.png")
		assert.NoError(t, err,
			"the change must be picked up after the watcher is resumed")
	})
}

// TestThumbnailGenerationUpdatesThePackageWithoutSpammingTheWatcher
// verifies that the generation of the thumbnails updates the package and
// that the watcher is not flooded with the events of the generated files.
func TestThumbnailGenerationUpdatesThePackageWithoutSpammingTheWatcher(t *testing.T) {
	a := newWatcherApp(t)
	_, _ = loadWatchedPackage(t, a)

	count := waitForWatcherIdle(t, a)

	a.genThumbnails()

	// the package must know the thumbnails of its textures
	require.Eventually(t, func() bool {
		return thumbnailCount(a) == 4
	}, 10*time.Second, 50*time.Millisecond,
		"the generated thumbnails must be part of the package")

	// the watcher must not report the changes of the generation one by one
	updated := waitForWatcherIdle(t, a)
	assert.LessOrEqual(t, updated-count, 1,
		"the watcher must not report the changes of the thumbnail generation one by one")

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		for _, fi := range pkg.FileList().AsSlice() {
			if !fi.IsTexture() {
				continue
			}
			_, err := pkg.GetResourceInfo(fi.ThumbnailResPath)
			assert.NoError(t, err,
				"the thumbnail of %s must be part of the package", fi.RelPath)
		}
	})
}

// thumbnailCount returns the number of the thumbnails the package knows.
func thumbnailCount(a *App) int {
	count := 0
	for _, fi := range a.currentPackage().FileList().AsSlice() {
		if fi.IsThumbnail() {
			count++
		}
	}
	return count
}

// TestTagGenerationUpdatesTheTagsWithoutSpammingTheWatcher verifies that the
// generation of the tags updates the tags of the package and that the
// watcher is not flooded with the events of the generated tags.
func TestTagGenerationUpdatesTheTagsWithoutSpammingTheWatcher(t *testing.T) {
	a := newWatcherApp(t)
	dir, _ := loadWatchedPackage(t, a)

	// a resource in a subfolder, so that the tag generator finds a tag for it
	nestedPath := filepath.Join(dir, "textures", "objects", "nature", "Bush.png")
	resPath := "res://packs/testpack/textures/objects/nature/Bush.png"

	count := waitForWatcherIdle(t, a)
	writeTestTexture(t, nestedPath, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	waitForPackageUpdate(t, a, count)
	waitForResource(t, a, resPath)

	count = waitForWatcherIdle(t, a)

	generator := ddpackage.NewGenerateTags(&ddpackage.GenerateTagsOptions{
		BuildTagSetsFromPrefix: true,
	})
	a.generateTags(generator, nil)

	// the watcher must not report the change of the tags file more than once
	updated := waitForWatcherIdle(t, a)
	assert.LessOrEqual(t, updated-count, 1,
		"the watcher must not report the change of the tags file more than once")

	inspectPackage(t, a, func(pkg *ddpackage.Package) {
		assert.True(t, pkg.Tags().TagExists("nature"),
			"the generated tag must be part of the package")
	})

	// the tags must be saved to the tags file of the package
	tagsPath := filepath.Join(dir, "data", "default.dungeondraft_tags")
	data, err := os.ReadFile(tagsPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "nature",
		"the generated tags must be written to the tags file")
}
