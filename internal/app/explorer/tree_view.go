package explorer

import (
	"fmt"
	"strings"

	"mojian/internal/icons"
	"mojian/internal/motion"

	"github.com/egoist/mygo/ui"
)

// Sidebar draws the project tree beside the panes.
func (u *UI) Sidebar(c *ui.Context, k *colors) {
	if u.store() == nil {
		return
	}
	card := ui.Column(c.Key("tree:"+u.activeWorkspaceKey())).Width(treeW).Shrink(0).MinHeight(0).Radius(radius).Clip().
		Background(k.Card).Border(1, k.CardBorder).
		Shadow(0, 1, 2, 0, k.Shadow).Shadow(0, 6, 22, -2, k.Shadow).
		Transition(motion.Sidebar)
	card.Children(func() {
		u.sidebarHeader(c, k)
		u.sidebarSearch(c, k)
		if u.emptyTreeVisible() {
			u.emptyTree(c, k)
			return
		}
		scroll := ui.Scroll(c).Grow(1).MinHeight(0).Padding(2, 6)
		scroll.PopoverMenu(func(m *ui.Menu) { u.NodeMenu(m, "", true) })
		scroll.Children(func() {
			if strings.TrimSpace(u.TreeQuery()) != "" {
				u.searchResults(c, k)
				return
			}
			u.treeChildren(c, k, u.store().Tree, 0)
		})
	})
	if u.TreeReveal() != "" {
		u.SetTreeReveal("")
	}
}

// emptyTreeVisible reports whether the Explorer shows its empty prompt
// instead of the scrollable tree.
func (u *UI) emptyTreeVisible() bool {
	if strings.TrimSpace(u.TreeQuery()) != "" {
		return false
	}
	tree := u.store().Tree
	return tree != nil && len(tree.Children) == 0 && u.TreeDraft() == nil
}

func (u *UI) sidebarHeader(c *ui.Context, k *colors) {
	ui.Row(c).Height(headerH).Padding(0, 7, 0, 11).Gap(6).AlignItems(ui.Center).
		PopoverMenu(func(m *ui.Menu) { u.NodeMenu(m, "", true) }).Children(func() {
		ui.Icon(c, icons.FolderIcon()).Size(16, 16).TextColor(k.TextMuted)
		ui.Text(c, "Explorer").FontSize(12.5).FontWeight(600).TextColor(k.Text)
		ui.Spacer(c)
		if iconButton(c, k, "file-plus", "New File", 24, 14).
			Tooltip(u.tip("New File", "new-file")).Clicked() {
			u.later(func() { u.StartNewFile() })
		}
		if iconButton(c, k, "folder-plus", "New Folder", 24, 14).
			Tooltip(u.tip("New Folder", "new-folder")).Clicked() {
			u.later(func() { u.StartNewFolder() })
		}
		if iconButton(c, k, "refresh-cw", "Refresh Project", 24, 14).
			Tooltip("Refresh Project").Clicked() {
			u.later(func() { u.RefreshProject() })
		}
	})
}

func (u *UI) sidebarSearch(c *ui.Context, k *colors) {
	field := ui.Row(c).Height(30).Margin(0, 9, 6, 9).Padding(5, 8).Gap(6).Radius(radius).
		Background(k.Hover).AlignItems(ui.Center)
	field.Children(func() {
		ui.Icon(c, icons.Icon("search")).Size(14, 14).TextColor(k.TextFaint)
		in := ui.TextInputBase(c, u.TreeQueryPtr()).Grow(1).FontSize(12.5).
			Placeholder("Search files…").Label("Search files").Bind(u.TreeSearch())
		if in.Submitted() {
			if notes := u.filteredNotes(); len(notes) > 0 {
				u.SetTreeSel(notes[0].Rel)
				u.openNote(notes[0])
			}
		}
		if u.TreeQuery() != "" {
			if iconButton(c, k, "x", "Clear Search", 20, 11).Clicked() {
				u.SetTreeQuery("")
				u.later(func() { u.TreeSearch().Focus() })
			}
		}
	})
}

// treeChildren draws the children of n, and a draft that goes in it.
func (u *UI) treeChildren(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n == nil {
		return
	}
	if d := u.TreeDraft(); d != nil && d.Kind != DraftRename && d.Parent == n.Rel {
		u.treeDraftRow(c, k, depth, n.Dir)
	}
	for _, child := range n.Children {
		if child.Dir {
			u.treeFolder(c, k, child, depth)
		} else {
			u.treeFile(c, k, child, depth)
		}
	}
}

// treeFolder draws a folder and, when open, its children.
func (u *UI) treeFolder(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n == nil {
		return
	}
	open := u.dirOpen(n.Rel)
	if d := u.TreeDraft(); d != nil && d.Kind == DraftRename && d.Target == n.Rel {
		u.treeDraftRow(c, k, depth, true)
	} else {
		row := ui.Row(c.Key("dir:" + n.Rel)).Height(rowH).
			PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(radius).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
			Focusable().FocusRing(false).Label(n.Name).ClipX().
			PopoverMenu(func(m *ui.Menu) { u.NodeMenu(m, n.Rel, true) }).
			Transition(motion.Row)
		if n.Rel == u.TreeSel() {
			row.Background(k.PanelSel)
		} else if row.Hovered() {
			row.Background(k.Hover)
		}
		if row.Clicked() {
			u.toggleDir(n.Rel)
		}
		if row.Shortcut(0, ui.KeyEnter) || row.Shortcut(0, ui.KeySpace) {
			u.toggleDir(n.Rel)
		}
		u.treeRowKeys(row, n.Rel)
		row.Children(func() {
			arrow := "chevron-right"
			if open {
				arrow = "chevron-down"
			}
			ui.Icon(c, icons.Icon(arrow)).Size(12, 12).TextColor(k.TextFaint)
			ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
			ui.Text(c, n.Name).FontSize(12.5).FontWeight(550).TextColor(k.Text).
				SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		})
	}
	if !open {
		return
	}
	ui.Column(c.Key("kids:" + n.Rel)).Transition(motion.Tree(c.Preferences().ReduceMotion)).Children(func() {
		u.treeChildren(c, k, n, depth+1)
	})
}

// treeFile draws one text file in the tree.
func (u *UI) treeFile(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n.Note == nil && n.Asset == nil {
		return
	}
	path, title := n.Name, n.Name
	if n.Note != nil {
		path, title = n.Note.Path, n.Note.Title
	} else {
		path = n.Asset.Path
	}
	if d := u.TreeDraft(); d != nil && d.Kind == DraftRename && d.Target == n.Rel {
		u.treeDraftRow(c, k, depth, false)
		return
	}
	row := ui.Row(c.Key("file:" + n.Rel)).Height(rowH).
		PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(radius).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
		Focusable().FocusRing(false).Label(title).
		Tooltip(title + " — " + n.Rel).ClipX().
		PopoverMenu(func(m *ui.Menu) { u.NodeMenu(m, n.Rel, false) }).
		Transition(motion.Row)
	if n.Rel == u.TreeSel() {
		row.Background(k.PanelSel)
	} else if n.Note != nil && u.currentNote() == n.Note {
		row.Background(k.Hover)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		u.SetTreeSel(n.Rel)
		u.SetTreeDraft(nil)
		u.openRel(n.Rel)
	}
	if row.Shortcut(0, ui.KeyEnter) || row.Shortcut(0, ui.KeySpace) {
		u.SetTreeSel(n.Rel)
		u.openRel(n.Rel)
	}
	u.treeRowKeys(row, n.Rel)
	if n.Rel == u.TreeReveal() {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Box(c).Size(12, 12).Shrink(0)
		ui.Icon(c, icons.FileIcon(path)).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, n.Name).FontSize(12.5).FontWeight(500).TextColor(k.Text).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		ui.Spacer(c)
		if n.Note != nil && n.Note.Dirty() {
			ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).Shrink(0)
		}
	})
}

// treeRowKeys gives a focused row the keyboard commands of the Explorer.
func (u *UI) treeRowKeys(row ui.Element, rel string) {
	if row.Shortcut(0, ui.KeyF2) {
		u.later(func() { u.StartRenameNode(rel) })
	}
	if row.Shortcut(ui.Cmd, ui.KeyC) {
		u.CopySelectionTo(rel, false)
	}
	if row.Shortcut(ui.Cmd, ui.KeyX) {
		u.CopySelectionTo(rel, true)
	}
	if row.Shortcut(ui.Cmd, ui.KeyV) {
		u.later(func() { u.PasteInto(rel) })
	}
	if row.Shortcut(ui.Cmd, ui.KeyBackspace) {
		u.later(func() { u.Trash(rel) })
	}
}

// treeDraftRow is the field naming a new file or folder, or renaming an
// entry in place.
func (u *UI) treeDraftRow(c *ui.Context, k *colors, depth int, dir bool) {
	d := u.TreeDraft()
	if d == nil {
		return
	}
	row := ui.Row(c.Key(fmt.Sprintf("draft:%d", d.Gen))).Height(rowH).
		PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(radius).
		AlignItems(ui.Center).Background(k.PanelSel)
	row.Children(func() {
		ui.Box(c).Size(12, 12).Shrink(0)
		ic := icons.FileIcon(d.Name)
		if dir {
			ic = icons.FolderIcon()
		}
		ui.Icon(c, ic).Size(15, 15).TextColor(k.TextMuted)
		placeholder := "Folder name"
		switch d.Kind {
		case DraftFile:
			placeholder = "file name, e.g. app.py"
		case DraftRename:
			placeholder = "New name"
		}
		in := ui.TextInputBase(c, &d.Name).Grow(1).MinWidth(0).FontSize(12.5).
			Placeholder(placeholder).AutoFocus()
		if in.Focused() {
			d.Focused = true
		}
		switch {
		case in.Shortcut(0, ui.KeyEscape):
			u.CancelTreeDraft()
		case in.Submitted():
			u.later(func() { u.SubmitTreeDraft() })
		case d.Focused && !in.Focused():
			if d.Kind == DraftRename {
				u.later(func() { u.SubmitTreeDraft() })
			} else {
				u.CancelTreeDraft()
			}
		}
	})
}

// emptyTree is shown for a project or folder with no text files yet. It
// fills the space below the header so its prompt sits in the middle.
func (u *UI) emptyTree(c *ui.Context, k *colors) {
	ui.Column(c).Grow(1).MinHeight(0).Padding(18, 8).Gap(8).Center().
		PopoverMenu(func(m *ui.Menu) { u.NodeMenu(m, "", true) }).
		Children(func() {
			ui.Icon(c, icons.Icon("file-plus")).Size(20, 20).TextColor(k.TextFaint)
			ui.Text(c, "No files here").FontSize(12.5).FontWeight(600).TextColor(k.TextMuted)
			btn := ui.ButtonBase(c).Padding(5, 10).Radius(radius).
				Cursor(ui.CursorPointer).Label("New File").Background(k.Hover)
			if btn.Hovered() {
				btn.Background(k.Pressed)
			}
			if btn.Clicked() {
				u.later(func() { u.StartNewFile() })
			}
			btn.Children(func() {
				ui.Text(c, "New File").FontSize(12).FontWeight(500).TextColor(k.Text)
			})
		})
}

// searchResults lists the files matching the search field.
func (u *UI) searchResults(c *ui.Context, k *colors) {
	notes := u.filteredNotes()
	if len(notes) == 0 {
		ui.Text(c, "No files match").FontSize(12.5).TextColor(k.TextFaint).Padding(10)
		return
	}
	for _, n := range notes {
		n := n
		row := ui.Row(c.Key("result:"+n.Rel)).Padding(6, 7).Gap(8).Radius(radius).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
			Focusable().FocusRing(false).Label(n.Title).
			PopoverMenu(func(m *ui.Menu) { u.NodeMenu(m, n.Rel, false) })
		if n.Rel == u.TreeSel() {
			row.Background(k.PanelSel)
		} else if row.Hovered() {
			row.Background(k.Hover)
		}
		if row.Clicked() {
			u.SetTreeSel(n.Rel)
			u.openNote(n)
		}
		u.treeRowKeys(row, n.Rel)
		row.Children(func() {
			ui.Icon(c, icons.FileIcon(n.Path)).Size(15, 15).TextColor(k.TextMuted)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, n.Title).FontSize(12.5).FontWeight(500).TextColor(k.Text).
					SingleLine().Ellipsis("…")
				ui.Text(c, n.Rel).FontSize(11).TextColor(k.TextFaint).
					SingleLine().Ellipsis("…")
			})
			if n.Dirty() {
				ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).Shrink(0)
			}
		})
	}
}
