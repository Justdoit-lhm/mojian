package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRemoteBusyCoversTheWindow(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.remoteBusy = true
	a.remoteBusyMsg = "Connecting to Buildbox…"
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if !tt.HasText("Connecting to a server") {
		t.Fatalf("loading text: %q", tt.Texts())
	}
	if !tt.HasText("Connecting to Buildbox…") {
		t.Fatalf("loading detail: %q", tt.Texts())
	}
}
