// Package platform holds the OS-dependent spelling of the app's
// shortcuts and file-manager names, and the features a platform has.
package platform

import (
	"runtime"
	"strings"
)

// TerminalsSupported reports whether the session server, and with it
// terminals, SSH and remote projects, runs on this platform.
func TerminalsSupported() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "linux"
}

// NativeKeys writes a shortcut of the command tables, which use the Mac's
// symbols, in the keys of the platform: Ctrl+Shift+O rather than ⇧⌘O.
func NativeKeys(s string) string {
	return NativeKeysFor(runtime.GOOS, s)
}

// NativeKeysFor is NativeKeys for goos.
func NativeKeysFor(goos, s string) string {
	if goos == "darwin" || s == "" {
		return s
	}
	var out strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if !isShortcutModifier(runes[i]) {
			out.WriteRune(runes[i])
			i++
			continue
		}
		ctrl, alt, shift := false, false, false
		j := i
		for j < len(runes) && isShortcutModifier(runes[j]) {
			switch runes[j] {
			case '⌘', '⌃':
				ctrl = true
			case '⌥':
				alt = true
			case '⇧':
				shift = true
			}
			j++
		}
		key := ""
		if j < len(runes) {
			key = shortcutKey(runes[j])
		}
		if key == "" {
			out.WriteRune(runes[i])
			i++
			continue
		}
		var parts []string
		if ctrl {
			parts = append(parts, "Ctrl")
		}
		if alt {
			parts = append(parts, "Alt")
		}
		if shift {
			parts = append(parts, "Shift")
		}
		parts = append(parts, key)
		out.WriteString(strings.Join(parts, "+"))
		i = j + 1
	}
	return out.String()
}

func isShortcutModifier(r rune) bool {
	return r == '⌘' || r == '⌥' || r == '⇧' || r == '⌃'
}

func shortcutKey(r rune) string {
	switch r {
	case '⌫':
		return "Backspace"
	case '⇥':
		return "Tab"
	case '←':
		return "Left"
	case '→':
		return "Right"
	case '↑':
		return "Up"
	case '↓':
		return "Down"
	case '↩':
		return "Enter"
	case '⎋':
		return "Escape"
	}
	return string(r)
}

func fileManagerNameFor(goos string) string {
	if goos == "darwin" {
		return "Finder"
	}
	return "Files"
}

// ThisComputerName labels the local server on the Hosts page.
func ThisComputerName() string {
	if runtime.GOOS == "darwin" {
		return "This Mac"
	}
	return "This Computer"
}

// RevealLabel is the menu item that shows an item in the file manager.
func RevealLabel() string {
	return revealLabelFor(runtime.GOOS)
}

func revealLabelFor(goos string) string {
	return "Reveal in " + fileManagerNameFor(goos)
}
