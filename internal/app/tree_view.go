package app

import (
	"fmt"
	"strings"

	"mojian/internal/icons"
	"mojian/internal/motion"

	"github.com/egoist/mygo/ui"
)

// sidebar draws the project tree beside the panes.
func (a *app) sidebar(c *ui.Context, k *colors) {
	if a.store == nil {
		return
	}
	card := ui.Column(c).Width(treeW).Shrink(0).MinHeight(0).Radius(radius).Clip().
		Background(k.Card).Border(1, k.CardBorder).
		Shadow(0, 1, 2, 0, k.Shadow).Shadow(0, 6, 22, -2, k.Shadow)
	card.Children(func() {
		a.sidebarHeader(c, k)
		a.sidebarSearch(c, k)
		scroll := ui.Scroll(c).Grow(1).MinHeight(0).Padding(2, 6)
		scroll.PopoverMenu(func(m *ui.Menu) { a.nodeMenu(m, "", true) })
		scroll.Children(func() {
			if strings.TrimSpace(a.treeQuery) != "" {
				a.searchResults(c, k)
				return
			}
			if a.store.tree == nil {
				return
			}
			if len(a.store.tree.Children) == 0 && a.treeDraft == nil {
				a.emptyTree(c, k)
				return
			}
			a.treeChildren(c, k, a.store.tree, 0)
		})
	})
	if a.treeReveal != "" {
		a.treeReveal = ""
	}
}

func (a *app) sidebarHeader(c *ui.Context, k *colors) {
	ui.Row(c).Height(headerH).Padding(0, 7, 0, 11).Gap(6).AlignItems(ui.Center).
		PopoverMenu(func(m *ui.Menu) { a.nodeMenu(m, "", true) }).Children(func() {
		ui.Icon(c, icons.FolderIcon()).Size(16, 16).TextColor(k.TextMuted)
		ui.Text(c, "Explorer").FontSize(12.5).FontWeight(600).TextColor(k.Text)
		ui.Spacer(c)
		if iconButton(c, k, "file-plus", "New File", 24, 14).
			Tooltip(a.tip("New File", cmdNewNote)).Clicked() {
			a.later(func() { a.startNewFile() })
		}
		if iconButton(c, k, "folder-plus", "New Folder", 24, 14).
			Tooltip(a.tip("New Folder", cmdNewFolder)).Clicked() {
			a.later(func() { a.startNewFolder() })
		}
		if iconButton(c, k, "refresh-cw", "Refresh Project", 24, 14).
			Tooltip("Refresh Project").Clicked() {
			a.later(func() { a.refreshProject() })
		}
	})
}

func (a *app) sidebarSearch(c *ui.Context, k *colors) {
	field := ui.Row(c).Height(30).Margin(0, 9, 6, 9).Padding(5, 8).Gap(6).Radius(radius).
		Background(k.Hover).AlignItems(ui.Center)
	field.Children(func() {
		ui.Icon(c, icons.Icon("search")).Size(14, 14).TextColor(k.TextFaint)
		in := ui.TextInputBase(c, &a.treeQuery).Grow(1).FontSize(12.5).
			Placeholder("Search files…").Label("Search files").Bind(&a.treeSearch)
		if in.Submitted() {
			if notes := a.filteredNotes(); len(notes) > 0 {
				a.treeSel = notes[0].Rel
				a.openNote(notes[0])
			}
		}
		if a.treeQuery != "" {
			if iconButton(c, k, "x", "Clear Search", 20, 11).Clicked() {
				a.treeQuery = ""
				a.later(func() { a.treeSearch.Focus() })
			}
		}
	})
}

// treeChildren draws the children of n, and a draft that goes in it.
func (a *app) treeChildren(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n == nil {
		return
	}
	if d := a.treeDraft; d != nil && d.kind != draftRename && d.parent == n.Rel {
		a.treeDraftRow(c, k, depth, n.Dir)
	}
	for _, child := range n.Children {
		if child.Dir {
			a.treeFolder(c, k, child, depth)
		} else {
			a.treeFile(c, k, child, depth)
		}
	}
}

// treeFolder draws a folder and, when open, its children.
func (a *app) treeFolder(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n == nil {
		return
	}
	open := a.dirOpen(n.Rel)
	if d := a.treeDraft; d != nil && d.kind == draftRename && d.target == n.Rel {
		a.treeDraftRow(c, k, depth, true)
	} else {
		row := ui.Row(c.Key("dir:" + n.Rel)).Height(rowH).
			PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(radius).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
			Focusable().FocusRing(false).Label(n.Name).ClipX().
			PopoverMenu(func(m *ui.Menu) { a.nodeMenu(m, n.Rel, true) }).
			Transition(motion.Row)
		if n.Rel == a.treeSel {
			row.Background(k.PanelSel)
		} else if row.Hovered() {
			row.Background(k.Hover)
		}
		if row.Clicked() {
			a.toggleDir(n.Rel)
		}
		if row.Shortcut(0, ui.KeyEnter) || row.Shortcut(0, ui.KeySpace) {
			a.toggleDir(n.Rel)
		}
		a.treeRowKeys(row, n.Rel)
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
		a.treeChildren(c, k, n, depth+1)
	})
}

// treeFile draws one text file in the tree.
func (a *app) treeFile(c *ui.Context, k *colors, n *fileNode, depth int) {
	if n.Note == nil && n.Asset == nil {
		return
	}
	path, title := n.Name, n.Name
	if n.Note != nil {
		path, title = n.Note.Path, n.Note.Title
	} else {
		path = n.Asset.Path
	}
	if d := a.treeDraft; d != nil && d.kind == draftRename && d.target == n.Rel {
		a.treeDraftRow(c, k, depth, false)
		return
	}
	row := ui.Row(c.Key("file:" + n.Rel)).Height(rowH).
		PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(radius).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
		Focusable().FocusRing(false).Label(title).
		Tooltip(title + " — " + n.Rel).ClipX().
		PopoverMenu(func(m *ui.Menu) { a.nodeMenu(m, n.Rel, false) }).
		Transition(motion.Row)
	if n.Rel == a.treeSel {
		row.Background(k.PanelSel)
	} else if n.Note != nil && a.currentNote() == n.Note {
		row.Background(k.Hover)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		a.treeSel = n.Rel
		a.treeDraft = nil
		a.openRel(n.Rel)
	}
	if row.Shortcut(0, ui.KeyEnter) || row.Shortcut(0, ui.KeySpace) {
		a.treeSel = n.Rel
		a.openRel(n.Rel)
	}
	a.treeRowKeys(row, n.Rel)
	if n.Rel == a.treeReveal {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Box(c).Size(12, 12).Shrink(0)
		ui.Icon(c, icons.FileIcon(path)).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, n.Name).FontSize(12.5).FontWeight(500).TextColor(k.Text).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		ui.Spacer(c)
		if n.Note != nil && n.Note.dirty() {
			ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).Shrink(0)
		}
	})
}

// treeRowKeys gives a focused row the keyboard commands of the Explorer.
func (a *app) treeRowKeys(row ui.Element, rel string) {
	if row.Shortcut(0, ui.KeyF2) {
		a.later(func() { a.startRenameNode(rel) })
	}
	if row.Shortcut(ui.Cmd, ui.KeyC) {
		a.copySelectionTo(rel, false)
	}
	if row.Shortcut(ui.Cmd, ui.KeyX) {
		a.copySelectionTo(rel, true)
	}
	if row.Shortcut(ui.Cmd, ui.KeyV) {
		a.later(func() { a.pasteInto(rel) })
	}
	if row.Shortcut(ui.Cmd, ui.KeyBackspace) {
		a.later(func() { a.trash(rel) })
	}
}

// treeDraftRow is the field naming a new file or folder, or renaming an
// entry in place.
// treeDraftRow is the field naming a new file or folder, or renaming an
// entry in place.
func (a *app) treeDraftRow(c *ui.Context, k *colors, depth int, dir bool) {
	d := a.treeDraft
	if d == nil {
		return
	}
	row := ui.Row(c.Key(fmt.Sprintf("draft:%d", d.gen))).Height(rowH).
		PaddingStart(5 + float32(depth)*13).PaddingEnd(6).Gap(5).Radius(radius).
		AlignItems(ui.Center).Background(k.PanelSel)
	row.Children(func() {
		ui.Box(c).Size(12, 12).Shrink(0)
		ic := icons.FileIcon(d.name)
		if dir {
			ic = icons.FolderIcon()
		}
		ui.Icon(c, ic).Size(15, 15).TextColor(k.TextMuted)
		placeholder := "Folder name"
		switch d.kind {
		case draftFile:
			placeholder = "file name, e.g. app.py"
		case draftRename:
			placeholder = "New name"
		}
		in := ui.TextInputBase(c, &d.name).Grow(1).MinWidth(0).FontSize(12.5).
			Placeholder(placeholder).AutoFocus()
		if in.Focused() {
			d.focused = true
		}
		switch {
		case in.Shortcut(0, ui.KeyEscape):
			a.cancelTreeDraft()
		case in.Submitted():
			a.later(func() { a.submitTreeDraft() })
		case d.focused && !in.Focused():
			if d.kind == draftRename {
				a.later(func() { a.submitTreeDraft() })
			} else {
				a.cancelTreeDraft()
			}
		}
	})
}

// emptyTree is shown for a project or folder with no text files yet.
func (a *app) emptyTree(c *ui.Context, k *colors) {
	ui.Column(c).Padding(18, 8).Gap(8).Center().Children(func() {
		ui.Icon(c, icons.Icon("file-plus")).Size(20, 20).TextColor(k.TextFaint)
		ui.Text(c, "No files here").FontSize(12.5).FontWeight(600).TextColor(k.TextMuted)
		btn := ui.Row(c).Padding(5, 10).Radius(radius).Cursor(ui.CursorPointer).
			Role(ui.RoleButton).Label("New File").Background(k.Hover)
		if btn.Hovered() {
			btn.Background(k.Pressed)
		}
		if btn.Clicked() {
			a.later(func() { a.startNewFile() })
		}
		btn.Children(func() {
			ui.Text(c, "New File").FontSize(12).FontWeight(500).TextColor(k.Text)
		})
	})
}

// searchResults lists the files matching the search field.
func (a *app) searchResults(c *ui.Context, k *colors) {
	notes := a.filteredNotes()
	if len(notes) == 0 {
		ui.Text(c, "No files match").FontSize(12.5).TextColor(k.TextFaint).Padding(10)
		return
	}
	for _, n := range notes {
		n := n
		row := ui.Row(c.Key("result:"+n.Rel)).Padding(6, 7).Gap(8).Radius(radius).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleTreeItem).
			Focusable().FocusRing(false).Label(n.Title).
			PopoverMenu(func(m *ui.Menu) { a.nodeMenu(m, n.Rel, false) })
		if n.Rel == a.treeSel {
			row.Background(k.PanelSel)
		} else if row.Hovered() {
			row.Background(k.Hover)
		}
		if row.Clicked() {
			a.treeSel = n.Rel
			a.openNote(n)
		}
		a.treeRowKeys(row, n.Rel)
		row.Children(func() {
			ui.Icon(c, icons.FileIcon(n.Path)).Size(15, 15).TextColor(k.TextMuted)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, n.Title).FontSize(12.5).FontWeight(500).TextColor(k.Text).
					SingleLine().Ellipsis("…")
				ui.Text(c, n.Rel).FontSize(11).TextColor(k.TextFaint).
					SingleLine().Ellipsis("…")
			})
			if n.dirty() {
				ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).Shrink(0)
			}
		})
	}
}
