package settingsui

import (
	"strconv"

	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
)

// themeLabels name the three appearances, in the value the setting keeps:
// System, Light, Dark.
var themeLabels = []string{"System", "Light", "Dark"}

// settingsAppearancePage shows the appearance as the window it makes: the
// same gradient and card the user is looking at, in each palette.
func (u *UI) settingsAppearancePage(c *ui.Context, k *colors) {
	u.settingsSection(c, k, "THEME", func() {
		ui.Row(c).Padding(16).Gap(10).Role(ui.RoleRadioGroup).Label("Theme").
			Children(func() {
				for n, label := range themeLabels {
					n, label := n, label
					sel := u.Appearance == n
					tile := ui.ButtonBase(c.Key("theme-" + strconv.Itoa(n))).Grow(1).
						Height(96).Padding(6).Radius(10).Role(ui.RoleRadio).
						AccessSelected(sel).Label(label + " theme")
					if sel {
						tile.Border(2, c.Theme().Accent)
					} else {
						tile.Border(2, ui.Transparent)
					}
					if tile.Clicked() && !sel {
						u.chooseAppearance(n)
					}
					tile.Children(func() {
						ui.Column(c).FillWidth().Gap(6).Children(func() {
							ui.Box(c).Height(54).FillWidth().Radius(7).Clip().
								Draw(func(p *ui.Painter, r ui.Rect) {
									paintThemePreview(p, r, n)
								})
							col, weight := k.TextMuted, 500
							if sel {
								col, weight = k.Text, 650
							}
							ui.Text(c, label).FontSize(12.5).FontWeight(weight).
								TextColor(col).TextAlign(ui.Center).FillWidth()
						})
					})
				}
			})
	})
}

// chooseAppearance applies one of the three appearances and keeps it.
func (u *UI) chooseAppearance(n int) {
	u.Appearance = n
	u.syncAppearance()
	u.saveSettings()
	u.invalidate()
}

// paintThemePreview fills a preview with the window's background and its
// card, in one palette or in both.
func paintThemePreview(p *ui.Painter, r ui.Rect, which int) {
	light, dark := theme.Palette(false), theme.Palette(true)
	switch which {
	case 1:
		theme.PaintBackground(p, r, &light)
		p.Fill(previewCard(r), light.Card, 5)
	case 2:
		theme.PaintBackground(p, r, &dark)
		p.Fill(previewCard(r), dark.Card, 5)
	default:
		half := ui.Rect{X: r.X, Y: r.Y, W: r.W / 2, H: r.H}
		rest := ui.Rect{X: r.X + half.W, Y: r.Y, W: r.W - half.W, H: r.H}
		p.Clip(half, 0, func() { theme.PaintBackground(p, r, &light) })
		p.Clip(rest, 0, func() { theme.PaintBackground(p, r, &dark) })
		p.Fill(previewCard(half), light.Card, 5)
		p.Fill(previewCard(rest), dark.Card, 5)
	}
}

// previewCard is the inside of a preview: the mini section card, 12 DIPs
// in from the surface it sits on.
func previewCard(r ui.Rect) ui.Rect {
	return ui.Rect{X: r.X + 12, Y: r.Y + 12, W: r.W - 24, H: r.H - 24}
}
