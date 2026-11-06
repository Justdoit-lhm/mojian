package main

import (
	"embed"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// The window's own colors, in light and dark windows. They are the
// reference's: warm paper in the light, ink and moss in the dark.
type colors struct {
	// The window's background: a gradient from top through mid to bottom,
	// and a tint toward the right.
	bgTop, bgMid, bgLow, bgBottom, bgTint ui.Color

	card, cardFocused, cardBorder, cardBorderFocused ui.Color
	shadow, shadowFocused                            ui.Color

	text, textMuted, textFaint, iconMuted ui.Color
	hover, pressed                        ui.Color

	track, trackBorder, tabActive, tabInactive, tileRim ui.Color

	danger, attention          ui.Color
	diffAdd, diffDel, diffHunk ui.Color

	panel, sheet, panelBorder, panelSel, backdrop ui.Color
}

var lightColors = colors{
	bgTop:    ui.Hex("#f6e7f6"),
	bgMid:    ui.Hex("#efe1e6"),
	bgLow:    ui.Hex("#eee0d3"),
	bgBottom: ui.Hex("#ece1bd"),
	bgTint:   ui.RGBA(242, 214, 222, 0.35),

	card:              ui.RGBA(255, 255, 255, 0.78),
	cardFocused:       ui.RGBA(255, 255, 255, 0.97),
	cardBorder:        ui.RGBA(255, 255, 255, 0.82),
	cardBorderFocused: ui.RGBA(255, 255, 255, 1),
	shadow:            ui.RGBA(60, 40, 50, 0.07),
	shadowFocused:     ui.RGBA(60, 40, 50, 0.12),

	text:      ui.Hex("#1d1d1f"),
	textMuted: ui.Hex("#636366"),
	textFaint: ui.Hex("#6f6f75"),
	iconMuted: ui.Hex("#7c7c82"),
	hover:     ui.RGBA(0, 0, 0, 0.055),
	pressed:   ui.RGBA(0, 0, 0, 0.1),

	track:       ui.RGBA(255, 255, 255, 0.34),
	trackBorder: ui.RGBA(255, 255, 255, 0.45),
	tabActive:   ui.RGBA(255, 255, 255, 0.97),
	tabInactive: ui.RGBA(255, 255, 255, 0.34),
	tileRim:     ui.RGBA(255, 255, 255, 0.95),

	danger:    ui.Hex("#d92d20"),
	attention: ui.Hex("#f59e0b"),
	diffAdd:   ui.Hex("#1a7f37"),
	diffDel:   ui.Hex("#cf222e"),
	diffHunk:  ui.Hex("#0969da"),

	panel:       ui.RGBA(252, 252, 253, 0.97),
	sheet:       ui.Hex("#fbfbfc"),
	panelBorder: ui.RGBA(0, 0, 0, 0.08),
	panelSel:    ui.RGBA(46, 111, 208, 0.12),
	backdrop:    ui.RGBA(30, 15, 25, 0.18),
}

var darkColors = colors{
	bgTop:    ui.Hex("#2c2131"),
	bgMid:    ui.Hex("#231e27"),
	bgLow:    ui.Hex("#221f22"),
	bgBottom: ui.Hex("#2a2619"),
	bgTint:   ui.RGBA(80, 40, 60, 0.25),

	card:              ui.RGBA(24, 24, 28, 0.58),
	cardFocused:       ui.RGBA(36, 36, 41, 0.95),
	cardBorder:        ui.RGBA(255, 255, 255, 0.05),
	cardBorderFocused: ui.RGBA(255, 255, 255, 0.14),
	shadow:            ui.RGBA(0, 0, 0, 0.25),
	shadowFocused:     ui.RGBA(0, 0, 0, 0.4),

	text:      ui.Hex("#f2f2f7"),
	textMuted: ui.Hex("#aeaeb2"),
	textFaint: ui.Hex("#8e8e93"),
	iconMuted: ui.Hex("#98989d"),
	hover:     ui.RGBA(255, 255, 255, 0.08),
	pressed:   ui.RGBA(255, 255, 255, 0.14),

	track:       ui.RGBA(255, 255, 255, 0.06),
	trackBorder: ui.RGBA(255, 255, 255, 0.06),
	tabActive:   ui.RGBA(255, 255, 255, 0.14),
	tabInactive: ui.RGBA(255, 255, 255, 0.04),
	tileRim:     ui.RGBA(255, 255, 255, 0.22),

	danger:    ui.Hex("#f97066"),
	attention: ui.Hex("#fbbf24"),
	diffAdd:   ui.Hex("#7ee787"),
	diffDel:   ui.Hex("#ff7b72"),
	diffHunk:  ui.Hex("#79c0ff"),

	panel:       ui.RGBA(38, 38, 42, 0.97),
	sheet:       ui.Hex("#1f1f23"),
	panelBorder: ui.RGBA(255, 255, 255, 0.1),
	panelSel:    ui.RGBA(108, 178, 255, 0.18),
	backdrop:    ui.RGBA(0, 0, 0, 0.45),
}

func colorsOf(c *ui.Context) *colors {
	if c.Theme().Dark {
		return &darkColors
	}
	return &lightColors
}

// paintBackground paints the window's background below the cards.
func paintBackground(p *ui.Painter, r ui.Rect, k *colors) {
	h1, h2 := r.H*0.38, r.H*0.32
	p.FillGradient(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: h1 + 1},
		ui.LinearGradient{From: k.bgTop, To: k.bgMid, Angle: 180}, 0)
	p.FillGradient(ui.Rect{X: r.X, Y: r.Y + h1, W: r.W, H: h2 + 1},
		ui.LinearGradient{From: k.bgMid, To: k.bgLow, Angle: 180}, 0)
	p.FillGradient(ui.Rect{X: r.X, Y: r.Y + h1 + h2, W: r.W, H: r.H - h1 - h2},
		ui.LinearGradient{From: k.bgLow, To: k.bgBottom, Angle: 180}, 0)
	clear := k.bgTint
	clear.A = 0
	p.FillGradient(r, ui.LinearGradient{From: clear, To: k.bgTint, Angle: 90, Start: 0.35, End: 1}, 0)
}

// noteFont is the editor's face: the embedding makes it the same on every
// machine, so a note's column is as wide everywhere.
var noteFont = ui.Font{
	Family:     "JetBrains Mono, SF Mono, Menlo, monospace",
	Size:       12.5,
	LineHeight: 1.55,
}

// editorThemeOf colors an open note, over the card's paper.
func editorThemeOf(t *ui.Theme) *editor.Theme {
	th := editor.DefaultTheme(t)
	th.Background = ui.Transparent
	th.ActiveLine = t.Text.Alpha(0.035)
	th.InactiveSelection = t.Selection.Alpha(0.45)
	return th
}

//go:embed assets/icons/*.svg assets/icons/material/*.svg
var iconAssets embed.FS

var svgCache = map[string]*ui.SVG{}

// icon returns an embedded icon by name. UI actions are Lucide; files and
// folders come from Material Icon Theme (assets/icons/material).
func icon(name string) *ui.SVG {
	if s, ok := svgCache[name]; ok {
		return s
	}
	data, err := iconAssets.ReadFile(path.Join("assets/icons", name+".svg"))
	if err != nil {
		panic("mojian: icon " + name + ": " + err.Error())
	}
	s := ui.MustParseSVG(data)
	svgCache[name] = s
	return s
}

//go:embed assets/fonts/*.ttf
var fontAssets embed.FS

// registerFonts loads the editor's faces, under one family.
func registerFonts() {
	entries, err := fontAssets.ReadDir("assets/fonts")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".ttf") {
			continue
		}
		data, err := fontAssets.ReadFile("assets/fonts/" + e.Name())
		if err != nil {
			panic(err)
		}
		if err := ui.RegisterFont(data, "JetBrains Mono"); err != nil {
			panic(err)
		}
	}
}
