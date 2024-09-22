package unpack

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

// Unpacker unpacks dungeondraft_pack files in the pck archive format
// set the RipTexture field if you with .tex files to be extracted to image formats
// set the Overwrite field if you wish overwrite operations to overwrite an existing file
type Unpacker struct {
	log         logrus.FieldLogger
	name        string
	id          string
	RipTextures bool
	Overwrite   bool
	IgnoreJson  bool
	Thumbnails  bool
	FileList    []structures.FileInfo
	Pack        structures.Package
}

// NewUnpacker builds a new Unpacker
func NewUnpacker(log logrus.FieldLogger) *Unpacker {
	return &Unpacker{
		log: log,
	}
}

// ExtractPackage extracts the package contents to the filesystem
func (u *Unpacker) ExtractPackage(r io.ReadSeeker, outDir string) (err error) {
	err = u.ReadPackageFilelist(r)
	if err != nil {
		return
	}

	err = u.ExtractFilelist(r, outDir)

	return
}

var (
	resourcePathRegex  = regexp.MustCompile(`^res://packs/([\w\-. ]+)((\.json$)|(/))`)
	thumbnailPathRegex = regexp.MustCompile(`^res://packs/([\w\-. ]+)((\.json$)|(/))`)
	packJsonPathRegex  = regexp.MustCompile(`^res://packs/([\w\-. ]+).json`)
)

func (u *Unpacker) NormalizeResourcePath(resPath string) string {
	path := strings.Replace(string(resPath), "res://", u.name+"/", 1)
	match := resourcePathRegex.FindStringSubmatch(resPath)
	if match != nil {
		guid := strings.TrimSpace(match[1])
		clean := filepath.Clean(path)
		path = filepath.Clean(strings.Replace(clean, filepath.Join("packs", guid)+string(filepath.Separator), "", 1))
		path = filepath.Clean(strings.Replace(path, filepath.Join("packs", guid)+".json", "pack.json", 1))
	}
	return path
}

func (u *Unpacker) MapResourcePaths() {
	for i := 0; i < len(u.FileList); i++ {
		packedFile := &u.FileList[i]
		packedFile.Path = u.NormalizeResourcePath(packedFile.ResPath)
	}
}

// ExtractFilelist takes a slice of FileInfo and extracts the files from the package at the reader
func (u *Unpacker) ExtractFilelist(r io.ReadSeeker, outDir string) (err error) {
	outDirPath, err := filepath.Abs(outDir)
	if err != nil {
		return
	}

	fileExists := utils.FileExists(outDirPath)
	if fileExists {
		err = errors.New("out folder already exists as a file")
		return
	}
	dirExists := utils.DirExists(outDirPath)
	if !dirExists {
		err = os.MkdirAll(outDirPath, 0777)
		if err != nil {
			return
		}
	}

	valid, err := u.isValidPackage(r)

	if !valid {
		err = errors.New("not a valid package")
		return
	}

	err = u.ReadPackJson(r)
	if err != nil {
		return
	}

	u.MapResourcePaths()

	thumbnailPrefix := fmt.Sprintf("res://packs/%s/thumbnails/", u.id)

	extractedPaths := make(map[string]string)

	for i := 0; i < len(u.FileList); i++ {
		packedFile := &u.FileList[i]

		if strings.HasPrefix(packedFile.ResPath, thumbnailPrefix) && ! u.Thumbnails {
			continue
		}

		if resPath, ok := extractedPaths[packedFile.Path]; ok {
			u.log.
				WithField("packedPath", packedFile.ResPath).
				WithField("duplicateResPath", resPath == packedFile.ResPath).
				Warnf("ignoring previously extracted path %s", packedFile.Path)
			continue
		}

		path := filepath.Join(outDirPath, filepath.Dir(packedFile.Path))
		u.log.WithField("mappedPath", packedFile.Path).Debugf("%s -> %s", packedFile.ResPath, path)

		fileNameFull := filepath.Base(packedFile.ResPath)
		fileExt := filepath.Ext(fileNameFull)

		if fileExt == ".tex" && !u.RipTextures {
			continue
		}

		l := u.log.
			WithField("packedPath", packedFile.ResPath).
			WithField("offset", packedFile.Offset)

		err = os.MkdirAll(path, 0777)
		if err != nil {
			l.WithField("unpackedFile", path).WithError(err).
				Error("can not make target directory")
			return err
		}

		if _, err = u.ExtractFile(r, *packedFile, path); err != nil {
			return err
		}
		extractedPaths[packedFile.Path] = packedFile.ResPath
	}

	u.log.Info("unpacking complete")

	return
}

func (u *Unpacker) ExtractFile(r io.ReadSeeker, packedFile structures.FileInfo, outPath string) (string, error) {
	l := u.log.
		WithField("packedPath", packedFile.ResPath).
		WithField("offset", packedFile.Offset)

	fileNameFull := filepath.Base(packedFile.Path)
	fileExt := filepath.Ext(fileNameFull)
	fileName := strings.TrimSuffix(fileNameFull, fileExt)

	fileData, err := u.ReadFileFromPackage(r, packedFile)
	if err != nil {
		return "", err
	}

	if fileExt == ".tex" && u.RipTextures {
		var ext string
		var data []byte
		ext, data, err = utils.RipTexture(fileData)
		if err == nil {
			fileExt = ext
			fileData = data
			fileNameFull = fileName + ext
		}
	}

	filePath := filepath.Join(outPath, fileNameFull)
	l = l.WithField("unpackedFile", filePath)

	l.Info("writing file")

	fileExists := utils.FileExists(filePath)
	if fileExists {
		if u.Overwrite {
			l.Warn("overwriting file")
		} else {
			err = errors.New("file exists")
			l.WithError(err).Error("file already exists at destination and Overwrite not enabled")
			return "", err
		}
	}

	var p *os.File
	p, err = os.Create(filePath)
	if err != nil {
		l.WithError(err).Error("can not open file for writing")
		return "", err
	}
	_, err = p.Write(fileData)
	if err != nil {
		l.WithError(err).Error("failed to write file")
		return "", err
	}

	err = p.Close()
	if err != nil {
		l.WithError(err).Error("failed to close file")
		return "", err
	}
	return filePath, nil
}

func (u *Unpacker) ReadFileFromPackage(r io.ReadSeeker, packedFile structures.FileInfo) ([]byte, error) {
	l := u.log.
		WithField("packedPath", packedFile.ResPath).
		WithField("offset", packedFile.Offset)

	_, err := r.Seek(packedFile.Offset, io.SeekStart)
	if err != nil {
		l.WithError(err).
			Error("can not seek to offset for file data")
		return nil, err
	}

	fileData := make([]byte, packedFile.Size)
	var n int
	n, err = r.Read(fileData)
	if !utils.CheckErrorRead(l, err, n, int(packedFile.Size)) {
		return nil, err
	}

	// if the md5 isn't blank verify
	if packedFile.Md5 != "00000000000000000000000000000000" {
		hash := md5.Sum(fileData)
		md5Hash := hex.EncodeToString(hash[:])
		if packedFile.Md5 != md5Hash {
			err = errors.New("md5 hash mismatch")
			l.WithError(err).
				WithField("packedDataMd5", md5Hash).
				WithField("expectedMd5", packedFile.Md5).
				Error("hash verification failed")
			return nil, err
		}
	}

	return fileData, nil
}

// ReadPackageFilelist Takes an io.reader and attempts to extract a list of files stored in the package
func (u *Unpacker) ReadPackageFilelist(r io.ReadSeeker) (err error) {
	valid, err := u.isValidPackage(r)

	if !valid {
		err = errors.New("not a valid package")
		return
	}
	// reader is in position, start extracting

	err = u.getFileList(r)
	if err != nil {
		u.log.WithError(err).
			Error("could not read file list")
		return
	}

	return
}

func (u *Unpacker) ReadPackJson(r io.ReadSeeker) (err error) {
	if u.FileList == nil {
		return errors.New("empty file list")
	}
	var packJsonInfo *structures.FileInfo
	for _, packedFile := range u.FileList {
		match := packJsonPathRegex.MatchString(packedFile.ResPath)
		if match {
			packJsonInfo = &packedFile
			break
		}
	}

	if packJsonInfo == nil {
		return errors.New("can't find pack json in package file list")
	}

	packJSONBytes, err := u.ReadFileFromPackage(r, *packJsonInfo)
	if err != nil {
		u.log.WithError(err).WithField("res", packJsonInfo.ResPath).Error("failed to read pack json")
		return errors.Join(err, errors.New("failed to read pack json"))
	}

	err = json.Unmarshal(packJSONBytes, &u.Pack)
	if err != nil {
		u.log.WithError(err).WithField("res", packJsonInfo.ResPath).Error("failed to parse pack json")
		return errors.Join(err, errors.New("failed to parse pack json"))
	}

	u.id = u.Pack.ID
	u.name = u.Pack.Name

	return nil
}

func (u *Unpacker) checkedRead(r io.Reader, data any) error {
	err := binary.Read(r, binary.LittleEndian, data)
	if err != nil {
		dataType := reflect.TypeOf(data)
		u.log.WithError(err).
			WithField("DataType", dataType.Name()).
			WithField("Size", dataType.Size()).
			Error("unpack read error")
	}
	return err
}

func (u *Unpacker) checkedSeek(r io.Seeker, offset int64, whence int) (int64, error) {
	curPos, err := r.Seek(offset, whence)
	if err != nil {
		u.log.WithError(err).
			WithField("currentOffset", curPos).
			Error("unpack seek error")
	}
	return curPos, err
}

func (u *Unpacker) tell(r io.Seeker) (int64, error) {
	curPos, err := r.Seek(0, io.SeekCurrent) // tell
	if err != nil {
		u.log.WithError(err).
			WithField("currentOffset", curPos).
			Error("unpack seek error")
	}
	return curPos, err
}

func (u *Unpacker) isValidPackage(r io.ReadSeeker) (bool, error) {
	// back to start
	if _, err := u.checkedSeek(r, 0, io.SeekStart); err != nil {
		return false, err
	}

	// find our magic to figure out where to start reading

	var magic uint32
	if err := u.checkedRead(r, &magic); err != nil {
		return false, err
	}

	if magic == structures.GODOT_PACKAGE_MAGIC {
		u.log.Debugf("looks like a pck archive")
	} else {
		u.log.
			WithField("magic", magic).
			WithField("expectedMagic", structures.GODOT_PACKAGE_MAGIC).
			Debug("Failed to read GDPC pck Magic")

		if _, err := u.checkedSeek(r, -4, io.SeekEnd); err != nil {
			return false, err
		}

		// attempt to read the GDPC Magic from the end of the file
		if err := u.checkedRead(r, &magic); err != nil {
			return false, err
		}

		if magic == structures.GODOT_PACKAGE_MAGIC {
			u.log.Debug("looks like a self-contained exe", u.name)

			// 12 bytes from end
			if _, err := u.checkedSeek(r, -12, io.SeekEnd); err != nil {
				return false, err
			}

			var mainOffset int64
			if err := u.checkedRead(r, &mainOffset); err != nil {
				u.log.
					WithError(err).Error("Could not read main offset of data in self-contained exe")
				return false, err
			}

			curPos, err := u.tell(r)
			if err != nil {
				return false, err
			}

			if _, err := u.checkedSeek(r, curPos-mainOffset-8, io.SeekStart); err != nil {
				return false, err
			}

			// attempt to read the GDPC Magic at offset
			if err := u.checkedRead(r, &magic); err != nil {
				return false, err
			}

			if magic != structures.GODOT_PACKAGE_MAGIC {
				u.log.
					WithField("magic", magic).
					WithField("expectedMagic", structures.GODOT_PACKAGE_MAGIC).
					Error("Failed to read GDPC self-contained exe Magic at main offset")
				return false, nil
			}

		} else {
			u.log.
				WithField("magic", magic).
				WithField("expectedMagic", structures.GODOT_PACKAGE_MAGIC).
				Error("Failed to read GDPC self-contained exe Magic")
			return false, nil
		}
	}

	// seek before magic
	if _, err := u.checkedSeek(r, -4, io.SeekCurrent); err != nil {
		return false, err
	}

	return true, nil
}

func (u *Unpacker) ReadPackageHeaders(r io.ReadSeeker) (headers structures.PackageHeaders, err error) {
	if err = u.checkedRead(r, &headers); err != nil {
		u.log.WithError(err).Error("Could not read package headers")
	}
	if headers.PackFormatVersion != structures.GODOT_PACKAGE_FORMAT {
		err = errors.New("unsupported Pack version")
		u.log.
			WithError(err).
			WithField("PackFormat", headers.PackFormatVersion).
			WithField("SupportedPackFormat", structures.GODOT_PACKAGE_FORMAT).
			Error("Pack version unsupported")
	} else if headers.VersionMajor > structures.GODOT_MAJOR ||
		(headers.VersionMajor == structures.GODOT_MAJOR &&
			headers.VersionMinor > structures.GODOT_MINOR) {

		err = errors.New("unsupported GoDot engine version")
		u.log.
			WithError(err).
			WithField("GoDotMajor", headers.VersionMajor).
			WithField("GoDotMinor", headers.VersionMinor).
			WithField("SupportedGoDotMajor", structures.GODOT_MAJOR).
			WithField("SupportedGoDotMinor", structures.GODOT_MINOR).
			Error("Package build with a newer GoDot Engine")
	}
	return
}

func (u *Unpacker) getFileList(r io.ReadSeeker) (err error) {
	headers, err := u.ReadPackageHeaders(r)
	if err != nil {
		return
	}
	u.log.WithField("headers", headers).
		Debug("info")

	fileCount := headers.FileCount

	for fileNum := uint32(1); fileNum <= fileCount; fileNum++ {
		var filePathLength int32
		err = binary.Read(r, binary.LittleEndian, &filePathLength)
		if err != nil {
			u.log.WithError(err).
				WithField("FileNum", fileNum).Error("could not read file path length")
			return
		}

		var infoBytes structures.FileInfoBytes
		pathBytes := make([]byte, filePathLength)
		err = binary.Read(r, binary.LittleEndian, &pathBytes)
		if err != nil {
			u.log.WithError(err).
				WithField("filePathLength", filePathLength).
				WithField("FileNum", fileNum).Error("could not read file path")
			return
		}

		err = binary.Read(r, binary.LittleEndian, &infoBytes)
		if err != nil {
			u.log.
				WithError(err).
				WithField("FileNum", fileNum).Error("could not read file info")
			return
		}

		info := structures.FileInfo{
			ResPath:     string(pathBytes),
			ResPathSize: filePathLength,
			Offset:      int64(infoBytes.Offset),
			Size:        int64(infoBytes.Size),
			Md5:         hex.EncodeToString(infoBytes.Md5[:]),
		}

		u.log.
			// WithField("infoBytes", infoBytes).
			// WithField("FileNum", fileCount).
			// WithField("FileCount", fileNum).
			WithField("info", info).
			Infof("found file [%v/%v]", fileNum, fileCount)
		u.FileList = append(u.FileList, info)

	}

	return
}
