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
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp" // decode webp format

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/sunshineplan/imgconv"
)

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
		".jpg", ".jpeg", ".png", ".webp", ".gif", ".tif", ".tiff", ".bmp",
	})
}
