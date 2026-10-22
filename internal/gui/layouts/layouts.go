package layouts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

func isHorizontalSpacer(obj fyne.CanvasObject) bool {
	spacer, ok := obj.(layout.SpacerObject)
	return ok && spacer.ExpandHorizontal()
}

func isVerticalSpacer(obj fyne.CanvasObject) bool {
	spacer, ok := obj.(layout.SpacerObject)
	return ok && spacer.ExpandVertical()
}

// ** Padded container **

// paddedLayout puts the objects in a container with a padding of its own on
// every side.
type paddedLayout struct {
	top, right, bottom, left float32
}

// NewPaddedLayout returns a layout that pads the objects with the given
// padding on every side.
func NewPaddedLayout(top, right, bottom, left float32) fyne.Layout {
	return paddedLayout{top: top, right: right, bottom: bottom, left: left}
}

// NewPadded returns a container with the given padding on every side.
func NewPadded(top, right, bottom, left float32, objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewPaddedLayout(top, right, bottom, left), objs...)
}

func (l paddedLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pos := fyne.NewPos(l.left, l.top)
	siz := fyne.NewSize(size.Width-l.left-l.right, size.Height-l.top-l.bottom)
	for _, child := range objects {
		child.Resize(siz)
		child.Move(pos)
	}
}

func (l paddedLayout) MinSize(objects []fyne.CanvasObject) (min fyne.Size) {
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		childMin := child.MinSize()
		min.Width = fyne.Max(childMin.Width, min.Width)
		min.Height = fyne.Max(childMin.Height, min.Height)
	}
	min = min.Add(fyne.NewSize(l.left+l.right, l.top+l.bottom))
	return min
}

// ** Leading object, vertically centered **

// leadingCenterLayout puts the first object on the left, vertically centered,
// and the other objects next to it.
type leadingCenterLayout struct {
	padding float32
}

// NewLeadingCenterLayout returns a layout that puts the first object on the
// left, vertically centered, and the other objects next to it.
func NewLeadingCenterLayout(padding float32) fyne.Layout {
	return leadingCenterLayout{padding}
}

// NewLeadingCenter returns a container that puts the first object on the
// left, vertically centered, and the other objects next to it.
func NewLeadingCenter(leading fyne.CanvasObject, padding float32, objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewLeadingCenterLayout(padding), append([]fyne.CanvasObject{leading}, objs...)...)
}

func (l leadingCenterLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	leading := objects[0]
	leadingWidth := leading.MinSize().Width + l.padding
	leading.Resize(fyne.NewSize(leadingWidth, size.Height))
	leading.Move(fyne.NewPos(0, 0))

	for _, obj := range objects[1:] {
		obj.Resize(fyne.NewSize(size.Width-leadingWidth, size.Height))
		obj.Move(fyne.NewPos(leadingWidth, 0))
	}
}

func (l leadingCenterLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}

	leading := objects[0].MinSize()
	min := fyne.NewSize(leading.Width, leading.Height)
	for _, obj := range objects[1:] {
		objMin := obj.MinSize()
		min.Width += objMin.Width
		min.Height = fyne.Max(min.Height, objMin.Height)
	}
	return min
}

// ** Stacked container **

// stackedLayout puts the objects on top of each other with a spacing of its
// own between them.
type stackedLayout struct {
	spacing float32
}

// NewStackedLayout returns a layout that stacks the objects with the given
// spacing between them.
func NewStackedLayout(spacing float32) fyne.Layout {
	return stackedLayout{spacing: spacing}
}

// NewStacked returns a container that stacks the objects with the given
// spacing between them.
func NewStacked(spacing float32, objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewStackedLayout(spacing), objs...)
}

func (l stackedLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}

		height := obj.MinSize().Height
		obj.Resize(fyne.NewSize(size.Width, height))
		obj.Move(fyne.NewPos(0, y))
		y += height + l.spacing
	}
}

func (l stackedLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	min := fyne.NewSize(0, 0)
	visible := 0
	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}

		objMin := obj.MinSize()
		min.Width = fyne.Max(objMin.Width, min.Width)
		min.Height += objMin.Height
		visible++
	}
	if visible > 1 {
		min.Height += l.spacing * float32(visible-1)
	}
	return min
}

// ** Fixed size container **

// fixedSizeLayout puts the objects in a container of a fixed size.
type fixedSizeLayout struct {
	size fyne.Size
}

// NewFixedSizeLayout returns a layout that puts the objects in a container of
// the given size.
func NewFixedSizeLayout(size fyne.Size) fyne.Layout {
	return fixedSizeLayout{size: size}
}

// NewFixedSize returns a container of the given size.
func NewFixedSize(size fyne.Size, objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewFixedSizeLayout(size), objs...)
}

func (l fixedSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		obj.Resize(l.size)
		obj.Move(fyne.NewPos(0, 0))
	}
}

func (l fixedSizeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return l.size
}

// ** Left expand HBox **

type leftExpandHBoxLayout struct {
	paddingFunc func() float32
}

func NewLeftExpandHBoxLayout() fyne.Layout {
	return leftExpandHBoxLayout{
		paddingFunc: theme.Padding,
	}
}

func NewLeftExpandHBox(objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewLeftExpandHBoxLayout(), objs...)
}

func (g leftExpandHBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	visibleObjects := 0
	// Size taken up by rightmost objects
	rightmost := float32(0)

	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isHorizontalSpacer(child) {
			spacers++
			continue
		}

		visibleObjects++
		if i != 0 {
			rightmost += child.MinSize().Width
		}
	}

	padding := g.paddingFunc()

	leftExtra := size.Width - rightmost - (float32(spacers) * padding) - (padding * float32(visibleObjects-1))
	extra := size.Width - rightmost - leftExtra - (padding * float32(visibleObjects-1))

	// Spacers split extra space equally
	spacerSize := float32(0)
	if spacers > 0 {
		spacerSize = extra / float32(spacers)
	}

	x, y := float32(0), float32(0)
	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isHorizontalSpacer(child) {
			x += spacerSize
			continue
		}
		child.Move(fyne.NewPos(x, y))

		width := child.MinSize().Width
		if i == 0 {
			width = leftExtra
		}
		x += padding + width
		child.Resize(fyne.NewSize(width, size.Height))
	}
}

func (g leftExpandHBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := g.paddingFunc()
	for _, child := range objects {
		if !child.Visible() || isHorizontalSpacer(child) {
			continue
		}

		childMin := child.MinSize()
		minSize.Height = fyne.Max(childMin.Height, minSize.Height)
		minSize.Width += childMin.Width
		if addPadding {
			minSize.Width += padding
		}
		addPadding = true
	}
	return minSize
}

// ** right expand HBox **

type rightExpandHBoxLayout struct {
	paddingFunc func() float32
}

func NewRightExpandHBoxLayout() fyne.Layout {
	return rightExpandHBoxLayout{
		paddingFunc: theme.Padding,
	}
}

func NewRightExpandHBox(objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewRightExpandHBoxLayout(), objs...)
}

func (g rightExpandHBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	visibleObjects := 0
	// Size taken up by rightmost objects
	leftmost := float32(0)

	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isHorizontalSpacer(child) {
			spacers++
			continue
		}

		visibleObjects++
		if i != len(objects)-1 {
			leftmost += child.MinSize().Width
		}
	}

	padding := g.paddingFunc()

	rightExtra := size.Width - leftmost - (float32(spacers) * padding) - (padding * float32(visibleObjects-1))
	extra := size.Width - leftmost - rightExtra - (padding * float32(visibleObjects-1))

	// Spacers split extra space equally
	spacerSize := float32(0)
	if spacers > 0 {
		spacerSize = extra / float32(spacers)
	}

	x, y := float32(0), float32(0)
	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isHorizontalSpacer(child) {
			x += spacerSize
			continue
		}
		child.Move(fyne.NewPos(x, y))

		width := child.MinSize().Width
		if i == len(objects)-1 {
			width = rightExtra
		}
		x += padding + width
		child.Resize(fyne.NewSize(width, size.Height))
	}
}

func (g rightExpandHBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := g.paddingFunc()
	for _, child := range objects {
		if !child.Visible() || isHorizontalSpacer(child) {
			continue
		}

		childMin := child.MinSize()
		minSize.Height = fyne.Max(childMin.Height, minSize.Height)
		minSize.Width += childMin.Width
		if addPadding {
			minSize.Width += padding
		}
		addPadding = true
	}
	return minSize
}

// ** bottom expand VBox **

type bottomExpandVBoxLayout struct {
	paddingFunc func() float32
}

func NewBottomExpandVBoxLayout() fyne.Layout {
	return bottomExpandVBoxLayout{
		paddingFunc: theme.Padding,
	}
}

func NewBottomExpandVBox(objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewBottomExpandVBoxLayout(), objs...)
}

func (v bottomExpandVBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	visibleObjects := 0
	// Size taken up by visible objects
	topmost := float32(0)

	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isVerticalSpacer(child) {
			spacers++
			continue
		}

		visibleObjects++
		if i != len(objects)-1 {
			topmost += child.MinSize().Height
		}
	}

	padding := v.paddingFunc()

	// Amount of space not taken up by visible objects and inter-object padding
	bottomExtra := size.Height - topmost - (float32(spacers) * padding) - (padding * float32(visibleObjects-1))
	extra := size.Height - topmost - bottomExtra - (padding * float32(visibleObjects-1))

	// Spacers split extra space equally
	spacerSize := float32(0)
	if spacers > 0 {
		spacerSize = extra / float32(spacers)
	}

	x, y := float32(0), float32(0)
	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isVerticalSpacer(child) {
			y += spacerSize
			continue
		}
		child.Move(fyne.NewPos(x, y))

		height := child.MinSize().Height
		if i == len(objects)-1 {
			height = bottomExtra
		}
		y += padding + height
		child.Resize(fyne.NewSize(size.Width, height))
	}
}

// MinSize finds the smallest size that satisfies all the child objects.
// For a BoxLayout this is the width of the widest item and the height is
// the sum of all children combined with padding between each.
func (v bottomExpandVBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := v.paddingFunc()
	for _, child := range objects {
		if !child.Visible() || isVerticalSpacer(child) {
			continue
		}

		childMin := child.MinSize()
		minSize.Width = fyne.Max(childMin.Width, minSize.Width)
		minSize.Height += childMin.Height
		if addPadding {
			minSize.Height += padding
		}
		addPadding = true
	}
	return minSize
}

// ** top expand VBox **

type topExpandVBoxLayout struct {
	paddingFunc func() float32
}

func NewTopExpandVBoxLayout() fyne.Layout {
	return topExpandVBoxLayout{
		paddingFunc: theme.Padding,
	}
}

func NewTopExpandVBox(objs ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewTopExpandVBoxLayout(), objs...)
}

func (v topExpandVBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	visibleObjects := 0
	// Size taken up by visible objects
	bottommost := float32(0)

	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isVerticalSpacer(child) {
			spacers++
			continue
		}

		visibleObjects++
		if i != 0 {
			bottommost += child.MinSize().Height
		}
	}

	padding := v.paddingFunc()

	// Amount of space not taken up by visible objects and inter-object padding
	topExtra := size.Height - bottommost - (float32(spacers) * padding) - (padding * float32(visibleObjects-1))
	extra := size.Height - bottommost - topExtra - (padding * float32(visibleObjects-1))

	// Spacers split extra space equally
	spacerSize := float32(0)
	if spacers > 0 {
		spacerSize = extra / float32(spacers)
	}

	x, y := float32(0), float32(0)
	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if isVerticalSpacer(child) {
			y += spacerSize
			continue
		}
		child.Move(fyne.NewPos(x, y))

		height := child.MinSize().Height
		if i == 0 {
			height = topExtra
		}
		y += padding + height
		child.Resize(fyne.NewSize(size.Width, height))
	}
}

// MinSize finds the smallest size that satisfies all the child objects.
// For a BoxLayout this is the width of the widest item and the height is
// the sum of all children combined with padding between each.
func (v topExpandVBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := v.paddingFunc()
	for _, child := range objects {
		if !child.Visible() || isVerticalSpacer(child) {
			continue
		}

		childMin := child.MinSize()
		minSize.Width = fyne.Max(childMin.Width, minSize.Width)
		minSize.Height += childMin.Height
		if addPadding {
			minSize.Height += padding
		}
		addPadding = true
	}
	return minSize
}
