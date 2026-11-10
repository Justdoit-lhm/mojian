package explorer

import (
	"path/filepath"
	"strings"
)

const (
	treeW = 238
	rowH  = 26
)

type DraftKind int

const (
	DraftFile DraftKind = iota
	DraftFolder
	DraftRename
)

// Draft is a name being typed in the tree: a new file, a new folder,
// or the new name of an existing entry.
type Draft struct {
	Kind    DraftKind
	Parent  string
	Target  string
	Name    string
	Gen     int
	Focused bool
}

func (u *UI) dirOpen(rel string) bool {
	return rel == "" || u.OpenDirs()[rel]
}

// OpenPath opens every folder on the way to rel.
func (u *UI) OpenPath(rel string) {
	if u.store() == nil || rel == "" {
		return
	}
	if !u.store().DirExists(rel) {
		rel = filepath.Dir(rel)
	}
	if rel == "." {
		return
	}
	if u.OpenDirs() == nil {
		u.SetOpenDirs(map[string]bool{})
	}
	for d := rel; d != ""; {
		u.OpenDirs()[d] = true
		parent := filepath.Dir(d)
		if parent == "." || parent == d || parent == string(filepath.Separator) {
			break
		}
		d = parent
	}
}

// remapOpenDirs moves the open folders under oldRel to newRel.
func (u *UI) remapOpenDirs(oldRel, newRel string) {
	if oldRel == "" || u.OpenDirs() == nil {
		return
	}
	next := make(map[string]bool, len(u.OpenDirs()))
	prefix := oldRel + string(filepath.Separator)
	for dir, open := range u.OpenDirs() {
		switch {
		case dir == oldRel:
			dir = newRel
		case strings.HasPrefix(dir, prefix):
			dir = newRel + strings.TrimPrefix(dir, oldRel)
		}
		next[dir] = open
	}
	u.SetOpenDirs(next)
}

func (u *UI) toggleDir(rel string) {
	if rel == "" {
		return
	}
	if u.OpenDirs() == nil {
		u.SetOpenDirs(map[string]bool{})
	}
	u.OpenDirs()[rel] = !u.OpenDirs()[rel]
	u.SetTreeSel(rel)
	u.SetTreeDraft(nil)
}

// RefreshProject reads the project from disk, so files made in a terminal
// appear in the tree. The host may do it in the background.
func (u *UI) RefreshProject() { u.ReloadProject(true) }

// refreshProjectSoon is RefreshProject that reuses a listing from the
// last few seconds.
func (u *UI) refreshProjectSoon() { u.ReloadProject(false) }

// RefreshApplied finishes a listing that the host read and applied: the
// tree keeps its selection, vanished editors close, and Git refreshes.
func (u *UI) RefreshApplied() {
	u.afterRefresh()
	u.keepTreeSelection()
	if u.gitOpen() && !u.autoRefreshGit() {
		u.refreshGit()
	}
	u.invalidate()
}

// keepTreeSelection keeps the selected row if it still exists; otherwise
// it falls back to the open note's folder.
func (u *UI) keepTreeSelection() {
	if u.store() == nil || u.TreeSel() == "" {
		return
	}
	if u.store().ExistsRel(u.TreeSel()) {
		return
	}
	u.SetTreeSel("")
	if n := u.currentNote(); n != nil {
		u.RevealInTree(n)
	}
}

// ToggleTree shows or hides the tree, reading the project when it
// appears.
func (u *UI) ToggleTree() {
	u.SetTreeOpen(!u.TreeOpen())
	if u.TreeOpen() {
		// A focus refresh moments ago already has the files; opening the
		// sidebar does not need to force another remote listing.
		u.refreshProjectSoon()
	} else {
		u.SetTreeDraft(nil)
		u.TreeSearch().CancelFocus()
	}
	u.invalidate()
}

// RevealInTree opens the folders of n and selects its row.
func (u *UI) RevealInTree(n *note) {
	if n == nil {
		return
	}
	u.RevealRel(n.Rel)
}

// RevealRel opens the folders of rel and selects its row.
func (u *UI) RevealRel(rel string) {
	if rel == "" {
		return
	}
	if u.OpenDirs() == nil {
		u.SetOpenDirs(map[string]bool{})
	}
	u.SetTreeSel(rel)
	u.OpenPath(rel)
	u.SetTreeReveal(rel)
}
