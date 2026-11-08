package explorer

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"mojian/internal/platform"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (u *UI) selectedDir() string {
	if u.store() == nil || u.TreeSel() == "" {
		return ""
	}
	if u.store().DirExists(u.TreeSel()) {
		return u.TreeSel()
	}
	dir := filepath.Dir(u.TreeSel())
	if dir == "." {
		return ""
	}
	if u.store().DirExists(dir) {
		return dir
	}
	return ""
}

// destinationFor returns the folder a paste into rel goes to.
func (u *UI) destinationFor(rel string) string {
	if rel == "" {
		return ""
	}
	if u.store().DirExists(rel) {
		return rel
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return ""
	}
	return dir
}

func (u *UI) Duplicate(rel string) {
	if rel == "" || u.store() == nil || u.store().IsMemory() {
		return
	}
	newRel, err := u.store().DuplicateRel(rel, u.destinationFor(rel))
	if err != nil {
		u.fail(err)
		return
	}
	u.SelectRel(newRel)
}

func (u *UI) reveal(rel string) {
	if u.store() == nil || runtime.GOOS == "js" {
		return
	}
	if u.store().IsRemote() {
		u.fail(errors.New("remote projects have no Finder window"))
		return
	}
	abs, err := u.store().AbsRel(rel)
	if err != nil {
		u.fail(err)
		return
	}
	mygo.Shell.ShowItemInFolder(abs)
}

func (u *UI) RenameSelection() { u.StartRenameNode(u.TreeSel()) }

func (u *UI) TrashSelection() { u.Trash(u.TreeSel()) }

func (u *UI) Trash(rel string) {
	if rel == "" || u.store() == nil {
		return
	}
	if runtime.GOOS == "js" || u.store().IsMemory() {
		u.fail(errors.New("Trash needs the desktop app"))
		return
	}
	if err := u.store().FS.Trash(rel); err != nil {
		u.fail(err)
		return
	}
	u.closeNotes(func(n *note) bool {
		return n.Rel == rel || strings.HasPrefix(n.Rel, rel+string(filepath.Separator))
	})
	u.closeAssets(func(f *assetFile) bool {
		return f.Rel == rel || strings.HasPrefix(f.Rel, rel+string(filepath.Separator))
	})
	u.store().Forget(rel)
	if u.store().IsServer() {
		if err := u.store().Refresh(); err != nil {
			u.fail(err)
			return
		}
	}
	u.SetTreeSel(u.destinationFor(rel))
	u.OpenPath(u.TreeSel())
	u.SetTreeReveal(u.TreeSel())
	u.changed()
	u.invalidate()
}

// SelectRel selects a tree row and reveals it.
func (u *UI) SelectRel(rel string) {
	u.SetTreeSel(rel)
	u.SetTreeReveal(rel)
	u.OpenPath(filepath.Dir(rel))
	u.invalidate()
}

// NodeMenu fills a folder or file's context menu.
func (u *UI) NodeMenu(m *ui.Menu, rel string, dir bool) {
	u.SetTreeSel(rel)
	if dir {
		if m.Item("New File…").Chosen() {
			u.later(func() { u.StartNewFileIn(rel) })
		}
		if m.Item("New Folder…").Chosen() {
			u.later(func() { u.startTreeDraft(DraftFolder, rel, "", "") })
		}
		m.Separator()
		if m.Item("Paste").Chosen() {
			u.later(func() { u.PasteInto(rel) })
		}
	} else {
		if m.Item("Open in Current Window").Chosen() {
			u.later(func() { u.openRel(rel) })
		}
		if m.Item("Open in New Tab").Chosen() {
			u.later(func() {
				if n := u.store().ByRelPath(rel); n != nil {
					u.openNoteInTab(n)
				} else if f := u.store().AssetByRel(rel); f != nil {
					u.openAssetInTab(f)
				}
			})
		}
	}
	if rel == "" {
		m.Separator()
		if m.Item("Reveal Project").Chosen() {
			u.later(func() { u.reveal(rel) })
		}
		return
	}
	m.Separator()
	if m.Item("Rename…").Shortcut(0, ui.KeyF2).Chosen() {
		u.later(func() { u.StartRenameNode(rel) })
	}
	if m.Item("Duplicate").Chosen() {
		u.later(func() { u.Duplicate(rel) })
	}
	m.Separator()
	if m.Item("Copy").Chosen() {
		u.later(func() { u.CopySelectionTo(rel, false) })
	}
	if m.Item("Cut").Chosen() {
		u.later(func() { u.CopySelectionTo(rel, true) })
	}
	m.Separator()
	if m.Item("Copy Relative Path").Chosen() {
		u.later(func() { u.CopyPath(rel, false) })
	}
	if m.Item("Copy Path").Chosen() {
		u.later(func() { u.CopyPath(rel, true) })
	}
	if runtime.GOOS != "js" && m.Item(platform.RevealLabel()).Chosen() {
		u.later(func() { u.reveal(rel) })
	}
	m.Separator()
	if m.Item("Move to Trash").Shortcut(ui.Cmd, ui.KeyBackspace).Chosen() {
		u.later(func() { u.Trash(rel) })
	}
}
