package app

import (
	"strings"
	"testing"

	"mojian/internal/platform"
)

func TestEveryShortcutHasABrowserSpelling(t *testing.T) {
	a := &app{workspace: &workspace{}}
	for _, c := range paletteCommands {
		got := platform.NativeKeysFor("js", a.commandKeys(c))
		if strings.ContainsAny(got, "⌘⇧⌥⌃") {
			t.Errorf("%s: %q was not translated", c.Title, got)
		}
	}
}
