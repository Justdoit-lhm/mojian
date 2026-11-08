package app

import (
	"testing"

	"mojian/internal/session"
)

func poolApp() *app {
	return &app{
		workspace:  &workspace{},
		workspaces: map[string]*workspace{},
		remotes:    map[string]*remoteConn{"box": {host: "box"}},
	}
}

func TestRemotePoolReleasesTheLastUser(t *testing.T) {
	a := poolApp()
	a.reapRemotes()
	if _, ok := a.remotes["box"]; ok {
		t.Fatal("an unused connection stayed in the pool")
	}

	a.workspaces["project"] = &workspace{serverAlias: "box", remotePath: "/srv/app"}
	a.remotes["box"] = &remoteConn{host: "box"}
	a.reapRemotes()
	if _, ok := a.remotes["box"]; !ok {
		t.Fatal("a project's connection was released")
	}

	// A terminal alone still holds the connection.
	p := newTermPane(newTermPaneSpec(termSpec{Server: "box"}, "sid"))
	a.workspaces = map[string]*workspace{
		"terminal": {tabs: []*tab{{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}}},
	}
	a.reapRemotes()
	if _, ok := a.remotes["box"]; !ok {
		t.Fatal("a terminal's connection was released")
	}

	// The picker, while open, is a user too.
	a.workspaces = map[string]*workspace{}
	a.remotePickHost = "box"
	a.reapRemotes()
	if _, ok := a.remotes["box"]; !ok {
		t.Fatal("the picker's connection was released")
	}
	a.remotePickHost = ""
	a.reapRemotes()
	if _, ok := a.remotes["box"]; ok {
		t.Fatal("the last-user release did not close the connection")
	}
}

func TestPickerConnectionSurvivesPendingOpen(t *testing.T) {
	a := poolApp()
	a.remotes["box"] = &remoteConn{host: "box"}
	a.remotePickHost = "box"
	a.remoteOpening = true
	a.dropPickerRemote()
	if _, ok := a.remotes["box"]; !ok {
		t.Fatal("the picker connection was dropped while a project was opening")
	}
	if a.remotePickHost != "box" {
		t.Fatal("the pending picker host was cleared")
	}
	a.remoteOpening = false
	a.dropPickerRemote()
	if _, ok := a.remotes["box"]; ok {
		t.Fatal("the picker connection stayed after the pending open ended")
	}
}

func TestInstallRemoteConnPatchesCachedStores(t *testing.T) {
	oldClient, newClient := &session.Client{}, &session.Client{}
	s := &store{fs: &serverFS{client: oldClient, root: "/srv/app", local: false}}
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
		remotes:    map[string]*remoteConn{},
	}
	rc := &remoteConn{host: "box", client: newClient}
	a.installRemoteConn(rc)
	if s.client() != newClient {
		t.Fatal("the cached store kept its old client")
	}
	if p.Term.Client != newClient || p.Term.startErr != "" {
		t.Fatal("the cached terminal kept its old client")
	}
	if a.remotes["box"] != rc {
		t.Fatal("the connection was not pooled")
	}
}
