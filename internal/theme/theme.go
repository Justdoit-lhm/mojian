package theme

import (
	"strings"

	"mojian/internal/assets"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Radius is the one corner radius used by the window's surfaces: cards,
// panes, fields, buttons, popovers and dialogs.
const Radius = 10

// The window's own colors, in light and dark windows. They are the
// reference's: warm paper in the light, ink and moss in the dark.
type Colors struct {
	// The window's background: a gradient from top through mid to bottom,
	// and a tint toward the right.
	BgTop, BgMid, BgLow, BgBottom, BgTint ui.Color

	Card, CardFocused, CardBorder, CardBorderFocused ui.Color
	Shadow, ShadowFocused                            ui.Color

	Text, TextMuted, TextFaint, IconMuted ui.Color
	Hover, Pressed                        ui.Color

	Track, TrackBorder, TabActive, TabInactive, TileRim ui.Color

	Danger, Attention          ui.Color
	DiffAdd, DiffDel, DiffHunk ui.Color

	Panel, Sheet, PanelBorder, PanelSel, Backdrop ui.Color
}

var lightColors = Colors{
	BgTop:    ui.Hex("#f6e7f6"),
	BgMid:    ui.Hex("#efe1e6"),
	BgLow:    ui.Hex("#eee0d3"),
	BgBottom: ui.Hex("#ece1bd"),
	BgTint:   ui.RGBA(242, 214, 222, 0.35),

	Card:              ui.RGBA(255, 255, 255, 0.78),
	CardFocused:       ui.RGBA(255, 255, 255, 0.97),
	CardBorder:        ui.RGBA(255, 255, 255, 0.82),
	CardBorderFocused: ui.RGBA(255, 255, 255, 1),
	Shadow:            ui.RGBA(60, 40, 50, 0.07),
	ShadowFocused:     ui.RGBA(60, 40, 50, 0.12),

	Text:      ui.Hex("#1d1d1f"),
	TextMuted: ui.Hex("#636366"),
	TextFaint: ui.Hex("#6f6f75"),
	IconMuted: ui.Hex("#7c7c82"),
	Hover:     ui.RGBA(0, 0, 0, 0.055),
	Pressed:   ui.RGBA(0, 0, 0, 0.1),

	Track:       ui.RGBA(255, 255, 255, 0.34),
	TrackBorder: ui.RGBA(255, 255, 255, 0.45),
	TabActive:   ui.RGBA(255, 255, 255, 0.97),
	TabInactive: ui.RGBA(255, 255, 255, 0.34),
	TileRim:     ui.RGBA(255, 255, 255, 0.95),

	Danger:    ui.Hex("#d92d20"),
	Attention: ui.Hex("#f59e0b"),
	DiffAdd:   ui.Hex("#1a7f37"),
	DiffDel:   ui.Hex("#cf222e"),
	DiffHunk:  ui.Hex("#0969da"),

	Panel:       ui.RGBA(252, 252, 253, 0.97),
	Sheet:       ui.Hex("#fbfbfc"),
	PanelBorder: ui.RGBA(0, 0, 0, 0.08),
	PanelSel:    ui.RGBA(46, 111, 208, 0.12),
	Backdrop:    ui.RGBA(30, 15, 25, 0.18),
}

var darkColors = Colors{
	BgTop:    ui.Hex("#2c2131"),
	BgMid:    ui.Hex("#231e27"),
	BgLow:    ui.Hex("#221f22"),
	BgBottom: ui.Hex("#2a2619"),
	BgTint:   ui.RGBA(80, 40, 60, 0.25),

	Card:              ui.RGBA(24, 24, 28, 0.58),
	CardFocused:       ui.RGBA(36, 36, 41, 0.95),
	CardBorder:        ui.RGBA(255, 255, 255, 0.05),
	CardBorderFocused: ui.RGBA(255, 255, 255, 0.14),
	Shadow:            ui.RGBA(0, 0, 0, 0.25),
	ShadowFocused:     ui.RGBA(0, 0, 0, 0.4),

	Text:      ui.Hex("#f2f2f7"),
	TextMuted: ui.Hex("#aeaeb2"),
	TextFaint: ui.Hex("#8e8e93"),
	IconMuted: ui.Hex("#98989d"),
	Hover:     ui.RGBA(255, 255, 255, 0.08),
	Pressed:   ui.RGBA(255, 255, 255, 0.14),

	Track:       ui.RGBA(255, 255, 255, 0.06),
	TrackBorder: ui.RGBA(255, 255, 255, 0.06),
	TabActive:   ui.RGBA(255, 255, 255, 0.14),
	TabInactive: ui.RGBA(255, 255, 255, 0.04),
	TileRim:     ui.RGBA(255, 255, 255, 0.22),

	Danger:    ui.Hex("#f97066"),
	Attention: ui.Hex("#fbbf24"),
	DiffAdd:   ui.Hex("#7ee787"),
	DiffDel:   ui.Hex("#ff7b72"),
	DiffHunk:  ui.Hex("#79c0ff"),

	Panel:       ui.RGBA(38, 38, 42, 0.97),
	Sheet:       ui.Hex("#1f1f23"),
	PanelBorder: ui.RGBA(255, 255, 255, 0.1),
	PanelSel:    ui.RGBA(108, 178, 255, 0.18),
	Backdrop:    ui.RGBA(0, 0, 0, 0.45),
}

func Of(c *ui.Context) *Colors {
	if c.Theme().Dark {
		return &darkColors
	}
	return &lightColors
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

// EditorTheme colors an open note, over the Card's paper.
func EditorTheme(t *ui.Theme) *editor.Theme {
	th := editor.DefaultTheme(t)
	th.Background = ui.Transparent
	th.ActiveLine = t.Text.Alpha(0.035)
	th.InactiveSelection = t.Selection.Alpha(0.45)
	return th
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
