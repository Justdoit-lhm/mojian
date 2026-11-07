package app

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

// command is an action of the menus, the shortcuts and the palette.
type command struct {
	title string
	id    string
	// accel is the menu's shortcut, keys what the palette shows, and glyph
	// the icon of its row.
	accel, keys, glyph string
	run                func(*app)
}

var (
	cmdOpenProject = &command{title: "Open Project…", id: "open-project", accel: "CmdOrCtrl+O", keys: "⌘O", glyph: "folder-open",
		run: func(a *app) { a.openProjectPicker() }}
	cmdNewNote = &command{title: "New File…", id: "new-file", accel: "CmdOrCtrl+N", keys: "⌘N", glyph: "file-plus",
		run: func(a *app) { a.startNewFile() }}
	cmdNewFolder = &command{title: "New Folder…", id: "new-folder", accel: "CmdOrCtrl+Shift+N", keys: "⇧⌘N", glyph: "folder-plus",
		run: func(a *app) { a.startNewFolder() }}
	cmdRenameNode = &command{title: "Rename…", id: "rename", accel: "F2", keys: "F2", glyph: "pencil",
		run: func(a *app) { a.renameSelection() }}
	cmdTrashNode = &command{title: "Move to Trash", id: "trash", keys: "⌘⌫", glyph: "trash-2",
		run: func(a *app) { a.trashSelection() }}
	cmdNewTab = &command{title: "New Tab", id: "new-tab", accel: "CmdOrCtrl+T", keys: "⌘T", glyph: "plus",
		run: func(a *app) { a.newTab() }}
	cmdSave = &command{title: "Save", id: "save", accel: "CmdOrCtrl+S", keys: "⌘S", glyph: "save",
		run: func(a *app) {
			if err := a.saveNow(); err != nil {
				a.fail(err)
			}
		}}
	cmdSplitLeft = &command{title: "Split Left", id: "split-left", accel: "CmdOrCtrl+Alt+Left", keys: "⌥⌘←", glyph: "columns-2",
		run: func(a *app) { a.splitFocusedSide(splitLeft) }}
	cmdSplitRight = &command{title: "Split Right", id: "split-right", accel: "CmdOrCtrl+Shift+D", keys: "⇧⌘D", glyph: "columns-2",
		run: func(a *app) { a.splitFocusedSide(splitRight) }}
	cmdSplitUp = &command{title: "Split Up", id: "split-up", accel: "CmdOrCtrl+Alt+Up", keys: "⌥⌘↑", glyph: "rows-2",
		run: func(a *app) { a.splitFocusedSide(splitUp) }}
	cmdSplitDown = &command{title: "Split Down", id: "split-down", accel: "CmdOrCtrl+Shift+E", keys: "⇧⌘E", glyph: "rows-2",
		run: func(a *app) { a.splitFocusedSide(splitDown) }}
	cmdZoom = &command{title: "Zoom Pane", id: "zoom", accel: "CmdOrCtrl+Shift+M", keys: "⇧⌘M", glyph: "maximize-2",
		run: func(a *app) { a.toggleZoom() }}
	cmdClosePane = &command{title: "Close Pane", id: "close-pane", accel: "CmdOrCtrl+W", keys: "⌘W", glyph: "x",
		run: func(a *app) { a.closePane(a.focusPane()) }}
	cmdCloseTab = &command{title: "Close Tab", id: "close-tab", accel: "CmdOrCtrl+Shift+W", keys: "⇧⌘W", glyph: "x",
		run: func(a *app) { a.closeTab(a.tab()) }}
	cmdRenameTab = &command{title: "Rename Tab…", id: "rename-tab", accel: "CmdOrCtrl+Shift+R", keys: "⇧⌘R", glyph: "pencil",
		run: func(a *app) { a.startRename(a.tab()) }}
	cmdNextTab = &command{title: "Next Tab", id: "next-tab", accel: "Ctrl+Tab", keys: "⌃⇥", glyph: "chevron-down",
		run: func(a *app) { a.cycleTab(1) }}
	cmdPrevTab = &command{title: "Previous Tab", id: "prev-tab", accel: "Ctrl+Shift+Tab", keys: "⌃⇧⇥", glyph: "chevron-down",
		run: func(a *app) { a.cycleTab(-1) }}
	cmdPalette = &command{title: "Command Palette…", id: "palette", accel: "CmdOrCtrl+Shift+P", keys: "⇧⌘P", glyph: "command",
		run: func(a *app) { a.openPalette() }}
	cmdNotes = &command{title: "Open File…", id: "open-file", accel: "CmdOrCtrl+Shift+O", keys: "⇧⌘O", glyph: "file-text",
		run: func(a *app) { a.openQuickOpen() }}
	cmdTree = &command{title: "Toggle File Tree", id: "toggle-tree", accel: "CmdOrCtrl+Shift+B", keys: "⇧⌘B", glyph: "panel-left",
		run: func(a *app) { a.toggleTree() }}
	cmdTerminal = &command{title: "Toggle Terminal", id: "toggle-terminal", accel: "CmdOrCtrl+Shift+T", keys: "⇧⌘T", glyph: "square-terminal",
		run: func(a *app) { a.toggleTerminal() }}
	cmdGit = &command{title: "Git Diff", id: "git", accel: "CmdOrCtrl+Shift+G", keys: "⇧⌘G", glyph: "git-branch",
		run: func(a *app) { a.toggleGit() }}
	cmdSettings = &command{title: "Settings…", id: "settings", accel: "CmdOrCtrl+,", keys: "⌘,", glyph: "settings",
		run: func(a *app) { openSettingsCommand(a) }}
	cmdSSH = &command{title: "New SSH Terminal…", id: "ssh", accel: "CmdOrCtrl+Shift+L", keys: "⇧⌘L", glyph: "server",
		run: func(a *app) { a.openSSHPicker() }}
)

// paletteCommands are the commands the palette lists.
var paletteCommands = []*command{
	cmdOpenProject, cmdNewNote, cmdNewFolder, cmdNewTab, cmdSave, cmdPalette, cmdNotes, cmdTree, cmdTerminal, cmdGit, cmdSettings, cmdSSH,
	cmdRenameNode, cmdTrashNode,
	cmdSplitLeft, cmdSplitRight, cmdSplitUp, cmdSplitDown, cmdZoom,
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
	a.projectPicker = false
	a.quickOpen = false
	a.paletteOpen = true
	a.paletteQuery = ""
	a.paletteSel = 0
	a.invalidate()
}

// paletteItem is a row of the command palette: a command, a file, or a tab.
type paletteItem struct {
	title, detail, keys, glyph string
	run                        func()
}

func (a *app) paletteItems() []paletteItem {
	var items []paletteItem
	q := strings.ToLower(strings.TrimSpace(a.paletteQuery))
	match := func(s string) bool { return q == "" || fuzzy(strings.ToLower(s), q) }
	for _, c := range paletteCommands {
		if !terminalsSupported() && (c == cmdTerminal || c == cmdSSH) {
			continue
		}
		if runtime.GOOS == "js" && (c == cmdTerminal || c == cmdOpenProject || c == cmdNewFolder) {
			continue
		}
		if !match(c.title) {
			continue
		}
		c := c
		items = append(items, paletteItem{title: c.title, keys: a.commandKeys(c), glyph: c.glyph,
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
		keys := ""
		if i < 9 {
			keys = nativeKeys(fmt.Sprintf("⌘%d", i+1))
		}
		tabDetail := "Tab"
		if detail != "" {
			tabDetail += " " + detail
		}
		items = append(items, paletteItem{title: name, detail: tabDetail, glyph: "layout-grid", keys: keys,
			run: func() { a.selectTab(i); a.updateTitle() }})
	}
	return items
}

// fuzzy reports whether the runes of q appear in s, in order.
func fuzzy(s, q string) bool {
	if strings.Contains(s, q) {
		return true
	}
	want := []rune(q)
	i := 0
	for _, r := range s {
		if i < len(want) && r == want[i] {
			i++
		}
	}
	return i == len(want)
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
		backdrop.Background(k.Backdrop).Justify(ui.Start).Padding(78, 0, 0, 0)
		panel.Width(560).MaxHeight(440).Radius(radius).Background(k.Panel).Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.06)).Clip()
		ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(17, 17).TextColor(k.TextFaint)
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
				ui.Text(c, "No matches").FontSize(13).TextColor(k.TextFaint).Padding(14)
			}
			for i, it := range items {
				it := it
				row := ui.Row(c.Key(i)).Height(34).Padding(0, 10).Gap(10).Radius(radius).
					AlignItems(ui.Center).Cursor(ui.CursorPointer)
				if i == a.paletteSel {
					row.Background(k.PanelSel)
					if reveal {
						row.ScrollIntoView()
					}
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					a.paletteOpen = false
					it.run()
				}
				row.Children(func() {
					ui.Icon(c, icons.Icon(it.glyph)).Size(15, 15).TextColor(k.TextMuted)
					ui.Text(c, it.title).FontSize(13.5).FontWeight(500).TextColor(k.Text).
						SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
					if it.detail != "" {
						ui.Text(c, it.detail).FontSize(13).TextColor(k.TextFaint).
							SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
					}
					ui.Spacer(c)
					if it.keys != "" {
						ui.Text(c, it.keys).FontSize(12).TextColor(k.TextFaint).
							Padding(2, 7).Radius(radius).Background(k.Hover).Shrink(0)
					}
				})
			}
		})
	})
	if was && !a.paletteOpen {
		if t := a.tab(); t != nil && a.focusReq == nil {
			a.focusReq = t.editorPane()
		}
	}
}

func (a *app) filteredNotes() []*note {
	if a.store == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(a.treeQuery))
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
	if runtime.GOOS != "js" || a.paletteOpen {
		return
	}
	for _, cmd := range paletteCommands {
		accel := a.commandAccel(cmd)
		if accel == "" {
			continue
		}
		mods, key, ok := parseAccel(accel)
		if !ok {
			continue
		}
		cmd := cmd
		c.OnShortcut(mods, key, func() { a.run(cmd) })
	}
	for i := 1; i <= 9; i++ {
		i := i
		key := ui.Key(int(ui.Key1) + i - 1)
		c.OnShortcut(ui.Cmd, key, func() { a.showTab(i) })
	}
}
