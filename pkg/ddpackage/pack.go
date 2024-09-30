package ddpackage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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

func (p *Package) UpdateFromPathsProgress(paths []string, progressCallback func(p float64, curPath string)) (errs []error) {
	return p.updateFromPaths(paths, progressCallback)
}

func (p *Package) UpdateFromPaths(paths []string) (errs []error) {
	return p.updateFromPaths(paths, nil)
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
	return p.updateFromPaths([]string{p.unpackedPath}, progressCallback)
}

// updates the current list of files at the target directory for inclusion in a .dungeondraft_pack file
// on duplicate entries updates the current info
func (p *Package) updateFromPaths(paths []string, progressCallback func(p float64, curPath string)) (errs []error) {
	if p.unpackedPath == "" {
		return []error{ErrUnsetUnpackedPath}
	}
	if p.mode != PackageModeUnpacked {
		return []error{ErrPackageNotUnpacked}
	}

	dirs := structures.NewSet[string]()
	files := structures.NewSet[string]()
	toRemove := structures.NewSet[string]()

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		statInfo, err := os.Stat(absPath)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if statInfo.IsDir() {
			dirs.Add(absPath)
			p.log.WithField("listPath", absPath).Debug("beginning directory traversal to collect file list")
			toAdd, _, _ := utils.ListDir(absPath)
			files.AddM(toAdd...)
		} else {
			files.Add(absPath)
		}

	}

	p.flLock.Lock()
	defer p.flLock.Unlock()

	for _, dir := range dirs.AsSlice() {
		for _, fi := range p.fileList {
			if isSub, _ := utils.PathIsSub(dir, fi.Path); isSub && !files.Has(fi.Path) {
				toRemove.Add(fi.ResPath)
			}
		}
	}

	p.fileList.SetCapacity(files.Size())

	cbPoint := max(files.Size()/200, 1)

	for i, file := range files.AsSlice() {
		// filter extensions
		ext := strings.ToLower(filepath.Ext(file))
		if !slices.Contains(p.packOptions.ValidExts, ext) {
			continue
		}
		// construct resource path
		relPath, err := filepath.Rel(p.unpackedPath, file)
		if err != nil {
			p.log.WithField("scanFile", file).Error("can not get path relative to package root")
			errs = append(errs, err)
			continue
		}

		if runtime.GOOS == "windows" { // windows path separators.....
			relPath = strings.ReplaceAll(relPath, "\\", "/")
		}
		resPath := fmt.Sprintf("res://packs/%s/%s", p.id, relPath)

		// update or add
		_, ok := p.resourceMap[resPath]
		if !ok {
			fInfo, err := p.NewFileInfo(NewFileInfoOptions{Path: file, ResPath: &resPath, RelPath: &relPath})
			if err != nil {
				errs = append(errs, err)
				continue
			}
			p.log.Infof("including %s", file)
			p.addResource(fInfo)
		}
		if i%cbPoint == 0 {
			if progressCallback != nil {
				progressCallback(float64(i)/float64(files.Size()), file)
			}
		}
	}

	for _, res := range toRemove.AsSlice() {
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

	// remove duplicates
	p.fileList = slices.CompactFunc(p.fileList, func(a, b *structures.FileInfo) bool {
		return a.ResPath == b.ResPath
	})

	// sort list and assure pack json is first
	p.fileList.Sort()
	p.fileList = slices.Insert(p.fileList, 0, packJSONInfo)
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

	err = p.fileList.Write(l, out, p.alignment, progressCallbacks...)
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
