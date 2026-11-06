package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// newTestApp opens a vault in a temporary directory with one note open.
func newTestApp(t *testing.T) *app {
	t.Helper()
	t.Setenv("MOJIAN_VAULT", t.TempDir())
	s, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	a := &app{store: s}
	a.focusFirstNote()
	if a.tab() == nil {
		t.Fatal("no tab after opening the vault")
	}
	return a
}

func TestSplitAndClose(t *testing.T) {
	a := newTestApp(t)
	t0 := a.tab()
	note := t0.Focus.Note

	a.split(false)
	if got := len(t0.panes()); got != 2 {
		t.Fatalf("panes after split: %d", got)
	}
	if t0.Focus.Note != note {
		t.Error("split changed the note")
	}
	if !t0.Focus.Preview {
		t.Error("the new pane of an editor should show the preview")
	}
	if t0.Root.Vertical {
		t.Error("split right made a stacked split")
	}

	a.split(true)
	if got := len(t0.panes()); got != 3 {
		t.Fatalf("panes after second split: %d", got)
	}
	a.closePane(t0.Focus)
	if got := len(t0.panes()); got != 2 {
		t.Fatalf("panes after closing: %d", got)
	}
	if t0.Focus == nil {
		t.Fatal("no focus after closing a pane")
	}
	if !t0.contains(t0.Focus) {
		t.Fatal("focus is not in the tab")
	}

	for len(t0.panes()) > 0 {
		a.closePane(t0.panes()[0])
	}
	if a.tab() != nil {
		t.Fatal("closing every pane should close the tab")
	}
}

func TestLayoutRoundTrip(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("# "+f+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MOJIAN_VAULT", dir)
	s, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	notes := s.sorted()
	if len(notes) < 2 {
		t.Fatalf("notes: %d", len(notes))
	}
	a := &app{store: s, layoutPath: filepath.Join(dir, "layout.json")}
	t1 := newTab(notes[0])
	a.tabs = []*tab{t1}
	a.active = 0
	second := newPane(notes[1], true)
	if !t1.split(t1.Focus, second, false) {
		t.Fatal("split failed")
	}
	a.saveLayout()

	b := &app{store: s, layoutPath: a.layoutPath}
	if !b.restoreLayout() {
		t.Fatal("restoreLayout returned false")
	}
	if len(b.tabs) != 1 {
		t.Fatalf("tabs: %d", len(b.tabs))
	}
	if got := len(b.tabs[0].panes()); got != 2 {
		t.Fatalf("panes: %d", got)
	}
	if !b.tabs[0].Focus.Preview {
		t.Error("the layout lost the preview pane's focus")
	}
	if b.tabs[0].Root.Vertical {
		t.Error("the layout lost the side by side split")
	}
}

func TestNoteSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MOJIAN_VAULT", dir)
	s, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.newNote()
	if err != nil {
		t.Fatal(err)
	}
	body := "# Edited\n\nThe new text.\n"
	n.state = &editor.State{Buffer: editor.NewBuffer(body)}
	if !n.dirty() {
		t.Fatal("a new buffer should be dirty")
	}
	if err := s.save(n); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != body {
		t.Errorf("file: %q", data)
	}
	if n.Title != "Edited" {
		t.Errorf("title: %q", n.Title)
	}
	if n.dirty() {
		t.Error("note still dirty after save")
	}
}

func TestView(t *testing.T) {
	a := newTestApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	if !tt.HasText("Welcome to Mojian") {
		t.Fatalf("texts: %q", tt.Texts())
	}
	if err := tt.Click("Split Right"); err != nil {
		t.Fatal(err)
	}
	if got := len(a.tab().panes()); got != 2 {
		t.Fatalf("panes after clicking Split Right: %d", got)
	}
	a.openPalette()
	tt.Frame()
	tt.Type("Split Down")
	tt.Key(0, ui.KeyEnter)
	if got := len(a.tab().panes()); got != 3 {
		t.Fatalf("panes after the palette: %d", got)
	}
	if err := tt.Click("New Note"); err != nil {
		t.Fatal(err)
	}
	if a.currentNote() == nil || a.currentNote().Title != "Untitled" {
		t.Errorf("current note: %+v", a.currentNote())
	}
}
