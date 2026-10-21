package gui

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	log "github.com/sirupsen/logrus"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

// newSpecApp creates a software rendered App for the UI specification tests.
//
// The locale is pinned to English before the app is set up: the app otherwise
// uses the locale of the machine the tests run on, which would make the
// assertions on the texts of the interface fail on non English systems.
func newSpecApp(t *testing.T) *App {
	t.Helper()

	a := NewAppWithFyneApp(test.NewApp())
	a.app.Preferences().SetString("locale", "en")
	// the filesystem watcher would keep running past the end of the test
	a.packageWatcherDisabled = true
	a.setup()
	t.Cleanup(a.clean)

	return a
}

// namedPart returns the part of the user interface with the given name.
//
// It fails the test when the interface has no part with that name, so that a
// renamed part does not turn into a puzzling "found none" failure.
func namedPart(t *testing.T, a *App, name string) fyne.CanvasObject {
	t.Helper()

	obj := a.ui.Get(name)
	if obj == nil {
		t.Fatalf("the user interface has no part named %q", name)
	}
	return obj
}

// blockUnpackedLoad replaces the folder loader of the app with one that
// blocks until the returned function is called. This makes the state of the
// interface while the pack is loading observable.
func blockUnpackedLoad(a *App) (release func()) {
	releaseCh := make(chan struct{})
	a.loadUnpacked = func(
		path string,
		progress func(p float64, curPath string),
	) (unpackedLoadResult, error) {
		<-releaseCh
		return a.loadUnpackedSync(path, progress)
	}

	var once sync.Once
	return func() {
		once.Do(func() { close(releaseCh) })
	}
}

// blockPackedLoad replaces the packed file loader of the app with one that
// blocks until the returned function is called. This makes the state of the
// interface while the pack is loading observable.
func blockPackedLoad(a *App) (release func()) {
	releaseCh := make(chan struct{})
	a.loadPacked = func(
		path string,
		progress func(p float64, curRes string),
	) (*ddpackage.Package, error) {
		<-releaseCh
		return a.loadPackSync(path, progress)
	}

	var once sync.Once
	return func() {
		once.Do(func() { close(releaseCh) })
	}
}

// waitForPackLoad waits until the pack that is being loaded has been shown
// and the interface has been laid out.
//
// The buttons of the window are enabled again as the last step of showing
// the loaded pack, so they are the signal that the load is done. The layout
// of the new content is settled as well, because it is built while the pack
// is loaded.
//
// The load itself runs in the background and updates the interface when it is
// done, so it is waited for as well: the interface must not be touched while
// the load is still running.
func waitForPackLoad(t *testing.T, a *App) {
	t.Helper()

	require.Eventually(t, func() bool {
		disabled, err := a.disableButtons.Get()
		return err == nil && !disabled
	}, 10*time.Second, 20*time.Millisecond, "the pack must be loaded")

	loaded := make(chan struct{})
	go func() {
		a.bgWG.Wait()
		close(loaded)
	}()
	select {
	case <-loaded:
	case <-time.After(10 * time.Second):
		t.Fatal("the background load must finish")
	}

	waitForStableLayout(t, a)
}

// waitForBackgroundWork waits until the background work that touches the
// interface is done.
//
// The loads of a package and the computations of the md5 hashes run in the
// background and update the interface, so the interface must not be touched
// while they are running.
func waitForBackgroundWork(t *testing.T, a *App) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		a.bgWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the background work must finish")
	}
}

// waitForStableLayout waits until the layout of the main content stops
// changing.
//
// The content is built in the background, so it can already be part of the
// object tree while the widgets of it are still being laid out. Waiting for
// the layout to settle keeps the assertions on the positions of the widgets
// from running against a half finished layout.
func waitForStableLayout(t *testing.T, a *App) {
	t.Helper()

	// the content object itself stays the same, only its contents change,
	// so it is resolved once, outside of the polling below
	content := namedPart(t, a, nameContent)

	var last string
	stable := 0
	require.Eventually(t, func() bool {
		current := uitest.LayoutFingerprint(content)
		if current == last {
			stable++
			return stable >= 2
		}
		last = current
		stable = 0
		return false
	}, 10*time.Second, 10*time.Millisecond, "the layout of the main content must settle")
}

// specTexts returns the texts the interface is expected to show, read from
// the same translation catalog the application uses.
func specTexts() map[string]string {
	return map[string]string{
		"pathPlaceholder":   lang.X("pathInput.placeholder", "Path to .dungeondraft_pack or folder"),
		"openPack":          lang.X("packBtn.text", "Open pack"),
		"openFolder":        lang.X("folderBtn.text", "Open folder"),
		"outPath":           lang.X("pack.outPath.placeholder", "Where to save .dungeondraft_pack file"),
		"unpackOutPath":     lang.X("unpack.outPath.placeholder", "Where to extract resources"),
		"browse":            lang.X("browse", "Browse"),
		"thumbnails":        lang.X("pack.option.thumbnails.text", "Generate thumbnails"),
		"editPack":          lang.X("pack.editPackBtn.text", "Edit settings"),
		"generateTags":      lang.X("pack.generateTagsBtn.label", "Generate Tags"),
		"tagSets":           lang.X("pack.tagSetsBtn.text", "Edit Tag Sets"),
		"overwrite":         lang.X("pack.option.overwrite.text", "Overwrite existing files"),
		"package":           lang.X("pack.packBtn.text", "Package"),
		"packageInfo":       lang.X("unpack.infoBtn.text", "Package Information"),
		"viewTagSets":       lang.X("unpack.tagSetsBtn.text", "View Tag Sets"),
		"unpackOverwrite":   lang.X("unpack.option.overwrite.text", "Overwrite existing files"),
		"ripTextures":       lang.X("unpack.option.repTex.text", "Rip Textures to Png"),
		"extractThumbnails": lang.X("unpack.option.thumbnails.text", "Extract thumbnails"),
		"extract":           lang.X("unpack.extractBtn.text", "Extract"),
	}
}

// defaultTextLines returns the lines of the text the main content shows when
// no pack has been opened yet.
func defaultTextLines() []string {
	return strings.Split(lang.X(
		"main.defaultText",
		"Enter a path to get started.\n"+
			"You can also drag and drop a resource pack file or unpacked folder.",
	), "\n")
}

// packFixtureDir writes the default fixture into a temporary directory and
// returns the path of the unpacked pack.
func packFixtureDir(t *testing.T) string {
	t.Helper()

	return testutil.DefaultFixture().Write(t)
}

// packedFixtureFile writes the default fixture, packs it up and returns the
// path of the resulting .dungeondraft_pack file.
func packedFixtureFile(t *testing.T) string {
	t.Helper()

	dir := packFixtureDir(t)
	return packFixture(t, dir)
}

// packFixture packs the fixture in the given directory up and returns the
// path of the resulting .dungeondraft_pack file.
func packFixture(t *testing.T, dir string) string {
	t.Helper()

	result, err := loadUnpackedFixture(t, dir)
	require.NoError(t, err)

	outDir := t.TempDir()
	require.NoError(t, result.PackPackage(outDir, ddpackage.PackOptions{}))

	return filepath.Join(outDir, result.Name()+".dungeondraft_pack")
}

// loadUnpackedFixture loads the package of an unpacked fixture folder.
func loadUnpackedFixture(t *testing.T, dir string) (*ddpackage.Package, error) {
	t.Helper()

	pkg := ddpackage.NewPackage(log.WithField("path", dir))
	if err := pkg.LoadUnpackedFromFolder(dir); err != nil {
		return nil, err
	}
	if errs := pkg.BuildFileList(); len(errs) > 0 {
		return nil, errs[0]
	}
	return pkg, nil
}
