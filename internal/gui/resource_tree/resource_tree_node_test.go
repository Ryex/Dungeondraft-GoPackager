package resource_tree

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
)

const rockResPath = widget.TreeNodeID("res://packs/testpack/textures/objects/Rock.png")

// treeNodes lays the tree out and returns the node objects it holds, keyed
// by the tree entry they represent.
func treeNodes(t *testing.T, tree *ResourceTree, branches ...string) map[widget.TreeNodeID]*ResourceTreeNode {
	t.Helper()

	tree.OpenBranch(PackageTreeRootID)
	for _, branch := range branches {
		tree.OpenBranch(branch)
	}
	tree.Resize(fyne.NewSize(400, 400))
	tree.Refresh()

	nodes := map[widget.TreeNodeID]*ResourceTreeNode{}
	for _, node := range collectTreeNodes(test.WidgetRenderer(tree)) {
		nodes[node.uid] = node
	}
	return nodes
}

// collectTreeNodes walks the renderers of the tree and collects the node
// objects of the app.
func collectTreeNodes(renderer fyne.WidgetRenderer) []*ResourceTreeNode {
	var found []*ResourceTreeNode
	if renderer == nil {
		return found
	}
	for _, object := range renderer.Objects() {
		if node, ok := object.(*ResourceTreeNode); ok {
			found = append(found, node)
			continue
		}
		widgetObject, ok := object.(fyne.Widget)
		if !ok {
			continue
		}
		found = append(found, collectTreeNodes(test.WidgetRenderer(widgetObject))...)
	}
	return found
}

// nodeBackground returns the rectangle the node draws its highlight with.
func nodeBackground(t *testing.T, node *ResourceTreeNode) *canvas.Rectangle {
	t.Helper()

	renderer, ok := test.WidgetRenderer(node).(*resourceTreeNodeRenderer)
	require.True(t, ok, "the node must be rendered by the resource tree renderer")
	require.NotNil(t, renderer.background)
	return renderer.background
}

func TestResourceTreeNodeKnowsItsTreeEntry(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")

	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")
	assert.Equal(t, rockResPath, node.uid)
}

func TestResourceTreeNodeHighlightsTheSelectedResource(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	background := nodeBackground(t, node)
	assert.False(t, background.Visible(), "an unselected node must not be highlighted")

	tree.Select(rockResPath)
	assert.True(t, background.Visible(), "a selected node must be highlighted")

	tree.Unselect(rockResPath)
	assert.False(t, background.Visible(), "an unselected node must not be highlighted")
}

func TestResourceTreeContextMenuTogglesTheSelection(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	// an unselected node offers to be selected
	items := node.contextMenuItems()
	require.Len(t, items, 1)
	assert.Equal(t, lang.X("tree.context.select", "Select"), items[0].Label)

	items[0].Action()
	assert.True(t, tree.IsSelected(rockResPath), "the select action must select the node")

	// a selected node offers to be unselected
	items = node.contextMenuItems()
	require.Len(t, items, 1)
	assert.Equal(t, lang.X("tree.context.unselect", "Unselect"), items[0].Label)

	items[0].Action()
	assert.False(t, tree.IsSelected(rockResPath), "the unselect action must unselect the node")
}

func TestResourceTreeAllParentsOf(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	parents := tree.AllParentsOf(rockResPath)
	assert.Equal(t, []widget.TreeNodeID{
		"dir://textures/objects",
		"dir://textures",
		PackageTreeRootID,
		binding.DataTreeRootID,
	}, parents, "the parents must be listed from the closest to the farthest")
}

func TestResourceTreeNodeIsIndented(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	assert.Greater(t, node.Indent(), float32(0),
		"a node below the package root must be indented")
}

func TestResourceTreeIsParentSelected(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	tree.Select("dir://textures")

	assert.True(t, tree.IsParentSelected(rockResPath),
		"the parents of a selected node are selected as well")
	assert.False(t, tree.IsParentSelected("dir://data"),
		"the parents of a node outside of the selection are not selected")
}

// twoTagFixture returns a fixture in which a resource carries two tags.
func twoTagFixture() *testutil.Fixture {
	fixture := testutil.DefaultFixture()
	fixture.Tags["nature"] = []string{
		"textures/objects/Tree.png",
		"textures/objects/Rock.png",
	}
	return fixture
}

// nodesOfResource returns the node objects the tree holds for a resource.
func nodesOfResource(nodes []*ResourceTreeNode, resPath string) []*ResourceTreeNode {
	var found []*ResourceTreeNode
	for _, node := range nodes {
		info, ok := node.data.(*ResourceFileInfoNode)
		if !ok || info.Info == nil {
			continue
		}
		if info.Info.ResPath == resPath {
			found = append(found, node)
		}
	}
	return found
}

// nodePositions returns the position of every node object of the tree,
// relative to the tree content. The position of a node is the position of
// the row it is shown in.
func nodePositions(renderer fyne.WidgetRenderer, offset fyne.Position) map[*ResourceTreeNode]fyne.Position {
	positions := map[*ResourceTreeNode]fyne.Position{}
	for _, object := range renderer.Objects() {
		if node, ok := object.(*ResourceTreeNode); ok {
			positions[node] = offset
			continue
		}
		widgetObject, ok := object.(fyne.Widget)
		if !ok {
			continue
		}
		for node, position := range nodePositions(test.WidgetRenderer(widgetObject), offset.Add(object.Position())) {
			positions[node] = position
		}
	}
	return positions
}

func TestResourceTreeTagModeShowsAResourceUnderEveryTag(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)

	tree.SetMode(ResourceTreeTagMode)
	tree.OpenBranch(PackageTreeRootID)
	tree.OpenBranch("tag://nature")
	tree.OpenBranch("tag://rock")
	tree.Resize(fyne.NewSize(400, 400))
	tree.Refresh()

	nodes := collectTreeNodes(test.WidgetRenderer(tree))

	shown := nodesOfResource(nodes, rockResPath)
	require.Len(t, shown, 2, "the resource must be shown under both of its tags")

	assert.NotEqual(t, shown[0].uid, shown[1].uid,
		"every occurrence of a resource needs an id of its own")
	assert.Same(t, shown[0].data, shown[1].data,
		"the occurrences must share the data of the resource")

	positions := nodePositions(test.WidgetRenderer(tree), fyne.NewPos(0, 0))
	require.Contains(t, positions, shown[0])
	require.Contains(t, positions, shown[1])
	assert.NotEqual(t, positions[shown[0]], positions[shown[1]],
		"the occurrences must be shown in different rows")
}

func TestResourceTreeTagModeSelectsAResourceOnce(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)

	tree.SetMode(ResourceTreeTagMode)

	// the same resource shown under two tags is still one resource
	tree.Select(tagResourceUID("tag://nature", rockResPath))
	tree.Select(tagResourceUID("tag://rock", rockResPath))

	data := tree.GetSelectionAndChildData()
	require.Len(t, data, 1, "a resource shown under two tags must be selected once")

	node, ok := data[0].(*ResourceFileInfoNode)
	require.True(t, ok)
	require.NotNil(t, node.Info)
	assert.Equal(t, rockResPath, node.Info.ResPath)
}

func TestResourceTreeTagModeHighlightsEveryNodeOfTheResource(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)
	tree.SetMode(ResourceTreeTagMode)

	nodes := treeNodes(t, tree, "tag://nature", "tag://rock")

	natureUID := tagResourceUID("tag://nature", rockResPath)
	rockUID := tagResourceUID("tag://rock", rockResPath)
	natureNode, ok := nodes[natureUID]
	require.True(t, ok, "the tree must hold a node for the resource under the nature tag")
	rockNode, ok := nodes[rockUID]
	require.True(t, ok, "the tree must hold a node for the resource under the rock tag")

	tree.Select(natureUID)

	assert.True(t, tree.IsVisiblySelected(natureUID))
	assert.True(t, tree.IsVisiblySelected(rockUID),
		"every node of the resource must be shown as selected")

	assert.True(t, nodeBackground(t, natureNode).Visible(),
		"the selected node must be highlighted")
	assert.True(t, nodeBackground(t, rockNode).Visible(),
		"every node of the resource must be highlighted")
}

func TestResourceTreeTagModeUnselectsEveryNodeOfTheResource(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)
	tree.SetMode(ResourceTreeTagMode)

	nodes := treeNodes(t, tree, "tag://nature", "tag://rock")

	natureUID := tagResourceUID("tag://nature", rockResPath)
	rockUID := tagResourceUID("tag://rock", rockResPath)
	natureNode, ok := nodes[natureUID]
	require.True(t, ok, "the tree must hold a node for the resource under the nature tag")
	rockNode, ok := nodes[rockUID]
	require.True(t, ok, "the tree must hold a node for the resource under the rock tag")

	tree.Select(natureUID)
	require.True(t, tree.IsVisiblySelected(rockUID))

	// unselecting the other node of the resource unselects the resource
	tree.Unselect(rockUID)

	assert.False(t, tree.IsVisiblySelected(natureUID),
		"no node of the resource must be shown as selected")
	assert.False(t, tree.IsVisiblySelected(rockUID),
		"no node of the resource must be shown as selected")
	assert.Empty(t, tree.GetSelection())

	assert.False(t, nodeBackground(t, natureNode).Visible(),
		"the unselected node must not be highlighted")
	assert.False(t, nodeBackground(t, rockNode).Visible(),
		"no node of the resource must be highlighted")
}

func TestResourceTreeTagModeContextMenuOfEveryNodeOfTheResource(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)
	tree.SetMode(ResourceTreeTagMode)

	nodes := treeNodes(t, tree, "tag://nature", "tag://rock")

	natureUID := tagResourceUID("tag://nature", rockResPath)
	rockUID := tagResourceUID("tag://rock", rockResPath)
	_, ok := nodes[natureUID]
	require.True(t, ok, "the tree must hold a node for the resource under the nature tag")
	rockNode, ok := nodes[rockUID]
	require.True(t, ok, "the tree must hold a node for the resource under the rock tag")

	tree.Select(natureUID)

	// the node that was not selected is shown as selected as well, so it
	// offers to be unselected too
	items := rockNode.contextMenuItems()
	require.Len(t, items, 1)
	assert.Equal(t, lang.X("tree.context.unselect", "Unselect"), items[0].Label)

	items[0].Action()
	assert.Empty(t, tree.GetSelection(),
		"unselecting a node of the resource must unselect the resource")
	assert.False(t, tree.IsVisiblySelected(natureUID))
}

func TestResourceTreeTagModeSelectionReportsEveryNode(t *testing.T) {
	pkg, _ := loadFixtureWith(t, twoTagFixture())
	tree := newTree(t, pkg)
	tree.SetMode(ResourceTreeTagMode)

	tree.Select(tagResourceUID("tag://nature", rockResPath))

	selection := tree.GetSelection()
	assert.Len(t, selection, 2, "every node of the resource must be reported as selected")
	assert.Contains(t, selection, tagResourceUID("tag://nature", rockResPath))
	assert.Contains(t, selection, tagResourceUID("tag://rock", rockResPath))

	data := tree.GetSelectionAndChildData()
	require.Len(t, data, 1, "the resource must be reported once")
}

// click simulates a primary mouse click on the node.
func click(node *ResourceTreeNode) {
	clickWith(node, 0)
}

// clickWith simulates a primary mouse click with the given modifiers on the
// node.
func clickWith(node *ResourceTreeNode, modifier fyne.KeyModifier) {
	node.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, Modifier: modifier})
}

// doubleClick simulates a double click on the node.
func doubleClick(node *ResourceTreeNode) {
	click(node)
	click(node)
}

// doubleClickWith simulates a double click with the given modifiers on the
// node.
func doubleClickWith(node *ResourceTreeNode, modifier fyne.KeyModifier) {
	clickWith(node, modifier)
	clickWith(node, modifier)
}

func TestResourceTreeDoubleClickSelectsAnUnselectedResource(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")
	require.False(t, tree.IsVisiblySelected(rockResPath))

	doubleClick(node)

	assert.True(t, tree.IsVisiblySelected(rockResPath),
		"a double click must select a resource that was not selected")
}

func TestResourceTreeDoubleClickUnselectsASelectedResource(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	tree.Select(rockResPath)
	require.True(t, tree.IsVisiblySelected(rockResPath))

	doubleClick(node)

	assert.False(t, tree.IsVisiblySelected(rockResPath),
		"a double click must unselect a resource that was selected")
}

func TestResourceTreeDoubleClickKeepsTheRestOfTheSelection(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")
	_, ok = nodes["res://packs/testpack/textures/objects/Tree.png"]
	require.True(t, ok, "the tree must hold a node for the resource")

	tree.Select("res://packs/testpack/textures/objects/Tree.png")

	doubleClick(node)

	assert.True(t, tree.IsVisiblySelected("res://packs/testpack/textures/objects/Tree.png"),
		"a double click must keep the rest of the selection")
	assert.True(t, tree.IsVisiblySelected(rockResPath),
		"a double click must select the resource it is on")
}

func TestResourceTreeClickAfterAPauseIsNoDoubleClick(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	click(node)
	time.Sleep(fyne.CurrentApp().Driver().DoubleTapDelay() + 50*time.Millisecond)
	click(node)
	time.Sleep(fyne.CurrentApp().Driver().DoubleTapDelay() + 50*time.Millisecond)

	assert.True(t, tree.IsVisiblySelected(rockResPath),
		"two clicks that are too far apart must not toggle the selection")
}

func TestResourceTreeDoubleClickWithControlDoesNotToggle(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	// the control key removes the node from the selection, it must not
	// toggle it
	doubleClickWith(node, fyne.KeyModifierControl)

	assert.False(t, tree.IsVisiblySelected(rockResPath),
		"a double click with the control key must not select the node")
}

func TestResourceTreeDoubleClickWithControlUnselects(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	tree.Select(rockResPath)
	require.True(t, tree.IsVisiblySelected(rockResPath))

	doubleClickWith(node, fyne.KeyModifierControl)

	assert.False(t, tree.IsVisiblySelected(rockResPath),
		"a double click with the control key must unselect the node")
}

func TestResourceTreeDoubleClickWithShiftKeepsTheSelection(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")
	otherUID := widget.TreeNodeID("res://packs/testpack/textures/objects/Tree.png")
	_, ok = nodes[otherUID]
	require.True(t, ok, "the tree must hold a node for the resource")

	tree.Select(otherUID)

	// the shift key adds the node to the selection, it must not toggle it
	doubleClickWith(node, fyne.KeyModifierShift)

	assert.True(t, tree.IsVisiblySelected(rockResPath),
		"a double click with the shift key must keep the node selected")
	assert.True(t, tree.IsVisiblySelected(otherUID),
		"a double click with the shift key must keep the rest of the selection")
}

func TestResourceTreeDoubleClickWithShiftKeepsASelectedNode(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	tree.Select(rockResPath)
	require.True(t, tree.IsVisiblySelected(rockResPath))

	// the shift key adds the node to the selection, so a node that is
	// already selected stays selected
	doubleClickWith(node, fyne.KeyModifierShift)

	assert.True(t, tree.IsVisiblySelected(rockResPath),
		"a double click with the shift key must not unselect the node")
}

func TestResourceTreeModifiedClickEndsTheClickSequence(t *testing.T) {
	pkg, _ := loadFixture(t)
	tree := newTree(t, pkg)

	nodes := treeNodes(t, tree, "dir://textures", "dir://textures/objects")
	node, ok := nodes[rockResPath]
	require.True(t, ok, "the tree must hold a node for the resource")

	click(node)
	time.Sleep(fyne.CurrentApp().Driver().DoubleTapDelay() + 50*time.Millisecond)
	require.True(t, tree.IsVisiblySelected(rockResPath))

	// the control key removes the node from the selection, it must not be
	// swallowed by a double click of the click before it
	clickWith(node, fyne.KeyModifierControl)
	assert.False(t, tree.IsVisiblySelected(rockResPath),
		"a modified click must not be swallowed by a double click")

	// the modified click ended the click sequence, so this click is not the
	// second click of a double click
	click(node)
	time.Sleep(fyne.CurrentApp().Driver().DoubleTapDelay() + 50*time.Millisecond)
	assert.True(t, tree.IsVisiblySelected(rockResPath),
		"a modified click must end the click sequence")
}
