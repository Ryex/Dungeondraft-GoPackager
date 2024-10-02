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
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	// decode webp format
	// missing important alpha channel support https://github.com/golang/go/issues/60437
	// gowebp "golang.org/x/image/webp"

	libwebp_decoder "github.com/kolesa-team/go-webp/decoder"
	libwebp "github.com/kolesa-team/go-webp/webp"

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

	ext := filepath.Ext(path)
	switch ext {
	case ".svg":
		img, err := ReadSvg(file)
		if err != nil {
			return nil, "", err
		}
		return img, "svg", nil
	case ".webp":
		libimg, liberr := libwebp.Decode(file, &libwebp_decoder.Options{})
		if liberr != nil {
			return nil, "", liberr
		}
		return libimg, "webp", nil

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

type Model string

const (
	ModelRGBA    = "rgba"
	ModelRGBA64  = "rgba64"
	ModelNRGBA   = "nrgba"
	ModelNRGBA64 = "nrgba64"
	ModelAlpha   = "alpha"
	ModelAlpha16 = "alpha16"
	ModelGray    = "gray"
	ModelGray16  = "gray16"
	ModelYCbCr   = "ycbcr"
	ModelNYCbCr  = "nycbcra"
	ModelCMYK    = "cmyk"
	ModelUnknown = "unknown"
)

func ColorModel(img image.Image) Model {
	switch img.ColorModel() {
	case color.RGBAModel:
		return ModelRGBA
	case color.RGBA64Model:
		return ModelRGBA64
	case color.NRGBAModel:
		return ModelNRGBA
	case color.NRGBA64Model:
		return ModelNRGBA64
	case color.AlphaModel:
		return ModelAlpha
	case color.Alpha16Model:
		return ModelAlpha16
	case color.GrayModel:
		return ModelGray
	case color.Gray16Model:
		return ModelGray16
	case color.YCbCrModel:
		return ModelYCbCr
	case color.NYCbCrAModel:
		return ModelNYCbCr
	case color.CMYKModel:
		return ModelCMYK
	default:
		return ModelUnknown
	}
}

func ForceNRGBA(img image.Image) image.Image {
	ni := image.NewNRGBA(img.Bounds())
	if model := ColorModel(img); model != ModelNRGBA {
		for x := 0; x < img.Bounds().Dx(); x++ {
			for y := 0; y < img.Bounds().Dy(); y++ {
				pix := img.At(x, y)
				ni.Set(x, y, color.NRGBAModel.Convert(pix))
			}
		}
	}
	return img
}

func PngImageBytes(img image.Image, buf *bytes.Buffer) (err error) {
	w := bufio.NewWriter(buf)
	err = png.Encode(w, img)
	return
}

func BytesToImage(byts []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(byts))
}

func Resize(img image.Image, width, height int) image.Image {
	if width == 0 && height == 0 {
		return img
	}
	if width == 0 {
		w := float64(height) * float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
		width = int(math.Max(1.0, math.Floor(w+0.5)))
	}
	if height == 0 {
		h := float64(width) * float64(img.Bounds().Dy()) / float64(img.Bounds().Dx())
		width = int(math.Max(1.0, math.Floor(h+0.5)))
	}
	return resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
}

func Crop(img image.Image, rect image.Rectangle) image.Image {
	imgSub, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if ok {
		return imgSub.SubImage(rect)
	}
	return img
}

// func Square(img image.Image) image.Image {}

func ResizeVirticalAndCropWidth(img image.Image, height, maxWidth int) image.Image {
	// resized := imgconv.Resize(img, &imgconv.ResizeOption{Width: 0, Height: height})
	resized := Resize(img, maxWidth, height)
	resizedWidth := resized.Bounds().Dx()
	if resizedWidth > maxWidth {
		start := (resizedWidth - maxWidth) / 2
		resized = resized.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(image.Rect(start, 0, start+maxWidth, height))
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
