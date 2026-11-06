package main

import (
	"os"
	"path/filepath"
	"strings"
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
	a := &app{store: s, treeOpen: true}
	a.focusFirstNote()
	if a.tab() == nil {
		t.Fatal("no tab after opening the vault")
	}
	return a
}

func TestSplitAndClose(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	first := t0.Focus.Note
	second := a.store.byRelPath("Reading notes.md")
	third := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))

	a.splitWith(second, false)
	if got := len(t0.panes()); got != 2 {
		t.Fatalf("panes after split: %d", got)
	}
	if t0.Focus.Note != second {
		t.Error("split did not focus the chosen file")
	}
	if t0.panes()[1].Note != second || t0.panes()[0].Note != first {
		t.Error("the split panes show the wrong files")
	}
	if t0.Root.Vertical {
		t.Error("split right made a stacked split")
	}

	a.splitWith(third, true)
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

func TestSplitStopsAtFourPanes(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	for i, n := range a.store.sorted() {
		if i == 0 {
			continue
		}
		a.splitWith(n, false)
	}
	if got := len(t0.panes()); got != min(len(a.store.sorted()), maxPanes) {
		t.Fatalf("panes: %d", got)
	}
	if len(t0.panes()) < maxPanes {
		t.Skip("the fixture has too few files")
	}
	a.beginSplit(false)
	if !strings.Contains(a.err, "at most 4 panes") {
		t.Fatalf("the fifth pane was allowed: %q", a.err)
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
	second := newPane(notes[1])
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
	buf := editor.NewBuffer(n.Saved)
	buf.Set(body)
	n.buf = buf
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
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	if !tt.HasText("Welcome to Mojian") {
		t.Fatalf("texts: %q", tt.Texts())
	}
	if err := tt.Click("Split Right"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.splitPick || !tt.HasText("Choose a file for the new pane") {
		t.Fatalf("Split Right did not ask for a file: %q", tt.Texts())
	}
	a.chooseSplit(a.store.byRelPath("Reading notes.md"))
	tt.Frame()
	if got := len(a.tab().panes()); got != 2 {
		t.Fatalf("panes after choosing a file: %d", got)
	}
	a.openPalette()
	tt.Frame()
	tt.Type("Split Down")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.splitPick {
		t.Fatal("the Split Down palette command did not ask for a file")
	}
	a.chooseSplit(a.store.byRelPath(filepath.Join("scripts", "tasks.py")))
	tt.Frame()
	if got := len(a.tab().panes()); got != 3 {
		t.Fatalf("panes after the second choice: %d", got)
	}
	if err := tt.Click("New File"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.treeDraft == nil {
		t.Fatal("clicking New File did not start a name field")
	}
	tt.Type("Untitled.py")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.currentNote() == nil || !strings.HasSuffix(a.currentNote().Title, "Untitled.py") {
		t.Errorf("current note: %+v", a.currentNote())
	}
}
