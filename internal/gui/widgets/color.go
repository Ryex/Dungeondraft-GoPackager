package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TappableRect struct {
	widget.BaseWidget
	rect   *canvas.Rectangle
	OnTapped func(*fyne.PointEvent)
}

func NewTappableRect(fill color.Color, radius float32) *TappableRect {
	r := &TappableRect{
		rect: &canvas.Rectangle{
			StrokeColor:  color.NRGBA{255, 255, 255, 255},
			StrokeWidth:  1,
			FillColor:    fill,
			CornerRadius: radius,
		},
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *TappableRect) SetColor(c color.Color) {
	r.rect.FillColor = c
	r.Refresh()
}

func (r *TappableRect) GetColor() color.Color {
  return r.rect.FillColor
}

func (r *TappableRect) CreateRenderer() fyne.WidgetRenderer {
	return &tappableRectRenderer{rect: r.rect}
}

func (r *TappableRect) SetMinSize(size fyne.Size) {
	r.rect.SetMinSize(size)
}

func (r *TappableRect) MinSize() fyne.Size {
	return r.rect.MinSize()
}

func (r *TappableRect) Tapped(e *fyne.PointEvent) {
	if r.OnTapped != nil {
		r.OnTapped(e)
	}
}

func (r *TappableRect) TappedSecondary(*fyne.PointEvent) {}

type tappableRectRenderer struct {
	rect *canvas.Rectangle
}

func (r *tappableRectRenderer) Layout(size fyne.Size) {
	r.rect.Resize(size)
}

func (r *tappableRectRenderer) MinSize() fyne.Size {
	return r.rect.MinSize()
}

func (r *tappableRectRenderer) Refresh() {
	canvas.Refresh(r.rect)
}

func (r *tappableRectRenderer) BackgroundColor() color.Color {
	return theme.Color(theme.ColorNameBackground)
}

func (r *tappableRectRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect}
}

func (r *tappableRectRenderer) Destroy() {}
