package theme

import (
	"math"
	"testing"

	"mojian/internal/prefs"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func TestResolve(t *testing.T) {
	got := map[string]string{
		"":     DefaultID,
		"nope": DefaultID,
		"dusk": "dusk",
		"DUSK": "dusk",
	}
	for in, want := range got {
		if id := Resolve(in); id != want {
			t.Fatalf("Resolve(%q) = %q, want %q", in, id, want)
		}
	}
}

// TestDefaultKeepsCurrentPalette pins the default theme to the colors
// Mojian drew before themes existed: choosing it must not change a window.
func TestDefaultKeepsCurrentPalette(t *testing.T) {
	for _, dark := range []bool{false, true} {
		base := lightColors
		if dark {
			base = darkColors
		}
		k := Window(DefaultID, dark)
		for name, pair := range map[string][2]ui.Color{
			"BgTop":       {k.BgTop, base.BgTop},
			"BgMid":       {k.BgMid, base.BgMid},
			"BgLow":       {k.BgLow, base.BgLow},
			"BgBottom":    {k.BgBottom, base.BgBottom},
			"Card":        {k.Card, base.Card},
			"CardFocused": {k.CardFocused, base.CardFocused},
			"Text":        {k.Text, base.Text},
		} {
			if pair[0] != pair[1] {
				t.Fatalf("dark=%v: %s = %v, want %v", dark, name, pair[0], pair[1])
			}
		}
	}
}

// TestDark pins the appearance preference to a window's dark flag.
func TestDark(t *testing.T) {
	cases := []struct {
		appearance int
		system     bool
		want       bool
	}{
		{0, false, false},
		{0, true, true},
		{1, true, false},
		{2, false, true},
		{9, true, true},
	}
	for _, tc := range cases {
		if got := Dark(tc.appearance, tc.system); got != tc.want {
			t.Fatalf("Dark(%d, %v) = %v, want %v", tc.appearance, tc.system, got, tc.want)
		}
	}
}

// luminance is a color's WCAG relative luminance.
func luminance(c ui.Color) float64 {
	ch := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

func contrast(a, b ui.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// TestThemesReadable holds every theme to the contrast a window's text
// needs over the card it sits on.
func TestThemesReadable(t *testing.T) {
	for _, ch := range Choices {
		for _, dark := range []bool{false, true} {
			k := Window(ch.ID, dark)
			bg := k.CardFocused.Over(k.BgMid)
			if c := contrast(k.Text, bg); c < 4.5 {
				t.Fatalf("%s dark=%v: text contrast %.2f, want 4.5", ch.ID, dark, c)
			}
			if c := contrast(k.TextMuted, bg); c < 3.0 {
				t.Fatalf("%s dark=%v: muted contrast %.2f, want 3.0", ch.ID, dark, c)
			}
		}
	}
}

func TestTerminalMatchesWindow(t *testing.T) {
	for _, ch := range Choices {
		for _, dark := range []bool{false, true} {
			k := Window(ch.ID, dark)
			th := Terminal(ch.ID, dark)
			if th.Foreground != k.Text {
				t.Fatalf("%s dark=%v: terminal text %v, window %v", ch.ID, dark, th.Foreground, k.Text)
			}
			if th.Cursor != k.Accent {
				t.Fatalf("%s dark=%v: cursor %v, accent %v", ch.ID, dark, th.Cursor, k.Accent)
			}
			for i, c := range th.Palette {
				if c == (ui.Color{}) {
					t.Fatalf("%s dark=%v: palette %d is unset", ch.ID, dark, i)
				}
			}
		}
	}
}

func TestEditorFollowsTheme(t *testing.T) {
	got := EditorTheme(ui.DarkTheme(), "meadow", true).Tokens[editor.TokenKeyword].Color
	if want := Terminal("meadow", true).Palette[1]; got != want {
		t.Fatalf("meadow keyword %v, want the palette's red %v", got, want)
	}
	def := EditorTheme(ui.DarkTheme(), DefaultID, true)
	if got, want := def.Tokens[editor.TokenKeyword].Color, ui.Hex("#ff7b72"); got != want {
		t.Fatalf("the default keyword %v, want GitHub's %v", got, want)
	}
}

func TestDefaultThemeMatchesPrefs(t *testing.T) {
	if got := prefs.Default().Theme; got != DefaultID {
		t.Fatalf("default theme preference %q, want %q", got, DefaultID)
	}
}
