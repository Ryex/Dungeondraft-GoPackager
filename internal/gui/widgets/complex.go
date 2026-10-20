package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
)

type EntryWCopyBtn struct {
	fyne.Container

	OnChanged func(string)

	entry      *widget.Entry
	label      *widget.Label
	copyButton *ToolTipButton
}

func NewEntryWCopyBtn() *EntryWCopyBtn {
	entry := widget.NewEntry()
	copyButton := NewToolTipButtonWithIcon("", theme.Icon(theme.IconNameContentCopy), func() {
		fyne.CurrentApp().Clipboard().SetContent(entry.Text)
	})
	e := &EntryWCopyBtn{
		entry:      entry,
		copyButton: copyButton,
	}
	copyButton.SetToolTip(lang.X("btn.copy.tooltip", "Copy to Clipboard"))

	entry.OnChanged = func(s string) {
		if e.OnChanged != nil {
			e.OnChanged(s)
		}
	}

	e.Layout = layout.NewBorderLayout(nil, nil, nil, copyButton)
	e.Add(copyButton)
	e.Add(entry)
	return e
}

func (entry *EntryWCopyBtn) Refresh() {
	entry.copyButton.SetToolTip(lang.X("btn.copy.tooltip", "Copy to Clipboard"))
	entry.Container.Refresh()
}

func (entry *EntryWCopyBtn) Disable() {
	entry.entry.Disable()
}

func (entry *EntryWCopyBtn) SetText(text string) {
	entry.entry.SetText(text)
}

func (entry *EntryWCopyBtn) Enable() {
	entry.entry.Enable()
}

func (entry *EntryWCopyBtn) CreateRenderer() fyne.WidgetRenderer {
	return &translatedEntryWCopyBtnRender{entry}
}

type translatedEntryWCopyBtnRender struct {
	entry *EntryWCopyBtn
}

func (e *translatedEntryWCopyBtnRender) Layout(size fyne.Size) {
	e.entry.Layout.Layout(e.entry.Objects, size)
}

func (e *translatedEntryWCopyBtnRender) MinSize() fyne.Size {
	return e.entry.MinSize()
}

func (e *translatedEntryWCopyBtnRender) Refresh() {
	e.entry.Refresh()
}

func (e *translatedEntryWCopyBtnRender) BackgroundColor() color.Color {
	return theme.Color(theme.ColorNameBackground)
}

func (e *translatedEntryWCopyBtnRender) Objects() []fyne.CanvasObject {
	return e.entry.Objects
}

func (e *translatedEntryWCopyBtnRender) Destroy() {}
