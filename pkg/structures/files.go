package structures

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"image"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"

	dderrors "github.com/ryex/dungeondraft-gopackager/internal/errors"

	log "github.com/sirupsen/logrus"
)

type Md5Hash struct {
	Md5 [16]byte
}

func (m *Md5Hash) String() string {
	return hex.EncodeToString(m.Md5[:])
}

func (m *Md5Hash) IsValid() bool {
	encoded := m.String()
	return encoded != "00000000000000000000000000000000" && encoded != ""
}

func (m *Md5Hash) UpdateWith(data []byte) {
	copy(m.Md5[:], data)
}

var ErrHashLenIncorrect = errors.New("length of hex encoded hash incorrect, != 16")

func DecodeMd5HashFromString(str string) (Md5Hash, error) {
	h := Md5Hash{}
	if hex.DecodedLen(len(str)) != 16 {
		return h, ErrHashLenIncorrect
	}
	decoded, err := hex.DecodeString(str)
	copy(h.Md5[:], decoded)
	return h, err
}

// FileInfoBytes is a struct used for readign and writing the encoded file information bytes
type FileInfoBytes struct {
	Offset uint64
	Size   uint64
	Md5    [16]byte
}

// Write out binary bytes to io
func (fi *FileInfoBytes) Write(out io.Writer) error {
	return binary.Write(out, binary.LittleEndian, fi)
}

// SizeOf the headers in bytes
func (fi *FileInfoBytes) SizeOf() int64 {
	return int64(binary.Size(fi))
}

// FileInfo stores file information
type FileInfo struct {
	Path string

	Size        int64
	Md5         Md5Hash
	ResPath     string
	ResPathSize int32
	RelPath     string

	// used whenreading and writing files
	Offset       int64
	HeaderOffset int64

	// Package
	Package  *PackageInfo

	// if the file should have metadata this resource path points to that metadata
	// but that resource may not exist
	MetadataPath string
	// if the file could have a thumbnail this resource path points to it
	// but the resource may not exist
	ThumbnailPath    string
	ThumbnailResPath string
	ImageFormat      string
	ThubnailFor      string

	// used internally for conversion of non dungeondraft supported image formats
	Image    image.Image
	PngImage []byte
}

// var idTrimPrefixRegex = regexp.MustCompile(`^([\w\-. ]+)/`)

func (fi *FileInfo) CalcRelPath() string {
	var path string
	if fi.RelPath != "" {
		path = fi.RelPath
	} else {
		path = utils.CleanRelativeResourcePath(fi.ResPath)
		// path = strings.TrimPrefix(fi.ResPath, "res://packs/")
		// path = idTrimPrefixRegex.ReplaceAllString(path, "")
	}
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")
	}
	return path
}

func (fi *FileInfo) IsMetadata() bool {
	return !fi.IsData() && !fi.IsTexture() && !fi.IsThumbnail()
}

func (fi *FileInfo) ShouldHaveMetadata() bool {
	return (fi.IsWall() &&
		!strings.HasSuffix(
			strings.TrimSuffix(filepath.Base(fi.ResPath), filepath.Ext(fi.ResPath)),
			"_end",
		)) || fi.IsTileset()
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
	return strings.HasPrefix(fi.CalcRelPath(), "textures/objects/")
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

func (fi *FileInfo) ToInfoBytes() FileInfoBytes {
	infoBytes := FileInfoBytes{
		Offset: uint64(fi.Offset),
		Size: uint64(fi.Size),
	}
	copy(infoBytes.Md5[:], fi.Md5.Md5[:])
	return infoBytes
}

type FileInfoList struct {
	data []*FileInfo
}

func NewFileInfoList() *FileInfoList {
	fil := &FileInfoList{ make([]*FileInfo, 0) }
	return fil
}

func (fil *FileInfoList) Clear() {
	fil.data = nil
}

func (fil *FileInfoList) CopyTo(other *FileInfoList) {
	other.data = make([]*FileInfo, len(fil.data))
	copy(other.data, fil.data)
}

func (fil *FileInfoList) Length() int {
	return len(fil.data)
}

func (fil *FileInfoList) IsEmpty() bool {
	return len(fil.data) == 0
}

func (fil *FileInfoList) Copy() *FileInfoList {
	other := NewFileInfoList()
	fil.CopyTo(other)
	return other
}

func (fil *FileInfoList) AsSlice() []*FileInfo {
	return fil.data
}

func (fil *FileInfoList) GetRessource(path string) *FileInfo {
	for _, fi := range fil.data {
		if fi.ResPath == path {
			return fi
		}
	}
	return nil
}

func (fil *FileInfoList) Find(P func(fi *FileInfo) bool) *FileInfo {
	for _, fi := range fil.data {
		if P(fi) {
			return fi
		}
	}
	return nil
}

func (fil *FileInfoList) Filter(P func(fi *FileInfo) bool) *FileInfoList {
	res := []*FileInfo{}
	for _, fi := range fil.data {
		if P(fi) {
			res = append(res, fi)
		}
	}
	return &FileInfoList { data: res }
}

func (fil *FileInfoList) DeduplicateBy(P func(a, b *FileInfo) bool) {
	fil.data = slices.CompactFunc(fil.data, P)
}

func (fil *FileInfoList) Remove(i int) *FileInfo {
	res := fil.data[i]
	fil.data = slices.Delete(fil.data, i, i+1)
	return res
}

func (fil *FileInfoList) Add(info *FileInfo) {
	if index := fil.IndexOfRes(info.ResPath); index == -1 {
		fil.data = append(fil.data, info)
	}
}

func (fil *FileInfoList) Insert(i int, infos ...*FileInfo) {
	fil.data = slices.Insert(fil.data, i, infos...)
}

func (fil *FileInfoList) IndexOf(info *FileInfo) int {
	for i, fi := range fil.data {
		if fi == info {
			return i
		}
	}
	return -1
}

func (fil *FileInfoList) IndexOfRes(res string) int {
	for i, fi := range fil.data {
		if fi.ResPath == res {
			return i
		}
	}
	return -1
}

func (fil *FileInfoList) RemoveRes(res string) *FileInfo {
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

func (fil *FileInfoList) Glob(filter FileInfoFilterFunc, patterns ...string) (*FileInfoList, error) {
	matches := make(map[string]*FileInfo)

	for _, pattern := range patterns {
		regexpPat, err := GlobToRelPathRegexp(pattern)
		if err != nil {
			return &FileInfoList{ data: nil }, dderrors.CausedBy(ErrBadFileInfoListGlobPattern, err)
		}
		log.Debugf("compiled glob pattern %s", regexpPat.String())

		for _, info := range fil.data {
			if filter != nil && !filter(info) {
				continue
			}
			relPath := info.CalcRelPath()
			if regexpPat.MatchString(relPath) {
				matches[info.ResPath] = info
			}
		}
	}

	matchesS := make([]*FileInfo, len(matches))
	i := 0
	for _, fi := range matches {
		matchesS[i] = fi
		i++
	}

	return &FileInfoList{ data: matchesS}, nil
}

func (fil *FileInfoList) Paths() (paths []string) {
	for _, info := range fil.data {
		paths = append(paths, info.Path)
	}
	return
}

func (fil *FileInfoList) ResPaths() (paths []string) {
	for _, info := range fil.data {
		paths = append(paths, info.ResPath)
	}
	return
}

func (fil *FileInfoList) RelPaths() (paths []string) {
	for _, info := range fil.data {
		paths = append(paths, info.CalcRelPath())
	}
	return
}

func (fil *FileInfoList) SetCapacity(capacity int) {
	if capacity < len(fil.data) {
		capacity = len(fil.data)
	}
	if capacity > cap(fil.data) {
		sized := make([]*FileInfo, len(fil.data), capacity)
		copy(sized, fil.data)
		fil.data = sized
	}
}

func (fil *FileInfoList) UpdateThumbnailRefrences() {
	thumbnailMap := make(map[string]string)
	for _, fi := range fil.data {
		if fi.IsTexture() && fi.ThumbnailPath != "" {
			thumbnailMap[fi.ThumbnailResPath] = fi.ResPath
		}
	}
	toRemove := NewSet[string]()
	for _, fi := range fil.data {
		if fi.IsThumbnail() {
			forRes, ok := thumbnailMap[fi.ResPath]
			if ok {
				fi.ThubnailFor = forRes
			} else {
				toRemove.Add(fi.ResPath)
			}
		}
	}

	for _, res := range toRemove.AsSlice() {
		log.Warnf("removing thumbnail %s (does not have a linked texture)", filepath.Base(res))
		fil.RemoveRes(res)
	}
}

func cmpResPaths(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpResAndThumb(a, b *FileInfo) int {
	aIsThumb := a.IsThumbnail()
	bIsThumb := b.IsThumbnail()
	if (aIsThumb && bIsThumb) || (!aIsThumb && !bIsThumb) {
		return cmpResPaths(a.ResPath, b.ResPath)
	} else if aIsThumb && !bIsThumb {
		if a.ThubnailFor != "" {
			if a.ThubnailFor == b.ResPath {
				return -1
			}
			return cmpResPaths(a.ThubnailFor, b.ResPath)
		}
		return -1
	} else if !aIsThumb && bIsThumb {
		if b.ThubnailFor != "" {
			if a.ResPath == b.ThubnailFor {
				return 1
			}
			return cmpResPaths(a.ResPath, b.ThubnailFor)
		}
		return -1
	}
	return 0
}

// Sort the file list in place
func (fil *FileInfoList) Sort() {
	fil.UpdateThumbnailRefrences()

	slices.SortFunc(fil.data, func(a, b *FileInfo) int {
		return cmpResPaths(a.ResPath, b.ResPath)
	})
}

func (fil *FileInfoList) Write(
	log log.FieldLogger,
	out io.WriteSeeker,
	alignment int,
	progressCallback func(p float64),
) error {
	err := fil.WriteHeaders(log, out, alignment)
	if err != nil {
		return err
	}

	return fil.WriteFiles(log, out, alignment, progressCallback)
}

func (fil *FileInfoList) WriteHeaders(
	log log.FieldLogger,
	out io.WriteSeeker,
	alignment int,
) error {
	log.Debug("writing headers...")
	for _, fi := range fil.data {
		// write path length
		err := binary.Write(out, binary.LittleEndian, fi.ResPathSize)
		if !utils.CheckErrorWrite(log, err) {
			return err
		}

		// write filepath
		err = binary.Write(out, binary.LittleEndian, []byte(fi.ResPath))
		if !utils.CheckErrorWrite(log, err) {
			return err
		}

		curPos, err := utils.Tell(out)
		if err != nil {
			return err
		}
		fi.HeaderOffset = curPos

		fInfoBytes := fi.ToInfoBytes()

		// write fileinfo
		err = fInfoBytes.Write(out)
		if !utils.CheckErrorWrite(log, err) {
			return err
		}
	}

	return nil
}

func (fil *FileInfoList) WriteFiles(
	log log.FieldLogger,
	out io.WriteSeeker,
	alignment int,
	progressCallback func(p float64),
) error {
	// alignment
	curPos, err := utils.Tell(out)
	if err != nil {
		return err
	}
	offset := utils.Align(curPos, alignment)
	err = utils.Pad(out, offset-curPos)
	if err != nil {
		return err
	}

	for i, fi := range fil.data {

		{

			// collect file data
			var data []byte
			if fi.Image != nil && fi.PngImage != nil {
				log.Debug("using png image data")
				data = fi.PngImage
			} else {
				var err error
				data, err = os.ReadFile(fi.Path)
				if err != nil {
					log.WithError(err).Error("error reading file")
					return err
				}
			}
			// store the size of the data
			fi.Size = int64(len(data))
			// store reall offset
			fi.Offset = offset
			// update md5
			{
			  hash := md5.New()
				if _, err := hash.Write(data); err == nil {
					fi.Md5.UpdateWith(hash.Sum(nil))
				}
			}

			// write out the data
			n, err := out.Write(data)
			if !utils.CheckErrorWrite(log, err) {
				return err
			}
			if int64(n) != fi.Size {
				err = errors.New("write of wrong size")
				log.WithField("expectedWriteSize", fi.Size).
					WithField("writeSize", n).
					WithError(err).Error("failed to write file")
				return err
			}

			curPos, err = utils.Tell(out)
			if err != nil {
				return err
			}

			// go back to update the stored size, offset, and md5
			_, err = out.Seek(fi.HeaderOffset, io.SeekStart)
			if err != nil {
				return err
			}

			fInfoBytes := fi.ToInfoBytes()

			// write fileinfo
			err = fInfoBytes.Write(out)
			if !utils.CheckErrorWrite(log, err) {
				return err
			}

			// return to post file position
			_, err = out.Seek(curPos, io.SeekStart)
			if err != nil {
				return err
			}

		}

		// alignment
		curPos, err := utils.Tell(out)
		if err != nil {
			return err
		}

		offset = utils.Align(curPos, alignment)
		err = utils.Pad(out, offset-curPos)
		if err != nil {
			return err
		}

		if progressCallback != nil {
			progressCallback(float64(i+1) / float64(len(fil.data)))
		}
	}

	return nil
}
