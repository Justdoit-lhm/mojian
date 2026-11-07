//go:build darwin || linux

package app

import (
	"os"
	"testing"
	"time"

	"mojian/internal/session"
)

func TestTerminalSessionSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MOJIAN_DIR", dir)
	go session.Serve()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(session.SocketPath()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("session server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	a := newProjectApp(t)
	a.layoutPath = dir + "/layout.json"
	tt, err := a.startTerminal()
	if err != nil {
		t.Fatal(err)
	}
	a.addTerminalPane(tt)
	a.saveLayout()
	sid := tt.SID
	if sid == "" {
		t.Fatal("no session id")
	}
	a.closeTerminals()
	_ = a.sessionClient.Close()
	a.sessionClient = nil
	a.tabs = nil
	a.active = -1

	b := &app{store: a.store, layoutPath: a.layoutPath, treeOpen: true}
	if !b.restoreLayout() {
		t.Fatal("layout did not restore")
	}
	p := b.tab().Focus
	if p == nil || p.Term == nil || p.Term.SID != sid {
		t.Fatalf("restored pane: %+v", p)
	}
	b.startRestoredTerminal(p)
	if p.Term.Term == nil || p.Term.startErr != "" {
		t.Fatalf("reattach failed: %v", p.Term.startErr)
	}
	b.endTerminal(p.Term)
	if b.sessionClient != nil {
		_ = b.sessionClient.Shutdown()
		_ = b.sessionClient.Close()
	}
}
