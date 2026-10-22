// The about dialog below is a custom version of the about dialog of
// fyne.io/x/fyne (dialog/about.go), which is released under the BSD 3-Clause
// license:
//
// Copyright (C) 2018 Fyne.io developers (see AUTHORS)
// https://github.com/fyne-io/fyne/blob/master/AUTHORS
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//   - Redistributions of source code must retain the above copyright
//     notice, this list of conditions and the following disclaimer.
//   - Redistributions in binary form must reproduce the above copyright
//     notice, this list of conditions and the following disclaimer in the
//     documentation and/or other materials provided with the distribution.
//   - Neither the name of Fyne.io nor the names of its contributors may be
//     used to endorse or promote products derived from this software without
//     specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/assets"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/version"
)

// showAbout opens the about dialog of the application.
//
// The dialog of fyne.io/x/fyne builds its version line from the app metadata
// alone. This one amends it with the commit the binary was built from, which
// the metadata does not carry (see internal/version).
func (a *App) showAbout(links []*widget.Hyperlink) {
	dlg := dialog.NewCustom(
		lang.X("about.title", "About"),
		lang.X("about.close", "Close"),
		newAboutContent(a.app, links),
		a.window,
	)
	dlg.Resize(
		fyne.NewSize(
			fyne.Min(a.window.Canvas().Size().Width, 640),
			fyne.Min(a.window.Canvas().Size().Height, 480),
		),
	)
	dlg.Show()
}

// newAboutContent builds the content of the about dialog: the icon, the name
// and the version of the app, a description of it and the links that are
// given.
func newAboutContent(a fyne.App, links []*widget.Hyperlink) fyne.CanvasObject {
	meta := a.Metadata()

	icon := meta.Icon
	if icon == nil || len(icon.Content()) == 0 {
		icon = assets.Icon
	}

	logo := canvas.NewImageFromResource(icon)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 128))

	name := meta.Name
	if name == "" {
		name = lang.X("greeting.title", "Dungeondraft-GoPackager")
	}

	// the app data is built from segments instead of markdown, so that a
	// translated label cannot break the markup
	appData := widget.NewRichText(
		&widget.TextSegment{
			Text:  name,
			Style: widget.RichTextStyleSubHeading,
		},
		// a sub heading is an inline style, so it only ends the line when
		// an empty, non inline segment follows it - this is the segment
		// the markdown renderer of Fyne inserts after a heading
		&widget.TextSegment{Style: widget.RichTextStyleParagraph},
		&widget.TextSegment{
			Text:  lang.X("about.version", "Version") + ": " + version.Display(meta.Version),
			Style: widget.RichTextStyleParagraph,
		},
	)
	centerText(appData)

	space := canvas.NewRectangle(color.Transparent)
	space.SetMinSize(fyne.NewSquareSize(theme.Padding() * 4))

	description := widget.NewRichText(&widget.TextSegment{
		Text: lang.X(
			"greeting.sub-title",
			"Package, edit, and prepare Dungeondraft resource packs",
		),
		Style: widget.RichTextStyleParagraph,
	})
	centerText(description)

	body := container.NewVBox(
		space,
		logo,
		appData,
		container.NewCenter(description),
	)

	bg := newAlphaRect(theme.ColorNameBackground, 0xe0)
	footerBG := newAlphaRect(theme.ColorNameBackground, 0x33)

	// the icon scrolls with the content, at a third of its speed, which
	// gives the dialog its parallax look
	underlay := canvas.NewImageFromResource(icon)
	underlay.Resize(fyne.NewSize(512, 512))
	underlayer := &underLayout{}
	slideBG := container.New(underlayer, underlay)

	scroll := container.NewScroll(body)
	scroll.OnScrolled = func(p fyne.Position) {
		underlayer.offset = -p.Y / 3
		underlayer.Layout(slideBG.Objects, slideBG.Size())
	}

	bgClip := container.NewScroll(slideBG)
	bgClip.Direction = container.ScrollNone

	return container.NewStack(
		container.New(unpad{top: true}, bgClip, bg),
		container.NewBorder(
			nil,
			container.NewStack(footerBG, aboutFooter(links)),
			nil, nil,
			container.New(unpad{top: true, bottom: true}, scroll),
		),
	)
}

// aboutFooter lays the links of the dialog out in a row, separated by dashes.
func aboutFooter(links []*widget.Hyperlink) fyne.CanvasObject {
	footer := container.NewHBox(layout.NewSpacer())
	for i, link := range links {
		footer.Add(link)
		if i < len(links)-1 {
			footer.Add(widget.NewLabel("-"))
		}
	}
	footer.Add(layout.NewSpacer())

	return footer
}

// centerText centers the text of every segment of a rich text.
func centerText(rich *widget.RichText) {
	for _, s := range rich.Segments {
		if text, ok := s.(*widget.TextSegment); ok {
			text.Style.Alignment = fyne.TextAlignCenter
		}
	}
}

// alphaRect is a rectangle that is filled with a theme color at the given
// opacity.
//
// Fyne refreshes the content of a window when the theme changes, so the
// rectangle follows the theme without listening to the settings. A settings
// listener cannot be removed again and would outlive the dialog, which is
// created anew every time it is shown.
type alphaRect struct {
	widget.BaseWidget

	rect  *canvas.Rectangle
	name  fyne.ThemeColorName
	alpha uint8
}

// newAlphaRect returns a rectangle that is filled with the given theme color
// at the given opacity.
func newAlphaRect(name fyne.ThemeColorName, alpha uint8) *alphaRect {
	r := &alphaRect{
		rect:  canvas.NewRectangle(withAlpha(theme.Color(name), alpha)),
		name:  name,
		alpha: alpha,
	}
	r.ExtendBaseWidget(r)

	return r
}

// Refresh causes the rectangle to be redrawn in the color of the current
// theme.
func (r *alphaRect) Refresh() {
	r.rect.FillColor = withAlpha(theme.Color(r.name), r.alpha)
	r.BaseWidget.Refresh()
}

// CreateRenderer returns the renderer of the rectangle.
func (r *alphaRect) CreateRenderer() fyne.WidgetRenderer {
	return &alphaRectRenderer{rect: r}
}

type alphaRectRenderer struct {
	rect *alphaRect
}

func (r *alphaRectRenderer) Layout(size fyne.Size) {
	r.rect.rect.Resize(size)
}

func (r *alphaRectRenderer) MinSize() fyne.Size {
	return r.rect.rect.MinSize()
}

func (r *alphaRectRenderer) Refresh() {
	canvas.Refresh(r.rect.rect)
}

func (r *alphaRectRenderer) BackgroundColor() color.Color {
	return theme.Color(theme.ColorNameBackground)
}

func (r *alphaRectRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect.rect}
}

func (r *alphaRectRenderer) Destroy() {}

// withAlpha returns the color c with its opacity replaced by alpha.
func withAlpha(c color.Color, alpha uint8) color.Color {
	r, g, b, _ := c.RGBA()

	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: alpha}
}

// underLayout places the background icon in the middle of the dialog at the
// vertical offset of the parallax.
type underLayout struct {
	offset float32
}

func (u *underLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	under := objs[0]
	left := size.Width/2 - under.Size().Width/2
	under.Move(fyne.NewPos(left, u.offset-50))
}

func (u *underLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.Size{}
}

// unpad removes the padding of the theme from its content on the sides that
// are set.
type unpad struct {
	top, bottom bool
}

func (u unpad) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	pad := theme.Padding()

	var pos fyne.Position
	size := s
	if u.top {
		pos.Y = -pad
		size.Height += pad
	}
	if u.bottom {
		size.Height += pad
	}

	for _, o := range objs {
		o.Move(pos)
		o.Resize(size)
	}
}

func (u unpad) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(100, 100)
}
