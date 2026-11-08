package app

import (
	"strings"
	"testing"
)

func TestNativeKeys(t *testing.T) {
	for _, tt := range []struct {
		goos, in, want string
	}{
		{"darwin", "⇧⌘O", "⇧⌘O"},
		{"darwin", "⌥⌘←", "⌥⌘←"},
		{"linux", "⇧⌘O", "Ctrl+Shift+O"},
		{"js", "⇧⌘O", "Ctrl+Shift+O"},
		{"linux", "⌥⌘←", "Ctrl+Alt+Left"},
		{"linux", "⌥⇧←  ⌥⇧→", "Alt+Shift+Left  Alt+Shift+Right"},
		{"linux", "⌘⌫", "Ctrl+Backspace"},
		{"linux", "⌃⇥", "Ctrl+Tab"},
		{"linux", "F2", "F2"},
		{"js", "**⌘O** opens", "**Ctrl+O** opens"},
	} {
		if got := nativeKeysFor(tt.goos, tt.in); got != tt.want {
			t.Errorf("%s %q: %q, want %q", tt.goos, tt.in, got, tt.want)
		}
	}
}

func TestEveryShortcutHasABrowserSpelling(t *testing.T) {
	a := &app{workspace: &workspace{}}
	for _, c := range paletteCommands {
		got := nativeKeysFor("js", a.commandKeys(c))
		if strings.ContainsAny(got, "⌘⇧⌥⌃") {
			t.Errorf("%s: %q was not translated", c.title, got)
		}
	}
}

func TestFileManagerNames(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin": "Finder", "linux": "Files",
	} {
		if got := fileManagerNameFor(goos); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
	for goos, want := range map[string]string{
		"darwin": "Reveal in Finder", "linux": "Reveal in Files",
	} {
		if got := revealLabelFor(goos); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}
