package ddimage

import (
	"image"
	"image/color"
	"testing"
)

var (
	testRed   = color.NRGBA{R: 0xff, A: 0xff}
	testGreen = color.NRGBA{G: 0xff, A: 0xff}
	testGray  = color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
)

// newTestImage builds a w x h NRGBA image filled with fill(x, y).
func newTestImage(w, h int, fill func(x, y int) color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, fill(x, y))
		}
	}
	return img
}

// solidImage builds a w x h image of a single opaque color.
func solidImage(w, h int, c color.NRGBA) *image.NRGBA {
	return newTestImage(w, h, func(_, _ int) color.NRGBA { return c })
}

func checkSize(t *testing.T, img image.Image, wantW, wantH int) {
	t.Helper()
	b := img.Bounds()
	if b.Dx() != wantW || b.Dy() != wantH {
		t.Fatalf("size = %dx%d, want %dx%d", b.Dx(), b.Dy(), wantW, wantH)
	}
}

func checkPixel(t *testing.T, img image.Image, x, y int, want color.NRGBA) {
	t.Helper()
	if got := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA); got != want {
		t.Errorf("pixel (%d, %d) = %v, want %v", x, y, got, want)
	}
}

// checkRed checks that a pixel is dominated by red, tolerating the rounding of
// a bicubic resize.
func checkRed(t *testing.T, img image.Image, x, y int) {
	t.Helper()
	c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	if c.R < 0xf0 || c.G > 0x10 || c.B > 0x10 {
		t.Errorf("pixel (%d, %d) = %v, want red", x, y, c)
	}
}

// checkGray checks that a pixel is gray, tolerating the rounding of a bicubic
// resize.
func checkGray(t *testing.T, img image.Image, x, y int) {
	t.Helper()
	c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	if c.R < 0x30 || c.R > 0x50 || c.G != c.R || c.B != c.R {
		t.Errorf("pixel (%d, %d) = %v, want gray", x, y, c)
	}
}

func TestRepeatHorizontallyExactMultiple(t *testing.T) {
	src := newTestImage(64, 8, func(x, _ int) color.NRGBA {
		switch {
		case x == 0:
			return testRed
		case x == 63:
			return testGreen
		default:
			return testGray
		}
	})

	got := RepeatHorizontally(src, 192)

	checkSize(t, got, 192, 8)
	checkPixel(t, got, 0, 0, testRed)     // start of the first tile
	checkPixel(t, got, 64, 0, testRed)    // start of the second tile
	checkPixel(t, got, 127, 0, testGreen) // end of the second tile
	checkPixel(t, got, 128, 0, testRed)   // start of the third tile
	checkPixel(t, got, 191, 0, testGreen) // end of the third tile
	checkPixel(t, got, 30, 0, testGray)
}

func TestRepeatHorizontallyClipsLastTile(t *testing.T) {
	src := newTestImage(100, 8, func(x, _ int) color.NRGBA {
		if x == 0 {
			return testRed
		}
		return testGray
	})

	got := RepeatHorizontally(src, 256)

	// the result stops at the target width, the last tile is clipped
	checkSize(t, got, 256, 8)
	checkPixel(t, got, 0, 0, testRed)
	checkPixel(t, got, 100, 0, testRed)
	checkPixel(t, got, 200, 0, testRed)
	checkPixel(t, got, 255, 0, testGray) // source column 55 of the clipped tile
}

func TestRepeatHorizontallyWideEnoughIsUnchanged(t *testing.T) {
	src := solidImage(300, 8, testGray)

	if got := RepeatHorizontally(src, 256); got != src {
		t.Error("an image that is already wide enough must be returned unchanged")
	}
}

func TestWallThumbnailRepeatsShortSeamlessTexture(t *testing.T) {
	// 100x33 has an aspect ratio of ~3.03, smaller than the thumbnail box
	// 228/32 (= 7.125), so the texture is assumed to be seamless and
	// repeated to a target width of ceil(33*228/32) = 236. The height matches
	// wallThumbnailHeight after the transparent crop trims one row, so no
	// vertical resize happens and the repeated marker columns stay exact.
	src := newTestImage(100, wallThumbnailHeight+1, func(x, _ int) color.NRGBA {
		if x == 0 {
			return testRed
		}
		return testGray
	})

	got := WallThumbnail(src)

	checkSize(t, got, maxThumbnailWidth, wallThumbnailHeight)
	checkPixel(t, got, 0, 16, testRed)   // start of the first repeat
	checkPixel(t, got, 100, 16, testRed) // start of the second repeat
	checkPixel(t, got, 200, 16, testRed) // start of the third repeat
	checkPixel(t, got, 50, 16, testGray)
	checkPixel(t, got, 227, 16, testGray) // filled edge to edge
}

func TestPathThumbnailRepeatsShortSeamlessTexture(t *testing.T) {
	// 100x49 has an aspect ratio of ~2.04, smaller than the thumbnail box
	// 228/48 (= 4.75), so the texture is repeated to a target width of
	// ceil(49*228/48) = 233.
	src := newTestImage(100, pathThumbnailHeight+1, func(x, _ int) color.NRGBA {
		if x == 0 {
			return testRed
		}
		return testGray
	})

	got := PathThumbnail(src)

	checkSize(t, got, maxThumbnailWidth, pathThumbnailHeight)
	checkPixel(t, got, 0, 24, testRed)
	checkPixel(t, got, 100, 24, testRed)
	checkPixel(t, got, 200, 24, testRed)
	checkPixel(t, got, 50, 24, testGray)
	checkPixel(t, got, 227, 24, testGray)
}

func TestWallThumbnailTallTextureIsRepeated(t *testing.T) {
	// 300x64 is wider than 228 but its aspect ratio of 4.69 is still below
	// the thumbnail box 228/32, so it is repeated to
	// ceil(64*228/32) = 456 wide. The thumbnail then fills the full 228px
	// width, with the repeats visible in it.
	src := newTestImage(300, 64, func(x, _ int) color.NRGBA {
		if x < 150 {
			return testRed
		}
		return testGray
	})

	got := WallThumbnail(src)

	checkSize(t, got, maxThumbnailWidth, wallThumbnailHeight)
	checkRed(t, got, 30, 16)   // first tile
	checkGray(t, got, 110, 16) // first tile
	checkRed(t, got, 200, 16)  // second tile: proof that the texture repeated
}

func TestWallThumbnailWideTextureIsNotRepeated(t *testing.T) {
	// an aspect ratio at or above the thumbnail box keeps the old behavior:
	// crop the transparent bounds, resize and crop the width
	src := newTestImage(512, wallThumbnailHeight+1, func(x, _ int) color.NRGBA {
		if x == 0 {
			return testRed
		}
		return testGray
	})

	got := WallThumbnail(src)

	checkSize(t, got, maxThumbnailWidth, wallThumbnailHeight)
	checkPixel(t, got, 0, 16, testRed)
	checkPixel(t, got, 100, 16, testGray) // no repeat of the marker column
	checkPixel(t, got, 227, 16, testGray)
}

func TestWallEndThumbnailKeepsAspectRatio(t *testing.T) {
	// 96x64: cropped to its transparent bounds (95x63) and fitted to
	// endThumbnailSize height, the width follows the aspect ratio:
	// 48 * 95 / 63 = 72
	got := WallEndThumbnail(solidImage(96, 64, testRed))

	checkSize(t, got, 72, endThumbnailSize)
	if c := color.NRGBAModel.Convert(got.At(36, 24)).(color.NRGBA); c.R < 0xf0 || c.A != 0xff {
		t.Errorf("center pixel = %v, want opaque red", c)
	}
}
