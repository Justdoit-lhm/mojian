package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mojian/internal/fs"
)

func newSaveStore(t *testing.T, body string) (*Store, *Note) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(fs.NewDirect(root))
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	n := s.ByRelPath("note.txt")
	if n == nil {
		t.Fatal("the fixture note was not listed")
	}
	return s, n
}

func writeSaveFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertSaveFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("file = %q, want %q", data, want)
	}
}

func TestSaveRejectsExternalChange(t *testing.T) {
	s, n := newSaveStore(t, "original\n")
	n.Buffer().Set("local edit\n")
	writeSaveFile(t, n.Path, "external edit\n")
	err := s.SaveAll()
	if !errors.Is(err, ErrFileChanged) || !strings.Contains(err.Error(), "note.txt") {
		t.Fatalf("expected a conflict identifying the file, got %v", err)
	}
	assertSaveFile(t, n.Path, "external edit\n")
	if !n.Dirty() || n.Text() != "local edit\n" || n.Saved != "original\n" {
		t.Fatalf("conflict lost the local edit or its baseline: %+v", n)
	}
	// Restoring the baseline lets a later explicit save succeed.
	writeSaveFile(t, n.Path, "original\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	assertSaveFile(t, n.Path, "local edit\n")
	if n.Dirty() {
		t.Fatal("successful retry left the note dirty")
	}
}

func TestRefreshKeepsDirtySaveBaseline(t *testing.T) {
	s, n := newSaveStore(t, "original\n")
	n.Buffer().Set("local edit\n")
	info, err := os.Stat(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	// Same-length content and unchanged mtime still constitute a conflict.
	writeSaveFile(t, n.Path, "external\n")
	if err := os.Chtimes(n.Path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	if n.Saved != "original\n" || n.Text() != "local edit\n" {
		t.Fatal("refresh replaced a dirty buffer or its saved baseline")
	}
	if err := s.SaveAll(); err == nil {
		t.Fatal("refresh allowed overwriting an external edit")
	}
	assertSaveFile(t, n.Path, "external\n")
}

func TestRefreshReloadsCleanSharedBuffer(t *testing.T) {
	s, n := newSaveStore(t, "original\n")
	buf := n.Buffer()
	writeSaveFile(t, n.Path, "external\r\n")
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	if s.ByRelPath(n.Rel) != n || n.Buf != buf {
		t.Fatal("refresh replaced the shared note or buffer")
	}
	if n.Text() != "external\n" || n.Saved != "external\r\n" || n.Dirty() {
		t.Fatalf("clean note was not reloaded: text=%q saved=%q dirty=%v", n.Text(), n.Saved, n.Dirty())
	}
	if buf.LineEnding() != "\r\n" {
		t.Fatal("reload lost the external file's line ending")
	}
	buf.Set("local edit\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	assertSaveFile(t, n.Path, "local edit\r\n")
}

func TestSavePreservesCRLFAcrossRepeatedEdits(t *testing.T) {
	s, n := newSaveStore(t, "first\r\nsecond\r\n")
	buf := n.Buffer()
	buf.Set("edited first\nsecond\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	assertSaveFile(t, n.Path, "edited first\r\nsecond\r\n")
	if n.Saved != "edited first\r\nsecond\r\n" || n.Dirty() {
		t.Fatal("save did not update the on-disk baseline")
	}
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	buf.Set("edited twice\nsecond\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	assertSaveFile(t, n.Path, "edited twice\r\nsecond\r\n")
	if n.Dirty() {
		t.Fatal("second save left the note dirty")
	}
}

func TestSavePreservesLFAndAcceptsUnchangedContent(t *testing.T) {
	s, n := newSaveStore(t, "first\nsecond\n")
	n.Buffer().Set("edited\nsecond\n")
	// A rewrite of identical bytes must not count as a conflict.
	writeSaveFile(t, n.Path, "first\nsecond\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	assertSaveFile(t, n.Path, "edited\nsecond\n")
	if n.Dirty() {
		t.Fatal("LF note stayed dirty")
	}
}

func TestSaveDoesNotRecreateExternallyDeletedFile(t *testing.T) {
	s, n := newSaveStore(t, "original\n")
	n.Buffer().Set("local edit\n")
	if err := os.Remove(n.Path); err != nil {
		t.Fatal(err)
	}
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAll(); err == nil {
		t.Fatal("save silently recreated an externally deleted file")
	}
	if _, err := os.Stat(n.Path); !os.IsNotExist(err) {
		t.Fatalf("deleted file was recreated: %v", err)
	}
	if s.ByRelPath(n.Rel) != n || !n.Dirty() || n.Text() != "local edit\n" {
		t.Fatal("the deleted file's unsaved edit was lost")
	}
}

func TestMemorySaveKeepsWorking(t *testing.T) {
	s := New(fs.NewMemory())
	n, err := s.NewMemoryFile("note.txt", "original\r\n")
	if err != nil {
		t.Fatal(err)
	}
	n.Buffer().Set("edited\n")
	if err := s.SaveAll(); err != nil {
		t.Fatal(err)
	}
	if n.Dirty() {
		t.Fatal("memory save left the note dirty")
	}
}
