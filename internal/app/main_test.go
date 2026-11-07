package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestSplitsPastFourPanes(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	notes := a.store.sorted()
	for i, n := range notes {
		if i == 0 {
			continue
		}
		a.splitWith(n, i%2 == 1)
	}
	if got, want := len(t0.panes()), len(notes); got != want {
		t.Fatalf("panes: %d, want %d", got, want)
	}
	if len(t0.panes()) <= 4 {
		t.Skip("the fixture has too few files")
	}
	if a.err != "" {
		t.Fatalf("splitting past four failed: %q", a.err)
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
	if got := b.tabs[0].Root.A.Pane.W; got != paneColumnWidth {
		t.Errorf("the layout lost the canvas width: %v", got)
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
	tt.Frame()
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

func TestNewTabIsEmpty(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	before := len(a.tabs)
	if err := tt.Click("New Tab"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := len(a.tabs); got != before+1 {
		t.Fatalf("tabs after New Tab: %d, want %d", got, before+1)
	}
	tb := a.tab()
	if tb.Root != nil || len(tb.panes()) != 0 || a.currentNote() != nil {
		t.Fatalf("the new tab copied a scene: %+v", tb)
	}
	if !tt.HasText("No file open") {
		t.Fatalf("empty tab texts: %q", tt.Texts())
	}
}

func TestSplitUsesCanvasSizes(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	root := a.tab().Root
	if root == nil || root.Vertical {
		t.Fatal("a right split should be side by side")
	}
	if first.W != paneColumnWidth || a.tab().Focus.W != paneColumnWidth {
		t.Fatalf("right split widths: %v and %v, want half a viewport",
			first.W, a.tab().Focus.W)
	}
	if first.H != 1 || a.tab().Focus.H != 1 {
		t.Fatalf("right split heights: %v and %v, want a full viewport",
			first.H, a.tab().Focus.H)
	}
	focus := a.tab().Focus
	a.splitWith(a.store.byRelPath("scripts/tasks.py"), true)
	if a.tab().Focus == focus || a.tab().Focus.H != 1 || a.tab().Focus.W != 1 {
		t.Fatalf("down split size: %v x %v, want a full screen",
			a.tab().Focus.W, a.tab().Focus.H)
	}
}

func TestDragPaneSwapsPlaces(t *testing.T) {
	a := newProjectApp(t)
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	first, second := a.tab().panes()[0], a.tab().panes()[1]
	handle, ok := tt.Find("Move Pane")
	if !ok {
		t.Fatal("the pane has no move handle")
	}
	tt.Press(handle.X+handle.W/2, handle.Y+handle.H/2)
	tt.Move(handle.X+40, handle.Y+10)
	tt.Move(880, 380)
	tt.Release(880, 380)
	tt.Frame()
	if got := a.tab().panes(); got[0] != second || got[1] != first {
		t.Fatalf("the drag did not swap the panes: %v, %v", got[0].Note.Title, got[1].Note.Title)
	}
	if a.tab().Focus != first {
		t.Fatal("the moved pane is not focused")
	}
}
