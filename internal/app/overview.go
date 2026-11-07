package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

// toggleOverview opens the pane overview, or closes it. The canvas is
// shown at a smaller scale as a spatial overview, not as a list.
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
	a.overviewSeq++
	a.overview = true
	a.overviewFocus = true
	a.overviewLeave = false
	a.focusReq = nil
	a.changed()
	a.invalidate()
}

// closeOverview leaves the overview, keeping the selected pane focused.
func (a *app) closeOverview() {
	if !a.overview {
		return
	}
	a.overview = false
	a.overviewFocus = false
	a.focusReq = nil
	a.leaveOverview()
	a.changed()
	a.invalidate()
}

// selectOverviewPane chooses p, exits the overview and shows p's view.
func (a *app) selectOverviewPane(t *tab, p *pane) {
	if t == nil || t != a.tab() || p == nil || !t.contains(p) {
		return
	}
	t.Focus = p
	t.Zoom = nil
	a.overview = false
	a.overviewFocus = false
	a.focusReq = nil
	a.leaveOverview()
	a.changed()
	a.invalidate()
}

// leaveOverview keeps the static previews for the closing animation,
// then reveals the live canvas and gives the keyboard back to the pane.
func (a *app) leaveOverview() {
	a.overviewSeq++
	seq := a.overviewSeq
	a.overviewLeave = true
	win := a.win
	if win == nil {
		a.overviewLeave = false
		a.focusAfterOverview()
		return
	}
	time.AfterFunc(200*time.Millisecond, func() {
		win.Update(func() {
			if seq != a.overviewSeq || a.overview {
				return
			}
			a.overviewLeave = false
			a.focusAfterOverview()
			a.changed()
		})
	})
}

func (a *app) focusAfterOverview() {
	if t := a.tab(); t != nil {
		focus := t.Focus
		if t.Zoom != nil {
			focus = t.Zoom
		}
		a.focusReq = focus
	} else {
		a.focusReq = nil
	}
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
	t := a.tab()
	if t == nil {
		return
	}
	target := a.paneInDirection(dir)
	if target == nil {
		return
	}
	t.Focus = target
	a.focusReq = nil
	a.changed()
}

// paneInDirection prefers the pane that really lies that way on the
// canvas, falling back to the split tree when the geometry is ambiguous.
func (a *app) paneInDirection(dir paneDirection) *pane {
	t := a.tab()
	if t == nil {
		return nil
	}
	vw, vh := a.canvasW, a.canvasH
	if vw <= 0 || vh <= 0 {
		vw, vh = 900, 600
	}
	if target := t.paneAtDirection(dir, vw, vh); target != nil {
		return target
	}
	return t.paneInDirection(dir)
}

// overviewKeys register the overview's own keys. The focusable pill
// keeps the editor from taking Tab while the overview is open.
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
}

// paneKeys moves the selection, and with Shift the selected pane, with
// Option and the arrows. No mouse wheel or scrollbar moves the canvas.
func (a *app) paneKeys(c *ui.Context) {
	for _, k := range []struct {
		key ui.Key
		dir paneDirection
	}{
		{ui.KeyLeft, paneLeft},
		{ui.KeyRight, paneRight},
		{ui.KeyUp, paneUp},
		{ui.KeyDown, paneDown},
	} {
		k := k
		c.OnShortcut(ui.Alt, k.key, func() { a.moveFocusDirection(k.dir) })
		c.OnShortcut(ui.Alt|ui.Shift, k.key, func() { a.moveSelectedPane(k.dir) })
	}
}

func (a *app) moveFocusDirection(dir paneDirection) {
	if a.overview {
		a.focusOverviewDirection(dir)
		return
	}
	t := a.tab()
	if t == nil {
		return
	}
	target := a.paneInDirection(dir)
	if target == nil {
		return
	}
	t.Focus = target
	a.focusReq = t.Focus
	a.changed()
	a.invalidate()
}

// moveSelectedPane moves the pane itself toward a neighbor with Option
// and Shift. The canvas follows the moved pane.
func (a *app) moveSelectedPane(dir paneDirection) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	if a.overview {
		a.focusOverviewDirection(dir)
		return
	}
	target := a.paneInDirection(dir)
	if target == nil || target == t.Focus {
		return
	}
	zone := dropLeft
	switch dir {
	case paneRight:
		zone = dropRight
	case paneUp:
		zone = dropAbove
	case paneDown:
		zone = dropBelow
	}
	from := t.Focus
	if !t.movePane(from, target, zone) {
		return
	}
	a.focusReq = from
	a.changed()
	a.invalidate()
}

// overviewChrome is only the hint and the keyboard focus holder; the
// shrunken panes themselves are the overview.
func (a *app) overviewChrome(c *ui.Context, k *colors) {
	if !a.overview {
		return
	}
	t := a.tab()
	if t == nil || len(t.panes()) == 0 {
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
				Label("Pane overview. Tab or arrow keys switch panes; Return focuses; Escape closes.")
			if a.overviewFocus {
				pill.Focus()
				a.overviewFocus = false
			}
			pill.Children(func() {
				ui.Icon(c, icons.Icon("layout-grid")).Size(14, 14).TextColor(k.TextMuted)
				ui.Text(c, "Panes").FontSize(12.5).FontWeight(650).TextColor(k.Text)
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

// overviewPaneCard is the static, readable stand-in used while the
// canvas animates into and out of the overview. It reads text only for
// the handful of visible preview lines, so nothing live is repainted
// during the transition.
func (a *app) overviewPaneCard(c *ui.Context, k *colors, t *tab, p *pane, inv float32) ui.Element {
	title := paneName(p)
	card := ui.Column(c.Key("preview-"+p.ID)).Radius(cardR).Clip().
		Background(k.Card).Border(1, k.CardBorder)
	if t.Focus == p {
		card.Background(k.CardFocused).Border(2, k.Text)
	} else if card.Hovered() {
		card.Background(k.CardFocused)
	}
	pad := min(max(9*inv, 8), 24)
	head := min(max(headerH*inv, 28), 76)
	titleSize := max(11, 12.5*inv)
	iconSize := min(max(14*inv, 14), 32)
	bodySize := max(9, 10.5*inv)
	card.Children(func() {
		ui.Row(c).Height(head).Padding(0, pad).Gap(min(max(6*inv, 5), 14)).
			AlignItems(ui.Center).Children(func() {
			ic := icons.Icon("file-text")
			switch {
			case p.Term != nil:
				ic = icons.Icon("square-terminal")
			case p.Asset != nil:
				ic = icons.FileIcon(p.Asset.Path)
			case p.Note != nil:
				ic = icons.FileIcon(p.Note.Path)
			}
			ui.Icon(c, ic).Size(iconSize, iconSize).TextColor(k.Text)
			ui.Text(c, title).FontSize(titleSize).FontWeight(600).TextColor(k.Text).
				SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
		})
		switch {
		case p.Asset != nil:
			ui.Box(c).Grow(1).Center().Children(func() {
				ui.Icon(c, icons.FileIcon(p.Asset.Path)).Size(max(36, 44*inv), max(36, 44*inv)).
					TextColor(k.TextFaint)
			})
		default:
			lines := panePreviewText(p, 36)
			if lines == "" {
				lines = " "
			}
			ui.Text(c, lines).Font(noteFont.Family).FontSize(bodySize).
				TextColor(k.TextMuted).NoWrap().Padding(pad).Grow(1)
		}
		hit := ui.Box(c.Key("overview-hit")).Absolute().Fill().
			Background(ui.Transparent).Cursor(ui.CursorPointer).
			Role(ui.RoleButton).Label("Focus pane " + title)
		if hit.Clicked() {
			a.later(func() { a.selectOverviewPane(t, p) })
		}
	})
	return card
}

func panePreviewText(p *pane, maxLines int) string {
	if p == nil {
		return ""
	}
	var lines []string
	switch {
	case p.Note != nil:
		var all []string
		if p.Note.buf != nil {
			snap := p.Note.buf.Snapshot()
			n := min(snap.LineCount(), maxLines)
			for i := 0; i < n; i++ {
				all = append(all, snap.Line(i))
			}
		} else {
			all = strings.SplitN(p.Note.Saved, "\n", maxLines)
		}
		lines = all
	case p.Term != nil:
		if p.Term.Term == nil {
			return "Attaching…"
		}
		text := p.Term.Term.Text()
		all := strings.Split(text, "\n")
		if len(all) > maxLines {
			all = all[len(all)-maxLines:]
		}
		lines = all
	default:
		return ""
	}
	for i, line := range lines {
		r := []rune(line)
		if len(r) > 160 {
			if len(r) > 161 {
				line = string(r[:160]) + "…"
			}
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
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
