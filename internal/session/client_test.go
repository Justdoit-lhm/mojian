//go:build darwin || linux

package session

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConnectAtMissingSocketReturnsError(t *testing.T) {
	if _, err := ConnectAt(filepath.Join(t.TempDir(), "missing.sock")); err == nil {
		t.Fatal("ConnectAt returned a client for a missing socket")
	}
}

func TestStreamUsesTheClientsSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "mj")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	controlAccepted := make(chan struct{})
	go func() {
		control, err := ln.Accept()
		if err != nil {
			return
		}
		defer control.Close()
		// ConnectAt says hello; a server answers before the test goes on.
		rd := bufio.NewReader(control)
		if line, err := rd.ReadBytes('\n'); err == nil {
			var req Request
			if json.Unmarshal(line, &req) == nil {
				data, _ := json.Marshal(Hello{Version: ProtocolVersion})
				res, _ := json.Marshal(Response{ID: req.ID, Data: data})
				_, _ = control.Write(append(res, '\n'))
			}
		}
		close(controlAccepted)
		stream, err := ln.Accept()
		if err != nil {
			return
		}
		defer stream.Close()
		line, err := bufio.NewReader(stream).ReadBytes('\n')
		if err != nil {
			return
		}
		var attach Attach
		if json.Unmarshal(line, &attach) != nil || attach.Op != "attach" {
			return
		}
		_, _ = stream.Write([]byte(`{"ok":true}` + "\n"))
		_, _ = stream.Write([]byte("stream-ok"))
	}()

	c, err := ConnectAt(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	<-controlAccepted
	s := c.Stream("sid", 20, 10)
	defer s.Close()

	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)
	buf := make([]byte, 64)
	go func() {
		n, err := s.Read(buf)
		ch <- result{n, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("stream read: %v", r.err)
		}
		if got := string(buf[:r.n]); got != "stream-ok" {
			t.Fatalf("stream read %q, want %q", got, "stream-ok")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the stream did not attach through the client's socket")
	}
}

func TestKillReturnsBeforeTheInterruptGrace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MOJIAN_DIR", dir)
	done := make(chan error, 1)
	go func() { done <- Serve() }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(SocketPath()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	c, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	info, err := c.Create(CreateOptions{
		Command: []string{"/bin/sh", "-c", `trap "" INT; sleep 1`},
		Cols:    40, Rows: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := c.Kill(info.ID); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("kill waited %v for the shell's interrupt grace", elapsed)
	}
	if err := c.Shutdown(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}

func TestExitedSessionsArePruned(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MOJIAN_DIR", dir)
	done := make(chan error, 1)
	go func() { done <- Serve() }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(SocketPath()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	c, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Create(CreateOptions{
		Command: []string{"/bin/sh", "-c", "exit 0"},
		Cols:    40, Rows: 10,
	}); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for {
		list, err := c.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(list) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("exited session still listed: %+v", list)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := c.Shutdown(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}
