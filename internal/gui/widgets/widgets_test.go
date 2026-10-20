package widgets

import (
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
)

func TestMain(m *testing.M) {
	lang.SetupLang("en")
	testutil.SetupFyneApp()
	os.Exit(m.Run())
}

func TestToggleSetToggledNotifies(t *testing.T) {
	toggle := NewToggle(nil)

	var changed []bool
	toggle.OnChanged = func(b bool) {
		changed = append(changed, b)
	}

	toggle.SetToggled(true)
	assert.True(t, toggle.Toggled)
	assert.Equal(t, []bool{true}, changed)

	// setting the same value again is a no-op
	toggle.SetToggled(true)
	assert.Equal(t, []bool{true}, changed)

	toggle.SetToggled(false)
	assert.False(t, toggle.Toggled)
	assert.Equal(t, []bool{true, false}, changed)
}

func TestToggleReflectsBoundValue(t *testing.T) {
	data := binding.NewBool()
	require.NoError(t, data.Set(true))

	toggle := NewToggleWithData(data)
	assert.True(t, toggle.Toggled, "the toggle must pick up the current bound value")

	require.NoError(t, data.Set(false))
	assert.False(t, toggle.Toggled, "the toggle must follow later changes of the bound value")
}

func TestToggleWritesBackToBinding(t *testing.T) {
	data := binding.NewBool()

	toggle := NewToggleWithData(data)
	toggle.SetToggled(true)

	got, err := data.Get()
	require.NoError(t, err)
	assert.True(t, got, "toggling must write the new value back to the binding")
}

func TestNumericalEntryRejectsNonNumericInput(t *testing.T) {
	entry := NewNumericalEntry()

	entry.TypedRune('1')
	entry.TypedRune('a')
	entry.TypedRune('2')

	assert.Equal(t, "12", entry.Text)
	assert.Equal(t, 12, entry.AsInt())
}

func TestNumericalEntryAllowFloat(t *testing.T) {
	entry := NewNumericalEntry()
	entry.AllowFloat = true

	entry.TypedRune('1')
	entry.TypedRune('.')
	entry.TypedRune('5')

	assert.Equal(t, "1.5", entry.Text)
	assert.Equal(t, 1.5, entry.AsFloat())
}

func TestNumericalEntryIsNumber(t *testing.T) {
	entry := NewNumericalEntry()

	assert.True(t, entry.isNumber("123"))
	assert.False(t, entry.isNumber("12a"))
	assert.False(t, entry.isNumber("1.5"), "floats are rejected unless AllowFloat is set")

	entry.AllowFloat = true
	assert.True(t, entry.isNumber("1.5"))
}

func TestSpinnerClampsToMinAndMax(t *testing.T) {
	spin := NewIntSpinner(0, 10, 1)

	spin.SetValue(5)
	assert.Equal(t, 5.0, spin.GetValue())

	spin.SetValue(100)
	assert.Equal(t, 10.0, spin.GetValue(), "values above the maximum must be clamped")

	spin.SetValue(-100)
	assert.Equal(t, 0.0, spin.GetValue(), "values below the minimum must be clamped")
}

func TestSpinnerStepsWithButtons(t *testing.T) {
	spin := NewIntSpinner(0, 10, 1)
	spin.SetValue(5)

	test.Tap(spin.buttonUp)
	assert.Equal(t, 6.0, spin.GetValue())

	test.Tap(spin.buttonDown)
	assert.Equal(t, 5.0, spin.GetValue())
}

func TestSpinnerReportsUserChanges(t *testing.T) {
	spin := NewIntSpinner(0, 10, 1)
	spin.SetValue(4)

	var changed []float64
	spin.OnChanged = func(v float64) {
		changed = append(changed, v)
	}

	test.Tap(spin.buttonUp)
	assert.Equal(t, []float64{5}, changed)

	test.Tap(spin.buttonDown)
	assert.Equal(t, []float64{5, 4}, changed)
}

func TestSpinnerSetValueWritesToBinding(t *testing.T) {
	data := binding.NewFloat()

	spin := NewSpinner(0, 10, 1)
	spin.Bind(data)

	spin.SetValue(6)

	got, err := data.Get()
	require.NoError(t, err)
	assert.Equal(t, 6.0, got, "setting the value must write it back to the binding")
}

func TestSpinnerFollowsBoundValue(t *testing.T) {
	data := binding.NewFloat()
	require.NoError(t, data.Set(3))

	spin := NewSpinner(0, 10, 1)
	spin.Bind(data)

	assert.Equal(t, 3.0, spin.GetValue())

	require.NoError(t, data.Set(7))
	assert.Equal(t, 7.0, spin.GetValue(), "the spinner must follow the bound value")
}

func TestMajorMinorVersionSpinner(t *testing.T) {
	spin := NewMajorMinorVersionSpinner(1, 3, 5, 1)

	spin.SetValue("2.3")
	assert.Equal(t, "2.3", spin.GetValue())

	// a minor version above the maximum rolls into the next major version,
	// which is then clamped back to the maximum major version
	spin.SetValue("3.6")
	assert.Equal(t, "3", spin.GetValue())
}

func TestToolTipButtonCallsOnTapped(t *testing.T) {
	tapped := false
	button := NewToolTipButton("Press", func() {
		tapped = true
	})

	test.Tap(button)
	assert.True(t, tapped)
}

func TestEntryWCopyBtnCopiesToClipboard(t *testing.T) {
	entry := NewEntryWCopyBtn()
	entry.SetText("res://packs/testpack")

	test.Tap(entry.copyButton)

	content := fyne.CurrentApp().Clipboard().Content()
	assert.Equal(t, "res://packs/testpack", content)
}
