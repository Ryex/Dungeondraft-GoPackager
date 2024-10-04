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
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

func (p *Package) GenerateThumbnails() []error {
	return p.generateThumbnails(nil)
}

func (p *Package) GenerateThumbnailsProgress(progressCallback func(p float64)) []error {
	return p.generateThumbnails(progressCallback)
}

func (p *Package) generateThumbnails(progressCallback func(p float64)) []error {
	if p.unpackedPath == "" {
		return []error{ErrUnsetUnpackedPath}
	}
	thumbnailDir := filepath.Join(p.unpackedPath, "thumbnails")

	if dirExists := utils.DirExists(thumbnailDir); !dirExists {
		err := os.MkdirAll(thumbnailDir, 0o777)
		if err != nil {
			return []error{errors.Join(err, fmt.Errorf("failed to create thumbnail directory %s", thumbnailDir))}
		}
	}

	var texCount float64

	for _, info := range p.fileList {
		if info.IsTexture() {
			texCount += 1
		}
	}

	type result struct {
		Err      error
		Resource string
	}

	makeThumb := func(fi *structures.FileInfo, l logrus.FieldLogger, ch chan result) {
		var img image.Image
		var err error
		if fi.Image == nil {
			img, _, err = ddimage.OpenImage(fi.Path)
			if err != nil {
				err = errors.Join(err, fmt.Errorf("failed to open %s as an image", fi.Path))
				ch <- result{err, fi.ResPath}
				return
			}
		} else {
			img = fi.Image
		}

		var thumbnail image.Image

		if fi.IsTerrain() {
			thumbnail = ddimage.TerrainThumbnail(img)
		} else if fi.IsWall() {
			thumbnail = ddimage.WallThumbnail(img)
		} else if fi.IsPath() {
			thumbnail = ddimage.PathThumbnail(img)
		} else {
			thumbnail = ddimage.DefaultThumbnail(img)
		}

		file, err := os.OpenFile(
			fi.ThumbnailPath,
			os.O_RDWR|os.O_CREATE,
			0o644,
		)
		if err != nil {
			l.WithError(err).
				WithField("thumbnail", fi.ThumbnailPath).
				Error("failed to open thumbnail file for writing")
			err = errors.Join(
				err,
				fmt.Errorf("failed to open thumbnail file %s for writing", fi.ThumbnailPath),
				fmt.Errorf("failed generate thumbnail for %s", fi.RelPath),
			)
			ch <- result{err, fi.ResPath}
			return
		}

		err = png.Encode(file, thumbnail)
		if err != nil {
			l.WithError(err).
				WithField("thumbnail", fi.ThumbnailPath).
				Error("failed to encode thumbnail png")
			err = errors.Join(
				err,
				fmt.Errorf("failed to encode thumbnail png"),
				fmt.Errorf("failed generate thumbnail for %s", fi.RelPath),
			)
			ch <- result{err, fi.ResPath}
			return
		}
		ch <- result{nil, fi.ResPath}
	}

	ch := make(chan result)

	p.flLock.RLock()
	defer p.flLock.RUnlock()

	for _, fi := range p.fileList {
		if fi.IsTexture() {
			go makeThumb(fi, p.log.WithField("res", fi.ResPath), ch)
		}
	}

	var thumbCount float64
	var errs []error

	for i := 0; i < int(texCount); i++ {
		r := <-ch
		if r.Err != nil {
			errs = append(errs, r.Err)
		}
		thumbCount += 1
		if progressCallback != nil {
			progressCallback(thumbCount / texCount)
		}
		p.log.WithField("res", r.Resource).Trace("thumbnail generated")
	}

	return errs
}
