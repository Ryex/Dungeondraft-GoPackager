package ddpackage

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
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
