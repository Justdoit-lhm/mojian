package app

import (
	"errors"
	"fmt"
	"github.com/egoist/mygo/plugins/terminal"
	"runtime"
)

// termTab is the shell shown by a terminal pane.
type termTab struct {
	Term   *terminal.Terminal
	Exited bool
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
	t := a.tab()
	if t == nil {
		a.fail(errors.New("open a file before starting a terminal"))
		return
	}
	if t.Focus != nil && t.Focus.Term != nil {
		a.closePane(t.Focus)
		return
	}
	if p := t.terminalPane(); p != nil {
		t.Focus = p
		a.focusReq = p
		a.changed()
		a.invalidate()
		return
	}
	a.newTerminalPane()
}

// splitFocused splits the focused pane: a terminal makes another shell,
// a file asks which file the new editor should show.
func (a *app) splitFocused(down bool) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	if t.Focus.Term != nil {
		a.splitTerminal(t.Focus, down)
		return
	}
	a.beginSplit(down)
}

// splitTerminal starts another shell beside or below the terminal pane
// that asked for the split.
func (a *app) splitTerminal(from *pane, down bool) {
	t := a.tab()
	if t == nil || from == nil || from.Term == nil || !t.contains(from) {
		return
	}
	if len(t.panes()) >= maxPanes {
		a.fail(fmt.Errorf("a tab can show at most %d panes", maxPanes))
		return
	}
	tt, err := a.startTerminal()
	if err != nil {
		a.fail(err)
		return
	}
	a.addSplitTerminal(from, tt, down)
}

// addSplitTerminal inserts an already started shell beside or below a
// terminal pane. Tests use it without starting a process.
func (a *app) addSplitTerminal(from *pane, tt *termTab, down bool) *pane {
	t := a.tab()
	if t == nil || from == nil || tt == nil || !t.contains(from) {
		return nil
	}
	if len(t.panes()) >= maxPanes {
		a.fail(fmt.Errorf("a tab can show at most %d panes", maxPanes))
		return nil
	}
	with := newTermPane(tt)
	if !t.split(from, with, down) {
		return nil
	}
	t.Focus = with
	a.focusReq = with
	a.changed()
	a.invalidate()
	return with
}

// newTerminalPane starts a shell and puts it in a pane on the right edge
// of the active tab, as one of the tab's four panes.
func (a *app) newTerminalPane() {
	if runtime.GOOS == "js" {
		a.fail(errors.New("terminals need the desktop app"))
		return
	}
	if a.store == nil || a.store.memory {
		a.fail(errors.New("terminals need a project on disk"))
		return
	}
	t := a.tab()
	if t == nil {
		a.fail(errors.New("open a file before starting a terminal"))
		return
	}
	if len(t.panes()) >= maxPanes {
		a.fail(fmt.Errorf("a tab can show at most %d panes", maxPanes))
		return
	}
	tt, err := a.startTerminal()
	if err != nil {
		a.fail(err)
		return
	}
	a.addTerminalPane(tt)
}

// addTerminalPane wraps the active tab so the shell sits at its right
// edge, beside every editor.
func (a *app) addTerminalPane(tt *termTab) *pane {
	t := a.tab()
	if t == nil || tt == nil {
		return nil
	}
	if len(t.panes()) >= maxPanes {
		a.fail(fmt.Errorf("a tab can show at most %d panes", maxPanes))
		return nil
	}
	p := newTermPane(tt)
	if t.Root == nil {
		t.Root = &node{Pane: p}
	} else {
		t.Root = &node{
			Vertical: false,
			Ratio:    0.66,
			A:        t.Root,
			B:        &node{Pane: p},
		}
		t.Root.startSplitAnim(0.66)
	}
	t.Zoom = nil
	t.Focus = p
	a.git.Open = false
	a.focusReq = p
	a.changed()
	a.invalidate()
	return p
}

// startTerminal starts a shell without adding it to a tab, so callers can
// put it in a pane.
func (a *app) startTerminal() (*termTab, error) {
	if a.shellStarter != nil {
		return a.shellStarter()
	}
	dir := a.terminalDir()
	tt := &termTab{}
	win := a.win
	update := func(fn func()) {
		if win != nil {
			win.Update(fn)
		}
	}
	copySelect := terminal.CopyOnSelectNone
	if a.terminalCopySelect {
		copySelect = terminal.CopyOnSelectClipboard
	}
	term, err := terminal.New(terminal.Options{
		Dir: dir,
		Font: terminal.Font{
			Family:     a.terminalFontFamily,
			Size:       float32(a.termFontSize()),
			LineHeight: float32(a.termLineHeight()),
		},
		OptionAsAlt:  a.terminalOptionAsAlt,
		CopyOnSelect: copySelect,
		NoBlink:      a.noCursorBlink,
		Scrollback:   a.termScrollback() * 1 << 20,
		Transparent:  true,
		Padding:      terminal.Padding{X: 10, Y: 8},
		OnExit: func(int) {
			update(func() {
				tt.Exited = true
				a.invalidate()
			})
		},
	})
	if err != nil {
		return nil, fmt.Errorf("terminal: %w", err)
	}
	tt.Term = term
	return tt, nil
}

// closeTerminals ends every shell, before the window or project goes.
func (a *app) closeTerminals() {
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.Term != nil && p.Term.Term != nil {
				_ = p.Term.Term.Close()
			}
		}
	}
}
