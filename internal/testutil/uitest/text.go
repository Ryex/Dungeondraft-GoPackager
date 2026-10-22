package uitest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
)

// TextStart returns the position of the start of the text a widget shows,
// relative to the widget itself.
//
// The text of a label is inset by the inner padding of the theme, while the
// text of a plain text object is drawn at the position of the object, so the
// text does not start at the same place for every kind of widget.
//
// The widget must have been laid out, otherwise its text is not placed yet.
func TextStart(w fyne.Widget) fyne.Position {
	start, _ := textStart(test.WidgetRenderer(w).Objects(), fyne.NewPos(0, 0))
	return start
}

// textStart returns the leftmost position of the texts of the given objects.
func textStart(objs []fyne.CanvasObject, acc fyne.Position) (fyne.Position, bool) {
	var start fyne.Position
	found := false

	for _, obj := range objs {
		switch o := obj.(type) {
		case *canvas.Text:
			pos := acc.Add(o.Position())
			if !found || pos.X < start.X {
				start = pos
				found = true
			}
		case fyne.Widget:
			inner, ok := textStart(test.WidgetRenderer(o).Objects(), acc.Add(o.Position()))
			if !ok {
				continue
			}
			if !found || inner.X < start.X {
				start = inner
				found = true
			}
		default:
			inner, ok := textStart(named.Children(obj), acc.Add(obj.Position()))
			if !ok {
				continue
			}
			if !found || inner.X < start.X {
				start = inner
				found = true
			}
		}
	}

	return start, found
}
