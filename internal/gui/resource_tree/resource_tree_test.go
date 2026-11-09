package resource_tree

import (
	"fmt"
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

	return loadFixtureWith(t, testutil.DefaultFixture())
}

// loadFixtureWith writes the given pack fixture to disk and loads it as a
// package.
func loadFixtureWith(t *testing.T, fixture *testutil.Fixture) (*ddpackage.Package, string) {
	t.Helper()

	dir := fixture.Write(t)

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
	assert.Contains(t, nature, tagResourceUID("tag://nature", "res://packs/testpack/textures/objects/Tree.png"))

	tagData := tree.DataFor("tag://nature")
	require.NotNil(t, tagData)

	branch, ok := tagData.(*ResourceTagBranch)
	require.True(t, ok)
	assert.Equal(t, "nature", branch.Tag)
	assert.False(t, branch.Untagged)

	// a tagged resource must still expose its file info so that the
	// application can show the resource details
	resData := tree.DataFor(tagResourceUID("tag://nature", "res://packs/testpack/textures/objects/Tree.png"))
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

// TestGetSelectedResourcesReportsDirectlySelectedResources guards the
// Selected flag of GetSelectedResources: a resource is selected on its
// own when one of the nodes that show it is selected, not only when the
// uid of the reported entry is selected.
//
// The tree is in tag mode and the resource carries two tags, so it is
// shown under both branches: the walk sees the branch that is selected
// for roll-up before it sees the node that was selected directly, and the
// entry is therefore created from an occurrence that is not the selected
// one.
func TestGetSelectedResourcesReportsDirectlySelectedResources(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)
	tree.SetMode(ResourceTreeTagMode)

	// Rock.png carries the "nature" and the "rock" tag, so it is shown
	// under both branches
	natureRockUID := tagResourceUID("tag://nature", rockResPath)
	rockBranchUID := widget.TreeNodeID("tag://rock")
	natureBranchUID := widget.TreeNodeID("tag://nature")

	entryFor := func(resources []SelectedResource, resPath string) SelectedResource {
		for _, res := range resources {
			if res.Info != nil && res.Info.ResPath == resPath {
				return res
			}
		}
		require.FailNowf(t, "resource not reported", "%s must be reported", resPath)
		return SelectedResource{}
	}

	// Rock is selected on its own under one tag while the branch of the
	// other tag is selected for roll-up: the branch is popped first by the
	// LIFO stack, so the uid that is seen first is the branch child and
	// not the selected one
	tree.Select(natureRockUID)
	tree.Select(rockBranchUID)

	resources := tree.GetSelectedResources()
	rock := entryFor(resources, string(rockResPath))
	assert.True(t, rock.Selected, "Rock.png is selected on its own")

	// only the branch is selected: Rock.png is part of the selection
	// through its branch, but not selected on its own
	tree.UnselectAll()
	tree.Select(rockBranchUID)

	resources = tree.GetSelectedResources()
	rock = entryFor(resources, string(rockResPath))
	assert.False(t, rock.Selected, "Rock.png must not be selected on its own")

	// a resource with no selection of its own under another selected
	// branch must not be reported as selected either
	tree.UnselectAll()
	tree.Select(rockBranchUID)
	tree.Select(natureBranchUID)

	resources = tree.GetSelectedResources()
	rock = entryFor(resources, string(rockResPath))
	assert.False(t, rock.Selected, "Rock.png must not be selected on its own")
	treeRes := entryFor(resources, "res://packs/testpack/textures/objects/Tree.png")
	assert.False(t, treeRes.Selected, "Tree.png must not be selected on its own")
}

// TestUnselectIncludingAncestorsUnselectsTheSelectedBranch guards the
// ancestor walk of UnselectIncludingAncestors: a node below a branch that is
// selected for roll-up is shown as selected but is not selected itself, and
// the selection can not hold "the branch selected except this one child" -
// unselecting the node therefore has to unselect the branch that holds it.
//
// The context menu of such a roll-up selected node offers to be unselected
// as well, so it has to show the unselect item.
func TestUnselectIncludingAncestorsUnselectsTheSelectedBranch(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)
	tree.SetMode(ResourceTreeTagMode)

	tagBranchUID := widget.TreeNodeID("tag://nature")
	childUID := tagResourceUID("tag://nature", rockResPath)

	// the branch is selected, so the nodes below it are shown as selected
	// without being selected themselves
	tree.Select(tagBranchUID)
	require.True(t, tree.IsVisiblySelected(childUID),
		"the node below a selected branch must be shown as selected")
	require.False(t, tree.IsSelected(childUID),
		"the node below a selected branch must not be selected itself")

	// the context menu of the roll-up selected node offers to be unselected
	nodes := treeNodes(t, tree, "tag://nature")
	childNode, ok := nodes[childUID]
	require.True(t, ok, "the tree must hold a node for the resource")
	items := childNode.contextMenuItems()
	require.Len(t, items, 1)
	assert.Equal(t, lang.X("tree.context.unselect", "Unselect"), items[0].Label)

	tree.UnselectIncludingAncestors(childUID)

	assert.False(t, tree.IsSelected(tagBranchUID),
		"the selected ancestor must be unselected")
	assert.False(t, tree.IsVisiblySelected(childUID),
		"the node must not be shown as selected any more")
	assert.Empty(t, tree.GetSelection())
}

// TestSelectionAtScale guards the selection bookkeeping at the scale of a
// real pack: every resource is shown under every one of its tags, so the
// tree holds far more nodes than there are resources.
//
// It deliberately makes no assertion about how long anything takes - the
// review warned that wall-clock assertions flake.
func TestSelectionAtScale(t *testing.T) {
	const (
		resourceCount = 500
		tagCount      = 3
	)

	// build a package with resourceCount objects in a temporary directory
	names := make([]string, resourceCount)
	for i := range names {
		names[i] = fmt.Sprintf("Res%04d", i)
	}
	fixture := &testutil.Fixture{
		ID:      "scalepack",
		Name:    "Scale Pack",
		Objects: names,
	}
	dir := fixture.Write(t)

	pkg := ddpackage.NewPackage(log.WithField("test", t.Name()))
	require.NoError(t, pkg.LoadUnpackedFromFolder(dir))
	require.Empty(t, pkg.BuildFileList())
	require.NoError(t, pkg.LoadTags())
	t.Cleanup(pkg.Close)

	// objects are the only taggable resources, so these are all of them
	var relPaths []string
	var resPaths []string
	for _, fi := range pkg.FileList().AsSlice() {
		if fi.IsObject() {
			relPaths = append(relPaths, fi.RelPath)
			resPaths = append(resPaths, fi.ResPath)
		}
	}
	require.Len(t, relPaths, resourceCount)

	// tag every resource with every tag
	tags := make([]string, tagCount)
	for i := range tags {
		tags[i] = fmt.Sprintf("tag%d", i)
		pkg.Tags().Tag(tags[i], relPaths...)
	}

	tree := newTree(t, pkg)
	tree.SetMode(ResourceTreeTagMode)
	tree.RebuildTree()

	// select every tag branch
	for _, tag := range tags {
		tree.Select(widget.TreeNodeID("tag://" + tag))
	}

	// every resource is reported exactly once, no matter how often it is
	// shown
	resources := tree.GetSelectedResources()
	require.Len(t, resources, resourceCount,
		"every resource must be reported exactly once")
	seen := make(map[string]bool, resourceCount)
	for _, res := range resources {
		require.NotNil(t, res.Info)
		require.Falsef(t, seen[res.Info.ResPath],
			"resource %s must be reported once", res.Info.ResPath)
		seen[res.Info.ResPath] = true
	}

	// every occurrence of every resource is shown as selected
	for _, tag := range tags {
		for i, resPath := range resPaths {
			uid := tagResourceUID("tag://"+tag, resPath)
			require.Truef(t, tree.IsVisiblySelected(uid),
				"occurrence %s of %s must be shown as selected", uid, relPaths[i])
		}
	}

	// an occurrence resolves to the tag branch it is shown under
	sample := tagResourceUID("tag://"+tags[0], resPaths[0])
	parent := tree.ParentOf(sample)
	require.NotNilf(t, parent, "occurrence %s must have a parent", sample)
	assert.Equal(t, widget.TreeNodeID("tag://"+tags[0]), *parent,
		"the occurrence must be parented by its tag branch")

	// the indexes hold the same answers as the lookups above
	require.NotNil(t, tree.parentOf, "the parent index must be built")
	require.NotNil(t, tree.visibleSet, "the visible selection index must be built")
	assert.Equal(t, widget.TreeNodeID("tag://"+tags[0]), tree.parentOf[sample],
		"the parent index must hold the tag branch of the occurrence")
	assert.True(t, tree.visibleSet[sample],
		"the visible selection index must hold the occurrence")
}
