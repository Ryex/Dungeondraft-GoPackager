package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFileExistsAndDirExists covers the existence checks including the
// regression case from CI: a path whose parent is a file makes os.Stat fail
// with ENOTDIR (not os.ErrNotExist) on Linux/macOS, which used to dereference
// a nil os.FileInfo and panic.
func TestFileExistsAndDirExists(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "afile.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	subDir := filepath.Join(dir, "adir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		wantFile bool
		wantDir  bool
	}{
		{
			name:     "existing file",
			path:     filePath,
			wantFile: true,
			wantDir:  false,
		},
		{
			name:     "existing directory",
			path:     subDir,
			wantFile: false,
			wantDir:  true,
		},
		{
			name:     "non-existent path",
			path:     filepath.Join(dir, "does-not-exist"),
			wantFile: false,
			wantDir:  false,
		},
		{
			name:     "path whose parent is a file (regression: ENOTDIR)",
			path:     filepath.Join(filePath, "child"),
			wantFile: false,
			wantDir:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				assert.Equal(t, tt.wantFile, FileExists(tt.path))
				assert.Equal(t, tt.wantDir, DirExists(tt.path))
			})
		})
	}
}
