package ddpackage

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// countingFieldLogger counts the md5 computations: the unpacked branch of the
// md5 worker builds its logger with WithField("filePath", ...) exactly once
// per computation, so the number of those calls is the number of times the
// file was read.
type countingFieldLogger struct {
	logrus.FieldLogger
	filePathFields atomic.Int64
}

func (c *countingFieldLogger) WithField(key string, value interface{}) *logrus.Entry {
	if key == "filePath" {
		c.filePathFields.Add(1)
	}
	return c.FieldLogger.WithField(key, value)
}

// TestGetOrUpdateResourceMd5SingleFlight fires eight concurrent requests for
// the md5 of one unpacked file: every callback must receive the identical,
// correct hash with no error, and the file must be read exactly once — the
// other seven callers queue behind the one inflight computation.
func TestGetOrUpdateResourceMd5SingleFlight(t *testing.T) {
	path := filepath.Join(t.TempDir(), "single.bin")
	content := []byte("single flight payload")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	log := &countingFieldLogger{FieldLogger: logrus.New()}
	pkg := NewPackage(log)
	pkg.mode = PackageModeUnpacked

	fi := &structures.FileInfo{
		Path:    path,
		ResPath: "res://packs/x/single.bin",
	}
	require.False(t, fi.Md5.IsValid())

	want := md5.Sum(content)
	wantHex := hex.EncodeToString(want[:])

	const callers = 8
	var (
		mu        sync.Mutex
		gotHashes []string
		gotErrs   []error
	)
	var callbacks atomic.Int64
	done := make(chan struct{})

	for i := 0; i < callers; i++ {
		go func() {
			pkg.GetOrUpdateResourceMd5(fi, func(hash string, err error) {
				mu.Lock()
				gotHashes = append(gotHashes, hash)
				gotErrs = append(gotErrs, err)
				mu.Unlock()
				if callbacks.Add(1) == callers {
					close(done)
				}
			})
		}()
	}

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("only %d of %d md5 callbacks fired within 30s", callbacks.Load(), callers)
	}

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, gotHashes, callers)
	for i, got := range gotHashes {
		require.NoError(t, gotErrs[i], "callback %d must not report an error", i)
		require.Equal(t, wantHex, got, "callback %d must receive the correct hash", i)
	}
	require.Equal(t, int64(1), log.filePathFields.Load(),
		"the file must be read exactly once (single-flight)")
	require.Equal(t, wantHex, fi.Md5.String())
}

// TestGetOrUpdateResourceMd5PublishedUnderLock pins the publication order of
// the single-flight: fi.Md5 may only become valid once the hash is published,
// and the publish must have happened before the callback runs. The lock itself
// is proven by the -race run of Task 32; this test asserts the observable
// order.
func TestGetOrUpdateResourceMd5PublishedUnderLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "publish.bin")
	content := []byte("published under flLock")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	pkg := NewPackage(logrus.New())
	pkg.mode = PackageModeUnpacked

	fi := &structures.FileInfo{
		Path:    path,
		ResPath: "res://packs/x/publish.bin",
	}
	require.False(t, fi.Md5.IsValid(), "md5 must start invalid")

	want := md5.Sum(content)
	wantHex := hex.EncodeToString(want[:])

	var (
		validAtCallback bool
		hashAtCallback  string
		errAtCallback   error
	)
	done := make(chan struct{})
	pkg.GetOrUpdateResourceMd5(fi, func(hash string, err error) {
		// the hash must already be published when the callback runs
		validAtCallback = fi.Md5.IsValid()
		hashAtCallback = fi.Md5.String()
		errAtCallback = err
		close(done)
	})

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("md5 callback did not fire within 30s")
	}

	require.NoError(t, errAtCallback)
	require.True(t, validAtCallback,
		"fi.Md5 must be published before the callback runs")
	require.Equal(t, wantHex, hashAtCallback)
	require.Equal(t, wantHex, fi.Md5.String())
}

// TestUpdateResourcesMd5DoesNotDeadlock guards the lock scope of
// UpdateResources: the md5 single-flight publishes hashes under flLock, so
// UpdateResources must not hold flLock while its workers wait for those
// callbacks — that combination deadlocks, because the publish in the md5
// worker can never acquire the lock the workers' caller still holds.
func TestUpdateResourcesMd5DoesNotDeadlock(t *testing.T) {
	pkg := NewPackage(logrus.New())
	pkg.mode = PackageModeUnpacked

	dir := t.TempDir()
	var files []*structures.FileInfo
	for _, name := range []string{"a.bin", "b.bin", "c.bin"} {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte("resource "+name), 0o644))
		fi := &structures.FileInfo{
			Path:    path,
			ResPath: "res://packs/x/" + name,
		}
		pkg.AddResource(fi)
		files = append(files, fi)
	}

	done := make(chan []error, 1)
	go func() {
		done <- pkg.UpdateResources(UpdateResourcesOptions{Md5: true}, func(p float64, max int64) {})
	}()

	select {
	case errs := <-done:
		require.Empty(t, errs, "md5 update must not report errors")
	case <-time.After(30 * time.Second):
		t.Fatal("UpdateResources(UpdateResourcesOptions{Md5: true}) did not return within 30s — " +
			"it deadlocked: flLock is held across the workers while the md5 publish needs it")
	}

	for _, fi := range files {
		require.True(t, fi.Md5.IsValid(), "md5 of %s must be published", fi.ResPath)
	}
}
