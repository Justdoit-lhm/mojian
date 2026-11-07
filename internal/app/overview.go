package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

// overviewScale is how much the workspace shrinks in the pane switcher.
const overviewScale = 0.84

func (a *app) overviewScaleValue() float32 {
	if a.overview && a.tab() != nil {
		return overviewScale
	}
	return 1
}

// toggleOverview opens niri's zoomed-out switcher, or closes it. The
// panes keep running underneath; only their presentation shrinks.
func (a *app) toggleOverview() {
	if a.overview {
		a.closeOverview()
		return
	}
	t := a.tab()
	if t == nil || t.Root == nil || a.git.Open {
		return
	}
	a.paletteOpen = false
	a.projectPicker = false
	a.splitPick = false
	a.sshPick = false
	a.remotePicker = false
	a.overview = true
	a.overviewFocus = true
	a.focusReq = nil
	a.changed()
	a.invalidate()
}

// closeOverview leaves the switcher, keeping the selected pane focused.
func (a *app) closeOverview() {
	if !a.overview {
		return
	}
	a.overview = false
	a.overviewFocus = false
	if t := a.tab(); t != nil {
		focus := t.Focus
		if t.Zoom != nil {
			focus = t.Zoom
		}
		a.focusReq = focus
	}
	a.changed()
	a.invalidate()
}

// selectOverviewPane chooses p, exits the switcher and shows p's view.
func (a *app) selectOverviewPane(t *tab, p *pane) {
	if t == nil || t != a.tab() || p == nil || !t.contains(p) {
		return
	}
	t.Focus = p
	t.Zoom = nil
	a.overview = false
	a.overviewFocus = false
	if p.Note != nil || p.Term != nil {
		a.focusReq = p
	} else {
		a.focusReq = nil
	}
	a.changed()
	a.invalidate()
}

// cycleOverview focuses the next or previous pane, as Tab does in niri's
// overview.
func (a *app) cycleOverview(delta int) {
	t := a.tab()
	if t == nil {
		return
	}
	panes := t.panes()
	if len(panes) == 0 {
		return
	}
	i := 0
	for j, p := range panes {
		if p == t.Focus {
			i = j
			break
		}
	}
	t.Focus = panes[(i+delta+len(panes))%len(panes)]
	a.focusReq = nil
	a.changed()
}

func (a *app) focusOverviewDirection(dir paneDirection) {
	if t := a.tab(); t != nil && t.focusDirection(dir) {
		a.focusReq = nil
		a.changed()
	}
}

// overviewKeys registers the switcher's keys. The focusable pill keeps
// the editor from taking Tab and the arrows while the switcher is open.
func (a *app) overviewKeys(c *ui.Context) {
	c.OnShortcut(ui.Alt, ui.KeyTab, func() { a.toggleOverview() })
	if !a.overview {
		return
	}
	c.OnShortcut(ui.Alt|ui.Shift, ui.KeyTab, func() { a.cycleOverview(-1) })
	c.OnShortcut(0, ui.KeyTab, func() { a.cycleOverview(1) })
	c.OnShortcut(ui.Shift, ui.KeyTab, func() { a.cycleOverview(-1) })
	c.OnShortcut(0, ui.KeyEscape, func() { a.closeOverview() })
	c.OnShortcut(0, ui.KeyEnter, func() {
		if t := a.tab(); t != nil && t.Focus != nil {
			a.selectOverviewPane(t, t.Focus)
		}
	})
	c.OnShortcut(ui.Alt, ui.KeyLeft, func() { a.focusOverviewDirection(paneLeft) })
	c.OnShortcut(ui.Alt, ui.KeyRight, func() { a.focusOverviewDirection(paneRight) })
	c.OnShortcut(ui.Alt, ui.KeyUp, func() { a.focusOverviewDirection(paneUp) })
	c.OnShortcut(ui.Alt, ui.KeyDown, func() { a.focusOverviewDirection(paneDown) })
}

// overviewChrome draws the switcher's hint and takes keyboard focus once
// on entry, so Tab and the arrow keys reach the shortcuts above.
func (a *app) overviewChrome(c *ui.Context, k *colors) {
	if !a.overview {
		return
	}
	t := a.tab()
	if t == nil {
		return
	}
	name, _ := t.label()
	count := len(t.panes())
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Bottom(18).Left(0).Right(0).Justify(ui.Center).
			PassThrough().Children(func() {
			pill := ui.Row(c.Key("overview-pill")).Padding(7, 12).Gap(9).
				AlignItems(ui.Center).Radius(12).Background(k.Panel).
				Border(1, k.PanelBorder).
				Shadow(0, 10, 30, -6, k.ShadowFocused).
				Focusable().FocusRing(false).
				Label("Overview. Tab or arrow keys switch panes; Return focuses; Escape closes.")
			if a.overviewFocus {
				pill.Focus()
				a.overviewFocus = false
			}
			pill.Children(func() {
				ui.Icon(c, icons.Icon("layout-grid")).Size(14, 14).TextColor(k.TextMuted)
				ui.Text(c, "Overview").FontSize(12.5).FontWeight(650).TextColor(k.Text)
				ui.Text(c, name).FontSize(12).TextColor(k.TextFaint).SingleLine().
					Ellipsis("…").MaxWidth(150)
				ui.Box(c).Width(1).Height(13).Background(k.PanelBorder)
				ui.Text(c, fmt.Sprintf("%d %s", count, plural(count, "pane", "panes"))).
					FontSize(11.5).TextColor(k.TextFaint)
				keyHint(c, k, "⇥", "switch")
				keyHint(c, k, "←↑↓→", "move")
				keyHint(c, k, "↵", "focus")
				keyHint(c, k, "esc", "close")
			})
		})
	})
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func keyHint(c *ui.Context, k *colors, key, label string) {
	ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.Text(c, key).FontSize(10.5).FontWeight(600).TextColor(k.TextMuted).
			Padding(1, 5).Radius(4).Background(k.Hover)
		ui.Text(c, label).FontSize(11).TextColor(k.TextFaint)
	})
}
