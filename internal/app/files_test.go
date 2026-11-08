package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	integration "mojian/test"

	"github.com/egoist/mygo/ui"
)

func TestRenameFileUpdatesOpenNote(t *testing.T) {
	a := newProjectApp(t)
	n := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))
	a.openNote(n)
	rel, changed, err := a.store.rename(filepath.Join("scripts", "tasks.py"), "worker.py")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("scripts", "worker.py")
	if rel != want || n.Rel != want {
		t.Fatalf("rel=%q note=%q", rel, n.Rel)
	}
	if filepath.Base(n.Path) != "worker.py" {
		t.Errorf("path: %q", n.Path)
	}
	if len(changed) != 1 || changed[0] != n {
		t.Errorf("changed: %+v", changed)
	}
	if a.currentNote() != n {
		t.Fatal("the open editor lost its note")
	}
	if a.store.byRelPath(want) != n {
		t.Fatal("the store still has the old path")
	}
}

func TestRenameFolderRemapsNotesAndOpenFolders(t *testing.T) {
	a := newProjectApp(t)
	a.openDirs = map[string]bool{"scripts": true}
	py := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))
	js := a.store.byRelPath(filepath.Join("scripts", "app.js"))
	a.openNote(py)
	rel, _, err := a.store.rename("scripts", "src")
	if err != nil {
		t.Fatal(err)
	}
	if rel != "src" {
		t.Fatalf("rel: %q", rel)
	}
	if py.Rel != filepath.Join("src", "tasks.py") || js.Rel != filepath.Join("src", "app.js") {
		t.Fatalf("notes: %q %q", py.Rel, js.Rel)
	}
	if _, ok := a.store.byRel[py.Rel]; !ok {
		t.Fatal("the note map was not rebuilt")
	}
}

func TestRenameThroughTreeDraft(t *testing.T) {
	a := newProjectApp(t)
	a.treeSel = filepath.Join("scripts", "tasks.py")
	a.openNote(a.store.byRelPath(a.treeSel))
	a.startRenameNode(a.treeSel)
	if a.treeDraft == nil || a.treeDraft.kind != draftRename {
		t.Fatalf("draft: %+v", a.treeDraft)
	}
	a.treeDraft.name = "worker.py"
	a.submitTreeDraft()
	want := filepath.Join("scripts", "worker.py")
	if a.treeSel != want {
		t.Fatalf("selection: %q", a.treeSel)
	}
	if a.currentNote() == nil || a.currentNote().Rel != want {
		t.Fatalf("current note: %+v", a.currentNote())
	}
	if a.treeDraft != nil {
		t.Fatal("draft did not close")
	}
}

func TestCopyPasteAndCut(t *testing.T) {
	a := newProjectApp(t)
	py := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))
	a.openNote(py)
	a.copySelectionTo(py.Rel, false)
	a.pasteInto("Ideas")
	want := filepath.Join("Ideas", "tasks copy.py")
	if a.store.byRelPath(want) == nil {
		t.Fatalf("copy not found (err=%q)", a.err)
	}
	if py.Rel != filepath.Join("scripts", "tasks.py") {
		t.Fatal("copying moved the source note")
	}

	js := a.store.byRelPath(filepath.Join("scripts", "app.js"))
	a.openNote(js)
	a.copySelectionTo(js.Rel, true)
	a.pasteInto("Ideas")
	moved := filepath.Join("Ideas", "app.js")
	if js.Rel != moved {
		t.Fatalf("cut note rel: %q", js.Rel)
	}
	if _, err := os.Stat(filepath.Join(a.store.dir, filepath.Join("scripts", "app.js"))); !os.IsNotExist(err) {
		t.Fatal("the cut source still exists")
	}
	if a.store.byRelPath(moved) != js {
		t.Fatal("the store lost the moved note")
	}
}

func TestTrashClosesOpenTabs(t *testing.T) {
	old := trashPath
	trashPath = func(p string) error { return os.RemoveAll(p) }
	t.Cleanup(func() { trashPath = old })

	a := newProjectApp(t)
	py := a.store.byRelPath(filepath.Join("scripts", "tasks.py"))
	a.openNote(py)
	a.trash("scripts")
	if a.store.byRelPath(py.Rel) != nil {
		t.Fatal("the trashed note is still in the store")
	}
	if a.currentNote() != nil {
		t.Fatalf("a tab still shows the trashed note: %+v", a.currentNote())
	}
	if a.treeSel != "" {
		t.Errorf("selection after trash: %q", a.treeSel)
	}
}

func TestRefreshForgetsDeletedFile(t *testing.T) {
	a := newProjectApp(t)
	path := filepath.Join(a.store.dir, "gone.py")
	if err := os.WriteFile(path, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.refreshProject()
	if a.store.byRelPath("gone.py") == nil {
		t.Fatal("the new file was not loaded")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	a.refreshProject()
	if a.store.byRelPath("gone.py") != nil {
		t.Fatal("the deleted file stayed in the store")
	}
}

func TestProjectPickerIsInApp(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	a.openProjectPicker()
	for range 3 {
		tt.Frame()
	}
	if !a.projectPicker {
		t.Fatal("the picker did not open")
	}
	if !tt.HasText("Open Folder") || !tt.HasText("PLACES") {
		t.Fatalf("picker texts: %q", tt.Texts())
	}
	if a.pickerPath == "" || !filepath.IsAbs(a.pickerPath) {
		t.Fatalf("picker path: %q", a.pickerPath)
	}
}

func TestTerminalPaneAndSharedLimit(t *testing.T) {
	if !terminalsSupported() {
		t.Skip("terminals need macOS or Linux")
	}
	registerFonts()
	a := newProjectApp(t)
	a.shellStarter = func() (*termTab, error) { return &termTab{}, nil }
	a.addTerminalPane(&termTab{})
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	if got := len(a.tab().panes()); got != 2 {
		t.Fatalf("panes: %d", got)
	}
	if !tt.HasText("Terminal") {
		t.Fatalf("terminal title missing: %q", tt.Texts())
	}
	if tt.HasText("zsh") || tt.HasText("1/4") {
		t.Fatalf("the terminal still has a shell tab: %q", tt.Texts())
	}
	if err := tt.Click("Split Right Terminal"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := len(a.tab().panes()); got != 3 {
		t.Fatalf("panes after terminal split: %d", got)
	}
	if a.tab().Focus.Term == nil {
		t.Fatal("the new terminal pane is not focused")
	}
	a.splitTerminalSide(a.tab().Focus, splitRight)
	if got := len(a.tab().panes()); got != 4 {
		t.Fatalf("panes after fourth: %d", got)
	}
	a.splitTerminalSide(a.tab().Focus, splitRight)
	if got := len(a.tab().panes()); got != 5 {
		t.Fatalf("panes after fifth: %d", got)
	}
	if a.err != "" {
		t.Fatalf("splitting past four failed: %q", a.err)
	}
}

func TestPickerOpensChosenProject(t *testing.T) {
	startLocalServerForTest(t)
	root := t.TempDir()
	alpha := filepath.Join(root, "Alpha")
	beta := filepath.Join(root, "Beta")
	integration.WriteVault(t, alpha, map[string]string{"a.md": "# A\n"})
	integration.WriteVault(t, beta, map[string]string{"b.py": "print(1)\n"})
	a := &app{workspace: &workspace{treeOpen: true}, dataDir: t.TempDir()}
	s, err := openStoreAt(alpha)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.dir)
	a.rememberProject(s.dir)
	a.focusFirstNote()
	a.openProjectPicker()
	a.setPickerPath(root)
	for i, e := range a.pickerEntries {
		if e.Path == beta {
			a.pickerSel = i
		}
	}
	if a.pickerSel < 0 {
		t.Fatalf("Beta is not in the picker: %+v", a.pickerEntries)
	}
	a.pickerOpen()
	if a.projectPicker {
		t.Fatal("the picker stayed open")
	}
	if a.store.dir != beta {
		t.Fatalf("project: %q", a.store.dir)
	}
}

func TestOpenFileInNewTab(t *testing.T) {
	a := newProjectApp(t)
	n := a.store.byRelPath("README.md")
	if n == nil {
		n = a.store.sorted()[0]
	}
	a.openNote(n)
	before := len(a.tabs)
	a.openNoteInTab(n)
	if len(a.tabs) != before+1 {
		t.Fatalf("tabs: %d, want %d", len(a.tabs), before+1)
	}
	if a.currentNote() != n || a.active != len(a.tabs)-1 {
		t.Fatal("the new tab does not show the file")
	}
}

func TestDuplicateFile(t *testing.T) {
	a := newProjectApp(t)
	a.duplicate("Welcome.md")
	want := "Welcome copy.md"
	if _, err := os.Stat(filepath.Join(a.store.dir, want)); err != nil {
		t.Fatalf("duplicate: %v (err=%q)", err, a.err)
	}
	if a.treeSel != want {
		t.Errorf("selection: %q", a.treeSel)
	}
}

func TestRenameCaseOnlySpelling(t *testing.T) {
	a := newProjectApp(t)
	rel, changed, err := a.store.rename(filepath.Join("scripts", "tasks.py"), "TASKS.py")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("scripts", "TASKS.py"); rel != want {
		t.Fatalf("rename: %q, want %q", rel, want)
	}
	if len(changed) != 1 || changed[0].Rel != rel {
		t.Fatalf("changed notes: %+v", changed)
	}
	if a.store.byRelPath(rel) == nil {
		t.Fatal("the renamed note is missing")
	}
}
