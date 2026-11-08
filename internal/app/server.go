package app

import (
	"fmt"
	"path/filepath"
	"runtime"

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
