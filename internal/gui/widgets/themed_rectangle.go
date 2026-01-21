package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

// Declare conformity with CanvasObject interface
var _ fyne.CanvasObject = (*ThemedRectangle)(nil)

// ThemedRectangle describes a colored rectangle primitive in a Fyne canvas that refreshes it color with the theme
type ThemedRectangle struct {
	canvas.Rectangle
	Color fyne.ThemeColorName
}

// Refresh causes this rectangle to be redrawn with its configured state.
func (r *ThemedRectangle) Refresh() {
	r.FillColor = theme.Color(r.Color)
	r.Rectangle.Refresh()
}

// NewRectangle returns a new Rectangle instance
func NewRectangle(color fyne.ThemeColorName) *ThemedRectangle {
	return &ThemedRectangle{
		Color:     color,
		Rectangle: canvas.Rectangle{FillColor: theme.Color(color)},
	}
}
