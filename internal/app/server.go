package app

import (
	"path/filepath"
	"runtime"
)

// localServerID names the session server on this machine in terminals
// and layout files. It is not an SSH host alias.
const localServerID = "local"

// openLocalStore opens a local project through the local session server,
// so files, Git and terminals share one server. Windows and the browser
// keep their direct paths: they have no session server.
func (a *app) openLocalStore(dir string) (*store, error) {
	if runtime.GOOS == "js" {
		return openStore()
	}
	if !terminalsSupported() {
		return openStoreAt(dir)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	client, err := a.sessions()
	if err != nil {
		return nil, err
	}
	return openServerStore(client, abs, a.serverAssetCache(localServerID, abs), true)
}
