package ddpackage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

func (p *Package) LoadPackedTags(r io.ReadSeeker) error {
	if p.FileList == nil {
		return EmptyFileListError
	}
	if p.id == "" {
		return UnsetPackIdError
	}
	tagsResPath := fmt.Sprintf("res://packs/%s/data/default.dungeondraft_tags", p.id)

	var tagsInfo *structures.FileInfo
	for i := 0; i < len(p.FileList); i++ {
		packedFile := &p.FileList[i]
		if packedFile.ResPath == tagsResPath {
			tagsInfo = packedFile
			break
		}
	}

	if tagsInfo == nil {
		p.log.Info("no default.dungeondraft_tags file in pack")
		return nil
	}

	tagsBytes, err := p.ReadFileFromPackage(r, *tagsInfo)
	if err != nil {
		p.log.WithError(err).WithField("res", tagsResPath).Error("failed to read tags file")
		return errors.Join(err, TagsReadError)
	}

	err = json.Unmarshal(tagsBytes, &p.Tags)
	if err != nil {
		p.log.WithError(err).WithField("res", tagsResPath).Error("failed to parse tags file")
		return errors.Join(err, TagsParseError)
	}

	return nil
}

func (p *Package) LoadPackedResourceMetadata(r io.ReadSeeker) error {
	if p.id == "" {
		return UnsetPackIdError
	}

	for i := 0; i < len(p.FileList); i++ {
		info := &p.FileList[i]

		if !(info.IsWallData() || info.IsTilesetData()) {
			continue
		}

		fileData, err := p.ReadFileFromPackage(r, *info)
		if err != nil {
			p.log.WithError(err).WithField("res", info.ResPath).Error("failed to read data file")
			return errors.Join(err, MetadataReadError, fmt.Errorf("failed to read data file %s", info.ResPath))
		}

		if info.IsWallData() {
			wall := structures.NewPackageWall()
			err = json.Unmarshal(fileData, wall)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, WallParseError, fmt.Errorf("failed to parse data file %s", info.ResPath))
			}
			p.Walls[info.ResPath] = *wall
		} else if info.IsTilesetData() {
			ts := structures.NewPackageTileset()
			err = json.Unmarshal(fileData, ts)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, TilesetParseError, fmt.Errorf("failed to parse data file %s", info.ResPath))
			}
			p.Tilesets[info.ResPath] = *ts
		}
	}

	return nil
}

func (p *Package) ReadUnpackedTags() error {
	if p.UnpackedPath == "" {
		return UnsetUnpackedPathError
	}

	tagsPath := filepath.Join(p.UnpackedPath, "data", "default.dungeondraft_tags")

	if tagsExist := utils.FileExists(tagsPath); !tagsExist {
		p.log.WithField("tagsPath", tagsPath).Info("no default.dungeondraft_tags file in pack")
		return nil
	}

	tagsBytes, err := os.ReadFile(tagsPath)
	if err != nil {
		p.log.WithError(err).
			WithField("path", p.UnpackedPath).
			WithField("tagsPath", tagsPath).
			Error("can't read tags file")
		return errors.Join(err, TagsReadError)
	}

	err = json.Unmarshal(tagsBytes, &p.Tags)
	if err != nil {
		p.log.WithError(err).WithField("tagsPath", tagsPath).Error("failed to parse tags file")
		return errors.Join(err, TagsParseError)
	}

	return nil
}

func (p *Package) ReadUnpackedResourceMetadata() error {
	if p.UnpackedPath == "" {
		return UnsetUnpackedPathError
	}

	for i := 0; i < len(p.FileList); i++ {
		info := &p.FileList[i]

		if !(info.IsWallData() || info.IsTilesetData()) {
			continue
		}

		fileData, err := os.ReadFile(info.Path)
		if err != nil {
			p.log.WithError(err).WithField("res", info.ResPath).Error("failed to read data file")
			return errors.Join(err, MetadataReadError, fmt.Errorf("failed to read data file %s", info.Path))
		}

		if info.IsWallData() {
			wall := structures.NewPackageWall()
			err = json.Unmarshal(fileData, wall)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, WallParseError, fmt.Errorf("failed to parse data file %s", info.Path))
			}
			p.Walls[info.ResPath] = *wall
		} else if info.IsTilesetData() {
			ts := structures.NewPackageTileset()
			err = json.Unmarshal(fileData, ts)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, TilesetParseError, fmt.Errorf("failed to parse data file %s", info.Path))
			}
			p.Tilesets[info.ResPath] = *ts
		}
	}

	return nil
}

func (p *Package) WriteUnpackedTags() error {
	if p.UnpackedPath == "" {
		return UnsetUnpackedPathError
	}

	tagsPath := filepath.Join(p.UnpackedPath, "data", "default.dungeondraft_tags")
	dirPath := filepath.Dir(tagsPath)

	if dirExists := utils.DirExists(dirPath); !dirExists {
		err := os.MkdirAll(dirPath, 0777)
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed to make directory %s", dirPath))
		}
	}

	tagsBytes, err := json.MarshalIndent(&p.Tags, "", "  ")
	if err != nil {
		p.log.WithError(err).
			Error("failed to create tags json")
		return errors.Join(err, errors.New("failed to create tags json"))
	}

	err = os.WriteFile(tagsPath, tagsBytes, 0644)
	if err != nil {
		p.log.WithError(err).
			Error("failed to write tags file")
		return errors.Join(err, errors.New("failed to write tags file"))
	}

	return nil
}

func (p *Package) WriteResourceMetadata() error {
	if p.UnpackedPath == "" {
		return UnsetUnpackedPathError
	}

	for i := 0; i < len(p.FileList); i++ {
		info := &p.FileList[i]

		if !(info.IsWallData() || info.IsTilesetData()) {
			continue
		}

		dirPath := filepath.Dir(info.Path)

		if dirExists := utils.DirExists(dirPath); !dirExists {
			err := os.MkdirAll(dirPath, 0777)
			if err != nil {
				return errors.Join(err, fmt.Errorf("failed to make directory %s", dirPath))
			}
		}

		var fileBytes []byte
		var err error
		if info.IsWallData() {
			wall := p.Walls[info.ResPath]
			fileBytes, err = json.MarshalIndent(&wall, "", "  ")
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.ResPath).
					Error("failed to create wall json")
				return errors.Join(err, fmt.Errorf("failed to create wall json for %s", info.ResPath))
			}
		} else {
			tileset := p.Tilesets[info.ResPath]
			fileBytes, err = json.MarshalIndent(&tileset, "", "  ")
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.ResPath).
					Error("failed to create tileset json")
				return errors.Join(err, fmt.Errorf("failed to create tileset json for %s", info.ResPath))
			}
		}

		if fileBytes != nil && len(fileBytes) > 0 {
			err = os.WriteFile(info.Path, fileBytes, 0644)
			if err != nil {
				p.log.WithError(err).
					Error("failed to write metadata file")
				return errors.Join(err, fmt.Errorf("failed to write metadata file for %s", info.ResPath))
			}
		}
	}

	return nil
}
