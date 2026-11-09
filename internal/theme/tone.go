package theme

import (
	"math"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// tone is what makes one theme family look like itself: the hue the window
// leans toward, how far it leans (Tint, 0 to 1), how vivid the program
// colors are (Vivid) and the hue of the accent and the cursor (Accent).
type tone struct {
	Hue, Tint, Vivid, Accent float32
}

// oklch returns the sRGB color of CSS's oklch(l c h). The channels are
// clamped into the gamut instead of being mapped through a wider space, so
// a preview thumbnail shows the same color the window does on every
// display.
func oklch(l, c, h float32) ui.Color {
	rad := float64(math.Remainder(float64(h), 360)) * math.Pi / 180
	a := float64(c) * math.Cos(rad)
	b := float64(c) * math.Sin(rad)
	lc := float64(l)
	// Oklab to linear sRGB, then the sRGB transfer function.
	conv := func(p float64) float64 { return p * p * p }
	lo := conv(lc + 0.3963377774*a + 0.2158037573*b)
	mo := conv(lc - 0.1055613458*a - 0.0638541728*b)
	so := conv(lc - 0.0894841775*a - 1.2914855480*b)
	lin := [3]float64{
		4.0767416621*lo - 3.3077115913*mo + 0.2309699292*so,
		-1.2684380046*lo + 2.6097574011*mo - 0.3413193965*so,
		-0.0041960863*lo - 0.7034186147*mo + 1.7076147010*so,
	}
	var out [3]uint8
	for i, v := range lin {
		v = math.Max(0, math.Min(1, v))
		var e float64
		if v <= 0.0031308 {
			e = 12.92 * v
		} else {
			e = 1.055*math.Pow(v, 1.0/2.4) - 0.055
		}
		out[i] = uint8(math.Round(e * 255))
	}
	return ui.RGB(out[0], out[1], out[2])
}

func absf(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// toneTheme builds a readable window and program palette around a tone: the
// window's paper takes the family's tint, while the sixteen program colors
// keep their own hues and bend only as far toward it as staying legible
// allows.
func toneTheme(t tone, dark bool) (*terminal.Theme, ui.Color) {
	tint := func(l, c float32) ui.Color { return oklch(l, c*t.Tint, t.Hue) }
	hues := [6]float32{28, 145, 85, 255, 325, 195}
	ls := [6]float32{0.56, 0.56, 0.62, 0.54, 0.54, 0.56}
	cs := [6]float32{0.15, 0.13, 0.12, 0.13, 0.14, 0.09}
	bright := float32(0.05)
	th := &terminal.Theme{}
	var accent ui.Color
	if dark {
		ls = [6]float32{0.70, 0.74, 0.80, 0.70, 0.72, 0.76}
		cs = [6]float32{0.14, 0.13, 0.12, 0.12, 0.13, 0.10}
		bright = 0.06
		th.Background, th.Foreground = tint(0.255, 0.02), tint(0.92, 0.015)
		th.Palette[0], th.Palette[7], th.Palette[8], th.Palette[15] =
			tint(0.36, 0.02), tint(0.82, 0.012), tint(0.5, 0.02), tint(0.96, 0.01)
		th.Selection = oklch(0.38, 0.06, t.Accent)
		accent = oklch(0.74, 0.13, t.Accent)
	} else {
		// A faint tint leaves the paper nearly neutral, which needs a
		// lighter paper than a strong one to keep the same calm.
		paper := 0.978 + 0.022*max(0, 1-t.Tint/0.5)
		th.Background, th.Foreground = tint(paper, 0.012), tint(0.28, 0.03)
		th.Palette[0], th.Palette[7], th.Palette[8], th.Palette[15] =
			th.Foreground, tint(0.7, 0.02), tint(0.52, 0.02), tint(0.8, 0.015)
		th.Selection = oklch(0.9, 0.045, t.Accent)
		accent = oklch(0.55, 0.13, t.Accent)
	}
	for i := range 6 {
		d := float32(math.Remainder(float64(t.Hue-hues[i]), 360))
		h := hues[i] + d*0.35*(1-absf(d)/180)
		c := cs[i] * t.Vivid * (1 + 0.45*max(0, 1-absf(d)/60))
		th.Palette[1+i] = oklch(ls[i], c, h)
		th.Palette[9+i] = oklch(ls[i]+bright, c+0.01, h)
	}
	th.Cursor, th.CursorText, th.SelectionText = accent, th.Background, th.Foreground
	return th, accent
}
