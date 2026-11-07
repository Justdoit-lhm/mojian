package app

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"mojian/internal/session"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
)

// termTab is the shell shown by a terminal pane. Its process lives in the
// session server, so closing the app only detaches it.
type termTab struct {
	Term   *terminal.Terminal
	Exited bool

	SID    string
	Spec   termSpec
	Client *session.Client
	stream *session.Stream

	starting bool
	startErr string
}

// termSpec is what a terminal pane opens: the login shell, or SSH to a
// host alias from ~/.ssh/config.
type termSpec struct {
	Host string
}

func (s termSpec) title() string {
	if s.Host != "" {
		return "SSH · " + s.Host
	}
	return "Terminal"
}

// terminalDir is the project root. Terminals are about the repository,
// not the file or folder selected in the Explorer, and one can be opened
// before any file is.
func (a *app) terminalDir() string {
	if a.store == nil || a.store.memory {
		return ""
	}
	return a.store.dir
}

func (a *app) toggleTerminal() {
	if runtime.GOOS == "js" {
		a.fail(errors.New("terminals need the desktop app"))
		return
	}
	t := a.terminalTab()
	if t == nil {
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
	side := splitRight
	if down {
		side = splitDown
	}
	a.splitFocusedSide(side)
}

// splitFocusedSide splits the focused pane to any side.
func (a *app) splitFocusedSide(side splitSide) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	a.overview = false
	a.overviewFocus = false
	if t.Focus.Term != nil {
		a.splitTerminalSide(t.Focus, side)
		return
	}
	a.beginSplitSide(side)
}

// splitTerminal starts another shell beside or below the terminal pane
// that asked for the split.
func (a *app) splitTerminal(from *pane, down bool) {
	side := splitRight
	if down {
		side = splitDown
	}
	a.splitTerminalSide(from, side)
}

func (a *app) splitTerminalSide(from *pane, side splitSide) {
	t := a.tab()
	if t == nil || from == nil || from.Term == nil || !t.contains(from) {
		return
	}
	tt, err := a.startTerminalSpec(from.Term.Spec)
	if err != nil {
		a.fail(err)
		return
	}
	a.addSplitTerminalSide(from, tt, side)
}

// addSplitTerminal inserts an already started shell beside or below a
// terminal pane. Tests use it without starting a process.
func (a *app) addSplitTerminal(from *pane, tt *termTab, down bool) *pane {
	side := splitRight
	if down {
		side = splitDown
	}
	return a.addSplitTerminalSide(from, tt, side)
}

func (a *app) addSplitTerminalSide(from *pane, tt *termTab, side splitSide) *pane {
	t := a.tab()
	if t == nil || from == nil || tt == nil || !t.contains(from) {
		return nil
	}
	with := newTermPane(tt)
	if !t.splitSide(from, with, side) {
		return nil
	}
	t.Focus = with
	a.focusReq = with
	a.changed()
	a.invalidate()
	return with
}

// newTerminalPane starts a shell and puts it in a pane on the right edge
// of the active tab, as one of the tab's panes.
func (a *app) newTerminalPane() {
	if runtime.GOOS == "js" {
		a.fail(errors.New("terminals need the desktop app"))
		return
	}
	if a.store == nil || a.store.memory {
		a.fail(errors.New("terminals need a project on disk"))
		return
	}
	t := a.terminalTab()
	if t == nil {
		return
	}
	tt, err := a.startTerminal()
	if err != nil {
		a.fail(err)
		return
	}
	a.addTerminalPane(tt)
}

// terminalTab returns the active tab, opening an empty one when no file
// is open yet. A terminal does not need a file to sit beside.
func (a *app) terminalTab() *tab {
	if t := a.tab(); t != nil {
		return t
	}
	a.git.Open = false
	t := &tab{ID: nextID("t")}
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.revealTab = true
	a.treeDraft = nil
	return t
}

// addTerminalPane wraps the active tab so the shell sits at its right
// edge, beside every editor.
func (a *app) addTerminalPane(tt *termTab) *pane {
	t := a.tab()
	if t == nil || tt == nil {
		return nil
	}
	p := newTermPane(tt)
	p.W = 0.5
	if t.Root == nil {
		p.W = 1
		t.Root = &node{Pane: p}
	} else {
		t.Root = &node{
			Vertical: false,
			Ratio:    0.5,
			A:        t.Root,
			B:        &node{Pane: p},
		}
	}
	t.Zoom = nil
	t.Focus = p
	a.git.Open = false
	a.focusReq = p
	a.changed()
	a.invalidate()
	return p
}

// startTerminal starts the login shell without adding it to a tab.
func (a *app) startTerminal() (*termTab, error) {
	return a.startTerminalSpec(termSpec{})
}

// startTerminalSpec starts a login shell, or SSH to a configured host, in
// the session server. The process outlives the app.
func (a *app) startTerminalSpec(spec termSpec) (*termTab, error) {
	if a.shellStarter != nil {
		return a.shellStarter()
	}
	client, err := a.terminalClient(spec)
	if err != nil {
		return nil, err
	}
	var cmd []string
	dir := a.terminalDir()
	if spec.Host != "" {
		cmd = sshArgs(spec.Host)
		dir = ""
	} else if a.remote != nil {
		dir = a.remotePath
	} else if strings.TrimSpace(a.shellPath) != "" {
		cmd = []string{a.shellPath, "-l"}
	}
	info, err := client.Create(session.CreateOptions{
		Command: cmd, Dir: dir, Cols: 80, Rows: 24,
	})
	if err != nil {
		return nil, fmt.Errorf("session: %w", err)
	}
	return a.attachTerminal(client, info.ID, spec)
}

// terminalClient chooses the local session server, or the current remote
// one for a project on a server. A direct SSH pane still starts from the
// local server, so the SSH process lives with this machine.
func (a *app) terminalClient(spec termSpec) (*session.Client, error) {
	if spec.Host == "" && a.remote != nil && a.remote.client != nil {
		return a.remote.client, nil
	}
	return a.sessions()
}

// attachTerminal makes a terminal view attached to a session.
func (a *app) attachTerminal(client *session.Client, sid string, spec termSpec) (*termTab, error) {
	stream := client.Stream(sid, 80, 24)
	tt := &termTab{SID: sid, Spec: spec, Client: client, stream: stream}
	term, err := terminal.New(a.terminalOptions(stream, tt))
	if err != nil {
		_ = stream.Close()
		return nil, fmt.Errorf("terminal: %w", err)
	}
	term.Resize(80, 24)
	tt.Term = term
	return tt, nil
}

func (a *app) terminalOptions(conn *session.Stream, tt *termTab) terminal.Options {
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
	return terminal.Options{
		Conn: conn,
		Font: terminal.Font{
			Family:     a.terminalFontFamily,
			Size:       float32(a.termFontSize()),
			LineHeight: float32(a.termLineHeight()),
		},
		OptionAsAlt:  a.terminalOptionAsAlt,
		CopyOnSelect: copySelect,
		Cursor:       terminal.CursorStyle(a.terminalCursor),
		NoBlink:      a.noCursorBlink,
		Scrollback:   a.termScrollback() * 1 << 20,
		Transparent:  true,
		Padding:      terminal.Padding{X: 10, Y: 8},
		OnBell: func() {
			update(func() {
				if a.notifyBell && (a.win == nil || !a.win.IsFocused()) {
					mygo.NewNotification(mygo.NotificationOptions{
						Title: "Terminal", Body: tt.Spec.title(),
					}).Show()
				}
				a.invalidate()
			})
		},
		OnExit: func(int) {
			update(func() {
				tt.Exited = true
				if a.notifyExit && !a.quitting && (a.win == nil || !a.win.IsFocused()) {
					mygo.NewNotification(mygo.NotificationOptions{
						Title: "Terminal finished", Body: tt.Spec.title(),
					}).Show()
				}
				a.invalidate()
			})
		},
	}
}

// sessions returns the control connection to the session server, starting
// it when needed.
func (a *app) sessions() (*session.Client, error) {
	if a.sessionClient != nil {
		return a.sessionClient, nil
	}
	c, err := session.Connect()
	if err != nil {
		return nil, err
	}
	a.sessionClient = c
	return c, nil
}

// startRestoredTerminal starts or reattaches a terminal restored by the
// layout, once its pane is first shown.
func (a *app) startRestoredTerminal(p *pane) {
	if p == nil || p.Term == nil || p.Term.Term != nil || p.Term.starting || p.Term.startErr != "" {
		return
	}
	if p.Term.SID == "" && p.Term.Spec.Host == "" {
		return
	}
	t := p.Term
	t.starting = true
	defer func() { t.starting = false }()
	client, err := a.sessions()
	if err != nil {
		t.startErr = err.Error()
		return
	}
	sid := t.SID
	if sid != "" {
		if list, listErr := client.List(); listErr == nil {
			found := false
			for _, info := range list {
				if info.ID == sid {
					found = true
					break
				}
			}
			if !found {
				sid = ""
			}
		}
	}
	client, err = a.terminalClient(t.Spec)
	if err != nil {
		t.startErr = err.Error()
		return
	}
	var tt *termTab
	if sid != "" {
		tt, err = a.attachTerminal(client, sid, t.Spec)
	} else {
		tt, err = a.startTerminalSpec(t.Spec)
	}
	if err != nil {
		t.startErr = err.Error()
		a.invalidate()
		return
	}
	t.Term, t.SID, t.stream = tt.Term, tt.SID, tt.stream
	a.changed()
	a.invalidate()
}

// endTerminal ends a session when its pane is closed by the user.
func (a *app) endTerminal(t *termTab) {
	if t == nil || t.SID == "" {
		return
	}
	client := t.Client
	if client == nil {
		client = a.sessionClient
	}
	if client == nil {
		return
	}
	_ = client.Kill(t.SID)
}

// closeTerminals detaches every terminal before the window or project
// goes: the server keeps the shells running.
func (a *app) closeTerminals() {
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.Term != nil {
				if p.Term.Term != nil {
					_ = p.Term.Term.Close()
					p.Term.Term = nil
				}
				if p.Term.stream != nil {
					_ = p.Term.stream.Close()
					p.Term.stream = nil
				}
			}
		}
	}
}
