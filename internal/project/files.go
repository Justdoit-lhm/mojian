package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui/editor"
)

// Reindex rebuilds the path indexes after notes moved.
func (s *Store) Reindex() {
	s.ByPath = make(map[string]*Note, len(s.Notes))
	s.ByRel = make(map[string]*Note, len(s.Notes))
	for _, n := range s.Notes {
		s.ByPath[n.Path] = n
		s.ByRel[n.Rel] = n
	}
}

// Forget drops the notes under rel from memory, without touching disk.
func (s *Store) Forget(rel string) {
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" {
		return
	}
	kept := s.Notes[:0]
	for _, n := range s.Notes {
		if n.Rel == rel || strings.HasPrefix(n.Rel, rel+string(filepath.Separator)) {
			delete(s.ByPath, n.Path)
			delete(s.ByRel, n.Rel)
			continue
		}
		kept = append(kept, n)
	}
	s.Notes = kept
	for assetRel := range s.Assets {
		if assetRel == rel || strings.HasPrefix(assetRel, rel+string(filepath.Separator)) {
			delete(s.Assets, assetRel)
		}
	}
	for d := range s.Dirs {
		if d == rel || strings.HasPrefix(d, rel+string(filepath.Separator)) {
			delete(s.Dirs, d)
		}
	}
	s.BuildTree()
}

// Remap moves the notes under oldRel to newRel after a rename or move.
func (s *Store) Remap(oldRel, newRel string, dir bool) {
	oldRel = filepath.Clean(oldRel)
	prefix := oldRel + string(filepath.Separator)
	for _, n := range s.Notes {
		switch {
		case n.Rel == oldRel:
		case dir && strings.HasPrefix(n.Rel, prefix):
		default:
			continue
		}
		rest := strings.TrimPrefix(n.Rel, oldRel)
		n.Rel = filepath.Join(newRel, rest)
		n.Path = s.AbsFor(n.Rel)
		n.Title = NoteTitle(n.Text(), n.Path)
		if n.Buf != nil {
			n.Buf.SetLanguage(editor.LanguageFor(n.Path))
		}
	}
	for assetRel, f := range s.Assets {
		switch {
		case assetRel == oldRel:
		case dir && strings.HasPrefix(assetRel, prefix):
		default:
			continue
		}
		rest := strings.TrimPrefix(assetRel, oldRel)
		f.Rel = filepath.Join(newRel, rest)
		f.Path = s.AbsFor(f.Rel)
		f.Name = filepath.Base(f.Path)
		delete(s.Assets, assetRel)
		s.Assets[f.Rel] = f
	}
	s.Reindex()
	s.BuildTree()
}

// Rename changes a file or folder's name in its own folder. It returns
// the new path and the notes whose path changed.
func (s *Store) Rename(oldRel, name string) (string, []*Note, error) {
	if oldRel == "" {
		return "", nil, errors.New("the project itself cannot be renamed")
	}
	oldRel = filepath.Clean(oldRel)
	base := filepath.Base(oldRel)
	newName := fileName(name, base)
	if newName == base {
		return oldRel, nil, nil
	}
	newRel := filepath.Join(filepath.Dir(oldRel), newName)
	if filepath.Dir(oldRel) == "." {
		newRel = newName
	}
	if !s.FS.SameFile(oldRel, newRel) && s.ExistsRel(newRel) {
		return "", nil, fmt.Errorf("%s already exists", newName)
	}
	isDir := s.Dirs[oldRel]
	if !isDir {
		if e, err := s.FS.Stat(oldRel); err == nil {
			isDir = e.Dir
		}
	}
	if err := s.FS.Rename(oldRel, newRel); err != nil {
		return "", nil, err
	}
	changed := s.NotesUnder(oldRel, isDir)
	s.Remap(oldRel, newRel, isDir)
	if err := s.Refresh(); err != nil {
		return newRel, changed, err
	}
	return newRel, changed, nil
}

// NotesUnder returns the notes at rel or, for a folder, below it.
func (s *Store) NotesUnder(rel string, dir bool) []*Note {
	rel = filepath.Clean(rel)
	prefix := rel + string(filepath.Separator)
	var out []*Note
	for _, n := range s.Notes {
		if n.Rel == rel || dir && strings.HasPrefix(n.Rel, prefix) {
			out = append(out, n)
		}
	}
	return out
}

// CopyInto copies a file or folder into a folder of the project, and
// returns its new path within the project.
func (s *Store) CopyInto(srcAbs, destDirRel string) (string, error) {
	destDir, err := s.AbsRel(destDirRel)
	if err != nil {
		return "", err
	}
	srcAbs = filepath.Clean(srcAbs)
	if _, err := os.Lstat(srcAbs); err != nil {
		return "", err
	}
	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	if inside(srcAbs, destDir) {
		return "", errors.New("a folder cannot be copied into itself")
	}
	destRel := s.UniqueRel(destDirRel, filepath.Base(srcAbs), true)
	if err := s.FS.CopyFrom(srcAbs, destRel); err != nil {
		return "", err
	}
	if err := s.Refresh(); err != nil {
		return destRel, err
	}
	return destRel, nil
}

// MoveInto moves a file or folder into a folder of the project. Moves
// within the project update the open notes in place; a move from outside
// is a copy and a removal.
func (s *Store) MoveInto(srcAbs, destDirRel string) (string, error) {
	destDir, err := s.AbsRel(destDirRel)
	if err != nil {
		return "", err
	}
	srcAbs = filepath.Clean(srcAbs)
	if _, err := os.Lstat(srcAbs); err != nil {
		return "", err
	}
	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	if inside(srcAbs, destDir) {
		return "", errors.New("a folder cannot be moved into itself")
	}
	if oldRel, insideProject := s.RelOf(srcAbs); insideProject {
		if filepath.Dir(srcAbs) == destDir {
			return oldRel, nil
		}
		e, err := s.FS.Stat(oldRel)
		if err != nil {
			return "", err
		}
		newRel := s.UniqueRel(destDirRel, filepath.Base(srcAbs), false)
		if err := s.FS.MoveFrom(srcAbs, newRel); err != nil {
			return "", err
		}
		s.Remap(oldRel, newRel, e.Dir)
		if err := s.Refresh(); err != nil {
			return newRel, err
		}
		return newRel, nil
	}
	destRel := s.UniqueRel(destDirRel, filepath.Base(srcAbs), false)
	if err := s.FS.MoveFrom(srcAbs, destRel); err != nil {
		return "", err
	}
	if err := s.Refresh(); err != nil {
		return destRel, err
	}
	return destRel, nil
}

// MoveRel moves rel into dest, keeping its name when free.
func (s *Store) MoveRel(rel, dest string) (string, error) {
	e, err := s.FS.Stat(rel)
	if err != nil {
		return "", err
	}
	newRel := s.UniqueRel(dest, filepath.Base(rel), false)
	if err := s.FS.Rename(rel, newRel); err != nil {
		return "", err
	}
	s.Remap(rel, newRel, e.Dir)
	if err := s.Refresh(); err != nil {
		return newRel, err
	}
	return newRel, nil
}

// DuplicateRel copies rel beside it, named "name copy".
func (s *Store) DuplicateRel(rel, dest string) (string, error) {
	newRel := s.UniqueRel(dest, filepath.Base(rel), true)
	if err := s.FS.Copy(rel, newRel); err != nil {
		return "", err
	}
	if err := s.Refresh(); err != nil {
		return newRel, err
	}
	return newRel, nil
}

// inside reports whether p is dir or below it.
func inside(dir, p string) bool {
	dir, p = filepath.Clean(dir), filepath.Clean(p)
	return p == dir || strings.HasPrefix(p, dir+string(filepath.Separator))
}
