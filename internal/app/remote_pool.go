package app

import "mojian/internal/remote"

// hostUsed reports whether a connection to host still has a user: a
// loaded project, a terminal pane, or the open project picker.
func (a *app) hostUsed(host string) bool {
	if host == "" {
		return false
	}
	if a.remotePickHost == host {
		return true
	}
	used := func(ws *workspace) bool {
		if ws == nil {
			return false
		}
		if ws.serverAlias == host && ws.remotePath != "" {
			return true
		}
		for _, t := range ws.tabs {
			for _, p := range t.panes() {
				if p.Term != nil && p.Term.Spec.Server == host {
					return true
				}
			}
		}
		return false
	}
	if used(a.workspace) {
		return true
	}
	for _, ws := range a.workspaces {
		if ws != a.workspace && used(ws) {
			return true
		}
	}
	return false
}

// keepRemote marks a host's connection as owned by setup or a loaded
// project: closing the picker and pane reaping do not release it.
func (a *app) keepRemote(host string) {
	if host == "" {
		return
	}
	if a.remoteKeep == nil {
		a.remoteKeep = map[string]bool{}
	}
	a.remoteKeep[host] = true
}

// releaseRemote closes host's pooled connection when nothing uses it any
// more. The server and its sessions keep running on the host.
func (a *app) releaseRemote(host string) {
	if host == "" || a.remoteBusy || a.remoteDirsBusy || a.hostUsed(host) || a.remoteKeep[host] {
		return
	}
	rc := a.remoteConn(host)
	if rc == nil {
		return
	}
	delete(a.remoteKeep, host)
	a.deleteRemoteConn(host)
	remote.Close(rc)
}

// reapRemotes drops every pooled connection whose last project or
// terminal has gone.
func (a *app) reapRemotes() {
	if a.remoteBusy || a.remoteDirsBusy {
		return
	}
	for host := range a.hostService().Conns() {
		a.releaseRemote(host)
	}
}
