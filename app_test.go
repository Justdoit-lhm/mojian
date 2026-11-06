package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// newProjectApp opens a project with prose and code files, and a tree.
func newProjectApp(t *testing.T) *app {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"Welcome.md":              welcomeNote,
		"Reading notes.md":        sampleNote,
		"Ideas/Quiet mornings.md": "# Quiet mornings\n\nCoffee, a blank page, and no notifications.\n",
		"scripts/tasks.py":        "#!/usr/bin/env python3\n\nTASKS = [\"write\", \"read\"]\n\n\ndef open_tasks():\n    return TASKS\n",
		"scripts/app.js":          "const tasks = [];\n\nfunction openTasks() {\n  return tasks;\n}\n",
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MOJIAN_VAULT", dir)
	s, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	a := &app{store: s, treeOpen: true, openDirs: map[string]bool{"scripts": true}}
	a.focusFirstNote()
	return a
}

func TestCodeFilesLoad(t *testing.T) {
	a := newProjectApp(t)
	py := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))
	if py == nil {
		t.Fatal("the Python file was not loaded")
	}
	if py.Title != "tasks.py" {
		t.Errorf("Python title: %q", py.Title)
	}
	if l := editor.LanguageFor(py.Path); l == nil || l.Name != "Python" {
		t.Errorf("Python language: %v", l)
	}
	js := a.store.byRelPath(filepath.Join("scripts", "app.js"))
	if js == nil || js.Title != "app.js" {
		t.Fatalf("JavaScript file: %+v", js)
	}
	if isMarkdownPath(py.Path) {
		t.Error("a Python file was taken for Markdown")
	}
}

func TestSplitChoosesAnotherFile(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	py := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))
	a.openNote(py)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	a.beginSplit(false)
	tt.Frame()
	if !a.splitPick {
		t.Fatal("Split Right did not open the file chooser")
	}
	if !tt.HasText("Choose a file for the new pane") {
		t.Fatalf("split chooser texts: %q", tt.Texts())
	}
	var other *note
	for _, n := range a.splitCandidates() {
		if n != py {
			other = n
			break
		}
	}
	if other == nil {
		t.Fatal("the split chooser has no other file")
	}
	a.chooseSplit(other)
	tt.Frame()
	if a.splitPick {
		t.Fatal("the split chooser stayed open")
	}
	tb := a.tab()
	panes := tb.panes()
	if len(panes) != 2 {
		t.Fatalf("panes: %d", len(panes))
	}
	if panes[0].Note == panes[1].Note {
		t.Fatal("a split opened the same file twice")
	}
	if tb.Focus != panes[1] || panes[1].Note != other {
		t.Fatal("the new pane did not receive the chosen file and focus")
	}
	for _, p := range panes {
		if p.state == nil {
			t.Fatal("a pane has no editor state")
		}
	}
	if panes[0].state == panes[1].state {
		t.Error("the panes share one caret and scroll state")
	}
	if panes[0].state.Buffer == panes[1].state.Buffer {
		t.Error("different files share one buffer")
	}
}

func TestFocusAfterTabSwitch(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	n1 := a.currentNote()
	a.newTab()
	n2 := a.store.byRelPath("Reading notes.md")
	a.openNote(n2)
	if a.currentNote() != n2 {
		t.Fatal("the second tab does not show the other file")
	}
	tt.Frame()
	a.showTab(1)
	tt.Frame()
	before := n1.text()
	tt.Type("X")
	tt.Frame()
	if n1.text() == before {
		t.Fatal("typing after a tab switch did not reach the editor")
	}
}

func TestCloseTabSelectsNextAndFocuses(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	want := a.currentNote()
	a.newTab()
	a.openNote(a.store.byRelPath("Reading notes.md"))
	a.closeTab(a.tab())
	if len(a.tabs) != 1 || a.active != 0 {
		t.Fatalf("tabs=%d active=%d", len(a.tabs), a.active)
	}
	if a.currentNote() != want {
		t.Fatal("closing the active tab did not return to the other tab")
	}
	tt.Frame()
	before := want.text()
	tt.Type("Y")
	tt.Frame()
	if want.text() == before {
		t.Fatal("typing after closing a tab did not reach the editor")
	}
}

func TestCloseActiveTabPicksNext(t *testing.T) {
	a := newProjectApp(t)
	a.newTab()
	a.newTab()
	if len(a.tabs) != 3 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	a.active = 1
	third := a.tabs[2]
	a.closeTab(a.tabs[1])
	if len(a.tabs) != 2 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	if a.tabs[1] != third {
		t.Fatal("closing a middle tab did not select the next one")
	}
	if a.active != 1 {
		t.Fatalf("active: %d", a.active)
	}
}

func TestEditsWaitForExplicitSave(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	n := a.currentNote()
	if n.buf == nil {
		t.Fatal("the editor buffer was not made")
	}
	body := "# Saved only on request\n\nBody.\n"
	n.buf.Set(body)
	if !n.dirty() {
		t.Fatal("the edited file is not dirty")
	}
	for range 2 {
		tt.Frame()
	}
	data, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == body {
		t.Fatal("editing wrote the file without an explicit save")
	}
	if !n.dirty() {
		t.Fatal("the edited file is no longer dirty")
	}
	a.saveNow()
	data, err = os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != body {
		t.Fatalf("file after save: %q", data)
	}
	if n.dirty() {
		t.Error("file still dirty after save")
	}
}

func TestFolderCreation(t *testing.T) {
	a := newProjectApp(t)
	a.treeSel = "Ideas"
	a.startNewFolder()
	if a.treeDraft == nil || a.treeDraft.kind != draftFolder || a.treeDraft.parent != "Ideas" {
		t.Fatalf("draft: %+v", a.treeDraft)
	}
	a.treeDraft.name = "Field Notes"
	a.submitTreeDraft()
	if a.treeDraft != nil {
		t.Fatal("the draft is still open")
	}
	if !a.store.dirExists(filepath.Join("Ideas", "Field Notes")) {
		t.Fatal("the folder was not created")
	}
	if a.treeSel != filepath.Join("Ideas", "Field Notes") {
		t.Errorf("selection: %q", a.treeSel)
	}
	n, err := a.store.newFileIn(a.treeSel, "note.md", "# Note\n")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(n.Rel) != a.treeSel {
		t.Errorf("new file at %q, want %q", n.Rel, a.treeSel)
	}
}

func TestFolderThroughUI(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.treeSel = ""
	tt := ui.NewTester(a.view, 1100, 700)
	if err := tt.Click("New Folder"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.treeDraft == nil || a.treeDraft.kind != draftFolder {
		t.Fatalf("clicking New Folder did not start a draft: %+v", a.treeDraft)
	}
	tt.Type("Field Notes")
	tt.Frame()
	if a.treeDraft == nil || a.treeDraft.name != "Field Notes" {
		t.Fatalf("draft after typing: %+v", a.treeDraft)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.treeDraft != nil {
		t.Fatalf("the draft is still open: %+v", a.treeDraft)
	}
	if !a.store.dirExists("Field Notes") {
		t.Fatalf("the folder was not created (treeSel=%q err=%q)", a.treeSel, a.err)
	}
}

func TestNewFileThroughUI(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.treeSel = "scripts"
	tt := ui.NewTester(a.view, 1100, 700)
	if err := tt.Click("New File"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.treeDraft == nil || a.treeDraft.kind != draftFile || a.treeDraft.parent != "scripts" {
		t.Fatalf("draft: %+v", a.treeDraft)
	}
	tt.Type("worker.py")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.treeDraft != nil {
		t.Fatalf("the draft is still open: %+v", a.treeDraft)
	}
	n := a.store.byRelPath(filepath.Join("scripts", "worker.py"))
	if n == nil {
		t.Fatalf("the file was not created (treeSel=%q err=%q)", a.treeSel, a.err)
	}
	if a.currentNote() != n {
		t.Fatal("the new file did not open")
	}
	if !strings.HasSuffix(n.Title, "worker.py") {
		t.Errorf("title: %q", n.Title)
	}
}

func TestTabStripRevealsActive(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	for i := range 11 {
		a.newTab()
		a.tab().Name = fmt.Sprintf("Workspace %d", i+2)
	}
	tt := ui.NewTester(a.view, 760, 560)
	tt.Frame()
	a.showTab(12)
	a.tab().Name = "Workspace 12"
	a.showTab(12)
	tt.Frame()
	r, ok := tt.Find("Workspace 12")
	if !ok {
		t.Fatal("the active tab has no label")
	}
	if r.W < 40 || r.X < 0 || r.X+r.W > 760 {
		t.Fatalf("the active tab is not revealed: x=%.1f w=%.1f", r.X, r.W)
	}
}

func TestTreeTogglesWithCommand(t *testing.T) {
	a := newProjectApp(t)
	a.treeOpen = false
	a.run(cmdTree)
	if !a.treeOpen {
		t.Fatal("the file tree did not open")
	}
	a.run(cmdTree)
	if a.treeOpen {
		t.Fatal("the file tree did not close")
	}
}

func TestRefreshProject(t *testing.T) {
	a := newProjectApp(t)
	path := filepath.Join(a.store.dir, "made-in-terminal.py")
	if err := os.WriteFile(path, []byte("print('hi')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if a.store.byRelPath("made-in-terminal.py") != nil {
		t.Fatal("the file appeared before the refresh")
	}
	a.refreshProject()
	n := a.store.byRelPath("made-in-terminal.py")
	if n == nil {
		t.Fatal("the refresh did not pick up a file made in a terminal")
	}
	if n.Title != "made-in-terminal.py" {
		t.Errorf("title: %q", n.Title)
	}
}

func TestMenuBuilds(t *testing.T) {
	a := &app{}
	if m := a.menu(); m == nil {
		t.Fatal("no menu")
	}
}
