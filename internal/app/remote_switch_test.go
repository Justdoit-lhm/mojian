//go:build darwin || linux

package app

import (
	"path/filepath"
	"testing"

	"mojian/internal/remote"
	"mojian/internal/session"
	"mojian/internal/testutil"

	"github.com/egoist/mygo/ui"
)

// Switching to Local and back used to show the folder picker although the
// remote workspace, its tabs and its connection were still loaded.
func TestServerRowResumesLoadedProject(t *testing.T) {
	registerFonts()
	startLocalServerForTest(t)
	local := t.TempDir()
	remoteRoot := t.TempDir()
	testutil.WriteVault(t, local, map[string]string{"local.md": "# Local\n"})
	testutil.WriteVault(t, remoteRoot, map[string]string{"server.md": "# Server\n"})

	client, err := session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	rs, err := openServerStore(client, remoteRoot, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	key := workspaceKeyRemote("buildbox", remoteRoot)
	ws := &workspace{
		key:         key,
		project:     local,
		store:       rs,
		serverAlias: "buildbox",
		remotePath:  remoteRoot,
		layoutPath:  "",
		treeOpen:    true,
		active:      -1,
		openDirs:    map[string]bool{},
	}
	a := &app{workspace: ws, dataDir: t.TempDir()}
	a.workspaces = map[string]*workspace{key: ws}
	a.projects = []string{local}
	a.project = local
	a.rememberRemoteProject(remoteProject{Host: "buildbox", Path: remoteRoot})
	a.setRemoteConn(&remote.Conn{Host: "buildbox", Client: client})
	a.keepRemote("buildbox")
	a.addHost("buildbox", "buildbox")
	a.remoteHomes = map[string]string{"buildbox": filepath.Dir(remoteRoot)}
	n := rs.ByRelPath("server.md")
	if n == nil {
		t.Fatalf("the server listing has no server.md: %+v", rs.Notes)
	}
	a.openNote(n)
	if a.currentNote() != n {
		t.Fatal("the server note did not open")
	}

	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("Switch server: buildbox"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Server Local"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if a.store == rs || a.remotePath != "" {
		t.Fatalf("switch to Local kept the server project: path=%q", a.remotePath)
	}
	if a.workspaces[key] != ws {
		t.Fatal("switching to Local dropped the loaded server workspace")
	}

	if err := tt.Click("Switch server: Local"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Server buildbox"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()

	if a.remotePicker {
		t.Fatal("choosing a connected server opened the folder picker")
	}
	if a.serverAlias != "buildbox" || a.remotePath != remoteRoot {
		t.Fatalf("server=%q path=%q, want the loaded project", a.serverAlias, a.remotePath)
	}
	if a.workspace != ws || a.store != rs {
		t.Fatal("the loaded workspace was not restored")
	}
	if a.currentNote() != n {
		t.Fatal("the restored workspace lost its open note")
	}
}
