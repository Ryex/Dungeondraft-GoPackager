package pack

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
)

func (p *Packer) GenerateThumbnails() error {
	thumbnailDir := filepath.Join(p.path, "thumbnails")

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
	for _, info := range p.FileList {
		if info.Image != nil && !strings.HasPrefix(info.ResPath, thumbnailPrefix) {
			p.log.WithField("res", info.ResPath).Trace("generating thumbnail")

			hash := md5.Sum([]byte(info.ResPath))
			thumbnailName := hex.EncodeToString(hash[:]) + ".png"
			thumbnailPath := filepath.Join(thumbnailDir, thumbnailName)

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

			thumbnail := ResizeVirticalAndCropWidth(info.Image, height, maxWidth)

			file, err := os.OpenFile(
				thumbnailPath,
				os.O_RDWR|os.O_CREATE,
				0644,
			)
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.Path).
					WithField("thumbnail", thumbnailPath).
					Error("failed to open thumbnail file for writing")
				return err
			}

			err = png.Encode(file, thumbnail)
			if err != nil {
				p.log.WithError(err).
					WithField("res", info.Path).
					WithField("thumbnail", thumbnailPath).
					Error("failed to encode thumbnail png")
				return err
			}
		}
	}

	return nil
}
