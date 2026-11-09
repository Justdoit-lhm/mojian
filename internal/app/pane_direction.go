package app

// focusPaneDirection gives the focus to the pane on the side, and no
// pane there leaves it where it is.
func (a *app) focusPaneDirection(dir paneDirection) {
	t := a.tab()
	if t == nil {
		return
	}
	target := t.paneInDirection(dir)
	if target == nil || target == t.Focus {
		return
	}
	a.unzoom(t)
	t.Focus = target
	a.focusReq = target
	a.changed()
	a.invalidate()
}

func (a *app) movePaneDirection(dir paneDirection) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	target := t.paneInDirection(dir)
	if target == nil || target == t.Focus {
		return
	}
	from := t.Focus
	zone := dropLeft
	switch dir {
	case paneRight:
		zone = dropRight
	case paneUp:
		zone = dropAbove
	case paneDown:
		zone = dropBelow
	}
	a.clearZoom(t)
	if !t.movePane(from, target, zone) {
		return
	}
	t.Focus = from
	a.focusReq = from
	a.changed()
	a.invalidate()
}
