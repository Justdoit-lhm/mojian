package app

import (
	"errors"
	"fmt"
	"os"

	"mojian/internal/session"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
)

// termTab is the shell shown by a terminal pane. Its process lives in the
// session server, so closing the app only detaches it.
type termTab struct {
	Term *terminal.Terminal

	SID    string
	Spec   termSpec
	Client *session.Client
	stream *session.Stream

	startErr   string
	connecting bool
}

// termSpec is what a terminal pane opens: the login shell, or SSH to a
// configured host address. Server names the session server that owns the
// terminal, so a restored pane reattaches on the same machine.
type termSpec struct {
	Host   string
	Server string
}

func (s termSpec) title() string {
	if s.Host != "" {
		return "SSH · " + s.Host
	}
	return "Terminal"
}

// warmSessionServer starts the local session server before the first
// terminal asks for it, so opening a terminal does not wait for a spawn.
func warmSessionServer() {
	if !terminalsSupported() {
		return
	}
	if _, err := os.Stat(session.SocketPath()); err == nil {
		return
	}
	_ = session.Spawn()
}

// terminalDir is the project root. Terminals are about the repository,
// not the file or folder selected in the Explorer, and one can be opened
// before any file is.
func (a *app) terminalDir() string {
	if a.store == nil || a.store.IsMemory() {
		return ""
	}
	return a.store.Dir
}

func (a *app) toggleTerminal() {
	if !terminalsSupported() {
		a.fail(errors.New("terminals need macOS or Linux"))
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

// splitFocusedSide splits the focused pane to either side.
func (a *app) splitFocusedSide(side splitSide) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	if t.Focus.Term != nil {
		a.splitTerminalSide(t.Focus, side)
		return
	}
	a.emptyPaneSide(side)
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

func (a *app) addSplitTerminalSide(from *pane, tt *termTab, side splitSide) *pane {
	t := a.tab()
	if t == nil || from == nil || tt == nil || !t.contains(from) {
		return nil
	}
	with := newTermPane(tt)
	with.enterSide(side)
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
	if !terminalsSupported() {
		a.fail(errors.New("terminals need macOS or Linux"))
		return
	}
	if a.store == nil || a.store.IsMemory() {
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

// addTerminalPane adds the shell at the right edge of the active tab,
// beside every editor.
func (a *app) addTerminalPane(tt *termTab) *pane {
	t := a.tab()
	if t == nil || tt == nil {
		return nil
	}
	p := newTermPane(tt)
	p.enterSide(splitRight)
	a.unzoom(t)
	if t.Root == nil {
		t.Root = &node{Pane: p}
	} else {
		t.Root = &node{Vertical: false, Share: 50, A: t.Root, B: &node{Pane: p}}
	}
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

// defaultTerminalServer is the server a new terminal belongs to: the
// remote project's server when one is open, else this machine.
func (a *app) defaultTerminalServer() string {
	if a.remotePath != "" && a.serverAlias != "" {
		return a.serverAlias
	}
	return localServerID
}

// startTerminalSpec starts a login shell, or SSH to a configured host, in
// the session server. The process outlives the app.
func (a *app) startTerminalSpec(spec termSpec) (*termTab, error) {
	if !terminalsSupported() {
		return nil, errors.New("terminals need macOS or Linux")
	}
	if a.shellStarter != nil {
		return a.shellStarter()
	}
	if spec.Host != "" {
		// A direct SSH pane's ssh process lives in the local server.
		spec.Server = localServerID
	} else if spec.Server == "" {
		spec.Server = a.defaultTerminalServer()
	}
	client, err := a.terminalClient(spec)
	if err != nil {
		return nil, err
	}
	info, err := client.Create(a.terminalCreateOptions(spec))
	if err != nil {
		return nil, fmt.Errorf("session: %w", err)
	}
	return a.attachTerminal(client, info.ID, spec)
}

// terminalClientExisting chooses an already connected server for a
// restored pane; it never dials SSH from the window thread.
func (a *app) terminalClientExisting(spec termSpec) (*session.Client, error) {
	if spec.Host != "" || spec.Server == "" || spec.Server == localServerID {
		return a.sessions()
	}
	if rc := a.remoteForHost(spec.Server); rc != nil {
		return rc.Client, nil
	}
	return nil, fmt.Errorf("%s is not connected", spec.Server)
}

// terminalClient chooses the session server that owns a terminal. A
// direct SSH pane still starts from the local server, so the SSH process
// lives with this machine.
func (a *app) terminalClient(spec termSpec) (*session.Client, error) {
	if spec.Host != "" || spec.Server == "" || spec.Server == localServerID {
		return a.sessions()
	}
	return a.remoteServerClient(spec.Server)
}

// remoteServerClient returns a live control connection to host without
// changing the open project.
func (a *app) remoteServerClient(host string) (*session.Client, error) {
	a.initWorkspaces()
	if rc := a.remoteForHost(host); rc != nil {
		return rc.Client, nil
	}
	rc, err := a.connectRemote(host)
	if err != nil {
		return nil, err
	}
	return rc.Client, nil
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
	if a.TerminalCopySelect {
		copySelect = terminal.CopyOnSelectClipboard
	}
	return terminal.Options{
		Conn: conn,
		Font: terminal.Font{
			Family:     a.TerminalFontFamily,
			Size:       float32(a.termFontSize()),
			LineHeight: float32(a.termLineHeight()),
		},
		OptionAsAlt:  a.TerminalOptionAsAlt,
		CopyOnSelect: copySelect,
		Cursor:       terminal.CursorStyle(a.TerminalCursor),
		NoBlink:      a.NoCursorBlink,
		Scrollback:   a.termScrollback() * 1 << 20,
		Transparent:  true,
		Padding:      terminal.Padding{X: 10, Y: 8},
		OnBell: func() {
			update(func() {
				if a.NotifyBell && (a.win == nil || !a.win.IsFocused()) {
					mygo.NewNotification(mygo.NotificationOptions{
						Title: "Terminal", Body: tt.Spec.title(),
					}).Show()
				}
				a.invalidate()
			})
		},
		OnExit: func(int) {
			update(func() {
				if a.NotifyExit && !a.quitting && (a.win == nil || !a.win.IsFocused()) {
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
		select {
		case <-a.sessionClient.Closed():
			a.sessionClient = nil
		default:
			return a.sessionClient, nil
		}
	}
	c, err := session.Connect()
	if err != nil {
		return nil, err
	}
	a.sessionClient = c
	return c, nil
}
