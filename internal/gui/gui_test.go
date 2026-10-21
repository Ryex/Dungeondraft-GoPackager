package gui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/resource_tree"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	ddcolor "github.com/ryex/dungeondraft-gopackager/pkg/structures/color"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.ErrorLevel)
	lang.SetupLang("en")
	testutil.SetupFyneApp()
	os.Exit(m.Run())
}

// newTestApp creates an App backed by a software rendered Fyne application.
func newTestApp(t *testing.T) *App {
	t.Helper()

	a := NewAppWithFyneApp(test.NewApp())
	// the filesystem watcher would keep running past the end of the test
	a.packageWatcherDisabled = true
	a.setup()
	t.Cleanup(a.clean)

	return a
}

// loadFixturePackage loads a pack fixture and returns its folder and package.
func loadFixturePackage(t *testing.T, a *App) (string, *ddpackage.Package) {
	t.Helper()

	dir := testutil.DefaultFixture().Write(t)

	result, err := a.loadUnpackedSync(dir, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Pkg)

	return dir, result.Pkg
}

// childObjects returns the direct children of an object, if it has any.
func childObjects(obj fyne.CanvasObject) []fyne.CanvasObject {
	return named.Children(obj)
}

// collectObjects walks an object tree and returns every object of type T.
func collectObjects[T fyne.CanvasObject](obj fyne.CanvasObject) []T {
	var found []T

	if typed, ok := obj.(T); ok {
		found = append(found, typed)
	}

	for _, child := range childObjects(obj) {
		found = append(found, collectObjects[T](child)...)
	}

	return found
}

// collectTexts returns the text of every label in an object tree.
func collectTexts(obj fyne.CanvasObject) []string {
	var texts []string

	switch o := obj.(type) {
	case *widget.Label:
		texts = append(texts, o.Text)
	case *widgets.ThemedText:
		texts = append(texts, o.Text.Text)
	}

	for _, child := range childObjects(obj) {
		texts = append(texts, collectTexts(child)...)
	}

	return texts
}

func TestAppSetupBuildsTheMainWindow(t *testing.T) {
	a := newTestApp(t)

	require.NotNil(t, a.window)
	require.NotNil(t, a.window.Content())
	require.NotNil(t, a.mainContentContainer)
	assert.NotEmpty(t, a.mainContentContainer.Objects)
}

func TestAppLoadUnpackedSyncLoadsThePackage(t *testing.T) {
	a := newTestApp(t)
	dir := testutil.DefaultFixture().Write(t)

	result, err := a.loadUnpackedSync(dir, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Pkg)
	require.NoError(t, result.TagsErr)
	require.NoError(t, result.MetadataErr)

	pkg := result.Pkg
	assert.Equal(t, "testpack", pkg.ID())
	assert.Equal(t, "Test Pack", pkg.Name())
	assert.NotEmpty(t, pkg.FileList().AsSlice())
	assert.True(t, pkg.Tags().TagExists("nature"), "the tags of the pack must be loaded")
	assert.NotEmpty(t, pkg.Walls(), "the wall metadata must be loaded")
	assert.NotEmpty(t, pkg.Tilesets(), "the tileset metadata must be loaded")
}

func TestAppLoadUnpackedSyncFailsWithoutPackJSON(t *testing.T) {
	a := newTestApp(t)

	_, err := a.loadUnpackedSync(t.TempDir(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errUnpackedFolder)
}

func TestAppSetUnpackedContentShowsTheResourceTree(t *testing.T) {
	a := newTestApp(t)
	_, pkg := loadFixturePackage(t, a)

	a.setUnpackedContent(pkg)

	trees := collectObjects[*resource_tree.ResourceTree](a.mainContentContainer)
	require.Len(t, trees, 1, "the unpacked view must show the resource tree")
	assert.Equal(t, resource_tree.ResourceTreePathMode, trees[0].GetMode())
}

func TestAppSetNotAPackageContent(t *testing.T) {
	a := newTestApp(t)
	dir := t.TempDir()

	a.setNotAPackageContent(dir)

	texts := collectTexts(a.mainContentContainer)
	assert.Contains(t, texts, "This does not appear to be a resource pack.")
}

func TestAppSetErrContent(t *testing.T) {
	a := newTestApp(t)

	a.setErrContent("Something went wrong", errors.New("boom"))

	texts := collectTexts(a.mainContentContainer)
	assert.Contains(t, texts, "Something went wrong")
	assert.Contains(t, texts, "1) boom")
}

func TestAppSaveUnpackedTagsWritesToDisk(t *testing.T) {
	a := newTestApp(t)
	dir, pkg := loadFixturePackage(t, a)
	a.setUnpackedContent(pkg)

	tagsPath := filepath.Join(dir, "data", "default.dungeondraft_tags")

	a.pkg.Tags().Tag("newtag", "textures/objects/Rock.png")
	a.saveUnpackedTags()

	// the save is debounced, so wait for the file to be written
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(tagsPath)
		return err == nil && strings.Contains(string(data), "newtag")
	}, 5*time.Second, 20*time.Millisecond, "the tags must be written to the tags file")
}

func TestAppAddTagThroughTheTagPane(t *testing.T) {
	a := newTestApp(t)
	dir, pkg := loadFixturePackage(t, a)
	a.setUnpackedContent(pkg)

	info, err := a.pkg.GetResourceInfo("res://packs/testpack/textures/objects/Rock.png")
	require.NoError(t, err)

	pane := a.buildTagInfo(info, true)

	selectEntries := collectObjects[*widget.SelectEntry](pane)
	require.Len(t, selectEntries, 1, "the tag pane must offer a tag selector")
	selectEntries[0].SetText("addedtag")

	var addButton *widget.Button
	for _, button := range collectObjects[*widget.Button](pane) {
		if button.Text == lang.X("tags.addBtn.text", "Add") {
			addButton = button
			break
		}
	}
	require.NotNil(t, addButton, "the tag pane must offer an add button")

	test.Tap(addButton)

	tagsPath := filepath.Join(dir, "data", "default.dungeondraft_tags")
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(tagsPath)
		return err == nil && strings.Contains(string(data), "addedtag")
	}, 5*time.Second, 20*time.Millisecond, "the new tag must be written to the tags file")
}

func TestAppSaveWallMetadataWritesToDisk(t *testing.T) {
	a := newTestApp(t)
	dir, pkg := loadFixturePackage(t, a)
	a.setUnpackedContent(pkg)

	metaPath := "res://packs/testpack/data/walls/Stone.dungeondraft_wall"

	wall, ok := a.pkg.Walls()[metaPath]
	require.True(t, ok, "the wall metadata must be loaded")

	wall.Color = ddcolor.Color{R: 255, G: 0, B: 0, A: 255}
	a.pkg.Walls()[metaPath] = wall

	require.NoError(t, a.pkg.SaveUnpackedWall(metaPath))

	data, err := os.ReadFile(filepath.Join(dir, "data", "walls", "Stone.dungeondraft_wall"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "ff0000", "the new color must be written to the wall metadata")
}

func TestAppPackageWatcherIsNotStartedWhenDisabled(t *testing.T) {
	a := newTestApp(t)
	_, pkg := loadFixturePackage(t, a)
	a.setUnpackedContent(pkg)

	a.setupPackageWatcher()

	assert.Nil(t, a.packageWatcher, "the watcher must not be started when it is disabled")
}
