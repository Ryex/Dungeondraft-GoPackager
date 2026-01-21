package ddimage

import (
	"image/color"
	"math"
)

func wrap(a float64, d_optional ...float64) float64 {
	d := 1.0
	if len(d_optional) > 0 {
		d = d_optional[0]
	}
	r := math.Mod(a, d)
	if r < 0.0 {
		return d + r
	} else if r > 0.0 {
		return r
	} else {
		return 0.0
	}
}

func normalize(a float64) float64 {
	if a < 1.0 {
		if a > 0.0 {
			return a
		} else {
			return 0.0
		}
	} else {
		return 1.0
	}
}

var yc = [3]float64{0.2126, 0.7152, 0.0722}

func gamma(n float64) float64 {
	return math.Pow(normalize(n), 2.2)
}

func igamma(n float64) float64 {
	return math.Pow(normalize(n), 1.0/2.2)
}

func lumag(r, g, b float64) float64 {
	return r*yc[0] + g*yc[1] + b*yc[2]
}

type HCYAColor struct {
	h,
	c,
	y,
	a float64
}

func NewHCYAFromColor(col color.Color) HCYAColor {
	rgba := color.NRGBAModel.Convert(col).(color.NRGBA)
	r_int, g_int, b_int, a_int := rgba.R, rgba.G, rgba.B, rgba.A
	r := gamma(float64(r_int) / float64(0xff))
	g := gamma(float64(g_int) / float64(0xff))
	b := gamma(float64(b_int) / float64(0xff))
	a := float64(a_int) / float64(0xff)

	// luma
	y := lumag(r, g, b)

	// hue
	p := math.Max(math.Max(r, g), b)
	n := math.Max(math.Min(r, g), b)
	d := 6.0 * (p - n)
	var h float64
	if n == p {
		h = 0.0
	} else if r == p {
		h = ((g - b) / d)
	} else if g == p {
		h = ((b - r) / d) + (1.0 / 3.0)
	} else {
		h = ((r - g) / d) + (2.0 / 3.0)
	}

	var c float64
	// chroma
	if r == g && g == b {
		c = 0.0
	} else {
		c = math.Max((y-n)/y, (p-y)/(1-y))
	}

	return HCYAColor{
		h, c, y, a,
	}
}

func (hyca HCYAColor) ToNRGBA() color.NRGBA {
	// start with sane component values
	h := wrap(hyca.h)
	c := normalize(hyca.c)
	y := normalize(hyca.y)
	a := normalize(hyca.a)

	hs := h * 6.0
	var th, tm float64

	if hs < 1.0 {
		th = hs
		tm = yc[0] + yc[1]*th
	} else if hs < 2.0 {
		th = 2.0 - hs
		tm = yc[1] + yc[0]*th
	} else if hs < 3.0 {
		th = hs - 2.0
		tm = yc[1] + yc[2]*th
	} else if hs < 4.0 {
		th = 4.0 - hs
		tm = yc[2] + yc[1]*th
	} else if hs < 5.0 {
		th = hs - 4.0
		tm = yc[2] + yc[0]*th
	} else {
		th = 6.0 - hs
		tm = yc[0] + yc[2]*th
	}

	// calculate RGB channels in sorted order
	var tn, to, tp float64
	if tm >= y {
		tp = y + y*c*(1.0-tm)/tm
		to = y + y*c*(th-tm)/tm
		tn = y - (y * c)
	} else {
		tp = y + (1.0-y)*c
		to = y + (1.0-y)*c*(th-tm)/(1.0-tm)
		tn = y - (1.0-y)*c*tm/(1.0-tm)
	}

	// return RGB channels in appropriate order
	if hs < 1.0 {
		return color.NRGBA{R: uint8(igamma(tp) * 0xff), G: uint8(igamma(to) * 0xff), B: uint8(igamma(tn) * 0xff), A: uint8(a * 0xff)}
	} else if hs < 2.0 {
		return color.NRGBA{R: uint8(igamma(to) * 0xff), G: uint8(igamma(tp) * 0xff), B: uint8(igamma(tn) * 0xff), A: uint8(a * 0xff)}
	} else if hs < 3.0 {
		return color.NRGBA{R: uint8(igamma(tn) * 0xff), G: uint8(igamma(tp) * 0xff), B: uint8(igamma(to) * 0xff), A: uint8(a * 0xff)}
	} else if hs < 4.0 {
		return color.NRGBA{R: uint8(igamma(tn) * 0xff), G: uint8(igamma(to) * 0xff), B: uint8(igamma(tp) * 0xff), A: uint8(a * 0xff)}
	} else if hs < 5.0 {
		return color.NRGBA{R: uint8(igamma(to) * 0xff), G: uint8(igamma(tn) * 0xff), B: uint8(igamma(tp) * 0xff), A: uint8(a * 0xff)}
	} else {
		return color.NRGBA{R: uint8(igamma(tp) * 0xff), G: uint8(igamma(tn) * 0xff), B: uint8(igamma(to) * 0xff), A: uint8(a * 0xff)}
	}
}

func (hyca HCYAColor) RGBA() (r, g, b, a uint32) {
	return hyca.ToNRGBA().RGBA()
}

func Luma(col color.Color) float64 {
	rgba := color.NRGBAModel.Convert(col).(color.NRGBA)
	r_int, g_int, b_int := rgba.R, rgba.G, rgba.B
	r := gamma(float64(r_int) / float64(0xff))
	g := gamma(float64(g_int) / float64(0xff))
	b := gamma(float64(b_int) / float64(0xff))
	return lumag(r, g, b)
}

func lignten(col color.Color, ky float64, kc float64) color.Color {
	c := NewHCYAFromColor(col)
	c.y = normalize((1.0 - c.y) * (1.0 - ky))
	c.c = normalize((1.0 - c.c) * kc)
	return c.ToNRGBA()
}

// Lignten (c color.Color, ammount = 0.5, chomaInversionGain = 1.0)
// 
// Adjust the luma of a color by changing its distance from white.
//
// amount == 1.0 gives white
// amount == 0.5 results in a color whose luma is halfway between 1.0
// and that of the original color
// amount == 0.0 gives the original color
// amount == -1.0 gives a color that is 'twice as far from white' as
// the original color, that is luma(result) == 1.0 - 2*(1.0 - luma(color))
//
// amount factor by which to adjust the luma component of the color
// chromaInverseGain (optional) factor by which to adjust the chroma
// component of the color; 1.0 means no change, 0.0 maximizes chroma
func Lignten(c color.Color, optionals ...float64) color.Color {
	amount := 0.5
	chromaInverseGain := 1.0
	if len(optionals) > 0 {
		amount = optionals[0]
	}
	if len(optionals) > 1 {
		chromaInverseGain = optionals[1]
	}
	return lignten(c, amount, chromaInverseGain)
}

func darken(col color.Color, ky float64, kc float64) color.Color {
	c := NewHCYAFromColor(col)
	c.y = normalize(c.y * (1.0 - ky))
	c.c = normalize(c.c * kc)
	return c.ToNRGBA()
}

// Darken (c color.Color, ammount = 0.5, chomaInversionGain = 1.0)
// 
// Adjust the luma of a color by changing its distance from black.
//
// amount == 1.0 gives black
// amount == 0.5 results in a color whose luma is halfway between 0.0
// and that of the original color
// amount == 0.0 gives the original color
// amount == -1.0 gives a color that is 'twice as far from black' as
// the original color, that is luma(result) == 2*luma(color)
//
// amount factor by which to adjust the luma component of the color
// chromaGain (optional) factor by which to adjust the chroma
// component of the color; 1.0 means no change, 0.0 minimizes chroma
func Darken(c color.Color, optionals ...float64) color.Color {
	amount := 0.5
	chromaInverseGain := 1.0
	if len(optionals) > 0 {
		amount = optionals[0]
	}
	if len(optionals) > 1 {
		chromaInverseGain = optionals[1]
	}
	return darken(c, amount, chromaInverseGain)
}
