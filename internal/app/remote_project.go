package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"mojian/internal/remote"
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

// workspaceReachable reports whether a cached remote workspace still has
// a live connection. A local workspace always does.
func (a *app) workspaceReachable(ws *workspace) bool {
	if ws == nil || ws.remotePath == "" {
		return true
	}
	return a.remoteConn(ws.serverAlias).Healthy()
}

// installRemoteConn makes rc the pool's connection for its host and
// points every loaded workspace on that host at its client. A new
// connection also detaches terminal views so their panes reattach.
func (a *app) installRemoteConn(rc *remote.Conn) {
	if rc == nil {
		return
	}
	a.initWorkspaces()
	same := a.setRemoteConn(rc)
	patch := func(w *workspace) {
		if w == nil || w.store == nil || !w.store.IsServer() || w.serverAlias != rc.Host {
			return
		}
		w.store.SetClient(rc.Client)
		if !same {
			detachTerminalsOn(w, rc.Host, rc.Client)
		}
	}
	patch(a.workspace)
	for _, w := range a.workspaces {
		if w != a.workspace {
			patch(w)
		}
	}
}

// eachTerminalOn calls fn for every terminal belonging to server in the
// project on screen and in the cached projects.
func (a *app) eachTerminalOn(server string, fn func(*pane)) {
	each := func(ws *workspace) {
		if ws == nil {
			return
		}
		for _, t := range ws.tabs {
			for _, p := range t.panes() {
				if p.Term != nil && p.Term.Spec.Server == server {
					fn(p)
				}
			}
		}
	}
	each(a.workspace)
	for _, ws := range a.workspaces {
		if ws != a.workspace {
			each(ws)
		}
	}
}

// failTerminalsOn ends the waiting state of terminals on host with the
// connection error, so their Try Again is real rather than a spinner.
func (a *app) failTerminalsOn(host string, err error) {
	if err == nil {
		return
	}
	a.eachTerminalOn(host, func(p *pane) {
		p.Term.connecting = false
		p.Term.startErr = err.Error()
	})
}

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
			p.Term.connecting = false
		}
	}
}

// reconnectAsync dials host behind the loading card and hands the live
// connection to ready. It reports whether a task was started.
func (a *app) reconnectAsync(host, msg string, ready func(*remote.Conn)) bool {
	if host == "" || a.remoteBusy {
		return false
	}
	if existing := a.remoteForHost(host); existing != nil && existing.Healthy() {
		a.keepRemote(host)
		a.installRemoteConn(existing)
		if ready != nil {
			ready(existing)
		}
		a.invalidate()
		return true
	}
	seq, win, ctx := a.beginRemoteTask(msg)
	if seq == 0 {
		return false
	}
	go func() {
		rc, err := remote.DialContext(ctx, host)
		onWindow(win, func() {
			if !a.endRemoteTask(seq) {
				if rc != nil {
					a.detachRemoteConn(rc)
				}
				return
			}
			if err != nil {
				a.failTerminalsOn(host, err)
				a.fail(err)
				return
			}
			a.installRemoteConn(rc)
			a.keepRemote(host)
			if ready != nil {
				ready(rc)
			}
			a.saveSettings()
			a.invalidate()
		})
	}()
	return true
}

// reconnectActiveRemoteAsync restores the control client of the project
// on screen after a window reopen, without blocking the new window.
func (a *app) reconnectActiveRemoteAsync() {
	host := a.serverAlias
	if host == "" {
		return
	}
	a.reconnectAsync(host, "Reconnecting to "+a.hostLabel(host)+"…", nil)
}

// reconnectWorkspaceAsync reconnects before switching, behind the same
// loading state as opening a project.
func (a *app) reconnectWorkspaceAsync(ws *workspace, host string) {
	if ws == nil || a.remoteBusy {
		return
	}
	if err := a.saveNow(); err != nil {
		a.dropPickerRemote()
		a.fail(fmt.Errorf("could not save the open project: %w", err))
		return
	}
	a.reconnectAsync(host, "Reconnecting to "+a.hostLabel(host)+"…", func(*remote.Conn) {
		a.switchWorkspace(ws)
	})
}

// closeRemoteProject forgets one project from the remembered list. When
// it is loaded it is unloaded too; closing the open project returns to
// the last local project, and the host's connection is released when no
// project or terminal on it remains.
func (a *app) closeRemoteProject(host, path string) {
	a.projectOpen = false
	key := workspaceKeyRemote(host, path)
	a.initWorkspaces()
	active := a.serverAlias == host && a.remotePath == path
	ws := a.workspaces[key]
	if active {
		if err := a.saveNow(); err != nil {
			a.fail(fmt.Errorf("could not save the open project: %w", err))
			return
		}
		a.dropPaneOverlays(nil)
	}
	if ws != nil {
		closeWorkspaceTerminals(ws)
		delete(a.workspaces, key)
	}
	a.hostService().ForgetProject(host, path)
	if a.remotePickHost == host {
		a.remotePickHost = ""
	}
	if active {
		a.serverAlias, a.remotePath = "", ""
		a.abandonActiveWorkspace()
		a.updateTitle()
		a.later(func() {
			if a.project != "" {
				a.openProject(a.project)
			}
		})
	}
	if !a.hostUsed(host) {
		delete(a.remoteKeep, host)
	}
	a.releaseRemote(host)
	a.saveSettings()
	a.invalidate()
}

// lastLoadedProjectOn returns the most recently opened project on host
// that is still in the workspace cache. The remembered list puts the
// latest first, so choosing a server after working on another project
// resumes the one that was left open.
func (a *app) lastLoadedProjectOn(host string) (remoteProject, bool) {
	a.initWorkspaces()
	for _, p := range a.remoteProjects() {
		if p.Host != host || p.Path == "" {
			continue
		}
		if a.workspaces[workspaceKeyRemote(host, p.Path)] != nil {
			return p, true
		}
	}
	return remoteProject{}, false
}

// switchToServer is the server chip's choice: it resumes host's loaded
// project instead of asking for a folder again, and opens the folder
// picker only when nothing on that server is loaded. Choosing the server
// that is already on screen still opens the picker, so another folder on
// it can be opened.
func (a *app) switchToServer(host string) {
	if host == "" {
		return
	}
	if a.serverAlias != host {
		if p, ok := a.lastLoadedProjectOn(host); ok {
			a.openRemoteProjectAsync(p.Host, p.Path)
			return
		}
	}
	a.openRemotePicker(host)
}

// openRemoteProjectAsync opens a remote project away from the UI thread,
// with a loading state. A project already in the workspace cache switches
// at once, without connecting again.
func (a *app) openRemoteProjectAsync(host, remotePath string) {
	a.remoteOpening = false
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
		a.dropPickerRemote()
		return
	}
	a.initWorkspaces()
	key := workspaceKeyRemote(host, remotePath)
	if ws := a.workspaces[key]; ws != nil {
		a.rememberRemoteProject(remoteProject{Host: host, Path: remotePath})
		if a.workspaceReachable(ws) {
			a.keepRemote(host)
			a.switchWorkspace(ws)
			a.dropPickerRemote()
			a.saveSettings()
		} else {
			a.dropPickerRemote()
			a.reconnectWorkspaceAsync(ws, host)
		}
		return
	}
	if err := a.saveNow(); err != nil {
		a.dropPickerRemote()
		a.fail(fmt.Errorf("could not save the open project: %w", err))
		return
	}
	if a.layoutPath != "" {
		a.saveLayout()
	}
	a.layoutDirty = false
	existing := a.remoteForHost(host)
	seq, win, ctx := a.beginRemoteTask("Opening " + filepath.Base(remotePath) + " on " + a.hostLabel(host) + "…")
	if seq == 0 {
		a.dropPickerRemote()
		return
	}
	cache := a.serverAssetCache(host, remotePath)
	go func() {
		rc := existing
		var err error
		if rc == nil {
			rc, err = remote.DialContext(ctx, host)
		}
		var s *store
		if err == nil {
			onWindow(win, func() {
				a.setRemoteBusyMsg(seq, "Opening "+filepath.Base(remotePath)+" on "+a.hostLabel(host)+"…")
			})
			s, err = openServerStore(rc.Client, remotePath, cache, false)
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
				a.dropPickerRemote()
				a.fail(err)
				return
			}
			if rc != nil {
				a.installRemoteConn(rc)
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
			if a.remotePickHost == host {
				a.remotePickHost = ""
			}
			a.keepRemote(host)
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
	for _, q := range a.remoteProjects() {
		if q.Host == p.Host && q.Path == p.Path {
			continue
		}
		out = append(out, q)
	}
	if len(out) > 12 {
		out = out[:12]
	}
	a.hostService().SetProjects(out)
}

// beginRemoteTask covers the window with a loading state and returns the
// sequence that guards its result and the context that stops its work.
func (a *app) beginRemoteTask(msg string) (int, *mygo.Window, context.Context) {
	if a.remoteBusy {
		return 0, nil, nil
	}
	a.remoteTaskSeq++
	ctx, cancel := context.WithCancel(context.Background())
	a.remoteCancel = cancel
	a.remoteBusy = true
	a.remoteBusyMsg = msg
	a.invalidate()
	if a.win != nil {
		return a.remoteTaskSeq, a.win, ctx
	}
	return a.remoteTaskSeq, a.settingsWin, ctx
}

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
func (a *app) endRemoteTask(seq int) bool {
	if a.remoteTaskSeq != seq {
		return false
	}
	a.remoteBusy = false
	a.remoteBusyMsg = ""
	a.stopRemoteTask()
	a.invalidate()
	return true
}

// cancelRemoteTask stops the connection behind the loading card and
// uncovers the window; every result that task still produces is dropped
// by its sequence.
func (a *app) cancelRemoteTask() {
	if !a.remoteBusy {
		return
	}
	a.remoteTaskSeq++
	a.remoteBusy = false
	a.remoteBusyMsg = ""
	a.stopRemoteTask()
	// A picker connection made for the folder that was about to open has
	// no user once the card is gone. The open project's host is kept by
	// remoteKeep, so this does not release it.
	a.dropPickerRemote()
	a.invalidate()
}

func (a *app) stopRemoteTask() {
	if a.remoteCancel != nil {
		a.remoteCancel()
		a.remoteCancel = nil
	}
}

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
func (a *app) remoteBusyView(c *ui.Context, k *colors) {
	if !a.remoteBusy {
		return
	}
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Center().Background(k.Backdrop).Modal()
		back.Children(func() {
			panel := ui.Row(c).Width(390).Padding(18, 20).Gap(14).Radius(radius).
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
				a.pickerButton(c, k, "Cancel", false, func() {
					a.cancelRemoteTask()
				})
			})
		})
	})
}

// remoteProject is a project on an SSH host, remembered by the switcher.
