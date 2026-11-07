//go:build darwin || linux

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitpage "mojian/internal/git"
	"mojian/internal/session"
)

// newRemoteApp opens root through a local session server, as a remote
// project would through its forward.
func newRemoteApp(t *testing.T, client *session.Client, root string) *app {
	t.Helper()
	s, err := openRemoteStore(client, root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &app{
		store: s, treeOpen: true, openDirs: map[string]bool{},
		remote:      &remoteConn{host: "buildbox", client: client},
		serverAlias: "buildbox", remotePath: root,
	}
	n := s.byRelPath("a.md")
	if n == nil {
		t.Fatalf("the listing has no a.md: %+v", s.notes)
	}
	a.openNote(n)
	if a.currentNote() == nil {
		t.Fatal("the remote note did not open")
	}
	return a
}

func TestRemoteProjectReadsAndSavesOnTheServer(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newRemoteApp(t, client, root)
	if got := a.currentNote().text(); got != "# A\n" {
		t.Fatalf("note text: %q", got)
	}
	a.currentNote().buffer().Set("# changed on the server\n")
	if err := a.saveNow(); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# changed on the server\n" {
		t.Fatalf("server file: %q", data)
	}
}

func TestRemoteProjectSavesBeforeLeaving(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newRemoteApp(t, client, root)
	a.currentNote().buffer().Set("# saved before switch\n")
	target := t.TempDir()
	a.openProject(target)
	if a.remotePath != "" || a.remote != nil {
		t.Fatalf("the remote project stayed open: %q %+v", a.remotePath, a.remote)
	}
	data, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# saved before switch\n" {
		t.Fatalf("server file after switching: %q", data)
	}
}

func TestFailedRemoteSaveKeepsTheNoteDirty(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newRemoteApp(t, client, root)
	n := a.currentNote()
	n.buffer().Set("# unsaved\n")
	_ = client.Close()
	if err := a.saveNow(); err == nil {
		t.Fatal("saveNow hid a lost remote connection")
	}
	if !n.dirty() {
		t.Fatal("a failed remote save marked the note saved")
	}
	if c, err := session.ConnectAt(session.SocketPath()); err == nil {
		_ = c.Shutdown()
		_ = c.Close()
	}
}

func TestRemoteProjectRenamesOnTheServer(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newRemoteApp(t, client, root)
	if _, _, err := a.store.rename("a.md", "b.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.md")); err == nil {
		t.Fatal("the old remote file remained")
	}
	data, err := os.ReadFile(filepath.Join(root, "b.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# A\n" {
		t.Fatalf("renamed file: %q", data)
	}
	if a.store.byRelPath("b.md") == nil || a.store.byRelPath("a.md") != nil {
		t.Fatalf("notes after rename: %+v", a.store.notes)
	}
}

func TestRemoteGitRunsOnTheServer(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Exec(root, []string{"git", "-C", root, "init", "-q"}, nil, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	runner := remoteGitRunner{client: client}
	data := gitpage.ReadStatus(runner, root)
	if data.Err != "" || len(data.Changes) != 1 || data.Changes[0].Status != "U" {
		t.Fatalf("remote status: %+v", data)
	}
	diff, err := gitpage.ReadDiff(runner, data.Root, data.Changes[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "new file") || !strings.Contains(diff, "+# A") {
		t.Fatalf("remote untracked diff: %q", diff)
	}
}
