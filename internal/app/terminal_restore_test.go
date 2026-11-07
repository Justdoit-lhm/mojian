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
	p := newTermPane(newTermPaneSpec(termSpec{}, info.ID))
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
