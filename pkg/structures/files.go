package structures

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/sirupsen/logrus"
)

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
	Path             string
	Offset           int64
	Size             int64
	Md5              string
	ResPath          string
	ResPathSize      int32
	RelPath          string
	ThumbnailPath    string
	ThumbnailResPath string
	Image            image.Image
	ImageFormat      string
	PngImage         []byte
}

var idTrimPrefixRegex = regexp.MustCompile(`^([\w\-. ]+)/`)

func (fi *FileInfo) CalcRelPath() string {
	if fi.RelPath != "" {
		return fi.RelPath
	}
	path := strings.TrimPrefix(fi.ResPath, "res://packs/")
	path = idTrimPrefixRegex.ReplaceAllString(path, "")
	return path
}

func (fi *FileInfo) IsData() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "data/")
}

func (fi *FileInfo) IsWallData() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "data/walls/")
}

func (fi *FileInfo) IsTilesetData() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "data/tilesets/")
}

func (fi *FileInfo) IsTexture() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/")
}

func (fi *FileInfo) IsThumbnail() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "thumbnails/")
}

func (fi *FileInfo) IsObject() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/paths/")
}

func (fi *FileInfo) IsTerrain() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/terrain/")
}

func (fi *FileInfo) IsMaterial() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/materials/")
}

func (fi *FileInfo) IsTileset() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/tilesets/")
}

func (fi *FileInfo) IsPattern() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/patterns/")
}

func (fi *FileInfo) IsWall() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/walls/")
}

func (fi *FileInfo) IsPath() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/paths/")
}

func (fi *FileInfo) IsPortal() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/portals/")
}

func (fi *FileInfo) IsLight() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/lights/")
}

// FileInfoPair groups a FileInfo and iot's Bytes equivalent
type FileInfoPair struct {
	Info      FileInfo
	InfoBytes FileInfoBytes
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
// Gogot has the ability to control alignment of packed file data.
// this function tries to handle this
func (fil *FileInfoList) UpdateOffsets(offset int64, alignment int) {
	for i := 0; i < len(fil.FileList); i++ {
		offset = utils.Align(offset, alignment)
		pair := &fil.FileList[i]
		pair.Info.Offset = offset
		pair.InfoBytes.Offset = uint64(offset)

		offset += pair.Info.Size
	}
}

// Write out headers and file contents to io
func (fil *FileInfoList) Write(log logrus.FieldLogger, out io.WriteSeeker, offset int64, alignment int, progressCallbacks ...func(p float64)) (err error) {
	log.Debug("updating offsets...")
	fil.UpdateOffsets(fil.Size+offset, alignment)

	log.Debug("writing files...")
	err = fil.WriteHeaders(log, out, alignment)
	if err != nil {
		return
	}

	err = fil.WriteFiles(log, out, alignment, progressCallbacks...)

	return
}

// WriteHeaders write out the headers to io
func (fil *FileInfoList) WriteHeaders(log logrus.FieldLogger, out io.WriteSeeker, alignment int) error {
	log.Debug("writing file headers")
	for i := 0; i < len(fil.FileList); i++ {
		pair := &fil.FileList[i]
		// write path length
		err := binary.Write(out, binary.LittleEndian, pair.Info.ResPathSize)
		if !utils.CheckErrorWrite(log, err) {
			return err
		}

		// write filepath
		err = binary.Write(out, binary.LittleEndian, []byte(pair.Info.ResPath))
		if !utils.CheckErrorWrite(log, err) {
			return err
		}

		// write fileinfo
		err = pair.InfoBytes.Write(out)
		if !utils.CheckErrorWrite(log, err) {
			return err
		}

	}

	curPos, err := utils.Tell(out)
	if err != nil {
		return err
	}
	offset := utils.Align(curPos, alignment)
	err = utils.Pad(out, offset-curPos)
	if err != nil {
		return err
	}

	return nil
}

var AlignmentError = errors.New("alignment error")

// WriteFiles write the contents of the files in the list to io
func (fil *FileInfoList) WriteFiles(log logrus.FieldLogger, out io.WriteSeeker, alignment int, progressCallbacks ...func(p float64)) error {
	log.Debug("writing file data")
	for i := 0; i < len(fil.FileList); i++ {
		pair := &fil.FileList[i]

		curPos, err := utils.Tell(out)
		if err != nil {
			return err
		}
		if pair.Info.Offset != curPos {
			err = errors.Join(AlignmentError, fmt.Errorf("%v != %v", curPos, pair.Info.Offset))
			log.WithError(err).WithField("file", pair.Info.Path).Error("misaligment of writer")
			return err
		}

		err = fil.writeFile(log.WithField("file", pair.Info.Path), out, &pair.Info)
		if err != nil {
			return err
		}

		curPos, err = utils.Tell(out)
		if err != nil {
			return err
		}

		offset := utils.Align(curPos, alignment)
		err = utils.Pad(out, offset-curPos)
		if err != nil {
			return err
		}

		for _, pcb := range progressCallbacks {
			pcb(float64(i+1) / float64(len(fil.FileList)))
		}
	}

	return nil
}

func (fil *FileInfoList) writeFile(l logrus.FieldLogger, out io.Writer, info *FileInfo) (err error) {
	l.Debug("writing")

	var data []byte
	if info.Image != nil && info.PngImage != nil {
		l.Debug("using png image data")
		data = info.PngImage
	} else {
		data, err = os.ReadFile(info.Path)
		if err != nil {
			l.WithError(err).Error("error reading file")
			return
		}
	}

	n, err := out.Write(data)
	if !utils.CheckErrorWrite(l, err) {
		return
	}
	if int64(n) != info.Size {
		err = errors.New("write of wrong size")
		l.WithField("expectedWriteSize", info.Size).
			WithField("writeSize", n).
			WithError(err).Error("failed to write file")
		return
	}

	return
}
