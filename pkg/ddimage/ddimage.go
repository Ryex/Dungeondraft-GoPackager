package ddimage

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	// decode webp format
	// missing important alpha channel support https://github.com/golang/go/issues/60437
	// _ "golang.org/x/image/webp"

	// uses libwebp >=1.0.3
	// replace with
	// github.com/chirino/webp@8b3bed1ecc92085133c77728637009906734715f
	// for webp 1.4.0
	_ "github.com/chai2010/webp"

	// "github.com/sunshineplan/imgconv"
	"github.com/nfnt/resize"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func unmultiplyAlpha(c color.Color) (r, g, b, a int) {
	red, green, blue, alpha := c.RGBA()
	if alpha != 0 && alpha != 0xffff {
		red = (red * 0xffff) / alpha
		green = (green * 0xffff) / alpha
		blue = (blue * 0xffff) / alpha
	}
	// Convert from range 0-65535 to range 0-255
	r = int(red >> 8)
	g = int(green >> 8)
	b = int(blue >> 8)
	a = int(alpha >> 8)
	return
}

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}
type Integer interface {
	Signed | Unsigned
}

func clamp[T Integer](v, m, mx T) T {
	if v < m {
		return m
	}
	if v > mx {
		return mx
	}
	return v
}

func ConvertRGBAToNRGBA(c color.RGBA) color.NRGBA {
	r, g, b, a := c.R, c.G, c.B, c.A
	if a != 0 && a != 0xff {
		r = (r * 0xff) / a
		g = (g * 0xff) / a
		b = (b * 0xff) / a
	}
	return color.NRGBA{
		R: r,
		G: g,
		B: b,
		A: a,
	}
}

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

var ErrInvalidSVG = errors.New("invalid svg")

func ReadSvg(r io.Reader) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(r)
	if err != nil {
		return nil, errors.Join(err, ErrInvalidSVG)
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

func BytesToImage(byts []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(byts))
}

func ResizeVirticalAndCropWidth(img image.Image, height int, width int) image.Image {
	// resized := imgconv.Resize(img, &imgconv.ResizeOption{Width: 0, Height: height})
	resized := resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
	resizedWidth := resized.Bounds().Dx()
	if resizedWidth > width {
		start := (resizedWidth - width) / 2
		resized = resized.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(image.Rect(start, 0, start+width, height))
	}
	return resized
}

func PathIsSupportedImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains([]string{
		".jpg", ".jpeg", ".png", ".webp", ".gif", ".tif", ".tiff", ".bmp", ".svg",
	}, ext)
}

func PathIsSupportedDDImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains([]string{
		".jpg", ".jpeg", ".png", ".webp", ".bmp", ".svg",
	}, ext)
}
