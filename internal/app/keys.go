package app

import "github.com/egoist/mygo/ui"

// paneKeys moves the focus with Option and the arrows: left and right
// between columns, up and down between panes stacked in the same column.
func (a *app) paneKeys(c *ui.Context) {
	c.OnShortcut(ui.Alt, ui.KeyLeft, func() { a.focusPaneDirection(paneLeft) })
	c.OnShortcut(ui.Alt, ui.KeyRight, func() { a.focusPaneDirection(paneRight) })
	c.OnShortcut(ui.Alt, ui.KeyUp, func() { a.focusPaneDirection(paneUp) })
	c.OnShortcut(ui.Alt, ui.KeyDown, func() { a.focusPaneDirection(paneDown) })
	c.OnShortcut(ui.Alt|ui.Shift, ui.KeyLeft, func() { a.movePaneDirection(paneLeft) })
	c.OnShortcut(ui.Alt|ui.Shift, ui.KeyRight, func() { a.movePaneDirection(paneRight) })
	c.OnShortcut(ui.Alt|ui.Shift, ui.KeyUp, func() { a.movePaneDirection(paneUp) })
	c.OnShortcut(ui.Alt|ui.Shift, ui.KeyDown, func() { a.movePaneDirection(paneDown) })
}

func (a *app) focusPaneDirection(dir paneDirection) {
	t := a.tab()
	if t == nil {
		return
	}
	target := t.paneInDirection(dir)
	if target == nil || target == t.Focus {
		return
	}
	t.Zoom = nil
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
	if !t.movePane(from, target, zone) {
		return
	}
	t.Zoom = nil
	t.Focus = from
	a.focusReq = from
	a.changed()
	a.invalidate()
}
