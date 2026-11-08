package settingsui

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

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
