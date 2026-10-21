package named_test

import (
	"image"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
)

func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

// buildUI returns the same user interface, with and without the wrapper.
func buildUI(wrap bool, tapped *bool) fyne.CanvasObject {
	btn := widget.NewButton("Click me", func() {
		*tapped = true
	})
	entry := widget.NewEntry()
	entry.SetPlaceHolder("type here")

	tree := widget.NewTreeWithStrings(map[string][]string{
		"": {"child"},
	})

	inner := container.NewVBox(
		widget.NewLabel("a label"),
		btn,
		entry,
		tree,
	)
	if !wrap {
		return inner
	}
	return named.New("panel", inner)
}

func render(t *testing.T, content fyne.CanvasObject) image.Image {
	t.Helper()
	w := test.NewWindow(content)
	w.Resize(fyne.NewSize(240, 160))
	t.Cleanup(w.Close)
	return w.Canvas().Capture()
}

// TestWrapperRendersIdentically checks that wrapping an object does not change
// the way it is rendered.
func TestWrapperRendersIdentically(t *testing.T) {
	plain := render(t, buildUI(false, new(bool)))
	wrapped := render(t, buildUI(true, new(bool)))

	if plain.Bounds() != wrapped.Bounds() {
		t.Fatalf("different image sizes: %v vs %v", plain.Bounds(), wrapped.Bounds())
	}

	diff := 0
	for y := 0; y < plain.Bounds().Dy(); y++ {
		for x := 0; x < plain.Bounds().Dx(); x++ {
			r1, g1, b1, a1 := plain.At(x, y).RGBA()
			r2, g2, b2, a2 := wrapped.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				diff++
			}
		}
	}
	if diff > 0 {
		t.Errorf("%d pixels differ between the wrapped and the unwrapped interface", diff)
	}
}

// TestWrappedButtonIsTappable checks that a wrapped widget still receives the
// mouse events.
func TestWrappedButtonIsTappable(t *testing.T) {
	tapped := false
	btn := widget.NewButton("Click me", func() {
		tapped = true
	})
	w := test.NewWindow(container.NewVBox(named.New("button", btn)))
	w.Resize(fyne.NewSize(240, 160))
	t.Cleanup(w.Close)

	abs := fyne.CurrentApp().Driver().AbsolutePositionForObject(btn)
	center := abs.Add(fyne.NewPos(btn.Size().Width/2, btn.Size().Height/2))
	test.TapCanvas(w.Canvas(), center)

	if !tapped {
		t.Error("the wrapped button did not receive the click")
	}
}

// TestWrappedContainerIsLaidOut checks that the children of a wrapped
// container are laid out as if it was added to its parent directly.
func TestWrappedContainerIsLaidOut(t *testing.T) {
	inner := container.NewVBox(widget.NewLabel("a label"))
	wrapped := named.New("panel", inner)

	w := test.NewWindow(container.NewVBox(wrapped))
	w.Resize(fyne.NewSize(240, 160))
	t.Cleanup(w.Close)

	if wrapped.Size().Height == 0 {
		t.Fatal("the wrapper was not laid out")
	}
	if inner.Size() != wrapped.Size() {
		t.Errorf("the wrapped container has size %v, the wrapper has %v", inner.Size(), wrapped.Size())
	}
}

// TestCacheFindsNamedObjects checks that a name can be resolved to the object
// it belongs to.
func TestCacheFindsNamedObjects(t *testing.T) {
	btn := widget.NewButton("Click me", nil)
	inner := container.NewVBox(named.New("button", btn))
	root := container.NewVBox(named.New("panel", inner))

	cache := named.NewCache(func() fyne.CanvasObject { return root })

	if got := cache.Get("button"); got != fyne.CanvasObject(btn) {
		t.Errorf("Get(button) = %v, want the button", got)
	}
	if got := cache.Get("panel"); got != fyne.CanvasObject(inner) {
		t.Errorf("Get(panel) = %v, want the container", got)
	}
	if got := cache.Get("nothing"); got != nil {
		t.Errorf("Get(nothing) = %v, want nil", got)
	}
}

// TestCacheFindsNamedObjectsInASplit checks that a name is found in the parts
// of a split pane.
func TestCacheFindsNamedObjectsInASplit(t *testing.T) {
	tree := widget.NewTreeWithStrings(map[string][]string{"": {"child"}})
	left := container.NewPadded(named.New("resourceTree", tree))
	right := container.NewPadded(named.New("resourcePane", container.NewStack()))

	root := container.NewVBox(container.NewHSplit(left, right))
	cache := named.NewCache(func() fyne.CanvasObject { return root })

	if got := cache.Get("resourceTree"); got != fyne.CanvasObject(tree) {
		t.Errorf("Get(resourceTree) = %v, want the tree", got)
	}
	if got := cache.Get("resourcePane"); got == nil {
		t.Error("Get(resourcePane) = nil, want the pane")
	}
}

// TestCacheFollowsRebuiltInterface checks that a name resolves to the object
// of the current interface after it was rebuilt.
func TestCacheFollowsRebuiltInterface(t *testing.T) {
	var root fyne.CanvasObject
	cache := named.NewCache(func() fyne.CanvasObject { return root })

	first := widget.NewButton("first", nil)
	root = container.NewVBox(named.New("button", first))
	if got := cache.Get("button"); got != fyne.CanvasObject(first) {
		t.Fatalf("Get(button) = %v, want the first button", got)
	}

	second := widget.NewButton("second", nil)
	root = container.NewVBox(named.New("button", second))
	cache.Reset()
	if got := cache.Get("button"); got != fyne.CanvasObject(second) {
		t.Errorf("Get(button) = %v, want the second button", got)
	}
}

// TestThemeChangeReachesWrappedContent checks that a theme change is applied
// to the content of a named wrapper.
func TestThemeChangeReachesWrappedContent(t *testing.T) {
	rect := widgets.NewThemedRect(theme.ColorNameHeaderBackground, 4)
	inner := container.NewStack(rect)

	w := test.NewWindow(container.NewVBox(named.New("panel", inner)))
	w.Resize(fyne.NewSize(240, 160))
	t.Cleanup(w.Close)

	fyne.CurrentApp().Settings().SetTheme(theme.LightTheme())
	light := rect.Rectangle.FillColor

	fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
	dark := rect.Rectangle.FillColor

	if light == dark {
		t.Errorf("the theme did not reach the wrapped content (%v)", light)
	}
}

// TestWrappedEntryCanBeFocused checks that a named widget can still take the
// keyboard focus.
func TestWrappedEntryCanBeFocused(t *testing.T) {
	// the same test without the wrapper, as a baseline
	plain := widget.NewEntry()
	pw := test.NewWindow(container.NewVBox(plain))
	pw.Resize(fyne.NewSize(240, 160))
	t.Cleanup(pw.Close)
	pw.Canvas().Focus(plain)
	if pw.Canvas().Focused() != plain {
		t.Fatal("the unwrapped entry could not be focused either")
	}

	entry := widget.NewEntry()

	w := test.NewWindow(container.NewVBox(named.New("entry", entry)))
	w.Resize(fyne.NewSize(240, 160))
	t.Cleanup(w.Close)

	w.Canvas().Focus(entry)
	if w.Canvas().Focused() != entry {
		t.Error("the wrapped entry could not be focused")
	}
}
