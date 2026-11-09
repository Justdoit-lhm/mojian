package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mojian/internal/remote"
	"mojian/internal/session"
	"mojian/internal/testutil"

	"github.com/egoist/mygo/ui"
)

func TestProjectSwitch(t *testing.T) {
	startLocalServerForTest(t)
	root := t.TempDir()
	alpha := filepath.Join(root, "Alpha")
	beta := filepath.Join(root, "Beta")
	testutil.WriteVault(t, alpha, map[string]string{"alpha.md": "# Alpha\n"})
	testutil.WriteVault(t, beta, map[string]string{"beta.py": "print('beta')\n"})

	data := t.TempDir()
	a := &app{
		workspace:    &workspace{treeOpen: true},
		dataDir:      data,
		settingsPath: filepath.Join(data, "settings.json"),
	}
	s, err := openStoreAt(alpha)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.Dir)
	a.rememberProject(s.Dir)
	a.focusFirstNote()
	a.saveLayout()
	a.saveSettings()
	alphaLayout := a.layoutPath
	alphaStore := a.store
	alphaNote := a.currentNote()
	alphaTabs := a.tabs

	a.openProject(beta)
	if a.store.Dir != beta {
		t.Fatalf("store dir: %q", a.store.Dir)
	}
	if a.currentNote() == nil || a.currentNote().Rel != "beta.py" {
		t.Fatalf("current note: %+v", a.currentNote())
	}
	if a.layoutPath == alphaLayout {
		t.Fatal("projects share a layout path")
	}
	if got := a.projectName(); got != "Beta" {
		t.Errorf("project name: %q", got)
	}
	if a.project != beta {
		t.Errorf("saved project: %q", a.project)
	}
	if len(a.projects) < 2 || a.projects[0] != beta {
		t.Errorf("recent projects: %q", a.projects)
	}

	var saved settingsFile
	dataJSON, err := os.ReadFile(a.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(dataJSON, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Project != beta {
		t.Errorf("saved project: %q", saved.Project)
	}

	// Switching back restores the loaded project, not a fresh read.
	a.openProject(alpha)
	if a.store != alphaStore || len(a.tabs) != len(alphaTabs) || a.tabs[0] != alphaTabs[0] {
		t.Fatal("switching back read the project again")
	}
	if a.currentNote() != alphaNote {
		t.Fatal("switching back opened another note")
	}
	if a.store.Dir != alpha {
		t.Fatalf("store dir: %q", a.store.Dir)
	}
	if len(a.tabs) != 1 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	if a.currentNote() == nil || a.currentNote().Rel != "alpha.md" {
		t.Fatalf("restored note: %+v", a.currentNote())
	}
	if a.project != alpha {
		t.Errorf("project after switching back: %q", a.project)
	}
	if a.projects[0] != alpha || len(a.projects) != 2 {
		t.Errorf("recent projects: %q", a.projects)
	}
}

func TestProjectChip(t *testing.T) {
	registerFonts()
	dir := filepath.Join(t.TempDir(), "Quiet")
	testutil.WriteVault(t, dir, map[string]string{"note.md": "# A note\n", "app.js": "console.log(1);\n"})
	a := &app{workspace: &workspace{treeOpen: true}, dataDir: t.TempDir()}
	s, err := openStoreAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.Dir)
	a.rememberProject(s.Dir)
	a.focusFirstNote()
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()

	if !tt.HasText("Quiet") {
		t.Fatalf("texts: %q", tt.Texts())
	}
	if tt.HasText("2 files") || tt.HasText("1 file") {
		t.Fatal("the project chip still shows a file count")
	}
	if err := tt.Click("Switch project: Quiet"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Projects") {
		t.Fatalf("no project popover: %q", tt.Texts())
	}
	if !tt.HasText("Open Folder…") {
		t.Fatal("the project popover has no Open Folder")
	}
}

func TestCloseRemoteProjectReleasesLastConnection(t *testing.T) {
	registerFonts()
	startLocalServerForTest(t)
	a := newProjectApp(t)
	key := workspaceKeyRemote("buildbox", "/srv/app")
	a.workspace.key, a.workspace.serverAlias, a.workspace.remotePath = key, "buildbox", "/srv/app"
	a.workspaces = map[string]*workspace{key: a.workspace}
	a.serverAlias, a.remotePath = "buildbox", "/srv/app"
	a.rememberRemoteProject(remoteProject{Host: "buildbox", Path: "/srv/app"})
	client, err := session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	a.setRemoteConn(&remote.Conn{Host: "buildbox", Client: client})
	a.keepRemote("buildbox")

	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("Switch project: app"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := tt.Find("Remove project /srv/app"); !ok {
		t.Fatalf("the remote project row has no remove button: %q", tt.Texts())
	}
	if err := tt.Click("Remove project /srv/app"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()

	if a.workspaces[key] != nil {
		t.Fatal("the closed project stayed in the workspace cache")
	}
	if a.workspace.store != nil || a.serverAlias != "" || a.remotePath != "" {
		t.Fatalf("the closed project is still open: server=%q path=%q", a.serverAlias, a.remotePath)
	}
	if a.remoteConn("buildbox") != nil {
		t.Fatal("the last project's connection stayed in the pool")
	}
	found := false
	for _, p := range a.remoteProjects() {
		if p.Host == "buildbox" && p.Path == "/srv/app" {
			found = true
		}
	}
	if found {
		t.Fatalf("the removed project stayed in the remembered list: %+v", a.remoteProjects())
	}
}

func TestCloseLocalProjectSwitchesToRecent(t *testing.T) {
	registerFonts()
	startLocalServerForTest(t)
	root := t.TempDir()
	active := filepath.Join(root, "Active")
	other := filepath.Join(root, "Other")
	testutil.WriteVault(t, active, map[string]string{"a.md": "# A\n"})
	testutil.WriteVault(t, other, map[string]string{"b.md": "# B\n"})
	a := &app{workspace: &workspace{treeOpen: true}, dataDir: t.TempDir()}
	s, err := openStoreAt(active)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.Dir)
	a.projects = []string{active, other}
	a.project = active
	a.focusFirstNote()
	a.registerActiveWorkspace()
	fallback, err := openStoreAt(other)
	if err != nil {
		t.Fatal(err)
	}
	key := workspaceKeyLocal(other)
	a.workspaces[key] = &workspace{
		key:      key,
		project:  other,
		store:    fallback,
		treeOpen: true,
		active:   -1,
		openDirs: map[string]bool{},
	}

	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("Switch project: Active"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := tt.Find("Remove project " + active); !ok {
		t.Fatalf("the local project row has no remove button: %q", tt.Texts())
	}
	if err := tt.Click("Remove project " + active); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()

	if a.store == nil || a.store.Dir != other {
		t.Fatalf("the fallback project did not open: %+v", a.store)
	}
	if a.project != other {
		t.Fatalf("last project after closing: %q", a.project)
	}
	if a.workspaces[workspaceKeyLocal(active)] != nil {
		t.Fatal("the closed project stayed in the workspace cache")
	}
	found := false
	for _, p := range a.projects {
		if p == active {
			found = true
		}
	}
	if found {
		t.Fatalf("the removed project stayed in the recent list: %q", a.projects)
	}
}

func TestRemoveUnloadedRecentProject(t *testing.T) {
	registerFonts()
	startLocalServerForTest(t)
	root := t.TempDir()
	active := filepath.Join(root, "Active")
	old := filepath.Join(root, "Old")
	testutil.WriteVault(t, active, map[string]string{"a.md": "# A\n"})
	testutil.WriteVault(t, old, map[string]string{"old.md": "# Old\n"})
	a := &app{workspace: &workspace{treeOpen: true}, dataDir: t.TempDir()}
	s, err := openStoreAt(active)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.Dir)
	a.projects = []string{active, old}
	a.project = active
	a.focusFirstNote()
	a.registerActiveWorkspace()

	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("Switch project: Active"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Remove project " + old); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	if a.store == nil || a.store.Dir != active {
		t.Fatal("removing a recent project changed the open project")
	}
	for _, p := range a.projects {
		if p == old {
			t.Fatalf("the removed project stayed in the recent list: %q", a.projects)
		}
	}
}

func TestRemoteWorkspaceSwitchUsesTheCache(t *testing.T) {
	if !terminalsSupported() {
		t.Skip("remote projects need terminals")
	}
	startLocalServerForTest(t)
	root := t.TempDir()
	dir := filepath.Join(root, "App")
	testutil.WriteVault(t, dir, map[string]string{"main.go": "package main\n"})
	a := &app{workspace: &workspace{treeOpen: true}, dataDir: t.TempDir()}
	s, err := openStoreAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.Dir)
	a.rememberProject(s.Dir)
	a.focusFirstNote()
	a.registerActiveWorkspace()

	client, err := session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	key := workspaceKeyRemote("buildbox", "/srv/app")
	a.setRemoteConn(&remote.Conn{Host: "buildbox", Client: client})
	a.workspaces[key] = &workspace{
		key:         key,
		project:     a.project,
		store:       s,
		serverAlias: "buildbox",
		remotePath:  "/srv/app",
		layoutPath:  a.projectLayoutPath("remote:buildbox:/srv/app"),
		treeOpen:    true,
		tabs:        a.tabs,
		active:      a.active,
		focusReq:    a.focusReq,
	}
	a.openRemoteProjectAsync("buildbox", "/srv/app")
	if a.remoteBusy || a.serverAlias != "buildbox" || a.remotePath != "/srv/app" {
		t.Fatalf("cached remote switch: busy=%v server=%q path=%q", a.remoteBusy, a.serverAlias, a.remotePath)
	}
	if a.store != s || len(a.tabs) == 0 {
		t.Fatal("the cached remote workspace was read again")
	}
	a.openProject(dir)
	if a.serverAlias != "" || a.remotePath != "" || a.store != s {
		t.Fatalf("switch back: server=%q path=%q store=%p", a.serverAlias, a.remotePath, a.store)
	}
}
