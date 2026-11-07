package app

// enterSide points a new pane's entrance away from where it lands, so it
// slides in from the edge of the split: a pane on the right starts to the
// right and slides left into place.
func (p *pane) enterSide(side splitSide) {
	const d = float32(28)
	p.enter = true
	switch side {
	case splitLeft:
		p.enterX = -d
	case splitRight:
		p.enterX = d
	case splitUp:
		p.enterY = -d
	case splitDown:
		p.enterY = d
	}
}

// splitWith opens n in a new pane; tests and direct callers that have
// already chosen a file use it.
func (a *app) splitWith(n *note) {
	a.splitWithSide(n, splitRight)
}

// splitWithSide opens n on the named side of the focused pane.
func (a *app) splitWithSide(n *note, side splitSide) {
	t := a.tab()
	if t == nil || t.Focus == nil || n == nil {
		return
	}
	with := newPane(n)
	with.enterSide(side)
	if !t.splitSide(t.Focus, with, side) {
		return
	}
	t.Focus = with
	a.focusReq = with
	a.revealInTree(n)
	a.changed()
	a.invalidate()
}

// emptyPaneSide opens a blank pane on the named side of the focused one,
// ready for the next file opened from the Explorer or the palette.
func (a *app) emptyPaneSide(side splitSide) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	with := &pane{ID: nextID("p")}
	with.enterSide(side)
	if !t.splitSide(t.Focus, with, side) {
		return
	}
	t.Focus = with
	a.focusReq = nil
	a.changed()
	a.invalidate()
}
