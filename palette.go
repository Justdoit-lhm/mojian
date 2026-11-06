package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

// command is an action of the menus, the shortcuts and the palette.
type command struct {
	title string
	// accel is the menu's shortcut, keys what the palette shows, and glyph
	// the icon of its row.
	accel, keys, glyph string
	run                func(*app)
	hidden             bool
}

var (
	cmdNewNote = &command{title: "New Note", accel: "CmdOrCtrl+N", keys: "⌘N", glyph: "file-plus",
		run: func(a *app) { a.newNote() }}
	cmdNewTab = &command{title: "New Tab", accel: "CmdOrCtrl+T", keys: "⌘T", glyph: "plus",
		run: func(a *app) { a.newTab() }}
	cmdSave = &command{title: "Save", accel: "CmdOrCtrl+S", keys: "⌘S", glyph: "save",
		run: func(a *app) { a.saveNow() }}
	cmdSplitRight = &command{title: "Split Right", accel: "CmdOrCtrl+Shift+D", keys: "⇧⌘D", glyph: "columns-2",
		run: func(a *app) { a.split(false) }}
	cmdSplitDown = &command{title: "Split Down", accel: "CmdOrCtrl+Shift+E", keys: "⇧⌘E", glyph: "rows-2",
		run: func(a *app) { a.split(true) }}
	cmdPreview = &command{title: "Toggle Preview", accel: "CmdOrCtrl+Shift+V", keys: "⇧⌘V", glyph: "eye",
		run: func(a *app) { a.togglePreview() }}
	cmdZoom = &command{title: "Zoom Pane", accel: "CmdOrCtrl+Shift+M", keys: "⇧⌘M", glyph: "maximize-2",
		run: func(a *app) { a.toggleZoom() }}
	cmdEqualize = &command{title: "Equalize Panes", accel: "CmdOrCtrl+Ctrl+=", keys: "⌃⌘=", glyph: "columns-2",
		run: func(a *app) { a.equalize() }}
	cmdClosePane = &command{title: "Close Pane", accel: "CmdOrCtrl+W", keys: "⌘W", glyph: "x",
		run: func(a *app) { a.closePane(a.focusPane()) }}
	cmdCloseTab = &command{title: "Close Tab", accel: "CmdOrCtrl+Shift+W", keys: "⇧⌘W", glyph: "x",
		run: func(a *app) { a.closeTab(a.tab()) }}
	cmdRenameTab = &command{title: "Rename Tab…", accel: "CmdOrCtrl+Shift+R", keys: "⇧⌘R", glyph: "pencil",
		run: func(a *app) { a.startRename(a.tab()) }}
	cmdNextTab = &command{title: "Next Tab", accel: "Ctrl+Tab", keys: "⌃⇥", glyph: "chevron-down",
		run: func(a *app) { a.cycleTab(1) }}
	cmdPrevTab = &command{title: "Previous Tab", accel: "Ctrl+Shift+Tab", keys: "⌃⇧⇥", glyph: "chevron-down",
		run: func(a *app) { a.cycleTab(-1) }}
	cmdPalette = &command{title: "Command Palette…", accel: "CmdOrCtrl+Shift+P", keys: "⇧⌘P", glyph: "command",
		run: func(a *app) { a.openPalette() }}
	cmdNotes = &command{title: "Open Note…", accel: "CmdOrCtrl+Shift+O", keys: "⇧⌘O", glyph: "file-text",
		run: func(a *app) {
			a.notesOpen = true
			a.noteQuery = ""
			if a.store != nil {
				_ = a.store.refresh()
			}
		}}
)

// paletteCommands are the commands the palette lists.
var paletteCommands = []*command{
	cmdNewNote, cmdNewTab, cmdSave, cmdPalette, cmdNotes,
	cmdSplitRight, cmdSplitDown, cmdPreview, cmdZoom, cmdEqualize,
	cmdClosePane, cmdCloseTab, cmdRenameTab, cmdNextTab, cmdPrevTab,
}

// focusPane returns the focused pane of the active tab.
func (a *app) focusPane() *pane {
	if t := a.tab(); t != nil {
		return t.Focus
	}
	return nil
}

func (a *app) openPalette() {
	a.paletteOpen = true
	a.paletteQuery = ""
	a.paletteSel = 0
	a.invalidate()
}

// paletteItem is a row of the command palette: a command, a note, or a tab.
type paletteItem struct {
	title, detail, keys, glyph string
	run                        func()
}

func (a *app) paletteItems() []paletteItem {
	var items []paletteItem
	q := strings.ToLower(strings.TrimSpace(a.paletteQuery))
	match := func(s string) bool { return q == "" || fuzzy(strings.ToLower(s), q) }
	for _, c := range paletteCommands {
		if !match(c.title) {
			continue
		}
		c := c
		items = append(items, paletteItem{title: c.title, keys: c.keys, glyph: c.glyph,
			run: func() { a.run(c) }})
	}
	if a.store != nil {
		for _, n := range a.store.sorted() {
			if !match(n.Title + " " + n.Rel) {
				continue
			}
			n := n
			items = append(items, paletteItem{title: n.Title, detail: n.Rel, glyph: "file-text",
				run: func() {
					a.openNote(n)
					a.updateTitle()
				}})
		}
	}
	for i, t := range a.tabs {
		name, detail := t.label()
		if !match(name + " " + detail) {
			continue
		}
		i := i
		items = append(items, paletteItem{title: name, detail: detail, glyph: "layout-grid", keys: fmt.Sprintf("⌘%d", i+1),
			run: func() { a.showTab(i + 1); a.updateTitle() }})
	}
	return items
}

// fuzzy reports whether the letters of q appear in s, in order.
func fuzzy(s, q string) bool {
	if strings.Contains(s, q) {
		return true
	}
	i := 0
	for _, r := range s {
		if i < len(q) && byte(r) == q[i] {
			i++
		}
	}
	return i == len(q)
}

// palette draws the command palette over the window.
func (a *app) palette(c *ui.Context, k *colors) {
	if !a.paletteOpen {
		return
	}
	items := a.paletteItems()
	a.paletteSel = min(max(a.paletteSel, 0), max(len(items)-1, 0))
	was := a.paletteOpen
	reveal := false
	ui.DialogBase(c, &a.paletteOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(k.backdrop).Justify(ui.Start).Padding(78, 0, 0, 0)
		panel.Width(560).MaxHeight(440).Radius(16).Background(k.panel).Border(1, k.panelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.06)).Clip()
		ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(k.panelBorder).Children(func() {
			ui.Icon(c, icon("search")).Size(17, 17).TextColor(k.textFaint)
			in := ui.TextInputBase(c, &a.paletteQuery).Grow(1).FontSize(15).
				Placeholder("Type a command, or the name of a note…").AutoFocus()
			if in.Changed() {
				a.paletteSel, reveal = 0, true
			}
			if in.Shortcut(0, ui.KeyDown) {
				a.paletteSel, reveal = min(a.paletteSel+1, len(items)-1), true
			}
			if in.Shortcut(0, ui.KeyUp) {
				a.paletteSel, reveal = max(a.paletteSel-1, 0), true
			}
			if in.Submitted() && len(items) > 0 {
				a.paletteOpen = false
				items[a.paletteSel].run()
			}
		})
		listH := float32(min(max(len(items), 1)*34+12, 384))
		ui.Scroll(c).Height(listH).Padding(6).Children(func() {
			if len(items) == 0 {
				ui.Text(c, "No matches").FontSize(13).TextColor(k.textFaint).Padding(14)
			}
			for i, it := range items {
				it := it
				row := ui.Row(c.Key(i)).Height(34).Padding(0, 10).Gap(10).Radius(8).
					AlignItems(ui.Center).Cursor(ui.CursorPointer)
				if i == a.paletteSel {
					row.Background(k.panelSel)
					if reveal {
						row.ScrollIntoView()
					}
				} else if row.Hovered() {
					row.Background(k.hover)
				}
				if row.Clicked() {
					a.paletteOpen = false
					it.run()
				}
				row.Children(func() {
					ui.Icon(c, icon(it.glyph)).Size(15, 15).TextColor(k.textMuted)
					ui.Text(c, it.title).FontSize(13.5).FontWeight(500).TextColor(k.text).
						SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
					if it.detail != "" {
						ui.Text(c, it.detail).FontSize(13).TextColor(k.textFaint).
							SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
					}
					ui.Spacer(c)
					if it.keys != "" {
						ui.Text(c, it.keys).FontSize(12).TextColor(k.textFaint).
							Padding(2, 7).Radius(6).Background(k.hover).Shrink(0)
					}
				})
			}
		})
	})
	if was && !a.paletteOpen {
		if t := a.tab(); t != nil && a.focusReq == nil {
			a.focusReq = t.Focus
		}
	}
}

// vaultChip is the title bar's left end: the vault, and its notes.
func (a *app) vaultChip(c *ui.Context, k *colors) {
	if a.store == nil {
		return
	}
	count := len(a.store.notes)
	chip := ui.Row(c).Gap(9).Padding(3, 8, 3, 6).Radius(9).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Notes in " + a.store.displayDir()).
		MaxWidth(230).Shrink(0)
	if chip.Hovered() || a.notesOpen {
		chip.Background(k.hover)
	}
	if chip.Clicked() {
		a.notesOpen = !a.notesOpen
		if a.notesOpen {
			a.noteQuery = ""
			if a.store != nil {
				_ = a.store.refresh()
			}
		}
	}
	chip.Children(func() {
		ui.Icon(c, icon("folder-open")).Size(19, 19).TextColor(k.textMuted)
		ui.Column(c).MinWidth(0).Children(func() {
			ui.Text(c, "Mojian").FontSize(13.5).FontWeight(700).TextColor(k.text.Alpha(0.85)).
				SingleLine().Ellipsis("…")
			ui.Text(c, noteCount(count)).FontSize(11.5).
				TextColor(k.textFaint).SingleLine().Ellipsis("…")
		})
	})
	was := a.notesOpen
	ui.PopoverBase(c, chip, &a.notesOpen, func(panel ui.Element) {
		panel.Margin(8, 0, 0, 0).Width(330).Padding(12).Radius(14).Background(k.panel).
			Border(1, k.panelBorder).Shadow(0, 12, 32, 0, k.shadowFocused).Gap(10)
		a.notesPanel(c, k)
	})
	if was && !a.notesOpen {
		if t := a.tab(); t != nil {
			a.focusReq = t.Focus
		}
	}
}

// notesPanel lists the vault, to open a note in the focused pane.
func (a *app) notesPanel(c *ui.Context, k *colors) {
	if a.store == nil {
		return
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon("search")).Size(15, 15).TextColor(k.textFaint)
		in := ui.TextInputBase(c, &a.noteQuery).Grow(1).FontSize(13).
			Placeholder("Search notes").AutoFocus()
		if in.Submitted() {
			for _, n := range a.filteredNotes() {
				a.openNote(n)
				return
			}
		}
	})
	notes := a.filteredNotes()
	ui.Scroll(c).MaxHeight(320).Children(func() {
		if len(notes) == 0 {
			ui.Text(c, "No notes match").FontSize(12.5).TextColor(k.textFaint).Padding(10)
		}
		for _, n := range notes {
			n := n
			row := ui.Row(c.Key(n.Rel)).Padding(7, 8).Gap(9).Radius(8).AlignItems(ui.Center).
				Cursor(ui.CursorPointer)
			if row.Hovered() {
				row.Background(k.hover)
			}
			if row.Clicked() {
				a.openNote(n)
				return
			}
			row.Children(func() {
				ui.Icon(c, icon("file-text")).Size(15, 15).TextColor(k.textMuted)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, n.Title).FontSize(13).TextColor(k.text).SingleLine().Ellipsis("…")
					ui.Text(c, fmt.Sprintf("%s · %d words", n.Rel, wordCount(n.text()))).
						FontSize(11).TextColor(k.textFaint).SingleLine().Ellipsis("…")
				})
			})
		}
	})
	ui.Divider(c)
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, noteCount(len(a.store.notes))).FontSize(11.5).TextColor(k.textFaint)
		ui.Spacer(c)
		btn := ui.Row(c).Gap(6).Padding(4, 9).Radius(7).AlignItems(ui.Center).
			Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("New Note")
		if btn.Hovered() {
			btn.Background(k.hover)
		}
		if btn.Clicked() {
			a.notesOpen = false
			a.later(func() { a.newNote() })
		}
		btn.Children(func() {
			ui.Icon(c, icon("plus")).Size(13, 13).TextColor(k.textMuted)
			ui.Text(c, "New Note").FontSize(12).TextColor(k.text)
		})
	})
}

// noteCount names a number of notes.
func noteCount(n int) string {
	if n == 1 {
		return "1 note"
	}
	return fmt.Sprintf("%d notes", n)
}

func (a *app) filteredNotes() []*note {
	if a.store == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(a.noteQuery))
	var out []*note
	for _, n := range a.store.sorted() {
		if q == "" || fuzzy(strings.ToLower(n.Title+" "+n.Rel), q) {
			out = append(out, n)
		}
	}
	return out
}

// shortcuts gives the browser the keys the menus give a native window,
// where it has no menu bar.
func (a *app) shortcuts(c *ui.Context) {
	if runtime.GOOS != "js" || a.paletteOpen || a.notesOpen {
		return
	}
	for _, b := range []struct {
		mods ui.Modifiers
		key  ui.Key
		run  func()
	}{
		{ui.Cmd, ui.KeyN, func() { a.run(cmdNewNote) }},
		{ui.Cmd, ui.KeyT, func() { a.run(cmdNewTab) }},
		{ui.Cmd, ui.KeyS, func() { a.run(cmdSave) }},
		{ui.Cmd | ui.Shift, ui.KeyP, func() { a.run(cmdPalette) }},
		{ui.Cmd | ui.Shift, ui.KeyO, func() { a.run(cmdNotes) }},
		{ui.Cmd | ui.Shift, ui.KeyV, func() { a.run(cmdPreview) }},
		{ui.Cmd | ui.Shift, ui.KeyD, func() { a.run(cmdSplitRight) }},
		{ui.Cmd | ui.Shift, ui.KeyE, func() { a.run(cmdSplitDown) }},
		{ui.Cmd | ui.Shift, ui.KeyM, func() { a.run(cmdZoom) }},
		{ui.Cmd, ui.KeyW, func() { a.run(cmdClosePane) }},
		{ui.Cmd | ui.Shift, ui.KeyW, func() { a.run(cmdCloseTab) }},
		{ui.Cmd | ui.Shift, ui.KeyR, func() { a.run(cmdRenameTab) }},
	} {
		b := b
		c.OnShortcut(b.mods, b.key, b.run)
	}
	for i := 1; i <= 9; i++ {
		i := i
		key := ui.Key(int(ui.Key1) + i - 1)
		c.OnShortcut(ui.Cmd, key, func() { a.showTab(i) })
	}
}
