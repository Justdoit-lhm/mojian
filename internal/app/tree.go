package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const (
	treeW = 238
	rowH  = 26
)

// fileNode is one entry of the project tree: a folder, a text file, or a
// file a viewer shows.
type fileNode struct {
	Name     string
	Rel      string
	Dir      bool
	Note     *note
	Asset    *assetFile
	Children []*fileNode
}

type draftKind int

const (
	draftFile draftKind = iota
	draftFolder
	draftRename
)

// treeDraft is a name being typed in the tree: a new file, a new folder,
// or the new name of an existing entry.
type treeDraft struct {
	kind    draftKind
	parent  string
	target  string
	name    string
	gen     int
	focused bool
}

// buildTree caches the project as a tree, so every frame reads what a
// refresh left instead of the disk.
func (s *store) buildTree() {
	if s.isMemory() {
		root := &fileNode{Name: "Mojian", Dir: true}
		for _, n := range s.sorted() {
			root.Children = append(root.Children, &fileNode{
				Name: filepath.Base(n.Rel),
				Rel:  n.Rel,
				Note: n,
			})
		}
		sortNodes(root.Children)
		s.tree = root
		return
	}
	s.tree = s.treeFromEntries()
}

// treeFromEntries builds the tree from the last listing, so no backend
// walks the disk twice.
func (s *store) treeFromEntries() *fileNode {
	root := &fileNode{Name: filepath.Base(s.dir), Dir: true}
	nodes := map[string]*fileNode{"": root}
	for _, e := range s.entries {
		rel := filepath.FromSlash(e.Rel)
		parent := filepath.Dir(rel)
		if parent == "." {
			parent = ""
		}
		p := nodes[parent]
		if p == nil {
			continue
		}
		n := &fileNode{Name: filepath.Base(rel), Rel: rel, Dir: e.Dir}
		if e.Dir {
			nodes[rel] = n
		} else if note := s.byRel[rel]; note != nil {
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

func sortNodes(ns []*fileNode) {
	sort.SliceStable(ns, func(i, j int) bool {
		a, b := strings.ToLower(ns[i].Name), strings.ToLower(ns[j].Name)
		if a == b {
			return ns[i].Name < ns[j].Name
		}
		return a < b
	})
}

// newFolder makes a folder in parent and returns its path within the
// project. An existing name takes the next number.
func (s *store) newFolder(parent, name string) (string, error) {
	name = folderName(name)
	for i := 1; i < 1000; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s %d", name, i)
		}
		rel := filepath.Join(parent, candidate)
		if s.existsRel(rel) {
			continue
		}
		if err := s.fs.Mkdir(rel); err != nil {
			return "", err
		}
		if err := s.refresh(); err != nil {
			return rel, err
		}
		return rel, nil
	}
	return "", fmt.Errorf("too many folders named %q", name)
}

// folderName keeps a typed name a single folder name.
func folderName(name string) string { return fileName(name, "New Folder") }

func (a *app) dirOpen(rel string) bool {
	return rel == "" || a.openDirs[rel]
}

// openPath opens every folder on the way to rel.
func (a *app) openPath(rel string) {
	if a.store == nil || rel == "" {
		return
	}
	if !a.store.dirExists(rel) {
		rel = filepath.Dir(rel)
	}
	if rel == "." {
		return
	}
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	for d := rel; d != ""; {
		a.openDirs[d] = true
		parent := filepath.Dir(d)
		if parent == "." || parent == d || parent == string(filepath.Separator) {
			break
		}
		d = parent
	}
}

// remapOpenDirs moves the open folders under oldRel to newRel.
func (a *app) remapOpenDirs(oldRel, newRel string) {
	if oldRel == "" || a.openDirs == nil {
		return
	}
	next := make(map[string]bool, len(a.openDirs))
	prefix := oldRel + string(filepath.Separator)
	for dir, open := range a.openDirs {
		switch {
		case dir == oldRel:
			dir = newRel
		case strings.HasPrefix(dir, prefix):
			dir = newRel + strings.TrimPrefix(dir, oldRel)
		}
		next[dir] = open
	}
	a.openDirs = next
}

func (a *app) toggleDir(rel string) {
	if rel == "" {
		return
	}
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	a.openDirs[rel] = !a.openDirs[rel]
	a.treeSel = rel
	a.treeDraft = nil
}

// refreshProject reads the project from disk, so files made in a terminal
// appear in the tree, and closes editors whose file disappeared.
func (a *app) refreshProject() {
	if a.store == nil || a.store.isMemory() {
		return
	}
	if err := a.store.refresh(); err != nil {
		a.fail(err)
		return
	}
	a.closeOrphanPanes()
	a.keepTreeSelection()
	if a.git.Open && !a.NoGitAutoRefresh {
		a.refreshGit()
	}
	a.invalidate()
}

func (a *app) closeOrphanPanes() {
	if a.store == nil {
		return
	}
	liveNotes := make(map[*note]bool, len(a.store.notes))
	for _, n := range a.store.notes {
		liveNotes[n] = true
	}
	a.closeNotes(func(n *note) bool { return !liveNotes[n] && !n.dirty() })
	// Keep an asset pane on the file it shows when the refresh reused
	// the entry, and rebind it when the file changed on disk.
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.Asset == nil {
				continue
			}
			live := a.store.assets[p.Asset.Rel]
			if live == nil || live == p.Asset {
				continue
			}
			releaseAsset(p)
			p.Asset = live
			p.assetView = &assetView{}
			p.Preview = false
		}
	}
	a.closeAssets(func(f *assetFile) bool {
		_, ok := a.store.assets[f.Rel]
		return !ok
	})
}

// keepTreeSelection keeps the selected row if it still exists; otherwise
// it falls back to the open note's folder.
func (a *app) keepTreeSelection() {
	if a.store == nil || a.treeSel == "" {
		return
	}
	if a.store.existsRel(a.treeSel) {
		return
	}
	a.treeSel = ""
	if n := a.currentNote(); n != nil {
		a.revealInTree(n)
	}
}

// toggleTree shows or hides the tree, reading the project when it
// appears.
func (a *app) toggleTree() {
	a.treeOpen = !a.treeOpen
	if a.treeOpen {
		a.refreshProject()
	} else {
		a.treeDraft = nil
		a.treeSearch.CancelFocus()
	}
	a.invalidate()
}

// revealInTree opens the folders of n and selects its row.
func (a *app) revealInTree(n *note) {
	if n == nil {
		return
	}
	a.revealRel(n.Rel)
}

// revealRel opens the folders of rel and selects its row.
func (a *app) revealRel(rel string) {
	if rel == "" {
		return
	}
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	a.treeSel = rel
	a.openPath(rel)
	a.treeReveal = rel
}
