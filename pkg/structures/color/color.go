package color

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
)

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

func FromColor(c color.Color) Color {
	r, g, b, a := unmultiplyAlpha(c)
	return Color{
		R: uint8(clamp(r, 0, 255)),
		G: uint8(clamp(g, 0, 255)),
		B: uint8(clamp(b, 0, 255)),
		A: uint8(clamp(a, 0, 255)),
	}
}

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

func clamp(v, m, mx int) int {
	if v < m {
		return m
	}
	if v > mx {
		return mx
	}
	return v
}

func (c *Color) ToColor() color.NRGBA {
	return color.NRGBA{
		R: c.R,
		G: c.G,
		B: c.B,
		A: c.A,
	}
}

func (c *Color) HexEncode() string {
	if c.A != 255 {
		return fmt.Sprintf("%.2x%.2x%.2x%.2x", c.R, c.G, c.B, c.A)
	}
	return fmt.Sprintf("%.2x%.2x%.2x", c.R, c.G, c.B)
}

func (c *Color) String() string {
	return c.HexEncode()
}

var errInvalidFormat = errors.New("invalid color format")

func ParseHexColorFast(s string) (c Color, err error) {
	c.A = 0xff

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errInvalidFormat
		return 0
	}

	off := 0
	switch len(s) {
	case 9:
		if s[0] != '#' {
			return c, errInvalidFormat
		}
		off = 1
		fallthrough
	case 8:
		c.R = hexToByte(s[0+off])<<4 + hexToByte(s[1+off])
		c.G = hexToByte(s[2+off])<<4 + hexToByte(s[3+off])
		c.B = hexToByte(s[4+off])<<4 + hexToByte(s[5+off])
		c.A = hexToByte(s[6+off])<<4 + hexToByte(s[7+off])
	case 7:
		if s[0] != '#' {
			return c, errInvalidFormat
		}
		off = 1
		fallthrough
	case 6:
		c.R = hexToByte(s[0+off])<<4 + hexToByte(s[1+off])
		c.G = hexToByte(s[2+off])<<4 + hexToByte(s[3+off])
		c.B = hexToByte(s[4+off])<<4 + hexToByte(s[5+off])
	case 4:
		if s[0] != '#' {
			return c, errInvalidFormat
		}
		off = 1
		fallthrough
	case 3:
		c.R = hexToByte(s[0+off]) * 17
		c.G = hexToByte(s[1+off]) * 17
		c.B = hexToByte(s[2+off]) * 17
	default:
		err = errInvalidFormat
	}
	return
}

func (c *Color) UnmarshalJSON(bytes []byte) error {
	var s string
	err := json.Unmarshal(bytes, &s)
	if err != nil {
		return err
	}
	color, err := ParseHexColorFast(s)
	if err != nil {
		return err
	}
	c.R = color.R
	c.G = color.G
	c.B = color.B
	c.A = color.A
	return nil
}

func (c *Color) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.HexEncode())
}
