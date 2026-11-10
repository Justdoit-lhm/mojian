//go:build darwin || linux || windows

package project

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"mojian/internal/fs"
	"mojian/internal/session"
)

func newServerSaveStore(t *testing.T, body string) (*Store, *Note) {
	t.Helper()
	t.Setenv("MOJIAN_DIR", t.TempDir())
	done := make(chan error, 1)
	go func() { done <- session.Serve() }()
	var client *session.Client
	t.Cleanup(func() {
		if client != nil {
			_ = client.Shutdown()
			_ = client.Close()
		} else {
			_ = session.Stop()
		}
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("session server: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("session server did not stop")
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	for !session.Ready() {
		if time.Now().After(deadline) {
			t.Fatal("session server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var err error
	client, err = session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeSaveFile(t, filepath.Join(root, "note.txt"), body)
	s := New(fs.NewServer(client, root, t.TempDir(), true))
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	n := s.ByRelPath("note.txt")
	if n == nil {
		t.Fatal("the server did not list note.txt")
	}
	if n.Loaded {
		t.Fatal("the server listing should load notes lazily")
	}
	if err := s.LoadNote(n); err != nil {
		t.Fatal(err)
	}
	return s, n
}

func TestServerSaveRejectsExternalChange(t *testing.T) {
	s, n := newServerSaveStore(t, "original\r\n")
	n.Buffer().Set("local edit\n")
	writeSaveFile(t, n.Path, "external edit\r\n")
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAll(); !errors.Is(err, ErrFileChanged) {
		t.Fatalf("expected a server save conflict, got %v", err)
	}
	assertSaveFile(t, n.Path, "external edit\r\n")
	if !n.Dirty() || n.Saved != "original\r\n" || n.Text() != "local edit\n" {
		t.Fatal("server conflict lost the local edit or its on-disk baseline")
	}
}

func TestServerSavePreservesCRLF(t *testing.T) {
	s, n := newServerSaveStore(t, "original\r\n")
	n.Buffer().Set("first edit\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	assertSaveFile(t, n.Path, "first edit\r\n")
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	n.Buffer().Set("second edit\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	assertSaveFile(t, n.Path, "second edit\r\n")
	if n.Dirty() {
		t.Fatal("server save left the note dirty")
	}
}
