package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
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
	a.md.Theme = a.markdownTheme(c)
	c.Root().Background(k.bgBottom)
	ui.Column(c).Fill().
		Draw(func(p *ui.Painter, r ui.Rect) { paintBackground(p, r, k) }).
		Children(func() {
			a.titleBar(c, k)
			ui.Column(c).Grow(1).MinHeight(0).Padding(0, gap, gap, gap).Children(func() {
				if t := a.tab(); t != nil {
					a.tabContent(c, k, t)
				} else {
					a.emptyState(c, k)
				}
			})
		})
	if a.err != "" {
		a.errorBanner(c, k)
	}
	a.palette(c, k)
	a.shortcuts(c)
	a.autosave(c)
	a.updateTitle()
}

// titleBar draws the room under the window's controls: the vault, the
// tabs, and the buttons of the palette and a new note.
func (a *app) titleBar(c *ui.Context, k *colors) {
	bar := c.TitleBar()
	left := bar.Left
	if left == 0 {
		left = 12 // in full screen, or beside no controls
	} else {
		left += 14
	}
	ui.Row(c).Height(titleH).Padding(0, max(bar.Right, 10), 0, left).Gap(14).
		AlignItems(ui.Center).DragWindow().Children(func() {
		a.vaultChip(c, k)
		a.tabStrip(c, k)
		// The title bar between the tabs and the buttons drags the
		// window, and a double click on it zooms the window.
		ui.Box(c).Grow(1).MinWidth(24)
		ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
			if iconButton(c, k, "command", "Command Palette", 30, 17).
				Tooltip(tip("Command Palette", cmdPalette)).Clicked() {
				a.openPalette()
			}
			if iconButton(c, k, "plus", "New Note", 30, 19).
				Tooltip(tip("New Note", cmdNewNote)).Clicked() {
				a.later(func() { a.newNote() })
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
		b.Background(k.pressed)
	} else if b.Hovered() {
		b.Background(k.hover)
	}
	b.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
	col := k.iconMuted
	if b.Hovered() {
		col = k.text
	}
	b.Children(func() {
		ui.Icon(c, icon(name)).Size(iconSize, iconSize).TextColor(col)
	})
	return b
}

// tabContent lays out the panes of a tab, or the one zoomed.
func (a *app) tabContent(c *ui.Context, k *colors, t *tab) {
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
						k.iconMuted.Alpha(0.6), 1.5)
				} else {
					p.Fill(ui.Rect{X: r.X + r.W/2 - 1.5, Y: r.Y + r.H/2 - 18, W: 3, H: 36},
						k.iconMuted.Alpha(0.6), 1.5)
				}
			})
		}
		a.node(c, k, t, second).Grow(1 - ratio).Basis(0).MinWidth(0).MinHeight(0)
	})
	return box
}

// paneCard draws one note: a card with a header over its editor or its
// preview.
func (a *app) paneCard(c *ui.Context, k *colors, t *tab, p *pane) ui.Element {
	focused := t.Focus == p
	card := ui.Column(c.Key(p.ID)).Radius(cardR).Clip().MinWidth(0).MinHeight(0)
	bg, border, shadow := k.card, k.cardBorder, k.shadow
	if focused {
		bg, border, shadow = k.cardFocused, k.cardBorderFocused, k.shadowFocused
	}
	card.Background(bg).Border(1, border).
		Shadow(0, 1, 2, 0, shadow).
		Shadow(0, 6, 22, -2, shadow)
	card.Transition(ui.ElementTransition{Colors: true, Duration: 160 * time.Millisecond})
	hovered := card.Hovered()
	card.Children(func() {
		a.paneHeader(c, k, t, p, focused, hovered)
		body := ui.Box(c).Grow(1).MinHeight(0).Padding(0, 5, 6, 5)
		body.Children(func() {
			if p.Note == nil {
				return
			}
			if p.Preview {
				ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
					ui.Column(c).FillWidth().Padding(14, 16, 24, 16).Children(func() {
						markdown.View(c, p.Note.text(), &a.md)
					})
				})
				return
			}
			st := p.Note.editorState()
			st.Theme = a.editorTheme(c)
			st.Font = noteFont
			st.Label = p.Note.Title
			note := p.Note
			st.OnInput = func(ui.InputEvent) bool {
				note.lastEdit = time.Now()
				a.scheduleSave()
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
		a.focusReq = p
		a.changed()
	}
	return card
}

// paneHeader draws a note's icon, title and size, its unsaved dot, and
// its buttons.
func (a *app) paneHeader(c *ui.Context, k *colors, t *tab, p *pane, focused, hovered bool) {
	name, detail := "Untitled", ""
	if p.Note != nil {
		name = p.Note.Title
		if p.Preview {
			detail = "Preview"
		} else {
			detail = fmt.Sprintf("%d words", wordCount(p.Note.text()))
		}
	}
	h := ui.Row(c).Height(headerH).Padding(0, 8, 0, 12).Gap(7).
		AlignItems(ui.Center).MinWidth(0)
	if h.DoubleClicked() {
		t.Focus = p
		a.toggleZoom()
	}
	if h.Clicked() {
		t.Focus = p
		a.focusReq = p
		a.changed()
	}
	h.Children(func() {
		ui.Icon(c, icon("file-text")).Size(14.5, 14.5).TextColor(k.text)
		ui.Row(c).Grow(1).MinWidth(0).Gap(5).AlignItems(ui.Center).ClipX().Children(func() {
			ui.Text(c, name).FontSize(12.5).FontWeight(600).TextColor(k.text).
				SingleLine().Ellipsis("…").Shrink(0).MaxWidthPercent(80)
			if detail != "" {
				ui.Text(c, detail).FontSize(12.5).FontWeight(500).TextColor(k.text.Alpha(0.86)).
					SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
			}
			if p.Note != nil && p.Note.dirty() {
				ui.Box(c).Size(7, 7).Radius(4).Background(k.attention).Margin(0, 0, 0, 2).
					Tooltip("Unsaved edits")
			}
		})
		show := focused || hovered
		ui.Row(c).Gap(1).AlignItems(ui.Center).Children(func() {
			if !show {
				return
			}
			mode, label := "eye", "Preview"
			if p.Preview {
				mode, label = "pencil", "Edit"
			}
			if iconButton(c, k, mode, label, 26, 16).Tooltip(tip(label, cmdPreview)).Clicked() {
				t.Focus = p
				p.Preview = !p.Preview
				a.focusReq = p
				a.changed()
			}
			if iconButton(c, k, "columns-2", "Split Right", 26, 16).
				Tooltip(tip("Split Right", cmdSplitRight)).Clicked() {
				a.later(func() {
					t.Focus = p
					a.split(false)
				})
			}
			if iconButton(c, k, "rows-2", "Split Down", 26, 16).
				Tooltip(tip("Split Down", cmdSplitDown)).Clicked() {
				a.later(func() {
					t.Focus = p
					a.split(true)
				})
			}
			zoom, label := "maximize-2", "Zoom"
			if t.Zoom == p {
				zoom, label = "minimize-2", "Unzoom"
			}
			if iconButton(c, k, zoom, label, 26, 16).Tooltip(tip(label, cmdZoom)).Clicked() {
				t.Focus = p
				a.toggleZoom()
			}
			if iconButton(c, k, "x", "Close Pane", 26, 17).
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
	ui.Column(c).Fill().Center().Gap(14).Children(func() {
		ui.Box(c).Size(54, 54).Radius(15).Background(k.cardFocused).
			Border(1, k.cardBorderFocused).Center().Children(func() {
			ui.Icon(c, icon("file-text")).Size(26, 26).TextColor(k.textMuted)
		})
		ui.Text(c, "No note open").FontSize(19).FontWeight(600).TextColor(k.text)
		ui.Text(c, "Write something, or open a note from the title bar.").
			FontSize(13).TextColor(k.textMuted)
		if ui.PrimaryButton(c, "New Note").Clicked() {
			a.newNote()
		}
	})
}

// errorBanner shows what went wrong with the vault.
func (a *app) errorBanner(c *ui.Context, k *colors) {
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Bottom(18).Left(0).Right(0).Justify(ui.Center).
			PassThrough().Children(func() {
			ui.Row(c).Gap(10).Padding(8, 10, 8, 14).Radius(12).Background(k.panel).
				Border(1, k.panelBorder).Shadow(0, 8, 24, 0, k.shadowFocused).
				AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(8, 8).Radius(4).Background(ui.Hex("#ef4444"))
				ui.Text(c, a.err).FontSize(12.5).TextColor(k.text).MaxWidth(520)
				if iconButton(c, k, "x", "Dismiss", 22, 14).Clicked() {
					a.err = ""
				}
			})
		})
	})
}
