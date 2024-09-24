package ddpackage

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
)

func (p *Package) GenerateThumbnails(progressCallbacks ...func(p float64)) error {
	utils.AssertTrue(p.UnpackedPath != "", "empty unpacked path")
	thumbnailDir := filepath.Join(p.UnpackedPath, "thumbnails")

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
	for i, info := range p.FileList {
		if info.Image != nil && !strings.HasPrefix(info.ResPath, thumbnailPrefix) {
			p.log.WithField("res", info.ResPath).Trace("generating thumbnail")

			var maxWidth, height int
			if strings.HasPrefix(info.ResPath, terrainPrefix) {
				maxWidth, height = 160, 160
			} else if strings.HasPrefix(info.ResPath, wallsPrefix) {
				maxWidth, height = 228, 32
			} else if strings.HasPrefix(info.ResPath, pathsPrefix) {
				maxWidth, height = 228, 48
			} else {
				maxWidth, height = 64, 64
			}

			thumbnail := ddimage.ResizeVirticalAndCropWidth(info.Image, height, maxWidth)

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
			pcb(float64(i+1) / float64(len(p.FileList)))
		}
	}

	return nil
}
