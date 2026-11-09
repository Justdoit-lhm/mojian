package app

import (
	"path/filepath"
	"testing"
)

func TestOpenStartupFile(t *testing.T) {
	a := newProjectApp(t)
	want := a.store.ByRelPath(filepath.Join("scripts", "tasks.py"))
	if want == nil {
		t.Fatal("the fixture has no scripts/tasks.py")
	}
	a.startup = Startup{File: filepath.Join(a.store.Dir, "scripts", "tasks.py")}
	a.openStartupFile()
	if got := a.currentNote(); got != want {
		t.Fatalf("the startup file was not focused: %+v", got)
	}
	tab := a.tab()
	if tab == nil || tab.Focus == nil || !tab.contains(tab.Focus) {
		t.Fatalf("focus is not a pane of the tab: %+v", tab)
	}
	if tab.Focus.Note != want {
		t.Fatalf("focus is not the startup file: %+v", tab.Focus.Note)
	}
}

func TestOpenStartupFileOutsideProject(t *testing.T) {
	a := newProjectApp(t)
	before := a.currentNote()
	a.startup = Startup{File: filepath.Join(a.store.Dir, "..", "elsewhere.md")}
	a.openStartupFile()
	if got := a.currentNote(); got != before {
		t.Fatalf("a file outside the project replaced the note: %+v", got)
	}
}
