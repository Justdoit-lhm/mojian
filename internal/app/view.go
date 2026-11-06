package app

import (
	"runtime"
	"time"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"mojian/internal/icons"
	"mojian/internal/motion"
)

const (
	gap     = 8  // between panes, and around them
	titleH  = 44 // the title bar
	cardR   = 12 // the panes' corners
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
			ui.Row(c).Grow(1).MinHeight(0).Gap(gap).Padding(0, gap, gap, gap).
				AlignItems(ui.Stretch).Children(func() {
				if a.treeOpen {
					a.sidebar(c, k)
				}
				a.workspace(c, k)
			})
		})
	if a.err != "" {
		a.errorBanner(c, k)
	}
	a.projectPickerModal(c, k)
	a.splitPicker(c, k)
	a.palette(c, k)
	a.shortcuts(c)
	a.persistLayout(c)
	a.updateTitle()
}

// workspace shows the Git page, the active tab or the empty state in a
// frame keyed by what it shows: changing the key fades the old page out
// and the new one in, so a switch never teleports.
func (a *app) workspace(c *ui.Context, k *colors) {
	key := "workspace:empty"
	build := func() { a.emptyState(c, k) }
	switch {
	case a.settingsOpen:
		key = "workspace:settings"
		build = func() { a.settingsPage(c, k) }
	case a.git.Open:
		key = "workspace:git"
		build = func() { a.gitPage(c, k) }
	case a.tab() != nil:
		t := a.tab()
		key = "workspace:tab:" + t.ID
		build = func() { a.tabContent(c, k, t) }
	}
	ui.Box(c.Key(key)).Grow(1).MinWidth(0).MinHeight(0).Clip().
		Transition(motion.Page).Children(build)
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
		a.projectChip(c, k)
		a.tabStrip(c, k)
		if iconButton(c, k, "plus", "New Tab", 26, 16).
			Tooltip(tip("New Tab", cmdNewTab)).Clicked() {
			a.later(func() { a.newTab() })
		}
		// The title bar between the tabs and the buttons drags the
		// window, and a double click on it zooms the window.
		ui.Box(c).Grow(1).MinWidth(24)
		ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
			if iconButton(c, k, "panel-left", "File Tree", 30, 17).
				Tooltip(tip("File Tree", cmdTree)).Clicked() {
				a.later(func() { a.toggleTree() })
			}
			if iconButton(c, k, "command", "Command Palette", 30, 17).
				Tooltip(tip("Command Palette", cmdPalette)).Clicked() {
				a.openPalette()
			}
			if iconButton(c, k, "git-branch", "Git Diff", 30, 17).
				Tooltip(tip("Git Diff", cmdGit)).Clicked() {
				a.later(func() { a.toggleGit() })
			}
			if runtime.GOOS != "js" {
				if iconButton(c, k, "square-terminal", "Toggle Terminal", 30, 17).
					Tooltip(tip("Toggle Terminal", cmdTerminal)).Clicked() {
					a.later(func() { a.toggleTerminal() })
				}
			}
			if iconButton(c, k, "settings", "Settings", 30, 17).
				Tooltip(tip("Settings", cmdSettings)).Clicked() {
				a.later(func() { a.settingsToggle() })
			}
		})
	})
}

// tip is the tooltip of a button that runs a command: its label and its
// keys.
func tip(label string, cmd *command) string {
	if cmd.keys == "" {
		return label
	}
	return label + "  " + cmd.keys
}

// iconButton is a borderless button of an icon, with a face on hover;
// label names it for screen readers and its tooltip.
func iconButton(c *ui.Context, k *colors, name, label string, size, iconSize float32) ui.Element {
	b := ui.Box(c).Size(size, size).Center().Radius(size / 2.6).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if b.Pressed() {
		b.Background(k.Pressed)
	} else if b.Hovered() {
		b.Background(k.Hover)
	}
	b.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
	col := k.IconMuted
	if b.Hovered() {
		col = k.Text
	}
	b.Children(func() {
		ui.Icon(c, icons.Icon(name)).Size(iconSize, iconSize).TextColor(col)
	})
	return b
}

// tabContent lays out the panes of a tab, or the one zoomed.
func (a *app) tabContent(c *ui.Context, k *colors, t *tab) {
	if t.Root == nil {
		a.emptyState(c, k)
		return
	}
	if t.Zoom != nil {
		a.paneCard(c, k, t, t.Zoom).Grow(1)
		return
	}
	a.node(c, k, t, t.Root).Grow(1)
}

// node lays out a node of the tree of splits.
func (a *app) node(c *ui.Context, k *colors, t *tab, n *node) ui.Element {
	if n == nil {
		return ui.Box(c)
	}
	if n.Pane != nil {
		return a.paneCard(c, k, t, n.Pane)
	}
	key := "split-"
	if p := firstPane(n); p != nil {
		key += p.ID
	}
	var box ui.Element
	if n.Vertical {
		box = ui.Column(c.Key(key))
	} else {
		box = ui.Row(c.Key(key))
	}
	box.MinWidth(0).MinHeight(0).AlignItems(ui.Stretch)
	bounds := box.Bounds()
	first, second, ratio := n.A, n.B, n.Ratio
	box.Children(func() {
		a.node(c, k, t, first).Grow(ratio).Basis(0).MinWidth(0).MinHeight(0)
		div := ui.Box(c.Key("divider")).Role(ui.RoleSplitter).Label("Divider")
		if n.Vertical {
			div.Height(gap).Cursor(ui.CursorResizeRow)
		} else {
			div.Width(gap).Cursor(ui.CursorResizeColumn)
		}
		if dx, dy, ok := div.Dragged(); ok {
			total := bounds.W - gap
			d := dx
			if n.Vertical {
				total, d = bounds.H-gap, dy
			}
			if total > 0 {
				n.Ratio = min(max(n.Ratio+d/total, 0.08), 0.92)
				a.changed()
			}
		}
		if div.DoubleClicked() {
			n.Ratio = 0.5
			a.changed()
		}
		if div.Hovered() || div.Dragging() || div.Pressed() {
			div.Draw(func(p *ui.Painter, r ui.Rect) {
				if n.Vertical {
					p.Fill(ui.Rect{X: r.X + r.W/2 - 18, Y: r.Y + r.H/2 - 1.5, W: 36, H: 3},
						k.IconMuted.Alpha(0.6), 1.5)
				} else {
					p.Fill(ui.Rect{X: r.X + r.W/2 - 1.5, Y: r.Y + r.H/2 - 18, W: 3, H: 36},
						k.IconMuted.Alpha(0.6), 1.5)
				}
			})
		}
		a.node(c, k, t, second).Grow(1 - ratio).Basis(0).MinWidth(0).MinHeight(0)
	})
	return box
}

// paneCard draws one file: a card with a header over its editor.
func (a *app) paneCard(c *ui.Context, k *colors, t *tab, p *pane) ui.Element {
	focused := t.Focus == p
	card := ui.Column(c.Key(p.ID)).Radius(cardR).Clip().MinWidth(0).MinHeight(0)
	bg, border, shadow := k.Card, k.CardBorder, k.Shadow
	if focused {
		bg, border, shadow = k.CardFocused, k.CardBorderFocused, k.ShadowFocused
	}
	card.Background(bg).Border(1, border).
		Shadow(0, 1, 2, 0, shadow).
		Shadow(0, 6, 22, -2, shadow)
	card.Transition(motion.Pane)
	hovered := card.Hovered()
	width := card.Bounds().W
	card.Children(func() {
		a.paneHeader(c, k, t, p, focused, hovered, width)
		body := ui.Box(c).Grow(1).MinHeight(0).Padding(0, 0, 6, 0)
		body.Children(func() {
			if p.Term != nil {
				if p.Term.Term == nil {
					return
				}
				tv := terminal.View(c, p.Term.Term).Fill()
				if a.focusReq == p {
					tv.Focus()
					a.focusReq = nil
				}
				if tv.Focused() && t.Focus != p {
					t.Focus = p
					a.changed()
				}
				if tv.Focused() && p.Term.Exited {
					p.Term.Exited = false
				}
				return
			}
			if p.Asset != nil {
				a.assetBody(c, k, p)
				return
			}
			if p.Note == nil {
				return
			}
			if p.Preview && isMarkdownPath(p.Note.Path) {
				a.markdownBody(c, k, p)
				return
			}
			st := p.state
			if st == nil {
				st = p.Note.newState()
				p.state = st
			}
			st.Theme = a.editorTheme(c)
			st.Font = noteFont
			st.Font.Size = float32(a.fontSize())
			st.Label = p.Note.Title
			if a.hideLineNumbers {
				st.LineNumbers = editor.LineNumbersOff
			} else {
				st.LineNumbers = lineNumbersFor(p.Note.Path)
			}
			st.OnInput = func(ui.InputEvent) bool {
				return false
			}
			tv := editor.View(c, st).Fill()
			if a.focusReq == p {
				tv.Focus()
				a.focusReq = nil
			}
			if tv.Focused() && t.Focus != p {
				t.Focus = p
				a.changed()
			}
		})
	})
	if card.Pressed() && t.Focus != p {
		t.Focus = p
		if !p.Preview && p.Note != nil {
			a.focusReq = p
		}
		a.changed()
	}
	return card
}

// paneHeader draws a note's icon, title and size, its unsaved dot, and
// its buttons.
func (a *app) paneHeader(c *ui.Context, k *colors, t *tab, p *pane, focused, hovered bool, width float32) {
	compact := width > 0 && width < 230
	name := "Untitled"
	ic := icons.FileIcon("")
	switch {
	case p.Note != nil:
		name, ic = p.Note.Title, icons.FileIcon(p.Note.Path)
	case p.Asset != nil:
		name, ic = p.Asset.Name, icons.FileIcon(p.Asset.Path)
	case p.Term != nil:
		name, ic = "Terminal", icons.Icon("square-terminal")
	}
	hpad, hgap := float32(12), float32(7)
	btn, isize := float32(26), float32(16)
	if compact {
		hpad, hgap, btn, isize = 4, 5, 21, 13
	}
	h := ui.Row(c).Height(headerH).Padding(0, 6, 0, hpad).Gap(hgap).
		AlignItems(ui.Center).MinWidth(0)
	if h.DoubleClicked() {
		t.Focus = p
		a.toggleZoom()
	}
	if h.Clicked() {
		t.Focus = p
		if !p.Preview {
			a.focusReq = p
		}
		a.changed()
	}
	h.Children(func() {
		iconSize := float32(16)
		if compact {
			iconSize = 14
		}
		ui.Icon(c, ic).Size(iconSize, iconSize).TextColor(k.Text).Tooltip(name)
		if compact {
			ui.Spacer(c)
			if p.Note != nil && p.Note.dirty() {
				ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).
					Tooltip("Unsaved edits")
			}
		} else {
			ui.Row(c).Grow(1).MinWidth(0).Gap(5).AlignItems(ui.Center).ClipX().Children(func() {
				ui.Text(c, name).FontSize(12.5).FontWeight(600).TextColor(k.Text).
					SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
				if p.Note != nil && p.Note.dirty() {
					ui.Box(c).Size(7, 7).Radius(4).Background(k.Attention).Margin(0, 0, 0, 2).
						Tooltip("Unsaved edits")
				}
			})
		}
		if p.Note != nil && isMarkdownPath(p.Note.Path) {
			glyph, label := "file-text", "Preview"
			if p.Preview {
				glyph, label = "pencil", "Edit"
			}
			if iconButton(c, k, glyph, label, btn, isize).Tooltip(label).Clicked() {
				p.Preview = !p.Preview
				if p.Preview {
					a.focusReq = nil
				} else {
					a.focusReq = p
				}
				a.changed()
			}
		}
		show := focused || hovered
		ui.Row(c).Gap(1).AlignItems(ui.Center).Children(func() {
			if !show {
				return
			}
			if iconButton(c, k, "columns-2", "Split Right", btn, isize).
				Tooltip(tip("Split Right", cmdSplitRight)).Disabled(len(t.panes()) >= maxPanes).Clicked() {
				a.later(func() {
					t.Focus = p
					a.splitFocused(false)
				})
			}
			if iconButton(c, k, "rows-2", "Split Down", btn, isize).
				Tooltip(tip("Split Down", cmdSplitDown)).Disabled(len(t.panes()) >= maxPanes).Clicked() {
				a.later(func() {
					t.Focus = p
					a.splitFocused(true)
				})
			}
			zoom, label := "maximize-2", "Zoom"
			if t.Zoom == p {
				zoom, label = "minimize-2", "Unzoom"
			}
			if iconButton(c, k, zoom, label, btn, isize).Tooltip(tip(label, cmdZoom)).Clicked() {
				t.Focus = p
				a.toggleZoom()
			}
			if iconButton(c, k, "x", "Close Pane", btn, isize+1).
				Tooltip(tip("Close Pane", cmdClosePane)).Clicked() {
				a.laterPane(p)
			}
		})
	})
}

// laterPane closes the pane after the frame that shows it, so the tab and
// its siblings are not changed while they build.
func (a *app) laterPane(p *pane) {
	a.later(func() { a.closePane(p) })
}

// emptyState shows when no note is open.
func (a *app) emptyState(c *ui.Context, k *colors) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(14).Children(func() {
		ui.Box(c).Size(54, 54).Radius(15).Background(k.CardFocused).
			Border(1, k.CardBorderFocused).Center().Children(func() {
			ui.Icon(c, icons.FileIcon("")).Size(28, 28).TextColor(k.TextMuted)
		})
		ui.Text(c, "No file open").FontSize(19).FontWeight(600).TextColor(k.Text)
		ui.Text(c, "Write something, or pick a file in Explorer.").
			FontSize(13).TextColor(k.TextMuted)
		btn := ui.Row(c).Padding(7, 16).Radius(8).Cursor(ui.CursorPointer).
			Role(ui.RoleButton).Label("New File").Background(k.Text)
		if btn.Hovered() {
			btn.Background(k.Text.Alpha(0.86))
		}
		btn.Tooltip(tip("New File", cmdNewNote))
		if btn.Clicked() {
			a.startNewFile()
		}
		btn.Children(func() {
			ui.Text(c, "New File").FontSize(12.5).FontWeight(600).TextColor(k.Panel)
		})
	})
}

// errorBanner shows what went wrong with the vault.
func (a *app) errorBanner(c *ui.Context, k *colors) {
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Bottom(18).Left(0).Right(0).Justify(ui.Center).
			PassThrough().Children(func() {
			ui.Row(c).Gap(10).Padding(8, 10, 8, 14).Radius(12).Background(k.Panel).
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
