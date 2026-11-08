package app

// splitWith opens n in a new pane; tests use it to set up a layout
// without going through a blank pane and the file picker.
func (a *app) splitWith(n *note) {
	a.splitWithSide(n, splitRight)
}

// splitWithSide opens n on the named side of the focused pane.
func (a *app) splitWithSide(n *note, side splitSide) {
	t := a.tab()
	if t == nil || t.Focus == nil || !a.loadNote(n) {
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
