// Package theme holds Mojian's light and dark colors, the one corner
// radius, background painting, the editor theme and font registration.
package theme

import (
	"strings"

	"mojian/internal/assets"

	"github.com/egoist/mygo/ui"
)

// Radius is the one corner radius used by the window's surfaces: cards,
// panes, fields, buttons, popovers and dialogs.
const Radius = 10

// The window's own colors, in light and dark windows. They are the
// default theme's: porcelain with a breath of rose in the light, a deep
// neutral plum in the dark. The window leans to one tint; nothing in it
// turns yellow or olive.
type Colors struct {
	// The window's background: a gradient from top through mid to bottom,
	// and a tint toward the right.
	BgTop, BgMid, BgLow, BgBottom, BgTint ui.Color

	Card, CardFocused, CardBorder, CardBorderFocused ui.Color
	Shadow, ShadowFocused                            ui.Color

	Text, TextMuted, TextFaint, IconMuted ui.Color
	Hover, Pressed                        ui.Color

	// Accent is the theme's own color: the selected row, the focus ring,
	// the cursor and the terminal's selection all take it.
	Accent ui.Color

	Track, TrackBorder, TabActive, TabInactive, TileRim ui.Color

	Danger, Attention          ui.Color
	DiffAdd, DiffDel, DiffHunk ui.Color

	Panel, Sheet, PanelBorder, PanelSel, Backdrop ui.Color
}

var lightColors = Colors{
	BgTop:    ui.Hex("#f7f2f6"),
	BgMid:    ui.Hex("#f4eff4"),
	BgLow:    ui.Hex("#efeaf1"),
	BgBottom: ui.Hex("#e9e5ed"),
	BgTint:   ui.RGBA(235, 222, 232, 0.50),

	Card:              ui.RGBA(255, 255, 255, 0.72),
	CardFocused:       ui.RGBA(255, 255, 255, 0.96),
	CardBorder:        ui.RGBA(255, 255, 255, 0.85),
	CardBorderFocused: ui.RGBA(255, 255, 255, 1),
	Shadow:            ui.RGBA(48, 36, 48, 0.07),
	ShadowFocused:     ui.RGBA(48, 36, 48, 0.12),

	Text:      ui.Hex("#1f1b20"),
	TextMuted: ui.Hex("#66616a"),
	TextFaint: ui.Hex("#75707a"),
	IconMuted: ui.Hex("#807b84"),
	Hover:     ui.RGBA(35, 24, 34, 0.05),
	Pressed:   ui.RGBA(35, 24, 34, 0.09),

	Accent: ui.Hex("#a84e7c"),

	Track:       ui.RGBA(255, 255, 255, 0.38),
	TrackBorder: ui.RGBA(255, 255, 255, 0.50),
	TabActive:   ui.RGBA(255, 255, 255, 0.96),
	TabInactive: ui.RGBA(255, 255, 255, 0.34),
	TileRim:     ui.RGBA(255, 255, 255, 0.95),

	Danger:    ui.Hex("#d92d20"),
	Attention: ui.Hex("#f59e0b"),
	DiffAdd:   ui.Hex("#1a7f37"),
	DiffDel:   ui.Hex("#cf222e"),
	DiffHunk:  ui.Hex("#0969da"),

	Panel:       ui.RGBA(251, 249, 251, 0.97),
	Sheet:       ui.Hex("#fbfafc"),
	PanelBorder: ui.RGBA(28, 18, 28, 0.08),
	PanelSel:    ui.RGBA(168, 78, 124, 0.12),
	Backdrop:    ui.RGBA(28, 16, 26, 0.18),
}

var darkColors = Colors{
	BgTop:    ui.Hex("#241d25"),
	BgMid:    ui.Hex("#1f1a21"),
	BgLow:    ui.Hex("#1b181d"),
	BgBottom: ui.Hex("#18151a"),
	BgTint:   ui.RGBA(120, 72, 100, 0.18),

	Card:              ui.RGBA(25, 23, 28, 0.58),
	CardFocused:       ui.RGBA(37, 35, 41, 0.95),
	CardBorder:        ui.RGBA(255, 255, 255, 0.06),
	CardBorderFocused: ui.RGBA(255, 255, 255, 0.14),
	Shadow:            ui.RGBA(0, 0, 0, 0.28),
	ShadowFocused:     ui.RGBA(0, 0, 0, 0.45),

	Text:      ui.Hex("#f3f0f4"),
	TextMuted: ui.Hex("#b1abb4"),
	TextFaint: ui.Hex("#928c96"),
	IconMuted: ui.Hex("#9c96a0"),
	Hover:     ui.RGBA(255, 255, 255, 0.07),
	Pressed:   ui.RGBA(255, 255, 255, 0.13),

	Accent: ui.Hex("#e887b6"),

	Track:       ui.RGBA(255, 255, 255, 0.06),
	TrackBorder: ui.RGBA(255, 255, 255, 0.07),
	TabActive:   ui.RGBA(255, 255, 255, 0.13),
	TabInactive: ui.RGBA(255, 255, 255, 0.04),
	TileRim:     ui.RGBA(255, 255, 255, 0.22),

	Danger:    ui.Hex("#f97066"),
	Attention: ui.Hex("#fbbf24"),
	DiffAdd:   ui.Hex("#7ee787"),
	DiffDel:   ui.Hex("#ff7b72"),
	DiffHunk:  ui.Hex("#79c0ff"),

	Panel:       ui.RGBA(35, 32, 38, 0.97),
	Sheet:       ui.Hex("#1c1a1f"),
	PanelBorder: ui.RGBA(255, 255, 255, 0.10),
	PanelSel:    ui.RGBA(232, 135, 182, 0.18),
	Backdrop:    ui.RGBA(0, 0, 0, 0.45),
}

// PaintBackground paints the window's background below the cards.
func PaintBackground(p *ui.Painter, r ui.Rect, k *Colors) {
	h1, h2 := r.H*0.38, r.H*0.32
	p.FillGradient(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: h1 + 1},
		ui.LinearGradient{From: k.BgTop, To: k.BgMid, Angle: 180}, 0)
	p.FillGradient(ui.Rect{X: r.X, Y: r.Y + h1, W: r.W, H: h2 + 1},
		ui.LinearGradient{From: k.BgMid, To: k.BgLow, Angle: 180}, 0)
	p.FillGradient(ui.Rect{X: r.X, Y: r.Y + h1 + h2, W: r.W, H: r.H - h1 - h2},
		ui.LinearGradient{From: k.BgLow, To: k.BgBottom, Angle: 180}, 0)
	clear := k.BgTint
	clear.A = 0
	p.FillGradient(r, ui.LinearGradient{From: clear, To: k.BgTint, Angle: 90, Start: 0.35, End: 1}, 0)
}

// NoteFont is the editor's face: the embedding makes it the same on every
// machine, so a note's column is as wide everywhere.
var NoteFont = ui.Font{
	Family:     "JetBrains Mono, SF Mono, Menlo, monospace",
	Size:       12.5,
	LineHeight: 1.55,
}

// RegisterFonts loads the editor's faces, under one family.
func RegisterFonts() {
	entries, err := assets.Files.ReadDir("fonts")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".ttf") {
			continue
		}
		data, err := assets.Files.ReadFile("fonts/" + e.Name())
		if err != nil {
			panic(err)
		}
		if err := ui.RegisterFont(data, "JetBrains Mono"); err != nil {
			panic(err)
		}
	}
}
