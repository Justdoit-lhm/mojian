package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestTerminalWithoutFileOpensAtProjectRoot(t *testing.T) {
	if !terminalsSupported() {
		t.Skip("terminals need macOS or Linux")
	}
	a := newProjectApp(t)
	a.shellStarter = func() (*termTab, error) { return &termTab{}, nil }
	a.tabs = nil
	a.active = -1
	a.treeSel = "scripts/tasks.py"
	if got := a.terminalDir(); got != a.store.Dir {
		t.Fatalf("terminal directory %q, want project root %q", got, a.store.Dir)
	}
	a.toggleTerminal()
	if len(a.tabs) != 1 {
		t.Fatalf("tabs: %d, want a terminal tab", len(a.tabs))
	}
	t0 := a.tab()
	if t0 == nil || t0.Root == nil || t0.Root.Pane == nil || t0.Root.Pane.Term == nil {
		t.Fatalf("no terminal pane: %+v", t0)
	}
	if t0.Focus != t0.Root.Pane {
		t.Fatal("the terminal pane is not focused")
	}
	if a.currentNote() != nil {
		t.Fatal("opening a terminal should not open a file")
	}
}

func TestSplitPutsPaneOnEdge(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	second := a.tab().Focus
	a.splitWithSide(a.store.ByRelPath("scripts/tasks.py"), splitDown)
	third := a.tab().Focus
	root := a.tab().Root
	if root == nil || root.Vertical || root.A.Pane != first || !root.B.Vertical {
		t.Fatalf("root after right and down splits: %+v", root)
	}
	if root.B.A.Pane != second || root.B.B.Pane != third {
		t.Fatalf("down split did not stack below the right pane: %+v", root.B)
	}
	if a.tab().Focus != third {
		t.Fatal("the last split did not focus its pane")
	}
}

func TestPaneFocusShortcutsMoveInAllDirections(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	second := a.tab().Focus
	a.splitWithSide(a.store.ByRelPath("scripts/tasks.py"), splitDown)
	third := a.tab().Focus
	// A harness window has no menu bar, so the view binds the commands
	// itself, as the browser does.
	a.harness = true
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Key(ui.Cmd|ui.Alt, ui.KeyUp)
	tt.Frame()
	if a.tab().Focus != second {
		t.Fatal("Cmd+Option+Up did not focus the pane above")
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyDown)
	tt.Frame()
	if a.tab().Focus != third {
		t.Fatal("Cmd+Option+Down did not focus the pane below")
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyLeft)
	tt.Frame()
	if a.tab().Focus != first {
		t.Fatal("Cmd+Option+Left did not focus the pane on the left")
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyRight)
	tt.Frame()
	if a.tab().Focus != second {
		t.Fatal("Cmd+Option+Right did not focus the pane on the right")
	}
}

func TestPaneSplitShortcutsSplitRightAndDown(t *testing.T) {
	for _, dir := range []struct {
		mods ui.Modifiers
		name string
	}{
		{ui.Cmd, "Cmd+D"},
		{ui.Cmd | ui.Shift, "Shift+Cmd+D"},
	} {
		a := newProjectApp(t)
		a.harness = true
		tt := ui.NewTester(a.view, 1100, 700)
		tt.Frame()
		tt.Key(dir.mods, ui.KeyD)
		tt.Frame()
		if got := len(a.tab().panes()); got != 2 {
			t.Fatalf("%s left %d panes, want 2", dir.name, got)
		}
	}
}

func TestPaneZoomShortcutTogglesTheZoom(t *testing.T) {
	a := newProjectApp(t)
	a.harness = true
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	focus := a.tab().Focus
	tt.Key(ui.Cmd|ui.Shift, ui.KeyEnter)
	tt.Frame()
	if a.tab().Zoom != focus {
		t.Fatal("Shift+Cmd+Enter did not zoom the focused pane")
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyEnter)
	tt.Frame()
	if a.tab().Zoom != nil {
		t.Fatal("Shift+Cmd+Enter did not restore the panes")
	}
}

func TestMovePaneDirectionInsertsOnEdge(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	second := a.tab().Focus
	a.splitWithSide(a.store.ByRelPath("scripts/tasks.py"), splitDown)
	third := a.tab().Focus

	a.movePaneDirection(paneUp)
	root := a.tab().Root
	if root == nil || root.Vertical || !root.B.Vertical || root.B.A.Pane != third || root.B.B.Pane != second {
		t.Fatalf("moving up did not put the pane above its neighbor: %+v", root)
	}
	if a.tab().Focus != third {
		t.Fatal("moving a pane lost its focus")
	}

	a.movePaneDirection(paneLeft)
	panes := a.tab().panes()
	if len(panes) != 3 || panes[0] != third || panes[1] != first || panes[2] != second {
		t.Fatalf("moving left did not insert the pane before its neighbor: %v", panes)
	}
}

func TestDropOnEdgeInsertsPane(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	second := a.tab().Focus
	a.movePaneTo(a.tab(), first, second, dropRight)
	panes := a.tab().panes()
	if len(panes) != 2 || panes[0] != second || panes[1] != first {
		t.Fatalf("dropping on the right edge did not insert the pane there: %v", panes)
	}
	if a.tab().Focus != first {
		t.Fatal("the dragged pane is not focused after the drop")
	}
}

func TestPaneHeaderStartsADrag(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	handle, ok := tt.Find("Move Pane " + paneName(first))
	if !ok {
		t.Fatal("no move handle for the pane")
	}
	tt.Press(handle.X+handle.W/2, handle.Y+handle.H/2)
	tt.Frame()
	tt.Move(handle.X+20, handle.Y+20)
	tt.Frame()
	if !first.dragging {
		t.Fatal("dragging the pane header did not start a drag")
	}
	tt.Release(handle.X+20, handle.Y+20)
	tt.Frame()
}

func TestOpenedFileFillsAnEmptyTab(t *testing.T) {
	a := newProjectApp(t)
	if a.tab().Root == nil || a.tab().Root.Pane == nil {
		t.Fatalf("the new tab has no pane: %+v", a.tab())
	}
	if got := a.terminalDir(); got != a.store.Dir {
		t.Fatalf("terminal directory: %q", got)
	}
}

func TestEmptyPaneSearchOpensFile(t *testing.T) {
	a := newProjectApp(t)
	a.splitFocusedSide(splitRight)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	p := a.tab().Focus
	if p == nil || p.Note != nil {
		t.Fatal("the split did not focus an empty pane")
	}
	tt.Type("Reading")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if p.Note == nil || p.Note.Title != "Reading notes" {
		t.Fatalf("the empty pane search opened %+v", p.Note)
	}
	if a.tab().Focus != p {
		t.Fatal("the opened file lost the pane focus")
	}
}

func TestEmptyPaneSearchFiltersAssets(t *testing.T) {
	a := newProjectApp(t)
	choices := a.paneChoices("tasks")
	if len(choices) != 1 || choices[0].note == nil || choices[0].rel != filepath.Join("scripts", "tasks.py") {
		t.Fatalf("search choices: %+v", choices)
	}
	if a.paneChoices("no such file") != nil {
		t.Fatal("an impossible query returned choices")
	}
}
