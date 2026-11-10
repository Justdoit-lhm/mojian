package platform

import "testing"

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
		if got := NativeKeysFor(tt.goos, tt.in); got != tt.want {
			t.Errorf("%s %q: %q, want %q", tt.goos, tt.in, got, tt.want)
		}
	}
}

func TestFileManagerNames(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin": "Finder", "linux": "Files", "windows": "Explorer",
	} {
		if got := fileManagerNameFor(goos); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
	for goos, want := range map[string]string{
		"darwin": "Reveal in Finder", "linux": "Reveal in Files", "windows": "Reveal in Explorer",
	} {
		if got := revealLabelFor(goos); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}
