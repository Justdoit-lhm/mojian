package app

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// openSettingsCommand opens Settings, assigned after the view is built:
// a direct call from cmdSettings' initializer would make the command
// table depend on the window that lists it.
var openSettingsCommand func(*app)

func init() {
	openSettingsCommand = func(a *app) { a.openSettingsWindow() }
}

// commandAccel is the accelerator the menus give a command: its own,
// unless the user rebound it in Settings.
func (a *app) commandAccel(c *command) string {
	if c == nil {
		return ""
	}
	if v, ok := a.shortcutKeys[c.id]; ok {
		return v
	}
	return c.accel
}

// commandKeys is the keys a command shows beside its name, in the
// platform's notation.
func (a *app) commandKeys(c *command) string {
	if c == nil {
		return ""
	}
	if v, ok := a.shortcutKeys[c.id]; ok {
		return shortcutDisplay(v)
	}
	if c.display != "" {
		return nativeKeys(c.display)
	}
	return shortcutDisplay(a.commandAccel(c))
}

// menuAccel gives a menu item its keys, none while a shortcut is being
// typed: macOS takes a key bound to a menu before the focused control
// sees it, which would split a pane instead of recording its keys.
func (a *app) menuAccel(c *command) string {
	if a.settingsRecording != "" {
		return ""
	}
	return a.commandAccel(c)
}

// shortcutDisplay writes an accelerator in the keys of the platform, as
// command keys show them: ⌥⇧⌘D on a Mac, Alt+Shift+Ctrl+D elsewhere.
func shortcutDisplay(accel string) string {
	if accel == "" {
		return "None"
	}
	parts := strings.Split(accel, "+")
	if runtime.GOOS != "darwin" {
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			switch p {
			case "CmdOrCtrl":
				out = append(out, "Ctrl")
			default:
				out = append(out, p)
			}
		}
		return strings.Join(out, "+")
	}
	var ctrl, alt, shift, cmd bool
	key := ""
	for _, p := range parts {
		switch p {
		case "CmdOrCtrl", "Super":
			cmd = true
		case "Ctrl":
			ctrl = true
		case "Alt":
			alt = true
		case "Shift":
			shift = true
		default:
			key = shortcutGlyph(p)
		}
	}
	var b strings.Builder
	if ctrl {
		b.WriteString("⌃")
	}
	if alt {
		b.WriteString("⌥")
	}
	if shift {
		b.WriteString("⇧")
	}
	if cmd {
		b.WriteString("⌘")
	}
	b.WriteString(key)
	return b.String()
}

func shortcutGlyph(key string) string {
	switch key {
	case "Left":
		return "←"
	case "Right":
		return "→"
	case "Up":
		return "↑"
	case "Down":
		return "↓"
	case "Enter":
		return "↩"
	case "Escape":
		return "⎋"
	case "Backspace":
		return "⌫"
	case "Delete":
		return "⌦"
	case "Tab":
		return "⇥"
	case "Space":
		return "Space"
	}
	return key
}

// parseAccel reads an accelerator into the modifiers and key a browser
// build binds, where there is no menu bar to take it.
func parseAccel(accel string) (ui.Modifiers, ui.Key, bool) {
	var mods ui.Modifiers
	key := ui.KeyUnknown
	for _, p := range strings.Split(accel, "+") {
		switch p {
		case "CmdOrCtrl":
			if runtime.GOOS == "darwin" {
				mods |= ui.Super
			} else {
				mods |= ui.Ctrl
			}
		case "Super":
			mods |= ui.Super
		case "Ctrl":
			mods |= ui.Ctrl
		case "Alt":
			mods |= ui.Alt
		case "Shift":
			mods |= ui.Shift
		default:
			k, ok := shortcutKeyByName(p)
			if !ok {
				return 0, 0, false
			}
			key = k
		}
	}
	return mods, key, key != ui.KeyUnknown
}

// shortcutKeyByName is the key an accelerator's last part names.
func shortcutKeyByName(name string) (ui.Key, bool) {
	if k, ok := shortcutKeys[name]; ok {
		return k, true
	}
	return 0, false
}

// shortcutKeys are the keys a shortcut may bind, by accelerator name.
var shortcutKeys = func() map[string]ui.Key {
	m := map[string]ui.Key{
		"Enter": ui.KeyEnter, "Tab": ui.KeyTab, "Space": ui.KeySpace,
		"Backspace": ui.KeyBackspace, "Delete": ui.KeyDelete, "Escape": ui.KeyEscape,
		"Home": ui.KeyHome, "End": ui.KeyEnd, "PageUp": ui.KeyPageUp, "PageDown": ui.KeyPageDown,
		"Left": ui.KeyLeft, "Right": ui.KeyRight, "Up": ui.KeyUp, "Down": ui.KeyDown,
		"-": ui.KeyMinus, "=": ui.KeyEqual, ",": ui.KeyComma, ".": ui.KeyPeriod, "/": ui.KeySlash,
		";": ui.KeySemicolon, "'": ui.KeyQuote, "[": ui.KeyBracketLeft, "]": ui.KeyBracketRight,
		"\\": ui.KeyBackslash, "`": ui.KeyBackquote,
	}
	for i, k := range []ui.Key{ui.KeyF1, ui.KeyF2, ui.KeyF3, ui.KeyF4, ui.KeyF5, ui.KeyF6, ui.KeyF7, ui.KeyF8, ui.KeyF9, ui.KeyF10, ui.KeyF11, ui.KeyF12} {
		m["F"+strconv.Itoa(i+1)] = k
	}
	for i, k := range []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9} {
		m[strconv.Itoa(i)] = k
	}
	letters := []ui.Key{ui.KeyA, ui.KeyB, ui.KeyC, ui.KeyD, ui.KeyE, ui.KeyF, ui.KeyG, ui.KeyH, ui.KeyI, ui.KeyJ,
		ui.KeyK, ui.KeyL, ui.KeyM, ui.KeyN, ui.KeyO, ui.KeyP, ui.KeyQ, ui.KeyR, ui.KeyS, ui.KeyT,
		ui.KeyU, ui.KeyV, ui.KeyW, ui.KeyX, ui.KeyY, ui.KeyZ}
	for i, k := range letters {
		m[string(rune('A'+i))] = k
	}
	return m
}()

// shortcutKeyName is the accelerator name of a key.
func shortcutKeyName(key ui.Key) (string, bool) {
	for name, k := range shortcutKeys {
		if k == key {
			return name, true
		}
	}
	return "", false
}

// recordShortcut takes a key typed for a command, reporting whether it
// took it.
func (a *app) recordShortcut(c *command, mods ui.Modifiers, key ui.Key) bool {
	switch {
	case key == ui.KeyEscape && mods == 0:
		a.stopRecording()
		return true
	case (key == ui.KeyBackspace || key == ui.KeyDelete) && mods == 0:
		a.setShortcut(c.id, "")
		a.stopRecording()
		return true
	}
	name, ok := shortcutKeyName(key)
	if !ok {
		return false
	}
	if mods&(ui.Super|ui.Ctrl|ui.Alt) == 0 && !strings.HasPrefix(name, "F") {
		a.settingsNotice = "Add ⌘, ⌥ or ⌃ to a letter; function keys work alone."
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
	a.setShortcut(c.id, strings.Join(append(parts, name), "+"))
	a.stopRecording()
	return true
}

// setShortcut binds a command, taking the keys from any other that had
// them; the default's own keys remove the binding.
func (a *app) setShortcut(id, accel string) {
	if a.shortcutKeys == nil {
		a.shortcutKeys = map[string]string{}
	}
	sc := make(map[string]string, len(a.shortcutKeys))
	for k, v := range a.shortcutKeys {
		sc[k] = v
	}
	a.settingsNotice = ""
	if accel != "" {
		for _, other := range paletteCommands {
			if other.id == id {
				continue
			}
			if strings.EqualFold(a.commandAccel(other), accel) {
				sc[other.id] = ""
				a.settingsNotice = shortcutDisplay(accel) + " was used by " + strings.TrimSuffix(other.title, "…") + "; it is now unbound."
			}
		}
	}
	c := commandByID(id)
	if c != nil && strings.EqualFold(accel, c.accel) {
		delete(sc, id)
	} else {
		sc[id] = accel
	}
	if len(sc) == 0 {
		sc = nil
	}
	a.shortcutKeys = sc
	a.saveSettings()
	a.refreshMenu()
	a.invalidate()
}

func commandByID(id string) *command {
	for _, c := range paletteCommands {
		if c.id == id {
			return c
		}
	}
	return nil
}

// stopRecording ends typing a shortcut and gives the menus their keys
// back.
func (a *app) stopRecording() {
	if a.settingsRecording == "" {
		return
	}
	a.settingsRecording = ""
	a.settingsRecordSeen = false
	a.refreshMenu()
	a.invalidate()
}

// refreshMenu gives the menu bar the accelerators the settings now say.
func (a *app) refreshMenu() {
	if a.win != nil || a.settingsWin != nil {
		mygo.App.SetMenu(a.menu())
	}
}
