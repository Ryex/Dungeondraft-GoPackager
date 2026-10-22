package gui

import (
	"crypto/md5"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/resource_tree"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

// bulkResourceList returns the list of the resources that are selected in the
// resource tree.
func bulkResourceList(t *testing.T, a *App) *widget.List {
	t.Helper()

	list, ok := namedPart(t, a, nameBulkResourceList).(*widget.List)
	if !ok {
		t.Fatal("the list of the selected resources is not a list")
	}
	return list
}

// bulkTagList returns the list of the tags that all the selected resources
// have in common.
func bulkTagList(t *testing.T, a *App) *widget.List {
	t.Helper()

	list, ok := namedPart(t, a, nameBulkTagList).(*widget.List)
	if !ok {
		t.Fatal("the list of the common tags is not a list")
	}
	return list
}

// computeMd5s computes the md5 hashes of the resources of the package.
//
// The hash of a resource is computed in the background when it is selected
// and the interface is updated from that goroutine. The tests compute the
// hashes up front, so that the interface is not updated while it is inspected.
func computeMd5s(t *testing.T, pkg *ddpackage.Package) {
	t.Helper()

	for _, fi := range pkg.FileList().AsSlice() {
		data, err := os.ReadFile(fi.Path)
		require.NoError(t, err)

		sum := md5.Sum(data)
		fi.Md5.UpdateWith(sum[:])
	}
}

// bulkFixture writes a fixture whose two objects have a tag in common and
// returns the path of the unpacked package.
func bulkFixture(t *testing.T) string {
	t.Helper()

	fixture := testutil.DefaultFixture()
	fixture.Tags["shared"] = []string{
		"textures/objects/Rock.png",
		"textures/objects/Tree.png",
	}
	return fixture.Write(t)
}

// selectObjects selects the two objects of the fixture in the resource tree.
func selectObjects(t *testing.T, a *App, tree *resource_tree.ResourceTree) {
	t.Helper()

	selectResource(t, a, tree, resPath("textures/objects/Rock.png"))
	tree.Select(resPath("textures/objects/Tree.png"))
	waitForBackgroundWork(t, a)
}

// TestSpecBulkOperations verifies the "when more than one resource is
// selected" section of the UI specification.
//
// Covered by this test:
//
//	when more than one resource is selected
//	  - the resource pane shows the bulk operations pane
//	    - with a tags tab
//	      - with a "Bulk Tag Operations" header
//	      - with a list of the selected resources
//	        - with the file name and the path of every resource
//	      - with a list of the tags that all the resources have in common
//	      - with an entry to add a tag to all the resources
func TestSpecBulkOperations(t *testing.T) {
	a := newSpecApp(t)
	dir := bulkFixture(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	// the hashes are computed up front, so that the interface is not
	// updated from a background goroutine while it is inspected
	computeMd5s(t, a.pkg)

	tree := resourceTree(t, a)
	selectObjects(t, a, tree)

	uitest.Expect(t, uitest.In(namedPart(t, a, nameBulkOpsPane)),
		uitest.Group("when more than one resource is selected",
			uitest.Has("a tags tab",
				uitest.Tab(lang.X("preview.tab.tags", "Tags")),
			),
			uitest.Has("a bulk tag operations header",
				uitest.Label(lang.X("bulk.tagOps.header", "Bulk Tag Operations")),
			),
			uitest.Has("the list of the selected resources",
				uitest.Named(nameBulkResourceList),
			),
			uitest.Has("the list of the common tags",
				uitest.Named(nameBulkTagList),
			),
			uitest.Has("the entry to add a tag to all the resources",
				uitest.Named(nameBulkTagSelector),
			),
		),
	)

	// the list shows every selected resource
	resourceList := bulkResourceList(t, a)
	require.Equal(t, 2, resourceList.Length())

	// the node of a resource shows its file name and the path it is under
	node := resourceList.CreateItem()
	resourceList.UpdateItem(0, node)

	icon := node.(*fyne.Container).Objects[0].(*widget.FileIcon)
	texts := node.(*fyne.Container).Objects[1].(*fyne.Container)
	rows := texts.Objects[0].(*fyne.Container)

	nameRow := rows.Objects[0].(*fyne.Container)
	pathRow := rows.Objects[1].(*fyne.Container)

	nameLbl := nameRow.Objects[0].(*widget.Label)
	require.NotNil(t, nameRow.Objects[1])
	pathLbl := pathRow.Objects[0].(*fyne.Container).Objects[0].(*widgets.ThemedText)

	assert.Equal(t, "Rock.png", nameLbl.Text)
	assert.Equal(t, "textures/objects/", pathLbl.Text.Text)
	assert.NotNil(t, icon.URI, "the node shows the icon of the resource")

	// the list shows the tags that all the selected resources have in common
	tagList := bulkTagList(t, a)
	require.Equal(t, 1, tagList.Length())
}

// TestSpecBulkResourceNodeLayout verifies the layout of the node of the list
// of the selected resources.
//
// Covered by this test:
//
//	when more than one resource is selected
//	  - the list of the selected resources
//	    - the path of a resource starts at the same place as its file name
//	    - the path is shown below the file name
//	    - the deselect button is in the top right corner of the node
func TestSpecBulkResourceNodeLayout(t *testing.T) {
	a := newSpecApp(t)
	dir := bulkFixture(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	computeMd5s(t, a.pkg)

	tree := resourceTree(t, a)
	selectObjects(t, a, tree)

	resourceList := bulkResourceList(t, a)
	node := resourceList.CreateItem()
	resourceList.UpdateItem(0, node)

	texts := node.(*fyne.Container).Objects[1].(*fyne.Container)
	rows := texts.Objects[0].(*fyne.Container)

	nameRow := rows.Objects[0].(*fyne.Container)
	pathRow := rows.Objects[1].(*fyne.Container)

	nameLbl := nameRow.Objects[0].(*widget.Label)
	pathLbl := pathRow.Objects[0].(*fyne.Container).Objects[0].(*widgets.ThemedText)

	// the node is laid out on its own, so that the widgets of it have their
	// final positions
	layoutNode(t, node)

	scope := uitest.In(node)

	// the path starts at the same place as the file name
	nameStart := textStartX(t, scope, nameLbl)
	pathStart := textStartX(t, scope, pathLbl)
	assert.InDelta(t, nameStart, pathStart, 0.5,
		"the path must start at the same place as the file name")

	// the path is shown below the file name
	namePos, ok := scope.Position(nameLbl)
	require.True(t, ok)
	pathPos, ok := scope.Position(pathLbl)
	require.True(t, ok)
	assert.Greater(t, pathPos.Y, namePos.Y,
		"the path must be shown below the file name")

	// the deselect button is in the top right corner of the node
	deselect := nameRow.Objects[1].(*fyne.Container)
	deselectPos, ok := scope.Position(deselect)
	require.True(t, ok)
	assert.InDelta(t, node.Size().Width, deselectPos.X+deselect.Size().Width, 0.5,
		"the deselect button must be at the right edge of the node")
	assert.InDelta(t, namePos.Y, deselectPos.Y, 0.5,
		"the deselect button must be at the top of the file name")
}

// layoutNode lays a node of the list of the selected resources out, so that
// the widgets of it have their final positions.
func layoutNode(t *testing.T, node fyne.CanvasObject) {
	t.Helper()

	win := test.NewWindow(node)
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(300, 60))
}

// textStartX returns the horizontal position of the start of the text of the
// widget, relative to the root of the scope.
func textStartX(t *testing.T, scope *uitest.Scope, w fyne.Widget) float32 {
	t.Helper()

	pos, ok := scope.Position(w)
	require.True(t, ok, "the widget is not part of the scope")

	return pos.X + uitest.TextStart(w).X
}

// TestSpecBulkDeselectFolder verifies that a resource that is selected with
// its folder can be deselected.
//
// Covered by this test:
//
//	when a folder is selected in the resource tree
//	  - the bulk operations pane shows the resources of the folder
//	    - deselecting a resource unselects the folder
func TestSpecBulkDeselectFolder(t *testing.T) {
	a := newSpecApp(t)
	dir := bulkFixture(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	computeMd5s(t, a.pkg)

	tree := resourceTree(t, a)

	// the folder of the objects of the fixture is selected
	folder := findFolder(t, tree, "textures/objects")
	selectResource(t, a, tree, folder)

	// the resources of the folder are listed
	resourceList := bulkResourceList(t, a)
	require.Equal(t, 2, resourceList.Length())

	node := resourceList.CreateItem()
	resourceList.UpdateItem(0, node)
	texts := node.(*fyne.Container).Objects[1].(*fyne.Container)
	rows := texts.Objects[0].(*fyne.Container)
	nameRow := rows.Objects[0].(*fyne.Container)
	deselectBtn := nameRow.Objects[1].(*fyne.Container).Objects[0].(*widgets.ToolTipButton)
	test.Tap(deselectBtn)

	// the folder is not selected any more
	assert.Empty(t, tree.GetSelection())
}

// findFolder returns the id of the node of the folder with the given path.
func findFolder(t *testing.T, tree *resource_tree.ResourceTree, path string) widget.TreeNodeID {
	t.Helper()

	var found widget.TreeNodeID
	var walk func(uid widget.TreeNodeID)
	walk = func(uid widget.TreeNodeID) {
		for _, child := range tree.ChildUIDs(uid) {
			if _, ok := tree.DataFor(child).(*resource_tree.ResourceDirBranch); ok &&
				strings.HasSuffix(string(child), "/"+path) {
				found = child
				return
			}
			walk(child)
		}
	}
	walk(resource_tree.PackageTreeRootID)

	if found == "" {
		t.Fatalf("the tree has no folder %q", path)
	}
	return found
}

// TestSpecBulkDeselect verifies that a resource can be deselected from the
// bulk operations pane.
//
// Covered by this test:
//
//	when more than one resource is selected
//	  - the bulk operations pane shows a button to deselect a resource
//	    - the resource is deselected from the resource tree
func TestSpecBulkDeselect(t *testing.T) {
	a := newSpecApp(t)
	dir := bulkFixture(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	computeMd5s(t, a.pkg)

	tree := resourceTree(t, a)
	selectObjects(t, a, tree)

	resourceList := bulkResourceList(t, a)
	node := resourceList.CreateItem()
	resourceList.UpdateItem(0, node)

	texts := node.(*fyne.Container).Objects[1].(*fyne.Container)
	rows := texts.Objects[0].(*fyne.Container)
	nameRow := rows.Objects[0].(*fyne.Container)
	deselectBtn := nameRow.Objects[1].(*fyne.Container).Objects[0].(*widgets.ToolTipButton)
	test.Tap(deselectBtn)

	// the resource is not selected any more, so the information of the
	// remaining one is shown again
	assert.Len(t, tree.GetSelection(), 1)
	uitest.Expect(t, uitest.In(namedPart(t, a, nameResourcePane)),
		uitest.NotExists("the bulk operations pane", uitest.Named(nameBulkOpsPane)),
	)
}

// TestSpecBulkTags verifies that the tags of all the selected resources can be
// changed at once.
//
// Covered by this test:
//
//	when more than one resource is selected
//	  - the tags of all the selected resources can be removed
//	  - a tag can be added to all the selected resources
func TestSpecBulkTags(t *testing.T) {
	a := newSpecApp(t)
	dir := bulkFixture(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	computeMd5s(t, a.pkg)

	tree := resourceTree(t, a)
	selectObjects(t, a, tree)

	// the tag that both resources have in common is listed
	tagList := bulkTagList(t, a)
	require.Equal(t, 1, tagList.Length())

	// removing it removes it from all the selected resources
	tagNode := tagList.CreateItem()
	tagList.UpdateItem(0, tagNode)
	removeBtn := tagNode.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widgets.ToolTipButton)
	test.Tap(removeBtn)

	assert.Empty(t, a.pkg.Tags().TagsFor(
		resPath("textures/objects/Rock.png"),
		resPath("textures/objects/Tree.png"),
	).AsSlice())
	assert.Empty(t, bulkTagList(t, a).Length())

	// a new tag is added to all the selected resources
	selector, ok := namedPart(t, a, nameBulkTagSelector).(*widget.SelectEntry)
	require.True(t, ok)
	selector.SetText("bulk")

	addBtn, ok := namedPart(t, a, nameBulkTagAddButton).(*widget.Button)
	require.True(t, ok)
	test.Tap(addBtn)

	assert.Equal(t, []string{"bulk"}, a.pkg.Tags().TagsFor(
		resPath("textures/objects/Rock.png"),
		resPath("textures/objects/Tree.png"),
	).AsSlice())
	assert.Equal(t, 1, bulkTagList(t, a).Length())
}
