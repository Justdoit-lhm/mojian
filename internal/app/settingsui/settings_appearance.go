package settingsui

import (
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
)

// themeLabels name the three appearances, in the value the setting keeps:
// System, Light, Dark.
var themeLabels = []string{"System", "Light", "Dark"}

// settingsAppearancePage offers the appearance as a compact control, since
// it is a choice of two worlds rather than a look, and the look itself as a
// grid of themes, each drawn the way the window would be.
func (u *UI) settingsAppearancePage(c *ui.Context, k *colors) {
	u.settingsSection(c, k, "", func() {
		u.settingsChoiceRow(c, k, "appearance-mode", "Mode",
			"System follows this Mac's light or dark appearance.",
			func() int { return u.Appearance }, func(n int) { u.setAppearance(n) }, themeLabels)
	})
	u.settingsSection(c, k, "THEME", func() {
		dark := u.dark(c)
		ui.Row(c).Padding(16).Gap(10).Wrap().Role(ui.RoleRadioGroup).Label("Theme").
			Children(func() {
				for _, ch := range theme.Choices {
					u.themeTile(c, k, ch, dark)
				}
			})
	})
}

// themeTile is one theme of the grid: a thumbnail of the window it makes,
// its name under it, and a ring in its own accent while it is the one.
func (u *UI) themeTile(c *ui.Context, k *colors, ch theme.Choice, dark bool) {
	sel := theme.Resolve(u.Theme) == ch.ID
	tk := theme.Window(ch.ID, dark)
	tile := ui.ButtonBase(c.Key("theme-" + ch.ID)).Width(132).Padding(6).
		Radius(10).Role(ui.RoleRadio).AccessSelected(sel).
		Label(ch.Name + " theme")
	if sel {
		tile.Border(2, tk.Accent)
	} else {
		tile.Border(2, ui.Transparent)
	}
	if tile.Clicked() && !sel {
		u.setTheme(ch.ID)
		u.saveSettings()
		u.invalidate()
	}
	tile.Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Box(c).Height(80).FillWidth().Radius(7).Clip().
				Draw(func(p *ui.Painter, r ui.Rect) {
					theme.PaintPreview(p, r, ch.ID, dark)
				})
			col, weight := k.TextMuted, 500
			if sel {
				col, weight = k.Text, 650
			}
			ui.Text(c, ch.Name).FontSize(12.5).FontWeight(weight).
				TextColor(col).TextAlign(ui.Center).FillWidth()
		})
	})
}
