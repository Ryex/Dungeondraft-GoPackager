package ddpackage

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

func (p *Package) ReadUnpackedPackJson(dirPath string) error {
	packJSONPath := filepath.Join(dirPath, `pack.json`)

	packExists := utils.FileExists(packJSONPath)
	if !packExists {
		err := fmt.Errorf("no pack.json in directory %s, generate one first.", dirPath)
		p.log.WithError(err).
			WithField("path", dirPath).
			Error("can't package without a pack.json")
		return errors.Join(err, MissingPackJsonError, errors.New("can't package without a pack.json"))
	}

	packJSONBytes, err := os.ReadFile(packJSONPath)
	if err != nil {
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("can't read pack.json")
		return errors.Join(err, PackJsonReadError)
	}

	var pack structures.PackageInfo

	err = json.Unmarshal(packJSONBytes, &pack)
	if err != nil {
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("can't parse pack.json")
		return errors.Join(err, InvalidPackJsonError)
	}

	pack.Keywords = strings.Split(pack.KeywordsRaw, ",")

	if pack.Name == "" {
		err = errors.New("pack.json's name field can not be empty")
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("invalid pack.json")
		return errors.Join(InvalidPackJsonError, err)
	}

	if pack.ID == "" {
		err = errors.New("pack.json's id field can not be empty")
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("invalid pack.json")
		return errors.Join(InvalidPackJsonError, err)
	}

	p.Info = pack
	p.id = pack.ID
	p.name = pack.Name

	return nil
}

// PackPackage packs up a directory into a .dungeondraft_pack file
// assumes BuildFileList has been called first
func (p *Package) PackPackage(outDir string, options PackOptions) (err error) {
	p.SetPackOptions(options)

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

	outPackagePath := filepath.Join(outDirPath, p.name+".dungeondraft_pack")

	l := p.log.WithField("outPackagePath", outPackagePath)

	packageExists := utils.FileExists(outPackagePath)
	if packageExists {
		if p.packOptions.Overwrite {
			l.Warn("overwriting file")
		} else {
			err = errors.New("file exists")
			l.WithError(err).Error("package file already exists at destination and Overwrite not enabled")
			return
		}
	}

	p.packedPath = outPackagePath

	l.Debug("writing package")
	var out *os.File
	out, err = os.Create(outPackagePath)
	if err != nil {
		l.WithError(err).Error("can not open package file for writing")
		return
	}
	err = p.writePackage(l, out)
	if err != nil {
		l.WithError(err).Error("failed to write package file")
		return
	}
	err = out.Close()
	if err != nil {
		l.WithError(err).Error("failed to close package file")
		return
	}

	l.Info("packing complete")

	return
}

type NewFileInfoOptions struct {
	Path    string
	ResPath *string
	RelPath *string
	Size    *int64
}

func (p *Package) NewFileInfo(options NewFileInfoOptions) (*structures.FileInfo, error) {
	utils.AssertTrue(p.unpackedPath != "", "empty unpacked path")

	if options.Size == nil {
		fileInfo, err := os.Stat(options.Path)
		if err != nil {
			p.log.WithError(err).Errorf("can't stat %s", options.Path)
			return nil, err
		}
		*options.Size = fileInfo.Size()
	}

	l := p.log.WithField("filePath", options.Path)
	relPath, err := filepath.Rel(p.unpackedPath, options.Path)
	if err != nil {
		l.Error("can not get path relative to package root")
		return nil, err
	}

	if options.ResPath == nil {
		*options.ResPath = fmt.Sprintf("res://packs/%s/%s", p.id, relPath)

		if runtime.GOOS == "windows" { // windows path separators.....
			*options.ResPath = strings.ReplaceAll(*options.ResPath, "\\", "/")
		}
	}

	if options.RelPath == nil {
		if runtime.GOOS == "windows" { // windows path separators.....
			relPath = strings.ReplaceAll(relPath, "\\", "/")
		}
		*options.RelPath = relPath
	}

	info := &structures.FileInfo{
		Path:        options.Path,
		ResPath:     *options.ResPath,
		RelPath:     *options.RelPath,
		ResPathSize: int32(binary.Size([]byte(*options.ResPath))),
		Size:        *options.Size,
	}

	if ddimage.PathIsSupportedImage(options.Path) {

		thumbnailDir := filepath.Join(p.unpackedPath, "thumbnails")
		hash := md5.Sum([]byte(*options.ResPath))
		thumbnailName := hex.EncodeToString(hash[:]) + ".png"
		thumbnailPath := filepath.Join(thumbnailDir, thumbnailName)
		info.ThumbnailPath = thumbnailPath
		info.ThumbnailResPath = fmt.Sprintf("res://packs/%s/thumbnails/%s", p.id, thumbnailName)

		img, format, err := ddimage.OpenImage(options.Path)
		if err != nil {
			l.WithError(err).Error("can not open path with image extension as image")
			return nil, err
		}
		l.WithField("imageFormat", format).Trace("read image")
		info.Image = img
		info.ImageFormat = format

		buf := new(bytes.Buffer)
		err = ddimage.PngImageBytes(img, buf)
		if err != nil {
			l.WithError(err).Error("failed to encode png version of image")
			return nil, err
		}
		imgBytes := buf.Bytes()
		info.PngImage = make([]byte, len(imgBytes))
		copy(info.PngImage, imgBytes)

		info.Size = int64(len(info.PngImage))

		ext := filepath.Ext(options.Path)
		info.ResPath = info.ResPath[0:len(info.ResPath)-len(ext)] + ".png"
		info.RelPath = info.RelPath[0:len(info.RelPath)-len(ext)] + ".png"

	}

	return info, nil
}

// BuildFileList builds a list of files at the target directory for inclusion in a .dungeondraft_pack file
func (p *Package) BuildFileList() (err error) {
	utils.AssertTrue(p.unpackedPath != "", "empty unpacked path")

	p.log.Debug("beginning directory traversal to collect file list")

	err = filepath.Walk(p.unpackedPath, p.fileListWalkFunc)
	if err != nil {
		p.log.WithError(err).Error("failed to walk directory")
		return
	}

	// inject <GUID>.json

	packJSONPath := filepath.Join(p.unpackedPath, `pack.json`)

	packJSONName := fmt.Sprintf(`%s.json`, p.id)
	packJSONResPath := "res://packs/" + packJSONName

	GUIDJSONInfo, err := p.NewFileInfo(NewFileInfoOptions{
		Path:    packJSONPath,
		ResPath: &packJSONResPath,
		RelPath: &packJSONName,
	})
	if err != nil {
		return err
	}

	// prepend the file to the list
	p.FileList = append(p.FileList, structures.FileInfo{}) // make space with empty struct
	copy(p.FileList[1:], p.FileList)                       // move things forward
	p.FileList[0] = *GUIDJSONInfo                          // set to first spot

	return
}

func (p *Package) makeResPath(l logrus.FieldLogger, path string) (string, error) {
	utils.AssertTrue(p.unpackedPath != "", "empty unpacked path")
	relPath, err := filepath.Rel(p.unpackedPath, path)
	if err != nil {
		l.Error("can not get path relative to package root")
		return "", err
	}

	resPath := fmt.Sprintf("res://packs/%s/%s", p.id, relPath)

	if runtime.GOOS == "windows" { // windows path separators.....
		resPath = strings.ReplaceAll(resPath, "\\", "/")
	}

	return resPath, nil
}

func (p *Package) fileListWalkFunc(path string, info os.FileInfo, err error) error {
	l := p.log.WithField("filePath", path)
	if err != nil {
		l.WithError(err).Error("can't access file")
		return err
	}

	if info.IsDir() {
		l.Trace("is directory, descending into...")
	} else {
		ext := strings.ToLower(filepath.Ext(path))
		if utils.StringInSlice(ext, p.packOptions.ValidExts) {
			if info.Mode().IsRegular() {

				fInfo, err := p.NewFileInfo(NewFileInfoOptions{Path: path})
				if err != nil {
					return err
				}

				l.Info("including")
				p.FileList = append(p.FileList, *fInfo)

			}
		} else {
			l.WithField("ext", ext).WithField("validExts", p.packOptions.ValidExts).Debug("Invalid file ext, not including.")
		}
	}

	return nil
}

func (p *Package) writePackage(l logrus.FieldLogger, out io.WriteSeeker) (err error) {
	headers := structures.DefaultPackageHeader()
	headers.FileCount = uint32(len(p.FileList))

	fileInfoList := structures.NewFileInfoList(p.FileList)

	l.Debug("writing package headers...")
	// write file header
	err = headers.Write(out)
	if !utils.CheckErrorWrite(l, err) {
		return
	}

	err = fileInfoList.Write(l, out, headers.SizeOf(), p.alignment)
	if !utils.CheckErrorWrite(l, err) {
		return
	}

	return
}
