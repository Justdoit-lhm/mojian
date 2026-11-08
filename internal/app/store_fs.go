package app

import (
	"fmt"

	"mojian/internal/session"
)

// loadNote reads a note's text the first time it opens. A direct project
// is already loaded from its listing; a server project is read here.
func (s *store) loadNote(n *note) error {
	if n == nil || n.loaded {
		return nil
	}
	data, err := s.fs.Read(n.Rel)
	if err != nil {
		return err
	}
	n.Saved = string(data)
	n.Title = noteTitle(n.Saved, n.Path)
	n.loaded = true
	return nil
}

// materializeAsset fetches a remote picture or PDF into the local cache
// its viewer reads, and returns that file.
func (s *store) materializeAsset(f *assetFile) (string, error) {
	if f == nil {
		return "", fmt.Errorf("no file")
	}
	p, err := s.fs.Fetch(f.Rel, f.Size, f.Mod, f.Hash)
	if err != nil {
		return "", err
	}
	if s.isRemote() {
		f.local = p
	}
	return p, nil
}

// existsRel reports whether rel names a file or folder in the project.
func (s *store) existsRel(rel string) bool {
	if rel == "" || rel == "." {
		return true
	}
	if s.isMemory() {
		return s.byRel[rel] != nil
	}
	if s.dirs[rel] || s.byRel[rel] != nil || s.assets[rel] != nil {
		return true
	}
	_, err := s.fs.Stat(rel)
	return err == nil
}

// dirExists reports whether rel names a folder in the project.
func (s *store) dirExists(rel string) bool {
	if s == nil || s.isMemory() {
		return false
	}
	if rel == "" || rel == "." {
		return true
	}
	if s.isServer() && s.dirs[rel] {
		return true
	}
	e, err := s.fs.Stat(rel)
	return err == nil && e.Dir
}

// absRel resolves a path within the project.
func (s *store) absRel(rel string) (string, error) {
	return s.fs.Abs(rel)
}

// relOf reports a path's place in the project, if it is inside it.
func (s *store) relOf(p string) (string, bool) {
	return s.fs.RelOf(p)
}

// setClient points a server-backed store at a new control client after
// the project reconnects.
func (s *store) setClient(client *session.Client) {
	if sf, ok := s.fs.(*serverFS); ok {
		sf.client = client
	}
}
