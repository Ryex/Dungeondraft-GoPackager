package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

// TestUnpackTargetPath checks that the name of a pack can not steer the
// extraction out of the chosen output directory: whatever the name is, the
// target has to stay a single segment under the base path.
func TestUnpackTargetPath(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name     string
		packName string
		// wantSeg is the expected single segment under base. It is empty
		// when the expectation depends on the platform, like for a Windows
		// style path that is one segment on POSIX systems.
		wantSeg string
	}{
		{name: "traversal escapes two levels", packName: "../../evil", wantSeg: "evil"},
		{name: "parent directory", packName: "..", wantSeg: "pack"},
		{name: "empty name", packName: "", wantSeg: "pack"},
		{name: "traversal through segments", packName: "a/../../b", wantSeg: "b"},
		{name: "windows drive path", packName: `C:\evil`},
		{name: "normal name with space", packName: "My Pack", wantSeg: "My Pack"},
		{name: "normal simple name", packName: "mypack", wantSeg: "mypack"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := unpackTargetPath(base, tc.packName)

			rel, err := filepath.Rel(base, result)
			require.NoError(t, err, "result %q is not relative to the base %q", result, base)
			require.False(t, filepath.IsAbs(rel), "relative result must not be absolute, got %q", rel)
			require.NotEqual(t, "..", rel, "result escapes the base")
			require.False(t,
				strings.HasPrefix(rel, ".."+string(filepath.Separator)),
				"result %q escapes the base", rel,
			)
			require.False(t,
				strings.Contains(rel, string(filepath.Separator)),
				"result must be a single segment under the base, got %q", rel,
			)

			if tc.wantSeg != "" {
				require.Equal(t, filepath.Join(base, tc.wantSeg), result,
					"pack name %q must pass through as one segment", tc.packName,
				)
			}
		})
	}
}

// TestExtractPackageFailureReenablesButtons verifies that the buttons of the
// window are enabled again when the extraction fails. They drive Open Pack
// and Open Folder — the controls a user needs to recover from the failure —
// so the error path of the task has to re-enable them like the success path
// does.
func TestExtractPackageFailureReenablesButtons(t *testing.T) {
	// the pack file is created before the app, so that the package is
	// closed before the temporary directory of the fixture is removed
	packFile := packedFixtureFile(t)

	a := newTestApp(t)

	result, err := a.loadPackSync(packFile, func(float64, string) {})
	require.NoError(t, err)
	require.NotNil(t, result.Pkg)
	a.setPackContent(result.Pkg)

	// extracting into a path that exists as a file fails
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a folder"), 0o644))

	a.extractPackage(blocker, ddpackage.UnpackOptions{})

	require.Eventually(t, func() bool {
		disabled, err := a.disableButtons.Get()
		return err == nil && !disabled
	}, 10*time.Second, 20*time.Millisecond,
		"the buttons must be enabled again after a failed extraction")
}
