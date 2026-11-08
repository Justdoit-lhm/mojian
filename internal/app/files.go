package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func (s *store) reindex() {
	s.byPath = make(map[string]*note, len(s.notes))
	s.byRel = make(map[string]*note, len(s.notes))
	for _, n := range s.notes {
		s.byPath[n.Path] = n
		s.byRel[n.Rel] = n
	}
}

// forget drops the notes under rel from memory, without touching disk.
func (s *store) forget(rel string) {
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" {
		return
	}
	kept := s.notes[:0]
	for _, n := range s.notes {
		if n.Rel == rel || strings.HasPrefix(n.Rel, rel+string(filepath.Separator)) {
			delete(s.byPath, n.Path)
			delete(s.byRel, n.Rel)
			continue
		}
		kept = append(kept, n)
	}
	s.notes = kept
	for assetRel := range s.assets {
		if assetRel == rel || strings.HasPrefix(assetRel, rel+string(filepath.Separator)) {
			delete(s.assets, assetRel)
		}
	}
	for d := range s.dirs {
		if d == rel || strings.HasPrefix(d, rel+string(filepath.Separator)) {
			delete(s.dirs, d)
		}
	}
	s.buildTree()
}

// remap moves the notes under oldRel to newRel after a rename or move.
func (s *store) remap(oldRel, newRel string, dir bool) {
	oldRel = filepath.Clean(oldRel)
	prefix := oldRel + string(filepath.Separator)
	for _, n := range s.notes {
		switch {
		case n.Rel == oldRel:
		case dir && strings.HasPrefix(n.Rel, prefix):
		default:
			continue
		}
		rest := strings.TrimPrefix(n.Rel, oldRel)
		n.Rel = filepath.Join(newRel, rest)
		n.Path = s.absFor(n.Rel)
		n.Title = noteTitle(n.text(), n.Path)
		if n.buf != nil {
			n.buf.SetLanguage(editorLanguage(n.Path))
		}
	}
	for assetRel, f := range s.assets {
		switch {
		case assetRel == oldRel:
		case dir && strings.HasPrefix(assetRel, prefix):
		default:
			continue
		}
		rest := strings.TrimPrefix(assetRel, oldRel)
		f.Rel = filepath.Join(newRel, rest)
		f.Path = s.absFor(f.Rel)
		f.Name = filepath.Base(f.Path)
		delete(s.assets, assetRel)
		s.assets[f.Rel] = f
	}
	s.reindex()
	s.buildTree()
}

// rename changes a file or folder's name in its own folder. It returns
// the new path and the notes whose path changed.
func (s *store) rename(oldRel, name string) (string, []*note, error) {
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
	if !s.fs.SameFile(oldRel, newRel) && s.existsRel(newRel) {
		return "", nil, fmt.Errorf("%s already exists", newName)
	}
	isDir := s.dirs[oldRel]
	if !isDir {
		if e, err := s.fs.Stat(oldRel); err == nil {
			isDir = e.Dir
		}
	}
	if err := s.fs.Rename(oldRel, newRel); err != nil {
		return "", nil, err
	}
	changed := s.notesUnder(oldRel, isDir)
	s.remap(oldRel, newRel, isDir)
	if err := s.refresh(); err != nil {
		return newRel, changed, err
	}
	return newRel, changed, nil
}

func (s *store) notesUnder(rel string, dir bool) []*note {
	rel = filepath.Clean(rel)
	prefix := rel + string(filepath.Separator)
	var out []*note
	for _, n := range s.notes {
		if n.Rel == rel || dir && strings.HasPrefix(n.Rel, prefix) {
			out = append(out, n)
		}
	}
	return out
}

// copyInto copies a file or folder into a folder of the project, and
// returns its new path within the project.
func (s *store) copyInto(srcAbs, destDirRel string) (string, error) {
	destDir, err := s.absRel(destDirRel)
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
	destRel := s.uniqueRel(destDirRel, filepath.Base(srcAbs), true)
	if err := s.fs.CopyFrom(srcAbs, destRel); err != nil {
		return "", err
	}
	if err := s.refresh(); err != nil {
		return destRel, err
	}
	return destRel, nil
}

// moveInto moves a file or folder into a folder of the project. Moves
// within the project update the open notes in place; a move from outside
// is a copy and a removal.
func (s *store) moveInto(srcAbs, destDirRel string) (string, error) {
	destDir, err := s.absRel(destDirRel)
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
	if oldRel, insideProject := s.relOf(srcAbs); insideProject {
		if filepath.Dir(srcAbs) == destDir {
			return oldRel, nil
		}
		e, err := s.fs.Stat(oldRel)
		if err != nil {
			return "", err
		}
		newRel := s.uniqueRel(destDirRel, filepath.Base(srcAbs), false)
		if err := s.fs.MoveFrom(srcAbs, newRel); err != nil {
			return "", err
		}
		s.remap(oldRel, newRel, e.Dir)
		if err := s.refresh(); err != nil {
			return newRel, err
		}
		return newRel, nil
	}
	destRel := s.uniqueRel(destDirRel, filepath.Base(srcAbs), false)
	if err := s.fs.MoveFrom(srcAbs, destRel); err != nil {
		return "", err
	}
	if err := s.refresh(); err != nil {
		return destRel, err
	}
	return destRel, nil
}

// moveRel moves rel into dest, keeping its name when free.
func (s *store) moveRel(rel, dest string) (string, error) {
	e, err := s.fs.Stat(rel)
	if err != nil {
		return "", err
	}
	newRel := s.uniqueRel(dest, filepath.Base(rel), false)
	if err := s.fs.Rename(rel, newRel); err != nil {
		return "", err
	}
	s.remap(rel, newRel, e.Dir)
	if err := s.refresh(); err != nil {
		return newRel, err
	}
	return newRel, nil
}

// duplicateRel copies rel beside it, named "name copy".
func (s *store) duplicateRel(rel, dest string) (string, error) {
	newRel := s.uniqueRel(dest, filepath.Base(rel), true)
	if err := s.fs.Copy(rel, newRel); err != nil {
		return "", err
	}
	if err := s.refresh(); err != nil {
		return newRel, err
	}
	return newRel, nil
}

// inside reports whether p is dir or below it.
func inside(dir, p string) bool {
	dir, p = filepath.Clean(dir), filepath.Clean(p)
	return p == dir || strings.HasPrefix(p, dir+string(filepath.Separator))
}

// copyPath copies a file, a symlink or a folder.
func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}

// editorLanguage is the highlighter for a path, or nil when the editor
// shows it as plain text.
func editorLanguage(path string) *editor.Language {
	return editor.LanguageFor(path)
}

func (a *app) selectedDir() string {
	if a.store == nil || a.treeSel == "" {
		return ""
	}
	if a.store.dirExists(a.treeSel) {
		return a.treeSel
	}
	dir := filepath.Dir(a.treeSel)
	if dir == "." {
		return ""
	}
	if a.store.dirExists(dir) {
		return dir
	}
	return ""
}

// destinationFor returns the folder a paste into rel goes to.
func (a *app) destinationFor(rel string) string {
	if rel == "" {
		return ""
	}
	if a.store.dirExists(rel) {
		return rel
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return ""
	}
	return dir
}

func (a *app) duplicate(rel string) {
	if rel == "" || a.store == nil || a.store.isMemory() {
		return
	}
	newRel, err := a.store.duplicateRel(rel, a.destinationFor(rel))
	if err != nil {
		a.fail(err)
		return
	}
	a.selectRel(newRel)
}

func (a *app) reveal(rel string) {
	if a.store == nil || runtime.GOOS == "js" {
		return
	}
	if a.store.isRemote() {
		a.fail(errors.New("remote projects have no Finder window"))
		return
	}
	abs, err := a.store.absRel(rel)
	if err != nil {
		a.fail(err)
		return
	}
	mygo.Shell.ShowItemInFolder(abs)
}

// trashPath moves a path to the system trash; tests replace it.
var trashPath = func(path string) error { return mygo.Shell.TrashItem(path) }

func (a *app) renameSelection() { a.startRenameNode(a.treeSel) }

func (a *app) trashSelection() { a.trash(a.treeSel) }

func (a *app) trash(rel string) {
	if rel == "" || a.store == nil {
		return
	}
	if runtime.GOOS == "js" || a.store.isMemory() {
		a.fail(errors.New("trash needs the desktop app"))
		return
	}
	if err := a.store.fs.Trash(rel); err != nil {
		a.fail(err)
		return
	}
	a.closeNotes(func(n *note) bool {
		return n.Rel == rel || strings.HasPrefix(n.Rel, rel+string(filepath.Separator))
	})
	a.closeAssets(func(f *assetFile) bool {
		return f.Rel == rel || strings.HasPrefix(f.Rel, rel+string(filepath.Separator))
	})
	a.store.forget(rel)
	if a.store.isServer() {
		if err := a.store.refresh(); err != nil {
			a.fail(err)
			return
		}
	}
	a.treeSel = a.destinationFor(rel)
	a.openPath(a.treeSel)
	a.treeReveal = a.treeSel
	a.changed()
	a.invalidate()
}

// closeNotes closes every pane whose note matches, then drops empty tabs.
func (a *app) closeNotes(match func(*note) bool) {
	a.closePanesWhere(func(p *pane) bool { return p.Note != nil && match(p.Note) })
}

// closeAssets closes every pane whose asset matches.
func (a *app) closeAssets(match func(*assetFile) bool) {
	a.closePanesWhere(func(p *pane) bool { return p.Asset != nil && match(p.Asset) })
}

func (a *app) closePanesWhere(match func(*pane) bool) {
	activeTab := a.tab()
	activeIndex := a.active
	for _, t := range a.tabs {
		for _, p := range append([]*pane(nil), t.panes()...) {
			if match(p) {
				releaseAsset(p)
				t.closePane(p)
				if len(t.panes()) == 0 {
					break
				}
			}
		}
	}
	kept := make([]*tab, 0, len(a.tabs))
	for _, t := range a.tabs {
		if len(t.panes()) > 0 {
			kept = append(kept, t)
		}
	}
	a.tabs = kept
	if len(a.tabs) == 0 {
		a.active = -1
		a.focusReq = nil
		return
	}
	if activeTab != nil {
		for i, t := range a.tabs {
			if t == activeTab {
				a.active = i
				a.focusReq = t.editorPane()
				return
			}
		}
	}
	a.active = min(max(activeIndex, 0), len(a.tabs)-1)
	a.focusReq = a.tabs[a.active].editorPane()
}

// selectRel selects a tree row and reveals it.
func (a *app) selectRel(rel string) {
	a.treeSel = rel
	a.treeReveal = rel
	a.openPath(filepath.Dir(rel))
	a.invalidate()
}

// nodeMenu fills a folder or file's context menu.
func (a *app) nodeMenu(m *ui.Menu, rel string, dir bool) {
	a.treeSel = rel
	if dir {
		if m.Item("New File…").Chosen() {
			a.later(func() { a.startNewFileIn(rel) })
		}
		if m.Item("New Folder…").Chosen() {
			a.later(func() { a.startTreeDraft(draftFolder, rel, "", "") })
		}
		m.Separator()
		if m.Item("Paste").Chosen() {
			a.later(func() { a.pasteInto(rel) })
		}
	} else {
		if m.Item("Open in Current Window").Chosen() {
			a.later(func() { a.openRel(rel) })
		}
		if m.Item("Open in New Tab").Chosen() {
			a.later(func() {
				if n := a.store.byRelPath(rel); n != nil {
					a.openNoteInTab(n)
				} else if f := a.store.assetByRel(rel); f != nil {
					a.openAssetInTab(f)
				}
			})
		}
	}
	if rel == "" {
		m.Separator()
		if m.Item("Reveal Project").Chosen() {
			a.later(func() { a.reveal(rel) })
		}
		return
	}
	m.Separator()
	if m.Item("Rename…").Shortcut(0, ui.KeyF2).Chosen() {
		a.later(func() { a.startRenameNode(rel) })
	}
	if m.Item("Duplicate").Chosen() {
		a.later(func() { a.duplicate(rel) })
	}
	m.Separator()
	if m.Item("Copy").Chosen() {
		a.later(func() { a.copySelectionTo(rel, false) })
	}
	if m.Item("Cut").Chosen() {
		a.later(func() { a.copySelectionTo(rel, true) })
	}
	m.Separator()
	if m.Item("Copy Relative Path").Chosen() {
		a.later(func() { a.copyPath(rel, false) })
	}
	if m.Item("Copy Path").Chosen() {
		a.later(func() { a.copyPath(rel, true) })
	}
	if runtime.GOOS != "js" && m.Item(revealLabel()).Chosen() {
		a.later(func() { a.reveal(rel) })
	}
	m.Separator()
	if m.Item("Move to Trash").Shortcut(ui.Cmd, ui.KeyBackspace).Chosen() {
		a.later(func() { a.trash(rel) })
	}
}
