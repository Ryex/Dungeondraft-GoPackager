package structures

import (
	"encoding/binary"
	"io"
)

// PackageHeaders is a struct used for reading and writing the encoded package headers
// most of these headers are hard coded and relate to the GoDot enging version that made the pack
// the defaults will need to be updated to reflect what version dungeondraft is built with
type PackageHeaders struct {
	Magic             uint32     // 1129333831 0x43504447 Godot's packed file magic header ("GDPC" in ASCII).
	PackFormatVersion uint32     // 1
	VersionMajor      uint32     // 3
	VersionMinor      uint32     // 1
	VersionPatch      uint32     // 0
	Reserved          [16]uint32 // [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0] This is reserved space in the V1 format
	FileCount         uint32
}

const (
	// PCK archive Magic
	GodotPackageMagic uint32 = 0x43504447

	GodotPackageFormat uint32 = 1 // package format should stay at 1 unless GoDot changes

	GodotMajor uint32 = 3 // latest dungeondraft is built with 3.4.2
	GodotMinor uint32 = 4 // these should update with dungeondraft but no harm should come if they don't (presumably)
	GodotPatch uint32 = 2
)

// DefaultPackageHeader gives the defaults Package Headers you would expect
func DefaultPackageHeader() *PackageHeaders {
	return &PackageHeaders{
		Magic:             GodotPackageMagic,
		PackFormatVersion: GodotPackageFormat,
		VersionMajor:      GodotMajor,
		VersionMinor:      GodotMinor,
		VersionPatch:      GodotPatch,
	}
}

// Write out binary bytes to io
func (ph *PackageHeaders) Write(out io.Writer) (err error) {
	err = binary.Write(out, binary.LittleEndian, ph)
	return
}

// SizeOf the headers in bytes
func (ph *PackageHeaders) SizeOf() int64 {
	return int64(binary.Size(ph))
}
