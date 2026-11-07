package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The welcome page's Open File row opens a picker over the project.
// Focusing the Explorer's search field looked like nothing happened when
// the tree was already open.
func TestWelcomeOpenFileShowsAPicker(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.treeOpen = false
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("New Tab"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Open File…"); err != nil {
		t.Fatal(err)
	}
	if !a.quickOpen {
		t.Fatal("Open File did not show the picker")
	}
	if !tt.Focused("Open a file") {
		t.Fatal("the picker's field did not take focus")
	}
	tt.Type("tasks")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.quickOpen {
		t.Fatal("the picker stayed open after Enter")
	}
	n := a.currentNote()
	if n == nil || n.Rel != filepath.Join("scripts", "tasks.py") {
		t.Fatalf("opened note: %+v", n)
	}
	if name, _ := a.tab().label(); name != "tasks.py" {
		t.Fatalf("active tab: %q", name)
	}
	// The other tab still shows Welcome.md, so check the welcome page's
	// own subtitle rather than its heading.
	if tt.HasText("A quiet editor for the files in your project.") {
		t.Fatal("the welcome page stayed after opening a file")
	}
}

func TestQuickOpenEscapeCloses(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	before := a.currentNote()
	a.openQuickOpen()
	tt.Frame()
	if !a.quickOpen {
		t.Fatal("openQuickOpen did not show the picker")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.quickOpen {
		t.Fatal("Escape did not close the picker")
	}
	if a.currentNote() != before {
		t.Fatal("Escape changed the open file")
	}
}
