package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
)

var _ fyne.Theme = (*forcedVariantTheme)(nil)

type forcedVariantTheme struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (f *forcedVariantTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}


var _ fyne.Theme = (*betterDisabledContrast)(nil)


type betterDisabledContrast struct {
	fyne.Theme
}

func (b *betterDisabledContrast) Color(name fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	c := b.Theme.Color(name, v)
	if name == theme.ColorNameDisabled {
		if v == theme.VariantDark {
			return ddimage.Lignten(c, 0.1)
		} else {
			return ddimage.Darken(c, 0.6)
		}
	}
	return c
}
