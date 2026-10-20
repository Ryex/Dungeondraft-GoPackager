package resource_tree

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage/filter"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.ErrorLevel)
	lang.SetupLang("en")
	testutil.SetupFyneApp()
	os.Exit(m.Run())
}

// loadFixture writes a pack fixture to disk and loads it as a package.
func loadFixture(t *testing.T) (*ddpackage.Package, string) {
	t.Helper()

	dir := testutil.DefaultFixture().Write(t)

	pkg := ddpackage.NewPackage(log.WithField("test", t.Name()))
	require.NoError(t, pkg.LoadUnpackedFromFolder(dir))
	require.Empty(t, pkg.BuildFileList())
	require.NoError(t, pkg.LoadTags())
	require.NoError(t, pkg.LoadResourceMetadata())

	t.Cleanup(pkg.Close)

	return pkg, dir
}

// newTree builds a resource tree for the given package.
func newTree(t *testing.T, pkg *ddpackage.Package) *ResourceTree {
	t.Helper()

	tree := NewResourceTree(pkg, test.NewWindow(nil))
	tree.RebuildTree()
	return tree
}

func TestResourceTreePathModeStructure(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	assert.Equal(t, ResourceTreePathMode, tree.GetMode())

	// the package root hangs off the data tree root
	assert.Equal(t, []widget.TreeNodeID{PackageTreeRootID}, tree.ChildUIDs(binding.DataTreeRootID))

	rootChildren := tree.ChildUIDs(PackageTreeRootID)
	assert.Contains(t, rootChildren, widget.TreeNodeID("dir://textures"))
	assert.Contains(t, rootChildren, widget.TreeNodeID("dir://data"))

	textures := tree.ChildUIDs("dir://textures")
	assert.Contains(t, textures, widget.TreeNodeID("dir://textures/objects"))
	assert.Contains(t, textures, widget.TreeNodeID("dir://textures/walls"))
	assert.Contains(t, textures, widget.TreeNodeID("dir://textures/tilesets"))

	objects := tree.ChildUIDs("dir://textures/objects")
	assert.Len(t, objects, 2)
	assert.Contains(t, objects, widget.TreeNodeID("res://packs/testpack/textures/objects/Rock.png"))
	assert.Contains(t, objects, widget.TreeNodeID("res://packs/testpack/textures/objects/Tree.png"))
}

func TestResourceTreeDataForResource(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	resPath := widget.TreeNodeID("res://packs/testpack/textures/objects/Rock.png")

	data := tree.DataFor(resPath)
	require.NotNil(t, data)

	node, ok := data.(*ResourceFileInfoNode)
	require.True(t, ok, "a resource leaf must hold a ResourceFileInfoNode")
	require.NotNil(t, node.Info)
	assert.Equal(t, resPath, node.Info.ResPath)
	assert.Equal(t, "textures/objects/Rock.png", node.Info.RelPath)
}

func TestResourceTreeDistinguishesBranchesAndLeaves(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	assert.True(t, tree.IsBranch("dir://textures"))
	assert.True(t, tree.IsBranch(PackageTreeRootID))
	assert.False(t, tree.IsBranch("res://packs/testpack/textures/objects/Rock.png"))
}

func TestResourceTreeDirectoryBranchLabels(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	data := tree.DataFor("dir://textures/objects")
	require.NotNil(t, data)

	branch, ok := data.(*ResourceDirBranch)
	require.True(t, ok)
	assert.Equal(t, "objects", branch.Dir)
}

func TestResourceTreeTagMode(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	tree.SetMode(ResourceTreeTagMode)
	assert.Equal(t, ResourceTreeTagMode, tree.GetMode())

	rootChildren := tree.ChildUIDs(PackageTreeRootID)
	assert.Contains(t, rootChildren, widget.TreeNodeID("tag://nature"))
	assert.Contains(t, rootChildren, widget.TreeNodeID("tag://rock"))

	nature := tree.ChildUIDs("tag://nature")
	assert.Contains(t, nature, widget.TreeNodeID("res://packs/testpack/textures/objects/Tree.png"))

	tagData := tree.DataFor("tag://nature")
	require.NotNil(t, tagData)

	branch, ok := tagData.(*ResourceTagBranch)
	require.True(t, ok)
	assert.Equal(t, "nature", branch.Tag)
	assert.False(t, branch.Untagged)

	// a tagged resource must still expose its file info so that the
	// application can show the resource details
	resData := tree.DataFor("res://packs/testpack/textures/objects/Tree.png")
	require.NotNil(t, resData)

	resNode, ok := resData.(*ResourceFileInfoNode)
	require.True(t, ok)
	require.NotNil(t, resNode.Info)
	assert.Equal(t, "textures/objects/Tree.png", resNode.Info.RelPath)
}

func TestResourceTreeFilterNarrowsTheTree(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	tree.SetFilter(filter.ResourceFilterIsObject)

	rootChildren := tree.ChildUIDs(PackageTreeRootID)
	assert.NotContains(t, rootChildren, widget.TreeNodeID("dir://data"),
		"filtered out resources must not produce branches")

	objects := tree.ChildUIDs("dir://textures/objects")
	assert.Len(t, objects, 2, "only the objects must remain")
	assert.NotContains(t, objects, widget.TreeNodeID("res://packs/testpack/textures/walls/Stone.png"))
}

func TestResourceTreeFilterCanBeCleared(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	tree.SetFilter(filter.ResourceFilterIsObject)
	tree.SetFilter(filter.ResourceFilterEmpty{})

	rootChildren := tree.ChildUIDs(PackageTreeRootID)
	assert.Contains(t, rootChildren, widget.TreeNodeID("dir://data"))
	assert.Contains(t, rootChildren, widget.TreeNodeID("dir://textures"))
}

func TestResourceTreeSelection(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	changes := 0
	tree.OnSelectionChanged = func() {
		changes++
	}

	resPath := widget.TreeNodeID("res://packs/testpack/textures/objects/Rock.png")
	tree.Select(resPath)

	assert.True(t, tree.IsSelected(resPath))
	assert.Equal(t, 1, changes)

	data := tree.GetSelectionAndChildData()
	require.Len(t, data, 1)

	node, ok := data[0].(*ResourceFileInfoNode)
	require.True(t, ok)
	assert.Equal(t, resPath, node.Info.ResPath)

	tree.Unselect(resPath)
	assert.False(t, tree.IsSelected(resPath))
}

func TestResourceTreeEmptyPackage(t *testing.T) {
	dir := t.TempDir()
	testutil.DefaultFixture().WriteTo(t, dir)

	// remove every resource so that the package has nothing but its pack.json
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "textures")))
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "data")))

	pkg := ddpackage.NewPackage(log.WithField("test", t.Name()))
	require.NoError(t, pkg.LoadUnpackedFromFolder(dir))
	require.Empty(t, pkg.BuildFileList())
	t.Cleanup(pkg.Close)

	tree := newTree(t, pkg)

	// pack.json is always part of the file list, so the package root is
	// always shown even when the package holds no resources
	assert.Equal(t, []widget.TreeNodeID{PackageTreeRootID}, tree.ChildUIDs(binding.DataTreeRootID))
	assert.Contains(t, tree.ChildUIDs(PackageTreeRootID), widget.TreeNodeID("res://packs/testpack.json"))
}
