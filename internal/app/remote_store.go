package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"mojian/internal/filekind"
	"mojian/internal/session"
)

// remoteFS is a project's files on an SSH host, reached through the
// session server. The app reads and writes them directly: there is no
// local copy of the project.
type remoteFS struct {
	client *session.Client
	root   string
	cache  string // where pictures and PDFs are fetched for their viewers
}

// openRemoteStore lists the project on the server and returns it as the
// open project. File contents are read when a file is opened, not when
// the tree is.
func openRemoteStore(client *session.Client, root, cache string) (*store, error) {
	if client == nil || root == "" {
		return nil, fmt.Errorf("remote project needs a connection and a folder")
	}
	s := &store{
		dir:    root,
		remote: &remoteFS{client: client, root: root, cache: cache},
		byPath: map[string]*note{}, byRel: map[string]*note{},
		dirs: map[string]bool{}, assets: map[string]*assetFile{},
	}
	if err := s.refresh(); err != nil {
		return nil, err
	}
	return s, nil
}

// abs is the path of rel on the server.
func (r *remoteFS) abs(rel string) string {
	if rel == "" || rel == "." {
		return r.root
	}
	return path.Join(r.root, filepath.ToSlash(rel))
}

// refreshRemote rebuilds the project from the server's listing: names,
// sizes and times, without carrying the files' bodies.
func (s *store) refreshRemote() error {
	entries, err := s.remote.client.ListTree(s.remote.root)
	if err != nil {
		return fmt.Errorf("listing the remote project: %w", err)
	}
	s.entries = entries
	s.dirs = map[string]bool{"": true}
	seen := map[string]bool{}
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
				continue
			}
			n := &note{
				Rel: rel, Path: s.remote.abs(rel),
				Title: noteTitle("", rel), Mod: e.Mod,
			}
			s.notes = append(s.notes, n)
			s.byRel[rel] = n
			s.byPath[n.Path] = n
		case filekind.Asset:
			f := previousAssets[rel]
			if f == nil || !f.Mod.Equal(e.Mod) || f.Size != e.Size {
				f = &assetFile{
					Rel: rel, Path: s.remote.abs(rel), Name: filepath.Base(rel),
					Size: e.Size, Mod: e.Mod,
				}
			}
			assets[rel] = f
		}
	}
	s.assets = assets
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

// remoteTree builds the cached tree from the server's listing.
func (s *store) remoteTree() *fileNode {
	root := &fileNode{Name: filepath.Base(s.dir), Dir: true}
	nodes := map[string]*fileNode{"": root}
	addDir := func(rel string) *fileNode {
		parent := filepath.Dir(rel)
		if parent == "." {
			parent = ""
		}
		p := nodes[parent]
		if p == nil {
			return nil
		}
		n := &fileNode{Name: filepath.Base(rel), Rel: rel, Dir: true}
		p.Children = append(p.Children, n)
		nodes[rel] = n
		return n
	}
	// Entries arrive in walk order, so a folder's parent is known.
	for _, e := range s.entries {
		rel := filepath.FromSlash(e.Rel)
		if e.Dir {
			addDir(rel)
			continue
		}
		parent := filepath.Dir(rel)
		if parent == "." {
			parent = ""
		}
		p := nodes[parent]
		if p == nil {
			continue
		}
		n := &fileNode{Name: filepath.Base(rel), Rel: rel}
		if note := s.byRel[rel]; note != nil {
			n.Note = note
		} else if asset := s.assets[rel]; asset != nil {
			n.Asset = asset
		} else {
			continue
		}
		p.Children = append(p.Children, n)
	}
	var sortRec func(*fileNode)
	sortRec = func(n *fileNode) {
		var folders, files []*fileNode
		for _, c := range n.Children {
			if c.Dir {
				folders = append(folders, c)
			} else {
				files = append(files, c)
			}
		}
		sortNodes(folders)
		sortNodes(files)
		n.Children = append(folders, files...)
		for _, c := range n.Children {
			sortRec(c)
		}
	}
	sortRec(root)
	return root
}

// loadNote reads a remote note's text the first time it opens. Local
// notes are already loaded.
func (s *store) loadNote(n *note) error {
	if n == nil || s.remote == nil || n.loaded {
		return nil
	}
	data, err := s.remote.client.ReadFile(s.remote.abs(n.Rel))
	if err != nil {
		return err
	}
	n.Saved = string(data)
	n.Title = noteTitle(n.Saved, n.Path)
	n.loaded = true
	return nil
}

// writeRel writes body to rel in the project.
func (s *store) writeRel(rel, body string) error {
	if s.memory {
		return nil
	}
	if s.remote != nil {
		return s.remote.client.WriteFile(s.remote.abs(rel), []byte(body))
	}
	p, err := s.absRel(rel)
	if err != nil {
		return err
	}
	return writeFileAtomic(p, []byte(body))
}

// existsRel reports whether rel names a file or folder in the project.
func (s *store) existsRel(rel string) bool {
	if rel == "" || rel == "." {
		return true
	}
	if s.remote == nil {
		p, err := s.absRel(rel)
		if err != nil {
			return false
		}
		_, err = os.Stat(p)
		return err == nil
	}
	if s.dirs[rel] || s.byRel[rel] != nil || s.assets[rel] != nil {
		return true
	}
	_, err := s.remote.client.Stat(s.remote.abs(rel))
	return err == nil
}

// mkdirRel makes a folder in the project.
func (s *store) mkdirRel(rel string) error {
	if s.remote != nil {
		return s.remote.client.Mkdir(s.remote.abs(rel))
	}
	p, err := s.absRel(rel)
	if err != nil {
		return err
	}
	return os.MkdirAll(p, 0o755)
}

// renameRel renames a file or folder within the project.
func (s *store) renameRel(oldRel, newRel string) error {
	if s.remote != nil {
		return s.remote.client.Rename(s.remote.abs(oldRel), s.remote.abs(newRel))
	}
	oldAbs, err := s.absRel(oldRel)
	if err != nil {
		return err
	}
	newAbs, err := s.absRel(newRel)
	if err != nil {
		return err
	}
	return os.Rename(oldAbs, newAbs)
}

// copyRel copies a file or folder within the project.
func (s *store) copyRel(oldRel, newRel string) error {
	if s.dirs[oldRel] && (newRel == oldRel || strings.HasPrefix(newRel, oldRel+string(filepath.Separator))) {
		return fmt.Errorf("a folder cannot be copied into itself")
	}
	if s.remote != nil {
		return s.remote.client.Copy(s.remote.abs(oldRel), s.remote.abs(newRel))
	}
	oldAbs, err := s.absRel(oldRel)
	if err != nil {
		return err
	}
	newAbs, err := s.absRel(newRel)
	if err != nil {
		return err
	}
	return copyPath(oldAbs, newAbs)
}

// movedEntry follows a remote rename in the cached listing and the open
// notes, so no second listing is needed.
func (s *store) movedEntry(oldRel, newRel string, dir bool) {
	oldRel = filepath.Clean(oldRel)
	prefix := oldRel + string(filepath.Separator)
	for i := range s.entries {
		rel := filepath.FromSlash(s.entries[i].Rel)
		switch {
		case rel == oldRel:
		case dir && strings.HasPrefix(rel, prefix):
		default:
			continue
		}
		rel = filepath.Join(newRel, strings.TrimPrefix(rel, oldRel))
		s.entries[i].Rel = filepath.ToSlash(rel)
	}
	if dir {
		for d := range s.dirs {
			switch {
			case d == oldRel:
			case strings.HasPrefix(d, prefix):
			default:
				continue
			}
			delete(s.dirs, d)
			s.dirs[filepath.Join(newRel, strings.TrimPrefix(d, oldRel))] = true
		}
	}
	s.remap(oldRel, newRel, dir)
}

// duplicateRel copies rel beside it, named "name copy".
func (s *store) duplicateRel(rel, dest string) (string, error) {
	newRel := uniqueCopyRel(s, dest, filepath.Base(rel))
	if err := s.copyRel(rel, newRel); err != nil {
		return "", err
	}
	return newRel, nil
}

// moveRel moves rel into dest, keeping its name when free.
func (s *store) moveRel(rel, dest string) (string, error) {
	if s.dirs[rel] && (dest == rel || strings.HasPrefix(dest, rel+string(filepath.Separator))) {
		return "", fmt.Errorf("a folder cannot be moved into itself")
	}
	newRel := uniqueCopyRel(s, dest, filepath.Base(rel))
	isDir := s.dirs[rel]
	if err := s.renameRel(rel, newRel); err != nil {
		return "", err
	}
	s.movedEntry(rel, newRel, isDir)
	return newRel, nil
}

// uniqueCopyRel names a copy "name copy", then "name copy 2".
func uniqueCopyRel(s *store, dir, name string) string {
	stem, ext := splitExt(name)
	candidate := stem + " copy" + ext
	for i := 2; ; i++ {
		rel := filepath.Join(dir, candidate)
		if !s.existsRel(rel) {
			return rel
		}
		candidate = fmt.Sprintf("%s copy %d%s", stem, i, ext)
	}
}

// materializeAsset fetches a remote picture or PDF into the local cache
// its viewer reads, and returns that file.
func (s *store) materializeAsset(f *assetFile) (string, error) {
	if f == nil {
		return "", fmt.Errorf("no file")
	}
	if s.remote == nil {
		return f.Path, nil
	}
	if f.local != "" {
		if info, err := os.Stat(f.local); err == nil && info.Size() == f.Size {
			return f.local, nil
		}
	}
	data, err := s.remote.client.ReadFile(s.remote.abs(f.Rel))
	if err != nil {
		return "", err
	}
	dir := filepath.Join(s.remote.cache, fmt.Sprintf("%d-%d", f.Mod.Unix(), f.Size))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	local := filepath.Join(dir, shortHash(f.Rel)+strings.ToLower(filepath.Ext(f.Rel)))
	if err := os.WriteFile(local, data, 0o644); err != nil {
		return "", err
	}
	f.local = local
	return local, nil
}
