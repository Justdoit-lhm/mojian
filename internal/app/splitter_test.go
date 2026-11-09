package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSplitDividerResizesPanes(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	root := a.tab().Root
	if root == nil || root.Pane != nil || root.Share != 50 {
		t.Fatalf("split: %+v", root)
	}
	first, second := a.tab().panes()[0], a.tab().panes()[1]
	if first.lastBounds.W != second.lastBounds.W {
		t.Fatalf("the split did not start even: %.1f vs %.1f", first.lastBounds.W, second.lastBounds.W)
	}
	d, ok := tt.Find("Divider")
	if !ok {
		t.Fatalf("no divider: %q", tt.Texts())
	}
	cx, cy := d.X+d.W/2, d.Y+d.H/2
	tt.Move(cx, cy)
	tt.Frame()
	if got := tt.Cursor(); got != ui.CursorResizeEW {
		t.Fatalf("the divider shows cursor %v, want a horizontal resize", got)
	}
	tt.Press(cx, cy)
	tt.Frame()
	tt.Move(cx+80, cy)
	tt.Frame()
	tt.Release(cx+80, cy)
	tt.Frame()
	tt.Frame()
	if root.Share <= 50 {
		t.Fatalf("dragging right did not grow the first pane: share %.1f", root.Share)
	}
	if first.lastBounds.W <= second.lastBounds.W {
		t.Fatalf("the panes did not follow the divider: %.1f vs %.1f", first.lastBounds.W, second.lastBounds.W)
	}
	// A double click puts the divider back in the middle.
	if err := tt.Click("Divider"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Divider"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if root.Share != 50 {
		t.Fatalf("a double click did not reset the split: %.1f", root.Share)
	}
}

func TestSplitDividerArrowKeysMoveIt(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	root := a.tab().Root
	d, ok := tt.Find("Divider")
	if !ok {
		t.Fatalf("no divider: %q", tt.Texts())
	}
	cx, cy := d.X+d.W/2, d.Y+d.H/2
	tt.Press(cx, cy)
	tt.Frame()
	tt.Release(cx, cy)
	tt.Frame()
	before := root.Share
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	tt.Frame()
	if root.Share <= before {
		t.Fatalf("the right arrow did not move the divider: %.1f", root.Share)
	}
}

func TestSplitShareKeepsBothPanesVisible(t *testing.T) {
	if got := splitShare(&node{}, 0, minPaneW); got != 50 {
		t.Fatalf("a split before its first layout: %.1f", got)
	}
	if got := splitShare(&node{Share: 10}, 1000, minPaneW); got <= 10 {
		t.Fatalf("a share narrower than the minimum was kept: %.1f", got)
	}
	if got := splitShare(&node{Share: 90}, 1000, minPaneW); got >= 90 {
		t.Fatalf("a share wider than the maximum was kept: %.1f", got)
	}
	if got := splitShare(&node{Share: 90}, 200, minPaneW); got != 90 {
		t.Fatalf("a window too narrow to clamp changed the stored share: %.1f", got)
	}
	a := &app{}
	n := &node{Share: 50}
	if !a.resizeSplit(n, 100, 1000, minPaneW) {
		t.Fatal("the drag did not move the divider")
	}
	if n.Share < 87 || n.Share > 88 {
		t.Fatalf("the share was not clamped to the minimum pane: %.2f", n.Share)
	}
}

func TestSplitVerticalDividerResizesPanes(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.splitWithSide(a.store.ByRelPath("Reading notes.md"), splitDown)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	root := a.tab().Root
	if root == nil || !root.Vertical {
		t.Fatalf("split: %+v", root)
	}
	first, second := a.tab().panes()[0], a.tab().panes()[1]
	d, ok := tt.Find("Divider")
	if !ok {
		t.Fatalf("no divider: %q", tt.Texts())
	}
	cx, cy := d.X+d.W/2, d.Y+d.H/2
	tt.Move(cx, cy)
	tt.Frame()
	if got := tt.Cursor(); got != ui.CursorResizeNS {
		t.Fatalf("the divider shows cursor %v, want a vertical resize", got)
	}
	tt.Press(cx, cy)
	tt.Frame()
	tt.Move(cx, cy+60)
	tt.Frame()
	tt.Release(cx, cy+60)
	tt.Frame()
	tt.Frame()
	if root.Share <= 50 {
		t.Fatalf("dragging down did not grow the first pane: share %.1f", root.Share)
	}
	if first.lastBounds.H <= second.lastBounds.H {
		t.Fatalf("the panes did not follow the divider: %.1f vs %.1f", first.lastBounds.H, second.lastBounds.H)
	}
}
