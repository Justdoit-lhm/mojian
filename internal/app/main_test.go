package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func TestSplitAndClose(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	first := t0.Focus.Note
	second := a.store.byRelPath("Reading notes.md")
	third := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))

	a.splitWith(second)
	if got := len(t0.panes()); got != 2 {
		t.Fatalf("panes after split: %d", got)
	}
	if t0.Focus.Note != second {
		t.Error("split did not focus the chosen file")
	}
	if t0.panes()[1].Note != second || t0.panes()[0].Note != first {
		t.Error("the split panes show the wrong files")
	}
	a.splitWith(third)
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
		a.splitWith(n)
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
	if !t1.splitSide(t1.Focus, second, splitRight) {
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
	if got, want := b.tabs[0].panes()[0].Note.Rel, notes[0].Rel; got != want {
		t.Errorf("the layout lost the first pane: %q, want %q", got, want)
	}
	if got, want := b.tabs[0].panes()[1].Note.Rel, notes[1].Rel; got != want {
		t.Errorf("the layout lost the second pane: %q, want %q", got, want)
	}
}

func TestRestoreFillsALonePane(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOJIAN_VAULT", dir)
	s, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	n := s.sorted()[0]
	a := &app{store: s, layoutPath: filepath.Join(dir, "layout.json")}
	p := newPane(n)
	a.tabs = []*tab{{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}}
	a.saveLayout()

	b := &app{store: s, layoutPath: a.layoutPath}
	if !b.restoreLayout() {
		t.Fatal("restoreLayout returned false")
	}
	if got := len(b.tab().panes()); got != 1 {
		t.Fatalf("restored panes: %d, want 1", got)
	}
	if b.tab().Focus != b.tab().panes()[0] {
		t.Fatal("the restored tab lost its focus")
	}
}

func TestRestoreReadsTheOldSplitTree(t *testing.T) {
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
	old := layoutFile{Tabs: []layoutTab{{
		Root: &layoutNode{
			Vertical: true,
			A:        &layoutNode{Pane: &layoutPane{ID: "a", Note: "a.md"}},
			B:        &layoutNode{Pane: &layoutPane{ID: "b", Note: "b.md"}},
		},
		Focus: "b",
	}}, Active: 0}
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "layout.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	app := &app{store: s, layoutPath: path}
	if !app.restoreLayout() {
		t.Fatal("restoreLayout returned false")
	}
	if got := len(app.tabs[0].panes()); got != 2 {
		t.Fatalf("panes: %d", got)
	}
	if got := app.tabs[0].panes()[0].Note.Rel; got != "a.md" {
		t.Errorf("first pane: %q", got)
	}
	if got := app.tabs[0].panes()[1].Note.Rel; got != "b.md" {
		t.Errorf("second pane: %q", got)
	}
	if app.tabs[0].Focus == nil || app.tabs[0].Focus.Note.Rel != "b.md" {
		t.Errorf("focus: %+v", app.tabs[0].Focus)
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
	if got := len(a.tab().panes()); got != 2 {
		t.Fatalf("panes after Split Right: %d", got)
	}
	if !tt.HasText("Open a file in this pane") {
		t.Fatalf("Split Right did not open the file search: %q", tt.Texts())
	}
	a.openNote(a.store.byRelPath("Reading notes.md"))
	tt.Frame()
	if got := a.tab().Focus.Note; got == nil || got.Title != "Reading notes" {
		t.Fatal("the file did not open in the new pane")
	}
	a.openPalette()
	tt.Frame()
	tt.Type("Split Left")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := len(a.tab().panes()); got != 3 {
		t.Fatalf("panes after Split Left: %d", got)
	}
	a.openNote(a.store.byRelPath(filepath.Join("scripts", "tasks.py")))
	tt.Frame()
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
	if name, _ := tb.label(); name != "Welcome" {
		t.Fatalf("empty tab label: %q, want Welcome", name)
	}
	if len(tb.panes()) != 0 || a.currentNote() != nil {
		t.Fatalf("the new tab copied a scene: %+v", tb)
	}
	if !tt.HasText("Welcome to Mojian") {
		t.Fatalf("welcome tab texts: %q", tt.Texts())
	}
}

func TestSplitAddsAColumn(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.byRelPath("Reading notes.md"))
	if got := a.tab().panes(); len(got) != 2 || got[0] != first || got[1] != a.tab().Focus {
		t.Fatalf("panes after a right split: %v", got)
	}
	third := a.store.byRelPath("scripts/tasks.py")
	a.splitWith(third)
	if got := len(a.tab().panes()); got != 3 {
		t.Fatalf("panes after a third file: %d", got)
	}
	if a.tab().panes()[2].Note != third {
		t.Fatal("a split appends the new pane to the right")
	}
}

func TestSplitLeftPutsTheNewPaneFirst(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWithSide(a.store.byRelPath("Reading notes.md"), splitLeft)
	got := a.tab().panes()
	if len(got) != 2 || got[0] == first || got[1] != first {
		t.Fatalf("a left split did not put the new pane first: %v", got)
	}
	if a.tab().Focus != got[0] {
		t.Fatal("a left split did not focus the new pane")
	}
}

func TestArrowKeysMoveInFourDirections(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.byRelPath("Reading notes.md"))
	second := a.tab().Focus
	a.splitWithSide(a.store.byRelPath("scripts/tasks.py"), splitDown)
	third := a.tab().Focus

	a.focusPaneDirection(paneUp)
	if a.tab().Focus != second {
		t.Fatal("up did not focus the pane above")
	}
	a.focusPaneDirection(paneDown)
	if a.tab().Focus != third {
		t.Fatal("down did not focus the pane below")
	}
	a.focusPaneDirection(paneLeft)
	if a.tab().Focus != first {
		t.Fatal("left did not focus the column on the left")
	}
	a.focusPaneDirection(paneRight)
	if a.tab().Focus != second {
		t.Fatal("right did not focus the column on the right")
	}
	if a.focusPaneDirection(paneLeft); a.tab().Focus != first {
		t.Fatal("left at the first column did not move")
	}
}
