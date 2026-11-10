package app

import (
	"runtime"
	"time"

	"mojian/internal/app/explorer"
	"mojian/internal/project"
	"mojian/internal/session"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// explorerHost adapts the window to explorer.Host; the tree's state stays
// on the active workspace.
type explorerHost struct{ a *app }

func (h explorerHost) Store() *store                           { return h.a.store }
func (h explorerHost) CurrentNote() *note                      { return h.a.currentNote() }
func (h explorerHost) OpenRel(rel string)                      { h.a.openRel(rel) }
func (h explorerHost) OpenNote(n *note)                        { h.a.openNote(n) }
func (h explorerHost) OpenNoteInTab(n *note)                   { h.a.openNoteInTab(n) }
func (h explorerHost) OpenAssetInTab(f *assetFile)             { h.a.openAssetInTab(f) }
func (h explorerHost) Later(fn func())                         { h.a.later(fn) }
func (h explorerHost) Fail(err error)                          { h.a.fail(err) }
func (h explorerHost) Invalidate()                             { h.a.invalidate() }
func (h explorerHost) Changed()                                { h.a.changed() }
func (h explorerHost) CloseNotes(match func(*note) bool)       { h.a.closeNotes(match) }
func (h explorerHost) CloseAssets(match func(*assetFile) bool) { h.a.closeAssets(match) }
func (h explorerHost) AfterRefresh()                           { h.a.afterRefresh() }
func (h explorerHost) ReloadProject(force bool)                { h.a.reloadProject(force) }
func (h explorerHost) Refreshing() bool {
	return h.a.reloadStore != nil && h.a.reloadStore == h.a.store
}
func (h explorerHost) RefreshGit()                 { h.a.refreshGit() }
func (h explorerHost) GitOpen() bool               { return h.a.git.Open }
func (h explorerHost) AutoRefreshGit() bool        { return !h.a.NoGitAutoRefresh }
func (h explorerHost) FilteredNotes() []*note      { return h.a.filteredNotes() }
func (h explorerHost) ActiveWorkspaceKey() string  { return h.a.activeWorkspaceKey() }
func (h explorerHost) NativeWindow() bool          { return h.a.win != nil }
func (h explorerHost) RemotePath() string          { return h.a.remotePath }
func (h explorerHost) ServerAlias() string         { return h.a.serverAlias }
func (h explorerHost) Tip(label, id string) string { return h.a.tip(label, commandByID(id)) }

func (h explorerHost) ResetRenamedStates(notes []*note, before map[*note]string) {
	h.a.resetRenamedStates(notes, before)
}

func (h explorerHost) SystemClipboardFiles() []string { return h.a.systemClipboardFiles() }

func (h explorerHost) WriteClipboardFiles(paths ...string) error {
	return writeClipboardFiles(paths...)
}

func (h explorerHost) TreeOpen() bool                 { return h.a.treeOpen }
func (h explorerHost) SetTreeOpen(v bool)             { h.a.treeOpen = v }
func (h explorerHost) TreeQuery() string              { return h.a.treeQuery }
func (h explorerHost) TreeQueryPtr() *string          { return &h.a.treeQuery }
func (h explorerHost) SetTreeQuery(v string)          { h.a.treeQuery = v }
func (h explorerHost) TreeSel() string                { return h.a.treeSel }
func (h explorerHost) SetTreeSel(v string)            { h.a.treeSel = v }
func (h explorerHost) TreeReveal() string             { return h.a.treeReveal }
func (h explorerHost) SetTreeReveal(v string)         { h.a.treeReveal = v }
func (h explorerHost) OpenDirs() map[string]bool      { return h.a.openDirs }
func (h explorerHost) SetOpenDirs(v map[string]bool)  { h.a.openDirs = v }
func (h explorerHost) TreeDraft() *explorer.Draft     { return h.a.treeDraft }
func (h explorerHost) SetTreeDraft(v *explorer.Draft) { h.a.treeDraft = v }
func (h explorerHost) NextDraftSeq() int              { h.a.draftSeq++; return h.a.draftSeq }
func (h explorerHost) TreeSearch() *ui.Handle         { return &h.a.treeSearch }

func (h explorerHost) Clip() explorer.ClipState {
	return explorer.ClipState{
		Paths: h.a.fileClip, Cut: h.a.fileCut, Remote: h.a.fileClipRemote,
		Host: h.a.fileClipHost, Root: h.a.fileClipRoot,
	}
}

func (h explorerHost) SetClip(c explorer.ClipState) {
	h.a.fileClip, h.a.fileCut, h.a.fileClipRemote = c.Paths, c.Cut, c.Remote
	h.a.fileClipHost, h.a.fileClipRoot = c.Host, c.Root
}

// explorerUI returns the Explorer with this window as its host.
func (a *app) explorerUI() *explorer.UI {
	a.explorer.Host = explorerHost{a}
	return &a.explorer
}

func (a *app) sidebar(c *ui.Context, k *colors) { a.explorerUI().Sidebar(c, k) }
func (a *app) toggleTree()                      { a.explorerUI().ToggleTree() }

// focusReloadAfter is how long a focus refresh reuses the last listing;
// a remote project can take seconds to list.
const focusReloadAfter = 15 * time.Second

// refreshProject reads the project now; explicit refreshes and tests
// bypass the focus throttle.
func (a *app) refreshProject() { a.reloadProject(true) }

// refreshProjectOnFocus runs when the window gains focus; a project
// listed moments ago is reused, since a remote listing can be slow.
func (a *app) refreshProjectOnFocus() { a.reloadProject(false) }

// reloadProject reads the project's listing. A server project is listed
// away from the UI thread and applied when the listing arrives; a direct
// project is small enough to read in place. force skips the throttle.
func (a *app) reloadProject(force bool) {
	s := a.store
	if s == nil || s.IsMemory() {
		return
	}
	if !force && a.lastReloadFor == s && time.Since(a.lastReload) < focusReloadAfter {
		return
	}
	if a.reloadStore == s {
		return
	}
	a.lastReloadFor, a.lastReload = s, time.Now()
	if !s.IsServer() {
		if err := s.Refresh(); err != nil {
			a.fail(err)
			return
		}
		a.explorerUI().RefreshApplied()
		return
	}
	a.reloadStore = s
	a.reloadSeq++
	seq := a.reloadSeq
	win := a.win
	a.invalidate()
	if win == nil {
		entries, bodies, err := s.List()
		a.finishReload(s, seq, entries, bodies, err)
		return
	}
	go func() {
		entries, bodies, err := s.List()
		onWindow(win, func() {
			a.finishReload(s, seq, entries, bodies, err)
		})
	}()
}

// finishReload applies a listing on the UI thread; a result for a store
// that is no longer loaded is dropped.
func (a *app) finishReload(s *store, seq int, entries []session.FileEntry, bodies map[string]string, err error) {
	if a.reloadStore != s || a.reloadSeq != seq {
		return
	}
	a.reloadStore = nil
	if a.store != s {
		return
	}
	if err != nil {
		a.fail(err)
		a.invalidate()
		return
	}
	s.Apply(entries, bodies)
	a.explorerUI().RefreshApplied()
}
func (a *app) revealInTree(n *note)                      { a.explorerUI().RevealInTree(n) }
func (a *app) revealRel(rel string)                      { a.explorerUI().RevealRel(rel) }
func (a *app) startNewFile()                             { a.explorerUI().StartNewFile() }
func (a *app) openPath(rel string)                       { a.explorerUI().OpenPath(rel) }
func (a *app) startNewFileIn(dir string)                 { a.explorerUI().StartNewFileIn(dir) }
func (a *app) startNewFolder()                           { a.explorerUI().StartNewFolder() }
func (a *app) startRenameNode(rel string)                { a.explorerUI().StartRenameNode(rel) }
func (a *app) submitTreeDraft()                          { a.explorerUI().SubmitTreeDraft() }
func (a *app) cancelTreeDraft()                          { a.explorerUI().CancelTreeDraft() }
func (a *app) trash(rel string)                          { a.explorerUI().Trash(rel) }
func (a *app) trashSelection()                           { a.explorerUI().TrashSelection() }
func (a *app) renameSelection()                          { a.explorerUI().RenameSelection() }
func (a *app) selectRel(rel string)                      { a.explorerUI().SelectRel(rel) }
func (a *app) duplicate(rel string)                      { a.explorerUI().Duplicate(rel) }
func (a *app) nodeMenu(m *ui.Menu, rel string, dir bool) { a.explorerUI().NodeMenu(m, rel, dir) }
func (a *app) pasteInto(rel string)                      { a.explorerUI().PasteInto(rel) }
func (a *app) copySelectionTo(rel string, cut bool)      { a.explorerUI().CopySelectionTo(rel, cut) }
func (a *app) copyPath(rel string, absolute bool)        { a.explorerUI().CopyPath(rel, absolute) }
func (a *app) filePasteSource() (explorer.PasteSource, error) {
	return a.explorerUI().FilePasteSource()
}

// closeNotes closes every pane whose note matches, then drops empty tabs.
func (a *app) closeNotes(match func(*note) bool) {
	a.closePanesWhere(func(p *pane) bool { return p.Note != nil && match(p.Note) })
}

// closeAssets closes every pane whose asset matches.
func (a *app) closeAssets(match func(*assetFile) bool) {
	a.closePanesWhere(func(p *pane) bool { return p.Asset != nil && match(p.Asset) })
}

func (a *app) closePanesWhere(match func(*pane) bool) {
	activeTab := a.tab()
	activeIndex := a.active
	for _, t := range a.tabs {
		for _, p := range append([]*pane(nil), t.panes()...) {
			if match(p) {
				releaseAsset(p)
				t.closePane(p)
				if len(t.panes()) == 0 {
					break
				}
			}
		}
	}
	kept := make([]*tab, 0, len(a.tabs))
	for _, t := range a.tabs {
		if len(t.panes()) > 0 {
			kept = append(kept, t)
		}
	}
	a.tabs = kept
	if len(a.tabs) == 0 {
		a.active = -1
		a.focusReq = nil
		return
	}
	if activeTab != nil {
		for i, t := range a.tabs {
			if t == activeTab {
				a.active = i
				a.focusReq = t.editorPane()
				return
			}
		}
	}
	a.active = min(max(activeIndex, 0), len(a.tabs)-1)
	a.focusReq = a.tabs[a.active].editorPane()
}

// afterRefresh closes panes whose file left the project and rebinds asset
// panes the refresh replaced.
func (a *app) afterRefresh() {
	if a.store == nil {
		return
	}
	liveNotes := make(map[*note]bool, len(a.store.Notes))
	for _, n := range a.store.Notes {
		liveNotes[n] = true
	}
	a.closeNotes(func(n *note) bool { return !liveNotes[n] && !n.Dirty() })
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.Asset == nil {
				continue
			}
			live := a.store.Assets[p.Asset.Rel]
			if live == nil || live == p.Asset {
				continue
			}
			releaseAsset(p)
			p.Asset = live
			p.assetView = &assetView{}
			p.Preview = false
		}
	}
	a.closeAssets(func(f *assetFile) bool {
		_, ok := a.store.Assets[f.Rel]
		return !ok
	})
}

func (a *app) resetRenamedStates(notes []*note, before map[*note]string) {
	renamed := make(map[*note]bool, len(notes))
	shapeChanged := make(map[*note]bool)
	for _, n := range notes {
		renamed[n] = true
		if before[n] != editorShape(n.Path) {
			shapeChanged[n] = true
		}
	}
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.Note == nil || !renamed[p.Note] {
				continue
			}
			p.md = nil
			if shapeChanged[p.Note] {
				p.state = nil
			}
		}
	}
	if len(shapeChanged) == 0 {
		return
	}
	if p := a.focusPane(); p != nil {
		a.focusReq = p
	}
}

func editorShape(path string) string {
	lang := ""
	if l := editor.LanguageFor(path); l != nil {
		lang = l.Name
	}
	return lang + "/" + boolString(project.IsMarkdownPath(path))
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// The system clipboard is read and written through these seams, so a
// test can say what a Finder copy holds without touching the real one.
var readClipboardFiles = func() ([]string, error) { return mygo.Clipboard.ReadFiles() }
var writeClipboardFiles = func(paths ...string) error { return mygo.Clipboard.WriteFiles(paths...) }

func (a *app) systemClipboardFiles() []string {
	if a.win == nil || runtime.GOOS == "js" {
		return nil
	}
	files, err := readClipboardFiles()
	if err != nil {
		return nil
	}
	return files
}
