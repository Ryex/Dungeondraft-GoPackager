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

func (p *Package) GenerateThumbnails() error {
	return p.generateThumbnails(nil)
}

func (p *Package) GenerateThumbnailsProgress(progressCallback func(p float64)) error {
	return p.generateThumbnails(progressCallback)
}

func (p *Package) generateThumbnails(progressCallback func(p float64)) error {
	if p.unpackedPath == "" {
		return ErrUnsetUnpackedPath
	}
	thumbnailDir := filepath.Join(p.unpackedPath, "thumbnails")

	if dirExists := utils.DirExists(thumbnailDir); !dirExists {
		err := os.MkdirAll(thumbnailDir, 0o777)
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed to create thumbnail directory %s", thumbnailDir))
		}
	}

	var texCount float64

	for _, info := range p.fileList {
		if info.IsTexture() {
			texCount += 1
		}
	}

	var thumbCount float64
	for _, info := range p.fileList {
		if info.IsTexture() {
			p.log.WithField("res", info.ResPath).Trace("generating thumbnail")

			var img image.Image
			var err error
			if info.Image == nil {
				img, _, err = ddimage.OpenImage(info.Path)
				if err != nil {
					err = errors.Join(err, fmt.Errorf("failed to open %s as an image", info.Path))
					return err
				}
			} else {
				img = info.Image
			}

			var thumbnail image.Image

			if info.IsTerrain() {
				thumbnail = ddimage.Resize(img, 0, 160)
			} else if info.IsWall() {
				thumbnail = ddimage.ResizeVirticalAndCropWidth(img, 32, 228)
			} else if info.IsPath() {
				thumbnail = ddimage.ResizeVirticalAndCropWidth(img, 48, 228)
			} else {
				thumbnail = ddimage.Resize(img, 0, 64)
			}

			file, err := os.OpenFile(
				info.ThumbnailPath,
				os.O_RDWR|os.O_CREATE,
				0o644,
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

			thumbCount += 1
			if progressCallback != nil {
				progressCallback(thumbCount / texCount)
			}
		}
	}

	return nil
}
