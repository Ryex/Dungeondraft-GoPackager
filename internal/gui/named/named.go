// Package named gives names to the parts of a Fyne user interface.
//
// The parts of the interface are named where they are created, so that they
// can be looked up later without keeping a reference to every one of them in
// a central struct.
package named

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Named wraps a canvas object and carries a name with it.
//
// The wrapper is transparent: the wrapped object keeps its appearance, it is
// laid out by the parent container as if it was added to it directly, and it
// still receives the mouse and keyboard events. It is only moved one level
// down in the object tree.
type Named struct {
	// BaseWidget gives the wrapper the behaviour Fyne expects from a
	// widget: it keeps the position and the size, and it makes sure that
	// the renderer of the wrapper is registered with Fyne, so that the
	// wrapper is part of the object tree for every walk over it.
	widget.BaseWidget

	inner fyne.CanvasObject
	name  string

	renderer *renderer
}

var _ fyne.Widget = (*Named)(nil)

// New wraps obj and gives it the given name.
func New(name string, obj fyne.CanvasObject) *Named {
	n := &Named{
		inner: obj,
		name:  name,
	}
	n.ExtendBaseWidget(n)
	n.renderer = &renderer{owner: n}
	return n
}

// Name returns the name of the wrapped object.
func (n *Named) Name() string {
	return n.name
}

// Object returns the wrapped object.
func (n *Named) Object() fyne.CanvasObject {
	return n.inner
}

// Visible reports whether the wrapper and the wrapped object are visible.
func (n *Named) Visible() bool {
	return n.BaseWidget.Visible() && n.inner.Visible()
}

// Show shows the wrapper and the wrapped object.
func (n *Named) Show() {
	n.BaseWidget.Show()
	n.inner.Show()
}

// Hide hides the wrapper and the wrapped object.
func (n *Named) Hide() {
	n.BaseWidget.Hide()
	n.inner.Hide()
}

// CreateRenderer returns the renderer of the wrapper. It exposes the wrapped
// object as the only child of the wrapper, so that Fyne keeps treating the
// wrapped object as a part of the object tree.
func (n *Named) CreateRenderer() fyne.WidgetRenderer {
	return n.renderer
}

// renderer draws the wrapped object as the only child of the wrapper.
type renderer struct {
	owner *Named
}

func (r *renderer) Layout(size fyne.Size) {
	r.owner.inner.Resize(size)
}

func (r *renderer) MinSize() fyne.Size {
	return r.owner.inner.MinSize()
}

func (r *renderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.owner.inner}
}

func (r *renderer) Refresh() {
	r.owner.inner.Refresh()
}

func (r *renderer) Destroy() {}

// Children returns the child objects of o that are part of the object tree.
//
// Widgets that render their children virtually (lists and trees) do not
// expose them here; they are located by their own type instead. The children
// of a widget are not returned either, because they belong to the renderer of
// the widget: a named part has to be a child of a container to be found.
func Children(o fyne.CanvasObject) []fyne.CanvasObject {
	switch c := o.(type) {
	case *Named:
		return []fyne.CanvasObject{c.inner}
	case *fyne.Container:
		return c.Objects
	case *container.Split:
		return []fyne.CanvasObject{c.Leading, c.Trailing}
	case *container.Scroll:
		return []fyne.CanvasObject{c.Content}
	case *container.AppTabs:
		items := make([]fyne.CanvasObject, 0, len(c.Items))
		for _, item := range c.Items {
			items = append(items, item.Content)
		}
		return items
	case *widget.PopUp:
		return []fyne.CanvasObject{c.Content}
	}
	return nil
}
