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
	if !tt.HasText("Cancel") {
		t.Fatalf("no cancel button: %q", tt.Texts())
	}
}

func TestRemoteBusyCancelStopsTask(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	seq, _, ctx := a.beginRemoteTask("Connecting to Buildbox…")
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.remoteBusy {
		t.Fatal("Cancel left the loading card up")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("Cancel did not stop the connection task")
	}
	if a.endRemoteTask(seq) {
		t.Fatal("a cancelled task's result still landed")
	}
	if a.remoteCancel != nil {
		t.Fatal("Cancel left the task's cancel function behind")
	}
}
