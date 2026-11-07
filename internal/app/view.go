package app

import (
	"runtime"
	"sort"
	"strings"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

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
				a.workspace(c, k)
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
			Tooltip(tip("File Tree", cmdTree)).Clicked() {
			a.later(func() { a.toggleTree() })
		}
		if terminalsSupported() {
			a.serverChip(c, k)
		}
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
			if iconButton(c, k, "command", "Command Palette", 30, 17).
				Tooltip(tip("Command Palette", cmdPalette)).Clicked() {
				a.openPalette()
			}
			if iconButton(c, k, "git-branch", "Git Diff", 30, 17).
				Tooltip(tip("Git Diff", cmdGit)).Clicked() {
				a.later(func() { a.toggleGit() })
			}
			if terminalsSupported() {
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

// paneCard draws one file: a card with a header over its editor.
func (a *app) paneCard(c *ui.Context, k *colors, t *tab, p *pane) ui.Element {
	return a.paneCardMode(c, k, t, p, false)
}

// paneCardMode draws a pane in the layout, or a copy of one that is
// closing over its old place. The copy takes no input and does not
// animate its place: only its fade and slide-away run.
func (a *app) paneCardMode(c *ui.Context, k *colors, t *tab, p *pane, overlay bool) ui.Element {
	focused := t.Focus == p
	card := ui.Column(c.Key(p.ID)).Radius(radius).Clip().MinWidth(0).MinHeight(0)
	if overlay {
		card.Fill()
	} else {
		if b := card.Bounds(); b.W > 0 && b.H > 0 {
			p.lastBounds = b
		}
		// A pane moved into another split comes from where it showed; its
		// colors follow focus. Terminals skip the match so their grid is
		// not laid out at every width on the way.
		tr := motion.Pane
		if p.Term == nil {
			tr.Match = p.ID
		}
		card.Transition(tr)
	}
	if p.enter {
		// The split is new with the pane, so no Enter can run: start at
		// the split's edge, transparent, and animate both here.
		card.Jump("enter-alpha", 0)
		card.Jump("enter-x", p.enterX)
		card.Jump("enter-y", p.enterY)
		p.enter = false
	}
	ex := card.AnimateTo("enter-x", 0, motion.PaneSlide)
	ey := card.AnimateTo("enter-y", 0, motion.PaneSlide)
	if p.closing {
		if !p.closeStarted {
			card.Jump("close-alpha", 1)
			card.Jump("close-x", 0)
			card.Jump("close-y", 0)
			p.closeStarted = true
		}
		ex += card.AnimateTo("close-x", p.enterX, motion.Close)
		ey += card.AnimateTo("close-y", p.enterY, motion.Close)
		o := card.AnimateTo("close-alpha", 0, motion.Close)
		if o < 1 {
			card.Opacity(o)
		}
		if card.Settled("close-alpha") && card.Settled("close-x") && card.Settled("close-y") {
			a.later(func() { a.finishClosingPane(p) })
		}
	}
	if ex != 0 || ey != 0 {
		card.Translate(ex, ey)
	}
	if !p.closing {
		if o := card.AnimateTo("enter-alpha", 1, motion.FadeIn); o < 1 {
			card.Opacity(o)
		}
	}
	bg, border, shadow := k.Card, k.CardBorder, k.Shadow
	borderW := float32(1)
	if focused {
		bg, border, shadow = k.CardFocused, k.CardBorderFocused, k.ShadowFocused
	}
	card.Background(bg).Border(borderW, border).
		Shadow(0, 1, 2, 0, shadow).
		Shadow(0, 6, 22, -2, shadow)
	hovered := card.Hovered()
	width, height := card.Bounds().W, card.Bounds().H
	overZone := dropNone
	overTarget := false
	if !overlay {
		if over, ok := ui.DragOver[*pane](card); ok && over != nil && over != p && t.contains(over) {
			overTarget = true
			if x, y, inside := card.PointerPosition(); inside && width > 0 && height > 0 {
				overZone = dropZoneAt(x, y, width, height)
			}
		}
	}
	card.Children(func() {
		a.paneHeader(c, k, t, p, focused, hovered, width, overlay)
		body := ui.Box(c).Grow(1).MinHeight(0).Padding(0, 0, 6, 0)
		body.Children(func() {
			if p.Note == nil && p.Asset == nil && p.Term == nil {
				a.emptyPaneBody(c, k, t, p)
				return
			}
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
				if !overlay && a.focusReq == p {
					tv.Focus()
					a.focusReq = nil
				}
				if !overlay && tv.Focused() && t.Focus != p {
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
			if !overlay && a.focusReq == p {
				tv.Focus()
				a.focusReq = nil
			}
			if !overlay && tv.Focused() && t.Focus != p {
				t.Focus = p
				a.changed()
			}
		})
		if overZone != dropNone && overZone != dropCenter {
			a.dropIndicator(c, k, overZone)
		}
	})
	if !overlay {
		if from, ok := ui.Drop[*pane](card); ok && from != nil && from != p && t.contains(from) {
			if x, y, inside := card.PointerPosition(); inside && width > 0 && height > 0 {
				zone := dropZoneAt(x, y, width, height)
				if zone != dropNone && zone != dropCenter {
					a.later(func() { a.movePaneTo(t, from, p, zone) })
				}
			}
		}
	}
	if p.dragging || overTarget {
		card.Border(2, k.Text)
	}
	if !overlay && card.Pressed() && t.Focus != p {
		t.Focus = p
		if !p.Preview && p.Note != nil {
			a.focusReq = p
		}
		a.changed()
	}
	return card
}

// movePaneTo drops from on the named edge of to and inserts it there.
func (a *app) movePaneTo(t *tab, from, to *pane, zone paneDrop) {
	if t == nil || t != a.tab() || !t.movePane(from, to, zone) {
		return
	}
	t.Focus = from
	a.focusReq = from
	a.changed()
	a.invalidate()
}

// dropZoneAt divides a pane into a middle and four edges.
func dropZoneAt(x, y, w, h float32) paneDrop {
	if w <= 0 || h <= 0 {
		return dropCenter
	}
	switch {
	case y < h*0.3:
		return dropAbove
	case y > h*0.7:
		return dropBelow
	case x < w*0.25:
		return dropLeft
	case x > w*0.75:
		return dropRight
	default:
		return dropCenter
	}
}

// dropIndicator paints the edge that will take the dragged pane.
func (a *app) dropIndicator(c *ui.Context, k *colors, zone paneDrop) {
	bar := ui.Box(c.Key("drop-indicator")).Absolute().PassThrough().
		Background(k.Text).Radius(2)
	switch zone {
	case dropAbove:
		bar.Top(2).Left(8).Right(8).Height(3)
	case dropBelow:
		bar.Bottom(2).Left(8).Right(8).Height(3)
	case dropLeft:
		bar.Top(8).Bottom(8).Left(2).Width(3)
	case dropRight:
		bar.Top(8).Bottom(8).Right(2).Width(3)
	}
}

// emptyPaneBody is the file picker inside a blank split: type to filter,
// Enter or a click opens the file in this pane.
func (a *app) emptyPaneBody(c *ui.Context, k *colors, t *tab, p *pane) {
	choices := a.paneChoices(p.emptyQuery)
	p.emptySel = min(max(p.emptySel, 0), max(len(choices)-1, 0))
	reveal := false
	ui.Column(c).Fill().Padding(14, 16).Gap(8).Children(func() {
		ui.Text(c, "Open a file in this pane").FontSize(12.5).
			FontWeight(600).TextColor(k.Text)
		ui.Row(c).Height(34).Padding(0, 9).Gap(7).Radius(radius).
			Background(k.Hover).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(14, 14).TextColor(k.TextFaint)
			in := ui.TextInputBase(c, &p.emptyQuery).Grow(1).FontSize(13).
				Placeholder("Type a file name…")
			if t.Focus == p {
				in.AutoFocus()
			}
			if in.Focused() && t.Focus != p {
				t.Focus = p
				a.changed()
			}
			if in.Changed() {
				p.emptySel, reveal = 0, true
			}
			if in.Shortcut(0, ui.KeyDown) {
				p.emptySel, reveal = min(p.emptySel+1, len(choices)-1), true
			}
			if in.Shortcut(0, ui.KeyUp) {
				p.emptySel, reveal = max(p.emptySel-1, 0), true
			}
			if in.Submitted() && len(choices) > 0 {
				ch := choices[p.emptySel]
				a.later(func() { a.openPaneChoice(t, p, ch) })
			}
		})
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(2).Children(func() {
			if len(choices) == 0 {
				ui.Text(c, "No files match").FontSize(12).TextColor(k.TextFaint).Padding(10)
				return
			}
			for i, ch := range choices {
				i, ch := i, ch
				row := ui.Row(c.Key("open:"+p.ID+":"+ch.rel)).Height(32).
					Padding(0, 9).Gap(8).Radius(radius).AlignItems(ui.Center).
					Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(ch.title)
				if i == p.emptySel {
					row.Background(k.PanelSel)
					if reveal {
						row.ScrollIntoView()
					}
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					a.later(func() { a.openPaneChoice(t, p, ch) })
				}
				row.Children(func() {
					ui.Icon(c, icons.FileIcon(ch.rel)).Size(15, 15).TextColor(k.TextMuted)
					ui.Text(c, ch.title).FontSize(12.5).TextColor(k.Text).
						SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
					ui.Text(c, ch.rel).FontSize(11.5).TextColor(k.TextFaint).
						SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
				})
			}
		})
	})
}

type fileChoice struct {
	title string
	rel   string
	note  *note
	asset *assetFile
}

// paneChoices lists the files a blank pane can open, filtered by q.
func (a *app) paneChoices(q string) []fileChoice {
	if a.store == nil {
		return nil
	}
	q = strings.ToLower(strings.TrimSpace(q))
	match := func(s string) bool { return q == "" || fuzzy(strings.ToLower(s), q) }
	var out []fileChoice
	for _, n := range a.store.sorted() {
		if !match(n.Title + " " + n.Rel) {
			continue
		}
		out = append(out, fileChoice{title: n.Title, rel: n.Rel, note: n})
	}
	var assets []*assetFile
	for _, f := range a.store.assets {
		assets = append(assets, f)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Rel < assets[j].Rel })
	for _, f := range assets {
		if !match(f.Name + " " + f.Rel) {
			continue
		}
		out = append(out, fileChoice{title: f.Name, rel: f.Rel, asset: f})
	}
	return out
}

func (a *app) openPaneChoice(t *tab, p *pane, ch fileChoice) {
	if t == nil || t != a.tab() || p == nil || !t.contains(p) {
		return
	}
	t.Focus = p
	switch {
	case ch.note != nil:
		a.openNote(ch.note)
	case ch.asset != nil:
		a.openAsset(ch.asset)
	}
}

func paneName(p *pane) string {
	switch {
	case p == nil:
		return "New Pane"
	case p.Note != nil:
		return p.Note.Title
	case p.Asset != nil:
		return p.Asset.Name
	case p.Term != nil:
		return p.Term.Spec.title()
	default:
		return "New Pane"
	}
}

// paneHeader draws a note's icon, title, its unsaved dot, and its
// buttons.
func (a *app) paneHeader(c *ui.Context, k *colors, t *tab, p *pane, focused, hovered bool, width float32, overlay bool) {
	compact := width > 0 && width < 230
	name := "New Pane"
	ic := icons.Icon("file-plus")
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
	if !overlay && h.DoubleClicked() {
		t.Focus = p
		a.toggleZoom()
	}
	if !overlay && h.Clicked() {
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
			Cursor(ui.CursorGrab).Role(ui.RoleButton).Label("Move Pane " + name)
		if !overlay {
			grip.Drag(p)
		}
		if grip.Hovered() {
			grip.Background(k.Hover)
		}
		p.dragging = grip.Dragging()
		grip.Children(func() {
			ui.Icon(c, icons.Icon("grip-vertical")).Size(gripSize, gripSize).TextColor(k.TextFaint)
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
			if !overlay && iconButton(c, k, glyph, label, btn, isize).Tooltip(label).Clicked() {
				p.Preview = !p.Preview
				if p.Preview {
					a.focusReq = nil
				} else {
					a.focusReq = p
				}
				a.changed()
			}
		}
		// A preview has no editor to take the focus or show a caret, so
		// its controls stay visible even when another pane has the focus.
		preview := p.Asset != nil || p.Note != nil && p.Preview
		show := focused || hovered || preview
		actions := ui.Row(c).Gap(1).AlignItems(ui.Center)
		if overlay {
			// Keep the buttons visible while the copy fades, but let no
			// click reach them.
			actions.Disabled(true)
		}
		actions.Children(func() {
			if !show {
				return
			}
			if iconButton(c, k, "columns-2", "Split Right", btn, isize).
				Tooltip(tip("Split Right", cmdSplitRight)).Clicked() {
				a.later(func() {
					t.Focus = p
					a.splitFocusedSide(splitRight)
				})
			}
			if iconButton(c, k, "rows-2", "Split Down", btn, isize).
				Tooltip(tip("Split Down", cmdSplitDown)).Clicked() {
				a.later(func() {
					t.Focus = p
					a.splitFocusedSide(splitDown)
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

// closingPane is a pane that has left the layout and is fading over the
// place it showed while the split closes under it.
type closingPane struct {
	p    *pane
	t    *tab
	rect ui.Rect
}

// emptyState is the welcome page of a tab with no file open: quick
// actions, then the files touched most recently.
func (a *app) emptyState(c *ui.Context, k *colors, vw, vh float32) {
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().MinHeight(vh).Justify(ui.Center).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(560).AlignSelf(ui.Center).
				Padding(28, 32, 36, 32).Gap(20).Children(func() {
				ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(58, 48).Radius(radius).Background(k.CardFocused).
						Border(1, k.CardBorderFocused).Center().Children(func() {
						ui.Icon(c, brandMark(c)).Size(34, 34)
					})
					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, "Welcome to Mojian").FontSize(22).FontWeight(650).
							TextColor(k.Text)
						ui.Text(c, "A quiet editor for the files in your project.").
							FontSize(12.5).TextColor(k.TextMuted)
					})
				})

				actions := ui.Column(c).FillWidth().Background(k.Card).
					Border(1, k.CardBorder).Radius(radius).Clip()
				actions.Children(func() {
					a.welcomeAction(c, k, "file-plus", "New File…", cmdNewNote, func() { a.startNewFile() })
					ui.Divider(c)
					a.welcomeAction(c, k, "search", "Open File…", cmdNotes, func() { a.openQuickOpen() })
					if runtime.GOOS != "js" {
						ui.Divider(c)
						a.welcomeAction(c, k, "folder-open", "Open Project…", cmdOpenProject, func() { a.openProjectPicker() })
						if terminalsSupported() {
							ui.Divider(c)
							a.welcomeAction(c, k, "square-terminal", "Open Terminal", cmdTerminal, func() { a.toggleTerminal() })
						}
					}
				})

				notes := a.welcomeNotes()
				if len(notes) > 0 {
					ui.Text(c, "RECENT FILES").FontSize(10).FontWeight(650).
						TextColor(k.TextFaint).Padding(2, 2)
					recent := ui.Column(c).FillWidth().Background(k.Card).
						Border(1, k.CardBorder).Radius(radius).Clip()
					recent.Children(func() {
						for i, n := range notes {
							n := n
							if i > 0 {
								ui.Divider(c)
							}
							row := ui.Row(c).Key("welcome:"+n.Rel).Height(40).Padding(0, 14).
								Gap(9).AlignItems(ui.Center).Cursor(ui.CursorPointer).
								Role(ui.RoleButton).Label(n.Title)
							if row.Hovered() {
								row.Background(k.Hover)
							}
							if row.Clicked() {
								a.later(func() { a.openNote(n) })
							}
							row.Children(func() {
								ui.Icon(c, icons.FileIcon(n.Path)).Size(15, 15).TextColor(k.TextMuted)
								ui.Text(c, n.Title).FontSize(12.5).TextColor(k.Text).
									SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
								ui.Text(c, n.Rel).FontSize(11).TextColor(k.TextFaint).
									SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
							})
						}
					})
				}

				ui.Text(c, nativeKeys("⌘,")+" Settings     "+nativeKeys("⇧⌘P")+" Commands     "+nativeKeys("⌘O")+" Open Project").
					FontSize(11).TextColor(k.TextFaint).TextAlign(ui.Center)
			})
		})
	})
}

func (a *app) welcomeAction(c *ui.Context, k *colors, glyph, title string, cmd *command, run func()) {
	row := ui.Row(c).Height(42).Padding(0, 14).Gap(10).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(title)
	if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		run()
	}
	row.Children(func() {
		ui.Icon(c, icons.Icon(glyph)).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, title).FontSize(12.5).TextColor(k.Text).Grow(1)
		if hint := a.commandKeys(cmd); hint != "" {
			ui.Text(c, hint).FontSize(11).TextColor(k.TextFaint)
		}
	})
}

func (a *app) welcomeNotes() []*note {
	if a.store == nil {
		return nil
	}
	notes := a.store.sorted()
	if len(notes) > 5 {
		notes = notes[:5]
	}
	return notes
}

// errorBanner shows what went wrong with the vault.
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
