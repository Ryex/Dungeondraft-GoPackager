package ddpackage

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
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

// ExtractPackage extracts the package contents to the filesystem
func (p *Package) ExtractPackage(r io.ReadSeeker, outDir string, options UnpackOptions) (err error) {
	p.SetUnpackOptions(options)
	p.unpackedPath = outDir
	err = p.ReadPackageFilelist(r)
	if err != nil {
		return
	}

	err = p.ExtractFilelist(r, outDir)

	return
}

var (
	resourcePathRegex  = regexp.MustCompile(`^res://packs/([\w\-. ]+)((\.json$)|(/))`)
	thumbnailPathRegex = regexp.MustCompile(`^res://packs/([\w\-. ]+)((\.json$)|(/))`)
	packJsonPathRegex  = regexp.MustCompile(`^res://packs/([\w\-. ]+).json`)
)

func (p *Package) NormalizeResourcePath(resPath string) string {
	path := strings.Replace(string(resPath), "res://", p.name+"/", 1)
	match := resourcePathRegex.FindStringSubmatch(resPath)
	if match != nil {
		guid := strings.TrimSpace(match[1])
		clean := filepath.Clean(path)
		path = filepath.Clean(strings.Replace(clean, filepath.Join("packs", guid)+string(filepath.Separator), "", 1))
		path = filepath.Clean(strings.Replace(path, filepath.Join("packs", guid)+".json", "pack.json", 1))
	}
	return path
}

func (p *Package) MapResourcePaths() {
	for i := 0; i < len(p.FileList); i++ {
		packedFile := &p.FileList[i]
		packedFile.Path = p.NormalizeResourcePath(packedFile.ResPath)
	}
}

// ExtractFilelist takes a slice of FileInfo and extracts the files from the package at the reader
func (p *Package) ExtractFilelist(r io.ReadSeeker, outDir string) (err error) {
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

	valid, err := p.isValidPackage(r)

	if !valid {
		err = errors.New("not a valid package")
		return
	}

	err = p.ReadPackedPackJson(r)
	if err != nil {
		return
	}

	p.MapResourcePaths()

	thumbnailPrefix := fmt.Sprintf("res://packs/%s/thumbnails/", p.id)

	extractedPaths := make(map[string]string)

	for i := 0; i < len(p.FileList); i++ {
		packedFile := &p.FileList[i]

		if strings.HasPrefix(packedFile.ResPath, thumbnailPrefix) && !p.unpackOptions.Thumbnails {
			continue
		}

		if resPath, ok := extractedPaths[packedFile.Path]; ok {
			p.log.
				WithField("packedPath", packedFile.ResPath).
				WithField("duplicateResPath", resPath == packedFile.ResPath).
				Warnf("ignoring previously extracted path %s", packedFile.Path)
			continue
		}

		path := filepath.Join(outDirPath, filepath.Dir(packedFile.Path))
		p.log.WithField("mappedPath", packedFile.Path).Debugf("%s -> %s", packedFile.ResPath, path)

		fileNameFull := filepath.Base(packedFile.ResPath)
		fileExt := filepath.Ext(fileNameFull)

		if fileExt == ".tex" && !p.unpackOptions.RipTextures {
			continue
		} 

		l := p.log.
			WithField("packedPath", packedFile.ResPath).
			WithField("offset", packedFile.Offset)

		err = os.MkdirAll(path, 0777)
		if err != nil {
			l.WithField("unpackedFile", path).WithError(err).
				Error("can not make target directory")
			return err
		}

		if _, err = p.ExtractFile(r, *packedFile, path); err != nil {
			return err
		}
		extractedPaths[packedFile.Path] = packedFile.ResPath
	}

	p.log.Info("unpacking complete")

	return
}

func (p *Package) ExtractFile(r io.ReadSeeker, packedFile structures.FileInfo, outPath string) (string, error) {
	l := p.log.
		WithField("packedPath", packedFile.ResPath).
		WithField("offset", packedFile.Offset)

	fileNameFull := filepath.Base(packedFile.Path)
	fileExt := filepath.Ext(fileNameFull)
	fileName := strings.TrimSuffix(fileNameFull, fileExt)

	fileData, err := p.ReadFileFromPackage(r, packedFile)
	if err != nil {
		return "", err
	}

	if fileExt == ".tex" && p.unpackOptions.RipTextures {
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
		if p.unpackOptions.Overwrite {
			l.Warn("overwriting file")
		} else {
			err = errors.New("file exists")
			l.WithError(err).Error("file already exists at destination and Overwrite not enabled")
			return "", err
		}
	}

	var f *os.File
	f, err = os.Create(filePath)
	if err != nil {
		l.WithError(err).Error("can not open file for writing")
		return "", err
	}
	_, err = f.Write(fileData)
	if err != nil {
		l.WithError(err).Error("failed to write file")
		return "", err
	}

	err = f.Close()
	if err != nil {
		l.WithError(err).Error("failed to close file")
		return "", err
	}
	return filePath, nil
}

func (p *Package) ReadFileFromPackage(r io.ReadSeeker, packedFile structures.FileInfo) ([]byte, error) {
	l := p.log.
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
func (p *Package) ReadPackageFilelist(r io.ReadSeeker) (err error) {
	valid, err := p.isValidPackage(r)

	if !valid {
		err = errors.New("not a valid package")
		return
	}
	// reader is in position, start extracting

	err = p.getFileList(r)
	if err != nil {
		p.log.WithError(err).
			Error("could not read file list")
		return
	}

	return
}

func (p *Package) ReadPackedPackJson(r io.ReadSeeker) (err error) {
	if p.FileList == nil {
		return errors.New("empty file list")
	}
	var packJsonInfo *structures.FileInfo
	for i := 0; i < len(p.FileList); i++ {
		packedFile := &p.FileList[i]
		match := packJsonPathRegex.MatchString(packedFile.ResPath)
		if match {
			packJsonInfo = packedFile
			break
		}
	}

	if packJsonInfo == nil {
		return errors.New("can't find pack json in package file list")
	}

	packJSONBytes, err := p.ReadFileFromPackage(r, *packJsonInfo)
	if err != nil {
		p.log.WithError(err).WithField("res", packJsonInfo.ResPath).Error("failed to read pack json")
		return errors.Join(err, errors.New("failed to read pack json"))
	}

	err = json.Unmarshal(packJSONBytes, &p.Info)
	if err != nil {
		p.log.WithError(err).WithField("res", packJsonInfo.ResPath).Error("failed to parse pack json")
		return errors.Join(err, errors.New("failed to parse pack json"))
	}

	p.id = p.Info.ID
	p.name = p.Info.Name

	return nil
}

func (p *Package) checkedRead(r io.Reader, data any) error {
	err := binary.Read(r, binary.LittleEndian, data)
	if err != nil {
		dataType := reflect.TypeOf(data)
		p.log.WithError(err).
			WithField("DataType", dataType.Name()).
			WithField("Size", dataType.Size()).
			Error("unpack read error")
	}
	return err
}

func (p *Package) checkedSeek(r io.Seeker, offset int64, whence int) (int64, error) {
	curPos, err := r.Seek(offset, whence)
	if err != nil {
		p.log.WithError(err).
			WithField("currentOffset", curPos).
			Error("unpack seek error")
	}
	return curPos, err
}

func (p *Package) isValidPackage(r io.ReadSeeker) (bool, error) {
	// back to start
	if _, err := p.checkedSeek(r, 0, io.SeekStart); err != nil {
		return false, err
	}

	// find our magic to figure out where to start reading

	var magic uint32
	if err := p.checkedRead(r, &magic); err != nil {
		return false, err
	}

	if magic == structures.GODOT_PACKAGE_MAGIC {
		p.log.Debugf("looks like a pck archive")
	} else {
		p.log.
			WithField("magic", magic).
			WithField("expectedMagic", structures.GODOT_PACKAGE_MAGIC).
			Debug("Failed to read GDPC pck Magic")

		if _, err := p.checkedSeek(r, -4, io.SeekEnd); err != nil {
			return false, err
		}

		// attempt to read the GDPC Magic from the end of the file
		if err := p.checkedRead(r, &magic); err != nil {
			return false, err
		}

		if magic == structures.GODOT_PACKAGE_MAGIC {
			p.log.Debug("looks like a self-contained exe", p.name)

			// 12 bytes from end
			if _, err := p.checkedSeek(r, -12, io.SeekEnd); err != nil {
				return false, err
			}

			var mainOffset int64
			if err := p.checkedRead(r, &mainOffset); err != nil {
				p.log.
					WithError(err).Error("Could not read main offset of data in self-contained exe")
				return false, err
			}

			curPos, err := utils.Tell(r)
			if err != nil {
				return false, err
			}

			if _, err := p.checkedSeek(r, curPos-mainOffset-8, io.SeekStart); err != nil {
				return false, err
			}

			// attempt to read the GDPC Magic at offset
			if err := p.checkedRead(r, &magic); err != nil {
				return false, err
			}

			if magic != structures.GODOT_PACKAGE_MAGIC {
				p.log.
					WithField("magic", magic).
					WithField("expectedMagic", structures.GODOT_PACKAGE_MAGIC).
					Error("Failed to read GDPC self-contained exe Magic at main offset")
				return false, nil
			}

		} else {
			p.log.
				WithField("magic", magic).
				WithField("expectedMagic", structures.GODOT_PACKAGE_MAGIC).
				Error("Failed to read GDPC self-contained exe Magic")
			return false, nil
		}
	}

	// seek before magic
	if _, err := p.checkedSeek(r, -4, io.SeekCurrent); err != nil {
		return false, err
	}

	return true, nil
}

func (p *Package) ReadPackageHeaders(r io.ReadSeeker) (headers structures.PackageHeaders, err error) {
	if err = p.checkedRead(r, &headers); err != nil {
		p.log.WithError(err).Error("Could not read package headers")
	}
	if headers.PackFormatVersion != structures.GODOT_PACKAGE_FORMAT {
		err = errors.New("unsupported Pack version")
		p.log.
			WithError(err).
			WithField("PackFormat", headers.PackFormatVersion).
			WithField("SupportedPackFormat", structures.GODOT_PACKAGE_FORMAT).
			Error("Pack version unsupported")
	} else if headers.VersionMajor > structures.GODOT_MAJOR ||
		(headers.VersionMajor == structures.GODOT_MAJOR &&
			headers.VersionMinor > structures.GODOT_MINOR) {

		err = errors.New("unsupported GoDot engine version")
		p.log.
			WithError(err).
			WithField("GoDotMajor", headers.VersionMajor).
			WithField("GoDotMinor", headers.VersionMinor).
			WithField("SupportedGoDotMajor", structures.GODOT_MAJOR).
			WithField("SupportedGoDotMinor", structures.GODOT_MINOR).
			Error("Package build with a newer GoDot Engine")
	}
	return
}

func (p *Package) newFileInfo(resPath []byte, infoBytes structures.FileInfoBytes) structures.FileInfo {
	info := structures.FileInfo{
		ResPath:     string(resPath),
		ResPathSize: int32(len(resPath)),
		Offset:      int64(infoBytes.Offset),
		Size:        int64(infoBytes.Size),
		Md5:         hex.EncodeToString(infoBytes.Md5[:]),
	}

	if ddimage.PathIsSupportedImage(strings.TrimPrefix(info.ResPath, "res://")) {
		hash := md5.Sum([]byte(resPath))
		thumbnailName := hex.EncodeToString(hash[:]) + ".png"
		info.ThumbnailResPath = fmt.Sprintf("res://packs/%s/thumbnails/%s", p.id, thumbnailName)
	}

	return info
}

func (p *Package) getFileList(r io.ReadSeeker) (err error) {
	headers, err := p.ReadPackageHeaders(r)
	if err != nil {
		return
	}
	p.log.WithField("headers", headers).
		Debug("info")

	fileCount := headers.FileCount

	for fileNum := uint32(1); fileNum <= fileCount; fileNum++ {
		var filePathLength int32
		err = binary.Read(r, binary.LittleEndian, &filePathLength)
		if err != nil {
			p.log.WithError(err).
				WithField("FileNum", fileNum).Error("could not read file path length")
			return
		}

		var infoBytes structures.FileInfoBytes
		pathBytes := make([]byte, filePathLength)
		err = binary.Read(r, binary.LittleEndian, &pathBytes)
		if err != nil {
			p.log.WithError(err).
				WithField("filePathLength", filePathLength).
				WithField("FileNum", fileNum).Error("could not read file path")
			return
		}

		err = binary.Read(r, binary.LittleEndian, &infoBytes)
		if err != nil {
			p.log.
				WithError(err).
				WithField("FileNum", fileNum).Error("could not read file info")
			return
		}

		info := p.newFileInfo(pathBytes, infoBytes)

		p.log.
			WithField("info", info).
			Infof("found file [%v/%v]", fileNum, fileCount)
		p.FileList = append(p.FileList, info)

	}

	return
}
