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
			content := ui.Row(c.Key("workspace-content")).Grow(1).MinHeight(0).Gap(gap).
				Padding(0, gap, gap, gap).AlignItems(ui.Stretch)
			content.Children(func() {
				if a.treeOpen {
					a.sidebar(c, k)
				}
				a.workspace(c, k)
			})
		})
	if a.err != "" {
		a.errorBanner(c, k)
	}
	a.overviewChrome(c, k)
	a.overviewKeys(c)
	a.paneKeys(c)
	a.projectPickerModal(c, k)
	a.splitPicker(c, k)
	a.remotePickerModal(c, k)
	a.sshPicker(c, k)
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
	a.canvasW, a.canvasH = vw, vh
	switch {
	case a.git.Open:
		box.Children(func() { a.gitPage(c, k) })
	case t != nil:
		box.Children(func() { a.tabContent(c, k, t, vw, vh) })
	default:
		box.Children(func() { a.emptyState(c, k) })
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
		a.serverChip(c, k)
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
				a.later(func() { a.openSettingsWindow() })
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

// tabContent lays out the panes of a tab, or the one zoomed. The
// viewport follows the focused pane; Option+Tab animates the same live
// canvas into a fit-all overview.
func (a *app) tabContent(c *ui.Context, k *colors, t *tab, vw, vh float32) {
	if t.Root == nil {
		a.emptyState(c, k)
		return
	}
	if t.Zoom != nil && !a.overview {
		a.paneCard(c, k, t, t.Zoom).Grow(1)
		return
	}
	preview := a.overview || a.overviewLeave
	if !preview {
		t.followFocus(vw, vh)
	}
	cw, ch := t.Root.extent(vw, vh)
	if cw < vw {
		cw = vw
	}
	if ch < vh {
		ch = vh
	}
	scale, x, y := float32(1), -t.panX, -t.panY
	anim := ui.Animation{Duration: 0}
	if preview {
		anim = ui.Animation{Duration: 180 * time.Millisecond, Ease: ui.EaseOut}
	}
	if a.overview {
		scale = min((vw-2*gap)/cw, (vh-2*gap)/ch, 1)
		scale = max(scale, 0.15)
		x, y = (vw-cw*scale)/2, (vh-ch*scale)/2
	}
	inv := float32(1)
	if preview {
		inv = min(1/scale, 2.5)
	}
	viewport := ui.Box(c.Key("canvas-viewport:" + t.ID)).Grow(1).
		MinWidth(0).MinHeight(0).Clip()
	viewport.Children(func() {
		canvas := ui.Box(c.Key("canvas:" + t.ID)).Absolute().Left(0).Top(0).
			Width(cw).Height(ch)
		canvas.Children(func() {
			a.canvasNode(c, k, t, t.Root, vw, vh, preview, inv)
		})
		canvas.Left(-t.panX).Top(-t.panY)
		if preview {
			// The previews are static text, so caching them makes the
			// whole transition a single moving bitmap.
			canvas.Cache()
		}
		// Keep an identity pose in the normal view so the first
		// overview transition has somewhere to animate from.
		pose := ui.Pose{Scale: 1}
		if a.overview {
			pose = ui.Pose{X: x + t.panX, Y: y + t.panY, Scale: scale}
		}
		canvas.TransformOrigin(0, 0).AnimatePose(pose, anim)
	})
}

// canvasNode lays out a split on the canvas. Its children keep their
// declared sizes; a shorter sibling leaves empty canvas rather than
// being stretched.
func (a *app) canvasNode(c *ui.Context, k *colors, t *tab, n *node, vw, vh float32, preview bool, inv float32) ui.Element {
	if n == nil {
		return ui.Box(c)
	}
	if n.Pane != nil {
		p := n.Pane
		w, h := p.width()*vw, p.height()*vh
		if preview {
			return a.overviewPaneCard(c, k, t, p, inv).Width(w).Height(h)
		}
		return a.paneCard(c, k, t, p).Width(w).Height(h)
	}
	aw, ah := n.A.extent(vw, vh)
	bw, bh := n.B.extent(vw, vh)
	key := "split-"
	if p := firstPane(n); p != nil {
		key += p.ID
	}
	var box ui.Element
	var width, height float32
	if n.Vertical {
		width, height = max(aw, bw), ah+gap+bh
		box = ui.Column(c.Key(key)).Width(width).Height(height).AlignItems(ui.Start)
	} else {
		width, height = aw+gap+bw, max(ah, bh)
		box = ui.Row(c.Key(key)).Width(width).Height(height).AlignItems(ui.Start)
	}
	box.Children(func() {
		a.canvasNode(c, k, t, n.A, vw, vh, preview, inv).Width(aw).Height(ah)
		if preview {
			if n.Vertical {
				ui.Box(c.Key("divider")).Width(width).Height(gap)
			} else {
				ui.Box(c.Key("divider")).Width(gap).Height(height)
			}
		} else {
			div := ui.Box(c.Key("divider")).Role(ui.RoleSplitter).Label("Divider")
			if n.Vertical {
				div.Height(gap).Width(width).Cursor(ui.CursorResizeRow)
			} else {
				div.Width(gap).Height(height).Cursor(ui.CursorResizeColumn)
			}
			if dx, dy, ok := div.Dragged(); ok {
				d := dx
				if n.Vertical {
					d = dy
				}
				n.resize(d, vw, vh)
				a.changed()
			}
			if div.DoubleClicked() {
				n.balance()
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
		}
		a.canvasNode(c, k, t, n.B, vw, vh, preview, inv).Width(bw).Height(bh)
	})
	return box
}

// paneCard draws one file: a card with a header over its editor.
func (a *app) paneCard(c *ui.Context, k *colors, t *tab, p *pane) ui.Element {
	focused := t.Focus == p
	card := ui.Column(c.Key(p.ID)).Radius(cardR).Clip().MinWidth(0).MinHeight(0)
	bg, border, shadow := k.Card, k.CardBorder, k.Shadow
	borderW := float32(1)
	if focused {
		bg, border, shadow = k.CardFocused, k.CardBorderFocused, k.ShadowFocused
	}
	card.Background(bg).Border(borderW, border).
		Shadow(0, 1, 2, 0, shadow).
		Shadow(0, 6, 22, -2, shadow)
	card.Transition(motion.Pane)
	hovered := card.Hovered()
	width, height := card.Bounds().W, card.Bounds().H
	overZone := dropNone
	if over, ok := ui.DragOver[*pane](card); ok && over != nil && over != p && t.contains(over) {
		if x, y, inside := card.PointerPosition(); inside {
			overZone = dropZoneAt(x, y, width, height)
		}
	}
	card.Children(func() {
		a.paneHeader(c, k, t, p, focused, hovered, width)
		body := ui.Box(c).Grow(1).MinHeight(0).Padding(0, 0, 6, 0)
		body.Children(func() {
			if p.Term != nil {
				if p.Term.Term == nil {
					a.startRestoredTerminal(p)
					if p.Term.startErr != "" {
						ui.Column(c).Fill().Center().Padding(20).Gap(8).Children(func() {
							ui.Text(c, "Could not open the terminal").FontSize(13).
								FontWeight(600).TextColor(k.Text)
							ui.Text(c, p.Term.startErr).FontSize(11.5).
								TextColor(k.TextFaint).TextAlign(ui.Center).MaxWidth(460)
						})
					} else {
						ui.Text(c, "Attaching…").FontSize(12).TextColor(k.TextFaint)
					}
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
			st.Font = a.editorFont()
			st.LineHeight = float32(a.lineHeight())
			st.TabSize = a.tabSize()
			st.SoftWrap = a.editorSoftWrap || wrapFor(p.Note.Path)
			st.NoActiveLine = a.noActiveLine
			st.NoIndentGuides = a.noIndentGuides
			st.NoAutoClose = a.noAutoClose
			st.NoScrollPastEnd = a.noScrollPastEnd
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
		if overZone != dropNone {
			a.dropIndicator(c, k, overZone)
		}
	})
	if p.dragging {
		card.Border(2, k.Text)
	}
	if from, ok := ui.Drop[*pane](card); ok && from != nil && from != p && t.contains(from) {
		zone := overZone
		if x, y, inside := card.PointerPosition(); inside {
			zone = dropZoneAt(x, y, width, height)
		}
		if zone == dropNone {
			zone = dropCenter
		}
		a.later(func() { a.movePane(t, from, p, zone) })
	}
	if overZone != dropNone {
		card.Border(2, k.Text)
	}
	if card.Pressed() && t.Focus != p {
		t.Focus = p
		if !p.Preview && p.Note != nil {
			a.focusReq = p
		}
		a.changed()
	}
	return card
}

// movePane rearranges the tiling after a pane is dropped on another:
// the middle swaps them, an edge inserts the dragged pane there.
func (a *app) movePane(t *tab, from, to *pane, zone paneDrop) {
	if t == nil || t != a.tab() || !t.movePane(from, to, zone) {
		return
	}
	a.focusReq = from
	a.changed()
	a.invalidate()
}

// dropZoneAt divides a pane into a swap target in the middle and insert
// targets on its edges. Top and bottom are generous: those are the
// directions a stack grows in.
func dropZoneAt(x, y, w, h float32) paneDrop {
	if w <= 0 || h <= 0 {
		return dropCenter
	}
	switch {
	case y < h*0.32:
		return dropAbove
	case y > h*0.68:
		return dropBelow
	case x < w*0.22:
		return dropLeft
	case x > w*0.78:
		return dropRight
	default:
		return dropCenter
	}
}

// dropIndicator draws where the dragged pane would land: a bar on the
// edge and a small label, so dropping above and below is unmistakable.
func (a *app) dropIndicator(c *ui.Context, k *colors, zone paneDrop) {
	if zone == dropNone || zone == dropCenter {
		return
	}
	bar := ui.Box(c.Key("drop-indicator")).Absolute().PassThrough().
		Background(k.Text).Radius(2)
	label := ""
	switch zone {
	case dropAbove:
		bar.Top(2).Left(8).Right(8).Height(3)
		label = "Drop above"
	case dropBelow:
		bar.Bottom(2).Left(8).Right(8).Height(3)
		label = "Drop below"
	case dropLeft:
		bar.Top(8).Bottom(8).Left(2).Width(3)
		label = "Drop left"
	case dropRight:
		bar.Top(8).Bottom(8).Right(2).Width(3)
		label = "Drop right"
	}
	hint := ui.Row(c.Key("drop-hint")).Absolute().PassThrough().Justify(ui.Center)
	switch zone {
	case dropAbove:
		hint.Top(10).Left(0).Right(0)
	case dropBelow:
		hint.Bottom(10).Left(0).Right(0)
	case dropLeft:
		hint.Left(10).Top(0).Bottom(0)
	case dropRight:
		hint.Right(10).Top(0).Bottom(0)
	}
	hint.Children(func() {
		ui.Text(c, label).FontSize(10.5).FontWeight(650).TextColor(k.Panel).
			Padding(3, 8).Radius(6).Background(k.Text)
	})
}

func paneName(p *pane) string {
	switch {
	case p == nil:
		return "Pane"
	case p.Note != nil:
		return p.Note.Title
	case p.Asset != nil:
		return p.Asset.Name
	case p.Term != nil:
		return p.Term.Spec.title()
	default:
		return "Pane"
	}
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
		name, ic = p.Term.Spec.title(), icons.Icon("square-terminal")
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
		handleW, gripSize := float32(15), float32(12)
		if compact {
			iconSize, handleW, gripSize = 14, 12, 10
		}
		grip := ui.Row(c).Width(handleW).Height(headerH).Center().
			Cursor(ui.CursorGrab).Role(ui.RoleButton).Label("Move Pane").
			Description("Drag to move pane").Drag(p)
		if grip.Hovered() {
			grip.Background(k.Hover)
		}
		p.dragging = grip.Dragging()
		grip.Children(func() {
			ui.Icon(c, icons.Icon("grip-vertical")).Size(gripSize, gripSize).
				TextColor(k.TextFaint)
		})
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
				Tooltip(tip("Split Right", cmdSplitRight)).Clicked() {
				a.later(func() {
					t.Focus = p
					a.splitFocused(false)
				})
			}
			if iconButton(c, k, "rows-2", "Split Down", btn, isize).
				Tooltip(tip("Split Down", cmdSplitDown)).Clicked() {
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
		ui.Box(c).Size(64, 54).Radius(15).Background(k.CardFocused).
			Border(1, k.CardBorderFocused).Center().Children(func() {
			ui.Icon(c, brandMark(c)).Size(38, 38)
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
