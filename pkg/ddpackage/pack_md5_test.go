package ddpackage

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackWritesMd5ThatLoadReadsBack round-trips the md5 hash of a resource
// through the pack header: the packer writes the hash of the bytes it stores,
// LoadFromPackedPath reads it back, and readPackedFileFromPackage verifies
// the stored bytes against it.
func TestPackWritesMd5ThatLoadReadsBack(t *testing.T) {
	fixture := testutil.DefaultFixture()
	unpackedDir := fixture.Write(t)

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	src := NewPackage(log.WithField("test", t.Name()))
	require.NoError(t, src.LoadUnpackedFromFolder(unpackedDir))
	require.Empty(t, src.BuildFileList())

	packDir := t.TempDir()
	require.NoError(t, src.PackPackageProgress(packDir, PackOptions{}, nil))
	packPath := filepath.Join(packDir, src.Name()+".dungeondraft_pack")

	loaded := NewPackage(log.WithField("test", t.Name()))
	require.NoError(t, loaded.LoadFromPackedPath(packPath, nil))
	t.Cleanup(loaded.Close)

	const resPath = "res://packs/testpack/textures/objects/Rock.png"
	fi := loaded.FileList().GetResource(resPath)
	require.NotNil(t, fi, "the packed package must contain %s", resPath)

	require.True(t, fi.Md5.IsValid(), "the md5 read back from the pack header must be valid")

	sourceBytes, err := os.ReadFile(filepath.Join(unpackedDir, "textures", "objects", "Rock.png"))
	require.NoError(t, err)
	sum := md5.Sum(sourceBytes)
	assert.Equal(t, hex.EncodeToString(sum[:]), fi.Md5.String(),
		"the stored md5 must be the hash of the file bytes of the unpacked source")

	// the md5 verification path accepts the hash that was written
	packedBytes, err := loaded.readPackedFileFromPackage(loaded.pkgFile, fi)
	require.NoError(t, err, "reading back a resource with a matching md5 must succeed")
	assert.Equal(t, sourceBytes, packedBytes,
		"the bytes read from the pack must be the bytes the stored hash covers")

	// a wrong hash must be rejected, so the verification really runs
	tampered := *fi
	tampered.Md5.Md5[0] ^= 0xff
	_, err = loaded.readPackedFileFromPackage(loaded.pkgFile, &tampered)
	assert.Error(t, err, "a mismatching md5 must fail the verification")
}

// writeDistinctPNG overwrites the texture at path with a 4x4 PNG whose bytes
// depend on the seed.
//
// The textures of the fixture are identical, which would hide a read that
// landed on the bytes of another resource of the pack.
func writeDistinctPNG(t *testing.T, path string, seed uint8) {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		for y := range 4 {
			img.SetNRGBA(x, y, color.NRGBA{R: seed, G: uint8(x*40 + y), B: 180, A: 255})
		}
	}

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	require.NoError(t, png.Encode(file, img))
}

// TestConcurrentPackedLoadAndMd5ReadsAreCorrect guards the pkgFileMu of
// readPackedFileFromPackage: the seek+read of the shared packed file moves
// its position, so concurrent readers must not interleave, or they serve
// each other's bytes.
//
// The two readers that overlap in the application are the preview of a
// resource (LoadResource on every click) and the md5 computation of the same
// resource (GetOrUpdateResourceMd5, which runs when the header hash is
// blank). Without the lock the preview gets bytes from the wrong offset and
// the computation publishes a wrong hash.
func TestConcurrentPackedLoadAndMd5ReadsAreCorrect(t *testing.T) {
	fixture := testutil.DefaultFixture()
	unpackedDir := fixture.Write(t)

	textures := []struct {
		rel  string
		seed uint8
	}{
		{"textures/objects/Rock.png", 1},
		{"textures/objects/Tree.png", 2},
		{"textures/walls/Stone.png", 3},
		{"textures/tilesets/Grass.png", 4},
	}
	for _, tex := range textures {
		writeDistinctPNG(t, filepath.Join(unpackedDir, filepath.FromSlash(tex.rel)), tex.seed)
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	src := NewPackage(log.WithField("test", t.Name()))
	require.NoError(t, src.LoadUnpackedFromFolder(unpackedDir))
	require.Empty(t, src.BuildFileList())

	packDir := t.TempDir()
	require.NoError(t, src.PackPackageProgress(packDir, PackOptions{}, nil))
	packPath := filepath.Join(packDir, src.Name()+".dungeondraft_pack")

	loaded := NewPackage(log.WithField("test", t.Name()))
	require.NoError(t, loaded.LoadFromPackedPath(packPath, nil))
	t.Cleanup(loaded.Close)

	type expectation struct {
		resPath string
		fi      *structures.FileInfo
		data    []byte
		hash    string
	}

	var want []expectation
	for _, tex := range textures {
		resPath := fmt.Sprintf("res://packs/%s/%s", fixture.ID, tex.rel)
		data, err := os.ReadFile(filepath.Join(unpackedDir, filepath.FromSlash(tex.rel)))
		require.NoError(t, err)
		sum := md5.Sum(data)

		fi := loaded.FileList().GetResource(resPath)
		require.NotNil(t, fi, "the packed package must contain %s", resPath)

		// a pack written without header hashes: the md5 has to be computed
		// while the preview reads the same file
		fi.Md5 = structures.Md5Hash{}
		require.False(t, fi.Md5.IsValid())

		want = append(want, expectation{
			resPath: resPath,
			fi:      fi,
			data:    data,
			hash:    hex.EncodeToString(sum[:]),
		})
	}
	require.Len(t, want, 4)

	const rounds = 32
	var (
		mu   sync.Mutex
		errs []error
	)
	record := func(err error) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}

	for round := 0; round < rounds; round++ {
		// every round publishes the hash again, so it is blanked again to
		// make the computation run for every round
		for _, e := range want {
			e.fi.Md5 = structures.Md5Hash{}
		}

		var wg sync.WaitGroup
		for _, e := range want {
			// the preview of the resource, as on a click in the tree
			wg.Add(1)
			go func() {
				defer wg.Done()
				data, err := loaded.LoadResource(e.resPath)
				if err != nil {
					record(fmt.Errorf("round %d LoadResource(%s): %w", round, e.resPath, err))
					return
				}
				if !bytes.Equal(data, e.data) {
					record(fmt.Errorf(
						"round %d LoadResource(%s) returned %d bytes that are not the bytes of the resource",
						round, e.resPath, len(data)))
				}
			}()

			// the md5 computation of the same resource
			wg.Add(1)
			go func() {
				defer wg.Done()
				var (
					hash   string
					md5Err error
				)
				done := make(chan struct{})
				loaded.GetOrUpdateResourceMd5(e.fi, func(h string, err error) {
					hash = h
					md5Err = err
					close(done)
				})
				<-done
				if md5Err != nil {
					record(fmt.Errorf("round %d md5(%s): %w", round, e.resPath, md5Err))
					return
				}
				if hash != e.hash {
					record(fmt.Errorf(
						"round %d md5(%s) = %s, want %s", round, e.resPath, hash, e.hash))
				}
			}()
		}
		wg.Wait()
	}

	mu.Lock()
	defer mu.Unlock()
	for _, err := range errs {
		t.Error(err)
	}
	require.Empty(t, errs,
		"concurrent packed reads must return the bytes of their own resource and the right hash")
}
