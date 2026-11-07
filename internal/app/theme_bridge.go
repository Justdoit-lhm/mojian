package app

import (
	"mojian/internal/icons"
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// colors is the window's palette; the app's view code names the theme
// through it so a palette change is one package's concern.
type colors = theme.Colors

var noteFont = theme.NoteFont

func colorsOf(c *ui.Context) *colors { return theme.Of(c) }

func paintBackground(p *ui.Painter, r ui.Rect, k *colors) { theme.PaintBackground(p, r, k) }

func editorThemeOf(t *ui.Theme) *editor.Theme { return theme.EditorTheme(t) }

func registerFonts() { theme.RegisterFonts() }

// brandMark is the Mojian mark, in the window's ink or reversed for dark.
func brandMark(c *ui.Context) *ui.SVG {
	if c.Theme().Dark {
		return icons.Brand("mark-reversed")
	}
	return icons.Brand("mark")
}
