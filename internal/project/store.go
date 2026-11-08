package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mojian/internal/filekind"
	"mojian/internal/fs"
	"mojian/internal/session"
)

// Store is a project: the text files it holds, by path. Its backend is a
// local folder, a session server, or the browser's memory.
type Store struct {
	Dir string
	FS  fs.Backend
	// Entries and Dirs are the last listing.
	Entries []session.FileEntry
	Dirs    map[string]bool
	Notes   []*Note
	ByPath  map[string]*Note
	ByRel   map[string]*Note
	Seq     int
	Tree    *TreeNode
	Assets  map[string]*Asset
}

// New returns a project over backend.
func New(backend fs.Backend) *Store {
	s := &Store{
		Dir: backend.Root(), FS: backend,
		ByPath: map[string]*Note{}, ByRel: map[string]*Note{},
		Dirs: map[string]bool{"": true}, Assets: map[string]*Asset{},
	}
	return s
}

func (s *Store) IsMemory() bool { return s.FS.Kind() == fs.KindMemory }

func (s *Store) IsServer() bool {
	k := s.FS.Kind()
	return k == fs.KindServerLocal || k == fs.KindServerRemote
}

func (s *Store) IsRemote() bool { return s.FS.Kind() == fs.KindServerRemote }

func (s *Store) Client() *session.Client { return s.FS.Client() }

// Refresh reads the project's listing. The direct backend also brings
// the text bodies, so it stays eager; a server project lists names and
// reads a file when it opens.
func (s *Store) Refresh() error {
	if s.IsMemory() {
		s.BuildTree()
		return nil
	}
	entries, bodies, err := s.FS.List()
	if err != nil {
		return err
	}
	s.Entries = entries
	s.Dirs = map[string]bool{"": true}
	seen := make(map[string]bool, len(entries))
	previousAssets := s.Assets
	assets := map[string]*Asset{}
	for _, e := range entries {
		rel := filepath.FromSlash(e.Rel)
		if e.Dir {
			s.Dirs[rel] = true
			continue
		}
		switch e.Kind {
		case filekind.Text:
			seen[rel] = true
			if n := s.ByRel[rel]; n != nil {
				n.Mod = e.Mod
				if body, ok := bodies[rel]; ok && !n.Loaded {
					n.Saved = body
					n.Title = NoteTitle(body, n.Path)
					n.Loaded = true
				}
				continue
			}
			n := &Note{
				Rel: rel, Path: s.AbsFor(rel),
				Title: NoteTitle("", rel), Mod: e.Mod,
			}
			if body, ok := bodies[rel]; ok {
				n.Saved = body
				n.Title = NoteTitle(body, n.Path)
				n.Loaded = true
			}
			s.Notes = append(s.Notes, n)
			s.ByRel[rel] = n
			s.ByPath[n.Path] = n
		case filekind.Asset:
			f := previousAssets[rel]
			if f == nil || !f.Mod.Equal(e.Mod) || f.Size != e.Size || f.Hash != e.Hash {
				f = &Asset{
					Rel: rel, Path: s.AbsFor(rel), Name: filepath.Base(rel),
					Size: e.Size, Mod: e.Mod, Hash: e.Hash,
				}
			}
			assets[rel] = f
		}
	}
	s.Assets = assets
	// A file removed or moved outside the app leaves the store. Dirty
	// notes stay: their text is still in memory and may be saved again.
	kept := s.Notes[:0]
	for _, n := range s.Notes {
		if !seen[n.Rel] && !n.Dirty() {
			delete(s.ByPath, n.Path)
			delete(s.ByRel, n.Rel)
			continue
		}
		kept = append(kept, n)
	}
	s.Notes = kept
	sort.SliceStable(s.Notes, func(i, j int) bool { return s.Notes[i].Mod.After(s.Notes[j].Mod) })
	s.BuildTree()
	return nil
}

// AssetByRel returns the viewer asset at rel, if it is in the project.
func (s *Store) AssetByRel(rel string) *Asset {
	if s == nil {
		return nil
	}
	return s.Assets[rel]
}

// AbsFor resolves a project path for a note that may not be loaded yet.
func (s *Store) AbsFor(rel string) string {
	if s == nil || s.FS == nil {
		return rel
	}
	p, err := s.FS.Abs(rel)
	if err != nil {
		return filepath.Join(s.Dir, rel)
	}
	return p
}

// Add puts n in the project's indexes, naming an untitled file.
func (s *Store) Add(n *Note) {
	if n.Path == "" {
		s.Seq++
		n.Path = fmt.Sprintf("Untitled %d.md", s.Seq)
	}
	if n.Rel == "" {
		n.Rel = relPath(s.Dir, n.Path)
	}
	s.Notes = append(s.Notes, n)
	s.ByPath[n.Path] = n
	s.ByRel[n.Rel] = n
}

// relPath is p within dir, for a layout the app can move with its vault.
func relPath(dir, p string) string {
	if dir == "" {
		return filepath.Base(p)
	}
	if rel, err := filepath.Rel(dir, p); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return filepath.Base(p)
}

// Save writes n if it changed.
func (s *Store) Save(n *Note) error {
	body := n.Text()
	if body == n.Saved {
		return nil
	}
	if err := s.FS.Write(n.Rel, []byte(body)); err != nil {
		return err
	}
	n.Saved = body
	n.Title = NoteTitle(body, n.Path)
	n.Mod = time.Now()
	if n.Buf != nil {
		n.Buf.MarkSaved()
	}
	return nil
}

// SaveAll writes every note with unsaved edits.
func (s *Store) SaveAll() error {
	var first error
	for _, n := range s.Notes {
		if !n.Dirty() {
			continue
		}
		if err := s.Save(n); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Sorted returns the notes, most recently touched first.
func (s *Store) Sorted() []*Note {
	out := append([]*Note(nil), s.Notes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mod.Equal(out[j].Mod) {
			return out[i].Title < out[j].Title
		}
		return out[i].Mod.After(out[j].Mod)
	})
	return out
}

// ByRelPath returns the note at a path within the vault, if it is in it.
func (s *Store) ByRelPath(rel string) *Note {
	if n, ok := s.ByRel[rel]; ok {
		return n
	}
	if n, ok := s.ByPath[rel]; ok {
		return n
	}
	if !s.IsMemory() && !filepath.IsAbs(rel) {
		return s.ByPath[filepath.Join(s.Dir, rel)]
	}
	return nil
}
