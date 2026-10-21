package uitest

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/resource_tree"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
)

// Object selects exactly the given object. It is the way to address a widget
// that the application exposes by name.
func Object(o fyne.CanvasObject) Selector {
	return objectSelector{obj: o}
}

type objectSelector struct {
	obj fyne.CanvasObject
}

func (s objectSelector) Select(scope *Scope) []fyne.CanvasObject {
	if s.obj == nil {
		return nil
	}
	if _, ok := offsetOf(scope.Root, s.obj); ok {
		return []fyne.CanvasObject{s.obj}
	}
	return nil
}

func (s objectSelector) String() string {
	return describeObject(s.obj)
}

// Named selects the part of the interface that was given the given name.
//
// The name is resolved in the scope, so that the expectation only holds when
// the named part of the interface is part of the scope.
func Named(name string) Selector {
	return namedSelector{name: name}
}

type namedSelector struct {
	name string
}

func (s namedSelector) Select(scope *Scope) []fyne.CanvasObject {
	for _, o := range objects(scope.Root) {
		if n, ok := o.(*named.Named); ok && n.Name() == s.name {
			return []fyne.CanvasObject{n.Object()}
		}
	}
	return nil
}

func (s namedSelector) String() string {
	return fmt.Sprintf("the part named %q", s.name)
}

// Tab selects the content of the tab with the given label.
//
// The label is matched exactly, so that a specification names the tab the
// way the interface shows it.
func Tab(label string) Selector {
	return tabSelector{label: label}
}

type tabSelector struct {
	label string
}

func (s tabSelector) Select(scope *Scope) []fyne.CanvasObject {
	for _, o := range objects(scope.Root) {
		tabs, ok := o.(*container.AppTabs)
		if !ok {
			continue
		}
		for _, item := range tabs.Items {
			if item.Text == s.label {
				return []fyne.CanvasObject{item.Content}
			}
		}
	}
	return nil
}

func (s tabSelector) String() string {
	return fmt.Sprintf("the tab %q", s.label)
}

// Widget selects every object of type T in the scope.
func Widget[T fyne.CanvasObject]() Selector {
	return widgetSelector[T]{}
}

type widgetSelector[T fyne.CanvasObject] struct{}

func (s widgetSelector[T]) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		if typed, ok := o.(T); ok {
			found = append(found, typed)
		}
	}
	return found
}

func (s widgetSelector[T]) String() string {
	return fmt.Sprintf("objects of type %T", *new(T))
}

// Button selects the buttons with the given label. The label is matched
// case insensitively, so that a specification may write "Open Pack" for a
// button labelled "Open pack".
func Button(text string) Selector {
	return buttonSelector{text: text}
}

// ButtonExact selects the buttons whose label matches exactly.
func ButtonExact(text string) Selector {
	return buttonSelector{text: text, exact: true}
}

type buttonSelector struct {
	text  string
	exact bool
}

func (s buttonSelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		label, ok := buttonLabel(o)
		if !ok {
			continue
		}
		if s.exact {
			if label == s.text {
				found = append(found, o)
			}
			continue
		}
		if strings.EqualFold(label, s.text) {
			found = append(found, o)
		}
	}
	return found
}

func (s buttonSelector) String() string {
	return fmt.Sprintf("the button %q", s.text)
}

// IconButton selects the buttons showing the icon with the given resource
// name, for example "info.svg" or "Icon.png".
func IconButton(iconName string) Selector {
	return iconButtonSelector{name: iconName}
}

type iconButtonSelector struct {
	name string
}

func (s iconButtonSelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		icon, ok := buttonIcon(o)
		if !ok || icon == nil {
			continue
		}
		if icon.Name() == s.name {
			found = append(found, o)
		}
	}
	return found
}

func (s iconButtonSelector) String() string {
	return fmt.Sprintf("the icon button %q", s.name)
}

// Entry selects the entry widgets. When placeholder is not empty only the
// entries with that placeholder are selected.
func Entry(placeholder string) Selector {
	return entrySelector{placeholder: placeholder}
}

type entrySelector struct {
	placeholder string
}

func (s entrySelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		ph, ok := entryPlaceholder(o)
		if !ok {
			continue
		}
		if s.placeholder == "" || ph == s.placeholder {
			found = append(found, o)
		}
	}
	return found
}

func (s entrySelector) String() string {
	if s.placeholder == "" {
		return "an entry"
	}
	return fmt.Sprintf("the entry with the placeholder %q", s.placeholder)
}

// Label selects the labels with the given text.
func Label(text string) Selector {
	return labelSelector{text: text}
}

type labelSelector struct {
	text string
}

func (s labelSelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		label, ok := o.(*widget.Label)
		if !ok {
			continue
		}
		if strings.EqualFold(label.Text, s.text) {
			found = append(found, o)
		}
	}
	return found
}

func (s labelSelector) String() string {
	return fmt.Sprintf("the label %q", s.text)
}

// ThemedText selects the themed text widgets with the given text.
func ThemedText(text string) Selector {
	return themedTextSelector{text: text}
}

type themedTextSelector struct {
	text string
}

func (s themedTextSelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		themed, ok := o.(*widgets.ThemedText)
		if !ok {
			continue
		}
		if strings.EqualFold(themed.Text.Text, s.text) {
			found = append(found, o)
		}
	}
	return found
}

func (s themedTextSelector) String() string {
	return fmt.Sprintf("the text %q", s.text)
}

// Toggle selects the toggle switches.
func Toggle() Selector {
	return widgetSelector[*widgets.Toggle]{}
}

// Select selects the drop down selects.
func Select() Selector {
	return widgetSelector[*widget.Select]{}
}

// Check selects the check boxes with the given text.
func Check(text string) Selector {
	return checkSelector{text: text}
}

type checkSelector struct {
	text string
}

func (s checkSelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		check, ok := o.(*widget.Check)
		if !ok {
			continue
		}
		if strings.EqualFold(check.Text, s.text) {
			found = append(found, o)
		}
	}
	return found
}

func (s checkSelector) String() string {
	return fmt.Sprintf("the check box %q", s.text)
}

// Tree selects the package resource tree.
func Tree() Selector {
	return widgetSelector[*resource_tree.ResourceTree]{}
}

// Split selects the split panes.
func Split() Selector {
	return widgetSelector[*container.Split]{}
}

// Activity selects the activity indicators.
func Activity() Selector {
	return widgetSelector[*widget.Activity]{}
}

// ProgressBar selects the progress bars.
func ProgressBar() Selector {
	return widgetSelector[*widget.ProgressBar]{}
}

// All selects the objects matching every one of the given selectors.
func All(sels ...Selector) Selector {
	return allSelector{sels: sels}
}

type allSelector struct {
	sels []Selector
}

func (s allSelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		if s.matches(o, scope) {
			found = append(found, o)
		}
	}
	return found
}

func (s allSelector) matches(o fyne.CanvasObject, scope *Scope) bool {
	for _, sel := range s.sels {
		ok := false
		for _, candidate := range sel.Select(scope) {
			if candidate == o {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func (s allSelector) String() string {
	return "all of " + joinSelectors(s.sels)
}

// Any selects the objects matching at least one of the given selectors.
func Any(sels ...Selector) Selector {
	return anySelector{sels: sels}
}

type anySelector struct {
	sels []Selector
}

func (s anySelector) Select(scope *Scope) []fyne.CanvasObject {
	seen := map[fyne.CanvasObject]bool{}
	var found []fyne.CanvasObject
	for _, sel := range s.sels {
		for _, candidate := range sel.Select(scope) {
			if !seen[candidate] {
				seen[candidate] = true
				found = append(found, candidate)
			}
		}
	}
	return found
}

func (s anySelector) String() string {
	return "any of " + joinSelectors(s.sels)
}

// None selects the objects matching none of the given selectors.
func None(sels ...Selector) Selector {
	return noneSelector{sels: sels}
}

type noneSelector struct {
	sels []Selector
}

func (s noneSelector) Select(scope *Scope) []fyne.CanvasObject {
	var found []fyne.CanvasObject
	for _, o := range objects(scope.Root) {
		matched := false
		for _, sel := range s.sels {
			for _, candidate := range sel.Select(scope) {
				if candidate == o {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			found = append(found, o)
		}
	}
	return found
}

func (s noneSelector) String() string {
	return "none of " + joinSelectors(s.sels)
}

func joinSelectors(sels []Selector) string {
	parts := make([]string, 0, len(sels))
	for _, sel := range sels {
		parts = append(parts, sel.String())
	}
	return strings.Join(parts, ", ")
}

// buttonLabel returns the label of a button widget.
func buttonLabel(o fyne.CanvasObject) (string, bool) {
	switch b := o.(type) {
	case *widget.Button:
		return b.Text, true
	case *widgets.ToolTipButton:
		return b.Text, true
	}
	return "", false
}

// buttonIcon returns the icon of a button widget.
func buttonIcon(o fyne.CanvasObject) (fyne.Resource, bool) {
	switch b := o.(type) {
	case *widget.Button:
		return b.Icon, true
	case *widgets.ToolTipButton:
		return b.Icon, true
	}
	return nil, false
}

// entryPlaceholder returns the placeholder of an entry widget.
func entryPlaceholder(o fyne.CanvasObject) (string, bool) {
	switch e := o.(type) {
	case *widget.Entry:
		return e.PlaceHolder, true
	case *widgets.ToolTipEntry:
		return e.PlaceHolder, true
	}
	return "", false
}
