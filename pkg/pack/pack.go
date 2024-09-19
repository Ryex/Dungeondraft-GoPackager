package pack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ryex/dungeondraft-gopackager/internal/structures"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/sirupsen/logrus"
)

// Packer packs up a folder into a dungeodraft_pack file
// set the Overwrite field if you wish pack operations to overwrite an existing file
type Packer struct {
	log       logrus.FieldLogger
	name      string
	id        string
	path      string
	Overwrite bool
	FileList  []structures.FileInfo
	ValidExts []string
}

// DefaultValidExt returns a slice of valid file extensions for inclusion in a .dungeondraft_pack
func DefaultValidExt() []string {
	return []string{
		".png", ".jpg", ".webp",
		".dungeondraft_wall", ".dungeondraft_tileset",
		".dungeondraft_tags", ".json",
	}
}

func GenPackID() string {
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))

	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, 8)

	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// NewPackerFromFolder builds a new Packer from a folder with a valid pack.json
func NewPackerFolder(log logrus.FieldLogger, folderPath string, name string, author string, version string, overwrite bool) (p *Packer, err error) {

	folderPath, err = filepath.Abs(folderPath)
	if err != nil {
		return
	}

	dirExists := utils.DirExists(folderPath)
	if !dirExists {
		err = errors.New("directory does not exists")
		log.WithError(err).WithField("path", folderPath).Error("can't package a non existent folder")
		return
	}

	packJSONPath := filepath.Join(folderPath, `pack.json`)

	packExists := utils.FileExists(packJSONPath)
	if packExists {
		if !overwrite {
			err = errors.New("a pack.json already exists and overwrite is not enabled")
			log.WithError(err).WithField("path", folderPath).Error("a pack.json already exists")
			return
		} else {
			log.WithField("path", folderPath).Warn("Overwriting pack.json")
		}
	}

	if name == "" {
		err = errors.New("name field can not be empty")
		log.WithError(err).Error("invalid pack info")
		return
	}

	if version == "" {
		err = errors.New("version field can not be empty")
		log.WithError(err).Error("invalid pack info")
		return
	}

	pack := structures.Package{
		Name:    name,
		Author:  author,
		Version: version,
		ID:      GenPackID(),
	}

	packJSONBytes, err := json.MarshalIndent(&pack, "", "  ")
	if err != nil {
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("can't create pack.json")
		return
	}

	err = os.WriteFile(packJSONPath, packJSONBytes, 0644)
	if err != nil {
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("can't write pack.json")
		return
	}

	p = NewPacker(log.WithField("path", folderPath).WithField("id", pack.ID).WithField("name", pack.Name), pack.Name, pack.ID, folderPath)
	return

}

// NewPackerFromFolder builds a new Packer from a folder with a valid pack.json
func NewPackerFromFolder(log logrus.FieldLogger, folderPath string) (p *Packer, err error) {

	folderPath, err = filepath.Abs(folderPath)
	if err != nil {
		return
	}

	dirExists := utils.DirExists(folderPath)
	if !dirExists {
		err = errors.New("directory does not exists")
		log.WithError(err).WithField("path", folderPath).Error("can't package a non existent folder")
		return
	}

	packJSONPath := filepath.Join(folderPath, `pack.json`)

	packExists := utils.FileExists(packJSONPath)
	if !packExists {
		err = errors.New("no pack.json in directory, generate one first.")
		log.WithError(err).WithField("path", folderPath).Error("can't package without a pack.json")
		return
	}

	packJSONBytes, err := os.ReadFile(packJSONPath)
	if err != nil {
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("can't read pack.json")
		return
	}

	var pack structures.Package

	err = json.Unmarshal(packJSONBytes, &pack)
	if err != nil {
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("can't parse pack.json")
		return
	}

	if pack.Name == "" {
		err = errors.New("pack.json's name field can not be empty")
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("invalid pack.json")
		return
	}

	if pack.ID == "" {
		err = errors.New("pack.json's id field can not be empty")
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("invalid pack.json")
		return
	}

	p = NewPacker(log.WithField("path", folderPath).WithField("id", pack.ID).WithField("name", pack.Name), pack.Name, pack.ID, folderPath)
	return

}

// NewPacker makes a new Packer, it does no validation so the subsequent pack operations may fail badly
func NewPacker(log logrus.FieldLogger, name string, id string, path string) *Packer {
	return &Packer{
		log:       log,
		name:      name,
		id:        id,
		path:      path,
		ValidExts: DefaultValidExt(),
	}
}

// PackPackage packs up a directory into a .dungeondraft_pack file
// assumes BuildFileList has been called first
func (p *Packer) PackPackage(outDir string) (err error) {

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
		if p.Overwrite {
			l.Warn("overwriting file")
		} else {
			err = errors.New("file exists")
			l.WithError(err).Error("package file already exists at destination and Overwrite not enabled")
			return
		}
	}

	l.Debug("writing package")
	var out *os.File
	out, err = os.Create(outPackagePath)
	if err != nil {
		l.WithError(err).Error("can not open package file for writing")
		return
	}
	err = p.write(l, out)
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
func (p *Packer) BuildFileList() (err error) {
	p.log.Debug("beginning directory traversal to collect file list")

	err = filepath.Walk(p.path, p.fileListWalkFunc)
	if err != nil {
		p.log.WithError(err).Error("failed to walk directory")
		return
	}

	// inject <GUID>.json

	packJSONPath := filepath.Join(p.path, `pack.json`)

	packJSONRelPath, err := filepath.Rel(p.path, filepath.Join(p.path, fmt.Sprintf(`%s.json`, p.id)))
	if err != nil {
		p.log.Error("can not get path relative to package root")
		return err
	}

	pathJSONResPath := "res://" + filepath.Join("packs", packJSONRelPath)

	if runtime.GOOS == "windows" { // windows path separators.....
		pathJSONResPath = strings.ReplaceAll(pathJSONResPath, "\\", "/")
	}

	packJSONInfo, err := os.Stat(packJSONPath)
	if err != nil {
		p.log.WithError(err).Error("can't stat pack.json")
	}

	GUIDJSONInfo := structures.FileInfo{
		Path:    packJSONPath,
		Size:    packJSONInfo.Size(),
		ResPath: pathJSONResPath,
	}

	// prepend the file to the list
	p.FileList = append(p.FileList, structures.FileInfo{}) // make space with empty struct
	copy(p.FileList[1:], p.FileList)                       // move things forward
	p.FileList[0] = GUIDJSONInfo                           // set to first spot

	return
}

func (p *Packer) makeResPath(l logrus.FieldLogger, path string) (string, error) {
	relPath, err := filepath.Rel(p.path, path)
	if err != nil {
		l.Error("can not get path relative to package root")
		return "", err
	}

	resPath := "res://" + filepath.Join("packs", p.id, relPath)

	if runtime.GOOS == "windows" { // windows path separators.....
		resPath = strings.ReplaceAll(resPath, "\\", "/")
	}

	return resPath, nil
}

func (p *Packer) fileListWalkFunc(path string, info os.FileInfo, err error) error {
	l := p.log.WithField("filePath", path)
	if err != nil {
		l.WithError(err).Error("can't access file")
		return err
	}

	if info.IsDir() {
		l.Debug("is directory, descending into...")
	} else {
		ext := strings.ToLower(filepath.Ext(path))
		if utils.StringInSlice(ext, p.ValidExts) {
			if info.Mode().IsRegular() {

				resPath, err := p.makeResPath(l, path)
				if err != nil {
					return err
				}

				fInfo := structures.FileInfo{
					Path:    path,
					Size:    info.Size(),
					ResPath: resPath,
				}
				l.Info("including")
				p.FileList = append(p.FileList, fInfo)
			}
		} else {
			l.WithField("ext", ext).WithField("validExts", p.ValidExts).Debug("Invalid file ext, not including.")
		}
	}

	return nil
}

func (p *Packer) write(l logrus.FieldLogger, out io.WriteSeeker) (err error) {

	headers := structures.DefaultPackageHeader()
	headers.FileCount = uint32(len(p.FileList))

	fileInfoList := structures.NewFileInfoList(p.FileList)

	l.Debug("writing package headers...")
	// write file header
	err = headers.Write(out)
	if !utils.CheckErrorWrite(l, err) {
		return
	}

	err = fileInfoList.Write(l, out, headers.SizeOf())
	if !utils.CheckErrorWrite(l, err) {
		return
	}

	return
}
