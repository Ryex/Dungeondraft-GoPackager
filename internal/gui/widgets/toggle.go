package widgets

import (
	"fmt"
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Toggle struct {
	widget.DisableableWidget
	Toggled bool

	togglePLock sync.RWMutex

	OnChanged func(bool) `json:"="`

	focused bool
	hovered bool

	binder binder

	minSize fyne.Size // cached for hover/top pos calcs
}

func NewToggle(changed func(bool)) *Toggle {
	t := &Toggle{
		OnChanged: changed,
	}
	t.ExtendBaseWidget(t)
	return t
}

func NewToggleWithData(data binding.Bool) *Toggle {
	toggle := NewToggle(nil)
	toggle.Bind(data)

	return toggle
}

func (t *Toggle) Bind(data binding.Bool) {
	t.binder.SetCallback(t.updateFromData)
	t.binder.Bind(data)

	t.OnChanged = func(_ bool) {
		t.binder.CallWithData(t.writeData)
	}
}

func (t *Toggle) SetToggled(toggled bool) {
	t.togglePLock.Lock()
	if toggled == t.Toggled {
		t.togglePLock.Unlock()
		return
	}

	t.Toggled = toggled
	onChanged := t.OnChanged
	t.togglePLock.Unlock()

	if onChanged != nil {
		onChanged(toggled)
	}

	t.Refresh()
}

func (t *Toggle) Hide() {
	if t.focused {
		t.FocusLost()
		if c := fyne.CurrentApp().Driver().CanvasForObject(t); c != nil {
			c.Focus(nil)
		}
	}
	t.BaseWidget.Hide()
}

func (t *Toggle) MouseIn(me *desktop.MouseEvent) {
	t.MouseMoved(me)
}

func (t *Toggle) MouseOut() {
	if t.hovered {
		t.hovered = false
		t.Refresh()
	}
}

func (t *Toggle) MouseMoved(me *desktop.MouseEvent) {
	if t.Disabled() {
		return
	}
	oldhovered := t.hovered

	t.hovered = t.minSize.IsZero() ||
		(me.Position.X <= t.minSize.Width && me.Position.Y <= t.minSize.Height)

	if oldhovered != t.hovered {
		t.Refresh()
	}
}

func (t *Toggle) Tapped(pe *fyne.PointEvent) {
	if t.Disabled() {
		return
	}
	if !t.minSize.IsZero() &&
		(pe.Position.X > t.minSize.Width || pe.Position.Y > t.minSize.Height) {
		// tapped outside
		return
	}

	if !t.focused {
		if !fyne.CurrentDevice().IsMobile() {
			if c := fyne.CurrentApp().Driver().CanvasForObject(t); c != nil {
				c.Focus(t)
			}
		}
	}
	t.SetToggled(!t.Toggled)
}

func (t *Toggle) MinSize() fyne.Size {
	t.ExtendBaseWidget(t)
	t.minSize = t.BaseWidget.MinSize()
	return t.minSize
}

func (t *Toggle) CreateRenderer() fyne.WidgetRenderer {
	th := t.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	t.ExtendBaseWidget(t)

	var bgColor fyne.ThemeColorName
	if t.Toggled {
		bgColor = theme.ColorNamePrimary
	} else {
		bgColor = theme.ColorNameInputBackground
	}
	bg := canvas.NewRectangle(th.Color(bgColor, v))
	bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	bg.CornerRadius = th.Size(theme.SizeNameInlineIcon) / 2
	bg.StrokeWidth = th.Size(theme.SizeNameInputBorder)

	indicator := canvas.NewCircle(th.Color(theme.ColorNameForegroundOnPrimary, v))
	indicator.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	indicator.StrokeWidth = th.Size(theme.SizeNameInputBorder)

	t.togglePLock.RLock()
	defer t.togglePLock.RUnlock()

	focusIndicator := canvas.NewCircle(th.Color(theme.ColorNameBackground, v))

	r := &toggleRenderer{
		bg:             bg,
		indicator:      indicator,
		focusIndicator: focusIndicator,
		toggle:         t,
	}

	r.applyTheme(th, v)
	r.updateToggle(th, v)
	r.updateFocusIndicator(th, v)
	return r
}

func (t *Toggle) FocusGained() {
	if t.Disabled() {
		return
	}

	t.focused = true

	t.Refresh()
}

func (t *Toggle) FocusLost() {
	t.focused = false
	t.Refresh()
}

func (t *Toggle) TypedRune(r rune) {
	if t.Disabled() {
		return
	}
	if r == ' ' {
		t.SetToggled(!t.Toggled)
	}
}

func (t *Toggle) TypedKey(key *fyne.KeyEvent) {}

func (t *Toggle) Unbind() {
	t.OnChanged = nil
	t.binder.Unbind()
}

func (t *Toggle) updateFromData(data binding.DataItem) {
	if data == nil {
		return
	}
	boolSource, ok := data.(binding.Bool)
	if !ok {
		return
	}
	val, err := boolSource.Get()
	if err != nil {
		fyne.LogError("Error getting current data value", err)
		return
	}
	t.SetToggled(val)
}

func (t *Toggle) writeData(data binding.DataItem) {
	if data == nil {
		return
	}
	boolTarget, ok := data.(binding.Bool)
	if !ok {
		return
	}
	currentValue, err := boolTarget.Get()
	if err != nil {
		return
	}
	if currentValue != t.Toggled {
		err := boolTarget.Set(t.Toggled)
		if err != nil {
			fyne.LogError(fmt.Sprintf("Failed to set binding value to %t", t.Toggled), err)
		}
	}
}

type toggleRenderer struct {
	bg             *canvas.Rectangle
	indicator      *canvas.Circle
	focusIndicator *canvas.Circle
	toggle         *Toggle

	indicatorOffPos      fyne.Position
	indicatorOnPos       fyne.Position
	focusIndicatorOffPos fyne.Position
	focusIndicatorOnPos  fyne.Position
}

func (r *toggleRenderer) Destroy() {}
func (r *toggleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.bg,
		r.focusIndicator,
		r.indicator,
	}
}

func (t *toggleRenderer) MinSize() fyne.Size {
	th := t.toggle.Theme()

	pad4 := th.Size(theme.SizeNameInnerPadding)
	iconInline := th.Size(theme.SizeNameInlineIcon)
	borderSize := th.Size(theme.SizeNameInputBorder)
	min := fyne.NewSize(
		(iconInline*2)+pad4*2+borderSize,
		iconInline+pad4+borderSize,
	)

	return min
}

func (t *toggleRenderer) Layout(size fyne.Size) {
	th := t.toggle.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	borderSize := th.Size(theme.SizeNameInputBorder)
	iconInlineSize := th.Size(theme.SizeNameInlineIcon)

	t.indicatorOffPos = fyne.NewPos(
		innerPadding/2+borderSize,
		(size.Height-iconInlineSize-borderSize-innerPadding/2)/2,
	)
	indicatorSize := fyne.NewSquareSize(iconInlineSize + innerPadding/2)

	focusIndicatorSize := fyne.NewSquareSize(iconInlineSize + innerPadding)
	t.focusIndicatorOffPos = fyne.NewPos(
		innerPadding/4+borderSize,
		(size.Height-focusIndicatorSize.Height)/2,
	)
	t.indicatorOnPos = t.indicatorOffPos.AddXY(iconInlineSize+innerPadding, 0)
	t.focusIndicatorOnPos = t.focusIndicatorOffPos.AddXY(iconInlineSize+innerPadding, 0)

	t.toggle.togglePLock.RLock()
	toggled := t.toggle.Toggled
	t.toggle.togglePLock.RUnlock()
	t.focusIndicator.Resize(focusIndicatorSize)
	if toggled {
		t.indicator.Move(t.indicatorOnPos)
		t.focusIndicator.Move(t.focusIndicatorOnPos)
	} else {
		t.indicator.Move(t.indicatorOffPos)
		t.focusIndicator.Move(t.focusIndicatorOffPos)
	}

	bgPos := fyne.NewPos(
		innerPadding/2,
		(size.Height-iconInlineSize)/2,
	)
	bgSize := fyne.NewSize(iconInlineSize*2+innerPadding, iconInlineSize)
	t.bg.Resize(bgSize)
	t.bg.Move(bgPos)
	t.indicator.Resize(indicatorSize)
}

func (t *toggleRenderer) applyTheme(th fyne.Theme, v fyne.ThemeVariant) {
	if t.toggle.Disabled() {
		t.indicator.FillColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		t.indicator.FillColor = th.Color(theme.ColorNameForegroundOnPrimary, v)
	}

	t.indicator.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	t.indicator.StrokeWidth = th.Size(theme.SizeNameInputBorder)

	t.bg.CornerRadius = th.Size(theme.SizeNameInlineIcon) / 2
	t.bg.StrokeWidth = th.Size(theme.SizeNameInputBorder)
}

func (t *toggleRenderer) Refresh() {
	th := t.toggle.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	t.toggle.togglePLock.RLock()
	t.applyTheme(th, v)
	t.updateFocusIndicator(th, v)
	t.updateToggle(th, v)
	t.toggle.togglePLock.RUnlock()
}

func (t *toggleRenderer) updateFocusIndicator(th fyne.Theme, v fyne.ThemeVariant) {
	if t.toggle.Disabled() {
		t.focusIndicator.FillColor = color.Transparent
	} else if t.toggle.focused {
		t.focusIndicator.FillColor = th.Color(theme.ColorNameFocus, v)
	} else if t.toggle.hovered {
		t.focusIndicator.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		t.focusIndicator.FillColor = color.Transparent
	}

	if t.toggle.Toggled {
		t.focusIndicator.Move(t.focusIndicatorOnPos)
	} else {
		t.focusIndicator.Move(t.focusIndicatorOffPos)
	}
}

func (t *toggleRenderer) updateToggle(th fyne.Theme, v fyne.ThemeVariant) {
	if t.toggle.Toggled {
		t.indicator.Move(t.indicatorOnPos)
		t.bg.FillColor = th.Color(theme.ColorNamePrimary, v)
	} else {
		t.indicator.Move(t.indicatorOffPos)
		t.bg.FillColor = th.Color(theme.ColorNameInputBackground, v)
	}
}
