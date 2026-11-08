package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"mojian/internal/filekind"
	"mojian/internal/session"
)

// store is a project: the text files it holds, by path. Its backend is a
// local folder, a session server, or the browser's memory.
type store struct {
	dir string
	fs  fsBackend
	// entries and dirs are the last listing.
	entries []session.FileEntry
	dirs    map[string]bool
	notes   []*note
	byPath  map[string]*note
	byRel   map[string]*note
	seq     int
	tree    *fileNode
	assets  map[string]*assetFile
}

func newStore(fs fsBackend) *store {
	s := &store{
		dir: fs.Root(), fs: fs,
		byPath: map[string]*note{}, byRel: map[string]*note{},
		dirs: map[string]bool{"": true}, assets: map[string]*assetFile{},
	}
	return s
}

// defaultStoreDir is the folder of the app's own project: $MOJIAN_VAULT,
// else ~/Documents/Mojian.
func defaultStoreDir() (string, error) {
	if dir := os.Getenv("MOJIAN_VAULT"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents", "Mojian"), nil
}

// openDefaultStore opens the app's own project. In a browser there is no
// file system: the files live in memory for the session. A folder with no
// notes gets the welcome note; a project the user chose stays empty.
func (a *app) openDefaultStore() (*store, error) {
	dir, err := defaultStoreDir()
	if err != nil {
		return nil, err
	}
	s, err := a.openLocalStore(dir)
	if err != nil {
		return nil, err
	}
	if len(s.notes) == 0 {
		if _, err := s.newFileIn("", "Welcome.md", welcomeNote); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// openStore opens a project directly, without the session server. It is
// used by the browser and tests; macOS and Linux open projects
// through openLocalStore.
func openStore() (*store, error) {
	if runtime.GOOS == "js" {
		s := newStore(&memoryFS{})
		s.add(&note{Path: "Welcome.md", Title: "Welcome to Mojian", Saved: welcomeNote, Mod: time.Now(), loaded: true})
		s.add(&note{Path: "Reading notes.md", Title: "Reading notes", Saved: sampleNote, Mod: time.Now().Add(-time.Minute), loaded: true})
		s.buildTree()
		return s, nil
	}
	dir, err := defaultStoreDir()
	if err != nil {
		return nil, err
	}
	return openStoreAt(dir)
}

// openStoreAt opens dir as a project, creating the folder when it is
// missing. macOS and Linux open projects through the session server; the
// direct store now serves tests and the browser's memory fallback.
func openStoreAt(dir string) (*store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	s := newStore(&directFS{root: abs})
	if err := s.refresh(); err != nil {
		return nil, err
	}
	return s, nil
}

// refresh reads the project's listing. The direct backend also brings
// the text bodies, so it stays eager; a server project lists names and
// reads a file when it opens.
func (s *store) refresh() error {
	if s.isMemory() {
		s.buildTree()
		return nil
	}
	entries, bodies, err := s.fs.List()
	if err != nil {
		return err
	}
	s.entries = entries
	s.dirs = map[string]bool{"": true}
	seen := make(map[string]bool, len(entries))
	previousAssets := s.assets
	assets := map[string]*assetFile{}
	for _, e := range entries {
		rel := filepath.FromSlash(e.Rel)
		if e.Dir {
			s.dirs[rel] = true
			continue
		}
		switch e.Kind {
		case filekind.Text:
			seen[rel] = true
			if n := s.byRel[rel]; n != nil {
				n.Mod = e.Mod
				if body, ok := bodies[rel]; ok && !n.loaded {
					n.Saved = body
					n.Title = noteTitle(body, n.Path)
					n.loaded = true
				}
				continue
			}
			n := &note{
				Rel: rel, Path: s.absFor(rel),
				Title: noteTitle("", rel), Mod: e.Mod,
			}
			if body, ok := bodies[rel]; ok {
				n.Saved = body
				n.Title = noteTitle(body, n.Path)
				n.loaded = true
			}
			s.notes = append(s.notes, n)
			s.byRel[rel] = n
			s.byPath[n.Path] = n
		case filekind.Asset:
			f := previousAssets[rel]
			if f == nil || !f.Mod.Equal(e.Mod) || f.Size != e.Size || f.Hash != e.Hash {
				f = &assetFile{
					Rel: rel, Path: s.absFor(rel), Name: filepath.Base(rel),
					Size: e.Size, Mod: e.Mod, Hash: e.Hash,
				}
			}
			assets[rel] = f
		}
	}
	s.assets = assets
	// A file removed or moved outside the app leaves the store. Dirty
	// notes stay: their text is still in memory and may be saved again.
	kept := s.notes[:0]
	for _, n := range s.notes {
		if !seen[n.Rel] && !n.dirty() {
			delete(s.byPath, n.Path)
			delete(s.byRel, n.Rel)
			continue
		}
		kept = append(kept, n)
	}
	s.notes = kept
	sort.SliceStable(s.notes, func(i, j int) bool { return s.notes[i].Mod.After(s.notes[j].Mod) })
	s.buildTree()
	return nil
}

func (s *store) assetByRel(rel string) *assetFile {
	if s == nil {
		return nil
	}
	return s.assets[rel]
}

func (s *store) absFor(rel string) string {
	if s == nil || s.fs == nil {
		return rel
	}
	p, err := s.fs.Abs(rel)
	if err != nil {
		return filepath.Join(s.dir, rel)
	}
	return p
}

func (s *store) add(n *note) {
	if n.Path == "" {
		s.seq++
		n.Path = fmt.Sprintf("Untitled %d.md", s.seq)
	}
	if n.Rel == "" {
		n.Rel = relPath(s.dir, n.Path)
	}
	s.notes = append(s.notes, n)
	s.byPath[n.Path] = n
	s.byRel[n.Rel] = n
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

func (s *store) save(n *note) error {
	body := n.text()
	if body == n.Saved {
		return nil
	}
	if err := s.fs.Write(n.Rel, []byte(body)); err != nil {
		return err
	}
	n.Saved = body
	n.Title = noteTitle(body, n.Path)
	n.Mod = time.Now()
	if n.buf != nil {
		n.buf.MarkSaved()
	}
	return nil
}

// saveAll writes every note with unsaved edits.
func (s *store) saveAll() error {
	var first error
	for _, n := range s.notes {
		if !n.dirty() {
			continue
		}
		if err := s.save(n); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// sorted returns the notes, most recently touched first.
func (s *store) sorted() []*note {
	out := append([]*note(nil), s.notes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mod.Equal(out[j].Mod) {
			return out[i].Title < out[j].Title
		}
		return out[i].Mod.After(out[j].Mod)
	})
	return out
}

// byRelPath returns the note at a path within the vault, if it is in it.
func (s *store) byRelPath(rel string) *note {
	if n, ok := s.byRel[rel]; ok {
		return n
	}
	if n, ok := s.byPath[rel]; ok {
		return n
	}
	if !s.isMemory() && !filepath.IsAbs(rel) {
		return s.byPath[filepath.Join(s.dir, rel)]
	}
	return nil
}
