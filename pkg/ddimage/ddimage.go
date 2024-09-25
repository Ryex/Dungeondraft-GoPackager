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
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	// decode webp format
	// missing important alpha channel support https://github.com/golang/go/issues/60437
	// _ "golang.org/x/image/webp"

	// uses libwebp >=1.0.3
	_ "github.com/chai2010/webp"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/sunshineplan/imgconv"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func OpenImage(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", errors.Join(err, fmt.Errorf("cannot open file: %s", path))
	}
	defer file.Close()

	if filepath.Ext(path) == ".svg" {
		img, err := ReadSvg(file)
		if err != nil {
			return nil, "", err
		}
		return img, "svg", nil
	}

	return image.Decode(file)
}

var InvalidSVGError = errors.New("invalid svg")

func ReadSvg(r io.Reader) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(r)
	if err != nil {
		return nil, errors.Join(err, InvalidSVGError)
	}
	return SvgToImage(icon)
}

func SvgBytesToImage(byts []byte) (image.Image, error) {
	return ReadSvg(bytes.NewReader(byts))
}

func SvgToImage(icon *oksvg.SvgIcon) (image.Image, error) {
	w := icon.ViewBox.W
	h := icon.ViewBox.H
	rgba := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	icon.Draw(
		rasterx.NewDasher(
			int(w), int(h),
			rasterx.NewScannerGV(
				int(w), int(h),
				rgba, rgba.Bounds(),
			),
		),
		1,
	)

	return rgba, nil
}

func PngImageBytes(img image.Image, buf *bytes.Buffer) (err error) {
	w := bufio.NewWriter(buf)
	err = png.Encode(w, img)
	return
}

func BytesToImage(byts []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(byts))
	return img, err
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
	return utils.InSlice(ext, []string{
		".jpg", ".jpeg", ".png", ".webp", ".gif", ".tif", ".tiff", ".bmp", ".svg",
	})
}

func PathIsSupportedDDImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return utils.InSlice(ext, []string{
		".jpg", ".jpeg", ".png", ".webp", ".bmp", ".svg",
	})
}
