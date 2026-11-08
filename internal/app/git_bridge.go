package app

import (
	"errors"
	"path"
	"strings"
	"time"

	"mojian/internal/session"

	gitpage "mojian/internal/git"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// toggleGit shows the working-tree page or returns to the editor.
func (a *app) toggleGit() {
	a.git.Toggle(a)
	a.invalidate()
}

// refreshGit rereads the working tree for the open page.
func (a *app) refreshGit() { a.git.Refresh() }

// gitPage draws the page; the page owns its state and its view.
func (a *app) gitPage(c *ui.Context, k *colors) {
	a.git.View(c, k, a)
}

// The Host methods give the Git page the project, a callback into the
// window, and the file menu.
func (a *app) ProjectDir() (string, bool) {
	if a.store == nil || a.store.isMemory() {
		return "", false
	}
	return a.store.dir, true
}

// GitRunner runs Git where the project lives: locally, or on a server.
func (a *app) GitRunner() gitpage.Runner {
	if a.store != nil && a.store.isServer() {
		return serverGitRunner{client: a.store.client()}
	}
	return gitpage.LocalRunner{}
}

type serverGitRunner struct{ client *session.Client }

func (r serverGitRunner) Run(dir string, args ...string) (string, error) {
	cmd := append([]string{"git", "-C", dir}, args...)
	res, err := r.client.Exec(dir, cmd, nil, 10*time.Second)
	if err != nil {
		return res.Stdout, err
	}
	if res.Code != 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = "git failed"
		}
		return res.Stdout, errors.New(msg)
	}
	return res.Stdout, nil
}

func (r serverGitRunner) RealPath(path string) (string, error) {
	res, err := r.client.Exec("", []string{"sh", "-c", `cd "$1" && pwd -P`, "sh", path}, nil, 5*time.Second)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (r serverGitRunner) ReadFile(dir, rel string, limit int) ([]byte, error) {
	data, err := r.client.ReadFile(path.Join(dir, rel))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return data[:limit+1], nil
	}
	return data, nil
}

func (a *app) Window() *mygo.Window { return a.win }

func (a *app) Invalidate() { a.invalidate() }

func (a *app) Fail(err error) { a.fail(err) }

func (a *app) HasPath(rel string) bool {
	return a.store != nil && (a.store.byRelPath(rel) != nil || a.store.assetByRel(rel) != nil)
}

func (a *app) OpenPath(rel string) { a.openRel(rel) }

func (a *app) FileMenu(m *ui.Menu, rel string) { a.nodeMenu(m, rel, false) }
