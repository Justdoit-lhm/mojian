package settingsui

import (
	"strconv"
	"strings"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

func (u *UI) fontRow(c *ui.Context, k *colors, title, detail string, which int, family string) {
	label := strings.TrimSpace(family)
	if label == "" {
		if which == 0 {
			label = "JetBrains Mono · default"
		} else {
			label = "System monospace · default"
		}
	}
	u.settingRow(c, k, title, detail, func() {
		b := ui.ButtonBase(c.Key("font-"+strconv.Itoa(which))).Height(30).
			Padding(0, 10).Radius(8).Gap(7).Label(title + ": " + label)
		if b.Hovered() {
			b.Background(k.Hover)
		}
		if b.Clicked() {
			u.FontPick, u.FontPickFor, u.FontSearch = true, which, ""
		}
		b.Children(func() {
			ui.Text(c, label).FontSize(12).TextColor(k.Text).SingleLine().
				Ellipsis("…").MaxWidth(200)
			ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
		})
	})
}

// editorFont is the effective editor face: the chosen family with the
// app's monospace fallbacks, at the chosen size.
func (u *UI) editorFont() ui.Font {
	f := noteFont
	if fam := strings.TrimSpace(u.EditorFontFamily); fam != "" {
		f.Family = fam + ", " + f.Family
	}
	f.Size = float32(u.fontSize())
	return f
}

// installedFonts lists the families the system and the app provide,
// loaded once when a font chooser first opens.
func (u *UI) installedFonts() []ui.FontFamily {
	u.FontsOnce.Do(func() { u.Fonts = ui.FontFamilies() })
	return u.Fonts
}

// fontPicker is the searchable list of installed families, opened by the
// font rows.
func (u *UI) fontPicker(c *ui.Context, k *colors) {
	if !u.FontPick {
		return
	}
	ui.DialogBase(c, &u.FontPick, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Width(500).MaxHeight(480).Radius(radius).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).Clip()
		ui.Column(c).Children(func() {
			title := "Editor font"
			if u.FontPickFor == 1 {
				title = "Terminal font"
			}
			ui.Row(c).Padding(12, 14).Gap(9).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(k.Text)
			})
			ui.Row(c).Padding(10, 12).Children(func() {
				u.settingsSearchField(c, k, "font-search", &u.FontSearch,
					"Search fonts", "Search installed fonts…", nil).AutoFocus()
			})
			q := strings.ToLower(strings.TrimSpace(u.FontSearch))
			current := u.EditorFontFamily
			if u.FontPickFor == 1 {
				current = u.TerminalFontFamily
			}
			ui.Scroll(c).Height(350).Padding(6).Children(func() {
				shown := 0
				def := "JetBrains Mono · default"
				if u.FontPickFor == 1 {
					def = "System monospace · default"
				}
				row := ui.Row(c.Key("picker-default")).Height(32).Padding(0, 10).Gap(8).
					Radius(8).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(def)
				if current == "" {
					row.Background(k.PanelSel)
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					if u.FontPickFor == 1 {
						u.TerminalFontFamily = ""
					} else {
						u.EditorFontFamily = ""
					}
					u.FontPick = false
					u.saveSettings()
					u.invalidate()
				}
				row.Children(func() {
					ui.Text(c, def).FontSize(13).TextColor(k.Text)
					if current == "" {
						ui.Spacer(c)
						ui.Icon(c, icons.Icon("check")).Size(13, 13).TextColor(k.TextMuted)
					}
				})
				shown++
				for _, f := range u.installedFonts() {
					f := f
					if q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
						continue
					}
					shown++
					row := ui.Row(c.Key("picker-"+f.Name)).Height(32).Padding(0, 10).Gap(8).
						Radius(8).Cursor(ui.CursorPointer).Role(ui.RoleButton).
						Label(f.Name)
					if f.Name == current {
						row.Background(k.PanelSel)
					} else if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() {
						if u.FontPickFor == 1 {
							u.TerminalFontFamily = f.Name
						} else {
							u.EditorFontFamily = f.Name
						}
						u.FontPick = false
						u.saveSettings()
						u.invalidate()
					}
					row.Children(func() {
						ui.Text(c, f.Name).Font(f.Name).FontSize(13).
							TextColor(k.Text).SingleLine().Ellipsis("…").
							Shrink(1).MinWidth(0)
						if f.Monospaced {
							ui.Text(c, "Mono").FontSize(9).FontWeight(650).
								TextColor(k.TextFaint).Padding(1, 4).Radius(radius).
								Background(k.Hover).Shrink(0)
						}
						if f.Name == current {
							ui.Spacer(c)
							ui.Icon(c, icons.Icon("check")).Size(13, 13).TextColor(k.TextMuted)
						}
					})
				}
				if shown == 0 {
					ui.Text(c, "No matching font").FontSize(12).TextColor(k.TextFaint).Padding(14)
				}
			})
		})
	})
}
