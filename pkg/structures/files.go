package structures

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	log "github.com/sirupsen/logrus"
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
	Path        string
	Offset      int64
	Size        int64
	Md5         string
	ResPath     string
	ResPathSize int32
	RelPath     string

	// if the file should have metadata this resource path points to that metadata
	// but that resource may not exist
	MetadataPath string
	// if the file could have a thumbnail this resource path points to it
	// but the resource may not exist
	ThumbnailPath    string
	ThumbnailResPath string
	ImageFormat      string

	// used internally for conversion of non dungeondraft supported image formats
	Image    image.Image
	PngImage []byte
}

var idTrimPrefixRegex = regexp.MustCompile(`^([\w\-. ]+)/`)

func (fi *FileInfo) CalcRelPath() string {
	var path string
	if fi.RelPath != "" {
		path = fi.RelPath
	} else {
		path = strings.TrimPrefix(fi.ResPath, "res://packs/")
		path = idTrimPrefixRegex.ReplaceAllString(path, "")
	}
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")
	}
	return path
}

func (fi *FileInfo) IsMetadata() bool {
	return !fi.IsData() && !fi.IsTexture()
}

func (fi *FileInfo) ShouldHaveMetadata() bool {
	return fi.IsWall() || fi.IsTileset()
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

func (fi *FileInfo) IsCave() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/caves/")
}

func (fi *FileInfo) IsLight() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/lights/")
}

func (fi *FileInfo) IsMaterial() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/materials/")
}

func (fi *FileInfo) IsObject() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/paths/")
}

func (fi *FileInfo) IsPath() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/paths/")
}

func (fi *FileInfo) IsPattern() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/patterns/")
}

func (fi *FileInfo) IsPortal() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/portals/")
}

func (fi *FileInfo) IsRoof() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/roofs/")
}

func (fi *FileInfo) IsTerrain() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/terrain/")
}

func (fi *FileInfo) IsTileset() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/tilesets/")
}

func (fi *FileInfo) IsWall() bool {
	return strings.HasPrefix(fi.CalcRelPath(), "textures/walls/")
}

func (fi *FileInfo) IsTaggable() bool {
	return fi.IsObject()
}

type FileInfoList []*FileInfo

func (fil FileInfoList) ToInfoPairList() *FileInfoPairList {
	return NewFileInfoPairList(fil)
}

func (fil FileInfoList) Write(log log.FieldLogger, out io.WriteSeeker, offset int64, alignment int, progressCallbacks ...func(p float64)) (err error) {
	fipl := fil.ToInfoPairList()
	return fipl.Write(log, out, offset, alignment, progressCallbacks...)
}

func (fil FileInfoList) AsSlice() []*FileInfo {
	return fil
}

func (fil FileInfoList) GetRessource(path string) *FileInfo {
	for _, fi := range fil {
		if fi.ResPath == path {
			return fi
		}
	}
	return nil
}

func (fil FileInfoList) Find(P func(info *FileInfo) bool) *FileInfo {
	for _, fi := range fil {
		if P(fi) {
			return fi
		}
	}
	return nil
}

func (fil FileInfoList) Filter(P func(info *FileInfo) bool) FileInfoList {
	res := []*FileInfo{}
	for _, fi := range fil {
		if P(fi) {
			res = append(res, fi)
		}
	}
	return res
}

func (fil FileInfoList) Remove(i int) *FileInfo {
	res := fil[i]
	fil[i] = fil[len(fil)-1]
	fil = fil[:len(fil)-1]
	return res
}

func (fil FileInfoList) IndexOf(info *FileInfo) int {
	for i, fi := range fil {
		if fi == info {
			return i
		}
	}
	return -1
}

func (fil FileInfoList) IndexOfRes(res string) int {
	for i, fi := range fil {
		if fi.ResPath == res {
			return i
		}
	}
	return -1
}

func (fil FileInfoList) RemoveRes(res string) *FileInfo {
	index := fil.IndexOfRes(res)
	if index != -1 {
		return fil.Remove(index)
	}
	return nil
}

var replaces = regexp.MustCompile(`(\.)|(\*\*/)|(\*\*$)|(\*)|(\[)|(\])|(\})|(\{)|(\+)|(\()|(\))|([^/\*])`)

func GlobToRelPathRegexp(pattern string) (*regexp.Regexp, error) {
	pat := replaces.ReplaceAllStringFunc(pattern, func(s string) string {
		switch s {
		case ".":
			return "\\."
		case "**":
			fallthrough
		case "**/":
			return ".*"
		case "*":
			return "[^/]*"
		case "[":
			return "\\["
		case "]":
			return "\\]"
		case "{":
			return "\\{"
		case "}":
			return "\\}"
		case "(":
			return "\\)"
		case ")":
			return "\\)"
		case "+":
			return "\\+"
		default:
			if s == "\\" && runtime.GOOS == "windows" {
				return "/"
			}
			return s
		}
	})
	return regexp.Compile("^" + pat + "$")
}

var ErrBadFileInfoListGlobPattern = errors.New("could not compile glob pattern")

type FileInfoFilterFunc func(*FileInfo) bool

func (fil FileInfoList) Glob(filter FileInfoFilterFunc, patterns ...string) (FileInfoList, error) {
	matches := []*FileInfo{}

	for _, pattern := range patterns {
		regexpPat, err := GlobToRelPathRegexp(pattern)
		if err != nil {
			return nil, errors.Join(err, ErrBadFileInfoListGlobPattern)
		}
		log.Debugf("compiled glob pattern %s", regexpPat.String())

		for _, info := range fil {
			if filter != nil && !filter(info) {
				continue
			}
			relPath := info.CalcRelPath()
			if regexpPat.MatchString(relPath) {
				matches = append(matches, info)
			}
		}
	}

	return matches, nil
}

func (fil FileInfoList) Paths() (paths []string) {
	for _, info := range fil {
		paths = append(paths, info.Path)
	}
	return
}

func (fil FileInfoList) ResPaths() (paths []string) {
	for _, info := range fil {
		paths = append(paths, info.ResPath)
	}
	return
}

func (fil FileInfoList) RelPaths() (paths []string) {
	for _, info := range fil {
		paths = append(paths, info.CalcRelPath())
	}
	return
}

// FileInfoPair groups a FileInfo and iot's Bytes equivalent
type FileInfoPair struct {
	Info      *FileInfo
	InfoBytes FileInfoBytes
}

// FileInfoPairList used to calculate the size of the list and properly set offsets in the info
type FileInfoPairList struct {
	FileList []*FileInfoPair
	Size     int64
}

// NewFileInfoPairList builds a valid FileInfoList with size information
func NewFileInfoPairList(fileList []*FileInfo) *FileInfoPairList {
	pairList := &FileInfoPairList{}

	var totalSize int64

	for _, fInfo := range fileList {
		fInfoBytes := FileInfoBytes{}

		fInfoBytes.Size = uint64(fInfo.Size)
		fInfoBytes.Offset = uint64(fInfo.Offset)

		fInfo.ResPathSize = int32(binary.Size([]byte(fInfo.ResPath)))
		totalSize += int64(binary.Size(fInfo.ResPathSize))
		totalSize += int64(fInfo.ResPathSize)
		totalSize += int64(binary.Size(fInfoBytes))

		pairList.FileList = append(pairList.FileList, &FileInfoPair{
			Info:      fInfo,
			InfoBytes: fInfoBytes,
		})
	}

	pairList.Size = totalSize

	return pairList
}

// UpdateOffsets updates all offset information to start from the passed point
// Gogot has the ability to control alignment of packed file data.
// this function tries to handle this
func (fipl *FileInfoPairList) UpdateOffsets(offset int64, alignment int) {
	for _, pair := range fipl.FileList {
		offset = utils.Align(offset, alignment)
		pair.Info.Offset = offset
		pair.InfoBytes.Offset = uint64(offset)

		offset += pair.Info.Size
	}
}

// Write out headers and file contents to io
func (fipl *FileInfoPairList) Write(log log.FieldLogger, out io.WriteSeeker, offset int64, alignment int, progressCallbacks ...func(p float64)) (err error) {
	log.Debug("updating offsets...")
	fipl.UpdateOffsets(fipl.Size+offset, alignment)

	log.Debug("writing files...")
	err = fipl.WriteHeaders(log, out, alignment)
	if err != nil {
		return
	}

	err = fipl.WriteFiles(log, out, alignment, progressCallbacks...)

	return
}

// WriteHeaders write out the headers to io
func (fipl *FileInfoPairList) WriteHeaders(log log.FieldLogger, out io.WriteSeeker, alignment int) error {
	log.Debug("writing file headers")
	for _, pair := range fipl.FileList {
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

var ErrAlignment = errors.New("alignment error")

// WriteFiles write the contents of the files in the list to io
func (fipl *FileInfoPairList) WriteFiles(log log.FieldLogger, out io.WriteSeeker, alignment int, progressCallbacks ...func(p float64)) error {
	log.Debug("writing file data")
	for i, pair := range fipl.FileList {

		curPos, err := utils.Tell(out)
		if err != nil {
			return err
		}
		if pair.Info.Offset != curPos {
			err = errors.Join(ErrAlignment, fmt.Errorf("%v != %v", curPos, pair.Info.Offset))
			log.WithError(err).WithField("file", pair.Info.Path).Error("misaligment of writer")
			return err
		}

		err = fipl.writeFile(log.WithField("file", pair.Info.Path), out, pair.Info)
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
			pcb(float64(i+1) / float64(len(fipl.FileList)))
		}
	}

	return nil
}

func (fipl *FileInfoPairList) writeFile(l log.FieldLogger, out io.Writer, info *FileInfo) (err error) {
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
