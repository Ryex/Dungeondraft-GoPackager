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

func (p *Package) loadPackedTags(r io.ReadSeeker) error {
	if p.fileList == nil || len(p.fileList) == 0 {
		return ErrEmptyFileList
	}
	if p.id == "" {
		return ErrUnsetPackID
	}
	tagsResPath := fmt.Sprintf("res://packs/%s/data/default.dungeondraft_tags", p.id)

	var tagsInfo *structures.FileInfo
	for i := 0; i < len(p.fileList); i++ {
		packedFile := &p.fileList[i]
		if packedFile.ResPath == tagsResPath {
			tagsInfo = packedFile
			break
		}
	}

	if tagsInfo == nil {
		p.log.Info("no default.dungeondraft_tags file in pack")
		return nil
	}

	tagsBytes, err := p.readPackedFileFromPackage(r, tagsInfo)
	if err != nil {
		p.log.WithError(err).WithField("res", tagsResPath).Error("failed to read tags file")
		return errors.Join(err, ErrTagsRead)
	}

	err = json.Unmarshal(tagsBytes, &p.tags)
	if err != nil {
		p.log.WithError(err).WithField("res", tagsResPath).Error("failed to parse tags file")
		return errors.Join(err, ErrTagsParse)
	}

	return nil
}

func (p *Package) loadPackedResourceMetadata(r io.ReadSeeker) error {
	if p.id == "" {
		return ErrUnsetPackID
	}

	for i := 0; i < len(p.fileList); i++ {
		info := &p.fileList[i]

		if !info.IsWallData() && !info.IsTilesetData() {
			continue
		}

		fileData, err := p.readPackedFileFromPackage(r, info)
		if err != nil {
			p.log.WithError(err).WithField("res", info.ResPath).Error("failed to read data file")
			return errors.Join(err, ErrMetadataRead, fmt.Errorf("failed to read data file %s", info.ResPath))
		}

		if info.IsWallData() {
			wall := structures.NewPackageWall()
			err = json.Unmarshal(fileData, wall)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, ErrWallParse, fmt.Errorf("failed to parse data file %s", info.ResPath))
			}
			p.walls[info.ResPath] = *wall
		} else if info.IsTilesetData() {
			ts := structures.NewPackageTileset()
			err = json.Unmarshal(fileData, ts)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, ErrTilesetParse, fmt.Errorf("failed to parse data file %s", info.ResPath))
			}
			p.tilesets[info.ResPath] = *ts
		}
	}

	return nil
}

func (p *Package) loadUnpackedTags() error {
	if p.unpackedPath == "" {
		return ErrUnsetUnpackedPath
	}

	tagsPath := filepath.Join(p.unpackedPath, "data", "default.dungeondraft_tags")

	if tagsExist := utils.FileExists(tagsPath); !tagsExist {
		p.log.WithField("tagsPath", tagsPath).Info("no default.dungeondraft_tags file in pack")
		return nil
	}

	tagsBytes, err := os.ReadFile(tagsPath)
	if err != nil {
		p.log.WithError(err).
			WithField("path", p.unpackedPath).
			WithField("tagsPath", tagsPath).
			Error("can't read tags file")
		return errors.Join(err, ErrTagsRead)
	}

	err = json.Unmarshal(tagsBytes, &p.tags)
	if err != nil {
		p.log.WithError(err).WithField("tagsPath", tagsPath).Error("failed to parse tags file")
		return errors.Join(err, ErrTagsParse)
	}

	return nil
}

func (p *Package) SaveUnpackedTags() error {
	if p.mode != PackageModeUnpacked {
		return ErrPackageNotUnpacked
	}
	tagsPath := filepath.Join(p.unpackedPath, "data", "default.dungeondraft_tags")

	l := p.log.
		WithField("path", p.unpackedPath).
		WithField("tagsPath", tagsPath)
	tagsBytes, err := json.MarshalIndent(&p.tags, "", "  ")
	if err != nil {
		l.WithError(err).
			Error("can't save tags file")
		return errors.Join(err, ErrTagsWrite)
	}

	err = os.MkdirAll(filepath.Dir(tagsPath), 0o777)
	if err != nil {
		l.WithError(err).Error("can't save wall data")
		return errors.Join(err, ErrTagsWrite)
	}

	err = os.WriteFile(tagsPath, tagsBytes, 0o644)
	if err != nil {
		l.WithError(err).
			Error("can't save tags file")
		return errors.Join(err, ErrTagsWrite)
	}
	return nil
}

func (p *Package) SaveUnpackedWall(resPath string) error {
	if p.mode != PackageModeUnpacked {
		return ErrPackageNotUnpacked
	}

	data, ok := p.walls[resPath]
	if !ok {
		return nil
	}

	p.log.Infof("%s normalised to %s", resPath, p.NormalizeResourcePath(resPath))

	wallDataPath := filepath.Join(p.unpackedPath, p.NormalizeResourcePath(resPath))

	l := p.log.WithField("res", wallDataPath)
	l.Info("saving wall")

	wallBytes, err := json.MarshalIndent(&data, "", "  ")
	if err != nil {
		l.WithError(err).Error("can't save wall data")
		return errors.Join(err, ErrWallSave)
	}

	err = os.MkdirAll(filepath.Dir(wallDataPath), 0o777)
	if err != nil {
		l.WithError(err).Error("can't save wall data")
		return errors.Join(err, ErrWallSave)
	}

	err = os.WriteFile(wallDataPath, wallBytes, 0o644)
	if err != nil {
		l.WithError(err).Error("can't save wall data")
		return errors.Join(err, ErrWallSave)
	}

	return nil
}

func (p *Package) SaveUnpackedTileset(resPath string) error {
	if p.mode != PackageModeUnpacked {
		return ErrPackageNotUnpacked
	}

	data, ok := p.tilesets[resPath]
	if !ok {
		return nil
	}

	tilesetDataPath := filepath.Join(p.unpackedPath, p.NormalizeResourcePath(resPath))

	l := p.log.WithField("res", tilesetDataPath)
	l.Info("saving tileset")

	tilesetBytes, err := json.MarshalIndent(&data, "", "  ")
	if err != nil {
		l.WithError(err).Error("can't save wall data")
		return errors.Join(err, ErrTilesetSave)
	}

	err = os.MkdirAll(filepath.Dir(tilesetDataPath), 0o777)
	if err != nil {
		l.WithError(err).Error("can't save wall data")
		return errors.Join(err, ErrTilesetSave)
	}

	err = os.WriteFile(tilesetDataPath, tilesetBytes, 0o644)
	if err != nil {
		l.WithError(err).Error("can't save wall data")
		return errors.Join(err, ErrTilesetSave)
	}

	return nil
}

func (p *Package) loadUnpackedResourceMetadata() error {
	if p.unpackedPath == "" {
		return ErrUnsetUnpackedPath
	}

	for i := 0; i < len(p.fileList); i++ {
		info := &p.fileList[i]

		if !info.IsWallData() && !info.IsTilesetData() {
			continue
		}

		fileData, err := os.ReadFile(info.Path)
		if err != nil {
			p.log.WithError(err).WithField("res", info.ResPath).Error("failed to read data file")
			return errors.Join(err, ErrMetadataRead, fmt.Errorf("failed to read data file %s", info.Path))
		}

		if info.IsWallData() {
			wall := structures.NewPackageWall()
			err = json.Unmarshal(fileData, wall)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, ErrWallParse, fmt.Errorf("failed to parse data file %s", info.Path))
			}
			p.walls[info.ResPath] = *wall
		} else if info.IsTilesetData() {
			ts := structures.NewPackageTileset()
			err = json.Unmarshal(fileData, ts)
			if err != nil {
				p.log.WithError(err).WithField("res", info.ResPath).Error("failed to parse data file")
				return errors.Join(err, ErrTilesetParse, fmt.Errorf("failed to parse data file %s", info.Path))
			}
			p.tilesets[info.ResPath] = *ts
		}
	}

	return nil
}

func (p *Package) WriteUnpackedTags() error {
	if p.unpackedPath == "" {
		return ErrUnsetUnpackedPath
	}

	tagsPath := filepath.Join(p.unpackedPath, "data", "default.dungeondraft_tags")
	dirPath := filepath.Dir(tagsPath)

	if dirExists := utils.DirExists(dirPath); !dirExists {
		err := os.MkdirAll(dirPath, 0o777)
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed to make directory %s", dirPath))
		}
	}

	tagsBytes, err := json.MarshalIndent(&p.tags, "", "  ")
	if err != nil {
		p.log.WithError(err).
			Error("failed to create tags json")
		return errors.Join(err, errors.New("failed to create tags json"))
	}

	err = os.WriteFile(tagsPath, tagsBytes, 0o644)
	if err != nil {
		p.log.WithError(err).
			Error("failed to write tags file")
		return errors.Join(err, errors.New("failed to write tags file"))
	}

	return nil
}

func (p *Package) WriteResourceMetadata() error {
	if p.unpackedPath == "" {
		return ErrUnsetUnpackedPath
	}

	for i := 0; i < len(p.fileList); i++ {
		info := &p.fileList[i]

		if !info.IsWallData() && !info.IsTilesetData() {
			continue
		}

		dirPath := filepath.Dir(info.Path)

		if dirExists := utils.DirExists(dirPath); !dirExists {
			err := os.MkdirAll(dirPath, 0o777)
			if err != nil {
				return errors.Join(err, fmt.Errorf("failed to make directory %s", dirPath))
			}
		}

		var fileBytes []byte
		var err error
		if info.IsWallData() {
			wall := p.walls[info.ResPath]
			fileBytes, err = json.MarshalIndent(&wall, "", "  ")
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.ResPath).
					Error("failed to create wall json")
				return errors.Join(err, fmt.Errorf("failed to create wall json for %s", info.ResPath))
			}
		} else {
			tileset := p.tilesets[info.ResPath]
			fileBytes, err = json.MarshalIndent(&tileset, "", "  ")
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.ResPath).
					Error("failed to create tileset json")
				return errors.Join(err, fmt.Errorf("failed to create tileset json for %s", info.ResPath))
			}
		}

		if len(fileBytes) > 0 {
			err = os.WriteFile(info.Path, fileBytes, 0o644)
			if err != nil {
				p.log.WithError(err).
					Error("failed to write metadata file")
				return errors.Join(err, fmt.Errorf("failed to write metadata file for %s", info.ResPath))
			}
		}
	}

	return nil
}
