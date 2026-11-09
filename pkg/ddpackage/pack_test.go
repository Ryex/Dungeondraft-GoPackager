package ddpackage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
)

// TestUpdateFromPathsIgnoresPathsOutsideTheRoot verifies that a path that
// is not under the unpacked folder of the package is not indexed.
func TestUpdateFromPathsIgnoresPathsOutsideTheRoot(t *testing.T) {
	dir := testutil.DefaultFixture().Write(t)

	pkg := NewPackage(logrus.New())
	if err := pkg.LoadUnpackedFromFolder(dir); err != nil {
		t.Fatalf("LoadUnpackedFromFolder() error = %v", err)
	}
	if errs := pkg.BuildFileList(); len(errs) > 0 {
		t.Fatalf("BuildFileList() errors = %v", errs)
	}

	before := pkg.FileList().Size()

	// a file outside of the unpacked folder of the package
	outsideFile := filepath.Join(t.TempDir(), "Outside.json")
	if err := os.WriteFile(outsideFile, []byte(`{"outside": true}`), 0o644); err != nil {
		t.Fatalf("could not write the file outside of the package: %v", err)
	}

	if errs := pkg.UpdateFromPaths([]string{outsideFile}); len(errs) > 0 {
		t.Fatalf("UpdateFromPaths() errors = %v", errs)
	}

	if got := pkg.FileList().Size(); got != before {
		t.Errorf("file list size = %d, want %d: the file outside of the package must not be indexed",
			got, before)
	}
}
