package ddpackage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

func (p *Package) LoadUnpackedPackJSON(dirPath string) error {
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

func (p *Package) BuildFileListProgress(progressCallback func(p float64, curPath string)) (errs []error) {
	return p.buildFileList(progressCallback)
}

func (p *Package) BuildFileList() (errs []error) {
	return p.buildFileList(nil)
}

func (p *Package) UpdateFromPathProgress(path string, progressCallback func(p float64, curPath string)) (errs []error) {
	return p.updateFromPath(path, progressCallback)
}

func (p *Package) UpdateFromPath(path string) (errs []error) {
	return p.updateFromPath(path, nil)
}

// Rebuilds the list of files at the target directory for inclusion in a .dungeondraft_pack file
func (p *Package) buildFileList(progressCallback func(p float64, curPath string)) (errs []error) {
	if p.unpackedPath == "" {
		return []error{ErrUnsetUnpackedPath}
	}
	if p.mode != PackageModeUnpacked {
		return []error{ErrPackageNotUnpacked}
	}
	p.resetData()
	return p.updateFromPath(p.unpackedPath, progressCallback)
}

// updates the current list of files at the target directory for inclusion in a .dungeondraft_pack file
// on duplicate entries updates the current info
func (p *Package) updateFromPath(path string, progressCallback func(p float64, curPath string)) (errs []error) {
	if p.unpackedPath == "" {
		return []error{ErrUnsetUnpackedPath}
	}
	if p.mode != PackageModeUnpacked {
		return []error{ErrPackageNotUnpacked}
	}

	p.log.WithField("listPath", path).Debug("beginning directory traversal to collect file list")

	path, err := filepath.Abs(path)
	if err != nil {
		return []error{err}
	}

	var toRemove []string
	var files []string
	pathIsDir := false

	statInfo, err := os.Stat(path)
	if err != nil {
		return []error{err}
	}
	if statInfo.IsDir() {
		pathIsDir = true
		files, _, _ = utils.ListDir(path)
	} else {
		files = []string{path}
	}
	filesSet := structures.SetFrom(files)

	p.flLock.Lock()
	defer p.flLock.Unlock()

	if pathIsDir {
		for _, fi := range p.fileList {
			if isSub, _ := utils.PathIsSub(path, fi.Path); isSub && !filesSet.Has(fi.Path) {
				toRemove = append(toRemove, fi.ResPath)
			}
		}
	}

	for i, file := range files {
		// filter extensions
		ext := strings.ToLower(filepath.Ext(file))
		if !utils.InSlice(ext, p.packOptions.ValidExts) {
			continue
		}
		// construct resource path
		relPath, err := filepath.Rel(p.unpackedPath, file)
		if err != nil {
			p.log.WithField("scanFile", file).Error("can not get path relative to package root")
			errs = append(errs, err)
			continue
		}
		resPath := fmt.Sprintf("res://packs/%s/%s", p.id, relPath)

		// update or add
		existing, ok := p.resourceMap[resPath]
		if ok {
			statInfo, err := os.Stat(file)
			if err != nil {
				p.log.WithError(err).Errorf("can't stat %s", file)
				errs = append(errs, err)
				continue
			}
			existing.Size = statInfo.Size()
		} else {
			fInfo, err := p.NewFileInfo(NewFileInfoOptions{Path: file})
			if err != nil {
				errs = append(errs, err)
				continue
			}
			p.log.Infof("including %s", file)
			p.addResource(fInfo)
		}
		if progressCallback != nil {
			progressCallback(float64(i)/float64(len(files)), file)
		}
	}

	for _, res := range toRemove {
		p.log.Warnf("removing %s (file missing)", res)
		p.removeResource(res)
	}

	// inject <GUID>.json

	packJSONPath := filepath.Join(p.unpackedPath, `pack.json`)
	packJSONName := fmt.Sprintf(`%s.json`, p.id)
	packJSONResPath := "res://packs/" + packJSONName

	packJSONindex := p.fileList.IndexOfRes(packJSONResPath)
	var packJSONInfo *structures.FileInfo
	if packJSONindex != -1 {
		packJSONInfo = p.fileList.Remove(packJSONindex)
	} else {
		fi, err := p.NewFileInfo(NewFileInfoOptions{
			Path:    packJSONPath,
			ResPath: &packJSONResPath,
			RelPath: &packJSONName,
		})
		if err != nil {
			p.log.WithError(err).Errorf("error adding base pack.json")
			errs = append(errs, err)
		}
		packJSONInfo = fi
	}
	p.resourceMap[packJSONResPath] = packJSONInfo

	// sort list and assure pack json it first
	sort.Sort(p.fileList)
	p.fileList = append(p.fileList, nil)
	copy(p.fileList[1:], p.fileList)
	p.fileList[0] = packJSONInfo

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
