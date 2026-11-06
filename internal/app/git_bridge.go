package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// toggleGit shows the working-tree page or returns to the editor.
func (a *app) toggleGit() {
	a.settingsOpen = false
	a.git.Toggle(a)
	a.invalidate()
}

// gitPage draws the page; the page owns its state and its view.
func (a *app) gitPage(c *ui.Context, k *colors) {
	a.git.View(c, k, a)
}

// The Host methods give the Git page the project, a callback into the
// window, and the file menu.

func (a *app) ProjectDir() (string, bool) {
	if a.store == nil || a.store.memory {
		return "", false
	}
	return a.store.dir, true
}

func (a *app) Window() *mygo.Window { return a.win }

func (a *app) Invalidate() { a.invalidate() }

func (a *app) Fail(err error) { a.fail(err) }

func (a *app) HasPath(rel string) bool {
	return a.store != nil && (a.store.byRelPath(rel) != nil || a.store.assetByRel(rel) != nil)
}

func (a *app) OpenPath(rel string) { a.openRel(rel) }

func (a *app) FileMenu(m *ui.Menu, rel string) { a.nodeMenu(m, rel, false) }
