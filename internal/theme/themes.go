package theme

import (
	"strings"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// DefaultID is the theme of a new install: Mojian's own warm paper, which
// every other theme was built beside.
const DefaultID = "petal"

// Choice is one theme the Appearance page offers.
type Choice struct {
	ID   string
	Name string
}

// Choices are the themes, in the order the grid shows them.
var Choices = []Choice{
	{"petal", "Petal"},
	{"mist", "Mist"},
	{"meadow", "Meadow"},
	{"dusk", "Dusk"},
	{"tide", "Tide"},
	{"apricot", "Apricot"},
	{"aurora", "Aurora"},
	{"dune", "Dune"},
	{"paper", "Paper"},
	{"plain", "Plain"},
}

// Resolve returns the theme an id names, or the default for one this
// version does not know, so a settings file from a newer build still opens
// a window.
func Resolve(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if _, ok := washes[id]; ok {
		return id
	}
	return DefaultID
}

// Dark reports whether an appearance preference (0 system, 1 light, 2 dark)
// means a dark window, given what the system says.
func Dark(appearance int, systemDark bool) bool {
	switch appearance {
	case 1:
		return false
	case 2:
		return true
	}
	return systemDark
}

// wash is a theme's surface: the four stops of the window's background
// gradient, the tint painted toward the right, and the tone its program
// colors are built from.
type wash struct {
	light, dark [4]ui.Color
	lightGlow   ui.Color
	darkGlow    ui.Color
	tone        tone
}

var washes = map[string]wash{
	"petal": {
		light:     [4]ui.Color{ui.Hex("#f6e7f6"), ui.Hex("#efe1e6"), ui.Hex("#eee0d3"), ui.Hex("#ece1bd")},
		dark:      [4]ui.Color{ui.Hex("#2c2131"), ui.Hex("#231e27"), ui.Hex("#221f22"), ui.Hex("#2a2619")},
		lightGlow: ui.RGBA(242, 214, 222, 0.35),
		darkGlow:  ui.RGBA(80, 40, 60, 0.25),
		tone:      tone{Hue: 350, Tint: 1, Vivid: 0.95, Accent: 350},
	},
	"mist": {
		light:     [4]ui.Color{ui.Hex("#e3e9f4"), ui.Hex("#ebeff6"), ui.Hex("#f1f2f3"), ui.Hex("#f2f1ec")},
		dark:      [4]ui.Color{ui.Hex("#0c1628"), ui.Hex("#0d1827"), ui.Hex("#111b23"), ui.Hex("#111b23")},
		lightGlow: ui.RGBA(210, 225, 248, 0.75),
		darkGlow:  ui.RGBA(46, 91, 166, 0.30),
		tone:      tone{Hue: 255, Tint: 1, Vivid: 0.8, Accent: 255},
	},
	"meadow": {
		light:     [4]ui.Color{ui.Hex("#e2eee5"), ui.Hex("#ebf2e7"), ui.Hex("#f3f2e2"), ui.Hex("#f5f0d6")},
		dark:      [4]ui.Color{ui.Hex("#081c0d"), ui.Hex("#101d0c"), ui.Hex("#1b1c0b"), ui.Hex("#1b1c0b")},
		lightGlow: ui.RGBA(214, 240, 222, 0.75),
		darkGlow:  ui.RGBA(6, 113, 50, 0.30),
		tone:      tone{Hue: 140, Tint: 1, Vivid: 0.9, Accent: 140},
	},
	"dusk": {
		light:     [4]ui.Color{ui.Hex("#e4dbf3"), ui.Hex("#efdeee"), ui.Hex("#f6e3dc"), ui.Hex("#f7e6cf")},
		dark:      [4]ui.Color{ui.Hex("#190f2e"), ui.Hex("#260f24"), ui.Hex("#2d140b"), ui.Hex("#2d140b")},
		lightGlow: ui.RGBA(246, 206, 222, 0.80),
		darkGlow:  ui.RGBA(125, 65, 138, 0.30),
		tone:      tone{Hue: 315, Tint: 1, Vivid: 1.1, Accent: 315},
	},
	"tide": {
		light:     [4]ui.Color{ui.Hex("#dbebef"), ui.Hex("#e3eff1"), ui.Hex("#ecf3f0"), ui.Hex("#f2f2e8")},
		dark:      [4]ui.Color{ui.Hex("#001b22"), ui.Hex("#001e1f"), ui.Hex("#0b1e1b"), ui.Hex("#0b1e1b")},
		lightGlow: ui.RGBA(194, 232, 238, 0.75),
		darkGlow:  ui.RGBA(0, 112, 128, 0.30),
		tone:      tone{Hue: 205, Tint: 1, Vivid: 0.85, Accent: 205},
	},
	"apricot": {
		light:     [4]ui.Color{ui.Hex("#f5e1d6"), ui.Hex("#f8e7dc"), ui.Hex("#f8eedf"), ui.Hex("#f6f0da")},
		dark:      [4]ui.Color{ui.Hex("#2a1207"), ui.Hex("#291507"), ui.Hex("#25190a"), ui.Hex("#25190a")},
		lightGlow: ui.RGBA(252, 212, 194, 0.78),
		darkGlow:  ui.RGBA(151, 62, 22, 0.30),
		tone:      tone{Hue: 50, Tint: 1, Vivid: 1.05, Accent: 50},
	},
	"aurora": {
		light:     [4]ui.Color{ui.Hex("#dbefe8"), ui.Hex("#e1ebf2"), ui.Hex("#e8e4f4"), ui.Hex("#f0e6f1")},
		dark:      [4]ui.Color{ui.Hex("#001f13"), ui.Hex("#06192b"), ui.Hex("#1e122a"), ui.Hex("#1e122a")},
		lightGlow: ui.RGBA(198, 238, 224, 0.75),
		darkGlow:  ui.RGBA(0, 115, 75, 0.30),
		tone:      tone{Hue: 170, Tint: 1, Vivid: 1.15, Accent: 170},
	},
	"dune": {
		light:     [4]ui.Color{ui.Hex("#ede4d8"), ui.Hex("#f1e9de"), ui.Hex("#f4eee4"), ui.Hex("#f3efe8")},
		dark:      [4]ui.Color{ui.Hex("#22190b"), ui.Hex("#211a0e"), ui.Hex("#201b12"), ui.Hex("#201b12")},
		lightGlow: ui.RGBA(242, 222, 194, 0.70),
		darkGlow:  ui.RGBA(135, 79, 0, 0.30),
		tone:      tone{Hue: 80, Tint: 0.7, Vivid: 0.55, Accent: 80},
	},
	"paper": {
		light:     [4]ui.Color{ui.Hex("#f6f6f7"), ui.Hex("#f7f7f8"), ui.Hex("#f8f8f8"), ui.Hex("#f8f8f8")},
		dark:      [4]ui.Color{ui.Hex("#141416"), ui.Hex("#151517"), ui.Hex("#161618"), ui.Hex("#161618")},
		lightGlow: ui.RGBA(255, 255, 255, 0),
		darkGlow:  ui.RGBA(90, 120, 200, 0.10),
		tone:      tone{Hue: 260, Tint: 0, Vivid: 0.9, Accent: 262},
	},
	"plain": {
		light:     [4]ui.Color{ui.Hex("#eceef4"), ui.Hex("#eceef4"), ui.Hex("#eceef4"), ui.Hex("#eceef4")},
		dark:      [4]ui.Color{ui.Hex("#141418"), ui.Hex("#141418"), ui.Hex("#141418"), ui.Hex("#141418")},
		lightGlow: ui.RGBA(0, 0, 0, 0),
		darkGlow:  ui.RGBA(0, 0, 0, 0),
		tone:      tone{Hue: 260, Tint: 0, Vivid: 0.9, Accent: 262},
	},
}

// look is one theme in one appearance: the window's palette and the
// colors of programs.
type look struct {
	colors *Colors
	term   *terminal.Theme
}

// looks holds every theme built ahead of the first frame, indexed by 0 for
// light and 1 for dark. A window reads them, so a theme switch costs a
// lookup.
var looks = buildLooks()

func buildLooks() map[string][2]*look {
	out := make(map[string][2]*look, len(Choices))
	for _, ch := range Choices {
		w := washes[ch.ID]
		var pair [2]*look
		for b, dark := range []bool{false, true} {
			k := lightColors
			if dark {
				k = darkColors
			}
			var term *terminal.Theme
			if ch.ID == DefaultID {
				// The default keeps the palette Mojian has always drawn,
				// and takes only its accent from the tone.
				_, accent := toneTheme(w.tone, dark)
				k.Accent = accent
				k.PanelSel = withAlpha(accent, pick(dark, float32(.18), float32(.12)))
				term = finishTerminal(defaultTerm(dark), dark, &k)
			} else {
				var accent ui.Color
				term, accent = toneTheme(w.tone, dark)
				k = deriveColors(w, dark, term, accent)
				finishTerminal(term, dark, &k)
			}
			pair[b] = &look{colors: &k, term: term}
		}
		out[ch.ID] = pair
	}
	return out
}

// defaultTerm is MyGo's own terminal palette for an appearance, freshly
// allocated so a theme can change it.
func defaultTerm(dark bool) *terminal.Theme {
	if dark {
		return terminal.DarkTheme()
	}
	return terminal.LightTheme()
}

// finishTerminal points the parts of a terminal a user sees most at the
// window it sits in: its text on the card, its cursor and selection in the
// accent.
func finishTerminal(th *terminal.Theme, dark bool, k *Colors) *terminal.Theme {
	th.Foreground = k.Text
	th.Background = k.CardFocused.Over(k.BgMid)
	th.Cursor = k.Accent
	th.CursorText = k.BgBottom
	th.Selection = withAlpha(k.Accent, pick(dark, float32(.4), float32(.25)))
	th.SelectionText = k.Text
	return th
}

// deriveColors builds a window palette from a wash: the background is the
// theme's own, the cards are white or ink tinted by the terminal's paper,
// and the text and the semantic colors come from the program palette so a
// theme reads as one piece.
func deriveColors(w wash, dark bool, term *terminal.Theme, accent ui.Color) Colors {
	k := lightColors
	if dark {
		k = darkColors
	}
	stop := w.light
	glow := w.lightGlow
	if dark {
		stop = w.dark
		glow = w.darkGlow
	}
	k.BgTop, k.BgMid, k.BgLow, k.BgBottom = stop[0], stop[1], stop[2], stop[3]
	k.BgTint = glow

	if dark {
		k.Card = withAlpha(ui.Hex("#18181c").Mix(term.Background, .35), .58)
		k.CardFocused = withAlpha(ui.Hex("#242429").Mix(term.Background, .35), .95)
	} else {
		k.Card = withAlpha(ui.Hex("#ffffff").Mix(term.Background, .10), .78)
		k.CardFocused = withAlpha(ui.Hex("#ffffff").Mix(term.Background, .06), .97)
	}

	k.Text = term.Foreground
	k.TextMuted = term.Foreground.Mix(term.Background, .38)
	k.TextFaint = term.Foreground.Mix(term.Background, .56)
	k.IconMuted = k.TextFaint

	k.Accent = accent
	k.PanelSel = withAlpha(accent, pick(dark, float32(.18), float32(.12)))

	k.Danger = term.Palette[1]
	k.Attention = term.Palette[3]
	k.DiffAdd = term.Palette[2]
	k.DiffDel = term.Palette[1]
	k.DiffHunk = term.Palette[4]

	panelMix := float32(.60)
	if dark {
		panelMix = .04
	}
	k.Panel = withAlpha(term.Background.Mix(ui.Hex("#ffffff"), panelMix), .97)
	k.Sheet = opaque(term.Background.Mix(ui.Hex("#ffffff"), panelMix))
	return k
}

// Window is a theme's window palette in one appearance.
func Window(id string, dark bool) *Colors {
	return lookFor(id, dark).colors
}

// Terminal is a theme's program palette in one appearance.
func Terminal(id string, dark bool) *terminal.Theme {
	return lookFor(id, dark).term
}

func lookFor(id string, dark bool) *look {
	pair := looks[Resolve(id)]
	if dark {
		return pair[1]
	}
	return pair[0]
}

// UI is the widget theme a window with this palette and appearance draws
// with: the control faces, borders and states of MyGo, recolored from the
// palette so buttons, fields, popovers and the editor agree with the
// window behind them.
func UI(c *ui.Context, k *Colors, dark bool) *ui.Theme {
	var t ui.Theme
	if cur := c.Theme(); cur.Dark == dark {
		t = *cur
	} else if dark {
		t = *ui.DarkTheme()
	} else {
		t = *ui.LightTheme()
	}
	t.Dark = dark
	t.Background = k.BgBottom
	t.Surface = k.Panel
	t.SurfaceHover = k.Panel.Mix(k.Text, 0.06)
	t.SurfacePressed = k.Panel.Mix(k.Text, 0.12)
	t.Border = k.PanelBorder
	t.Text, t.TextMuted = k.Text, k.TextMuted
	white, black := ui.RGB(255, 255, 255), ui.RGB(0, 0, 0)
	t.Accent = k.Accent
	if dark {
		t.AccentHover, t.AccentPressed = k.Accent.Mix(white, 0.2), k.Accent.Mix(black, 0.15)
		t.Selection, t.Focus = k.Accent.Alpha(0.4), k.Accent.Alpha(0.6)
	} else {
		t.AccentHover, t.AccentPressed = k.Accent.Mix(black, 0.12), k.Accent.Mix(black, 0.25)
		t.Selection, t.Focus = k.Accent.Alpha(0.25), k.Accent.Alpha(0.55)
	}
	t.AccentText = accentText(k.Accent)
	t.Radius = Radius
	return &t
}

// accentText is black on a light accent and white on a dark one, read from
// the accent's luminance so a pale theme keeps its accent readable.
func accentText(c ui.Color) ui.Color {
	lum := 0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B)
	if lum > 165 {
		return ui.RGB(24, 24, 27)
	}
	return ui.RGB(255, 255, 255)
}

// EditorTheme colors an open note. The default theme keeps GitHub's syntax
// colors; every other takes its tokens from the same program palette the
// terminal uses, so code and shell agree.
func EditorTheme(t *ui.Theme, id string, dark bool) *editor.Theme {
	th := editor.DefaultTheme(t)
	th.Background = ui.Transparent
	th.ActiveLine = t.Text.Alpha(0.035)
	th.InactiveSelection = t.Selection.Alpha(0.45)
	if Resolve(id) == DefaultID {
		return th
	}
	p := Terminal(id, dark).Palette
	tok := func(k editor.TokenKind, c ui.Color) { th.Tokens[k].Color = c }
	tok(editor.TokenKeyword, p[1])
	tok(editor.TokenOperator, p[1])
	tok(editor.TokenType, p[3])
	tok(editor.TokenFunction, p[5])
	tok(editor.TokenMeta, p[5])
	tok(editor.TokenProperty, p[4])
	tok(editor.TokenConstant, p[4])
	tok(editor.TokenNumber, p[4])
	tok(editor.TokenAttribute, p[4])
	tok(editor.TokenEscape, p[4])
	tok(editor.TokenString, p[2])
	tok(editor.TokenRegexp, p[2])
	tok(editor.TokenTag, p[2])
	tok(editor.TokenInserted, p[2])
	tok(editor.TokenComment, t.TextMuted)
	tok(editor.TokenHeading, p[4])
	tok(editor.TokenLink, p[4])
	tok(editor.TokenCode, p[4])
	tok(editor.TokenDeleted, p[1])
	tok(editor.TokenInvalid, p[1])
	th.Error, th.Warning, th.Info, th.Hint = p[1], p[3], p[4], t.TextMuted
	th.Added, th.Deleted, th.Modified = p[2], p[1], p[4]
	return th
}

// PaintPreview draws a thumbnail of a theme: its background, a card with a
// line of text, a selected chip and a cursor, and the program colors along
// the bottom.
func PaintPreview(p *ui.Painter, r ui.Rect, id string, dark bool) {
	k := Window(id, dark)
	PaintBackground(p, r, k)
	card := ui.Rect{X: r.X + 10, Y: r.Y + 10, W: r.W - 20, H: r.H - 20}
	p.Fill(card, k.CardFocused, 6)
	p.Stroke(card, k.PanelBorder, 6, 1)
	p.Text(card.X+8, card.Y+5, "Aa", 15, k.Text)
	chip := ui.Rect{X: card.X + card.W - 38, Y: card.Y + 6, W: 28, H: 12}
	p.Fill(chip, k.PanelSel, 3)
	p.Text(chip.X+4, chip.Y, "~/", 9, k.Text)
	cursor := ui.Rect{X: card.X + card.W - 10, Y: card.Y + 6, W: 2, H: 14}
	p.Fill(cursor, k.Accent, 1)
	term := Terminal(id, dark)
	for i := range 8 {
		dot := ui.Rect{
			X: card.X + 8 + float32(i)*8,
			Y: card.Y + card.H - 11,
			W: 5, H: 5,
		}
		p.Fill(dot, term.Palette[i], 2.5)
	}
}

func withAlpha(c ui.Color, a float32) ui.Color {
	return ui.RGBA(c.R, c.G, c.B, a)
}

func opaque(c ui.Color) ui.Color {
	return ui.RGB(c.R, c.G, c.B)
}

// pick chooses the first value in a dark window and the second in a light
// one.
func pick[T any](dark bool, whenDark, whenLight T) T {
	if dark {
		return whenDark
	}
	return whenLight
}
