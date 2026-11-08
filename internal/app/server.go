package app

import (
	"path/filepath"
	"runtime"
)

// localServerID names the session server on this machine in terminals
// and layout files. It is not an SSH host alias.
const localServerID = "local"

// openLocalStore opens a local project through the local session server,
// so files, Git and terminals share one server. The browser has no
// session server and keeps its in-memory editor.
func (a *app) openLocalStore(dir string) (*store, error) {
	if runtime.GOOS == "js" {
		return openStore()
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
