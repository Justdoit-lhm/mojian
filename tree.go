package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	treeW = 238
	rowH  = 26
)

// fileNode is one entry of the vault's tree: a folder, or a Markdown note.
type fileNode struct {
	Name     string
	Rel      string
	Dir      bool
	Note     *note
	Children []*fileNode
}

// folderDraft is a folder being named in the tree.
type folderDraft struct {
	parent  string
	name    string
	gen     int
	focused bool
}

// buildTree caches the vault as a tree, so every frame reads what a
// refresh left instead of the disk.
func (s *store) buildTree() {
	if s.memory {
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
	s.tree = s.readDirTree(s.dir, "")
}

// readDirTree reads the folders and the notes they hold, skipping hidden
// folders and anything that is not Markdown.
func (s *store) readDirTree(dir, rel string) *fileNode {
	n := &fileNode{Name: filepath.Base(dir), Rel: rel, Dir: true}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return n
	}
	var folders, files []*fileNode
	for _, e := range entries {
		name := e.Name()
		childRel := filepath.Join(rel, name)
		if e.IsDir() {
			if skipDir(name) {
				continue
			}
			folders = append(folders, s.readDirTree(filepath.Join(dir, name), childRel))
			continue
		}
		note := s.byRel[childRel]
		if note == nil {
			continue
		}
		files = append(files, &fileNode{Name: name, Rel: childRel, Note: note})
	}
	sortNodes(folders)
	sortNodes(files)
	n.Children = append(folders, files...)
	return n
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

// dirPath resolves a folder within the vault, refusing anything outside
// it.
func (s *store) dirPath(rel string) (string, error) {
	p := filepath.Join(s.dir, rel)
	r, err := filepath.Rel(s.dir, p)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("folder is outside the vault")
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", rel)
	}
	return p, nil
}

func (s *store) dirExists(rel string) bool {
	_, err := s.dirPath(rel)
	return err == nil
}

// newFolder makes a folder in parent and returns its path within the
// vault. An existing name takes the next number, so naming a folder twice
// never fails.
func (s *store) newFolder(parent, name string) (string, error) {
	if s.memory {
		return "", errors.New("folders need the desktop app")
	}
	dir, err := s.dirPath(parent)
	if err != nil {
		return "", err
	}
	name = folderName(name)
	for i := 1; i < 1000; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s %d", name, i)
		}
		path := filepath.Join(dir, candidate)
		if _, err := os.Stat(path); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
			if err := os.Mkdir(path, 0o755); err != nil {
				return "", err
			}
			return relPath(s.dir, path), nil
		}
	}
	return "", fmt.Errorf("too many folders named %q", name)
}

// folderName keeps a typed name a single folder name.
func folderName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	if name == "" || name == "." || name == ".." {
		return "New Folder"
	}
	return name
}

func (a *app) dirOpen(rel string) bool {
	return rel == "" || a.openDirs[rel]
}

// openPath opens every folder on the way to rel.
func (a *app) openPath(rel string) {
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

func (a *app) toggleDir(rel string) {
	if rel == "" {
		return
	}
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	a.openDirs[rel] = !a.openDirs[rel]
	a.treeSel = rel
	a.folderDraft = nil
}

// refreshProject reads the project from disk, so files made in a
// terminal appear in the tree.
func (a *app) refreshProject() {
	if a.store == nil || a.store.memory {
		return
	}
	if err := a.store.refresh(); err != nil {
		a.fail(err)
		return
	}
	if n := a.currentNote(); n != nil {
		a.revealInTree(n)
	}
	a.invalidate()
}

// toggleTree shows or hides the tree, reading the project when it
// appears.
func (a *app) toggleTree() {
	a.treeOpen = !a.treeOpen
	if a.treeOpen {
		a.refreshProject()
	} else {
		a.folderDraft = nil
		a.treeFocusSearch = false
	}
	a.invalidate()
}

// openTreeSearch shows the tree with the search field focused, for the
// Open Note command.
func (a *app) openTreeSearch() {
	a.paletteOpen = false
	a.treeOpen = true
	a.treeQuery = ""
	a.treeFocusSearch = true
	a.refreshProject()
	a.invalidate()
}

// revealInTree opens the folders of n and selects its folder.
func (a *app) revealInTree(n *note) {
	if n == nil {
		return
	}
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	dir := filepath.Dir(n.Rel)
	if dir == "." {
		dir = ""
	}
	a.treeSel = dir
	a.openPath(dir)
	a.treeReveal = n.Rel
}

// selectedDir is the folder new notes and folders go into.
func (a *app) selectedDir() string {
	if a.store == nil || a.treeSel == "" {
		return ""
	}
	if !a.store.dirExists(a.treeSel) {
		return ""
	}
	return a.treeSel
}

// newFolder starts naming a folder in the selected folder.
func (a *app) newFolder() {
	if a.store == nil {
		return
	}
	if a.store.memory {
		a.fail(errors.New("folders need the desktop app"))
		return
	}
	a.treeOpen = true
	parent := a.selectedDir()
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	a.openPath(parent)
	a.draftSeq++
	a.folderDraft = &folderDraft{parent: parent, gen: a.draftSeq}
	a.treeQuery = ""
	a.invalidate()
}

func (a *app) cancelFolder() {
	a.folderDraft = nil
	a.invalidate()
}

// createFolder makes the folder the draft names, then shows it.
func (a *app) createFolder() {
	d := a.folderDraft
	if d == nil || a.store == nil {
		return
	}
	rel, err := a.store.newFolder(d.parent, d.name)
	if err != nil {
		a.fail(err)
		return
	}
	a.folderDraft = nil
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	a.openDirs[rel] = true
	a.treeSel = rel
	if err := a.store.refresh(); err != nil {
		a.fail(err)
	}
	if p := a.focusPane(); p != nil {
		a.focusReq = p
	}
	a.invalidate()
}

// sidebar draws the vault's tree beside the panes.
func (a *app) sidebar(c *ui.Context, k *colors) {
	if a.store == nil {
		return
	}
	card := ui.Column(c).Width(treeW).Shrink(0).MinHeight(0).Radius(cardR).Clip().
		Background(k.card).Border(1, k.cardBorder).
		Shadow(0, 1, 2, 0, k.shadow).Shadow(0, 6, 22, -2, k.shadow)
	card.Children(func() {
		a.sidebarHeader(c, k)
		a.sidebarSearch(c, k)
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(2, 6).Children(func() {
			if strings.TrimSpace(a.treeQuery) != "" {
				a.searchResults(c, k)
				return
			}
			if a.store.tree == nil {
				return
			}
			a.treeRoot(c, k)
			a.treeFolder(c, k, a.store.tree, 0)
		})
		a.sidebarFooter(c, k)
	})
	if a.treeReveal != "" {
		a.treeReveal = ""
	}
}

func (a *app) sidebarHeader(c *ui.Context, k *colors) {
	ui.Row(c).Height(headerH).Padding(0, 7, 0, 11).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon("folder-open")).Size(15, 15).TextColor(k.textMuted)
		ui.Text(c, "Files").FontSize(12.5).FontWeight(600).TextColor(k.text)
		ui.Spacer(c)
		if iconButton(c, k, "refresh-cw", "Refresh Project", 26, 14).
			Tooltip("Refresh Project").Clicked() {
			a.refreshProject()
		}
		if iconButton(c, k, "file-plus", "New Note", 26, 15).
			Tooltip(tip("New Note", cmdNewNote)).Clicked() {
			a.later(func() { a.newNote() })
		}
	})
}

func (a *app) sidebarSearch(c *ui.Context, k *colors) {
	field := ui.Row(c).Height(30).Margin(0, 9, 6, 9).Padding(5, 8).Gap(6).Radius(8).
		Background(k.hover).AlignItems(ui.Center)
	field.Children(func() {
		ui.Icon(c, icon("search")).Size(14, 14).TextColor(k.textFaint)
		in := ui.TextInputBase(c, &a.treeQuery).Grow(1).FontSize(12.5).
			Placeholder("Search files…")
		if a.treeFocusSearch {
			in.AutoFocus()
			if in.Focused() {
				a.treeFocusSearch = false
			}
		}
		if in.Submitted() {
			if notes := a.filteredNotes(); len(notes) > 0 {
				a.openNote(notes[0])
			}
		}
		if a.treeQuery != "" {
			if iconButton(c, k, "x", "Clear Search", 20, 11).Clicked() {
				a.treeQuery = ""
				a.treeFocusSearch = true
			}
		}
	})
}

func (a *app) sidebarFooter(c *ui.Context, k *colors) {
	ui.Divider(c)
	ui.Row(c).Height(34).Padding(0, 7, 0, 11).Gap(1).AlignItems(ui.Center).Children(func() {
		ui.Spacer(c)
		if runtime.GOOS != "js" {
			if iconButton(c, k, "square-terminal", "Open Terminal", 24, 14).
				Tooltip(tip("Open Terminal", cmdTerminal)).Clicked() {
				a.openTerminal()
			}
			if iconButton(c, k, "folder-open", "Reveal Project", 24, 14).
				Tooltip("Reveal Project").Clicked() {
				a.revealVault()
			}
		}
	})
}

// treeRoot is the vault itself, the folder new things go in when nothing
// else is chosen.
func (a *app) treeRoot(c *ui.Context, k *colors) {
	name := "Mojian"
	if !a.store.memory {
		name = filepath.Base(a.store.dir)
	}
	row := ui.Row(c.Key("dir:")).Height(rowH).PaddingStart(5).PaddingEnd(6).Gap(5).
		Radius(7).AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
		Label(name).Tooltip(a.store.displayDir()).ClipX()
	if a.treeSel == "" || row.Hovered() {
		row.Background(k.hover)
	}
	if row.Clicked() {
		a.treeSel = ""
		a.folderDraft = nil
	}
	row.Children(func() {
		ui.Box(c).Size(12, 12).Shrink(0)
		ui.Icon(c, icon("folder-open")).Size(14, 14).TextColor(k.textMuted)
		ui.Text(c, name).FontSize(12.5).FontWeight(600).TextColor(k.text).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
	})
}

// treeFolder draws a folder's children, and the folder's row unless it is
// the root.
func (a *app) treeFolder(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n == nil {
		return
	}
	if n.Rel != "" {
		open := a.dirOpen(n.Rel)
		row := ui.Row(c.Key("dir:" + n.Rel)).Height(rowH).
			PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(7).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
			Label(n.Name).ClipX()
		if n.Rel == a.treeSel {
			row.Background(k.hover)
		} else if row.Hovered() {
			row.Background(k.hover)
		}
		if row.Clicked() {
			a.toggleDir(n.Rel)
		}
		row.Children(func() {
			arrow := "chevron-right"
			if open {
				arrow = "chevron-down"
			}
			ui.Icon(c, icon(arrow)).Size(12, 12).TextColor(k.textFaint)
			folder := "folder"
			if open {
				folder = "folder-open"
			}
			ui.Icon(c, icon(folder)).Size(14, 14).TextColor(k.textMuted)
			ui.Text(c, n.Name).FontSize(12.5).TextColor(k.text).
				SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		})
		if !open {
			return
		}
	}
	if a.folderDraft != nil && a.folderDraft.parent == n.Rel {
		a.folderDraftRow(c, k, depth+1)
	}
	for _, child := range n.Children {
		if child.Dir {
			a.treeFolder(c, k, child, depth+1)
		} else {
			a.treeFile(c, k, child, depth+1)
		}
	}
}

// treeFile draws one note in the tree.
func (a *app) treeFile(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n.Note == nil {
		return
	}
	row := ui.Row(c.Key("note:" + n.Rel)).Height(rowH).
		PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(7).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleButton).
		Label(n.Note.Title).Tooltip(n.Note.Title + " — " + n.Rel).ClipX()
	if a.currentNote() == n.Note {
		row.Background(k.panelSel)
	} else if row.Hovered() {
		row.Background(k.hover)
	}
	if row.Clicked() {
		a.openNote(n.Note)
	}
	if n.Rel == a.treeReveal {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Box(c).Size(12, 12).Shrink(0)
		ui.Icon(c, icon("file-text")).Size(14, 14).TextColor(k.textMuted)
		ui.Text(c, n.Name).FontSize(12.5).FontWeight(500).TextColor(k.text).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		ui.Spacer(c)
		if n.Note.dirty() {
			ui.Box(c).Size(6, 6).Radius(3).Background(k.attention).Shrink(0)
		}
	})
}

// folderDraftRow is the field naming a new folder.
func (a *app) folderDraftRow(c *ui.Context, k *colors, depth int) {
	d := a.folderDraft
	row := ui.Row(c.Key(fmt.Sprintf("draft:%d", d.gen))).Height(rowH).
		PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(7).
		AlignItems(ui.Center).Background(k.panelSel)
	row.Children(func() {
		ui.Box(c).Size(12, 12).Shrink(0)
		ui.Icon(c, icon("folder")).Size(14, 14).TextColor(k.textMuted)
		in := ui.TextInputBase(c, &d.name).Grow(1).MinWidth(0).FontSize(12.5).
			Placeholder("New Folder").AutoFocus()
		if in.Focused() {
			d.focused = true
		}
		switch {
		case in.Shortcut(0, ui.KeyEscape):
			a.cancelFolder()
		case in.Submitted():
			a.later(func() { a.createFolder() })
		case d.focused && !in.Focused():
			a.cancelFolder()
		}
	})
}

// searchResults lists the notes matching the search field.
func (a *app) searchResults(c *ui.Context, k *colors) {
	notes := a.filteredNotes()
	if len(notes) == 0 {
		ui.Text(c, "No files match").FontSize(12.5).TextColor(k.textFaint).Padding(10)
		return
	}
	for _, n := range notes {
		n := n
		row := ui.Row(c.Key("result:"+n.Rel)).Padding(6, 7).Gap(8).Radius(7).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleButton).
			Label(n.Title)
		if a.currentNote() == n {
			row.Background(k.panelSel)
		} else if row.Hovered() {
			row.Background(k.hover)
		}
		if row.Clicked() {
			a.openNote(n)
		}
		row.Children(func() {
			ui.Icon(c, icon("file-text")).Size(14, 14).TextColor(k.textMuted)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, n.Title).FontSize(12.5).FontWeight(500).TextColor(k.text).
					SingleLine().Ellipsis("…")
				ui.Text(c, n.Rel).FontSize(11).TextColor(k.textFaint).
					SingleLine().Ellipsis("…")
			})
			if n.dirty() {
				ui.Box(c).Size(6, 6).Radius(3).Background(k.attention).Shrink(0)
			}
		})
	}
}
