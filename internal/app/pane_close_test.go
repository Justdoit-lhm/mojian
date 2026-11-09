package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestClosingPaneFadesOverItsOldPlace(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.splitWith(a.store.ByRelPath("Reading notes.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if got := len(a.tab().panes()); got != 2 {
		t.Fatalf("panes: %d", got)
	}
	target := a.tab().Focus
	var survivor *pane
	for _, p := range a.tab().panes() {
		if p != target {
			survivor = p
		}
	}
	a.closePane(target)
	tt.Frame()
	if !target.closing {
		t.Fatal("the closing pane did not start fading")
	}
	if got := len(a.paneOverlays); got != 1 {
		t.Fatalf("closing overlays: %d", got)
	}
	if got := len(a.tab().panes()); got != 1 || a.tab().panes()[0] != survivor {
		t.Fatalf("the layout did not close at once: %+v", a.tab().panes())
	}
	if a.tab().Focus != survivor {
		t.Fatal("focus did not move to the survivor")
	}
	tt.Advance(60 * time.Millisecond)
	tt.Frame()
	if !target.closing || len(a.paneOverlays) != 1 {
		t.Fatal("the overlay left mid-fade")
	}
	tt.Advance(400 * time.Millisecond)
	tt.Frame()
	tt.Frame()
	if target.closing || len(a.paneOverlays) != 0 {
		t.Fatal("the closing overlay never finished")
	}
}
