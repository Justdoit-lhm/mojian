package app

import "testing"

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
