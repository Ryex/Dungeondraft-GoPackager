package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ThemedRect describes a colored rectangle primitive in a Fyne canvas that refreshes it color with the theme
type ThemedRect struct {
	widget.BaseWidget
	Rectangle *canvas.Rectangle
	Color     fyne.ThemeColorName
}

// Refresh causes this rectangle to be redrawn with its configured state.
func (r *ThemedRect) Refresh() {
	r.Rectangle.FillColor = theme.Color(r.Color)
	r.Rectangle.Refresh()
}

// NewThemedRect returns a new Rectangle instance
func NewThemedRect(color fyne.ThemeColorName, cornerRadius float32) *ThemedRect {
	r := &ThemedRect{
		Color:     color,
		Rectangle: &canvas.Rectangle{FillColor: theme.Color(color), CornerRadius: cornerRadius},
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *ThemedRect) CreateRenderer() fyne.WidgetRenderer {
	return &themedRectRenderer{rect: r}
}

func (r *ThemedRect) SetMinSize(size fyne.Size) {
	r.Rectangle.SetMinSize(size)
}

func (r *ThemedRect) MinSize() fyne.Size {
	return r.Rectangle.MinSize()
}

type themedRectRenderer struct {
	rect *ThemedRect
}

func (r *themedRectRenderer) Layout(size fyne.Size) {
	r.rect.Rectangle.Resize(size)
}

func (r *themedRectRenderer) MinSize() fyne.Size {
	return r.rect.Rectangle.MinSize()
}

func (r *themedRectRenderer) Refresh() {
	canvas.Refresh(r.rect.Rectangle)
}

func (r *themedRectRenderer) BackgroundColor() color.Color {
	return theme.Color(theme.ColorNameBackground)
}

func (r *themedRectRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect.Rectangle}
}

func (r *themedRectRenderer) Destroy() {}

type ThemedText struct {
	widget.BaseWidget
	Text  *canvas.Text
	Color fyne.ThemeColorName
}

func NewThemedText(text string, color fyne.ThemeColorName, sizeoptional ...float32) *ThemedText {
	size := theme.TextSize()
	if len(sizeoptional) > 0 {
		size  = sizeoptional[0]
	}
	t := &ThemedText{
		Color: color,
		Text: &canvas.Text{
			Color:    theme.Color(color),
			Text:     text,
			TextSize: size,
		},
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *ThemedText) Refresh() {
	t.Text.Color = theme.Color(t.Color)
	t.Text.Refresh()
}

func (t *ThemedText) CreateRenderer() fyne.WidgetRenderer {
	return &themedTextRenderer{t}
}

type themedTextRenderer struct {
	text *ThemedText
}

func (t *themedTextRenderer) Layout(size fyne.Size) {
	t.text.Text.Resize(size)
}

func (t *themedTextRenderer) MinSize() fyne.Size {
	return t.text.Text.MinSize()
}

func (t *themedTextRenderer) Refresh() {
	canvas.Refresh(t.text.Text)
}

func (t *themedTextRenderer) BackgroundColor() color.Color {
	return theme.Color(theme.ColorNameBackground)
}

func (t *themedTextRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{t.text.Text}
}

func (t *themedTextRenderer) Destroy() {}
