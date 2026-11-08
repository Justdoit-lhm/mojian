package project

import (
	"fmt"

	"mojian/internal/session"
)

// LoadNote reads a note's text the first time it opens. A direct project
// is already loaded from its listing; a server project is read here.
func (s *Store) LoadNote(n *Note) error {
	if n == nil || n.Loaded {
		return nil
	}
	data, err := s.FS.Read(n.Rel)
	if err != nil {
		return err
	}
	n.Saved = string(data)
	n.Title = NoteTitle(n.Saved, n.Path)
	n.Loaded = true
	return nil
}

// MaterializeAsset fetches a remote picture or PDF into the local cache
// its viewer reads, and returns that file.
func (s *Store) MaterializeAsset(f *Asset) (string, error) {
	if f == nil {
		return "", fmt.Errorf("no file")
	}
	p, err := s.FS.Fetch(f.Rel, f.Size, f.Mod, f.Hash)
	if err != nil {
		return "", err
	}
	if s.IsRemote() {
		f.Local = p
	}
	return p, nil
}

// ExistsRel reports whether rel names a file or folder in the project.
func (s *Store) ExistsRel(rel string) bool {
	if rel == "" || rel == "." {
		return true
	}
	if s.IsMemory() {
		return s.ByRel[rel] != nil
	}
	if s.Dirs[rel] || s.ByRel[rel] != nil || s.Assets[rel] != nil {
		return true
	}
	_, err := s.FS.Stat(rel)
	return err == nil
}

// DirExists reports whether rel names a folder in the project.
func (s *Store) DirExists(rel string) bool {
	if s == nil || s.IsMemory() {
		return false
	}
	if rel == "" || rel == "." {
		return true
	}
	if s.IsServer() && s.Dirs[rel] {
		return true
	}
	e, err := s.FS.Stat(rel)
	return err == nil && e.Dir
}

// AbsRel resolves a path within the project.
func (s *Store) AbsRel(rel string) (string, error) {
	return s.FS.Abs(rel)
}

// RelOf reports a path's place in the project, if it is inside it.
func (s *Store) RelOf(p string) (string, bool) {
	return s.FS.RelOf(p)
}

// SetClient points a server-backed store at a new control client after
// the project reconnects.
func (s *Store) SetClient(client *session.Client) {
	if sf, ok := s.FS.(interface{ SetClient(*session.Client) }); ok {
		sf.SetClient(client)
	}
}
