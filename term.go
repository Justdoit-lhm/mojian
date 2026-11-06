package main

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

const (
	maxTerms = 4
	termW    = 360
)

// termTab is one shell in the terminal panel.
type termTab struct {
	ID     int
	Title  string
	Dir    string
	Term   *terminal.Terminal
	Exited bool
}

func (a *app) activeTermTab() *termTab {
	if a.activeTerm >= 0 && a.activeTerm < len(a.terms) {
		return a.terms[a.activeTerm]
	}
	return nil
}

// terminalDir is the folder a new shell starts in: the selected folder,
// the selected file's folder, or the project root.
func (a *app) terminalDir() string {
	if a.store == nil || a.store.memory {
		return ""
	}
	rel := a.selectedDir()
	if rel == "" {
		return a.store.dir
	}
	if abs, err := a.store.absRel(rel); err == nil {
		return abs
	}
	return a.store.dir
}

func (a *app) toggleTerminal() {
	if runtime.GOOS == "js" {
		a.fail(errors.New("terminals need the desktop app"))
		return
	}
	if a.termOpen {
		a.termOpen = false
		a.termFocus = false
		a.invalidate()
		return
	}
	a.termOpen = true
	if len(a.terms) == 0 {
		a.newTerminal()
		return
	}
	a.termFocus = true
	a.invalidate()
}

func (a *app) newTerminal() {
	if runtime.GOOS == "js" {
		a.fail(errors.New("terminals need the desktop app"))
		return
	}
	if a.store == nil || a.store.memory {
		a.fail(errors.New("terminals need a project on disk"))
		return
	}
	if len(a.terms) >= maxTerms {
		a.fail(fmt.Errorf("at most %d terminals", maxTerms))
		return
	}
	dir := a.terminalDir()
	a.termSeq++
	t := &termTab{ID: a.termSeq, Title: fmt.Sprintf("Terminal %d", a.termSeq), Dir: dir}
	win := a.win
	update := func(fn func()) {
		if win != nil {
			win.Update(fn)
		}
	}
	term, err := terminal.New(terminal.Options{
		Dir:         dir,
		Font:        terminal.Font{Size: 12.5},
		Transparent: true,
		Padding:     terminal.Padding{X: 10, Y: 8},
		OnTitle: func(title string) {
			title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
			if title == "" {
				return
			}
			update(func() {
				t.Title = title
				a.invalidate()
			})
		},
		OnExit: func(int) {
			update(func() {
				t.Exited = true
				a.invalidate()
			})
		},
	})
	if err != nil {
		a.fail(fmt.Errorf("terminal: %w", err))
		return
	}
	t.Term = term
	a.terms = append(a.terms, t)
	a.activeTerm = len(a.terms) - 1
	a.termOpen = true
	a.termFocus = true
	a.invalidate()
}

func (a *app) closeTerminal(i int) {
	if i < 0 || i >= len(a.terms) {
		return
	}
	t := a.terms[i]
	if t.Term != nil {
		_ = t.Term.Close()
	}
	a.terms = append(a.terms[:i], a.terms[i+1:]...)
	if a.activeTerm >= len(a.terms) {
		a.activeTerm = len(a.terms) - 1
	}
	if a.activeTerm < 0 {
		a.activeTerm = 0
	}
	if len(a.terms) == 0 {
		a.termOpen = false
		a.termFocus = false
	}
	a.invalidate()
}

func (a *app) closeTerminals() {
	for _, t := range a.terms {
		if t.Term != nil {
			_ = t.Term.Close()
		}
	}
	a.terms = nil
	a.activeTerm = 0
	a.termOpen = false
	a.termFocus = false
}

// terminalPanel draws the shells as a card on the right of the editors,
// with the same header and corners as a file pane.
func (a *app) terminalPanel(c *ui.Context, k *colors) {
	card := ui.Column(c).Width(termW).Basis(termW).Shrink(0).MinWidth(280).MinHeight(0).FillHeight().
		Radius(cardR).Clip().Background(k.card).Border(1, k.cardBorder).
		Shadow(0, 1, 2, 0, k.shadow).Shadow(0, 6, 22, -2, k.shadow)
	card.Children(func() {
		a.terminalHeader(c, k)
		a.terminalSessions(c, k)
		ui.Divider(c)
		body := ui.Box(c).Grow(1).MinHeight(0).Padding(0, 0, 4, 0)
		body.Children(func() {
			t := a.activeTermTab()
			if t == nil || t.Term == nil {
				a.emptyTerminal(c, k)
				return
			}
			tv := terminal.View(c, t.Term).Fill()
			if a.termFocus {
				tv.Focus()
				a.termFocus = false
			}
			if tv.Focused() && t.Exited {
				t.Exited = false
			}
		})
	})
}

func (a *app) terminalHeader(c *ui.Context, k *colors) {
	ui.Row(c).Height(headerH).Padding(0, 8, 0, 8).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon("square-terminal")).Size(14.5, 14.5).TextColor(k.text)
		ui.Text(c, "Terminal").FontSize(12.5).FontWeight(600).TextColor(k.text).
			SingleLine().Shrink(0)
		ui.Spacer(c)
		if n := len(a.terms); n > 1 {
			ui.Text(c, fmt.Sprintf("%d/%d", n, maxTerms)).FontSize(10.5).
				TextColor(k.textFaint).Shrink(0)
		}
		if b := iconButton(c, k, "plus", "New Terminal", 23, 13).
			Tooltip("New Terminal  ⇧⌘T").Disabled(len(a.terms) >= maxTerms); b.Clicked() {
			a.later(func() { a.newTerminal() })
		}
		if iconButton(c, k, "x", "Hide Terminal", 23, 13).
			Tooltip(tip("Hide Terminal", cmdTerminal)).Clicked() {
			a.later(func() { a.toggleTerminal() })
		}
	})
}

// terminalSessions is the row of shells under the terminal header when
// there is more than one. A single shell needs no tab of its own.
func (a *app) terminalSessions(c *ui.Context, k *colors) {
	if len(a.terms) <= 1 {
		return
	}
	ui.Row(c).Height(31).Padding(0, 8, 0, 8).Gap(3).AlignItems(ui.Center).Children(func() {
		for i, t := range a.terms {
			i, t := i, t
			active := i == a.activeTerm
			tab := ui.Row(c.Key(fmt.Sprintf("term:%d", t.ID))).Height(23).
				Padding(0, 6, 0, 4).Gap(5).Radius(6).AlignItems(ui.Center).
				Role(ui.RoleTab).Selected(active).FocusRing(false).Label(t.Title).
				Grow(1).Shrink(1).MinWidth(0).Tooltip(t.Dir)
			if active {
				tab.Background(k.tabActive).Border(1, k.cardBorder)
			} else if tab.Hovered() {
				tab.Background(k.hover)
			}
			if tab.Clicked() && !active {
				a.activeTerm = i
				a.termFocus = true
				a.invalidate()
			}
			hovered := tab.Hovered()
			tab.Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(k.attention).Shrink(0).
					Opacity(cond(t.Exited, float32(1), float32(0)))
				ui.Text(c, t.Title).FontSize(11.5).TextColor(k.textMuted).
					SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
				if hovered {
					if iconButton(c, k, "x", "Close Terminal", 15, 9).Clicked() {
						a.later(func() { a.closeTerminal(i) })
					}
				} else {
					ui.Box(c).Size(15, 15).Shrink(0)
				}
			})
		}
	})
}

func (a *app) emptyTerminal(c *ui.Context, k *colors) {
	ui.Column(c).Fill().Center().Gap(8).Children(func() {
		ui.Icon(c, icon("square-terminal")).Size(22, 22).TextColor(k.textFaint)
		ui.Text(c, "No terminal").FontSize(13).FontWeight(600).TextColor(k.text)
		btn := ui.Row(c).Padding(5, 12).Radius(7).Cursor(ui.CursorPointer).
			Role(ui.RoleButton).Label("New Terminal").Background(k.text)
		if btn.Hovered() {
			btn.Background(k.text.Alpha(0.86))
		}
		if btn.Clicked() {
			a.later(func() { a.newTerminal() })
		}
		btn.Children(func() {
			ui.Text(c, "New Terminal").FontSize(12).FontWeight(600).TextColor(k.panel)
		})
	})
}

// cond is a tiny conditional for element options.
func cond[T any](ok bool, yes, no T) T {
	if ok {
		return yes
	}
	return no
}
