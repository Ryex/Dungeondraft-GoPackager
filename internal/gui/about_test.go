package gui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/version"
)

// metadataApp is a Fyne app with a fixed metadata, for the tests of the about
// dialog. The app of the test driver reports an empty metadata.
type metadataApp struct {
	fyne.App
	metadata fyne.AppMetadata
}

func (a *metadataApp) Metadata() fyne.AppMetadata {
	return a.metadata
}

// withBuildVersion runs the test with the given commit and modified state of
// the working tree, so that the version the dialog shows is predictable.
func withBuildVersion(t *testing.T, commit string, dirty bool) {
	t.Helper()

	oldVersion, oldCommit, oldDirty := version.Version, version.Commit, version.Dirty
	t.Cleanup(func() { version.Version, version.Commit, version.Dirty = oldVersion, oldCommit, oldDirty })

	version.Version = ""
	version.Commit = commit
	version.Dirty = dirty
}

// newAboutTestApp returns an app that reports the given name and version.
func newAboutTestApp(name, version string) fyne.App {
	return &metadataApp{
		App: test.NewApp(),
		metadata: fyne.AppMetadata{
			Name:    name,
			Version: version,
		},
	}
}

// aboutTexts returns the texts the content of the about dialog shows.
func aboutTexts(obj fyne.CanvasObject) []string {
	var texts []string

	for _, rich := range collectObjects[*widget.RichText](obj) {
		for _, segment := range rich.Segments {
			if text, ok := segment.(*widget.TextSegment); ok {
				texts = append(texts, text.Text)
			}
		}
	}

	return texts
}

func TestAboutContentShowsTheVersionWithTheCommit(t *testing.T) {
	withBuildVersion(t, "1a2b3c4d5e6f7a8b9c0d", false)

	texts := aboutTexts(newAboutContent(newAboutTestApp("Dungeondraft GoPackager", "2.1.0"), nil))

	assert.Contains(t, texts, "Dungeondraft GoPackager",
		"the about dialog must show the name of the app")
	assert.Contains(t, texts,
		lang.X("about.version", "Version")+": 2.1.0 (1a2b3c4)",
		"the about dialog must show the version with the commit of the build")
	assert.Contains(t, texts,
		lang.X("greeting.sub-title", "Package, edit, and prepare Dungeondraft resource packs"),
		"the about dialog must show the description of the app")
}

func TestAboutContentMarksAModifiedWorkingTree(t *testing.T) {
	withBuildVersion(t, "1a2b3c4d5e6f7a8b9c0d", true)

	texts := aboutTexts(newAboutContent(newAboutTestApp("Dungeondraft GoPackager", "2.1.0"), nil))

	assert.Contains(t, texts,
		lang.X("about.version", "Version")+": 2.1.0 (1a2b3c4-dirty)",
		"the about dialog must mark a build of a modified working tree")
}

func TestAboutContentFallsBackToTheNameOfTheApp(t *testing.T) {
	withBuildVersion(t, "1a2b3c4d5e6f7a8b9c0d", false)

	texts := aboutTexts(newAboutContent(newAboutTestApp("", ""), nil))

	assert.Contains(t, texts, lang.X("greeting.title", "Dungeondraft-GoPackager"),
		"the about dialog must show the name of the app when the metadata has none")
}

func TestAboutContentShowsTheVersionBelowTheName(t *testing.T) {
	withBuildVersion(t, "1a2b3c4d5e6f7a8b9c0d", false)

	name := "Dungeondraft GoPackager"
	content := newAboutContent(newAboutTestApp(name, "2.1.0"), nil)

	// the name and the version share one rich text, the description is in
	// a rich text of its own
	var appData *widget.RichText
	for _, rich := range collectObjects[*widget.RichText](content) {
		if len(rich.Segments) > 0 && rich.Segments[0].Textual() == name {
			appData = rich
		}
	}
	require.NotNil(t, appData, "the about dialog must show the name of the app")

	size := fyne.NewSize(400, 200)
	appData.Resize(size)
	renderer := test.WidgetRenderer(appData)
	renderer.Layout(size)

	positions := make(map[string]*canvas.Text)
	for _, obj := range renderer.Objects() {
		if text, ok := obj.(*canvas.Text); ok && text.Text != "" {
			positions[text.Text] = text
		}
	}

	nameText, ok := positions[name]
	require.True(t, ok, "the name must be rendered")

	versionLabel := lang.X("about.version", "Version") + ": " + version.Display("2.1.0")
	versionText, ok := positions[versionLabel]
	require.True(t, ok, "the version must be rendered")

	assert.GreaterOrEqual(t, versionText.Position().Y, nameText.Position().Y+nameText.Size().Height,
		"the version must be shown under the name, not next to it")
}

func TestAboutContentShowsTheLinks(t *testing.T) {
	link := widget.NewHyperlink("Website", nil)

	links := collectObjects[*widget.Hyperlink](
		newAboutContent(newAboutTestApp("Dungeondraft GoPackager", "2.1.0"), []*widget.Hyperlink{link}),
	)

	require.Len(t, links, 1, "the about dialog must show the links it is given")
	assert.Equal(t, "Website", links[0].Text)
}
