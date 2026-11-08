package app

import (
	"sort"
	"strings"

	"mojian/internal/icons"
	"mojian/internal/motion"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

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
		a.paneHeader(c, k, t, p, width, overlay)
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
							if ui.Button(c, "Try Again").Clicked() {
								p.Term.startErr = ""
								a.startRestoredTerminal(p)
							}
						})
					} else if p.Term.connecting {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Spinner(c).Size(12, 12).Label("Attaching terminal")
							ui.Text(c, "Attaching…").FontSize(12).TextColor(k.TextFaint)
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
			st.SoftWrap = a.EditorSoftWrap || wrapFor(p.Note.Path)
			st.NoActiveLine = a.NoActiveLine
			st.NoIndentGuides = a.NoIndentGuides
			st.NoAutoClose = a.NoAutoClose
			st.NoScrollPastEnd = a.NoScrollPastEnd
			st.Label = p.Note.Title
			if a.HideLineNumbers {
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
			if listNav(in, &p.emptySel, len(choices), &reveal) {
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
				row, clicked := fileChoiceRow(c, k, "open:"+p.ID+":"+ch.rel, ch, i == p.emptySel, 32, 9, 8, 12.5, 11.5, 400)
				if i == p.emptySel && reveal {
					row.ScrollIntoView()
				}
				if clicked {
					a.later(func() { a.openPaneChoice(t, p, ch) })
				}
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
func (a *app) paneHeader(c *ui.Context, k *colors, t *tab, p *pane, width float32, overlay bool) {
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
	// With more than one pane on screen, name each action's accessible
	// label after its pane, so the four visible button sets stay distinct.
	suffix := ""
	if len(t.panes()) > 1 {
		suffix = " " + name
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
			if !overlay && iconButton(c, k, glyph, label+suffix, btn, isize).Tooltip(label).Clicked() {
				p.Preview = !p.Preview
				if p.Preview {
					a.focusReq = nil
				} else {
					a.focusReq = p
				}
				a.changed()
			}
		}
		actions := ui.Row(c).Gap(1).AlignItems(ui.Center).Shrink(0)
		if overlay {
			// Keep the buttons visible while the copy fades, but let no
			// click reach them.
			actions.Disabled(true)
		}
		actions.Children(func() {
			if iconButton(c, k, "columns-2", "Split Right"+suffix, btn, isize).
				Tooltip(a.tip("Split Right", cmdSplitRight)).Clicked() {
				a.later(func() {
					t.Focus = p
					a.splitFocusedSide(splitRight)
				})
			}
			if iconButton(c, k, "rows-2", "Split Down"+suffix, btn, isize).
				Tooltip(a.tip("Split Down", cmdSplitDown)).Clicked() {
				a.later(func() {
					t.Focus = p
					a.splitFocusedSide(splitDown)
				})
			}
			zoom, label := "maximize-2", "Zoom"
			if t.Zoom == p {
				zoom, label = "minimize-2", "Unzoom"
			}
			if iconButton(c, k, zoom, label+suffix, btn, isize).Tooltip(a.tip(label, cmdZoom)).Clicked() {
				t.Focus = p
				a.toggleZoom()
			}
			if iconButton(c, k, "x", "Close Pane"+suffix, btn, isize+1).
				Tooltip(a.tip("Close Pane", cmdClosePane)).Clicked() {
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
