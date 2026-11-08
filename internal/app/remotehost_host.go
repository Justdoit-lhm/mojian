package app

import (
	"mojian/internal/remote"
	"mojian/internal/remotehost"
)

type hostConfig = remotehost.Host
type remoteProject = remotehost.Project

func (a *app) hostService() *remotehost.Service {
	if a.remoteHosts == nil {
		a.remoteHosts = remotehost.New()
	}
	return a.remoteHosts
}

func (a *app) hosts() []hostConfig                       { return a.hostService().Hosts() }
func (a *app) remoteProjects() []remoteProject           { return a.hostService().Projects() }
func (a *app) remoteConn(host string) *remote.Conn       { return a.hostService().Conn(host) }
func (a *app) setRemoteConn(c *remote.Conn) bool         { return a.hostService().SetConn(c) }
func (a *app) deleteRemoteConn(host string) *remote.Conn { return a.hostService().DeleteConn(host) }
