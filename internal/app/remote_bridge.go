package app

import (
	"fmt"
	"os"
	"path/filepath"

	"mojian/internal/fs"
	"mojian/internal/remote"
)

// The app reaches SSH connections through internal/remote; this file is
// where the pool and the open project meet that package.

// connectRemote prepares a connection to host without replacing the
// current server, so a failed or cancelled switch leaves the open project
// talking to the machine it came from.
func (a *app) connectRemote(host string) (*remote.Conn, error) {
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	if rc := a.remoteForHost(host); rc != nil {
		return rc, nil
	}
	rc, err := remote.Dial(host)
	if err != nil {
		return nil, err
	}
	a.installRemoteConn(rc)
	return rc, nil
}

// remoteForHost returns the live connection the pool owns for host,
// dropping it first when it has gone stale.
func (a *app) remoteForHost(host string) *remote.Conn {
	a.initWorkspaces()
	rc := a.remoteConn(host)
	if rc == nil {
		return nil
	}
	if rc.Healthy() {
		return rc
	}
	a.deleteRemoteConn(host)
	delete(a.remoteKeep, host)
	remote.Close(rc)
	return nil
}

// activeRemote is the connection of the project on screen, if it is a
// remote one. The pool owns it; nothing else keeps a copy.
func (a *app) activeRemote() *remote.Conn {
	if a.serverAlias == "" {
		return nil
	}
	return a.remoteConn(a.serverAlias)
}

// serverAssetCache is where pictures and PDFs of a server project are
// fetched for their viewers.
func (a *app) serverAssetCache(serverID, root string) string {
	base := a.dataDir
	if base == "" {
		base, _ = os.UserCacheDir()
	}
	return filepath.Join(base, "server-assets", fs.ShortHash(serverID+"\x00"+root))
}

func (a *app) detachRemoteConn(rc *remote.Conn) {
	if rc == nil {
		return
	}
	if a.remoteConn(rc.Host) == rc {
		a.deleteRemoteConn(rc.Host)
	}
	remote.Close(rc)
}
