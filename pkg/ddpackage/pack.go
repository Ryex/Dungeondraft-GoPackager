package ddpackage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

func (p *Package) loadUnpackedPackJSON(dirPath string) error {
	packJSONPath := filepath.Join(dirPath, `pack.json`)

	packExists := utils.FileExists(packJSONPath)
	if !packExists {
		err := fmt.Errorf("no pack.json in directory %s, generate one first.", dirPath)
		p.log.WithError(err).
			WithField("path", dirPath).
			Error("can't package without a pack.json")
		return errors.Join(err, ErrMissingPackJSON, errors.New("can't package without a pack.json"))
	}

	packJSONBytes, err := os.ReadFile(packJSONPath)
	if err != nil {
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("can't read pack.json")
		return errors.Join(err, ErrPackJSONRead)
	}

	var pack structures.PackageInfo

	err = json.Unmarshal(packJSONBytes, &pack)
	if err != nil {
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("can't parse pack.json")
		return errors.Join(err, ErrInvalidPackJSON)
	}

	pack.Keywords = strings.Split(pack.KeywordsRaw, ",")

	if pack.Name == "" {
		err = errors.New("pack.json's name field can not be empty")
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("invalid pack.json")
		return errors.Join(ErrInvalidPackJSON, err)
	}

	if pack.ID == "" {
		err = errors.New("pack.json's id field can not be empty")
		p.log.WithError(err).
			WithField("path", dirPath).
			WithField("packJSONPath", packJSONPath).
			Error("invalid pack.json")
		return errors.Join(ErrInvalidPackJSON, err)
	}

	p.info = pack
	p.id = pack.ID
	p.name = pack.Name

	return nil
}

// PackPackage packs up a directory into a .dungeondraft_pack file
// assumes BuildFileList has been called first
func (p *Package) PackPackage(outDir string, options PackOptions, progressCallbacks ...func(p float64)) (err error) {
	if p.mode != PackageModeUnpacked {
		return ErrPackageNotUnpacked
	}
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
		err = os.MkdirAll(outDirPath, 0o777)
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed to make directory %s", outDirPath))
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
	err = p.writePackage(l, out, progressCallbacks...)
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

// BuildFileList builds a list of files at the target directory for inclusion in a .dungeondraft_pack file
func (p *Package) BuildFileList(progressCallbacks ...func(path string)) (err error) {
	if p.unpackedPath == "" {
		return ErrUnsetUnpackedPath
	}
	if p.mode != PackageModeUnpacked {
		return ErrPackageNotUnpacked
	}

	p.log.Debug("beginning directory traversal to collect file list")

	walkFunc := func(path string, dir fs.DirEntry, err error) error {
		l := p.log.WithField("filePath", path)
		if err != nil {
			l.WithError(err).Error("can't access file")
			return errors.Join(err, fmt.Errorf("can't access %s", path))
		}

		if dir.IsDir() {
			l.Trace("is directory, descending into...")
		} else {
			ext := strings.ToLower(filepath.Ext(path))
			if utils.InSlice(ext, p.packOptions.ValidExts) {
				if dir.Type().IsRegular() {

					fInfo, err := p.NewFileInfo(NewFileInfoOptions{Path: path})
					if err != nil {
						return err
					}

					l.Info("including")
					p.fileList = append(p.fileList, *fInfo)

					for _, pcb := range progressCallbacks {
						pcb(path)
					}
				}
			} else {
				l.WithField("ext", ext).WithField("validExts", p.packOptions.ValidExts).Debug("Invalid file ext, not including.")
			}
		}

		return nil
	}

	err = filepath.WalkDir(p.unpackedPath, walkFunc)
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
	p.fileList = append(p.fileList, structures.FileInfo{}) // make space with empty struct
	copy(p.fileList[1:], p.fileList)                       // move things forward
	p.fileList[0] = *GUIDJSONInfo                          // set to first spot

	p.updateResourceMap()

	return
}

func (p *Package) makeResPath(l logrus.FieldLogger, path string) (string, error) {
	if p.unpackedPath == "" {
		return "", ErrUnsetUnpackedPath
	}
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

func (p *Package) writePackage(l logrus.FieldLogger, out io.WriteSeeker, progressCallbacks ...func(p float64)) (err error) {
	headers := structures.DefaultPackageHeader()
	headers.FileCount = uint32(len(p.fileList))

	l.Debug("writing package headers...")
	// write file header
	err = headers.Write(out)
	if !utils.CheckErrorWrite(l, err) {
		return
	}

	err = p.fileList.Write(l, out, headers.SizeOf(), p.alignment, progressCallbacks...)
	if !utils.CheckErrorWrite(l, err) {
		return
	}

	return
}

func (p *Package) readUnpackedFileFromPackage(info *structures.FileInfo) ([]byte, error) {
  l := p.log.
    WithField("res", info.ResPath).
    WithField("unpackedPath", info.Path)

  fileData, err := os.ReadFile(info.Path)
  if err != nil {
    l.WithError(err).Error("failed to read unpacked resource")
    return nil, errors.Join(err, ErrReadUnpacked, fmt.Errorf("failed to read %s", info.Path))
  }
  return fileData, nil
}
