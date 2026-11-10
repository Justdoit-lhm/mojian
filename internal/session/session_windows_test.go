//go:build windows

package session

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func waitReady(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !Ready() {
		if time.Now().After(deadline) {
			t.Fatal("session server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWindowsNamedPipeSession(t *testing.T) {
	t.Setenv("MOJIAN_DIR", t.TempDir())
	done := make(chan error, 1)
	go func() { done <- Serve() }()
	waitReady(t)

	c, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	defer func() {
		_ = c.Shutdown()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("session server did not stop")
		}
	}()

	info, err := c.Create(CreateOptions{
		Command: []string{"cmd.exe", "/c", "echo mojian-ok"},
		Cols:    40,
		Rows:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Kill(info.ID)

	stream := c.Stream(info.ID, 40, 10)
	defer stream.Close()
	chunks := make(chan []byte, 16)
	go func() {
		defer close(chunks)
		buf := make([]byte, 4096)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				chunks <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()

	var out []byte
	deadline := time.After(5 * time.Second)
	for !bytes.Contains(out, []byte("mojian-ok")) {
		select {
		case p, ok := <-chunks:
			if !ok {
				t.Fatalf("terminal ended without the marker: %q", out)
			}
			out = append(out, p...)
		case <-deadline:
			t.Fatalf("terminal did not print the marker: %q", out)
		}
	}
}

func TestWindowsNamedPipeIsUserScoped(t *testing.T) {
	t.Setenv("MOJIAN_DIR", t.TempDir())
	if path := SocketPath(); !strings.HasPrefix(path, `\\.\pipe\mojian-`) {
		t.Fatalf("socket path: %q", path)
	}

	first := make(chan error, 1)
	go func() { first <- Serve() }()
	waitReady(t)

	second := make(chan error, 1)
	go func() { second <- Serve() }()
	select {
	case err := <-second:
		if err == nil || !strings.Contains(err.Error(), "already runs") {
			t.Fatalf("second server: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("second server did not return")
	}

	c, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Shutdown()
	_ = c.Close()
	select {
	case <-first:
	case <-time.After(3 * time.Second):
		t.Error("first server did not stop")
	}
}
