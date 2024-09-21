package structures

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/sirupsen/logrus"
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
	GODOT_PACKAGE_MAGIC uint32 = 0x43504447 // PCK archive Magic

	GODOT_PACKAGE_FORMAT uint32 = 1 // package format should stay at 1 unless GoDot changes

	GODOT_MAJOR uint32 = 3 // latest dungeondraft is built with 3.4.2
	GODOT_MINOR uint32 = 4 // these should update with dungeondraft but no harm should come if they don't (presumably)
	GODOT_PATCH uint32 = 2
)

// DefaultPackageHeader gives the defaults Package Headers you would expect
func DefaultPackageHeader() *PackageHeaders {
	return &PackageHeaders{
		Magic:             GODOT_PACKAGE_MAGIC,
		PackFormatVersion: GODOT_PACKAGE_FORMAT,
		VersionMajor:      GODOT_MAJOR,
		VersionMinor:      GODOT_MINOR,
		VersionPatch:      GODOT_PATCH,
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

// FileInfoBytes is a struct used for readign and writing the encoded file information bytes
type FileInfoBytes struct {
	Offset uint64
	Size   uint64
	Md5    [16]byte
}

// Write out binary bytes to io
func (fi *FileInfoBytes) Write(out io.Writer) (err error) {
	err = binary.Write(out, binary.LittleEndian, fi)
	return
}

// SizeOf the headers in bytes
func (fi *FileInfoBytes) SizeOf() int64 {
	return int64(binary.Size(fi))
}

// FileInfo stores file information
type FileInfo struct {
	Path        string
	Offset      int64
	Size        int64
	Md5         string
	ResPath     string
	ResPathSize int32
}

// FileInfoPair groups a FileInfo and iot's Bytes equivalent
type FileInfoPair struct {
	Info      FileInfo
	InfoBytes FileInfoBytes
}

// Package stores package information for the pack.json
type Package struct {
	Name           string               `json:"name"`
	ID             string               `json:"id"`
	Version        string               `json:"version"`
	Author         string               `json:"author"`
	Keywords       []string             `json:"keywords"`
	Allow3rdParty  bool                 `json:"allow_3rd_party_mapping_software_to_read"`
	ColorOverrides CustomColorOverrides `json:"custom_color_overrides,omitempty"`
}

type CustomColorOverrides struct {
	Enabled       bool    `json:"enabled"`
	MinRedness    float64 `json:"min_redness"`
	MinSaturation float64 `json:"min_saturation"`
	RedTolerance  float64 `json:"red_tolerance"`
}

func (o *CustomColorOverrides) String() string {
	return fmt.Sprintf("%v", *o)
}

func (o *CustomColorOverrides) Set(value string) error {
	parts := strings.Split(value, ",")
	defaultErr := errors.New("Color Overrides format is <redness>,<saturation>,<red_tolerance>")
	if len(parts) != 3 {
		return defaultErr
	}
	if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
		o.MinRedness = v
	} else {
		return defaultErr
	}
	if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
		o.MinSaturation = v
	} else {
		return defaultErr
	}
	if v, err := strconv.ParseFloat(parts[2], 64); err == nil {
		o.RedTolerance = v
	} else {
		return defaultErr
	}
	o.Enabled = true
	return nil
}

// FileInfoList used to calculate the size of the list and properly set offsets in the info
type FileInfoList struct {
	FileList []FileInfoPair
	Size     int64
}

// NewFileInfoList builds a valid FileInfoList with size information
func NewFileInfoList(fileList []FileInfo) *FileInfoList {
	L := &FileInfoList{}

	var totalSize int64

	for _, fInfo := range fileList {
		fInfoBytes := FileInfoBytes{}

		fInfoBytes.Size = uint64(fInfo.Size)
		fInfoBytes.Offset = uint64(fInfo.Offset)

		fInfo.ResPathSize = int32(binary.Size([]byte(fInfo.ResPath)))
		totalSize += int64(binary.Size(fInfo.ResPathSize))
		totalSize += int64(fInfo.ResPathSize)
		totalSize += int64(binary.Size(fInfoBytes))

		L.FileList = append(L.FileList, FileInfoPair{
			Info:      fInfo,
			InfoBytes: fInfoBytes,
		})
	}

	L.Size = totalSize

	return L
}

// UpdateOffsets updates all offset information to start from the passed point
// there are indications that GoDot has the ability to control alignment of packed file data.
// this function does not handle this
func (fil *FileInfoList) UpdateOffsets(offset int64) {
	var newList []FileInfoPair
	for _, pair := range fil.FileList {
		pair.Info.Offset = offset
		pair.InfoBytes.Offset = uint64(offset)

		offset += pair.Info.Size

		newList = append(newList, pair)
	}

	fil.FileList = newList
}

// Write out headers and file contents to io
func (fil *FileInfoList) Write(log logrus.FieldLogger, out io.Writer, offset int64) (err error) {
	log.Debug("updating offsets...")
	fil.UpdateOffsets(fil.Size + offset)

	log.Debug("writing files...")
	err = fil.WriteHeaders(log, out)
	if err != nil {
		return
	}

	err = fil.WriteFiles(log, out)

	return
}

// WriteHeaders write out the headers to io
func (fil *FileInfoList) WriteHeaders(log logrus.FieldLogger, out io.Writer) (err error) {
	log.Debug("writing file headers")
	for _, pair := range fil.FileList {

		// write path length
		err = binary.Write(out, binary.LittleEndian, pair.Info.ResPathSize)
		if !utils.CheckErrorWrite(log, err) {
			return
		}

		// write filepath
		err = binary.Write(out, binary.LittleEndian, []byte(pair.Info.ResPath))
		if !utils.CheckErrorWrite(log, err) {
			return
		}

		// write fileinfo
		err = pair.InfoBytes.Write(out)
		if !utils.CheckErrorWrite(log, err) {
			return
		}

	}

	return
}

// WriteFiles write the contents of the files in the list to io
// this function does NOT handle padding in-between filedata. this may be a problem
func (fil *FileInfoList) WriteFiles(log logrus.FieldLogger, out io.Writer) (err error) {
	log.Debug("writing file data")
	for _, pair := range fil.FileList {
		err = fil.writeFile(log.WithField("file", pair.Info.Path), out, pair.Info)
		if err != nil {
			return
		}
	}

	return
}

func (fil *FileInfoList) writeFile(log logrus.FieldLogger, out io.Writer, info FileInfo) (err error) {
	log.Debug("writing")

	data, err := os.ReadFile(info.Path)
	if err != nil {
		log.WithError(err).Error("error reading file")
		return
	}

	n, err := out.Write(data)
	if !utils.CheckErrorWrite(log, err) {
		return
	}
	if int64(n) != info.Size {
		err = errors.New("write of wrong size")
		log.WithField("expectedWriteSize", info.Size).
			WithField("writeSize", n).
			WithError(err).Error("failed to write file")
		return
	}

	return
}
