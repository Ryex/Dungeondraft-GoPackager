package widgets

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
)

type NumericalEntry struct {
	widget.Entry
	AllowFloat bool
}

func NewNumericalEntry() *NumericalEntry {
	entry := &NumericalEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *NumericalEntry) TypedRune(r rune) {
	if r >= '0' && r <= '9' {
		e.Entry.TypedRune(r)
		return
	}
	if e.AllowFloat && (r == '.') {
		e.Entry.TypedRune(r)
	}
}

func (e *NumericalEntry) isNumber(text string) bool {
	if e.AllowFloat {
		_, err := strconv.ParseFloat(text, 64)
		return err == nil
	}
	_, err := strconv.Atoi(text)
	return err == nil
}

func (e *NumericalEntry) AsFloat() float64 {
	val, _ := strconv.ParseFloat(e.Text, 64)
	return val
}

func (e *NumericalEntry) AsInt() int {
	if e.AllowFloat {
		return int(e.AsFloat())
	}
	val, _ := strconv.Atoi(e.Text)
	return val
}

type extendedEntry struct {
	NumericalEntry
	onEnter     func()
	onScrolled  func(s *fyne.ScrollEvent)
	onFocusLost func()
}

func newExtendedEntry() *extendedEntry {
	ext := &extendedEntry{}
	ext.ExtendBaseWidget(ext)
	return ext
}

func (e *extendedEntry) KeyDown(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyReturn {
		if e.onEnter != nil {
			e.onEnter()
		}
	} else {
		e.NumericalEntry.KeyDown(key)
	}
}

func (e *extendedEntry) Scrolled(s *fyne.ScrollEvent) {
	if e.onScrolled != nil {
		e.onScrolled(s)
	}
}

func (e *extendedEntry) FocusLost() {
	e.NumericalEntry.FocusLost()
	if e.onFocusLost != nil {
		e.onFocusLost()
	}
}

type Spinner struct {
	fyne.Container
	value float64
	Min   float64
	Max   float64
	Step  float64
	start float64

	Precision uint

	buttonUp   *widget.Button
	buttonDown *widget.Button
	entry      *extendedEntry
	integer    bool

	OnChanged func(float64)

	binder binder
}

type MajorMinorVersionSpinner struct {
	Spinner
	value    string
	major    int
	minor    int
	Min      int
	MaxMajor int
	MaxMinor int
	Step     int

	OnChanged func(string)
}

func NewSpinner(min, max, step float64) *Spinner {
	return newSpinner(min, max, step, false)
}

func NewIntSpinner(min, max, step int) *Spinner {
	return newSpinner(float64(min), float64(max), float64(step), true)
}

func newSpinner(minVal, maxVal, step float64, integer bool) *Spinner {
	buttonUp := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {})
	buttonDown := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {})

	updown := container.NewHBox(buttonUp, buttonDown)
	entry := newExtendedEntry()

	spin := &Spinner{
		buttonUp:   buttonUp,
		buttonDown: buttonDown,
		entry:      entry,
		Min:        minVal,
		Max:        maxVal,
		value:      min(max(minVal, 1), maxVal),
		start:      min(max(minVal, 1), maxVal),
		Step:       step,
		integer:    integer,
	}
	buttonDown.OnTapped = spin.onDown
	buttonUp.OnTapped = spin.onUp

	entry.AllowFloat = !integer

	entry.onEnter = spin.onEnter
	entry.OnChanged = func(s string) {
		if strings.Contains(s, ".") {
			_, deci := utils.SplitOne(s, ".")
			if deci == "" {
				return
			}
		}
		spin.onEnter()
	}
	entry.onFocusLost = spin.onEnter
	entry.onScrolled = spin.onScrolled

	spin.Layout = layout.NewBorderLayout(nil, nil, nil, updown)
	spin.Add(updown)
	spin.Add(entry)
	spin.updateVal(spin.value, true)
	return spin
}

func NewMajorMinorVersionSpinner(minVal, maxMajor, maxMinor, step int) *MajorMinorVersionSpinner {
	buttonUp := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {})
	buttonDown := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {})

	updown := container.NewHBox(buttonUp, buttonDown)
	entry := newExtendedEntry()

	spin := &MajorMinorVersionSpinner{
		Spinner: Spinner{
			buttonUp:   buttonUp,
			buttonDown: buttonDown,
			entry:      entry,
		},
		Min:      minVal,
		MaxMajor: maxMajor,
		MaxMinor: maxMinor,
		major:    min(max(minVal, 1), maxMajor),
		minor:    0,
		value:    strconv.Itoa(min(max(minVal, 1), maxMajor)),
		Step:     step,
	}
	buttonDown.OnTapped = spin.onDown
	buttonUp.OnTapped = spin.onUp

	entry.AllowFloat = true

	entry.onEnter = spin.onEnter
	entry.onFocusLost = spin.onEnter
	entry.onScrolled = spin.onScrolled
	entry.OnChanged = func(s string) {
		if strings.Contains(s, ".") {
			_, minor := utils.SplitOne(s, ".")
			if minor == "" {
				return
			}
		}
		spin.onEnter()
	}

	spin.Layout = layout.NewBorderLayout(nil, nil, nil, updown)
	spin.Add(updown)
	spin.Add(entry)
	spin.updateVal(spin.major, spin.minor, true)
	return spin
}

func (s *Spinner) GetValue() float64 {
	s.updateVal(s.value, true)
	return s.value
}

func (s *Spinner) SetValue(value float64) {
	s.updateVal(value, false)
}

func (s *MajorMinorVersionSpinner) GetValue() string {
	s.updateVal(s.major, s.minor, true)
	return s.value
}

func (s *MajorMinorVersionSpinner) SetValue(value string) {
	majorS, minorS := utils.SplitOne(utils.TruncateToNumericString(s.entry.Text), ".")

	var major, minor int
	var err error
	major, err = strconv.Atoi(majorS)
	if err != nil {
		major = s.Min
	}
	minor, err = strconv.Atoi(minorS)
	if err != nil {
		minor = 0
	}
	s.updateVal(major, minor, false)
}

func (s *Spinner) onEnter() {
	var val float64
	if f, err := strconv.ParseFloat(s.entry.Text, 64); err != nil {
		val = s.start
	} else {
		val = min(max(s.Min, f), s.Max)
	}
	changed := s.updateVal(val, false)
	if changed && s.OnChanged != nil {
		s.OnChanged(val)
	}
}

func (s *MajorMinorVersionSpinner) onEnter() {
	var val string
	majorS, minorS := utils.SplitOne(utils.TruncateToNumericString(s.entry.Text), ".")

	var major, minor int
	var err error
	major, err = strconv.Atoi(majorS)
	if err != nil {
		major = s.Min
	}
	minor, err = strconv.Atoi(minorS)
	if err != nil {
		minor = 0
	}
	changed := s.updateVal(major, minor, false)
	if changed && s.OnChanged != nil {
		s.OnChanged(val)
	}
}

func sn(n float64) float64 {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return +1
	}
	return 0
}

func (s *Spinner) onScrolled(e *fyne.ScrollEvent) {
	if e.Scrolled.DY != 0 {

		val := s.value + s.Step*sn(float64(e.Scrolled.DY))
		changed := s.updateVal(val, false)
		if changed && s.OnChanged != nil {
			s.OnChanged(val)
		}
	}
}

func (s *MajorMinorVersionSpinner) onScrolled(e *fyne.ScrollEvent) {
	if e.Scrolled.DY != 0 {
		minor := s.minor + s.Step*int(sn(float64(e.Scrolled.DY)))
		changed := s.updateVal(s.major, minor, false)
		if changed && s.OnChanged != nil {
			s.OnChanged(s.value)
		}
	}
}

func (s *Spinner) onUp() {
	val := s.value + s.Step
	changed := s.updateVal(val, false)
	if changed && s.OnChanged != nil {
		s.OnChanged(val)
	}
}

func (s *Spinner) onDown() {
	val := s.value - s.Step
	changed := s.updateVal(val, false)
	if changed && s.OnChanged != nil {
		s.OnChanged(val)
	}
}

func (s *MajorMinorVersionSpinner) onUp() {
	val := s.minor + s.Step
	changed := s.updateVal(s.major, val, false)
	if changed && s.OnChanged != nil {
		s.OnChanged(s.value)
	}
}

func (s *MajorMinorVersionSpinner) onDown() {
	val := s.minor - s.Step
	changed := s.updateVal(s.major, val, false)
	if changed && s.OnChanged != nil {
		s.OnChanged(s.value)
	}
}

func (s *Spinner) updateVal(val float64, fromBinding bool) bool {
	val = min(max(s.Min, val), s.Max)
	if s.Precision > 0 {
		val = utils.ToFixed(val, int(s.Precision))
	}
	changed := val != s.value
	s.value = val
	if s.integer {
		s.value = math.Round(s.value)
		s.entry.SetText(fmt.Sprintf("%d", int(s.value)))
	} else {
		precision := -1
		if s.Precision > 0 {
			precision = int(s.Precision)
		}
		s.entry.SetText(strconv.FormatFloat(s.value, 'f', precision, 64))
	}
	if s.value <= s.Min {
		s.buttonDown.Disable()
	} else {
		s.buttonDown.Enable()
	}
	if s.value >= s.Max {
		s.buttonUp.Disable()
	} else {
		s.buttonUp.Enable()
	}
	if changed && !fromBinding {
		if s.binder.pair.listener != nil {
			s.binder.SetCallback(nil)
			s.binder.CallWithData(s.writeData)
			s.binder.SetCallback(s.updateFromData)
		}
	}
	return changed
}

func (s *MajorMinorVersionSpinner) updateVal(major, minor int, fromBinding bool) bool {
	s.major = major
	s.minor = minor
	if s.minor > s.MaxMinor {
		s.minor = 0
		s.major += 1
	}
	if s.major > s.MaxMajor {
		s.major = s.MaxMajor
	}
	if s.major < s.Min {
		s.major = s.Min
	}
	val := strconv.Itoa(s.major)
	if s.minor > 0 {
		val += fmt.Sprintf(".%d", s.minor)
	}
	changed := val != s.value
	s.value = val
	s.entry.SetText(s.value)

	if s.major <= s.Min && s.minor <= 0 {
		s.buttonDown.Disable()
	} else {
		s.buttonDown.Enable()
	}
	if s.major >= s.MaxMajor {
		s.buttonUp.Disable()
	} else {
		s.buttonUp.Enable()
	}
	if changed && !fromBinding {
		if s.binder.pair.listener != nil {
			s.binder.SetCallback(nil)
			s.binder.CallWithData(s.writeData)
			s.binder.SetCallback(s.updateFromData)
		}
	}

	return changed
}

func (s *Spinner) Disable() {
	s.entry.Disable()
	s.buttonUp.Disable()
	s.buttonDown.Disable()
}

func (s *Spinner) Enable() {
	s.entry.Enable()
	s.updateVal(s.value, true)
}

func (s *Spinner) CreateRenderer() fyne.WidgetRenderer {
	return &spinnerRenderer{
		spinner: s,
	}
}

func (s *Spinner) Bind(data binding.Float) {
	s.binder.SetCallback(s.updateFromData)
	s.binder.Bind(data)
}

func (s *MajorMinorVersionSpinner) Bind(data binding.String) {
	s.binder.SetCallback(s.updateFromData)
	s.binder.Bind(data)
}

func (s *Spinner) Unbind() {
	s.binder.Unbind()
}

func (s *Spinner) updateFromData(data binding.DataItem) {
	if data == nil {
		return
	}
	floatS, ok := data.(binding.Float)
	if !ok {
		return
	}
	val, err := floatS.Get()
	if err != nil {
		return
	}
	s.updateVal(val, true)
}

func (s *MajorMinorVersionSpinner) updateFromData(data binding.DataItem) {
	if data == nil {
		return
	}
	versionS, ok := data.(binding.String)
	if !ok {
		return
	}
	val, err := versionS.Get()
	majorS, minorS := utils.SplitOne(utils.TruncateToNumericString(val), ".")
	var major, minor int
	major, err = strconv.Atoi(majorS)
	if err != nil {
		major = s.Min
	}
	minor, err = strconv.Atoi(minorS)
	if err != nil {
		minor = 0
	}
	s.updateVal(major, minor, true)
}

func (s *Spinner) writeData(data binding.DataItem) {
	if data == nil {
		return
	}
	floatT, ok := data.(binding.Float)
	if !ok {
		return
	}
	curVal, err := floatT.Get()
	if err == nil && curVal == s.value {
		return
	}
	floatT.Set(s.value)
}

func (s *MajorMinorVersionSpinner) writeData(data binding.DataItem) {
	if data == nil {
		return
	}
	versionT, ok := data.(binding.String)
	if !ok {
		return
	}
	curVal, err := versionT.Get()
	if err == nil && curVal == s.value {
		return
	}
	versionT.Set(s.value)
}

type spinnerRenderer struct {
	spinner *Spinner
}

func (r *spinnerRenderer) Layout(size fyne.Size) {
	r.spinner.Layout.Layout(r.spinner.Objects, size)
}

func (r *spinnerRenderer) MinSize() fyne.Size {
	return r.spinner.MinSize()
}

func (r *spinnerRenderer) Refresh() {
	r.spinner.Refresh()
}

func (r *spinnerRenderer) BackgroundColor() color.Color {
	return theme.Color(theme.ColorNameBackground)
}

func (r *spinnerRenderer) Objects() []fyne.CanvasObject {
	return r.spinner.Objects
}

func (r *spinnerRenderer) Destroy() {}
