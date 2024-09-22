package ddimage

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp" // decode webp format

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/sunshineplan/imgconv"
)

type Format int

const (
	JPEG Format = iota
	PNG
	GIT
	TIFF
	BMP
	WEBP
)

var formatExts = [][]string{
	{"jpg", "jpeg"},
	{"png"},
	{"gif"},
	{"tif", "tiff"},
	{"bmp"},
	{"webp"},
}

func (f Format) String() (format string) {
	defer func() {
		if err := recover(); err != nil {
			format = "unknown"
		}
	}()
	return formatExts[f][0]
}

func FormatFromExtension(ext string) (Format, error) {
	ext = strings.ToLower(ext)
	for index, exts := range formatExts {
		for _, i := range exts {
			if ext == i {
				return Format(index), nil
			}
		}
	}

	return -1, image.ErrFormat
}

func OpenImage(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", errors.Join(err, fmt.Errorf("cannot open file: %s", path))
	}
	defer file.Close()

	return image.Decode(file)
}

func PngImageBytes(img image.Image, buf *bytes.Buffer) (err error) {
	w := bufio.NewWriter(buf)
	err = png.Encode(w, img)
	return
}

func ResizeVirticalAndCropWidth(img image.Image, height int, width int) image.Image {
	resized := imgconv.Resize(img, &imgconv.ResizeOption{Width: 0, Height: height})
	resizedWidth := resized.Bounds().Dx()
	if resizedWidth > width {
		resized = resized.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(image.Rect(0, 0, width, height))
	}
	return resized
}

func PathIsSupportedImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return utils.StringInSlice(ext, []string{
		".jpg", ".jpeg", ".png", ".webp", ".gif", ".tif", ".tiff", ".bmp",
	})
}
