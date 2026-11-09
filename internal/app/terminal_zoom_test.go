//go:build darwin || linux

package app

import (
	"testing"
	"time"

	"mojian/internal/session"

	"github.com/egoist/mygo/ui"
)

// A terminal cannot lay its grid out at every width without work, but its
// card still flies to the tab's box when the pane zooms, like a note's:
// the grid settles at each width on the way.
func TestTerminalPaneFliesWhenZoomed(t *testing.T) {
	registerFonts()
	client, done := startTestSessionServer(t, t.TempDir())
	defer func() {
		_ = client.Shutdown()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("the session server did not stop")
		}
	}()
	info, err := client.Create(session.CreateOptions{
		Command: []string{"/bin/sh", "-c", "sleep 5"},
		Cols:    40, Rows: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	a := newProjectApp(t)
	a.sessionClient = client
	tb := a.tab()
	term := newTermPane(newTermPaneSpec(termSpec{Server: localServerID}, info.ID))
	tb.Root = &node{Vertical: false, Share: 50, A: tb.Root, B: &node{Pane: term}}
	tb.Focus = term
	a.focusReq = term
	a.startRestoredTerminal(term)
	if term.Term.startErr != "" {
		t.Fatalf("the terminal did not attach: %s", term.Term.startErr)
	}
	if term.Term.Term == nil {
		t.Fatal("the terminal did not attach")
	}
	defer a.endTerminalPane(term, true)

	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	before, ok := tt.Find("Move Pane Terminal")
	if !ok {
		t.Fatalf("no terminal grip: %q", tt.Texts())
	}
	a.toggleZoom()
	tt.Frame()
	start, _ := tt.Find("Move Pane Terminal")
	if start.X < before.X-20 {
		t.Fatalf("the terminal jumped to the tab at once: %.1f, from %.1f", start.X, before.X)
	}
	tt.Advance(60 * time.Millisecond)
	mid, _ := tt.Find("Move Pane Terminal")
	if mid.X > start.X-20 {
		t.Fatalf("the terminal did not move toward the tab: %.1f, from %.1f", mid.X, start.X)
	}
	tt.Advance(400 * time.Millisecond)
	end, _ := tt.Find("Move Pane Terminal")
	if end.X > start.X-100 {
		t.Fatalf("the terminal did not reach the tab: %.1f, from %.1f", end.X, start.X)
	}
}
