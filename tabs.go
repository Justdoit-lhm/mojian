package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	tabH     = 28
	tabMaxW  = 214
	tabMinW  = 112
	tileW    = 20
	tileH    = 15
	tileStep = 5 // how far each tile behind the front one peeks out
)

// tabStrip draws the tabs in a track as wide as they are, which shrinks
// when the title bar has no room. The track is part of the title bar, so
// its edges drag the window.
func (a *app) tabStrip(c *ui.Context, k *colors) {
	if len(a.tabs) == 0 {
		return
	}
	widths := make([]float32, len(a.tabs))
	total := float32(4 + 2*max(len(a.tabs)-1, 0))
	for i, t := range a.tabs {
		widths[i] = tabWidth(c, t)
		total += widths[i]
	}
	// The strip scrolls rather than shrinking tabs into unreadable
	// slivers: every tab keeps its name, and the active one is revealed
	// when it changes.
	track := ui.ScrollHorizontal(c).Basis(total).Shrink(1).MinWidth(0).Height(tabH+4).
		Radius((tabH+4)/2).Background(k.track).Border(0.5, k.trackBorder).
		ClipX().DragWindow().Role(ui.RoleTabList).Label("Tabs").Scrollbars(false).
		TrackScroll(&a.tabScroll)
	tabs := append([]*tab(nil), a.tabs...)
	viewport := track.Bounds().W
	if a.revealTab {
		start := float32(2)
		for i := 0; i < a.active && i < len(widths); i++ {
			start += widths[i] + 2
		}
		end := start
		if a.active >= 0 && a.active < len(widths) {
			end += widths[a.active]
		}
		x := a.tabScroll.X
		switch {
		case viewport <= 0:
			x = start
		case start < x:
			x = start
		case end > x+viewport:
			x = end - viewport
		}
		if a.active <= 0 {
			x = 0
		}
		a.tabScroll.X = max(x, 0)
		a.revealTab = false
	}
	offset := a.tabScroll.X
	if viewport > 0 {
		offset = min(offset, max(total-viewport, 0))
	}
	track.Children(func() {
		ui.Row(c).Height(tabH + 4).Width(total).Padding(2).Gap(2).AlignItems(ui.Center).Children(func() {
			x := float32(2)
			for i, t := range tabs {
				// A tab cut by the left edge shows a blank pill rather
				// than a stray tail of its name.
				whole := viewport <= 0 || x >= offset-0.5
				a.tabItem(c, k, i, t, widths[i], whole)
				x += widths[i] + 2
			}
		})
	})
}

// tabWidth is the room a tab takes when the title bar has it: its tiles,
// its label, and its close button.
func tabWidth(c *ui.Context, t *tab) float32 {
	name, detail := t.label()
	w, _ := c.MeasureText(0, tabLabel(name, detail, ui.Color{}, ui.Color{})...)
	tiles := float32(tileW + tileStep*min(len(t.panes())-1, 2) + 2)
	return min(max(6+tiles+9+w+9+18+10, tabMinW), tabMaxW)
}

// tabLabel is the label of a tab: the name and, fainter, the detail.
func tabLabel(name, detail string, nameColor, detailColor ui.Color) []ui.Span {
	return []ui.Span{
		{Text: name, Size: 13, Weight: 500, Color: nameColor},
		{Text: " " + detail, Size: 12, Weight: 400, Color: detailColor},
	}
}

// tabItem draws one tab: the tiles of its panes and its label.
func (a *app) tabItem(c *ui.Context, k *colors, i int, t *tab, width float32, whole bool) {
	active := i == a.active
	e := ui.Row(c.Key(t.ID)).Height(tabH).Basis(width).Shrink(0).
		Padding(0, 9, 0, 5).Gap(8).AlignItems(ui.Center).Radius(tabH / 2).
		Role(ui.RoleTab).Selected(active).FocusRing(false)
	name, detail := t.label()
	e.Label(name + " " + detail)
	if active {
		e.Background(k.tabActive).Shadow(0, 1, 2, 0, k.shadow).Shadow(0, 2, 8, 0, k.shadow)
	} else if e.Hovered() {
		e.Background(k.hover)
	} else {
		e.Background(k.tabInactive)
	}
	e.Transition(ui.ElementTransition{Colors: true, Position: true, Duration: 160 * time.Millisecond})
	if e.Clicked() {
		if a.gitOpen {
			a.gitOpen = false
			a.invalidate()
		}
		if !active {
			a.active = i
			a.focusReq = t.Focus
			if t.Focus != nil {
				a.revealInTree(t.Focus.Note)
			}
			a.changed()
		}
	}
	if e.DoubleClicked() {
		a.active = i
		a.startRename(t)
	}
	e.ContextMenu(func(m *ui.Menu) { a.tabMenu(m, t) })
	hovered := e.Hovered()
	if whole {
		e.Children(func() {
			a.tiles(c, k, t)
			if a.rename == t {
				a.renameField(c, t, name)
				return
			}
			nameColor, detailColor := k.text, k.textFaint
			if !active {
				nameColor = k.textMuted
			}
			label := ui.Box(c).Grow(1).MinWidth(0).Height(18).ClipX()
			label.Children(func() {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Text(c, name).FontSize(13).FontWeight(500).TextColor(nameColor).
						SingleLine().Shrink(0).MaxWidth(max(width-70, 24))
					if detail != "" {
						ui.Text(c, detail).FontSize(12).TextColor(detailColor).
							SingleLine().Shrink(1).MinWidth(0)
					}
				})
			})
			switch {
			case hovered && len(a.tabs) > 1:
				if iconButton(c, k, "x", "Close Tab", 18, 12).Tooltip("Close Tab").Clicked() {
					a.later(func() { a.closeTab(t) })
				}
			case t.dirty():
				ui.Box(c).Size(7, 7).Radius(4).Background(k.attention).Margin(0, 5, 0, 5).
					Tooltip("Unsaved edits")
			default:
				ui.Box(c).Size(7, 7).Shrink(0)
			}
		})
	}
}

// renameField edits the name of a tab in place: Enter or moving the focus
// away keeps it, Escape goes back, and an empty name follows the file
// again.
func (a *app) renameField(c *ui.Context, t *tab, name string) {
	in := ui.TextInput(c, &a.renameText).Placeholder(name).Grow(1).MinWidth(0).Height(22).
		FontSize(13).AutoFocus()
	done := func(keep bool) {
		if keep {
			t.Name = a.renameText
		}
		a.rename = nil
		a.renameFocus = false
		a.changed()
	}
	switch {
	case in.Shortcut(0, ui.KeyEscape):
		done(false)
	case in.Submitted():
		done(true)
	case in.Focused():
		a.renameFocus = true
	case a.renameFocus:
		done(true) // the focus went elsewhere
	}
}

func (a *app) tabMenu(m *ui.Menu, t *tab) {
	if m.Item("Rename Tab…").Chosen() {
		a.startRename(t)
	}
	if t.Name != "" && m.Item("Reset Name").Chosen() {
		t.Name = ""
		a.changed()
	}
	m.Separator()
	if m.Item("New Tab").Chosen() {
		a.newTab()
	}
	if m.Item("Close Tab").Chosen() {
		a.closeTab(t)
	}
	if m.Item("Close Other Tabs").Disabled(len(a.tabs) < 2).Chosen() {
		for _, other := range append([]*tab(nil), a.tabs...) {
			if other != t {
				a.closeTab(other)
			}
		}
		a.active = 0
		a.focusReq = t.Focus
		a.changed()
	}
}

// tiles draws the tiles of a tab's notes: the focused pane's in front,
// the others peeking out behind it.
func (a *app) tiles(c *ui.Context, k *colors, t *tab) {
	panes := t.panes()
	n := min(len(panes), 3)
	w := float32(tileW + tileStep*(n-1))
	b := ui.Box(c).Width(w).Height(tileH).Shrink(0)
	b.Draw(func(p *ui.Painter, r ui.Rect) {
		for i := n - 1; i >= 0; i-- {
			x := r.X + float32(i)*tileStep
			bg, rim := k.tabActive, k.tileRim
			if i > 0 {
				bg, rim = k.card, k.cardBorderFocused
			}
			p.Fill(ui.Rect{X: x, Y: r.Y, W: tileW, H: r.H}, bg, 4)
			p.Stroke(ui.Rect{X: x, Y: r.Y, W: tileW, H: r.H}, rim, 4, 1)
		}
		s := float32(9)
		p.Icon(icon("file-text"), ui.Rect{
			X: r.X + (tileW-s)/2, Y: r.Y + (r.H-s)/2, W: s, H: s,
		}, k.text)
	})
}
