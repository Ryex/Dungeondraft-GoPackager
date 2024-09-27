package ddpackage

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
)

func (p *Package) GenerateThumbnails(progressCallbacks ...func(p float64)) error {
	if p.unpackedPath == "" {
		return ErrUnsetUnpackedPath
	}
	thumbnailDir := filepath.Join(p.unpackedPath, "thumbnails")

	if dirExists := utils.DirExists(thumbnailDir); !dirExists {
		err := os.MkdirAll(thumbnailDir, 0777)
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed to create thumbnail directory %s", thumbnailDir))
		}
	}

	thumbnailPrefix := fmt.Sprintf("res://packs/%s/thumbnails/", p.id)
	terrainPrefix := fmt.Sprintf("res://packs/%s/%s/textures/terrain/", p.id, p.name)
	wallsPrefix := fmt.Sprintf("res://packs/%s/%s/textures/walls/", p.id, p.name)
	pathsPrefix := fmt.Sprintf("res://packs/%s/%s/textures/paths/", p.id, p.name)
	fmt.Println(thumbnailPrefix)
	fmt.Println(terrainPrefix)
	fmt.Println(wallsPrefix)
	fmt.Println(pathsPrefix)
	for i, info := range p.fileList {
		if info.IsTexture() {
			p.log.WithField("res", info.ResPath).Trace("generating thumbnail")

			var maxWidth, height int
			if info.IsTerrain() {
				maxWidth, height = 160, 160
			} else if info.IsWall() {
				maxWidth, height = 228, 32
			} else if info.IsPath() {
				maxWidth, height = 228, 48
			} else {
				maxWidth, height = 64, 64
			}

			var image image.Image
			if info.Image == nil {
				img, _, err := ddimage.OpenImage(info.Path)
				if err != nil {
					err = errors.Join(err, fmt.Errorf("failed to open %s as an image", info.Path))
					return err
				}
				image = img
			} else {
				image = info.Image
			}

			thumbnail := ddimage.ResizeVirticalAndCropWidth(image, height, maxWidth)

			file, err := os.OpenFile(
				info.ThumbnailPath,
				os.O_RDWR|os.O_CREATE,
				0644,
			)
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.Path).
					WithField("thumbnail", info.ThumbnailPath).
					Error("failed to open thumbnail file for writing")
				return errors.Join(
					err,
					fmt.Errorf("failed to open thumbnail file %s for writing", info.ThumbnailPath),
					fmt.Errorf("failed generate thumbnail for %s", info.RelPath),
				)
			}

			err = png.Encode(file, thumbnail)
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.Path).
					WithField("thumbnail", info.ThumbnailPath).
					Error("failed to encode thumbnail png")
				return errors.Join(
					err,
					fmt.Errorf("failed to encode thumbnail png"),
					fmt.Errorf("failed generate thumbnail for %s", info.RelPath),
				)
			}
		}

		for _, pcb := range progressCallbacks {
			pcb(float64(i+1) / float64(len(p.fileList)))
		}
	}

	return nil
}
