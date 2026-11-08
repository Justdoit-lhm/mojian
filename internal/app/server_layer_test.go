//go:build darwin || linux

package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"mojian/internal/fs"
	gitpage "mojian/internal/git"
	"mojian/internal/remote"
	"mojian/internal/session"

	integration "mojian/test"
)

func TestLocalProjectStoreUsesSessionServer(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "folder", "nested.md"), []byte("# Nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := openServerStore(client, root, t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	if s.ByRelPath(filepath.Join("folder", "nested.md")) == nil {
		t.Fatalf("the server listing lost a nested file: %+v", s.Notes)
	}
	if !s.Dirs["folder"] {
		t.Fatal("the server listing lost a folder")
	}
	if s.FS.Kind() != fs.KindServerLocal {
		t.Fatal("the store was not marked as local-server backed")
	}
	n := s.ByRelPath("a.md")
	if n == nil {
		t.Fatalf("the listing has no a.md: %+v", s.Notes)
	}
	if err := s.LoadNote(n); err != nil {
		t.Fatal(err)
	}
	if got := n.Text(); got != "# A\n" {
		t.Fatalf("note text: %q", got)
	}
	n.Buffer().Set("# changed through the server\n")
	if err := s.Save(n); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# changed through the server\n" {
		t.Fatalf("server file: %q", data)
	}
	if _, _, err := s.Rename("a.md", "b.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.md")); err == nil {
		t.Fatal("the old server file remained")
	}
	data, err = os.ReadFile(filepath.Join(root, "b.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# changed through the server\n" {
		t.Fatalf("renamed file: %q", data)
	}
}

func TestAppOpenProjectUsesLocalServer(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &app{
		workspace:     &workspace{treeOpen: true},
		dataDir:       t.TempDir(),
		sessionClient: client,
	}
	a.openProject(project)
	if a.store == nil || a.store.FS.Kind() != fs.KindServerLocal {
		t.Fatalf("project did not use the local server: %+v", a.store)
	}
	n := a.currentNote()
	if n == nil || n.Rel != "a.md" {
		t.Fatalf("current note: %+v", n)
	}
	n.Buffer().Set("# saved through the local server\n")
	if err := a.saveNow(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(project, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# saved through the local server\n" {
		t.Fatalf("saved file: %q", data)
	}
}

func TestLocalServerGitRunner(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "base.md"), []byte("# Base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	integration.InitRepo(t, root)
	s, err := openServerStore(client, root, t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{workspace: &workspace{store: s}}
	if _, ok := a.GitRunner().(serverGitRunner); !ok {
		t.Fatalf("Git runner is %T, want serverGitRunner", a.GitRunner())
	}
	if _, err := s.NewFileIn("", "new.md", "# New\n"); err != nil {
		t.Fatal(err)
	}
	data := gitpage.ReadStatus(a.GitRunner(), root)
	if data.Err != "" {
		t.Fatal(data.Err)
	}
	found := false
	for _, c := range data.Changes {
		if c.Path == "new.md" && c.Status == "U" {
			found = true
		}
	}
	if !found {
		t.Fatalf("local server Git status: %+v", data.Changes)
	}
}

func TestDetachRemoteConnKeepsServer(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer func() {
		if c, err := session.ConnectAt(session.SocketPath()); err == nil {
			_ = c.Shutdown()
			_ = c.Close()
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("session server did not stop")
		}
	}()
	rc := &remote.Conn{Host: "buildbox", Client: client}
	a := &app{workspace: &workspace{}}
	a.detachRemoteConn(rc)
	select {
	case <-client.Closed():
	case <-time.After(time.Second):
		t.Fatal("detaching did not close the control client")
	}
	next, err := session.ConnectAt(session.SocketPath())
	if err != nil {
		t.Fatalf("the server stopped with the connection: %v", err)
	}
	_ = next.Close()
}

func TestLocalServerFileOperations(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := openServerStore(client, root, t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	copyPath := filepath.Join(outside, "copy.md")
	if err := os.WriteFile(copyPath, []byte("# Copy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest, err := s.CopyInto(copyPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if dest == "" {
		t.Fatal("copyInto returned no path")
	}
	if data, err := os.ReadFile(filepath.Join(root, dest)); err != nil || string(data) != "# Copy\n" {
		t.Fatalf("copied file: %q %v", data, err)
	}
	if _, err := os.Stat(copyPath); err != nil {
		t.Fatal("copyInto removed its source")
	}

	dest, err = s.MoveInto(filepath.Join(root, "a.md"), "folder")
	if err != nil {
		t.Fatal(err)
	}
	if dest == "" {
		t.Fatal("moveInto returned no path")
	}
	if _, err := os.Stat(filepath.Join(root, "a.md")); err == nil {
		t.Fatal("moveInto left the old file")
	}
	if data, err := os.ReadFile(filepath.Join(root, dest)); err != nil || string(data) != "# A\n" {
		t.Fatalf("moved file: %q %v", data, err)
	}

	movePath := filepath.Join(outside, "move.md")
	if err := os.WriteFile(movePath, []byte("# Move\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest, err = s.MoveInto(movePath, "")
	if err != nil {
		t.Fatal(err)
	}
	if dest == "" {
		t.Fatal("moveInto returned no path")
	}
	if _, err := os.Stat(movePath); err == nil {
		t.Fatal("moveInto left the outside source")
	}
	if data, err := os.ReadFile(filepath.Join(root, dest)); err != nil || string(data) != "# Move\n" {
		t.Fatalf("moved outside file: %q %v", data, err)
	}
}
