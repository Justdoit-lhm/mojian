package settingsui

import (
	"runtime"
	"strings"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

func (u *UI) settingsKeyboardPage(c *ui.Context, k *colors) {
	u.settingsNoticeLine(c, k)
	q := strings.ToLower(strings.TrimSpace(u.KeyFilter))
	shown := 0
	u.settingsSection(c, k, "SHORTCUTS", func() {
		ui.Row(c).Padding(10, 16).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1).Height(30).Padding(0, 8).Gap(6).Radius(radius).
				Background(k.Hover).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
				in := ui.TextInputBase(c, &u.KeyFilter).Grow(1).FontSize(12).
					Placeholder("Filter shortcuts…")
				if in.Changed() {
					u.invalidate()
				}
			})
			if len(u.shortcutKeys()) > 0 {
				if settingsButton(c, "Restore Defaults", false).Clicked() {
					u.stopRecording()
					u.resetShortcuts()
					u.Notice = ""
					u.saveSettings()
					u.refreshMenu()
					u.invalidate()
				}
			}
		})
		ui.Divider(c).Margin(0, 16)
		for _, cmd := range u.commands() {
			if cmd.Accel == "" || !cmd.Available {
				continue
			}
			label := strings.TrimSuffix(cmd.Title, "…")
			keys := cmd.Keys
			if q != "" && !strings.Contains(strings.ToLower(label+" "+keys+" "+cmd.ID), q) {
				continue
			}
			if u.Hits != nil && !u.Hits[label] {
				continue
			}
			u.shortcutRow(c, k, cmd)
			shown++
		}
		if shown == 0 {
			ui.Text(c, "No shortcuts match.").FontSize(12).TextColor(k.TextFaint).
				Padding(14, 16)
		}
	})
	u.settingsSection(c, k, "PANE NAVIGATION", func() {
		u.settingsFixedKeys(c, k, "Move focus", nativeKeys("⌥←  ⌥→  ⌥↑  ⌥↓"),
			"Focus the neighboring pane.")
		u.settingsFixedKeys(c, k, "Move the focused pane", nativeKeys("⌥⇧←  ⌥⇧→  ⌥⇧↑  ⌥⇧↓"),
			"Put it on the neighbor's edge.")
	})
	u.settingsSection(c, k, "TAB SWITCHING", func() {
		u.settingsFixedKeys(c, k, "Show tab", nativeKeys("⌘1…⌘9"), "Go to a tab by its position.")
		u.settingsFixedKeys(c, k, "Move to Trash", nativeKeys("⌘⌫"), "On a file in the file tree.")
	})
	u.settingsNote(c, k, "A key used twice is taken from the command that had it. Escape cancels; Backspace unbinds.")
}

func (u *UI) settingsFixedKeys(c *ui.Context, k *colors, title, keys, detail string) {
	u.settingRow(c, k, title, detail, func() {
		ui.Text(c, keys).FontSize(11.5).TextColor(k.TextFaint).Font(noteFont.Family)
	})
}

// stopRecording ends typing a shortcut and gives the menus their keys
// back.
func (u *UI) stopRecording() {
	if u.Recording == "" {
		return
	}
	u.Recording = ""
	u.RecordSeen = false
	u.refreshMenu()
	u.invalidate()
}

// StopRecording ends shortcut recording, for the window's close path.
func (u *UI) StopRecording() { u.stopRecording() }

// recordShortcut takes a key typed for a command, reporting whether it
// took it.
func (u *UI) recordShortcut(cmd Command, mods ui.Modifiers, key ui.Key) bool {
	switch {
	case key == ui.KeyEscape && mods == 0:
		u.stopRecording()
		return true
	case (key == ui.KeyBackspace || key == ui.KeyDelete) && mods == 0:
		u.setShortcut(cmd.ID, "")
		u.stopRecording()
		return true
	}
	name, ok := shortcutKeyName(key)
	if !ok {
		return false
	}
	if mods&(ui.Super|ui.Ctrl|ui.Alt) == 0 && !strings.HasPrefix(name, "F") {
		u.Notice = "Add ⌘, ⌥ or ⌃ to a letter; function keys work alone."
		return true
	}
	var parts []string
	if mods&ui.Super != 0 {
		if runtime.GOOS == "darwin" {
			parts = append(parts, "CmdOrCtrl")
		} else {
			parts = append(parts, "Super")
		}
	}
	if mods&ui.Ctrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&ui.Alt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&ui.Shift != 0 {
		parts = append(parts, "Shift")
	}
	u.setShortcut(cmd.ID, strings.Join(append(parts, name), "+"))
	u.stopRecording()
	return true
}

// not the default, and a chip that records the keys typed for it.
func (u *UI) shortcutRow(c *ui.Context, k *colors, cmd Command) {
	label := strings.TrimSuffix(cmd.Title, "…")
	keys := cmd.Keys
	custom := false
	if v, ok := u.shortcutKeys()[cmd.ID]; ok && !strings.EqualFold(v, cmd.Accel) {
		custom = true
	}
	recording := u.Recording == cmd.ID
	u.settingRow(c, k, label, "", func() {
		if custom {
			back := ui.ButtonBase(c.Key("default-"+cmd.ID)).Size(24, 24).Radius(6).
				Center().TextColor(k.TextFaint).Label("Use the default shortcut").
				Tooltip("Use " + cmd.Keys)
			if back.Hovered() {
				back.Background(k.Pressed).TextColor(k.Text)
			}
			back.Children(func() { ui.Icon(c, icons.Icon("refresh-cw")).Size(12, 12) })
			if back.Clicked() {
				u.ResetShortcut(cmd.ID)
				u.Notice = ""
				u.saveSettings()
				u.refreshMenu()
				u.invalidate()
			}
		}
		chip := ui.ButtonBase(c.Key("keys-"+cmd.ID)).Height(26).MinWidth(92).
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
				u.stopRecording()
			} else {
				u.Recording = cmd.ID
				u.RecordSeen = false
				u.Notice = ""
				u.RecordRef.Focus(c)
				u.refreshMenu()
				u.invalidate()
			}
		}
		if recording {
			if u.RecordRef.Focused(c) {
				u.RecordSeen = true
			} else if u.RecordSeen && !chip.Pressed() {
				u.later(u.stopRecording)
			}
			chip.Bind(&u.RecordRef).HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputText {
					return true
				}
				if ev.Kind != ui.InputKeyDown {
					return false
				}
				return u.recordShortcut(cmd, ev.Mods, ev.Key)
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
