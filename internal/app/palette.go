package app

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"

	"mojian/internal/app/commands"
	"mojian/internal/icons"
)

var (
	cmdOpenProject = commands.ByID("open-project")
	cmdNewNote     = commands.ByID("new-file")
	cmdNewFolder   = commands.ByID("new-folder")
	cmdRenameNode  = commands.ByID("rename")
	cmdTrashNode   = commands.ByID("trash")
	cmdNewTab      = commands.ByID("new-tab")
	cmdSave        = commands.ByID("save")
	cmdFocusLeft   = commands.ByID("focus-left")
	cmdFocusRight  = commands.ByID("focus-right")
	cmdFocusUp     = commands.ByID("focus-up")
	cmdFocusDown   = commands.ByID("focus-down")
	cmdMoveLeft    = commands.ByID("move-pane-left")
	cmdMoveRight   = commands.ByID("move-pane-right")
	cmdMoveUp      = commands.ByID("move-pane-up")
	cmdMoveDown    = commands.ByID("move-pane-down")
	cmdSplitLeft   = commands.ByID("split-left")
	cmdSplitRight  = commands.ByID("split-right")
	cmdSplitUp     = commands.ByID("split-up")
	cmdSplitDown   = commands.ByID("split-down")
	cmdZoom        = commands.ByID("zoom")
	cmdClosePane   = commands.ByID("close-pane")
	cmdCloseTab    = commands.ByID("close-tab")
	cmdRenameTab   = commands.ByID("rename-tab")
	cmdNextTab     = commands.ByID("next-tab")
	cmdPrevTab     = commands.ByID("prev-tab")
	cmdPalette     = commands.ByID("palette")
	cmdNotes       = commands.ByID("open-file")
	cmdTree        = commands.ByID("toggle-tree")
	cmdTerminal    = commands.ByID("toggle-terminal")
	cmdGit         = commands.ByID("git")
	cmdSettings    = commands.ByID("settings")
	cmdSSH         = commands.ByID("ssh")
)

// paletteCommands are the commands the palette lists.
var paletteCommands = commands.Palette

// focusPane returns the focused pane of the active tab.
func (a *app) focusPane() *pane {
	if t := a.tab(); t != nil {
		return t.Focus
	}
	return nil
}

func (a *app) openPalette() {
	a.closeModals()
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
		if !commandAvailable(c) {
			continue
		}
		if !match(c.Title) {
			continue
		}
		c := c
		items = append(items, paletteItem{title: c.Title, keys: a.commandKeys(c), glyph: c.Glyph,
			run: func() { a.run(c) }})
	}
	if a.store != nil {
		for _, n := range a.store.Sorted() {
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
			keys = shortcutDisplay(showTabAccel(i + 1))
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
	commandDialog(c, k, &a.paletteOpen, 560, 440, true, func() {
		ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(17, 17).TextColor(k.TextFaint)
			in := ui.TextInputBase(c, &a.paletteQuery).Grow(1).FontSize(15).
				Placeholder("Type a command, or the name of a note…").AutoFocus()
			if listNav(in, &a.paletteSel, len(items), &reveal) {
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
	for _, n := range a.store.Sorted() {
		if q == "" || fuzzy(strings.ToLower(n.Title+" "+n.Rel), q) {
			out = append(out, n)
		}
	}
	return out
}

// shortcuts gives the browser the keys the menus give a native window,
// where it has no menu bar. The harness has no menu either, so it binds
// the same commands in the view.
func (a *app) shortcuts(c *ui.Context) {
	if (runtime.GOOS != "js" && !a.harness) || a.paletteOpen {
		return
	}
	for _, cmd := range paletteCommands {
		if !commandAvailable(cmd) {
			continue
		}
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
		mods, key, ok := parseAccel(showTabAccel(i))
		if !ok {
			continue
		}
		c.OnShortcut(mods, key, func() { a.showTab(i) })
	}
}
