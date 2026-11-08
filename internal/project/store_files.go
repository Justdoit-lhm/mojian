package project

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"mojian/internal/fs"
)

// NewFileIn writes a text file with the given name and body in the folder
// rel, or at the root for an empty rel. It returns the loaded note.
func (s *Store) NewFileIn(rel, name, body string) (*Note, error) {
	if s.IsMemory() {
		return s.NewMemoryFile(name, body)
	}
	name = fileName(name, "untitled.txt")
	rel = s.UniqueRel(rel, name, false)
	if err := s.FS.Write(rel, []byte(body)); err != nil {
		return nil, err
	}
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	if n := s.ByRelPath(rel); n != nil {
		return n, nil
	}
	n := &Note{Rel: rel, Path: s.AbsFor(rel), Title: NoteTitle(body, rel), Saved: body, Mod: time.Now(), Loaded: true}
	s.Add(n)
	s.BuildTree()
	return n, nil
}

// NewMemoryFile adds a note to the browser's in-memory project.
func (s *Store) NewMemoryFile(name, body string) (*Note, error) {
	name = fileName(name, "untitled.txt")
	stem, ext := fs.SplitExt(name)
	path := name
	for i := 2; ; i++ {
		if _, ok := s.ByPath[path]; !ok {
			break
		}
		path = fmt.Sprintf("%s %d%s", stem, i, ext)
	}
	n := &Note{Path: path, Title: NoteTitle(body, path), Saved: body, Mod: time.Now(), Loaded: true}
	s.Add(n)
	s.BuildTree()
	return n, nil
}

// UniqueRel returns folder/name, numbered before its extension when the
// name is taken. copyMode names a duplicate "name copy".
func (s *Store) UniqueRel(folder, name string, copyMode bool) string {
	stem, ext := fs.SplitExt(name)
	candidate := name
	if copyMode {
		candidate = stem + " copy" + ext
	}
	rel := filepath.Join(folder, candidate)
	for i := 2; s.ExistsRel(rel); i++ {
		if copyMode {
			rel = filepath.Join(folder, fmt.Sprintf("%s copy %d%s", stem, i, ext))
		} else {
			rel = filepath.Join(folder, fmt.Sprintf("%s %d%s", stem, i, ext))
		}
	}
	return rel
}

// fileName keeps a typed name a single file name.
func fileName(name, fallback string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	if name == "" || name == "." || name == ".." {
		return fallback
	}
	return name
}
