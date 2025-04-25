package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

type ToolTipEntry struct {
	widget.Entry
	ttwidget.ToolTipWidgetExtend
}

func NewToolTipEntryWithData(data binding.String) *ToolTipEntry {
	w := &ToolTipEntry{
		Entry: widget.Entry{},
	}
	w.ExtendBaseWidget(w)
	w.Bind(data)
	return w
}

func (w *ToolTipEntry) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.Entry.ExtendBaseWidget(wid)
}
