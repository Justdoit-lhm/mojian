package app

import (
	"strconv"
	"strings"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

func (a *app) fontRow(c *ui.Context, k *colors, title, detail string, which int, family string) {
	label := strings.TrimSpace(family)
	if label == "" {
		if which == 0 {
			label = "JetBrains Mono · default"
		} else {
			label = "System monospace · default"
		}
	}
	a.settingRow(c, k, title, detail, func() {
		b := ui.ButtonBase(c.Key("font-"+strconv.Itoa(which))).Padding(5, 10).Radius(radius).
			Gap(7).Label(title + ": " + label)
		if b.Hovered() {
			b.Background(k.Pressed)
		}
		if b.Clicked() {
			a.fontPick, a.fontPickFor, a.fontSearch = true, which, ""
		}
		b.Children(func() {
			ui.Text(c, label).FontSize(11.5).TextColor(k.Text).SingleLine().
				Ellipsis("…").MaxWidth(190)
			ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
		})
	})
}

// editorFont is the effective editor face: the chosen family with the
// app's monospace fallbacks, at the chosen size.
func (a *app) editorFont() ui.Font {
	f := noteFont
	if fam := strings.TrimSpace(a.EditorFontFamily); fam != "" {
		f.Family = fam + ", " + f.Family
	}
	f.Size = float32(a.fontSize())
	return f
}

// installedFonts lists the families the system and the app provide,
// loaded once when a font chooser first opens.
func (a *app) installedFonts() []ui.FontFamily {
	a.fontsOnce.Do(func() { a.fonts = ui.FontFamilies() })
	return a.fonts
}

// fontPicker is the searchable list of installed families, opened by the
// font rows.
func (a *app) fontPicker(c *ui.Context, k *colors) {
	if !a.fontPick {
		return
	}
	ui.DialogBase(c, &a.fontPick, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Width(500).MaxHeight(480).Radius(radius).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).Clip()
		ui.Column(c).Children(func() {
			ui.Row(c).Padding(12, 14).Gap(9).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(15, 15).TextColor(k.TextFaint)
				ui.TextInputBase(c, &a.fontSearch).Grow(1).FontSize(13).
					Placeholder("Search installed fonts…").AutoFocus()
			})
			q := strings.ToLower(strings.TrimSpace(a.fontSearch))
			current := a.EditorFontFamily
			if a.fontPickFor == 1 {
				current = a.TerminalFontFamily
			}
			ui.Scroll(c).Height(350).Padding(6).Children(func() {
				shown := 0
				def := "JetBrains Mono · default"
				if a.fontPickFor == 1 {
					def = "System monospace · default"
				}
				row := ui.Row(c).Key("picker-default").Height(34).Padding(0, 10).Gap(8).
					Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(def)
				if current == "" {
					row.Background(k.PanelSel)
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					if a.fontPickFor == 1 {
						a.TerminalFontFamily = ""
					} else {
						a.EditorFontFamily = ""
					}
					a.fontPick = false
					a.saveSettings()
					a.invalidate()
				}
				row.Children(func() {
					ui.Text(c, def).FontSize(13).TextColor(k.Text)
				})
				shown++
				for _, f := range a.installedFonts() {
					f := f
					if q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
						continue
					}
					shown++
					row := ui.Row(c).Key("picker-"+f.Name).Height(34).Padding(0, 10).Gap(8).
						Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).
						Label(f.Name)
					if f.Name == current {
						row.Background(k.PanelSel)
					} else if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() {
						if a.fontPickFor == 1 {
							a.TerminalFontFamily = f.Name
						} else {
							a.EditorFontFamily = f.Name
						}
						a.fontPick = false
						a.saveSettings()
						a.invalidate()
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
					})
				}
				if shown == 0 {
					ui.Text(c, "No matching font").FontSize(12).TextColor(k.TextFaint).Padding(14)
				}
			})
		})
	})
}
