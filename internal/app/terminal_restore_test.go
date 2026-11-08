//go:build darwin || linux

package app

import (
	"os"
	"testing"
	"time"

	"mojian/internal/session"
)

// startTestSessionServer starts a server in dir and returns a client and
// the channel its Serve call reports on.
func startTestSessionServer(t *testing.T, dir string) (*session.Client, <-chan error) {
	t.Helper()
	t.Setenv("MOJIAN_DIR", dir)
	done := make(chan error, 1)
	go func() { done <- session.Serve() }()
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
	c, err := session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	return c, done
}

func TestRestoredTerminalUsesItsOwnServer(t *testing.T) {
	localDir := t.TempDir()
	oneDir := t.TempDir()
	twoDir := t.TempDir()
	local, localDone := startTestSessionServer(t, localDir)
	defer local.Close()
	one, oneDone := startTestSessionServer(t, oneDir)
	defer one.Close()
	two, twoDone := startTestSessionServer(t, twoDir)
	defer two.Close()

	info, err := two.Create(session.CreateOptions{
		Command: []string{"/bin/sh", "-c", "sleep 5"},
		Cols:    40, Rows: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	a := &app{
		sessionClient: local,
		serverAlias:   "one",
		remotePath:    "/srv/app",
		remote:        &remoteConn{host: "one", client: one},
		remotes: map[string]*remoteConn{
			"one": {host: "one", client: one},
			"two": {host: "two", client: two},
		},
	}
	p := newTermPane(newTermPaneSpec(termSpec{Server: "two"}, info.ID))
	a.startRestoredTerminal(p)
	if p.Term.startErr != "" {
		t.Fatalf("restoring the terminal: %s", p.Term.startErr)
	}
	if p.Term.Term == nil {
		t.Fatal("the terminal was not attached")
	}
	if p.Term.SID != info.ID || p.Term.Client != two {
		t.Fatalf("terminal used %q on %p, want %q on the second server", p.Term.SID, p.Term.Client, info.ID)
	}
	if a.remote == nil || a.remote.host != "one" || a.serverAlias != "one" {
		t.Fatal("restoring a terminal changed the active project's server")
	}
	if p.Term.stream != nil {
		_ = p.Term.stream.Close()
	}
	if p.Term.Term != nil {
		_ = p.Term.Term.Close()
	}
	for _, c := range []*session.Client{local, one, two} {
		if err := c.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}
	for name, done := range map[string]<-chan error{"local": localDone, "one": oneDone, "two": twoDone} {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("%s server did not stop", name)
		}
	}
}

func TestRestoredTerminalUsesTheRemoteServersSocket(t *testing.T) {
	localDir, err := os.MkdirTemp("", "mj-local")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(localDir)
	remoteDir, err := os.MkdirTemp("", "mj-remote")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(remoteDir)

	local, localDone := startTestSessionServer(t, localDir)
	defer local.Close()
	remote, remoteDone := startTestSessionServer(t, remoteDir)
	defer remote.Close()

	info, err := remote.Create(session.CreateOptions{
		Command: []string{"/bin/sh", "-c", "sleep 5"},
		Cols:    40, Rows: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	a := &app{
		sessionClient: local,
		remote:        &remoteConn{host: "remote", client: remote},
	}
	p := newTermPane(newTermPaneSpec(termSpec{Server: "remote"}, info.ID))
	a.startRestoredTerminal(p)
	if p.Term.startErr != "" {
		t.Fatalf("restoring the terminal: %s", p.Term.startErr)
	}
	if p.Term.Term == nil {
		t.Fatal("the terminal was not attached")
	}
	if p.Term.SID != info.ID {
		t.Fatalf("restored a new session %q instead of %q", p.Term.SID, info.ID)
	}
	if p.Term.Client != remote {
		t.Fatal("the terminal attached through the wrong client")
	}
	if p.Term.stream != nil {
		_ = p.Term.stream.Close()
	}
	if p.Term.Term != nil {
		_ = p.Term.Term.Close()
	}

	if err := remote.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if err := local.Shutdown(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-remoteDone:
	case <-time.After(3 * time.Second):
		t.Fatal("remote server did not stop")
	}
	select {
	case <-localDone:
	case <-time.After(3 * time.Second):
		t.Fatal("local server did not stop")
	}
}
