package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSplitLeftAndUp(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	first := t0.Focus
	second := a.store.byRelPath("Reading notes.md")

	a.splitWithSide(second, splitLeft)
	if t0.Root.Vertical {
		t.Fatal("a left split should be side by side")
	}
	if t0.Root.A.Pane.Note != second || t0.Root.B.Pane.Note != first.Note {
		t.Fatal("a left split did not put the new pane on the left")
	}
	if t0.Focus.Note != second {
		t.Fatal("a left split did not focus the new pane")
	}

	third := a.store.byRelPath("scripts/tasks.py")
	a.splitWithSide(third, splitUp)
	left := t0.Root.A
	if !left.Vertical {
		t.Fatal("an up split should stack panes")
	}
	if left.A.Pane.Note != third || left.B.Pane.Note != second {
		t.Fatal("an up split did not put the new pane above")
	}
	if t0.Focus.Note != third {
		t.Fatal("an up split did not focus the new pane")
	}
}

func TestMovePaneToEveryEdge(t *testing.T) {
	for _, tc := range []struct {
		name     string
		zone     paneDrop
		vertical bool
		toFirst  bool
	}{
		{"above", dropAbove, true, false},
		{"below", dropBelow, true, true},
		{"left", dropLeft, false, false},
		{"right", dropRight, false, true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := newProjectApp(t)
			t0 := a.tab()
			from := t0.Focus
			to := newPane(a.store.byRelPath("Reading notes.md"))
			if !t0.split(from, to, false) {
				t.Fatal("setup split failed")
			}
			if !t0.movePane(from, to, tc.zone) {
				t.Fatal("movePane returned false")
			}
			if t0.Root.Vertical != tc.vertical {
				t.Fatalf("vertical=%v, want %v", t0.Root.Vertical, tc.vertical)
			}
			wantA, wantB := from, to
			if tc.toFirst {
				wantA, wantB = to, from
			}
			if t0.Root.A.Pane != wantA || t0.Root.B.Pane != wantB {
				t.Fatalf("got %v / %v, want %v / %v", t0.Root.A.Pane, t0.Root.B.Pane, wantA, wantB)
			}
			if t0.Focus != from {
				t.Fatal("the moved pane is not focused")
			}
		})
	}
}

func TestMovePaneToCenterSwaps(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	from := t0.Focus
	to := newPane(a.store.byRelPath("Reading notes.md"))
	if !t0.split(from, to, false) {
		t.Fatal("setup split failed")
	}
	if !t0.movePane(from, to, dropCenter) {
		t.Fatal("center move returned false")
	}
	if t0.Root.A.Pane != to || t0.Root.B.Pane != from {
		t.Fatal("a center drop did not swap the panes")
	}
	if t0.Focus != from {
		t.Fatal("the dragged pane is not focused")
	}
}

func TestFocusDirection(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	aPane := t0.Focus
	bPane := newPane(a.store.byRelPath("Reading notes.md"))
	cPane := newPane(a.store.byRelPath("scripts/tasks.py"))
	if !t0.split(aPane, bPane, false) {
		t.Fatal("horizontal split failed")
	}
	if !t0.split(bPane, cPane, true) {
		t.Fatal("vertical split failed")
	}
	t0.Focus = aPane
	if !t0.focusDirection(paneRight) || t0.Focus != bPane {
		t.Fatalf("right focused %v, want %v", t0.Focus, bPane)
	}
	if !t0.focusDirection(paneDown) || t0.Focus != cPane {
		t.Fatalf("down focused %v, want %v", t0.Focus, cPane)
	}
	if !t0.focusDirection(paneUp) || t0.Focus != bPane {
		t.Fatalf("up focused %v, want %v", t0.Focus, bPane)
	}
	if !t0.focusDirection(paneLeft) || t0.Focus != aPane {
		t.Fatalf("left focused %v, want %v", t0.Focus, aPane)
	}
}

func TestDropZoneAt(t *testing.T) {
	const w, h = 200, 200
	for _, tc := range []struct {
		x, y float32
		want paneDrop
	}{
		{100, 10, dropAbove},
		{100, 190, dropBelow},
		{20, 100, dropLeft},
		{180, 100, dropRight},
		{100, 100, dropCenter},
	} {
		if got := dropZoneAt(tc.x, tc.y, w, h); got != tc.want {
			t.Errorf("zone at (%v,%v)=%v, want %v", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestOverviewKeys(t *testing.T) {
	a := newProjectApp(t)
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if len(a.tab().panes()) != 2 {
		t.Fatal("no second pane")
	}
	before := a.tab().Focus
	tt.Key(ui.Alt, ui.KeyTab)
	tt.Frame()
	tt.Advance(250 * time.Millisecond)
	tt.Frame()
	if !a.overview || !tt.HasText("Panes") {
		t.Fatalf("overview did not open: %q", tt.Texts())
	}
	if !tt.HasText("A quiet place") {
		t.Fatalf("overview has no preview text: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	if a.tab().Focus == before {
		t.Fatal("Tab in overview did not switch panes")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.overview {
		t.Fatal("Return did not leave the overview")
	}
	tt.Key(ui.Alt, ui.KeyTab)
	tt.Frame()
	tt.Key(ui.Alt, ui.KeyTab)
	tt.Frame()
	if a.overview {
		t.Fatal("Option+Tab did not leave the overview")
	}
}

func TestOverviewClickAndArrows(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Key(ui.Alt, ui.KeyTab)
	tt.Frame()
	tt.Advance(250 * time.Millisecond)
	tt.Frame()
	if !a.overview {
		t.Fatal("overview did not open")
	}
	tt.Key(ui.Alt, ui.KeyLeft)
	tt.Frame()
	if a.tab().Focus != first {
		t.Fatal("Option+Left did not move to the pane on the left")
	}
	label := "Focus pane Reading notes"
	if _, ok := tt.Find(label); !ok {
		t.Fatalf("no click target for the right pane: %q", tt.Texts())
	}
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.overview {
		t.Fatal("clicking a pane did not close the overview")
	}
	if got := a.tab().Focus.Note.Title; got != "Reading notes" {
		t.Fatalf("focused %q, want Reading notes", got)
	}
}

func TestMovePaneOutOfNestedSplit(t *testing.T) {
	a := newProjectApp(t)
	t0 := a.tab()
	aPane := t0.Focus
	bPane := newPane(a.store.byRelPath("Reading notes.md"))
	cPane := newPane(a.store.byRelPath("scripts/tasks.py"))
	t0.split(aPane, bPane, false)
	t0.split(bPane, cPane, true)
	if !t0.movePane(cPane, aPane, dropAbove) {
		t.Fatal("moving a nested pane failed")
	}
	if len(t0.panes()) != 3 {
		t.Fatalf("panes after move: %d", len(t0.panes()))
	}
	if t0.Root.Vertical || t0.Root.A == nil || !t0.Root.A.Vertical {
		t.Fatalf("root after moving C above A: %+v", t0.Root)
	}
	if t0.Root.A.A.Pane != cPane || t0.Root.A.B.Pane != aPane || t0.Root.B.Pane != bPane {
		t.Fatal("the nested move put the panes in the wrong order")
	}
}

func TestTerminalWithoutFileOpensAtProjectRoot(t *testing.T) {
	a := newProjectApp(t)
	a.shellStarter = func() (*termTab, error) { return &termTab{}, nil }
	a.tabs = nil
	a.active = -1
	a.treeSel = "scripts/tasks.py"
	if got := a.terminalDir(); got != a.store.dir {
		t.Fatalf("terminal directory %q, want project root %q", got, a.store.dir)
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

func TestCanvasExtendsPastTheViewport(t *testing.T) {
	a := newProjectApp(t)
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	t0 := a.tab()
	cw, ch := t0.Root.extent(840, 600)
	if cw < 840-1 {
		t.Fatalf("two columns did not fit the viewport: %v", cw)
	}
	if ch > 600+1 {
		t.Fatalf("a right split changed the height: %v", ch)
	}
	a.splitWith(a.store.byRelPath("scripts/tasks.py"), true)
	cw, ch = t0.Root.extent(840, 600)
	if ch <= 600 {
		t.Fatalf("canvas height %v did not grow after a down split", ch)
	}
	a.splitWith(a.store.byRelPath("scripts/app.js"), false)
	cw, _ = t0.Root.extent(840, 600)
	if cw <= 840 {
		t.Fatalf("another column did not extend the canvas: %v", cw)
	}
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	if t0.panY <= 0 {
		t.Fatalf("the viewport did not follow the focused pane: panY=%v", t0.panY)
	}
}

func TestCanvasDividerResize(t *testing.T) {
	a := newProjectApp(t)
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	t0 := a.tab()
	left := t0.Root.A.Pane
	right := t0.Root.B.Pane
	before := left.W
	t0.Root.resize(40, 840, 600)
	if left.W <= before || right.W >= paneColumnWidth {
		t.Fatalf("resize moved widths: left %v right %v", left.W, right.W)
	}
}

func TestOptionArrowFocusAndMove(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if a.tab().Focus == first {
		t.Fatal("the split did not focus the right pane")
	}
	tt.Key(ui.Alt, ui.KeyLeft)
	tt.Frame()
	if a.tab().Focus != first {
		t.Fatal("Option+Left did not select the pane on the left")
	}
	tt.Key(ui.Alt|ui.Shift, ui.KeyRight)
	tt.Frame()
	if got := a.tab().panes(); len(got) != 2 || got[0] == first || got[1] != first {
		t.Fatalf("Option+Shift+Right did not move the selected pane right: %v", got)
	}
}

func TestOptionDirectionUsesCanvasGeometry(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab().Focus
	a.splitWith(a.store.byRelPath("Reading notes.md"), false)
	a.splitWith(a.store.byRelPath("scripts/tasks.py"), true)
	below := a.tab().Focus
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	a.tab().Focus = first
	tt.Frame()
	tt.Key(ui.Alt, ui.KeyDown)
	tt.Frame()
	if a.tab().Focus != below {
		t.Fatalf("Option+Down focused %v, want the pane below", a.tab().Focus.ID)
	}
}
