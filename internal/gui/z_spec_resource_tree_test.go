package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/resource_tree"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
)

// resPath returns the resource path of a file of the test fixture.
func resPath(relPath string) string {
	return "res://packs/" + testutil.DefaultFixture().ID + "/" + relPath
}

// resourceTree returns the resource tree of the loaded package.
func resourceTree(t *testing.T, a *App) *resource_tree.ResourceTree {
	t.Helper()

	tree, ok := namedPart(t, a, nameResourceTree).(*resource_tree.ResourceTree)
	if !ok {
		t.Fatal("the resource tree of the interface is not a resource tree")
	}
	return tree
}

// filterEntry returns the filter entry of the resource tree.
func filterEntry(t *testing.T, a *App) *widgets.ToolTipEntry {
	t.Helper()

	entry, ok := namedPart(t, a, nameFilterEntry).(*widgets.ToolTipEntry)
	if !ok {
		t.Fatal("the filter entry of the interface is not a filter entry")
	}
	return entry
}

// shownResources returns the paths of the resources the resource tree shows.
//
// A resource that is shown under more than one tag is reported once.
func shownResources(t *testing.T, tree *resource_tree.ResourceTree) []string {
	t.Helper()

	var paths []string
	seen := map[resource_tree.ResourceTreeNodeData]bool{}
	var walk func(uid widget.TreeNodeID)
	walk = func(uid widget.TreeNodeID) {
		for _, child := range tree.ChildUIDs(uid) {
			data := tree.DataFor(child)
			if node, isResource := data.(*resource_tree.ResourceFileInfoNode); isResource && node.Info != nil {
				if !seen[data] {
					seen[data] = true
					paths = append(paths, node.Info.ResPath)
				}
				continue
			}
			walk(child)
		}
	}
	walk(resource_tree.PackageTreeRootID)

	return paths
}

// selectResource selects a single resource in the resource tree, the way a
// click on it does.
func selectResource(t *testing.T, a *App, tree *resource_tree.ResourceTree, path string) {
	t.Helper()

	tree.UnselectAll()
	tree.Select(path)

	// the information of the resource is built in the background, e.g. the
	// md5 of the resource is computed there
	waitForBackgroundWork(t, a)
}

// TestSpecResourceSelection verifies that selecting a resource in the resource
// tree shows the information of the resource in the resource pane.
//
// Covered by this test:
//
//	when a resource is selected in the resource tree
//	  - the resource pane shows the information of the resource
//	    - with a file preview tab
//	      - an image resource shows the image
//	        - and the thumbnail toggle
//	      - a text resource shows the text
//	        - unless the file is too big, then the text is not shown
//	    - with a tag tab when the resource is taggable
//	    - with a metadata tab when the resource has metadata
//	when more than one resource is selected
//	  - the resource pane shows the bulk operations pane
func TestSpecResourceSelection(t *testing.T) {
	// the packed fixture is used, so that the md5 of the resources is known
	// and no hash is computed in the background while the interface is
	// inspected
	dir := packFixtureDir(t)

	// a text file that is too big to be shown in the preview
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "data", "big.json"),
		[]byte(strings.Repeat("a line of text\n", 300)),
		0o644,
	))

	packFile := packFixture(t, dir)

	a := newSpecApp(t)
	a.loadPack(packFile)
	waitForPackLoad(t, a)

	tree := resourceTree(t, a)

	// as long as nothing is selected, the resource pane shows the default
	// preview
	uitest.Expect(t, uitest.In(namedPart(t, a, nameResourcePane)),
		uitest.Has("the default preview",
			uitest.Label(lang.X("preview.defaultText", "Select a resource")),
		),
	)

	// an image resource shows the image and the thumbnail toggle
	selectResource(t, a, tree, resPath("textures/objects/Rock.png"))

	uitest.Expect(t, uitest.In(namedPart(t, a, nameResourcePane)),
		uitest.Group("when an image resource is selected",
			uitest.Has("a file preview tab",
				uitest.Tab(lang.X("preview.tab.resource", "Resource")),
				uitest.Has("the image preview", uitest.Named(nameImagePreview)),
				uitest.Has("the thumbnail toggle", uitest.Named(nameThumbnailToggle)).Visible(),
			),
			uitest.Has("a tag tab", uitest.Tab(lang.X("preview.tab.tags", "Tags"))),
			uitest.NotExists("a metadata tab",
				uitest.Tab(lang.X("preview.tab.metadata", "Settings")),
			),
		),
	)

	// a resource that is not taggable has no tag tab
	selectResource(t, a, tree, resPath("textures/walls/Stone.png"))

	uitest.Expect(t, uitest.In(namedPart(t, a, nameResourcePane)),
		uitest.Group("when a wall is selected",
			uitest.Has("a file preview tab",
				uitest.Tab(lang.X("preview.tab.resource", "Resource")),
			),
			uitest.NotExists("a tag tab", uitest.Tab(lang.X("preview.tab.tags", "Tags"))),
			uitest.Has("a metadata tab",
				uitest.Tab(lang.X("preview.tab.metadata", "Settings")),
				uitest.Has("the metadata form", uitest.Named(nameMetadataForm)),
			),
		),
	)

	// a text resource shows the text of the file
	selectResource(t, a, tree, resPath("pack.json"))

	uitest.Expect(t, uitest.In(namedPart(t, a, nameResourcePane)),
		uitest.Group("when a text resource is selected",
			uitest.Has("a file preview tab",
				uitest.Tab(lang.X("preview.tab.resource", "Resource")),
				uitest.Has("the text preview", uitest.Named(nameTextPreview)),
				uitest.NotExists("the too large label", uitest.Named(nameTooLargeLabel)),
				uitest.NotExists("the thumbnail toggle", uitest.Named(nameThumbnailToggle)),
			),
		),
	)

	// a text resource that is too big is not shown
	selectResource(t, a, tree, resPath("data/big.json"))

	uitest.Expect(t, uitest.In(namedPart(t, a, nameResourcePane)),
		uitest.Group("when a text resource that is too big is selected",
			uitest.Has("a file preview tab",
				uitest.Tab(lang.X("preview.tab.resource", "Resource")),
				uitest.Has("the too large label", uitest.Named(nameTooLargeLabel)),
				uitest.NotExists("the text preview", uitest.Named(nameTextPreview)),
			),
		),
	)

	// selecting more than one resource shows the bulk operations pane
	tree.Select(resPath("textures/objects/Rock.png"))
	tree.Select(resPath("textures/objects/Tree.png"))
	waitForBackgroundWork(t, a)

	uitest.Expect(t, uitest.In(namedPart(t, a, nameResourcePane)),
		uitest.Group("when more than one resource is selected",
			uitest.Has("the bulk operations pane", uitest.Named(nameBulkOpsPane)),
			uitest.NotExists("the info tabs", uitest.Named(nameInfoTabs)),
		),
	)
}

// TestSpecResourceFilter verifies that the filter of the resource tree
// changes the resources the tree shows.
//
// Covered by this test:
//
//	when the filter of the resource tree is changed
//	  - a name filter shows the resources with a matching name only
//	  - a tag filter shows the resources with a matching tag only
//	  - a path filter shows the resources with a matching path only
//	  - a kind filter shows the resources of that kind only
//	  - an empty filter shows all the resources again
func TestSpecResourceFilter(t *testing.T) {
	a := newSpecApp(t)
	dir := packFixtureDir(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	tree := resourceTree(t, a)
	entry := filterEntry(t, a)

	all := shownResources(t, tree)
	require.Len(t, all, 9, "the tree shows all the resources of the pack")

	// a name filter shows the resources with a matching name only
	entry.SetText("%Rock")
	assert.Equal(t, []string{resPath("textures/objects/Rock.png")}, shownResources(t, tree))

	// a tag filter shows the resources with a matching tag only
	entry.SetText("#rock")
	assert.Equal(t, []string{resPath("textures/objects/Rock.png")}, shownResources(t, tree))

	// a path filter shows the resources with a matching path only
	entry.SetText("/textures/objects/**")
	assert.ElementsMatch(t, []string{
		resPath("textures/objects/Rock.png"),
		resPath("textures/objects/Tree.png"),
	}, shownResources(t, tree))

	// a kind filter shows the resources of that kind only
	entry.SetText("@IsTexture")
	assert.ElementsMatch(t, []string{
		resPath("textures/objects/Rock.png"),
		resPath("textures/objects/Tree.png"),
		resPath("textures/tilesets/Grass.png"),
		resPath("textures/walls/Stone.png"),
	}, shownResources(t, tree))

	// an empty filter shows all the resources again
	entry.SetText("")
	assert.ElementsMatch(t, all, shownResources(t, tree))
}
