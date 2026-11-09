package ddpackage

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

// TestGetOrUpdateResourceMd5UnknownPackageState verifies that a package that
// is not loaded reports an error for an unknown package state instead of
// falling through to the hash and reporting the md5 of the empty string as a
// success.
func TestGetOrUpdateResourceMd5UnknownPackageState(t *testing.T) {
	pkg := NewPackage(logrus.New())
	if pkg.mode != PackageModeUnloaded {
		t.Fatalf("NewPackage() mode = %v, want %v", pkg.mode, PackageModeUnloaded)
	}

	fi := &structures.FileInfo{
		Path:    "...",
		ResPath: "res://packs/x/y.png",
	}
	require.False(t, fi.Md5.IsValid())

	var (
		gotMd5 string
		gotErr error
	)
	done := make(chan struct{})
	pkg.GetOrUpdateResourceMd5(fi, func(md5 string, err error) {
		gotMd5 = md5
		gotErr = err
		close(done)
	})
	<-done

	require.Equal(t, "", gotMd5, "callback must not report a hash for an unknown package state")
	require.Error(t, gotErr, "callback must report an error for an unknown package state")
	require.ErrorIs(t, gotErr, ErrPackageNotLoaded)
	require.False(t, fi.Md5.IsValid(), "file info md5 must stay invalid")
}
