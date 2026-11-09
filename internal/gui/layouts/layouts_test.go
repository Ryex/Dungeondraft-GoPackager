package layouts

import (
	"image/color"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

// newTestObject returns an object with the given minimum size, which the
// layouts have to place.
func newTestObject(min fyne.Size) *canvas.Rectangle {
	obj := canvas.NewRectangle(color.White)
	obj.SetMinSize(min)
	return obj
}

// TestPaddedLayout verifies that the padded layout puts every object
// inside the padding on every side and that its minimum size grows by the
// padding.
func TestPaddedLayout(t *testing.T) {
	l := NewPaddedLayout(10, 20, 30, 40)

	t.Run("layout puts every object inside the padding", func(t *testing.T) {
		first := newTestObject(fyne.NewSize(50, 20))
		second := newTestObject(fyne.NewSize(30, 40))

		l.Layout([]fyne.CanvasObject{first, second}, fyne.NewSize(200, 100))

		// the layout does not order the objects: each one of them gets
		// the whole padded box
		for i, obj := range []fyne.CanvasObject{first, second} {
			assert.Equal(t, fyne.NewPos(40, 10), obj.Position(), "object %d", i)
			assert.Equal(t, fyne.NewSize(140, 60), obj.Size(), "object %d", i)
		}
	})

	t.Run("layout does not skip hidden objects", func(t *testing.T) {
		hidden := newTestObject(fyne.NewSize(500, 500))
		hidden.Hide()

		l.Layout([]fyne.CanvasObject{hidden}, fyne.NewSize(200, 100))

		assert.Equal(t, fyne.NewPos(40, 10), hidden.Position())
		assert.Equal(t, fyne.NewSize(140, 60), hidden.Size())
	})

	t.Run("min size is the biggest object plus the padding", func(t *testing.T) {
		first := newTestObject(fyne.NewSize(50, 20))
		second := newTestObject(fyne.NewSize(30, 40))

		min := l.MinSize([]fyne.CanvasObject{first, second})

		assert.Equal(t, fyne.NewSize(110, 80), min)
	})

	t.Run("min size skips hidden objects", func(t *testing.T) {
		visible := newTestObject(fyne.NewSize(50, 40))
		hidden := newTestObject(fyne.NewSize(500, 500))
		hidden.Hide()

		min := l.MinSize([]fyne.CanvasObject{visible, hidden})

		assert.Equal(t, fyne.NewSize(110, 80), min)
	})

	t.Run("min size of no objects is only the padding", func(t *testing.T) {
		assert.Equal(t, fyne.NewSize(60, 40), l.MinSize(nil))

		l.Layout(nil, fyne.NewSize(200, 100)) // must not panic
	})
}

// TestLeadingCenterLayout verifies that the leading center layout puts the
// first object on the left and the other objects next to it.
func TestLeadingCenterLayout(t *testing.T) {
	l := NewLeadingCenterLayout(8)

	t.Run("layout puts the leading object on the left", func(t *testing.T) {
		leading := newTestObject(fyne.NewSize(40, 30))
		trailing := newTestObject(fyne.NewSize(100, 50))

		l.Layout([]fyne.CanvasObject{leading, trailing}, fyne.NewSize(200, 60))

		// the leading object is as wide as its minimum size plus the
		// padding, the trailing one takes the rest of the width
		assert.Equal(t, fyne.NewPos(0, 0), leading.Position())
		assert.Equal(t, fyne.NewSize(48, 60), leading.Size())
		assert.Equal(t, fyne.NewPos(48, 0), trailing.Position())
		assert.Equal(t, fyne.NewSize(152, 60), trailing.Size())
	})

	t.Run("min size is the widths, the leading padding and the highest object", func(t *testing.T) {
		leading := newTestObject(fyne.NewSize(40, 30))
		trailing := newTestObject(fyne.NewSize(100, 50))

		min := l.MinSize([]fyne.CanvasObject{leading, trailing})

		// Layout gives the leading child its minimum size plus the
		// padding, so the minimum size of the container has to add the
		// padding too, or the trailing child would get less than its own
		// minimum at the reported size
		assert.Equal(t, fyne.NewSize(148, 50), min)
	})

	t.Run("layout does not skip hidden objects", func(t *testing.T) {
		leading := newTestObject(fyne.NewSize(40, 30))
		trailing := newTestObject(fyne.NewSize(100, 50))
		trailing.Hide()

		l.Layout([]fyne.CanvasObject{leading, trailing}, fyne.NewSize(200, 60))

		// the layout does not check the visibility: a hidden object is
		// placed like a visible one
		assert.Equal(t, fyne.NewPos(48, 0), trailing.Position())
		assert.Equal(t, fyne.NewSize(152, 60), trailing.Size())

		// and the min size counts it as well, padding included
		assert.Equal(t, fyne.NewSize(148, 50), l.MinSize([]fyne.CanvasObject{leading, trailing}))
	})

	t.Run("no objects", func(t *testing.T) {
		assert.Equal(t, fyne.NewSize(0, 0), l.MinSize(nil))

		l.Layout(nil, fyne.NewSize(200, 60)) // must not panic
	})
}

// TestStackedLayout verifies that the stacked layout puts the objects on
// top of each other with its spacing between them.
func TestStackedLayout(t *testing.T) {
	l := NewStackedLayout(5)

	t.Run("layout stacks the visible objects", func(t *testing.T) {
		first := newTestObject(fyne.NewSize(40, 30))
		second := newTestObject(fyne.NewSize(20, 50))

		l.Layout([]fyne.CanvasObject{first, second}, fyne.NewSize(200, 500))

		// the objects are as high as their minimum size, the height of
		// the container is not distributed between them
		assert.Equal(t, fyne.NewPos(0, 0), first.Position())
		assert.Equal(t, fyne.NewSize(200, 30), first.Size())
		assert.Equal(t, fyne.NewPos(0, 35), second.Position())
		assert.Equal(t, fyne.NewSize(200, 50), second.Size())
	})

	t.Run("layout skips hidden objects", func(t *testing.T) {
		first := newTestObject(fyne.NewSize(40, 30))
		second := newTestObject(fyne.NewSize(20, 50))
		second.Hide()

		l.Layout([]fyne.CanvasObject{first, second}, fyne.NewSize(200, 500))

		assert.Equal(t, fyne.NewPos(0, 0), first.Position())
		assert.Equal(t, fyne.NewSize(200, 30), first.Size())

		// the hidden object is not placed, it keeps the size it had
		assert.Equal(t, fyne.Position{}, second.Position())
		assert.Equal(t, fyne.Size{}, second.Size())
	})

	t.Run("min size adds the heights of the visible objects", func(t *testing.T) {
		first := newTestObject(fyne.NewSize(40, 30))
		second := newTestObject(fyne.NewSize(20, 50))

		min := l.MinSize([]fyne.CanvasObject{first, second})

		assert.Equal(t, fyne.NewSize(40, 85), min)
	})

	t.Run("min size skips hidden objects and their spacing", func(t *testing.T) {
		visible := newTestObject(fyne.NewSize(40, 30))
		hidden := newTestObject(fyne.NewSize(500, 500))
		hidden.Hide()

		min := l.MinSize([]fyne.CanvasObject{visible, hidden})

		assert.Equal(t, fyne.NewSize(40, 30), min)
	})

	t.Run("min size of no objects is zero", func(t *testing.T) {
		assert.Equal(t, fyne.NewSize(0, 0), l.MinSize(nil))

		l.Layout(nil, fyne.NewSize(200, 500)) // must not panic
	})
}

// TestFixedSizeLayout verifies that the fixed size layout puts every
// object into a box of its own fixed size.
func TestFixedSizeLayout(t *testing.T) {
	l := NewFixedSizeLayout(fyne.NewSize(64, 64))

	t.Run("layout puts every object into the fixed box", func(t *testing.T) {
		first := newTestObject(fyne.NewSize(50, 20))
		second := newTestObject(fyne.NewSize(30, 40))
		hidden := newTestObject(fyne.NewSize(500, 500))
		hidden.Hide()

		l.Layout([]fyne.CanvasObject{first, second, hidden}, fyne.NewSize(200, 100))

		// the layout does not check the visibility: a hidden object is
		// placed like a visible one
		for i, obj := range []fyne.CanvasObject{first, second, hidden} {
			assert.Equal(t, fyne.NewPos(0, 0), obj.Position(), "object %d", i)
			assert.Equal(t, fyne.NewSize(64, 64), obj.Size(), "object %d", i)
		}
	})

	t.Run("min size is the fixed size", func(t *testing.T) {
		obj := newTestObject(fyne.NewSize(50, 20))
		hidden := newTestObject(fyne.NewSize(500, 500))
		hidden.Hide()

		// the minimum size is the fixed size, no matter what the
		// objects of the container need
		assert.Equal(t, fyne.NewSize(64, 64), l.MinSize([]fyne.CanvasObject{obj, hidden}))
	})

	t.Run("min size of no objects is the fixed size", func(t *testing.T) {
		assert.Equal(t, fyne.NewSize(64, 64), l.MinSize(nil))

		l.Layout(nil, fyne.NewSize(200, 100)) // must not panic
	})
}
