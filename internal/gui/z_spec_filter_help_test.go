package gui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
)

// filterHelpPopup returns the pop up of the filter help dialog that is open
// over the window.
func filterHelpPopup(t *testing.T, a *App) *widget.PopUp {
	t.Helper()

	overlay := a.window.Canvas().Overlays().Top()
	require.NotNil(t, overlay, "the help of the filter language must be open")

	renderer := test.WidgetRenderer(overlay.(fyne.Widget))
	for _, obj := range renderer.Objects() {
		if popup, isPopup := obj.(*widget.PopUp); isPopup {
			return popup
		}
	}

	t.Fatal("the help of the filter language must be open")
	return nil
}

// filterHelpPart returns the part of the filter help dialog with the given
// name.
//
// The dialog is shown over the window, so its parts are not part of the tree
// of the named parts of the window and need a cache of their own.
func filterHelpPart(t *testing.T, popup *widget.PopUp, name string) fyne.CanvasObject {
	t.Helper()

	cache := named.NewCache(func() fyne.CanvasObject {
		return popup.Content
	})
	obj := cache.Get(name)
	require.NotNil(t, obj, "the help of the filter language has no part named %q", name)
	return obj
}

// filterHelpSectionTitle returns the title of the section of the filter help
// with the given text.
func filterHelpSectionTitle(t *testing.T, scroll *container.Scroll, text string) *widget.RichText {
	t.Helper()

	var found *widget.RichText
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if found != nil {
			return
		}
		if rich, isRich := o.(*widget.RichText); isRich && rich.String() == text {
			found = rich
			return
		}
		for _, child := range named.Children(o) {
			walk(child)
		}
	}
	walk(scroll.Content)

	require.NotNil(t, found, "the help of the filter language has no section titled %q", text)
	return found
}

// waitForStableDialogLayout waits until the layout of the dialog stops
// changing.
func waitForStableDialogLayout(t *testing.T, popup *widget.PopUp) {
	t.Helper()

	var last string
	stable := 0
	require.Eventually(t, func() bool {
		current := uitest.LayoutFingerprint(popup.Content)
		if current == last {
			stable++
			return stable >= 2
		}
		last = current
		stable = 0
		return false
	}, 10*time.Second, 10*time.Millisecond, "the layout of the dialog must settle")
}

// TestSpecFilterHelp verifies the "when the help of the filter language is
// opened" section of the UI specification.
//
// Covered by this test:
//
//	when the filter of the resource tree is changed
//	  - the filter entry has a help button to its right
//	when the help of the filter language is opened
//	  - the dialog has a title
//	  - and a close button
//	  - and a list of the sections on the left
//	  - and the help of the filter language on the right
//	  - clicking a section scrolls the help to that section
func TestSpecFilterHelp(t *testing.T) {
	a := newSpecApp(t)
	dir := packFixtureDir(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	// the help button of the filter is to the right of the filter entry
	uitest.Expect(t, uitest.In(namedPart(t, a, nameSplitPane)),
		uitest.Group("when the filter of the resource tree is changed",
			uitest.Has("a filter entry", uitest.Named(nameFilterEntry)),
			uitest.Has("a help button to the right of the filter entry",
				uitest.Named(nameFilterHelpButton)).RightOf(uitest.Named(nameFilterEntry)),
		),
	)

	// the help is opened by the help button
	helpBtn, ok := namedPart(t, a, nameFilterHelpButton).(fyne.Tappable)
	require.True(t, ok, "the help button of the filter must be tappable")
	test.Tap(helpBtn)

	popup := filterHelpPopup(t, a)
	waitForStableDialogLayout(t, popup)

	uitest.Expect(t, uitest.In(popup),
		uitest.Group("when the help of the filter language is opened",
			uitest.Has("a title",
				uitest.Label(lang.X("tree.filter.help.title", "Filter Language")),
			),
			uitest.Has("a close button",
				uitest.Button(lang.X("tree.filter.help.close", "Close")),
			),
			uitest.Has("a list of the sections on the left",
				uitest.Named(nameFilterHelpSectionList),
			).LeftOf(uitest.Named(nameFilterHelpScroll)),
			uitest.Has("the help of the filter language on the right",
				uitest.Named(nameFilterHelpScroll),
			),
		),
	)

	// the list shows a heading for every section of the help
	list, ok := filterHelpPart(t, popup, nameFilterHelpSectionList).(*widget.List)
	require.True(t, ok, "the list of the sections must be a list")
	require.Equal(t, 6, list.Length(), "the list must show every section of the help")

	scroll, ok := filterHelpPart(t, popup, nameFilterHelpScroll).(*container.Scroll)
	require.True(t, ok, "the help of the filter language must be scrollable")

	// clicking a section scrolls the help to it
	kindTitle := lang.X("tree.filter.help.kind.title", "Kind")
	title := filterHelpSectionTitle(t, scroll, kindTitle)

	list.Select(3)
	waitForStableDialogLayout(t, popup)

	pos, ok := uitest.In(scroll).Position(title)
	require.True(t, ok, "the title of the section must be part of the help")
	assert.InDelta(t, 0, pos.Y, 1,
		"the title of the section must be at the top of the help")
}
