package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

// toggleOverview opens the pane switcher, or closes it. The live canvas
// keeps running underneath; the switcher draws only lightweight tiles.
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
	a.overviewReveal = true
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
	a.overviewReveal = false
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
	a.overviewReveal = false
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
	a.overviewReveal = true
	a.focusReq = nil
	a.changed()
}

func (a *app) focusOverviewDirection(dir paneDirection) {
	if t := a.tab(); t != nil && t.focusDirection(dir) {
		a.overviewReveal = true
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

// overviewChrome draws the switcher as a compact grid of pane cards.
// It is deliberately lighter than the editors and terminals themselves:
// opening it never has to repaint or scale the live canvas.
func (a *app) overviewChrome(c *ui.Context, k *colors) {
	if !a.overview {
		return
	}
	t := a.tab()
	if t == nil {
		return
	}
	panes := t.panes()
	if len(panes) == 0 {
		return
	}
	name, _ := t.label()
	cols := 2
	if len(panes) > 4 {
		cols = 3
	}
	ui.Overlay(c, func() {
		back := ui.Box(c).Fill().Background(k.Backdrop.Alpha(0.55)).Center()
		back.Children(func() {
			panel := ui.Column(c.Key("overview-panel")).WidthPercent(86).MaxWidth(760).
				MaxHeightPercent(82).Padding(18, 20).Gap(12).Radius(16).
				Background(k.Panel).Border(1, k.PanelBorder).
				Shadow(0, 24, 60, -8, k.ShadowFocused).
				Focusable().FocusRing(false)
			if a.overviewFocus {
				panel.Focus()
				a.overviewFocus = false
			}
			panel.Label("Pane overview. Tab or arrow keys switch panes; Return focuses; Escape closes.")
			panel.Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(30, 30).Radius(8).Background(k.Hover).Center().Children(func() {
						ui.Icon(c, icons.Icon("layout-grid")).Size(15, 15).TextColor(k.TextMuted)
					})
					ui.Column(c).Gap(1).Children(func() {
						ui.Text(c, "Panes").FontSize(15).FontWeight(650).TextColor(k.Text)
						ui.Text(c, fmt.Sprintf("%s · %d %s", name, len(panes), plural(len(panes), "pane", "panes"))).
							FontSize(11.5).TextColor(k.TextFaint).SingleLine().Ellipsis("…")
					})
					ui.Spacer(c)
					ui.Text(c, "⌥⇥").FontSize(11).FontWeight(600).TextColor(k.TextMuted).
						Padding(3, 7).Radius(6).Background(k.Hover)
				})
				ui.Scroll(c).MaxHeight(430).Scrollbars(false).Children(func() {
					ui.Grid(c).Columns(cols).Gap(10).Children(func() {
						for _, p := range panes {
							p := p
							a.overviewTile(c, k, t, p)
						}
					})
				})
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					keyHint(c, k, "⇥", "switch")
					keyHint(c, k, "←↑↓→", "move")
					keyHint(c, k, "↵", "focus")
					keyHint(c, k, "esc", "close")
				})
			})
			if panel.PressedOutside() {
				a.later(func() { a.closeOverview() })
			}
		})
	})
}

func (a *app) overviewTile(c *ui.Context, k *colors, t *tab, p *pane) {
	title := paneName(p)
	detail, excerpt := paneOverviewText(p)
	tile := ui.Column(c.Key("overview-"+p.ID)).Padding(10, 11).Gap(6).Radius(10).
		Background(k.Card).Border(1, k.CardBorder).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label("Focus pane " + title)
	if t.Focus == p {
		tile.Background(k.CardFocused).Border(2, k.Text)
		if a.overviewReveal {
			tile.ScrollIntoView()
			a.overviewReveal = false
		}
	} else if tile.Hovered() {
		tile.Background(k.CardFocused)
	}
	if tile.Clicked() {
		a.later(func() { a.selectOverviewPane(t, p) })
	}
	tile.Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			ic := icons.Icon("file-text")
			switch {
			case p.Term != nil:
				ic = icons.Icon("square-terminal")
			case p.Asset != nil:
				ic = icons.FileIcon(p.Asset.Path)
			case p.Note != nil:
				ic = icons.FileIcon(p.Note.Path)
			}
			ui.Icon(c, ic).Size(15, 15).TextColor(k.Text)
			ui.Text(c, title).FontSize(12.5).FontWeight(600).TextColor(k.Text).
				SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
		})
		if detail != "" {
			ui.Text(c, detail).FontSize(10.5).TextColor(k.TextFaint).
				SingleLine().Ellipsis("…")
		}
		if excerpt != "" {
			ui.Text(c, excerpt).FontSize(10).Font(noteFont.Family).TextColor(k.TextMuted).
				SingleLine().Ellipsis("…")
		}
	})
}

func paneOverviewText(p *pane) (detail, excerpt string) {
	if p == nil {
		return "", ""
	}
	switch {
	case p.Note != nil:
		var lines []string
		if p.Note.buf != nil {
			// Read only the first few lines: a large note must not be
			// copied whole just to draw a one-line teaser.
			snap := p.Note.buf.Snapshot()
			n := min(snap.LineCount(), 8)
			for i := 0; i < n; i++ {
				lines = append(lines, snap.Line(i))
			}
		} else {
			lines = strings.SplitN(p.Note.Saved, "\n", 8)
		}
		for _, line := range lines {
			line = strings.TrimSpace(strings.TrimLeft(line, "#>-* "))
			if line != "" {
				excerpt = line
				break
			}
		}
		return p.Note.Rel, excerpt
	case p.Asset != nil:
		return p.Asset.Rel, ""
	case p.Term != nil:
		return "Terminal · project root", ""
	default:
		return "", ""
	}
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
