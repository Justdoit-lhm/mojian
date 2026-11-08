package app

import (
	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
	"mojian/internal/motion"
)

const (
	gap     = 8  // between panes, and around them
	titleH  = 44 // the title bar
	headerH = 33 // the panes' headers
)

func (a *app) view(c *ui.Context) {
	a.runPending()
	a.applyTheme(c)
	k := colorsOf(c)
	c.Root().Background(k.BgBottom)
	ui.Column(c).Fill().
		Draw(func(p *ui.Painter, r ui.Rect) { paintBackground(p, r, k) }).
		Children(func() {
			a.titleBar(c, k)
			content := ui.Row(c.Key("workspace-content")).Grow(1).MinHeight(0).Gap(gap).
				Padding(0, gap, gap, gap).AlignItems(ui.Stretch)
			content.Children(func() {
				if a.treeOpen {
					a.sidebar(c, k)
				}
				a.workspaceView(c, k)
			})
			a.closingPanesView(c, k)
		})
	if a.err != "" {
		a.errorBanner(c, k)
	}
	a.paneKeys(c)
	a.projectPickerModal(c, k)
	a.remotePickerModal(c, k)
	a.sshPicker(c, k)
	a.palette(c, k)
	a.quickOpenModal(c, k)
	a.remoteBusyView(c, k)
	if !a.remoteBusy {
		a.shortcuts(c)
	}
	a.persistLayout(c)
	a.updateTitle()
}

// closingPanesView draws the copies of panes that have left the layout
// while their split closes under them.
func (a *app) closingPanesView(c *ui.Context, k *colors) {
	if len(a.closingPanes) == 0 {
		return
	}
	active := a.tab()
	for _, cp := range append([]*closingPane(nil), a.closingPanes...) {
		cp := cp
		if cp.t != active {
			// Switching tabs ends a close that was still running.
			a.later(func() { a.finishClosingPane(cp.p) })
			continue
		}
		ui.Box(c.Key("closing:" + cp.p.ID)).Absolute().
			Left(cp.rect.X).Top(cp.rect.Y).Width(cp.rect.W).Height(cp.rect.H).Clip().
			Children(func() { a.paneCardMode(c, k, cp.t, cp.p, true) })
	}
}

// workspaceView shows the Git page, the active tab or the empty state in
// a frame keyed by what it shows: changing the key fades the old page out
// and the new one in, so a switch never teleports.
func (a *app) workspaceView(c *ui.Context, k *colors) {
	key := "workspace:empty"
	var t *tab
	switch {
	case a.git.Open:
		key = "workspace:git"
	case a.tab() != nil:
		t = a.tab()
		key = "workspace:tab:" + t.ID
	}
	box := ui.Box(c.Key(key)).Grow(1).MinWidth(0).MinHeight(0).Clip().
		Transition(motion.Page)
	vw, vh := box.Bounds().W, box.Bounds().H
	if vw <= 0 {
		vw = 900
	}
	if vh <= 0 {
		vh = 600
	}
	switch {
	case a.git.Open:
		box.Children(func() { a.gitPage(c, k) })
	case t != nil:
		box.Children(func() { a.tabContent(c, k, t, vw, vh) })
	default:
		box.Children(func() { a.emptyState(c, k, vw, vh) })
	}
}

// titleBar draws the room under the window's controls: the vault, the
// tabs, and the buttons of the palette and a new note.
func (a *app) titleBar(c *ui.Context, k *colors) {
	bar := c.TitleBar()
	dragW := bar.Left + 18
	if bar.Left == 0 {
		dragW = 30 // in full screen, or beside no controls
	}
	ui.Row(c).Height(titleH).Padding(0, max(bar.Right, 10), 0, 0).Gap(14).
		AlignItems(ui.Center).DragWindow().Children(func() {
		// The title bar band right of the traffic lights moves the
		// window, and leaves the corner itself to the resize edge.
		ui.Box(c).Width(dragW).Height(titleH).DragWindow().
			Cursor(ui.CursorGrab).Description("Drag Window")
		if iconButton(c, k, "panel-left", "File Tree", 30, 17).
			Tooltip(a.tip("File Tree", cmdTree)).Clicked() {
			a.later(func() { a.toggleTree() })
		}
		if terminalsSupported() {
			a.serverChip(c, k)
		}
		a.projectChip(c, k)
		a.tabStrip(c, k)
		if iconButton(c, k, "plus", "New Tab", 26, 16).
			Tooltip(a.tip("New Tab", cmdNewTab)).Clicked() {
			a.later(func() { a.newTab() })
		}
		// The title bar between the tabs and the buttons drags the
		// window, and a double click on it zooms the window.
		ui.Box(c).Grow(1).MinWidth(24)
		ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
			if iconButton(c, k, "command", "Command Palette", 30, 17).
				Tooltip(a.tip("Command Palette", cmdPalette)).Clicked() {
				a.openPalette()
			}
			if iconButton(c, k, "git-branch", "Git Diff", 30, 17).
				Tooltip(a.tip("Git Diff", cmdGit)).Clicked() {
				a.later(func() { a.toggleGit() })
			}
			if terminalsSupported() {
				if iconButton(c, k, "square-terminal", "Toggle Terminal", 30, 17).
					Tooltip(a.tip("Toggle Terminal", cmdTerminal)).Clicked() {
					a.later(func() { a.toggleTerminal() })
				}
			}
			if iconButton(c, k, "settings", "Settings", 30, 17).
				Tooltip(a.tip("Settings", cmdSettings)).Clicked() {
				a.later(func() { a.openSettingsWindow() })
			}
		})
	})
}

// tip is the tooltip of a button that runs a command: its label and its
// keys.
func (a *app) tip(label string, cmd *command) string {
	keys := a.commandKeys(cmd)
	if keys == "" || keys == "None" {
		return label
	}
	return label + "  " + keys
}

// iconButton is a borderless button of an icon, with a face on hover;
// label names it for screen readers and its tooltip.
func iconButton(c *ui.Context, k *colors, name, label string, size, iconSize float32) ui.Element {
	b := ui.Box(c).Size(size, size).Center().Radius(radius).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if b.Pressed() {
		b.Background(k.Pressed)
	} else if b.Hovered() {
		b.Background(k.Hover)
	}
	b.Transition(motion.Colors)
	col := k.IconMuted
	if b.Hovered() {
		col = k.Text
	}
	b.Children(func() {
		ui.Icon(c, icons.Icon(name)).Size(iconSize, iconSize).TextColor(col)
	})
	return b
}

// tabContent lays out the tab's split tree, or the one zoomed. Each
// split gives its two sides the same share of the space.
func (a *app) tabContent(c *ui.Context, k *colors, t *tab, vw, vh float32) {
	if t.Root == nil {
		a.emptyState(c, k, vw, vh)
		return
	}
	if t.Zoom != nil && t.contains(t.Zoom) {
		a.paneCard(c, k, t, t.Zoom).Grow(1)
		return
	}
	a.nodeView(c, k, t, t.Root)
}

func (a *app) nodeView(c *ui.Context, k *colors, t *tab, n *node) ui.Element {
	if n == nil {
		return ui.Box(c)
	}
	if n.Pane != nil {
		return a.paneCard(c, k, t, n.Pane).Grow(1).MinWidth(0)
	}
	key := "split:"
	if p := firstPane(n); p != nil {
		key += p.ID
	}
	var box ui.Element
	if n.Vertical {
		box = ui.Column(c.Key(key)).Grow(1).MinWidth(0).MinHeight(0).
			Gap(gap).AlignItems(ui.Stretch)
	} else {
		box = ui.Row(c.Key(key)).Grow(1).MinWidth(0).MinHeight(0).
			Gap(gap).AlignItems(ui.Stretch)
	}
	box.Children(func() {
		a.nodeView(c, k, t, n.A).Basis(0).Grow(1).MinWidth(0)
		a.nodeView(c, k, t, n.B).Basis(0).Grow(1).MinWidth(0)
	})
	return box
}

func (a *app) errorBanner(c *ui.Context, k *colors) {
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Bottom(18).Left(0).Right(0).Justify(ui.Center).
			PassThrough().Children(func() {
			ui.Row(c).Gap(10).Padding(8, 10, 8, 14).Radius(radius).Background(k.Panel).
				Border(1, k.PanelBorder).Shadow(0, 8, 24, 0, k.ShadowFocused).
				AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(8, 8).Radius(4).Background(k.Danger)
				ui.Text(c, a.err).FontSize(12.5).TextColor(k.Text).MaxWidth(520)
				if iconButton(c, k, "x", "Dismiss", 22, 14).Clicked() {
					a.err = ""
				}
			})
		})
	})
}
