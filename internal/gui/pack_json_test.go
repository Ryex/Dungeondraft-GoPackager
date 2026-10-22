package gui

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/sirupsen/logrus"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

// queuedApp is a Fyne app that holds the functions that are queued on the
// Fyne goroutine back, like the driver of the desktop app does.
//
// The test driver of Fyne runs them right away, which hides the fact that a
// widget only gets the value of its binding when the goroutine comes to it.
type queuedApp struct {
	fyne.App
	driver *queuedDriver
}

func (a *queuedApp) Driver() fyne.Driver {
	return a.driver
}

// queuedDriver collects the functions that are queued on the Fyne goroutine.
type queuedDriver struct {
	fyne.Driver
	queue []func()
}

func (d *queuedDriver) DoFromGoroutine(f func(), wait bool) {
	if wait {
		// the functions that are waited for are run right away, so that
		// nothing blocks
		d.Driver.DoFromGoroutine(f, wait)
		return
	}
	d.queue = append(d.queue, f)
}

// queuedAppWithDriver installs an app that holds the functions that are
// queued on the Fyne goroutine back, and returns it.
func queuedAppWithDriver(t *testing.T) *queuedApp {
	t.Helper()

	previous := fyne.CurrentApp()
	testApp := test.NewApp()
	queued := &queuedApp{
		App:    testApp,
		driver: &queuedDriver{Driver: testApp.Driver()},
	}
	fyne.SetCurrentApp(queued)
	t.Cleanup(func() {
		fyne.SetCurrentApp(previous)
	})

	return queued
}

// TestPackJSONDialogShowsTheValuesOfThePackage verifies that the dialog of
// the package settings shows the values of the package right away.
//
// A binding notifies its listeners on the Fyne goroutine, so the value of a
// bound widget is delivered asynchronously. The entries of the dialog are
// filled with the values of the package right away, so that they show them
// even when the goroutine has a lot of work to do (e.g. while the tags of a
// big package are generated).
func TestPackJSONDialogShowsTheValuesOfThePackage(t *testing.T) {
	queued := queuedAppWithDriver(t)

	dir := t.TempDir()
	packJSON := `{
  "name": "Test Pack",
  "id": "testpack",
  "version": "2.8",
  "author": "Test Author",
  "keywords": "one,two"
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pack.json"), []byte(packJSON), 0o644))

	pkg := ddpackage.NewPackage(log.WithField("path", dir))
	require.NoError(t, pkg.LoadUnpackedFromFolder(dir))

	dlg := NewPackJSONDialogPkg(pkg, true, nil, queued.NewWindow("test"))

	entries := collectObjects[*widget.Entry](dlg.content)
	require.Len(t, entries, 4, "the dialog must have an entry for the id, the name, the author and the keywords")
	assert.Equal(t, "testpack", entries[0].Text, "the id must be shown")
	assert.Equal(t, "Test Pack", entries[1].Text, "the name must be shown")
	assert.Equal(t, "Test Author", entries[2].Text, "the author must be shown")
	assert.Equal(t, "one,two", entries[3].Text, "the keywords must be shown")

	spinners := collectObjects[*widgets.MajorMinorVersionSpinner](dlg.content)
	require.Len(t, spinners, 1, "the dialog must have a spinner for the version")
	assert.Equal(t, "2.8", spinners[0].GetValue(), "the version must be shown")
}
