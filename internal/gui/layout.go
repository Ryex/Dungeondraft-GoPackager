package gui

import (
	"fyne.io/fyne/v2"
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

type leftExpandHBoxLayout struct {
	paddingFunc func() float32
}

func NewLeftExpandHBoxLayout() fyne.Layout {
	return leftExpandHBoxLayout{
		paddingFunc: theme.Padding,
	}
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

type bottomExpandVBoxLayout struct {
	paddingFunc func() float32
}

func NewBottomExpandVBoxLayout() fyne.Layout {
	return bottomExpandVBoxLayout{
		paddingFunc: theme.Padding,
	}
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

