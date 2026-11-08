package app

import (
	"strings"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

func (a *app) settingsKeyboardPage(c *ui.Context, k *colors) {
	a.settingsNoticeLine(c, k)
	q := strings.ToLower(strings.TrimSpace(a.settingsKeyFilter))
	shown := 0
	a.settingsSection(c, k, "SHORTCUTS", func() {
		ui.Row(c).Padding(10, 16).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1).Height(30).Padding(0, 8).Gap(6).Radius(radius).
				Background(k.Hover).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
				in := ui.TextInputBase(c, &a.settingsKeyFilter).Grow(1).FontSize(12).
					Placeholder("Filter shortcuts…")
				if in.Changed() {
					a.invalidate()
				}
			})
			if len(a.shortcutKeys) > 0 {
				if settingsButton(c, "Restore Defaults", false).Clicked() {
					a.stopRecording()
					a.shortcutKeys = nil
					a.settingsNotice = ""
					a.saveSettings()
					a.refreshMenu()
					a.invalidate()
				}
			}
		})
		ui.Divider(c).Margin(0, 16)
		for _, cmd := range paletteCommands {
			if cmd.accel == "" || !commandAvailable(cmd) {
				continue
			}
			label := strings.TrimSuffix(cmd.title, "…")
			keys := a.commandKeys(cmd)
			if q != "" && !strings.Contains(strings.ToLower(label+" "+keys+" "+cmd.id), q) {
				continue
			}
			if a.settingsHits != nil && !a.settingsHits[label] {
				continue
			}
			a.shortcutRow(c, k, cmd)
			shown++
		}
		if shown == 0 {
			ui.Text(c, "No shortcuts match.").FontSize(12).TextColor(k.TextFaint).
				Padding(14, 16)
		}
	})
	a.settingsSection(c, k, "PANE NAVIGATION", func() {
		a.settingsFixedKeys(c, k, "Move focus", nativeKeys("⌥←  ⌥→  ⌥↑  ⌥↓"),
			"Focus the neighboring pane.")
		a.settingsFixedKeys(c, k, "Move the focused pane", nativeKeys("⌥⇧←  ⌥⇧→  ⌥⇧↑  ⌥⇧↓"),
			"Put it on the neighbor's edge.")
	})
	a.settingsSection(c, k, "TAB SWITCHING", func() {
		a.settingsFixedKeys(c, k, "Show tab", nativeKeys("⌘1…⌘9"), "Go to a tab by its position.")
		a.settingsFixedKeys(c, k, "Move to Trash", nativeKeys("⌘⌫"), "On a file in the file tree.")
	})
	a.settingsNote(c, k, "A key used twice is taken from the command that had it. Escape cancels; Backspace unbinds.")
}

func (a *app) settingsFixedKeys(c *ui.Context, k *colors, title, keys, detail string) {
	a.settingRow(c, k, title, detail, func() {
		ui.Text(c, keys).FontSize(11.5).TextColor(k.TextFaint).Font(noteFont.Family)
	})
}

// shortcutRow is a command's shortcut: its keys, a reset when they are
// not the default, and a chip that records the keys typed for it.

func (a *app) shortcutRow(c *ui.Context, k *colors, cmd *command) {
	label := strings.TrimSuffix(cmd.title, "…")
	keys := a.commandKeys(cmd)
	custom := false
	if v, ok := a.shortcutKeys[cmd.id]; ok && !strings.EqualFold(v, cmd.accel) {
		custom = true
	}
	recording := a.settingsRecording == cmd.id
	a.settingRow(c, k, label, "", func() {
		if custom {
			back := ui.ButtonBase(c.Key("default-"+cmd.id)).Size(24, 24).Radius(6).
				Center().TextColor(k.TextFaint).Label("Use the default shortcut").
				Tooltip("Use " + a.commandKeys(cmd))
			if back.Hovered() {
				back.Background(k.Pressed).TextColor(k.Text)
			}
			back.Children(func() { ui.Icon(c, icons.Icon("refresh-cw")).Size(12, 12) })
			if back.Clicked() {
				delete(a.shortcutKeys, cmd.id)
				if len(a.shortcutKeys) == 0 {
					a.shortcutKeys = nil
				}
				a.settingsNotice = ""
				a.saveSettings()
				a.refreshMenu()
				a.invalidate()
			}
		}
		chip := ui.ButtonBase(c.Key("keys-"+cmd.id)).Height(26).MinWidth(92).
			Padding(0, 10).Radius(radius).Center().Background(k.Hover).
			Label("Shortcut for " + label)
		switch {
		case recording:
			chip.Background(k.PanelSel)
		case chip.Hovered():
			chip.Background(k.Pressed)
		}
		if chip.Clicked() {
			if recording {
				a.stopRecording()
			} else {
				a.settingsRecording = cmd.id
				a.settingsRecordSeen = false
				a.settingsNotice = ""
				a.settingsRecordRef.Focus(c)
				a.refreshMenu()
				a.invalidate()
			}
		}
		if recording {
			if a.settingsRecordRef.Focused(c) {
				a.settingsRecordSeen = true
			} else if a.settingsRecordSeen && !chip.Pressed() {
				a.later(a.stopRecording)
			}
			chip.Bind(&a.settingsRecordRef).HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputText {
					return true
				}
				if ev.Kind != ui.InputKeyDown {
					return false
				}
				return a.recordShortcut(cmd, ev.Mods, ev.Key)
			})
		}
		chip.Children(func() {
			switch {
			case recording:
				ui.Text(c, "Type shortcut…").FontSize(11.5).TextColor(k.Text)
			case keys == "None":
				ui.Text(c, "None").FontSize(11.5).TextColor(k.TextFaint)
			default:
				ui.Text(c, keys).FontSize(11.5).TextColor(k.Text).Font(noteFont.Family).
					SingleLine()
			}
		})
	})
}
