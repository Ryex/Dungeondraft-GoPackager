package ddpackage

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestJoinUnderBaseRejectsEscapingPaths(t *testing.T) {
	base := filepath.Join(t.TempDir(), "base")

	// rel "../../outside/x.png", "..", "", "C:\\evil", "/abs" -> ErrPathEscape
	rels := []string{
		"../../outside/x.png",
		"..",
		"",
		"/abs",
	}
	if filepath.VolumeName(`C:\`) != "" {
		// Drive-letter volume names only exist on windows, on other platforms
		// `C:\evil` is a plain relative file name inside base and stays there.
		rels = append(rels, "C:\\evil")
	}

	for _, rel := range rels {
		joined, err := joinUnderBase(base, rel)
		if !errors.Is(err, ErrPathEscape) {
			t.Errorf("joinUnderBase(%q, %q) = (%q, %v), want error %v", base, rel, joined, err, ErrPathEscape)
		}
		if joined != "" {
			t.Errorf("joinUnderBase(%q, %q) returned path %q, want empty path on error", base, rel, joined)
		}
	}
}

func TestJoinUnderBaseAcceptsNestedPaths(t *testing.T) {
	base := filepath.Join(t.TempDir(), "base")

	// rel "textures/objects" -> joined under base
	joined, err := joinUnderBase(base, "textures/objects")
	if err != nil {
		t.Fatalf("joinUnderBase(%q, %q) unexpected error: %v", base, "textures/objects", err)
	}
	if want := filepath.Join(base, "textures", "objects"); joined != want {
		t.Errorf("joinUnderBase(%q, %q) = %q, want %q", base, "textures/objects", joined, want)
	}

	// "a/../b" cleans to "b" inside base
	joined, err = joinUnderBase(base, "a/../b")
	if err != nil {
		t.Fatalf("joinUnderBase(%q, %q) unexpected error: %v", base, "a/../b", err)
	}
	if want := filepath.Join(base, "b"); joined != want {
		t.Errorf("joinUnderBase(%q, %q) = %q, want %q", base, "a/../b", joined, want)
	}
}
