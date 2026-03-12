package bubblepicker

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// HSL holds hue (0-360), saturation (0-100), lightness (0-100).
type HSL struct {
	H, S, L float64
}

// RGB holds red, green, blue in 0-1.
type RGB struct {
	R, G, B float64
}

// HSLToRGB converts HSL to RGB (all in 0-1 or H 0-360).
func HSLToRGB(h, s, l float64) (r, g, b float64) {
	s /= 100
	l /= 100
	if s == 0 {
		return l, l, l
	}
	var c, x, m float64
	c = (1 - math.Abs(2*l-1)) * s
	x = c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m = l - c/2
	switch {
	case h < 60:
		return c + m, x + m, m
	case h < 120:
		return x + m, c + m, m
	case h < 180:
		return m, c + m, x + m
	case h < 240:
		return m, x + m, c + m
	case h < 300:
		return x + m, m, c + m
	default:
		return c + m, m, x + m
	}
}

// RGBToHSL converts RGB (0-1) to HSL.
func RGBToHSL(r, g, b float64) (h, s, l float64) {
	max := math.Max(math.Max(r, g), b)
	min := math.Min(math.Min(r, g), b)
	l = (max + min) / 2
	if max == min {
		return 0, 0, l * 100
	}
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}
	s *= 100
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h = h/6 * 360
	if h < 0 {
		h += 360
	}
	return h, s, l * 100
}

// HexToHSL parses a hex color "#rrggbb" or "#rgb" and returns HSL.
func HexToHSL(hex string) (HSL, error) {
	hex = strings.TrimPrefix(strings.ToLower(hex), "#")
	var r, g, b float64
	switch len(hex) {
	case 6:
		rr, err := strconv.ParseInt(hex[0:2], 16, 0)
		if err != nil {
			return HSL{}, err
		}
		gg, err := strconv.ParseInt(hex[2:4], 16, 0)
		if err != nil {
			return HSL{}, err
		}
		bb, err := strconv.ParseInt(hex[4:6], 16, 0)
		if err != nil {
			return HSL{}, err
		}
		r, g, b = float64(rr)/255, float64(gg)/255, float64(bb)/255
	case 3:
		rr, err := strconv.ParseInt(hex[0:1]+hex[0:1], 16, 0)
		if err != nil {
			return HSL{}, err
		}
		gg, err := strconv.ParseInt(hex[1:2]+hex[1:2], 16, 0)
		if err != nil {
			return HSL{}, err
		}
		bb, err := strconv.ParseInt(hex[2:3]+hex[2:3], 16, 0)
		if err != nil {
			return HSL{}, err
		}
		r, g, b = float64(rr)/255, float64(gg)/255, float64(bb)/255
	default:
		return HSL{}, fmt.Errorf("invalid hex: %q", hex)
	}
	h, s, l := RGBToHSL(r, g, b)
	return HSL{H: h, S: s, L: l}, nil
}

// ToHex returns the color as "#rrggbb".
func (c HSL) ToHex() string {
	r, g, b := HSLToRGB(c.H, c.S, c.L)
	rr := uint8(math.Round(r * 255))
	gg := uint8(math.Round(g * 255))
	bb := uint8(math.Round(b * 255))
	return fmt.Sprintf("#%02x%02x%02x", rr, gg, bb)
}

// Clamp keeps H in [0,360), S and L in [0,100].
func (c HSL) Clamp() HSL {
	h := math.Mod(c.H, 360)
	if h < 0 {
		h += 360
	}
	s := math.Max(0, math.Min(100, c.S))
	l := math.Max(0, math.Min(100, c.L))
	return HSL{H: h, S: s, L: l}
}
