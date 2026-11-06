package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

const maxPanes = 4 // the most editors one tab shows

// beginSplit asks which file the new editor should show, rather than
// opening a second view of the file already focused.
func (a *app) beginSplit(down bool) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	if len(t.panes()) >= maxPanes {
		a.fail(fmt.Errorf("a tab can show at most %d panes", maxPanes))
		return
	}
	a.git.Open = false
	a.paletteOpen = false
	a.projectPicker = false
	a.splitPick = true
	a.splitDown = down
	a.splitFrom = t.Focus
	a.splitTab = t
	a.splitQuery = ""
	a.splitSel = 0
	a.invalidate()
}

// splitCandidates lists the files the new pane can show. The file in the
// pane being split is left out, so a split always opens another file.
func (a *app) splitCandidates() []*note {
	if a.store == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(a.splitQuery))
	var skip *note
	if a.splitFrom != nil {
		skip = a.splitFrom.Note
	}
	var out []*note
	for _, n := range a.store.sorted() {
		if n == skip {
			continue
		}
		if q == "" || fuzzy(strings.ToLower(n.Title+" "+n.Rel), q) {
			out = append(out, n)
		}
	}
	return out
}

// chooseSplit opens n in a new pane beside or below the pane the split
// began from.
func (a *app) chooseSplit(n *note) {
	t := a.splitTab
	from := a.splitFrom
	a.splitPick = false
	a.splitFrom = nil
	a.splitTab = nil
	if n == nil || t == nil || t != a.tab() || !t.contains(from) {
		return
	}
	if len(t.panes()) >= maxPanes {
		a.fail(fmt.Errorf("a tab can show at most %d panes", maxPanes))
		return
	}
	with := newPane(n)
	if !t.split(from, with, a.splitDown) {
		a.fail(fmt.Errorf("could not split the editor"))
		return
	}
	t.Focus = with
	a.focusReq = with
	a.revealInTree(n)
	a.changed()
}

// splitWith opens n in a new pane; tests and keyboard callers that have
// already chosen a file use it.
func (a *app) splitWith(n *note, down bool) {
	t := a.tab()
	if t == nil || t.Focus == nil || n == nil {
		return
	}
	if len(t.panes()) >= maxPanes {
		a.fail(fmt.Errorf("a tab can show at most %d panes", maxPanes))
		return
	}
	with := newPane(n)
	if !t.split(t.Focus, with, down) {
		return
	}
	t.Focus = with
	a.focusReq = with
	a.revealInTree(n)
	a.changed()
}

// splitPicker draws the file chooser of a split over the window.
func (a *app) splitPicker(c *ui.Context, k *colors) {
	if !a.splitPick {
		return
	}
	notes := a.splitCandidates()
	a.splitSel = min(max(a.splitSel, 0), max(len(notes)-1, 0))
	was := a.splitPick
	reveal := false
	title := "Split Right"
	glyph := "columns-2"
	if a.splitDown {
		title, glyph = "Split Down", "rows-2"
	}
	ui.DialogBase(c, &a.splitPick, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop).Justify(ui.Start).Padding(78, 0, 0, 0)
		panel.Width(540).MaxHeight(450).Radius(16).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.06)).Clip()
		ui.Column(c).Children(func() {
			ui.Row(c).Padding(14, 16).Gap(11).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Box(c).Size(32, 32).Radius(9).Background(k.Hover).Center().Children(func() {
					ui.Icon(c, icons.Icon(glyph)).Size(16, 16).TextColor(k.TextMuted)
				})
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, title).FontSize(14).FontWeight(650).TextColor(k.Text)
					ui.Text(c, "Choose a file for the new pane").
						FontSize(11.5).TextColor(k.TextFaint)
				})
				ui.Spacer(c)
				if iconButton(c, k, "x", "Cancel", 26, 14).Clicked() {
					a.splitPick = false
				}
			})
			ui.Row(c).Padding(10, 16).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(16, 16).TextColor(k.TextFaint)
				in := ui.TextInputBase(c, &a.splitQuery).Grow(1).FontSize(14).
					Placeholder("Type the name of a file…").AutoFocus()
				if in.Changed() {
					a.splitSel, reveal = 0, true
				}
				if in.Shortcut(0, ui.KeyDown) {
					a.splitSel, reveal = min(a.splitSel+1, len(notes)-1), true
				}
				if in.Shortcut(0, ui.KeyUp) {
					a.splitSel, reveal = max(a.splitSel-1, 0), true
				}
				if in.Shortcut(0, ui.KeyEscape) {
					a.splitPick = false
				}
				if in.Submitted() && len(notes) > 0 {
					n := notes[a.splitSel]
					a.later(func() { a.chooseSplit(n) })
				}
			})
			listH := float32(min(max(len(notes), 1)*34+12, 330))
			ui.Scroll(c).Height(listH).Padding(6).Children(func() {
				if len(notes) == 0 {
					ui.Text(c, "No other files in this project").FontSize(13).
						TextColor(k.TextFaint).Padding(14)
					return
				}
				for i, n := range notes {
					i, n := i, n
					row := ui.Row(c.Key("split:"+n.Rel)).Height(34).Padding(0, 10).Gap(9).
						Radius(8).AlignItems(ui.Center).Cursor(ui.CursorPointer).
						Role(ui.RoleButton).Label(n.Title)
					if i == a.splitSel {
						row.Background(k.PanelSel)
						if reveal {
							row.ScrollIntoView()
						}
					} else if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() || row.DoubleClicked() {
						a.later(func() { a.chooseSplit(n) })
					}
					row.Children(func() {
						ui.Icon(c, icons.FileIcon(n.Path)).Size(16, 16).TextColor(k.TextMuted)
						ui.Text(c, n.Title).FontSize(13).FontWeight(500).TextColor(k.Text).
							SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
						ui.Text(c, n.Rel).FontSize(12.5).TextColor(k.TextFaint).
							SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
					})
				}
			})
		})
	})
	if was && !a.splitPick {
		if t := a.tab(); t != nil && a.focusReq == nil {
			a.focusReq = t.editorPane()
		}
	}
}
