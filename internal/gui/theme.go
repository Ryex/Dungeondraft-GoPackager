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

var _ fyne.Theme = (*betterDefault)(nil)

type betterDefault struct {
	fyne.Theme
}

func BetterDefaultTheme() fyne.Theme {
	return &betterDefault{theme.DefaultTheme()}
}

func (b *betterDefault) Color(name fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	c := b.Theme.Color(name, v)
	switch name {
	case theme.ColorNameDisabled:
		{
			if v == theme.VariantDark {
				return ddimage.Lignten(c, 0.1)
			} else {
				return ddimage.Darken(c, 0.6)
			}
		}
	case theme.ColorNameSeparator:
		{
			if v == theme.VariantDark {
				return color.NRGBA{R: 0x39, G: 0x39, B: 0x3a, A: 0xff}
			}
		}
	}
	return c
}
