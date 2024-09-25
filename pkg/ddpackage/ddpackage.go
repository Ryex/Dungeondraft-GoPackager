package ddpackage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

type PackOptions struct {
	Overwrite bool
	ValidExts []string
}

type UnpackOptions struct {
	RipTextures bool
	Overwrite   bool
	Thumbnails  bool
}

type Package struct {
	log           logrus.FieldLogger
	name          string
	id            string
	unpackOptions *UnpackOptions
	packOptions   *PackOptions
	UnpackedPath  string
	PackedPath    string
	alignment     int
	FileList      []structures.FileInfo
	Info          structures.PackageInfo

	Walls    map[string]structures.PackageWall
	Tilesets map[string]structures.PackageTileset

	Tags structures.PackageTags
}

func (p *Package) Id() string {
	return p.id
}

func (p *Package) Name() string {
	return p.name
}

func NewPackage(log logrus.FieldLogger) *Package {
	return &Package{
		log:       log,
		alignment: 0,
		Walls: make(map[string]structures.PackageWall),
		Tilesets: make(map[string]structures.PackageTileset),
		Tags: *structures.NewPackageTags(),
	}
}

// set packed file alignment
func (p *Package) SetAlignment(alignment int) error {
	if alignment > 0 {
		return errors.New("alignment must be greater than 0")
	}
	p.alignment = alignment
	return nil
}

func (p *Package) SetUnpackOptions(options UnpackOptions) {
	p.unpackOptions = &options
}

func (p *Package) SetPackOptions(options PackOptions) {
	if options.ValidExts == nil || len(options.ValidExts) == 0 {
		options.ValidExts = DefaultValidExt()
	}
	p.packOptions = &options
}

// DefaultValidExt returns a slice of valid file extensions for inclusion in a .dungeondraft_pack
func DefaultValidExt() []string {
	return []string{
		".png", ".webp", ".jpg", ".jpeg",
		".gif", ".tif", ".tiff", ".bmp",
		".dungeondraft_wall", ".dungeondraft_tileset",
		".dungeondraft_tags", ".json",
	}
}

func (p *Package) LoadFromPackedPath(path string) (*os.File, error) {
	packFilePath, pathErr := filepath.Abs(path)
	if pathErr != nil {
		p.log.WithField("path", packFilePath).WithError(pathErr).Error("could not get absolute path for package file")
		return nil, errors.Join(pathErr, errors.New("could not get absolute path for package file"))
	}
	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		p.log.WithField("path", packFilePath).WithError(fileErr).Error("could not open package file for reading")
		return nil, errors.Join(fileErr, errors.New("could not open package file for reading"))
	}

	p.SetUnpackOptions(UnpackOptions{})

	p.PackedPath = packFilePath
	err := p.ReadPackageFilelist(file)
	if err != nil {
		p.log.WithError(err).Error("failed to read file list")
		file.Close()
		return nil, errors.Join(err, errors.New("failed to read file list"))
	}
	err = p.ReadPackedPackJson(file)
	if err != nil {
		p.log.WithError(err).Error("failed to read pack json")
		file.Close()
		return nil, errors.Join(err, errors.New("failed to read pack json"))
	}

	return file, nil
}

func (p *Package) LoadUnpackedFromFolder(dirPath string) error {
	dirPath, pathErr := filepath.Abs(dirPath)
	if pathErr != nil {
		p.log.WithField("path", dirPath).
			WithError(pathErr).
			Error("could not get absolute path for package folder")
		return errors.Join(pathErr, errors.New("could not get absolute path for package folder"))
	}

	if dirExists := utils.DirExists(dirPath); !dirExists {
		err := fmt.Errorf("path %s does not exists or is not a directory", dirPath)
		p.log.WithError(err).
			WithField("path", dirPath).
			Error("can't package a non existent folder")
		return err
	}

	if err := p.ReadUnpackedPackJson(dirPath); err != nil {
		return err
	}

	p.SetPackOptions(PackOptions{})

	p.UnpackedPath = dirPath

	return nil
}
