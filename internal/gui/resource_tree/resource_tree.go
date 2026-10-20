package resource_tree

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage/filter"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

// Declare conformity with interfaces
var (
	_ fyne.Focusable = (*ResourceTree)(nil)
	_ fyne.Widget    = (*ResourceTree)(nil)
)

type ResourceTreeNodeData interface {
	Update(uid widget.TreeNodeID, obj fyne.CanvasObject)
	IsBranch() bool
}

type ResourceTreeMode int

const (
	ResourceTreePathMode ResourceTreeMode = iota
	ResourceTreeTagMode
)

// Tag Branch
type ResourceTagBranch struct {
	tree     *ResourceTree
	Tag      string
	Untagged bool
}

func createBranchNodeObject() fyne.CanvasObject {
	icon := widget.NewIcon(nil)
	btn := widgets.NewToolTipButtonWithIcon("template", theme.ErrorIcon(), nil)
	btn.SetIcon(theme.DeleteIcon())
	return container.NewBorder(nil, nil, icon, container.NewPadded(btn), widget.NewLabel("label template"))
}

func createNodeObject() fyne.CanvasObject {
	icon := widget.NewFileIcon(nil)
	return container.NewBorder(nil, nil, icon, nil, widget.NewLabel("label template"))
}

func (tb *ResourceTagBranch) IsBranch() bool { return true }
func (tb *ResourceTagBranch) Update(uid widget.TreeNodeID, obj fyne.CanvasObject) {
	c := obj.(*fyne.Container)
	lbl := c.Objects[0].(*widget.Label)
	icn := c.Objects[1].(*widget.Icon)
	btn := c.Objects[2].(*fyne.Container).Objects[0].(*widgets.ToolTipButton)
	if tb.tree.editable && !tb.Untagged {
		btn.Show()
	} else {
		btn.Hide()
	}

	lbl.SetText(tb.Tag)
	var r fyne.Resource
	if tb.tree.IsBranchOpen(uid) {
		r = theme.FolderOpenIcon()
	} else {
		r = theme.FolderIcon()
	}

	icn.SetResource(r)

	btn.SetText("")
	btn.SetToolTip(lang.X(
		"package.tag.delete.tooltip",
		"Delete the '{{.Tag}}' tag",
		map[string]string{
			"Tag": tb.Tag,
		},
	))
	btn.Show()
}

// Dir Branch
type ResourceDirBranch struct {
	tree *ResourceTree
	Dir  string
}

func (tb *ResourceDirBranch) IsBranch() bool { return true }
func (tb *ResourceDirBranch) Update(uid widget.TreeNodeID, obj fyne.CanvasObject) {
	c := obj.(*fyne.Container)
	lbl := c.Objects[0].(*widget.Label)
	icn := c.Objects[1].(*widget.Icon)
	btn := c.Objects[2].(*fyne.Container).Objects[0].(*widgets.ToolTipButton)
	btn.Hide()

	lbl.SetText(tb.Dir)

	var r fyne.Resource
	if tb.tree.IsBranchOpen(uid) {
		r = theme.FolderOpenIcon()
	} else {
		r = theme.FolderIcon()
	}
	icn.SetResource(r)
}

// ResourceFileInfoNode node with file info
type ResourceFileInfoNode struct {
	tree *ResourceTree
	Info *structures.FileInfo
}

func (rn *ResourceFileInfoNode) IsBranch() bool { return false }
func (rn *ResourceFileInfoNode) Update(uid widget.TreeNodeID, obj fyne.CanvasObject) {
	c := obj.(*fyne.Container)
	lbl := c.Objects[0].(*widget.Label)
	icn := c.Objects[1].(*widget.FileIcon)
	if rn.Info == nil {
		lbl.TextStyle = fyne.TextStyle{Italic: true}
		lbl.SetText(lang.X("tree.empty", "No resources"))
		icn.SetURI(nil)
	} else {
		icn.SetURI(
			storage.NewFileURI(
				filepath.Join(rn.tree.pkg.UnpackedPath(), utils.NormalizeResourcePath(rn.Info.ResPath)),
			),
		)
		lbl.TextStyle = fyne.TextStyle{}
		lbl.SetText(filepath.Base(rn.Info.ResPath))
	}
}

// Declare conformity with interfaces
var (
	_ ResourceTreeNodeData = (*ResourceFileInfoNode)(nil)
	_ ResourceTreeNodeData = (*ResourceDirBranch)(nil)
	_ ResourceTreeNodeData = (*ResourceTagBranch)(nil)
)

type ResourceTree struct {
	widget.Tree

	parent fyne.Window

	selected []widget.TreeNodeID
	// visiblySelected holds every node that is shown as selected. A
	// resource that is shown under more than one tag is shown as selected
	// in every one of them, so this holds more nodes than selected.
	visiblySelected []widget.TreeNodeID
	// clickSelection holds the selection as it was before the current click
	// sequence started. A double click restores it before it toggles the
	// node, so that the double click only toggles the node it is on.
	clickSelection []widget.TreeNodeID
	// lastClickUID and lastClickTime hold the node and the time of the
	// last click, which are used to detect a double click.
	lastClickUID  widget.TreeNodeID
	lastClickTime time.Time
	nodeTree      map[widget.TreeNodeID][]widget.TreeNodeID
	nodeMap       map[widget.TreeNodeID]ResourceTreeNodeData
	mode          ResourceTreeMode

	editable bool
	filter   filter.ResourceFilter

	OnSelectionChanged func()

	pkg *ddpackage.Package
}

func NewResourceTree(pkg *ddpackage.Package, parent fyne.Window) *ResourceTree {
	t := &ResourceTree{pkg: pkg, parent: parent}
	t.Tree.ChildUIDs = t.ChildUIDs
	t.Tree.IsBranch = t.IsBranch
	t.Tree.CreateNode = t.CreateNode
	t.Tree.UpdateNode = t.UpdateNode
	t.filter = filter.ResourceFilterEmpty{}
	t.ExtendBaseWidget(t)
	return t
}

func (t *ResourceTree) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	if children, ok := t.nodeTree[uid]; ok {
		return children
	}
	return []widget.TreeNodeID{}
}

func (t *ResourceTree) SetEditable(editable bool) {
	t.editable = editable
	t.Refresh()
}

func (t *ResourceTree) GetMode() ResourceTreeMode {
	return t.mode
}

func (t *ResourceTree) SetMode(mode ResourceTreeMode) {
	if t.mode == mode {
		return
	}
	t.mode = mode
	t.RebuildTree()
	t.Refresh()
}

func (t *ResourceTree) SetFilter(f filter.ResourceFilter) {
	if t.filter.Eq(f) {
		return
	}
	t.filter = f
	t.RebuildTree()
	t.Refresh()
}

func (t *ResourceTree) DataFor(uid widget.TreeNodeID) ResourceTreeNodeData {
	if data, ok := t.nodeMap[uid]; ok {
		return data
	}
	return nil
}

func (t *ResourceTree) RefreshTree() {
	t.RebuildTree()
	t.Refresh()
}

// ==================

func nodeID(dir string) widget.TreeNodeID {
	id := "dir://" + dir
	if dir == "" {
		id = PackageTreeRootID
	}
	return id
}

const PackageTreeRootID string = "root://package"

// tagResourceUID returns the id of a resource that is shown under a tag
// branch.
//
// A resource can carry more than one tag and is then shown under more than
// one branch. The tree keeps a single node object per id and can therefore
// only draw a resource in one place, so every occurrence of a resource needs
// an id of its own.
func tagResourceUID(tagPath string, resPath string) string {
	return tagPath + "|" + resPath
}

func resourcesUnderTreeUID(tni string, tree map[string][]string) []string {
	res := []string{}
	stack := structures.NewStack[string]()
	stack.Push(tni)
	for !stack.IsEmpty() {
		uid, _ := stack.Pop()
		leaves, exists := tree[uid]
		if exists {
			for _, leaf := range leaves {
				if strings.HasPrefix(leaf, "res://") {
					res = append(res, uid)
				} else {
					stack.Push(uid)
				}
			}
		}
	}
	return res
}

// ==================

func (t *ResourceTree) RebuildTree() {
	pkgTags := t.pkg.Tags()
	fil := t.pkg.FileList().AsSlice()
	if f := t.filter; f != nil {
		fil = slices.Collect(utils.Filter(slices.Values(fil), func(fi *structures.FileInfo) bool {
			return f.Matches(fi, pkgTags)
		}))
	}

	logrus.Debugf("Rebuilding  Tree")
	t.nodeTree = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	t.nodeMap = make(map[widget.TreeNodeID]ResourceTreeNodeData)

	trimTrailingSlash := func(path string) string {
		for strings.HasSuffix(path, "/") {
			path = path[:max(len(path)-1, 0)] // chop trailing `/`
		}
		return path
	}
	ensureNode := func(node widget.TreeNodeID) bool {
		_, ok := t.nodeTree[node]
		if !ok {
			t.nodeTree[node] = []widget.TreeNodeID{}
		}
		return !ok
	}
	nodeHasLeaf := func(node widget.TreeNodeID, leaf widget.TreeNodeID) bool {
		return slices.Contains(t.nodeTree[node], leaf)
	}
	appendLeaf := func(node widget.TreeNodeID, leaf widget.TreeNodeID) {
		t.nodeTree[node] = append(t.nodeTree[node], leaf)
	}
	ensureLeaf := func(node widget.TreeNodeID, leaf widget.TreeNodeID) bool {
		if !nodeHasLeaf(node, leaf) {
			logrus.Debugf("adding leaf %v to node %v", leaf, node)
			appendLeaf(node, leaf)
			return true
		}
		return false
	}
	var ensureBranch func(widget.TreeNodeID, widget.TreeNodeID, string)
	ensureParentBranch := func(node widget.TreeNodeID) {
		if node == "" {
			return
		}
		parent, dir := filepath.Split(node)
		if parent != "" {
			parent = trimTrailingSlash(parent)
		}
		ensureBranch(parent, nodeID(node), dir)
	}
	ensureBranch = func(node string, leaf widget.TreeNodeID, dir string) {
		ensureNode(nodeID(node))
		if ensureLeaf(nodeID(node), leaf) {
			t.nodeMap[leaf] = &ResourceDirBranch{t, dir}
		}
		ensureParentBranch(node)
	}
	ensureResource := func(node string, leaf widget.TreeNodeID, fi *structures.FileInfo) {
		ensureNode(nodeID(node))
		if ensureLeaf(nodeID(node), leaf) {
			t.nodeMap[leaf] = &ResourceFileInfoNode{t, fi}
		}
		ensureParentBranch(node)
	}

	ensureNode(PackageTreeRootID)
	ensureNode(PackageTreeRootID)
	switch t.mode {
	case ResourceTreePathMode:
		{
			logrus.Debugf("Resource Mode")
			for _, fi := range fil {
				dir, _ := filepath.Split(fi.RelPath)
				dir = trimTrailingSlash(dir)
				ensureResource(dir, fi.ResPath, fi)
				logrus.Debugf("Node for resource %v", fi.ResPath)
			}
		}
	case ResourceTreeTagMode:
		{
			logrus.Debugf("Tag Mode")
			untaggedLbl := lang.X("tree.untagged.label", "Untagged")
			untaggedPath := "notag://untagged"
			for _, fi := range fil {
				logrus.Debugf("Nodes for resource %v", fi.ResPath)
				if fi.IsTaggable() {
					tags := pkgTags.TagsFor(fi.ResPath)
					if tags.Empty() {
						ensureNode(untaggedPath)
						uid := tagResourceUID(untaggedPath, fi.ResPath)
						if ensureLeaf(untaggedPath, uid) {
							t.nodeMap[uid] = &ResourceFileInfoNode{t, fi}
						}
					} else {
						// every branch the resource is shown under shares
						// the same data, so that the resource stays one
						// resource even though it is shown more than once
						data := &ResourceFileInfoNode{t, fi}
						for _, tag := range tags.AsSlice() {
							tagPath := "tag://" + tag
							if ensureNode(tagPath) {
								t.nodeMap[tagPath] = &ResourceTagBranch{t, tag, false}
							}
							uid := tagResourceUID(tagPath, fi.ResPath)
							if ensureLeaf(tagPath, uid) {
								t.nodeMap[uid] = data
							}
						}
					}
				}
			}
			if len(t.nodeTree[untaggedPath]) > 0 {
				t.nodeMap[untaggedPath] = &ResourceTagBranch{t, untaggedLbl, true}
			}

			allTags := pkgTags.AllTags()
			slices.Sort(allTags)

			for _, tag := range allTags {
				logrus.Debugf("Node for tag %v", tag)
				tagPath := "tag://" + tag
				if ensureNode(tagPath) {
					t.nodeMap[tagPath] = &ResourceTagBranch{t, tag, false}
				}
				ensureLeaf(PackageTreeRootID, tagPath)
				if len(t.nodeTree[tagPath]) == 0 {
					emptyTag := "empty://" + tag
					uid := tagResourceUID(tagPath, emptyTag)
					if ensureLeaf(tagPath, uid) {
						t.nodeMap[uid] = &ResourceFileInfoNode{t, nil}
					}
				}
			}

			if len(t.nodeTree[untaggedPath]) > 0 {
				ensureLeaf(PackageTreeRootID, untaggedPath)
			}

		}
	}
	logrus.Debugf("Resource Nodes Done")

	if len(t.nodeTree[PackageTreeRootID]) == 0 {
		ensureLeaf(binding.DataTreeRootID, "empty://")
		t.nodeMap["empty://"] = &ResourceDirBranch{t, lang.X("tree.empty", "No Resources")}
	} else {
		ensureLeaf(binding.DataTreeRootID, PackageTreeRootID)
		t.nodeMap[PackageTreeRootID] = &ResourceDirBranch{t, lang.X("tree.root.label", "Package")}
	}

	logrus.Debugf("Tree Rebuilt")

	// the nodes of the tree changed, so the nodes that are shown as
	// selected have to be looked up again
	t.updateVisibleSelection()
}

func (t *ResourceTree) IsBranch(uid widget.TreeNodeID) bool {
	if uid == binding.DataTreeRootID {
		return true
	}
	if data := t.DataFor(uid); data != nil {
		return data.IsBranch()
	}
	return false
}

func (t *ResourceTree) ParentOf(uid widget.TreeNodeID) *widget.TreeNodeID {
	for parent, nodes := range t.nodeTree {
		if slices.Contains(nodes, uid) {
			return &parent
		}
	}
	return nil
}

func (t *ResourceTree) AllParentsOf(uid widget.TreeNodeID) []widget.TreeNodeID {
	parents := []widget.TreeNodeID{}
	node := uid
	for {
		parent := t.ParentOf(node)
		if parent == nil {
			break
		}
		parents = append(parents, *parent)
		node = *parent
	}
	return parents
}

func (t *ResourceTree) IsSelected(uid widget.TreeNodeID) bool {
	return slices.Contains(t.selected, uid)
}

// IsVisiblySelected reports whether the node is shown as selected. Every
// node of a selected resource is shown as selected, so this holds for more
// nodes than IsSelected does.
func (t *ResourceTree) IsVisiblySelected(uid widget.TreeNodeID) bool {
	return slices.Contains(t.visiblySelected, uid)
}

func (t *ResourceTree) IsParentSelected(uid widget.TreeNodeID) bool {
	for _, parent := range t.AllParentsOf(uid) {
		if t.IsSelected(parent) {
			return true
		}
	}
	return false
}

// selectedIndexOfData returns the index of the selected node that shows the
// same resource as the given node, or -1 when there is none.
//
// A resource that is shown under more than one tag has one node per tag, and
// only one of them is selected - the others are shown as selected as well.
func (t *ResourceTree) selectedIndexOfData(uid widget.TreeNodeID) int {
	data := t.DataFor(uid)
	if data == nil {
		return -1
	}
	for i, selected := range t.selected {
		if t.DataFor(selected) == data {
			return i
		}
	}
	return -1
}

// updateVisibleSelection rebuilds the list of the nodes that are shown as
// selected. Every node that shows a selected resource is shown as selected.
func (t *ResourceTree) updateVisibleSelection() {
	visible := []widget.TreeNodeID{}
	for _, uid := range t.selected {
		data := t.DataFor(uid)
		if data == nil {
			continue
		}
		for otherUID, otherData := range t.nodeMap {
			if otherData == data {
				visible = append(visible, otherUID)
			}
		}
	}
	// the map of the nodes is unordered, so the list has to be sorted to
	// keep the order of the visible selection stable
	slices.Sort(visible)
	t.visiblySelected = visible
}

// syncTreeSelection keeps the selection of the Fyne tree in step with our
// own selection, so that the tree does not keep highlighting a node that is
// not selected any more.
//
// The callbacks of the Fyne tree are muted while the selection is synced:
// they report the selection of the single node the Fyne tree can hold, while
// this tree holds a selection of its own and reports it through
// OnSelectionChanged.
func (t *ResourceTree) syncTreeSelection() {
	onSelected, onUnselected := t.Tree.OnSelected, t.Tree.OnUnselected
	t.Tree.OnSelected, t.Tree.OnUnselected = nil, nil
	defer func() {
		t.Tree.OnSelected, t.Tree.OnUnselected = onSelected, onUnselected
	}()

	if len(t.selected) == 0 {
		t.Tree.UnselectAll()
		return
	}
	t.Tree.Select(t.selected[len(t.selected)-1])
}

// ToggleSelection selects a node that is not selected and unselects a node
// that is selected.
func (t *ResourceTree) ToggleSelection(uid widget.TreeNodeID) {
	if t.IsVisiblySelected(uid) {
		t.Unselect(uid)
		return
	}
	t.Select(uid)
}

// selectOnly selects a node and unselects everything else.
func (t *ResourceTree) selectOnly(uid widget.TreeNodeID) {
	t.UnselectAll()
	t.Select(uid)
}

// isDoubleClick reports whether a click on the given node continues the
// click sequence of the node, which makes it a double click.
//
// The interval of the driver is used, so that a double click is recognised
// as fast as the system recognises it.
func (t *ResourceTree) isDoubleClick(uid widget.TreeNodeID) bool {
	return uid == t.lastClickUID &&
		time.Since(t.lastClickTime) < fyne.CurrentApp().Driver().DoubleTapDelay()
}

// startClickSequence remembers the selection as it was before a click
// sequence starts, so that a double click can restore it.
func (t *ResourceTree) startClickSequence(uid widget.TreeNodeID) {
	t.clickSelection = slices.Clone(t.selected)
	t.lastClickUID = uid
	t.lastClickTime = time.Now()
}

// endClickSequence ends the current click sequence, so that the next click
// starts a new one.
func (t *ResourceTree) endClickSequence() {
	t.lastClickTime = time.Time{}
}

// restoreClickSelection restores the selection as it was before the current
// click sequence started.
func (t *ResourceTree) restoreClickSelection() {
	t.selected = slices.Clone(t.clickSelection)
	t.updateVisibleSelection()
	t.syncTreeSelection()
}

func (t *ResourceTree) Select(uid widget.TreeNodeID) {
	// a resource is only selected once, even when it is shown under more
	// than one tag - the node that is selected last stays selected
	if i := t.selectedIndexOfData(uid); i != -1 {
		if t.selected[i] == uid {
			return
		}
		t.selected = slices.Delete(t.selected, i, i+1)
	}
	t.selected = append(t.selected, uid)

	t.updateVisibleSelection()
	t.syncTreeSelection()
	if f := t.OnSelectionChanged; f != nil {
		f()
	}
	t.Refresh()
}

func (t *ResourceTree) Unselect(uid widget.TreeNodeID) {
	// the node that is unselected does not have to be the one that is
	// selected: unselecting a node of a resource unselects the resource
	// in all of the branches it is shown in
	index := slices.Index(t.selected, uid)
	if index == -1 {
		index = t.selectedIndexOfData(uid)
	}
	if index == -1 {
		return
	}

	unselected := t.selected[index]
	t.selected = slices.Delete(t.selected, index, index+1)

	t.updateVisibleSelection()
	t.syncTreeSelection()
	t.Refresh()
	if f := t.OnUnselected; f != nil {
		f(unselected)
	}
	if f := t.OnSelectionChanged; f != nil {
		f()
	}
}

func (t *ResourceTree) UnselectAll() {
	if len(t.selected) == 0 {
		return
	}

	selected := t.selected
	t.selected = nil
	t.updateVisibleSelection()
	t.syncTreeSelection()
	t.Refresh()
	if f := t.OnUnselected; f != nil {
		for _, uid := range selected {
			f(uid)
		}
	}
	if f := t.OnSelectionChanged; f != nil {
		f()
	}
}

// GetSelection returns the ids of the nodes that are shown as selected. A
// resource that is shown under more than one tag is shown as selected in
// every one of them, so this holds more nodes than are selected.
func (t *ResourceTree) GetSelection() []widget.TreeNodeID {
	dest := make([]widget.TreeNodeID, len(t.visiblySelected))
	copy(dest, t.visiblySelected)
	return dest
}

// GetSelectionAndChildData returns the data of the selected resources. A
// resource that is shown under more than one tag is only reported once.
func (t *ResourceTree) GetSelectionAndChildData() []ResourceTreeNodeData {
	results := []ResourceTreeNodeData{}
	seen := map[ResourceTreeNodeData]bool{}
	stack := structures.NewStackFromSlice(t.selected)
	for !stack.IsEmpty() {
		uid, _ := stack.Pop()
		if t.IsBranch(uid) {
			children := t.nodeTree[uid]
			stack.Push(children...)
		}
		// a resource that is shown under more than one tag shares its data
		// between the branches, so that it is only reported once
		if data := t.DataFor(uid); data != nil && !seen[data] {
			seen[data] = true
			results = append(results, data)
		}
	}
	return results
}

func (t *ResourceTree) CreateNode(branch bool) fyne.CanvasObject {
	var content fyne.CanvasObject
	if branch {
		content = createBranchNodeObject()
	} else {
		content = createNodeObject()
	}
	return NewResourceTreeNode(t, content)
}

func (t *ResourceTree) UpdateNode(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
	node, ok := obj.(*ResourceTreeNode)
	if ok {
		node.Update(uid)
	}
}

var (
	_ fyne.Widget        = (*ResourceTreeNode)(nil)
	_ desktop.Cursorable = (*ResourceTreeNode)(nil)
	_ desktop.Mouseable  = (*ResourceTreeNode)(nil)
)

type ResourceTreeNode struct {
	widget.BaseWidget
	tree     *ResourceTree
	uid      widget.TreeNodeID
	data     ResourceTreeNodeData
	content  fyne.CanvasObject
	selected bool
}

func NewResourceTreeNode(tree *ResourceTree, content fyne.CanvasObject) *ResourceTreeNode {
	node := &ResourceTreeNode{
		tree:    tree,
		content: content,
	}
	node.ExtendBaseWidget(node)
	return node
}

func (n *ResourceTreeNode) Update(uid widget.TreeNodeID) {
	n.uid = uid
	n.data = n.tree.DataFor(uid)
	if n.data != nil {
		n.data.Update(uid, n.content)
	} else {
		logrus.Warnf("data for `%v` is nil", n.uid)
	}
	// the node draws the highlight of the selection itself, and the tree
	// does not refresh the objects it holds, so the node has to refresh
	// itself when the tree updates it
	n.Refresh()
}

func (n *ResourceTreeNode) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (n *ResourceTreeNode) MouseDown(e *desktop.MouseEvent) {
	switch e.Button {
	case desktop.MouseButtonPrimary:
		{
			// the modifier keys take precedence over the double click: they
			// add or remove the node from the selection instead of toggling
			// it
			if e.Modifier&fyne.KeyModifierControl != 0 {
				// unselect
				n.tree.endClickSequence()
				n.tree.Unselect(n.uid)
				return
			}
			if e.Modifier&fyne.KeyModifierShift != 0 {
				// add select
				n.tree.endClickSequence()
				n.tree.Select(n.uid)
				return
			}
			if n.tree.isDoubleClick(n.uid) {
				// a double click toggles the selection of the node. The
				// selection of the click that started the sequence is
				// restored first, so that the double click only toggles
				// the node it is on and keeps the rest of the selection.
				n.tree.restoreClickSelection()
				n.tree.ToggleSelection(n.uid)
				n.tree.endClickSequence()
				return
			}
			n.tree.startClickSequence(n.uid)

			// select single
			n.tree.selectOnly(n.uid)
		}
	case desktop.MouseButtonSecondary:
		{
			// show context menu
			contextMenu := fyne.NewMenu("", n.contextMenuItems()...)
			widget.ShowPopUpMenuAtPosition(contextMenu, fyne.CurrentApp().Driver().CanvasForObject(n), e.AbsolutePosition)
		}
	}
}

// contextMenuItems returns the items of the context menu of the node. A
// node that is shown as selected offers to be unselected, an unselected one
// offers to be selected.
func (n *ResourceTreeNode) contextMenuItems() []*fyne.MenuItem {
	if n.tree.IsVisiblySelected(n.uid) {
		return []*fyne.MenuItem{fyne.NewMenuItem(lang.X("tree.context.unselect", "Unselect"), func() {
			n.tree.Unselect(n.uid)
		})}
	}
	return []*fyne.MenuItem{fyne.NewMenuItem(lang.X("tree.context.select", "Select"), func() {
		n.tree.Select(n.uid)
	})}
}

func (n *ResourceTreeNode) MouseUp(e *desktop.MouseEvent) {}

// Tapped handles a tap on the node.
//
// The mouse is handled through the desktop driver, which knows about the
// modifier keys and detects the double clicks. The touch drivers only
// report taps, so they are handled here.
func (n *ResourceTreeNode) Tapped(*fyne.PointEvent) {
	if !fyne.CurrentDevice().IsMobile() {
		return
	}
	n.tree.selectOnly(n.uid)
}

// DoubleTapped handles a double tap on the node, which toggles the
// selection of the node.
func (n *ResourceTreeNode) DoubleTapped(*fyne.PointEvent) {
	if !fyne.CurrentDevice().IsMobile() {
		return
	}
	n.tree.ToggleSelection(n.uid)
}

func (n *ResourceTreeNode) CreateRenderer() fyne.WidgetRenderer {
	th := n.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	background := canvas.NewRectangle(th.Color(theme.ColorNameSelection, v))
	background.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	background.Hide()
	return &resourceTreeNodeRenderer{n, background}
}

func (n *ResourceTreeNode) Indent() float32 {
	th := n.Theme()
	depth := len(n.tree.AllParentsOf(n.uid)) - 1
	return float32(depth) * (th.Size(theme.SizeNameInlineIcon) + th.Size(theme.SizeNamePadding))
}

var _ fyne.WidgetRenderer = (*resourceTreeNodeRenderer)(nil)

type resourceTreeNodeRenderer struct {
	node       *ResourceTreeNode
	background *canvas.Rectangle
}

func (r *resourceTreeNodeRenderer) Layout(size fyne.Size) {
	th := r.node.Theme()
	pad := th.Size(theme.SizeNamePadding)
	iconSize := th.Size(theme.SizeNameInlineIcon)
	x := pad + r.node.Indent() + iconSize + pad
	y := float32(0)
	r.background.Resize(fyne.NewSize(size.Width+x, size.Height))
	r.background.Move(fyne.NewPos(0-x, y))
	if r.node.content != nil {
		r.node.content.Resize(size)
		r.node.content.Move(fyne.NewPos(0, 0))
	}
}

func (r *resourceTreeNodeRenderer) MinSize() fyne.Size {
	return r.node.content.MinSize()
}

func (r *resourceTreeNodeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background, r.node.content}
}

func (r *resourceTreeNodeRenderer) Refresh() {
	th := r.node.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	r.background.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	if r.node.tree.IsVisiblySelected(r.node.uid) {
		r.background.FillColor = th.Color(theme.ColorNameSelection, v)
		r.background.Show()
	} else {
		r.background.Hide()
	}
	r.background.Refresh()
	r.node.content.Refresh()
	r.Layout(r.node.Size())
	canvas.Refresh(r.node)
}

func (r *resourceTreeNodeRenderer) Destroy() {}
