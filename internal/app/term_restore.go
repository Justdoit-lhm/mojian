package app

import (
	"strings"

	"mojian/internal/session"
)

// terminalCreateOptions is what a new session needs, read on the UI
// thread before the request goes out.
func (a *app) terminalCreateOptions(spec termSpec) session.CreateOptions {
	var cmd []string
	dir := a.terminalDir()
	if spec.Host != "" {
		cmd = sshArgs(spec.Host)
		dir = ""
	} else if spec.Server != localServerID {
		dir = a.remotePath
	} else if strings.TrimSpace(a.ShellPath) != "" {
		cmd = []string{a.ShellPath, "-l"}
	}
	return session.CreateOptions{Command: cmd, Dir: dir, Cols: 80, Rows: 24}
}

// prepareTerminalSession reuses a saved session when it still exists, or
// creates a new one. It may run away from the UI thread.
// prepareTerminalSession reuses a saved session when it still exists, or
// creates a new one. It may run away from the UI thread.
func prepareTerminalSession(client *session.Client, sid string, opts session.CreateOptions) (string, bool, error) {
	if sid != "" {
		if list, err := client.List(); err == nil {
			for _, info := range list {
				if info.ID == sid && !info.Exited {
					return sid, false, nil
				}
			}
		}
	}
	info, err := client.Create(opts)
	if err != nil {
		return "", false, err
	}
	return info.ID, true, nil
}

// startRestoredTerminal attaches a terminal restored by the layout. The
// session lookup and creation run away from the UI thread; the view is
// built once the answer is on the window's thread. A test has no window
// and runs the whole path synchronously.
// startRestoredTerminal attaches a terminal restored by the layout. The
// session lookup and creation run away from the UI thread; the view is
// built once the answer is on the window's thread. A test has no window
// and runs the whole path synchronously.
func (a *app) startRestoredTerminal(p *pane) {
	if p == nil || p.Term == nil || p.Term.Term != nil || p.Term.connecting || p.Term.startErr != "" {
		return
	}
	t := p.Term
	if t.SID == "" && t.Spec.Host == "" {
		return
	}
	if t.Spec.Server == "" {
		// Layouts written before terminals carried a server keep working:
		// a direct SSH pane lives locally; a plain shell follows the
		// project that was open when the layout was written.
		switch {
		case t.Spec.Host != "":
			t.Spec.Server = localServerID
		case a.remotePath != "" && a.serverAlias != "":
			t.Spec.Server = a.serverAlias
		default:
			t.Spec.Server = localServerID
		}
	}
	// The terminal belongs to the server that already owns the project.
	// Do not dial a new remote connection from a view: a missing one is
	// shown as an error with a retry.
	client, err := a.terminalClientExisting(t.Spec)
	if err != nil {
		t.startErr = err.Error()
		a.invalidate()
		return
	}
	opts := a.terminalCreateOptions(t.Spec)
	sid := t.SID
	t.connecting = true
	a.invalidate()
	finish := func(sid string, created bool, err error) {
		if p.Term != t {
			return
		}
		t.connecting = false
		if err != nil {
			t.startErr = err.Error()
			a.invalidate()
			return
		}
		if a.win != nil && a.tabOf(p) == nil {
			// The pane left the layout while its session was opening.
			if created {
				go func() { _ = client.Kill(sid) }()
			}
			return
		}
		tt, err := a.attachTerminal(client, sid, t.Spec)
		if err != nil {
			t.startErr = err.Error()
			a.invalidate()
			return
		}
		t.Term, t.SID, t.stream, t.Client = tt.Term, tt.SID, tt.stream, tt.Client
		t.startErr = ""
		a.changed()
		a.invalidate()
	}
	if a.win == nil {
		// Tests and a headless build have no window thread.
		sid, created, err := prepareTerminalSession(client, sid, opts)
		finish(sid, created, err)
		return
	}
	win := a.win
	go func() {
		sid, created, err := prepareTerminalSession(client, sid, opts)
		onWindow(win, func() { finish(sid, created, err) })
	}()
}

// detachTerminalView closes the local terminal and stream without
// ending the session on the server.
// detachTerminalView closes the local terminal and stream without
// ending the session on the server.
func detachTerminalView(p *pane) {
	if p == nil || p.Term == nil {
		return
	}
	if p.Term.Term != nil {
		_ = p.Term.Term.Close()
		p.Term.Term = nil
	}
	if p.Term.stream != nil {
		_ = p.Term.stream.Close()
		p.Term.stream = nil
	}
}

// endTerminalPane closes a pane's terminal view, ending its session on
// the server when kill is set.
// endTerminalPane closes a pane's terminal view, ending its session on
// the server when kill is set.
func (a *app) endTerminalPane(p *pane, kill bool) {
	if p == nil || p.Term == nil {
		return
	}
	if kill {
		a.endTerminal(p.Term)
	}
	detachTerminalView(p)
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
	sid := t.SID
	go func() { _ = client.Kill(sid) }()
}
