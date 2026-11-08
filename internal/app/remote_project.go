package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"mojian/internal/session"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) remoteName() string {
	if a.serverAlias == "" {
		return "Local"
	}
	return a.hostLabel(a.serverAlias)
}

// sshOutput runs a command over SSH with the user's config and agent.
// sshCommon is the non-interactive SSH base. It deliberately leaves out
// ControlMaster: a stale master socket makes every later ssh wait on a
// process that is gone, and the commands here are few enough that a fresh
// connection is the reliable choice.
// workspaceReachable reports whether a cached remote workspace still has
// a live connection. A local workspace always does.
func (a *app) workspaceReachable(ws *workspace) bool {
	if ws == nil || ws.remotePath == "" {
		return true
	}
	return remoteHealthy(a.remotes[ws.serverAlias])
}

// adoptWorkspaceConn points a reconnected workspace and every other
// workspace on the same host at the new control client. Terminals that
// were attached through the lost connection are detached so their panes
// reattach when they are shown again.
// adoptWorkspaceConn points a reconnected workspace and every other
// workspace on the same host at the new control client. Terminals that
// were attached through the lost connection are detached so their panes
// reattach when they are shown again.
func (a *app) adoptWorkspaceConn(ws *workspace, rc *remoteConn) {
	if rc == nil {
		return
	}
	a.initWorkspaces()
	a.remotes[rc.host] = rc
	for _, w := range a.workspaces {
		if w.store == nil || !w.store.isServer() || w.serverAlias != rc.host {
			continue
		}
		w.store.setClient(rc.client)
		detachTerminalsOn(w, rc.host, rc.client)
	}
}

// detachTerminalsOn closes the views of terminals owned by host, keeping
// the sessions on the server for a reattach.
// detachTerminalsOn closes the views of terminals owned by host, keeping
// the sessions on the server for a reattach.
func detachTerminalsOn(ws *workspace, host string, client *session.Client) {
	if ws == nil {
		return
	}
	for _, t := range ws.tabs {
		for _, p := range t.panes() {
			if p.Term == nil || p.Term.Spec.Server != host {
				continue
			}
			detachTerminalView(p)
			p.Term.Client = client
			p.Term.startErr = ""
		}
	}
}

// reconnectActiveRemoteAsync restores the control client of the project
// on screen after a window reopen, without blocking the new window.
// reconnectActiveRemoteAsync restores the control client of the project
// on screen after a window reopen, without blocking the new window.
func (a *app) reconnectActiveRemoteAsync() {
	host := a.serverAlias
	if host == "" || a.remoteBusy {
		return
	}
	if existing := a.remoteForHost(host); existing != nil && remoteHealthy(existing) {
		a.adoptWorkspaceConn(a.workspace, existing)
		a.invalidate()
		return
	}
	seq, win := a.beginRemoteTask("Reconnecting to " + a.hostLabel(host) + "…")
	if seq == 0 {
		return
	}
	go func() {
		rc, err := dialRemote(host)
		onWindow(win, func() {
			if !a.endRemoteTask(seq) {
				if rc != nil {
					a.detachRemoteConn(rc)
				}
				return
			}
			if err != nil {
				a.fail(err)
				return
			}
			a.adoptWorkspaceConn(a.workspace, rc)
			a.saveSettings()
			a.invalidate()
		})
	}()
}

// reconnectWorkspaceAsync reconnects before switching, behind the same
// loading state as opening a project.
// reconnectWorkspaceAsync reconnects before switching, behind the same
// loading state as opening a project.
func (a *app) reconnectWorkspaceAsync(ws *workspace, host string) {
	if ws == nil || a.remoteBusy {
		return
	}
	existing := a.remoteForHost(host)
	if existing != nil && remoteHealthy(existing) {
		a.adoptWorkspaceConn(ws, existing)
		a.switchWorkspace(ws)
		return
	}
	if err := a.saveNow(); err != nil {
		a.fail(fmt.Errorf("could not save the open project: %w", err))
		return
	}
	seq, win := a.beginRemoteTask("Reconnecting to " + a.hostLabel(host) + "…")
	if seq == 0 {
		return
	}
	go func() {
		rc, err := dialRemote(host)
		onWindow(win, func() {
			if !a.endRemoteTask(seq) {
				if rc != nil {
					a.detachRemoteConn(rc)
				}
				return
			}
			if err != nil {
				a.fail(err)
				return
			}
			a.adoptWorkspaceConn(ws, rc)
			a.switchWorkspace(ws)
			a.saveSettings()
			a.invalidate()
		})
	}()
}

// openRemoteProjectAsync opens a remote project away from the UI thread,
// with a loading state. A project already in the workspace cache switches
// at once, without connecting again.
// openRemoteProjectAsync opens a remote project away from the UI thread,
// with a loading state. A project already in the workspace cache switches
// at once, without connecting again.
func (a *app) openRemoteProjectAsync(host, remotePath string) {
	if !terminalsSupported() {
		a.fail(fmt.Errorf("remote projects need macOS or Linux"))
		return
	}
	if host == "" || remotePath == "" {
		a.fail(fmt.Errorf("choose a server and a project"))
		return
	}
	remotePath = strings.TrimRight(remotePath, "/")
	if a.remoteBusy {
		return
	}
	a.initWorkspaces()
	key := workspaceKeyRemote(host, remotePath)
	if ws := a.workspaces[key]; ws != nil {
		a.rememberRemoteProject(remoteProject{Host: host, Path: remotePath})
		if a.workspaceReachable(ws) {
			a.switchWorkspace(ws)
			a.saveSettings()
		} else {
			a.reconnectWorkspaceAsync(ws, host)
		}
		return
	}
	if err := a.saveNow(); err != nil {
		a.fail(fmt.Errorf("could not save the open project: %w", err))
		return
	}
	if a.layoutPath != "" {
		a.saveLayout()
	}
	a.layoutDirty = false
	existing := a.remoteForHost(host)
	seq, win := a.beginRemoteTask("Opening " + filepath.Base(remotePath) + " on " + a.hostLabel(host) + "…")
	if seq == 0 {
		return
	}
	cache := a.serverAssetCache(host, remotePath)
	go func() {
		rc := existing
		var err error
		if rc == nil {
			rc, err = dialRemote(host)
		}
		var s *store
		if err == nil {
			onWindow(win, func() {
				a.setRemoteBusyMsg(seq, "Opening "+filepath.Base(remotePath)+" on "+a.hostLabel(host)+"…")
			})
			s, err = openServerStore(rc.client, remotePath, cache, false)
		}
		onWindow(win, func() {
			if !a.endRemoteTask(seq) {
				if rc != nil && rc != existing {
					a.detachRemoteConn(rc)
				}
				return
			}
			if err != nil {
				if rc != nil && rc != existing {
					a.detachRemoteConn(rc)
				}
				a.fail(err)
				return
			}
			if rc != nil {
				a.remotes[host] = rc
			}
			a.stashActiveWorkspace()
			ws := &workspace{
				key:         key,
				project:     a.project,
				store:       s,
				serverAlias: host,
				remotePath:  remotePath,
				layoutPath:  a.projectLayoutPath("remote:" + host + ":" + remotePath),
				treeOpen:    true,
				active:      -1,
				openDirs:    map[string]bool{},
			}
			a.workspaces[key] = ws
			a.activateWorkspace(ws)
			a.rememberRemoteProject(remoteProject{Host: host, Path: remotePath})
			if !a.restoreLayout() {
				a.focusFirstNote()
			}
			if t := a.tab(); t != nil {
				a.focusReq = t.editorPane()
			}
			a.revealTab = true
			a.saveLayout()
			a.saveSettings()
			a.updateTitle()
			a.invalidate()
		})
	}()
}

func (a *app) rememberRemoteProject(p remoteProject) {
	if p.Host == "" || p.Path == "" {
		return
	}
	out := []remoteProject{p}
	for _, q := range a.remoteProjects {
		if q.Host == p.Host && q.Path == p.Path {
			continue
		}
		out = append(out, q)
	}
	if len(out) > 12 {
		out = out[:12]
	}
	a.remoteProjects = out
}

// beginRemoteTask covers the window with a loading state and returns
// the sequence that guards its result.
// beginRemoteTask covers the window with a loading state and returns
// the sequence that guards its result.
func (a *app) beginRemoteTask(msg string) (int, *mygo.Window) {
	if a.remoteBusy {
		return 0, nil
	}
	a.remoteTaskSeq++
	a.remoteBusy = true
	a.remoteBusyMsg = msg
	a.invalidate()
	if a.win != nil {
		return a.remoteTaskSeq, a.win
	}
	return a.remoteTaskSeq, a.settingsWin
}

// setRemoteBusyMsg changes the line under the spinner, once the task may
// still publish.
// setRemoteBusyMsg changes the line under the spinner, once the task may
// still publish.
func (a *app) setRemoteBusyMsg(seq int, msg string) {
	if !a.remoteBusy || a.remoteTaskSeq != seq {
		return
	}
	a.remoteBusyMsg = msg
	a.invalidate()
}

// endRemoteTask uncovers the window; a result of an older task is
// dropped by its sequence.
// endRemoteTask uncovers the window; a result of an older task is
// dropped by its sequence.
func (a *app) endRemoteTask(seq int) bool {
	if a.remoteTaskSeq != seq {
		return false
	}
	a.remoteBusy = false
	a.remoteBusyMsg = ""
	a.invalidate()
	return true
}

// onWindow runs fn on the window's thread when this build has one.
// onWindow runs fn on the window's thread when this build has one.
func onWindow(win *mygo.Window, fn func()) {
	if win != nil {
		win.Update(fn)
		return
	}
	fn()
}

// remoteBusyView covers the window while a server is reached, so the
// first connection does not look like a freeze.
// remoteBusyView covers the window while a server is reached, so the
// first connection does not look like a freeze.
func (a *app) remoteBusyView(c *ui.Context, k *colors) {
	if !a.remoteBusy {
		return
	}
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Center().Background(k.Backdrop).Modal()
		back.Children(func() {
			panel := ui.Row(c).Width(330).Padding(18, 20).Gap(14).Radius(radius).
				Background(k.Panel).Border(1, k.PanelBorder).
				Shadow(0, 18, 44, -8, k.ShadowFocused).AlignItems(ui.Center).
				Focusable().Focus()
			panel.Children(func() {
				ui.Spinner(c).Size(18, 18).Label("Working")
				ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
					ui.Text(c, "Connecting to a server").FontSize(13).
						FontWeight(600).TextColor(k.Text)
					ui.Text(c, a.remoteBusyMsg).FontSize(11.5).TextColor(k.TextMuted).
						LineHeight(1.35)
				})
			})
		})
	})
}

// openRemotePicker asks the selected server for its home, then shows the
// directories to choose a project from.
// remoteProject is a project on an SSH host, remembered by the switcher.
type remoteProject struct {
	Host string `json:"host"`
	Path string `json:"path"`
}

// remoteConn is this app's bridge to one remote session server: an SSH
// forward to its Unix socket, and the control connection through it.
