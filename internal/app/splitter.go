package app

import "github.com/egoist/mygo/ui"

const (
	// minPaneW and minPaneH are the narrowest a divider leaves a pane,
	// in DIPs: enough for the header's buttons and a line of text. The
	// split keeps the share the user chose when the window cannot show
	// both minimums, and takes it back when there is room again.
	minPaneW = 120
	minPaneH = 90
	// splitStep is how far an arrow key moves a focused divider, in DIPs.
	splitStep = 10
	// splitInset is how far the divider's visible line stays from the
	// ends of the gap, so it reads as a handle rather than a rule.
	splitInset = 8
)

// splitShare returns the percent of its room a split gives its first side,
// keeping both panes at least minSize wide when the room allows it. The
// stored share is left alone: a window too small to show it does not
// overwrite the size the user chose.
func splitShare(n *node, room, minSize float32) float32 {
	share := float32(50)
	if n != nil && n.Share > 0 && n.Share < 100 {
		share = n.Share
	}
	if room <= 0 {
		return share
	}
	usable := room - gap
	if usable <= 2*minSize {
		return min(max(share, 1), 99)
	}
	minPct := minSize / usable * 100
	return min(max(share, minPct), 100-minPct)
}

// resizeSplit moves a split's divider by delta percentage points, clamped
// so both panes keep their minimum. It reports whether the share moved.
func (a *app) resizeSplit(n *node, delta, room, minSize float32) bool {
	if n == nil || room <= 0 || delta == 0 {
		return false
	}
	share := splitShare(n, room, minSize) + delta
	usable := room - gap
	if usable > 2*minSize {
		minPct := minSize / usable * 100
		share = min(max(share, minPct), 100-minPct)
	} else {
		share = min(max(share, 1), 99)
	}
	if share == n.Share {
		return false
	}
	n.Share = share
	a.changed()
	return true
}

// splitDivider draws the gap between the two sides of a split as a
// draggable handle: the pointer over it shows a resize cursor, dragging it
// resizes the panes, and the arrows move it once it has the keyboard
// focus. room is the split's size along its axis from the frame before.
func (a *app) splitDivider(c *ui.Context, k *colors, n *node, room float32) {
	minSize := float32(minPaneW)
	if n.Vertical {
		minSize = minPaneH
	}
	d := ui.Box(c).Shrink(0).Focusable().FocusRing(false).
		Role(ui.RoleSplitter).Label("Divider")
	if n.Vertical {
		d.Height(gap).Cursor(ui.CursorResizeNS)
	} else {
		d.Width(gap).Cursor(ui.CursorResizeEW)
	}
	dx, dy, held := d.Dragged()
	if held {
		d.Focus()
		delta := dx
		if n.Vertical {
			delta = dy
		}
		if usable := room - gap; usable > 0 && a.resizeSplit(n, delta/usable*100, room, minSize) {
			// The layout this frame used the share from before the move;
			// draw the new one next.
			a.invalidate()
		}
	}
	back, forth := ui.KeyLeft, ui.KeyRight
	if n.Vertical {
		back, forth = ui.KeyUp, ui.KeyDown
	}
	step := float32(0)
	if usable := room - gap; usable > 0 {
		step = splitStep / usable * 100
	}
	d.OnShortcut(0, back, func() {
		if a.resizeSplit(n, -step, room, minSize) {
			a.invalidate()
		}
	})
	d.OnShortcut(0, forth, func() {
		if a.resizeSplit(n, step, room, minSize) {
			a.invalidate()
		}
	})
	if d.DoubleClicked() && n.Share != 50 {
		n.Share = 50
		a.changed()
		a.invalidate()
	}
	hovered := d.Hovered()
	focused := d.FocusVisible()
	// The draw callback outlives the build pass, so it may not ask the
	// element anything: take what it shows now.
	show := hovered || held || focused
	col := k.TextFaint
	if held || focused {
		col = k.Text
	}
	vertical := n.Vertical
	d.Draw(func(p *ui.Painter, r ui.Rect) {
		if !show {
			return
		}
		if vertical {
			p.Fill(ui.Rect{X: r.X + splitInset, Y: r.Y + r.H/2 - 1, W: max(r.W-2*splitInset, 0), H: 2}, col, 1)
		} else {
			p.Fill(ui.Rect{X: r.X + r.W/2 - 1, Y: r.Y + splitInset, W: 2, H: max(r.H-2*splitInset, 0)}, col, 1)
		}
	})
}
