package app

import (
	"errors"
	"testing"

	"mojian/internal/fs"
	"mojian/internal/remote"
	"mojian/internal/session"
)

func poolApp() *app {
	a := &app{workspace: &workspace{}, workspaces: map[string]*workspace{}}
	a.setRemoteConn(&remote.Conn{Host: "box"})
	return a
}

func TestKeptConnectionSurvivesReapAndPicker(t *testing.T) {
	a := poolApp()
	a.keepRemote("box")
	a.reapRemotes()
	if a.remoteConn("box") == nil {
		t.Fatal("a kept connection was released by reaping")
	}

	a.remotePickHost = "box"
	a.dropPickerRemote()
	if a.remoteConn("box") == nil {
		t.Fatal("closing the picker released a kept connection")
	}
	if a.remotePickHost != "" {
		t.Fatalf("the picker host stayed: %q", a.remotePickHost)
	}

	delete(a.remoteKeep, "box")
	a.reapRemotes()
	if a.remoteConn("box") != nil {
		t.Fatal("the connection stayed after its last project closed")
	}
}

func TestRemotePoolReleasesTheLastUser(t *testing.T) {
	a := poolApp()
	a.reapRemotes()
	if a.remoteConn("box") != nil {
		t.Fatal("an unused connection stayed in the pool")
	}

	a.workspaces["project"] = &workspace{serverAlias: "box", remotePath: "/srv/app"}
	a.setRemoteConn(&remote.Conn{Host: "box"})
	a.reapRemotes()
	if a.remoteConn("box") == nil {
		t.Fatal("a project's connection was released")
	}

	// A terminal alone still holds the connection.
	p := newTermPane(newTermPaneSpec(termSpec{Server: "box"}, "sid"))
	a.workspaces = map[string]*workspace{
		"terminal": {tabs: []*tab{{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}}},
	}
	a.reapRemotes()
	if a.remoteConn("box") == nil {
		t.Fatal("a terminal's connection was released")
	}

	// The picker, while open, is a user too.
	a.workspaces = map[string]*workspace{}
	a.remotePickHost = "box"
	a.reapRemotes()
	if a.remoteConn("box") == nil {
		t.Fatal("the picker's connection was released")
	}
	a.remotePickHost = ""
	a.reapRemotes()
	if a.remoteConn("box") != nil {
		t.Fatal("the last-user release did not close the connection")
	}
}

func TestPickerConnectionSurvivesPendingOpen(t *testing.T) {
	a := poolApp()
	a.setRemoteConn(&remote.Conn{Host: "box"})
	a.remotePickHost = "box"
	a.remoteOpening = true
	a.dropPickerRemote()
	if a.remoteConn("box") == nil {
		t.Fatal("the picker connection was dropped while a project was opening")
	}
	if a.remotePickHost != "box" {
		t.Fatal("the pending picker host was cleared")
	}
	a.remoteOpening = false
	a.dropPickerRemote()
	if a.remoteConn("box") != nil {
		t.Fatal("the picker connection stayed after the pending open ended")
	}
}

func TestInstallRemoteConnPatchesCachedStores(t *testing.T) {
	oldClient, newClient := &session.Client{}, &session.Client{}
	s := &store{FS: fs.NewServer(oldClient, "/srv/app", "", false)}
	p := newTermPane(newTermPaneSpec(termSpec{Server: "box"}, "sid"))
	p.Term.Client = oldClient
	p.Term.startErr = "lost"
	ws := &workspace{
		serverAlias: "box", remotePath: "/srv/app", store: s,
		tabs: []*tab{{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}},
	}
	a := &app{
		workspace:  &workspace{},
		workspaces: map[string]*workspace{"box": ws},
	}
	rc := &remote.Conn{Host: "box", Client: newClient}
	a.installRemoteConn(rc)
	if s.Client() != newClient {
		t.Fatal("the cached store kept its old client")
	}
	if p.Term.Client != newClient || p.Term.startErr != "" {
		t.Fatal("the cached terminal kept its old client")
	}
	if a.remoteConn("box") != rc {
		t.Fatal("the connection was not pooled")
	}
}

func TestTerminalWaitStateFollowsTheConnection(t *testing.T) {
	p := &pane{Term: &termTab{Spec: termSpec{Server: "box"}, connecting: true}}
	tb := &tab{Root: &node{Pane: p}, Focus: p}
	ws := &workspace{serverAlias: "box", remotePath: "/srv/app", tabs: []*tab{tb}}
	a := &app{workspace: ws, workspaces: map[string]*workspace{"remote:box:/srv/app": ws}}

	detachTerminalsOn(ws, "box", nil)
	if p.Term.connecting || p.Term.startErr != "" {
		t.Fatalf("a new connection did not reset the terminal: connecting=%v err=%q", p.Term.connecting, p.Term.startErr)
	}

	p.Term.connecting = true
	a.failTerminalsOn("box", errors.New("unreachable"))
	if p.Term.connecting {
		t.Fatal("a failed reconnect left the terminal waiting forever")
	}
	if p.Term.startErr != "unreachable" {
		t.Fatalf("a failed reconnect did not surface its error: %q", p.Term.startErr)
	}
}
