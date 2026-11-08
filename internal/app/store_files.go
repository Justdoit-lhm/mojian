package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// newFileIn writes a text file with the given name and body in the folder
// rel, or at the root for an empty rel. It returns the loaded note.
func (s *store) newFileIn(rel, name, body string) (*note, error) {
	if s.isMemory() {
		return s.newMemoryFile(name, body)
	}
	name = fileName(name, "untitled.txt")
	rel = s.uniqueRel(rel, name, false)
	if err := s.fs.Write(rel, []byte(body)); err != nil {
		return nil, err
	}
	if err := s.refresh(); err != nil {
		return nil, err
	}
	if n := s.byRelPath(rel); n != nil {
		return n, nil
	}
	n := &note{Rel: rel, Path: s.absFor(rel), Title: noteTitle(body, rel), Saved: body, Mod: time.Now(), loaded: true}
	s.add(n)
	s.buildTree()
	return n, nil
}

func (s *store) newMemoryFile(name, body string) (*note, error) {
	name = fileName(name, "untitled.txt")
	stem, ext := splitExt(name)
	path := name
	for i := 2; ; i++ {
		if _, ok := s.byPath[path]; !ok {
			break
		}
		path = fmt.Sprintf("%s %d%s", stem, i, ext)
	}
	n := &note{Path: path, Title: noteTitle(body, path), Saved: body, Mod: time.Now(), loaded: true}
	s.add(n)
	s.buildTree()
	return n, nil
}

// uniqueRel returns folder/name, numbered before its extension when the
// name is taken. copyMode names a duplicate "name copy".
// uniqueRel returns folder/name, numbered before its extension when the
// name is taken. copyMode names a duplicate "name copy".
func (s *store) uniqueRel(folder, name string, copyMode bool) string {
	stem, ext := splitExt(name)
	candidate := name
	if copyMode {
		candidate = stem + " copy" + ext
	}
	rel := filepath.Join(folder, candidate)
	for i := 2; s.existsRel(rel); i++ {
		if copyMode {
			rel = filepath.Join(folder, fmt.Sprintf("%s copy %d%s", stem, i, ext))
		} else {
			rel = filepath.Join(folder, fmt.Sprintf("%s %d%s", stem, i, ext))
		}
	}
	return rel
}

// splitExt separates a file name's extension for a numbered sibling.
// splitExt separates a file name's extension for a numbered sibling.
func splitExt(name string) (stem, ext string) {
	ext = filepath.Ext(name)
	stem = strings.TrimSuffix(name, ext)
	if stem == "" {
		stem = "untitled"
	}
	return stem, ext
}

// fileName keeps a typed name a single file name.
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

// save writes the note if it changed, atomically, so that a crash leaves
// the old text or the new, never half of each.
