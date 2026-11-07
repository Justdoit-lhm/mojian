package app

import "testing"

func TestNativeKeys(t *testing.T) {
	for _, tt := range []struct {
		goos, in, want string
	}{
		{"darwin", "⇧⌘O", "⇧⌘O"},
		{"darwin", "⌥⌘←", "⌥⌘←"},
		{"windows", "⇧⌘O", "Ctrl+Shift+O"},
		{"linux", "⇧⌘O", "Ctrl+Shift+O"},
		{"windows", "⌥⌘←", "Ctrl+Alt+Left"},
		{"linux", "⌥⇧←  ⌥⇧→", "Alt+Shift+Left  Alt+Shift+Right"},
		{"windows", "⌘⌫", "Ctrl+Backspace"},
		{"linux", "⌃⇥", "Ctrl+Tab"},
		{"windows", "F2", "F2"},
		{"windows", "**⌘O** opens", "**Ctrl+O** opens"},
	} {
		if got := nativeKeysFor(tt.goos, tt.in); got != tt.want {
			t.Errorf("%s %q: %q, want %q", tt.goos, tt.in, got, tt.want)
		}
	}
}

func TestFileManagerNames(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin": "Finder", "windows": "Explorer", "linux": "Files",
	} {
		if got := fileManagerNameFor(goos); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
	for goos, want := range map[string]string{
		"darwin": "Reveal in Finder", "windows": "Show in Explorer", "linux": "Reveal in Files",
	} {
		if got := revealLabelFor(goos); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}
