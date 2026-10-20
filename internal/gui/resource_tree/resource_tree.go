package resource_tree

import (
	"path/filepath"
	"slices"
	"strings"

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
	nodeTree map[widget.TreeNodeID][]widget.TreeNodeID
	nodeMap  map[widget.TreeNodeID]ResourceTreeNodeData
	mode     ResourceTreeMode

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
						if ensureLeaf(untaggedPath, fi.ResPath) {
							t.nodeMap[fi.ResPath] = &ResourceFileInfoNode{t, fi}
						}
					} else {
						t.nodeMap[fi.ResPath] = &ResourceFileInfoNode{t, fi}
						for _, tag := range tags.AsSlice() {
							tagPath := "tag://" + tag
							if ensureNode(tagPath) {
								t.nodeMap[tagPath] = &ResourceTagBranch{t, tag, false}
							}
							ensureLeaf(tagPath, fi.ResPath)
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
					if ensureLeaf(tagPath, emptyTag) {
						t.nodeMap[emptyTag] = &ResourceFileInfoNode{t, nil}
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
	for parent := t.ParentOf(node); parent != nil; parent = t.ParentOf(node) {
		parents = append(parents, *parent)
	}
	return parents
}

func (t *ResourceTree) IsSelected(uid widget.TreeNodeID) bool {
	return slices.Contains(t.selected, uid)
}

func (t *ResourceTree) IsParentSelected(uid widget.TreeNodeID) bool {
	for _, parent := range t.AllParentsOf(uid) {
		t.IsSelected(parent)
	}
	return false
}

func (t *ResourceTree) Select(uid widget.TreeNodeID) {
	if len(t.selected) > 0 {
		if slices.Contains(t.selected, uid) {
			return
		}
		t.selected = append(t.selected, uid)
	} else {
		t.selected = []widget.TreeNodeID{uid}
	}
	t.Tree.Select(uid)
	if f := t.OnSelectionChanged; f != nil {
		f()
	}
	t.Refresh()
}

func (t *ResourceTree) Unselect(uid widget.TreeNodeID) {
	if len(t.selected) > 0 {
		if i := slices.Index(t.selected, uid); i != -1 {
			t.selected = slices.Delete(t.selected, i, i+1)
			t.Refresh()
			if f := t.OnUnselected; f != nil {
				f(uid)
			}
			if f := t.OnSelectionChanged; f != nil {
				f()
			}
		}
	}
}

func (t *ResourceTree) UnselectAll() {
	if len(t.selected) == 0 {
		return
	}

	selected := t.selected
	t.selected = nil
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

func (t *ResourceTree) GetSelection() []widget.TreeNodeID {
	dest := make([]widget.TreeNodeID, len(t.selected))
	copy(dest, t.selected)
	return dest
}

func (t *ResourceTree) GetSelectionAndChildData() []ResourceTreeNodeData {
	results := []ResourceTreeNodeData{}
	stack := structures.NewStackFromSlice(t.selected)
	for !stack.IsEmpty() {
		uid, _ := stack.Pop()
		if t.IsBranch(uid) {
			children := t.nodeTree[uid]
			stack.Push(children...)
		}
		if data := t.DataFor(uid); data != nil {
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
	n.data = n.tree.DataFor(uid)
	if n.data != nil {
		n.data.Update(uid, n.content)
	} else {
		logrus.Warnf("data for `%v` is nil", n.uid)
	}
}

func (n *ResourceTreeNode) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (n *ResourceTreeNode) MouseDown(e *desktop.MouseEvent) {
	switch e.Button {
	case desktop.MouseButtonPrimary:
		{
			if e.Modifier&fyne.KeyModifierControl != 0 {
				// unselect
				n.tree.Unselect(n.uid)
			} else if e.Modifier&fyne.KeyModifierShift != 0 {
				// add select
				n.tree.Select(n.uid)
			} else {
				// select single
				n.tree.UnselectAll()
				n.tree.Select(n.uid)
			}
		}
	case desktop.MouseButtonSecondary:
		{
			// show context menu
			selected := slices.Contains(n.tree.selected, n.uid)
			menuItems := []*fyne.MenuItem{}
			if selected {
				menuItems = append(menuItems, fyne.NewMenuItem(lang.X("tree.context.select", "Select"), func() {
					n.tree.Select(n.uid)
				}))
			} else {
				menuItems = append(menuItems, fyne.NewMenuItem(lang.X("tree.context.unselect", "Unselect"), func() {
					n.tree.Select(n.uid)
				}))
			}
			contextMenu := fyne.NewMenu("", menuItems...)
			widget.ShowPopUpMenuAtPosition(contextMenu, fyne.CurrentApp().Driver().CanvasForObject(n), e.AbsolutePosition)
		}
	}
}

func (n *ResourceTreeNode) MouseUp(e *desktop.MouseEvent) {}

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
	if len(r.node.tree.selected) > 0 && slices.Contains(r.node.tree.selected, r.node.uid) {
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
