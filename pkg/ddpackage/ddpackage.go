package ddpackage

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
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

type PackageMode int

const (
	PackageModeUnloaded PackageMode = iota
	PackageModePacked
	PackageModeUnpacked
)

type Package struct {
	log           logrus.FieldLogger
	mode          PackageMode
	name          string
	id            string
	unpackOptions *UnpackOptions
	packOptions   *PackOptions
	unpackedPath  string
	packedPath    string
	alignment     int
	fileList      structures.FileInfoList
	resourceMap   map[string]*structures.FileInfo
	info          structures.PackageInfo

	walls    map[string]structures.PackageWall
	tilesets map[string]structures.PackageTileset

	tags structures.PackageTags

	pkgFile *os.File
}

func (p *Package) Close() {
	if p.pkgFile != nil {
		p.pkgFile.Close()
	}
}

func (p *Package) ID() string {
	return p.id
}

func (p *Package) Name() string {
	return p.name
}

func (p *Package) UnpackedPath() string {
	return p.unpackedPath
}

func (p *Package) PackedPath() string {
	return p.packedPath
}

func (p *Package) FileList() structures.FileInfoList {
	return p.fileList
}

func (p *Package) Info() structures.PackageInfo {
	return p.info
}

func (p *Package) Walls() *map[string]structures.PackageWall {
	return &p.walls
}

func (p *Package) Tags() *structures.PackageTags {
	return &p.tags
}

func (p *Package) Tilesets() *map[string]structures.PackageTileset {
	return &p.tilesets
}

func NewPackage(log logrus.FieldLogger) *Package {
	return &Package{
		log:         log,
		mode:        PackageModeUnloaded,
		alignment:   0,
		walls:       make(map[string]structures.PackageWall),
		tilesets:    make(map[string]structures.PackageTileset),
		resourceMap: make(map[string]*structures.FileInfo),
		tags:        *structures.NewPackageTags(),
	}
}

// set packed file alignment
func (p *Package) SetAlignment(alignment int) error {
	if alignment < 0 {
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

func (p *Package) LoadFromPackedPath(path string) error {
	packFilePath, pathErr := filepath.Abs(path)
	if pathErr != nil {
		p.log.WithField("path", packFilePath).WithError(pathErr).Error("could not get absolute path for package file")
		return errors.Join(pathErr, errors.New("could not get absolute path for package file"))
	}
	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		p.log.WithField("path", packFilePath).WithError(fileErr).Error("could not open package file for reading")
		return errors.Join(fileErr, errors.New("could not open package file for reading"))
	}

	p.SetUnpackOptions(UnpackOptions{})

	err := p.loadPackedFilelist(file)
	if err != nil {
		p.log.WithError(err).Error("failed to read file list")
		file.Close()
		return errors.Join(err, errors.New("failed to read file list"))
	}
	err = p.loadPackedPackJSON(file)
	if err != nil {
		p.log.WithError(err).Error("failed to read pack json")
		file.Close()
		return errors.Join(err, errors.New("failed to read pack json"))
	}
	p.packedPath = packFilePath
	p.pkgFile = file
	p.mode = PackageModePacked

	p.updateResourceMap()

	return nil
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

	if err := p.loadUnpackedPackJSON(dirPath); err != nil {
		return err
	}

	p.SetPackOptions(PackOptions{})

	p.unpackedPath = dirPath
	p.mode = PackageModeUnpacked

	return nil
}

func (p *Package) updateResourceMap() {
	p.resourceMap = make(map[string]*structures.FileInfo, len(p.fileList))
	for i := 0; i < len(p.fileList); i++ {
		info := &p.fileList[i]
		p.resourceMap[info.ResPath] = info
	}
}

// get the *FileInfo for a resource identified by the passed 'res://' path
func (p *Package) GetResourceInfo(resPath string) (*structures.FileInfo, error) {
	info, found := p.resourceMap[resPath]
	if found && info != nil {
		return info, nil
	}

	info = p.fileList.Find(func(fi *structures.FileInfo) bool {
		return fi.ResPath == resPath
	})
	if info != nil {
		return info, nil
	}
	return nil, ErrResourceNotFound
}

func (p *Package) ReadPackJSON() error {
	switch p.mode {
	case PackageModePacked:
		return p.loadPackedPackJSON(p.pkgFile)
	case PackageModeUnpacked:
		return p.loadUnpackedPackJSON(p.unpackedPath)
	}
	return ErrPackageNotLoaded
}

func (p *Package) LoadTags() error {
	switch p.mode {
	case PackageModePacked:
		return p.loadPackedTags(p.pkgFile)
	case PackageModeUnpacked:
		return p.loadUnpackedTags()
	}
	return ErrPackageNotLoaded
}

func (p *Package) LoadResourceMetadata() error {
	switch p.mode {
	case PackageModePacked:
		return p.loadPackedResourceMetadata(p.pkgFile)
	case PackageModeUnpacked:
		return p.loadUnpackedResourceMetadata()
	}
	return ErrPackageNotLoaded
}

// Load the resource identified by the passed 'res://' path
func (p *Package) LoadResource(resPath string) ([]byte, error) {
	if p.mode != PackageModePacked && p.mode != PackageModeUnpacked {
		return nil, ErrPackageNotLoaded
	}

	info, error := p.GetResourceInfo(resPath)
	if error != nil {
		return nil, error
	}

	if p.mode == PackageModePacked {
		return p.readPackedFileFromPackage(p.pkgFile, info)
	} else {
		return p.readUnpackedFileFromPackage(info)
	}
}

type NewFileInfoOptions struct {
	Path    string
	ResPath *string
	RelPath *string
	Size    *int64
}

func (p *Package) NewFileInfo(options NewFileInfoOptions) (*structures.FileInfo, error) {
	if p.unpackedPath == "" {
		return nil, ErrUnsetUnpackedPath
	}

	if options.Path != "" {

		if options.Size == nil {
			fileInfo, err := os.Stat(options.Path)
			if err != nil {
				p.log.WithError(err).Errorf("can't stat %s", options.Path)
				return nil, err
			}
			options.Size = new(int64)
			*options.Size = fileInfo.Size()
		}

		l := p.log.WithField("filePath", options.Path)
		relPath, err := filepath.Rel(p.unpackedPath, options.Path)
		if err != nil {
			l.Error("can not get path relative to package root")
			return nil, err
		}

		if options.ResPath == nil {
			options.ResPath = new(string)
			*options.ResPath = fmt.Sprintf("res://packs/%s/%s", p.id, relPath)

			if runtime.GOOS == "windows" { // windows path separators.....
				*options.ResPath = strings.ReplaceAll(*options.ResPath, "\\", "/")
			}
		}

		if options.RelPath == nil {
			options.RelPath = new(string)
			if runtime.GOOS == "windows" { // windows path separators.....
				relPath = strings.ReplaceAll(relPath, "\\", "/")
			}
			*options.RelPath = relPath
		}
	}

	info := &structures.FileInfo{
		Path:        options.Path,
		ResPath:     *options.ResPath,
		RelPath:     *options.RelPath,
		ResPathSize: int32(len([]byte(*options.ResPath))),
		Size:        *options.Size,
	}

	if options.Path != "" && ddimage.PathIsSupportedImage(options.Path) {

		l := p.log.WithField("filePath", options.Path)

		thumbnailDir := filepath.Join(p.unpackedPath, "thumbnails")
		hash := md5.Sum([]byte(*options.ResPath))
		thumbnailName := hex.EncodeToString(hash[:]) + ".png"
		thumbnailPath := filepath.Join(thumbnailDir, thumbnailName)
		info.ThumbnailPath = thumbnailPath
		info.ThumbnailResPath = fmt.Sprintf("res://packs/%s/thumbnails/%s", p.id, thumbnailName)

		img, format, err := ddimage.OpenImage(options.Path)
		if err != nil {
			l.WithError(err).Error("can not open path with image extension as image")
			err = errors.Join(err, fmt.Errorf("failed to open %s as an image", options.Path))
			// log but let info construction continue
		} else {
			l.WithField("imageFormat", format).Trace("read image")
			info.ImageFormat = format

			if !ddimage.PathIsSupportedDDImage(options.Path) {
				info.Image = img
				l.WithField("imageFormat", format).
					Info("format is not supported by dungeondraft, converting to png")
				buf := new(bytes.Buffer)
				err = ddimage.PngImageBytes(img, buf)
				if err != nil {
					l.WithError(err).Error("failed to encode png version of image")
					// log but let info construction continue
				} else {
					imgBytes := buf.Bytes()
					info.PngImage = make([]byte, len(imgBytes))
					copy(info.PngImage, imgBytes)

					info.Size = int64(len(info.PngImage))

					ext := filepath.Ext(options.Path)
					info.ResPath = info.ResPath[0:len(info.ResPath)-len(ext)] + ".png"
					info.RelPath = info.RelPath[0:len(info.RelPath)-len(ext)] + ".png"
				}
			}
		}

		isWall := info.IsWall()
		isTileset := info.IsTileset()

		if isWall || isTileset {
			fName := filepath.Base(info.RelPath)
			bName := fName[:len(fName)-len(filepath.Ext(fName))]

			if isWall {
				info.MetadataPath = fmt.Sprintf("res://packs/%s/data/walls/%s.dungeondraft_wall", p.id, bName)
			} else {
				info.MetadataPath = fmt.Sprintf("res://packs/%s/data/tilesets/%s.dungeondraft_tileset", p.id, bName)
			}
		}

	}

	return info, nil
}
