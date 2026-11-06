package app

import (
	"fmt"
	"runtime"

	"mojian/internal/icons"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// settingsPage is the app's preferences: appearance, editor type, the
// terminal's type, and where this project lives.
func (a *app) settingsPage(c *ui.Context, k *colors) {
	c.OnShortcut(0, ui.KeyEscape, func() { a.closeSettings() })
	card := ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Radius(cardR).Clip().
		Background(k.CardFocused).Border(1, k.CardBorderFocused).
		Shadow(0, 1, 2, 0, k.Shadow)
	card.Children(func() {
		ui.Row(c).Height(headerH).Padding(0, 8, 0, 12).Gap(7).
			AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Icon("settings")).Size(15, 15).TextColor(k.TextMuted)
			ui.Text(c, "Settings").FontSize(12.5).FontWeight(600).TextColor(k.Text)
			ui.Spacer(c)
			if iconButton(c, k, "x", "Close Settings", 25, 13).Clicked() {
				a.later(func() { a.closeSettings() })
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).MaxWidth(780).AlignSelf(ui.Center).
				Padding(28, 32, 44, 32).Gap(26).Children(func() {
				a.appearanceSettings(c, k)
				a.editorSettings(c, k)
				a.terminalSettings(c, k)
				a.aboutSettings(c, k)
			})
		})
	})
}

func (a *app) settingsSection(c *ui.Context, k *colors, title string, fn func()) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, title).FontSize(10).FontWeight(650).
			TextColor(k.TextFaint).Padding(2, 2)
		panel := ui.Column(c).Background(k.Card).Border(1, k.CardBorder).
			Radius(12).Clip()
		panel.Children(fn)
	})
}

func (a *app) settingRow(c *ui.Context, k *colors, title, detail string, control func()) {
	ui.Row(c).MinHeight(58).Padding(0, 16).Gap(18).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(12.5).FontWeight(550).TextColor(k.Text)
			if detail != "" {
				ui.Text(c, detail).FontSize(11.5).TextColor(k.TextFaint).
					SingleLine().Ellipsis("…")
			}
		})
		ui.Row(c).Shrink(0).AlignItems(ui.Center).Children(control)
	})
}

func (a *app) settingDivider(c *ui.Context, k *colors) {
	ui.Divider(c).Margin(0, 16)
}

func (a *app) appearanceSettings(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "APPEARANCE", func() {
		a.appearanceRow(c, k)
	})
}

func (a *app) appearanceRow(c *ui.Context, k *colors) {
	a.settingRow(c, k, "Theme", "Follow the system, or keep one look.", func() {
		ui.Row(c).Padding(3).Radius(9).Background(k.Track).Border(1, k.TrackBorder).
			Gap(2).Role(ui.RoleRadioGroup).Label("Theme").Children(func() {
			for i, label := range []string{"System", "Light", "Dark"} {
				i, label := i, label
				b := ui.Row(c).Padding(5, 12).Radius(7).Cursor(ui.CursorPointer).
					Role(ui.RoleRadio).Label(label).Selected(a.appearance == i)
				if a.appearance == i {
					b.Background(k.PanelSel)
				} else if b.Hovered() {
					b.Background(k.Hover)
				}
				if b.Clicked() {
					a.setAppearance(i)
					a.invalidate()
				}
				b.Children(func() {
					col := k.TextMuted
					if a.appearance == i {
						col = k.Text
					}
					ui.Text(c, label).FontSize(11.5).FontWeight(550).TextColor(col)
				})
			}
		})
	})
}

func (a *app) fontSizeRow(c *ui.Context, k *colors, title, detail string, value *float64, def, lo, hi float64, format string) {
	if *value <= 0 {
		*value = def
	}
	changed := false
	a.settingRow(c, k, title, detail, func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%"+format, *value)+" pt").FontSize(11.5).TextColor(k.TextMuted).
				Width(46).TextAlign(ui.End).Shrink(0)
			before := *value
			ui.Slider(c, value, lo, hi).Width(170).Label(title)
			if *value != before {
				changed = true
			}
		})
	})
	if changed {
		a.saveSettings()
		a.invalidate()
	}
}

func (a *app) editorSettings(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "EDITOR", func() {
		a.fontSizeRow(c, k, "Editor font size",
			"The size of source and prose, in points.", &a.editorFontSize, 12.5, 10, 20, ".1f")
		a.settingDivider(c, k)
		a.settingRow(c, k, "Line numbers",
			"Number the lines of code files; Markdown stays clean.", func() {
				show := !a.hideLineNumbers
				sw := ui.Switch(c, &show).Label("Line numbers")
				if sw.Changed() {
					a.hideLineNumbers = !show
					a.saveSettings()
					a.invalidate()
				}
			})
		a.settingDivider(c, k)
		a.settingRow(c, k, "Preview Markdown by default",
			"Open .md files rendered, with the Edit button to switch back.", func() {
				sw := ui.Switch(c, &a.markdownPreview).Label("Preview Markdown by default")
				if sw.Changed() {
					a.saveSettings()
					a.invalidate()
				}
			})
	})
}

func (a *app) terminalSettings(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "TERMINAL", func() {
		a.fontSizeRow(c, k, "Terminal font size",
			"Applies to new shells; existing terminals keep their size.",
			&a.terminalFontSize, 12.5, 9, 20, ".1f")
	})
}

func (a *app) aboutSettings(c *ui.Context, k *colors) {
	version := mygo.App.Version()
	if version == "" {
		version = "0.1.0"
	}
	project := "No project"
	if a.store != nil {
		project = a.store.dir
	}
	a.settingsSection(c, k, "ABOUT", func() {
		a.aboutRow(c, k, "Version", version)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "System", runtime.GOOS+" · "+runtime.GOARCH)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Project", project)
	})
}

func (a *app) aboutRow(c *ui.Context, k *colors, title, value string) {
	ui.Row(c).MinHeight(46).Padding(0, 16).Gap(18).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(12.5).TextColor(k.Text).Grow(1)
		ui.Text(c, value).FontSize(11.5).TextColor(k.TextFaint).
			SingleLine().Ellipsis("…").MaxWidth(430).TextAlign(ui.End)
	})
}
