package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

// writePreviewFixtureFile writes a file of the preview fixture, creating the
// directory it lives in.
func writePreviewFixtureFile(t *testing.T, path string, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// writePreviewFixtureFiles adds the two JSON files the size gate of the file
// preview has to decide about to a fixture folder:
//
//   - data/long.json has far more lines than the preview shows, but is small
//     enough to be read.
//   - data/huge.json has only a few lines, but is larger than the size gate,
//     so it must never be read for the preview.
func writePreviewFixtureFiles(t *testing.T, dir string) {
	t.Helper()

	var long strings.Builder
	long.WriteString("[\n")
	for i := 0; i < 300; i++ {
		long.WriteString(`  "a line of text"`)
		if i < 299 {
			long.WriteString(",")
		}
		long.WriteString("\n")
	}
	long.WriteString("]\n")

	huge := `{"blob":"` + strings.Repeat("a", 5<<20) + `"}` + "\n"

	writePreviewFixtureFile(t, filepath.Join(dir, "data", "long.json"), long.String())
	writePreviewFixtureFile(t, filepath.Join(dir, "data", "huge.json"), huge)
}

// previewInfo returns the info of a resource of the loaded package.
func previewInfo(t *testing.T, pkg *ddpackage.Package, relPath string) *structures.FileInfo {
	t.Helper()

	info, err := pkg.GetResourceInfo(resPath(relPath))
	require.NoError(t, err, "the fixture must be indexed under %s", relPath)

	return info
}

// TestFilePreviewSizeGate verifies that the preview of a text file is gated
// on the size of the file before the file is read:
//
//   - a file with too many lines shows the "too large" label, even though it
//     is small enough to be read.
//   - a file above the size gate shows the "too large" label as well, without
//     its content ever being read.
//   - a small file with few enough lines shows its text.
func TestFilePreviewSizeGate(t *testing.T) {
	a := newTestApp(t)

	dir := testutil.DefaultFixture().Write(t)
	writePreviewFixtureFiles(t, dir)

	// the JSON files of the fixture are only previewed when they are
	// indexed, and they are indexed because of their extension
	require.Contains(t, ddpackage.DefaultValidExt(), ".json")

	result, err := a.loadUnpackedSync(dir, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Pkg)
	require.NoError(t, result.TagsErr)
	require.NoError(t, result.MetadataErr)
	a.setCurrentPackage(result.Pkg)

	// every preview computes the md5 of its file in the background and
	// touches the interface while doing so, and building a preview measures
	// text as well: the background work of a preview has to be finished
	// before the next preview is built, or it races with the measurement
	// of the text of the new preview in the font cache of Fyne
	// long.json: over the line count rule, far below the size gate
	longPreview := a.buildFilePreview(previewInfo(t, result.Pkg, "data/long.json"))
	waitForBackgroundWork(t, a)
	// huge.json: only a few lines, above the size gate
	hugePreview := a.buildFilePreview(previewInfo(t, result.Pkg, "data/huge.json"))
	waitForBackgroundWork(t, a)
	// the tags file of the fixture: small and short enough to be shown
	tagsPreview := a.buildFilePreview(previewInfo(t, result.Pkg, "data/default.dungeondraft_tags"))
	// the assertions below measure the text of the previews, which may only
	// happen while no background work touches the interface
	waitForBackgroundWork(t, a)

	uitest.Expect(t, uitest.In(longPreview),
		uitest.Has("the too large label of the line count rule",
			uitest.Named(nameTooLargeLabel)),
		uitest.NotExists("a text preview", uitest.Named(nameTextPreview)),
	)

	uitest.Expect(t, uitest.In(hugePreview),
		uitest.Has("the too large label of the size gate",
			uitest.Named(nameTooLargeLabel)),
		uitest.NotExists("a text preview", uitest.Named(nameTextPreview)),
	)

	uitest.Expect(t, uitest.In(tagsPreview),
		uitest.Has("the text preview of a small file",
			uitest.Named(nameTextPreview)),
		uitest.NotExists("the too large label", uitest.Named(nameTooLargeLabel)),
	)
}
