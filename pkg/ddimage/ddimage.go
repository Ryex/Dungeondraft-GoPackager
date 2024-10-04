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

	// libwebp_decoder "github.com/kolesa-team/go-webp/decoder"
	// libwebp "github.com/kolesa-team/go-webp/webp"

	// libwebp "github.com/bep/gowebp/libwebp"
	"github.com/ryex/gowebp/libwebp"
	"github.com/ryex/gowebp/libwebp/webpoptions"

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
		img, err := libwebp.Decode(file, webpoptions.DecodingOptions{})
		if err != nil {
			return nil, "", err
		}
		return img, "webp", nil

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

func ToNRGBA(img image.Image) image.Image {
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

func ToNRGBA64(img image.Image) image.Image {
	ni := image.NewNRGBA64(img.Bounds())
	if model := ColorModel(img); model != ModelNRGBA64 {
		for x := 0; x < img.Bounds().Dx(); x++ {
			for y := 0; y < img.Bounds().Dy(); y++ {
				pix := img.At(x, y)
				ni.Set(x, y, color.NRGBA64Model.Convert(pix))
			}
		}
	}
	return img
}

func StripAlpha(img image.Image) image.Image {
	ni := image.NewNRGBA(img.Bounds())
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			pix := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			pix.A = 255
			ni.Set(x, y, pix)
		}
	}
	return ni
}

func TranparentBounds(img image.Image, threshold uint8) image.Rectangle {
	bounds := img.Bounds()
	tBounds := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)

	// from the left
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		colAlphaMax := uint8(0)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			nrgba := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if nrgba.A > colAlphaMax {
				colAlphaMax = nrgba.A
			}
		}
		if colAlphaMax >= threshold {
			tBounds.Min.X = x
			// fmt.Println("found left", tBounds.Min.X)
			break
		}
	}
	// from the right
	for x := bounds.Max.X - 1; x >= bounds.Min.X; x-- {
		colAlphaMax := uint8(0)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			nrgba := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if nrgba.A > colAlphaMax {
				colAlphaMax = nrgba.A
			}
		}
		if colAlphaMax >= threshold {
			tBounds.Max.X = x
			// fmt.Println("found right", tBounds.Max.X)
			break
		}
	}
	// from the top
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		lineAlphaMax := uint8(0)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if nrgba.A > lineAlphaMax {
				lineAlphaMax = nrgba.A
			}
		}
		if lineAlphaMax >= threshold {
			tBounds.Min.Y = y
			// fmt.Println("found top", tBounds.Min.Y)
			break
		}
	}
	// from the bottom
	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		lineAlphaMax := uint8(0)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if nrgba.A > lineAlphaMax {
				lineAlphaMax = nrgba.A
			}
		}
		if lineAlphaMax >= threshold {
			tBounds.Max.Y = y
			// fmt.Println("found bottom", tBounds.Max.Y)
			break
		}
	}
	return tBounds
}

func PngImageBytes(img image.Image, buf *bytes.Buffer) (err error) {
	w := bufio.NewWriter(buf)
	enc := &png.Encoder{
		CompressionLevel: png.NoCompression,
	}
	err = enc.Encode(w, img)
	w.Flush()
	return
}

func PngDecodeBytes(byts []byte) (image.Image, error) {
	return png.Decode(bytes.NewReader(byts))
}

func BytesToImage(byts []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(byts))
}

var (
	ResizeLancos2           = resize.Lanczos2
	ResizeLancos3           = resize.Lanczos3
	ResizeBicubic           = resize.Bicubic
	ResizeBilinear          = resize.Bilinear
	ResizeNearestNeighbor   = resize.NearestNeighbor
	ResizeMitchellNetravali = resize.MitchellNetravali
)

func Resize(img image.Image, width, height int, ifunc resize.InterpolationFunction) image.Image {
	if width == 0 && height == 0 {
		return img
	}
	if width == 0 {
		w := float64(height) * (float64(img.Bounds().Dx()) / float64(img.Bounds().Dy()))
		width = int(math.Max(1.0, math.Floor(w+0.5)))
	}
	if height == 0 {
		h := float64(width) * (float64(img.Bounds().Dy()) / float64(img.Bounds().Dx()))
		width = int(math.Max(1.0, math.Floor(h+0.5)))
	}
	return resize.Resize(uint(width), uint(height), img, ifunc)
}

func Crop(img image.Image, rect image.Rectangle) image.Image {
	ni := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for x, i := rect.Min.X, 0; x < rect.Max.X; x, i = x+1, i+1 {
		for y, j := rect.Min.Y, 0; y < rect.Max.Y; y, j = y+1, j+1 {
			ni.Set(i, j, color.NRGBAModel.Convert(img.At(x, y)))
		}
	}
	return ni
}

func CropTransparent(img image.Image, minWidth, minHeight int, threshold uint8) image.Image {
	bounds := TranparentBounds(img, threshold)
	if minWidth > 0 && bounds.Dx() < minWidth {
		middle := bounds.Min.X + bounds.Dx()/2
		bounds.Min.X = max(img.Bounds().Min.X, middle-minWidth/2)
		bounds.Max.X = min(img.Bounds().Max.X, bounds.Min.X+minWidth)
	}
	if minHeight > 0 && bounds.Dy() < minHeight {
		middle := bounds.Min.Y + bounds.Dy()/2
		bounds.Min.Y = max(img.Bounds().Min.Y, middle-minHeight/2)
		bounds.Max.Y = min(img.Bounds().Max.Y, bounds.Min.Y+minHeight)
	}
	return Crop(img, bounds)
}

// func Square(img image.Image) image.Image {}

func ResizeVirticalAndCropWidth(img image.Image, height, maxWidth int, ifunc resize.InterpolationFunction) image.Image {
	var resized image.Image
	if img.Bounds().Dy() == height {
		resized = img
	} else {
		resized = Resize(img, 0, height, ifunc)
	}
	bounds := resized.Bounds()
	resizedWidth := bounds.Dx()
	if resizedWidth > maxWidth {
		return Crop(resized, image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+maxWidth, bounds.Max.Y))
	}
	return resized
}

func TerrainThumbnail(img image.Image) image.Image {
	return StripAlpha(Resize(img, 0, 160, ResizeBicubic))
}

func WallThumbnail(img image.Image) image.Image {
	return ResizeVirticalAndCropWidth(CropTransparent(img, 0, 0, 5), 32, 228, ResizeBicubic)
}

func PathThumbnail(img image.Image) image.Image {
	return ResizeVirticalAndCropWidth(CropTransparent(img, 0, 0, 5), 48, 228, ResizeBicubic)
}

func DefaultThumbnail(img image.Image) image.Image {
	return Resize(img, 0, 64, ResizeBicubic)
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
