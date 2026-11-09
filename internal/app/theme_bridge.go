package app

import (
	"mojian/internal/icons"
	"mojian/internal/theme"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// radius is the package's corner radius, shared by cards, panes, fields,
// buttons, popovers and dialogs.
const radius = theme.Radius

// colors is the window's palette; the app's view code names the theme
// through it so a palette change is one package's concern.
type colors = theme.Colors

var noteFont = theme.NoteFont

// dark is the appearance the window draws in: the user's choice, or what
// the system says while the choice is System.
func (a *app) dark(c *ui.Context) bool { return theme.Dark(a.Appearance, c.Theme().Dark) }

// colorsOf is the palette of the open theme in that appearance.
func (a *app) colorsOf(c *ui.Context) *colors {
	return theme.Window(a.Theme, a.dark(c))
}

// themeName resolves a theme id, so a settings file naming one this build
// does not have still opens the default window.
func themeName(id string) string { return theme.Resolve(id) }

// windowTheme is the widget theme the frame draws with.
func windowTheme(c *ui.Context, k *colors, dark bool) *ui.Theme { return theme.UI(c, k, dark) }

// codeTheme colors an open note in a theme.
func codeTheme(t *ui.Theme, id string, dark bool) *editor.Theme {
	return theme.EditorTheme(t, id, dark)
}

// terminalTheme is a theme's program palette in one appearance.
func terminalTheme(id string, dark bool) *terminal.Theme { return theme.Terminal(id, dark) }

// themeBgMid is the middle stop of a theme's background gradient in one
// appearance, which a window paints before its view draws.
func themeBgMid(id string, dark bool) ui.Color { return theme.Window(id, dark).BgMid }

func paintBackground(p *ui.Painter, r ui.Rect, k *colors) { theme.PaintBackground(p, r, k) }

func registerFonts() { theme.RegisterFonts() }

// brandMark is the Mojian mark, in the window's ink or reversed for dark.
func brandMark(c *ui.Context) *ui.SVG {
	if c.Theme().Dark {
		return icons.Brand("mark-reversed")
	}
	return icons.Brand("mark")
}
