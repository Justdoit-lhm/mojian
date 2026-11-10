package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"mojian/internal/fs"
	"mojian/internal/project"
	"mojian/internal/session"
)

// localServerID names the session server on this machine in terminals
// and layout files. It is not an SSH host alias.
const localServerID = "local"

// openServerStore lists the project on the server and returns it as the
// open project. File contents are read when a file is opened, not when
// the tree is.
func openServerStore(client *session.Client, root, cache string, local bool) (*store, error) {
	if client == nil || root == "" {
		return nil, fmt.Errorf("server project needs a connection and a folder")
	}
	s := project.New(fs.NewServer(client, root, cache, local))
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	return s, nil
}

// openLocalStore opens a local project. Windows reads it directly because
// its session server only serves terminals. macOS and Linux route files,
// Git and terminals through the local session server.
func (a *app) openLocalStore(dir string) (*store, error) {
	if runtime.GOOS == "js" {
		return openStore()
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	abs = resolveProjectRoot(abs)
	if runtime.GOOS == "windows" {
		return openStoreAt(abs)
	}
	client, err := a.sessions()
	if err != nil {
		return nil, err
	}
	return openServerStore(client, abs, a.serverAssetCache(localServerID, abs), true)
}

// localServerRunning reports whether this machine's session server
// answers, unless it was stopped in this session.
func (a *app) localServerRunning() bool {
	if a.localServerStopped {
		return false
	}
	if c := a.sessionClient; c != nil {
		select {
		case <-c.Closed():
		default:
			return true
		}
	}
	if runtime.GOOS == "windows" {
		if time.Since(a.localServerProbe) < time.Second {
			return a.localServerProbeOK
		}
		a.localServerProbe = time.Now()
		a.localServerProbeOK = session.Ready()
		return a.localServerProbeOK
	}
	_, err := os.Stat(session.SocketPath())
	return err == nil
}

// startLocalServer starts this machine's session server and points the
// loaded local projects and terminals at it.
func (a *app) startLocalServer() {
	if a.localServerRunning() {
		return
	}
	c, err := session.Connect()
	if err != nil {
		a.localServerErr = err.Error()
		a.invalidate()
		return
	}
	a.sessionClient = c
	a.localServerStopped = false
	a.localServerErr = ""
	a.localServerProbe, a.localServerProbeOK = time.Time{}, true
	a.installLocalConn(c)
	a.resetLocalTerminals()
	a.invalidate()
}

// stopLocalServer ends this machine's session server after saving the
// project on screen; its terminals and the processes in them end. The
// server does not come back until Start Server or the next launch.
func (a *app) stopLocalServer() {
	a.localStopAsk = false
	if err := a.saveNow(); err != nil {
		a.localServerErr = err.Error()
		a.invalidate()
		return
	}
	if err := session.Stop(); err != nil {
		a.localServerErr = err.Error()
		a.invalidate()
		return
	}
	if a.sessionClient != nil {
		_ = a.sessionClient.Close()
		a.sessionClient = nil
	}
	a.localServerStopped = true
	a.localServerErr = ""
	a.localServerProbe, a.localServerProbeOK = time.Time{}, false
	a.detachLocalTerminals()
	a.invalidate()
}

// installLocalConn points every loaded local project at a new control
// client after this machine's server restarted.
func (a *app) installLocalConn(client *session.Client) {
	a.initWorkspaces()
	patch := func(w *workspace) {
		if w == nil || w.store == nil || !w.store.IsServer() || w.serverAlias != "" {
			return
		}
		w.store.SetClient(client)
	}
	patch(a.workspace)
	for _, w := range a.workspaces {
		if w != a.workspace {
			patch(w)
		}
	}
}

// detachLocalTerminals keeps the panes but closes the views of terminals
// this machine's server owned, so Start Server can attach fresh ones.
func (a *app) detachLocalTerminals() {
	a.eachTerminalOn(localServerID, func(p *pane) {
		detachTerminalView(p)
		p.Term.Client = nil
		p.Term.startErr = ""
	})
}

// resetLocalTerminals lets the panes of a stopped server try the new one.
func (a *app) resetLocalTerminals() {
	a.eachTerminalOn(localServerID, func(p *pane) {
		p.Term.startErr = ""
	})
}
