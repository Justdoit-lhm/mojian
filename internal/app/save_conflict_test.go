package app

import (
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSaveCommandShowsExternalConflict(t *testing.T) {
	a := newProjectApp(t)
	n := a.currentNote()
	n.Buffer().Set("local edit\n")
	if err := os.WriteFile(n.Path, []byte("external edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !a.commandRegistry().Run("save") {
		t.Fatal("the save command was not registered")
	}
	if !strings.Contains(a.err, "file changed outside mojian") || !strings.Contains(a.err, n.Rel) {
		t.Fatalf("save did not identify the conflict: %q", a.err)
	}
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if !tt.HasText(a.err) {
		t.Fatalf("conflict banner was not rendered: %q", tt.Texts())
	}
	data, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "external edit\n" || !n.Dirty() || n.Text() != "local edit\n" {
		t.Fatal("the save command lost external or unsaved local edits")
	}
}
