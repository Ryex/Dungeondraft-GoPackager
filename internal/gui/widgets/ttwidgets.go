package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

type ToolTipEntry struct {
	widget.Entry
	ttwidget.ToolTipWidgetExtend
}

func NewToolTipEntryWithData(data binding.String) *ToolTipEntry {
	ttentry := &ToolTipEntry{
		Entry: widget.Entry{},
	}
	ttentry.ExtendBaseWidget(ttentry)
	ttentry.Bind(data)
	return ttentry
}

func (ttentry *ToolTipEntry) ExtendBaseWidget(wid fyne.Widget) {
	ttentry.ExtendToolTipWidget(wid)
	ttentry.Entry.ExtendBaseWidget(wid)
}

type ToolTipButton struct {
	widget.Button
	ttwidget.ToolTipWidgetExtend
}

func NewToolTipButton(label string, tapped func()) *ToolTipButton {
	ttbutton := &ToolTipButton{
		Button: widget.Button{
			Text:     label,
			OnTapped: tapped,
		},
	}
	ttbutton.ExtendBaseWidget(ttbutton)
	return ttbutton
}

func NewToolTipButtonWithIcon(label string, icon fyne.Resource, tapped func()) *ToolTipButton {
	ttbutton := &ToolTipButton{
		Button: widget.Button{
			Text:     label,
			Icon:     icon,
			OnTapped: tapped,
		},
	}
	ttbutton.ExtendBaseWidget(ttbutton)
	return ttbutton
}

func (ttbutton *ToolTipButton) ExtendBaseWidget(wid fyne.Widget) {
	ttbutton.ExtendToolTipWidget(wid)
	ttbutton.Button.ExtendBaseWidget(wid)
}

func (ttbutton *ToolTipButton) MouseIn(e *desktop.MouseEvent) {
	ttbutton.ToolTipWidgetExtend.MouseIn(e)
	ttbutton.Button.MouseIn(e)
}

func (ttbutton *ToolTipButton) MouseOut() {
	ttbutton.ToolTipWidgetExtend.MouseOut()
	ttbutton.Button.MouseOut()
}

// MouseMoved is called when a desktop pointer hovers over the widget
func (ttbutton *ToolTipButton) MouseMoved(e *desktop.MouseEvent) {
	ttbutton.ToolTipWidgetExtend.MouseMoved(e)
	ttbutton.Button.MouseMoved(e)
}
