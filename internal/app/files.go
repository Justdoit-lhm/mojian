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

// absRel resolves a path within the project, refusing anything outside it.
func (s *store) absRel(rel string) (string, error) {
	if s.memory {
		return "", errors.New("files need the desktop app")
	}
	if rel == "" || rel == "." {
		return s.dir, nil
	}
	clean := filepath.Clean(rel)
	if filepath.IsAbs(clean) || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path is outside the project")
	}
	p := filepath.Join(s.dir, clean)
	r, err := filepath.Rel(s.dir, p)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("path is outside the project")
	}
	return p, nil
}

// relOf reports a path's place in the project, if it is inside it.
func (s *store) relOf(p string) (string, bool) {
	if s.memory {
		return "", false
	}
	r, err := filepath.Rel(s.dir, p)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", false
	}
	return r, true
}

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
		n.Path = filepath.Join(s.dir, n.Rel)
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
		f.Path = filepath.Join(s.dir, f.Rel)
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
	oldAbs, err := s.absRel(oldRel)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Lstat(oldAbs)
	if err != nil {
		return "", nil, err
	}
	newName := fileName(name, filepath.Base(oldAbs))
	if newName == filepath.Base(oldAbs) {
		return oldRel, nil, nil
	}
	newAbs := filepath.Join(filepath.Dir(oldAbs), newName)
	same := false
	if existing, err := os.Lstat(newAbs); err == nil {
		same = os.SameFile(info, existing)
		if !same {
			return "", nil, fmt.Errorf("%s already exists", newName)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", nil, err
	}
	if same {
		// A case-only rename on a case-insensitive volume needs a step
		// aside before the new spelling can take the old name.
		tmp := oldAbs + ".mojian-rename"
		if err := os.Rename(oldAbs, tmp); err != nil {
			return "", nil, err
		}
		if err := os.Rename(tmp, newAbs); err != nil {
			_ = os.Rename(tmp, oldAbs)
			return "", nil, err
		}
	} else if err := os.Rename(oldAbs, newAbs); err != nil {
		return "", nil, err
	}
	newRel, err := filepath.Rel(s.dir, newAbs)
	if err != nil {
		return "", nil, err
	}
	changed := s.notesUnder(oldRel, info.IsDir())
	s.remap(oldRel, newRel, info.IsDir())
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
	dest := uniqueCopyPath(destDir, filepath.Base(srcAbs))
	if err := copyPath(srcAbs, dest); err != nil {
		return "", err
	}
	return relPath(s.dir, dest), nil
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
	oldRel, insideProject := s.relOf(srcAbs)
	if !insideProject {
		dest, err := s.copyInto(srcAbs, destDirRel)
		if err != nil {
			return "", err
		}
		if err := os.RemoveAll(srcAbs); err != nil {
			return dest, err
		}
		if err := s.refresh(); err != nil {
			return dest, err
		}
		return dest, nil
	}
	if filepath.Dir(srcAbs) == destDir {
		return oldRel, nil
	}
	info, err := os.Lstat(srcAbs)
	if err != nil {
		return "", err
	}
	dest := uniquePath(destDir, filepath.Base(srcAbs))
	if err := os.Rename(srcAbs, dest); err != nil {
		// A volume boundary makes rename fail; copy, then remove.
		if err := copyPath(srcAbs, dest); err != nil {
			return "", err
		}
		if err := os.RemoveAll(srcAbs); err != nil {
			return "", err
		}
		s.forget(oldRel)
		if err := s.refresh(); err != nil {
			return relPath(s.dir, dest), err
		}
		return relPath(s.dir, dest), nil
	}
	newRel, err := filepath.Rel(s.dir, dest)
	if err != nil {
		return "", err
	}
	s.remap(oldRel, newRel, info.IsDir())
	return newRel, nil
}

// uniqueCopyPath names a copy "name copy", then "name copy 2".
func uniqueCopyPath(dir, name string) string {
	stem, ext := splitExt(name)
	candidate := stem + " copy" + ext
	for i := 2; ; i++ {
		path := filepath.Join(dir, candidate)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return path
		}
		candidate = fmt.Sprintf("%s copy %d%s", stem, i, ext)
	}
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

// startTreeDraft opens a name field in the tree.
func (a *app) startTreeDraft(kind draftKind, parent, target, name string) {
	if a.store == nil {
		return
	}
	if a.store.memory && kind != draftRename {
		a.fail(errors.New("files need the desktop app"))
		return
	}
	a.treeOpen = true
	a.treeQuery = ""
	a.treeFocusSearch = false
	if kind == draftRename {
		parent = filepath.Dir(target)
		if parent == "." {
			parent = ""
		}
	} else {
		a.treeSel = parent
	}
	a.openPath(parent)
	a.draftSeq++
	a.treeDraft = &treeDraft{kind: kind, parent: parent, target: target, name: name, gen: a.draftSeq}
	a.treeReveal = target
	a.invalidate()
}

func (a *app) startNewFile() {
	a.startTreeDraft(draftFile, a.selectedDir(), "", "")
}

func (a *app) startNewFileIn(dir string) {
	a.startTreeDraft(draftFile, dir, "", "")
}

func (a *app) startNewFolder() {
	a.startTreeDraft(draftFolder, a.selectedDir(), "", "")
}

func (a *app) startRenameNode(rel string) {
	if rel == "" || a.store == nil {
		return
	}
	abs, err := a.store.absRel(rel)
	if err != nil {
		a.fail(err)
		return
	}
	a.startTreeDraft(draftRename, "", rel, filepath.Base(abs))
}

func (a *app) cancelTreeDraft() {
	a.treeDraft = nil
	a.invalidate()
}

// submitTreeDraft creates the named file or folder, or renames the target.
func (a *app) submitTreeDraft() {
	d := a.treeDraft
	if d == nil || a.store == nil {
		return
	}
	name := strings.TrimSpace(d.name)
	if d.kind == draftRename && name == "" {
		a.treeDraft = nil
		return
	}
	var (
		rel     string
		n       *note
		err     error
		changed []*note
	)
	before := map[*note]string{}
	for _, n := range a.store.notes {
		before[n] = editorShape(n.Path)
	}
	switch d.kind {
	case draftFile:
		if name == "" {
			name = "untitled.txt"
		}
		n, err = a.store.newFileIn(d.parent, name, "")
		if n != nil {
			rel = n.Rel
		}
	case draftFolder:
		if name == "" {
			name = "New Folder"
		}
		rel, err = a.store.newFolder(d.parent, name)
	case draftRename:
		rel, changed, err = a.store.rename(d.target, name)
	}
	if err != nil {
		a.fail(err)
		return
	}
	a.treeDraft = nil
	a.treeSel = rel
	if d.kind == draftFolder {
		if a.openDirs == nil {
			a.openDirs = map[string]bool{}
		}
		a.openDirs[rel] = true
	}
	if d.kind == draftRename {
		a.remapOpenDirs(d.target, rel)
	}
	a.openPath(filepath.Dir(rel))
	a.treeReveal = rel
	if d.kind == draftFile && n != nil {
		a.openNote(n)
	}
	if d.kind == draftRename {
		a.resetRenamedStates(changed, before)
	}
	a.syncRemoteProject()
	a.changed()
	a.invalidate()
}

// editorShape is what decides an editor's line numbers and wrapping.
func editorShape(path string) string {
	lang := ""
	if l := editorLanguage(path); l != nil {
		lang = l.Name
	}
	return fmt.Sprintf("%s/%t", lang, isMarkdownPath(path))
}

func (a *app) resetRenamedStates(notes []*note, before map[*note]string) {
	renamed := make(map[*note]bool, len(notes))
	shapeChanged := make(map[*note]bool)
	for _, n := range notes {
		renamed[n] = true
		if before[n] != editorShape(n.Path) {
			shapeChanged[n] = true
		}
	}
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.Note == nil || !renamed[p.Note] {
				continue
			}
			// Markdown previews resolve images relative to the note's
			// folder, which a rename or move may have changed.
			p.md = nil
			if shapeChanged[p.Note] {
				p.state = nil
			}
		}
	}
	if len(shapeChanged) == 0 {
		return
	}
	if p := a.focusPane(); p != nil {
		a.focusReq = p
	}
}

// selectedDir is the folder new files and folders go into: the selected
// folder, the parent of the selected file, or the project root.
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

func (a *app) selectionAbs() (string, bool) {
	if a.store == nil || a.treeSel == "" {
		return "", false
	}
	abs, err := a.store.absRel(a.treeSel)
	if err != nil {
		return "", false
	}
	return abs, true
}

func (a *app) selectionIsDir() bool {
	if a.treeSel == "" {
		return true
	}
	abs, ok := a.selectionAbs()
	if !ok {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && info.IsDir()
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

func (a *app) copySelectionTo(rel string, cut bool) {
	if rel == "" || a.store == nil {
		return
	}
	abs, err := a.store.absRel(rel)
	if err != nil {
		a.fail(err)
		return
	}
	a.fileClip = []string{abs}
	a.fileCut = cut
	if a.win != nil && runtime.GOOS != "js" {
		if err := mygo.Clipboard.WriteFiles(abs); err != nil {
			a.fail(err)
		}
	}
	a.invalidate()
}

func (a *app) duplicate(rel string) {
	if rel == "" {
		return
	}
	abs, err := a.store.absRel(rel)
	if err != nil {
		a.fail(err)
		return
	}
	dest := a.destinationFor(rel)
	newRel, err := a.store.copyInto(abs, dest)
	if err != nil {
		a.fail(err)
		return
	}
	if err := a.store.refresh(); err != nil {
		a.fail(err)
	}
	a.selectRel(newRel)
	a.syncRemoteProject()
}

func (a *app) pasteInto(rel string) {
	if a.store == nil {
		return
	}
	dest := a.destinationFor(rel)
	var paths []string
	move := false
	if a.fileCut && len(a.fileClip) > 0 {
		paths, move = a.fileClip, true
	} else if a.win != nil && runtime.GOOS != "js" {
		if got, err := mygo.Clipboard.ReadFiles(); err == nil && len(got) > 0 {
			paths = got
		}
	}
	if len(paths) == 0 {
		paths = a.fileClip
	}
	if len(paths) == 0 {
		a.fail(errors.New("the clipboard has no files"))
		return
	}
	var first string
	var lastErr error
	for _, p := range paths {
		var newRel string
		var err error
		if move {
			newRel, err = a.store.moveInto(p, dest)
		} else {
			newRel, err = a.store.copyInto(p, dest)
		}
		if err != nil {
			lastErr = err
			continue
		}
		if first == "" {
			first = newRel
		}
	}
	a.fileCut = false
	if err := a.store.refresh(); err != nil {
		a.fail(err)
		return
	}
	if lastErr != nil {
		a.fail(lastErr)
	}
	if first != "" {
		a.selectRel(first)
	}
	a.syncRemoteProject()
}

func (a *app) copyPath(rel string, absolute bool) {
	if rel == "" || a.store == nil {
		return
	}
	if a.win == nil {
		return
	}
	if absolute {
		if abs, err := a.store.absRel(rel); err == nil {
			mygo.Clipboard.WriteText(abs)
		}
		return
	}
	mygo.Clipboard.WriteText(filepath.ToSlash(rel))
}

func (a *app) reveal(rel string) {
	if a.store == nil || runtime.GOOS == "js" {
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
	if runtime.GOOS == "js" {
		a.fail(errors.New("trash needs the desktop app"))
		return
	}
	abs, err := a.store.absRel(rel)
	if err != nil {
		a.fail(err)
		return
	}
	if err := trashPath(abs); err != nil {
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
	a.treeSel = a.destinationFor(rel)
	a.openPath(a.treeSel)
	a.treeReveal = a.treeSel
	a.syncRemoteProject()
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
				if t.Root == nil {
					break
				}
			}
		}
	}
	kept := make([]*tab, 0, len(a.tabs))
	for _, t := range a.tabs {
		if t.Root != nil {
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
	if runtime.GOOS != "js" && m.Item("Reveal in Finder").Chosen() {
		a.later(func() { a.reveal(rel) })
	}
	m.Separator()
	if m.Item("Move to Trash").Shortcut(ui.Cmd, ui.KeyBackspace).Chosen() {
		a.later(func() { a.trash(rel) })
	}
}
