//go:build darwin || linux

package session

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestServerKeepsSessionAcrossAttach(t *testing.T) {
	t.Setenv("MOJIAN_DIR", t.TempDir())
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
	info, err := c.Create(CreateOptions{
		Command: []string{"/bin/sh", "-c", "printf 'hello-session'; sleep 2"},
		Cols:    40, Rows: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := c.Stream(info.ID, 40, 10)
	defer s.Close()
	read := make([]byte, 0, 64)
	deadline = time.Now().Add(3 * time.Second)
	buf := make([]byte, 64)
	for time.Now().Before(deadline) && !strings.Contains(string(read), "hello-session") {
		n, err := s.Read(buf)
		if n > 0 {
			read = append(read, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	if !strings.Contains(string(read), "hello-session") {
		t.Fatalf("session output: %q", read)
	}
	// Detach and attach again: the server keeps the screen.
	_ = s.Close()
	s2 := c.Stream(info.ID, 40, 10)
	defer s2.Close()
	read = read[:0]
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) && !strings.Contains(string(read), "hello-session") {
		n, err := s2.Read(buf)
		if n > 0 {
			read = append(read, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	if !strings.Contains(string(read), "hello-session") {
		t.Fatalf("snapshot after attach: %q", read)
	}
	if err := c.Shutdown(); err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}
