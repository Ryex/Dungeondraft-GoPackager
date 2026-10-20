// Package uitest provides a declarative DSL for asserting the layout and the
// contents of a Fyne user interface.
//
// The DSL mirrors the shape of a written UI specification: a tree of
// expectations, each of them naming a part of the interface together with the
// constraints it has to satisfy.
//
//	uitest.Expect(t, uitest.In(mainColumn),
//	    uitest.Group("when the main window is shown",
//	        uitest.Has("a welcome header", uitest.Object(a.ui.Header)).Topmost(),
//	        uitest.Has("a theme toggle", uitest.Object(a.ui.ThemeToggle),
//	            uitest.Has("a language select dropdown", uitest.Object(a.ui.LanguageSelect)).
//	                RightOf(uitest.Object(a.ui.ThemeToggle)),
//	        ),
//	    ),
//	)
//
// # Scope rules
//
//   - A selector is resolved against the scope root and its descendants.
//   - Has requires exactly one match, HasAny requires at least one.
//   - Nested expectations are evaluated in the scope of the matched object.
//     When the matched object is not a container the parent scope is used
//     instead, because a specification groups sibling widgets under each
//     other ("with a language select dropdown to the right of the theme
//     toggle" describes two siblings, not a parent and a child).
//   - Relations compare positions relative to the scope root, so they keep
//     working across nesting levels.
//
// The package deliberately does not import internal/gui: the tests that use
// it live in package gui, so importing it here would be an import cycle.
package uitest

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Scope is the part of the interface an expectation is evaluated against.
type Scope struct {
	// Root is the object the scope covers.
	Root fyne.CanvasObject
	// App is the application the scope belongs to, if known.
	App fyne.App
}

// In returns a scope covering the given object and its descendants.
func In(root fyne.CanvasObject) *Scope {
	return &Scope{Root: root, App: fyne.CurrentApp()}
}

// Window returns a scope covering the content of the given window.
func Window(w fyne.Window) *Scope {
	return &Scope{Root: w.Content(), App: fyne.CurrentApp()}
}

// Position returns the position of o relative to the root of the scope.
// The second return value is false when o is not part of the scope.
func (s *Scope) Position(o fyne.CanvasObject) (fyne.Position, bool) {
	return offsetOf(s.Root, o)
}

// Bounds returns the position and the size of o relative to the root of the
// scope. The last return value is false when o is not part of the scope.
func (s *Scope) Bounds(o fyne.CanvasObject) (fyne.Position, fyne.Size, bool) {
	pos, ok := s.Position(o)
	if !ok {
		return fyne.Position{}, fyne.Size{}, false
	}
	return pos, o.Size(), true
}

// Selector locates the widgets an expectation applies to.
type Selector interface {
	// Select returns the objects of the scope matching the selector.
	Select(s *Scope) []fyne.CanvasObject
	// String describes the selector, for use in failure messages.
	String() string
}

// Relation is a geometric constraint a selected widget has to satisfy.
type Relation interface {
	// Check verifies that the subject satisfies the relation.
	Check(subject fyne.CanvasObject, s *Scope) error
	// String describes the relation, for use in failure messages.
	String() string
}

// Node is a single expectation of a specification tree.
type Node interface {
	// Check evaluates the node and returns the failures it found.
	Check(s *Scope) []Failure
	// String describes the node, for use in failure messages.
	String() string
}

// children returns the child objects of o that are part of the object tree.
//
// Widgets that render their children virtually (lists and trees) do not
// expose them here; they are located by their own type instead.
func children(o fyne.CanvasObject) []fyne.CanvasObject {
	switch c := o.(type) {
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

// objects returns o and all of its descendants.
func objects(o fyne.CanvasObject) []fyne.CanvasObject {
	if o == nil {
		return nil
	}
	found := []fyne.CanvasObject{o}
	for _, child := range children(o) {
		found = append(found, objects(child)...)
	}
	return found
}

// isContainer reports whether o holds other objects of the object tree.
func isContainer(o fyne.CanvasObject) bool {
	return len(children(o)) > 0
}

// ancestorsOf returns the objects the target is nested in, outermost first.
func ancestorsOf(root, target fyne.CanvasObject) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	var walk func(cur fyne.CanvasObject) bool
	walk = func(cur fyne.CanvasObject) bool {
		for _, child := range children(cur) {
			if child == target {
				found = append(found, cur)
				return true
			}
			if walk(child) {
				found = append(found, cur)
				return true
			}
		}
		return false
	}
	walk(root)
	slices.Reverse(found)
	return found
}

// offsetOf returns the position of target relative to root.
func offsetOf(root, target fyne.CanvasObject) (fyne.Position, bool) {
	if root == nil || target == nil {
		return fyne.Position{}, false
	}
	return offsetIn(root, target, fyne.NewPos(0, 0))
}

func offsetIn(cur, target fyne.CanvasObject, acc fyne.Position) (fyne.Position, bool) {
	if cur == target {
		return acc, true
	}
	for _, child := range children(cur) {
		if off, ok := offsetIn(child, target, acc.Add(child.Position())); ok {
			return off, true
		}
	}
	return fyne.Position{}, false
}

// describeObject renders a short, human readable description of a widget.
func describeObject(o fyne.CanvasObject) string {
	if o == nil {
		return "<nil>"
	}
	switch c := o.(type) {
	case *widget.Button:
		return fmt.Sprintf("*widget.Button %q", c.Text)
	case *widget.Entry:
		return fmt.Sprintf("*widget.Entry %q", c.PlaceHolder)
	case *widget.Label:
		return fmt.Sprintf("*widget.Label %q", c.Text)
	case *widget.Check:
		return fmt.Sprintf("*widget.Check %q", c.Text)
	case *widget.Select:
		return fmt.Sprintf("*widget.Select %q", c.Selected)
	case *widget.Activity:
		return "*widget.Activity"
	case *widget.ProgressBar:
		return "*widget.ProgressBar"
	case *container.Split:
		return "*container.Split"
	}
	return fmt.Sprintf("%T", o)
}
