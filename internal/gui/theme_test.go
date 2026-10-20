package gui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/stretchr/testify/assert"
)

func TestBetterDefaultThemeDefinesAllColors(t *testing.T) {
	test.AssertAllColorNamesDefined(t, BetterDefaultTheme(), "betterDefault")
}

func TestBetterDefaultThemeKeepsTheDefaultColors(t *testing.T) {
	th := BetterDefaultTheme()
	defaultTheme := theme.DefaultTheme()

	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		// the custom theme only overrides the disabled and separator colors
		assert.Equal(t,
			defaultTheme.Color(theme.ColorNamePrimary, variant),
			th.Color(theme.ColorNamePrimary, variant),
			"the primary color must not be changed",
		)
	}
}

func TestForcedVariantThemeForcesTheVariant(t *testing.T) {
	base := theme.DefaultTheme()
	forced := &forcedVariantTheme{Theme: base, variant: theme.VariantDark}

	// the forced variant is used no matter which variant is asked for
	assert.Equal(t,
		base.Color(theme.ColorNameBackground, theme.VariantDark),
		forced.Color(theme.ColorNameBackground, theme.VariantLight),
	)
}
