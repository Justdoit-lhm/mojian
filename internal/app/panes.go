package app

import (
	"fmt"
)

// tab returns the active workspace.
func (a *app) tab() *tab {
	if a.active < 0 || a.active >= len(a.tabs) {
		return nil
	}
	return a.tabs[a.active]
}

func (a *app) currentNote() *note {
	if t := a.tab(); t != nil && t.Focus != nil {
		return t.Focus.Note
	}
	return nil
}

// focusFirstNote opens the most recent note, in the first tab.
func (a *app) focusFirstNote() {
	if a.store == nil {
		return
	}
	notes := a.store.Sorted()
	if len(notes) == 0 {
		for _, f := range a.store.Assets {
			t := &tab{ID: nextID("t")}
			p := newAssetPane(f)
			t.Root = &node{Pane: p}
			t.Focus = p
			a.tabs = []*tab{t}
			a.active = 0
			a.focusReq = p
			a.revealTab = true
			a.revealRel(f.Rel)
			return
		}
		return
	}
	n := notes[0]
	if !a.loadNote(n) {
		return
	}
	a.tabs = []*tab{newTab(n)}
	if a.tabs[0].Focus != nil {
		a.tabs[0].Focus.Preview = a.MarkdownPreview && isMarkdownPath(n.Path)
	}
	a.active = 0
	a.focusReq = a.tabs[0].Focus
	a.revealTab = true
	a.revealInTree(n)
}

// newTab adds an empty workspace to the right of the tabs. It does not
// copy the current file; picking or creating a file fills it.
func (a *app) newTab() {
	a.git.Open = false
	t := &tab{ID: nextID("t")}
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = nil
	a.revealTab = true
	a.treeDraft = nil
	a.changed()
	a.invalidate()
}

// loadNote reads a remote note's text before its pane is built. A local
// note is already loaded; a remote one is fetched once.
func (a *app) loadNote(n *note) bool {
	if n == nil {
		return false
	}
	if a.store != nil {
		if err := a.store.LoadNote(n); err != nil {
			a.fail(fmt.Errorf("reading %s: %w", n.Rel, err))
			return false
		}
	}
	return true
}

// loadAsset fetches a remote picture or PDF into the local cache its
// viewer reads.
func (a *app) loadAsset(f *assetFile) bool {
	if f == nil {
		return false
	}
	if a.store != nil {
		if _, err := a.store.MaterializeAsset(f); err != nil {
			a.fail(fmt.Errorf("reading %s: %w", f.Rel, err))
			return false
		}
	}
	return true
}

// openNoteInTab shows n in a tab of its own.
func (a *app) openNoteInTab(n *note) {
	if !a.loadNote(n) {
		return
	}
	a.git.Open = false
	t := newTab(n)
	if t.Focus != nil {
		t.Focus.Preview = a.MarkdownPreview && isMarkdownPath(n.Path)
	}
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = t.Focus
	a.revealTab = true
	a.treeSel = n.Rel
	a.openPath(n.Rel)
	a.treeReveal = n.Rel
	a.changed()
	a.invalidate()
}

// openNote shows n in the focused pane, or in a new tab when none is
// open.
func (a *app) openNote(n *note) {
	if !a.loadNote(n) {
		return
	}
	a.git.Open = false
	a.treeDraft = nil
	a.treeQuery = ""
	t := a.tab()
	if t == nil {
		t = newTab(n)
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
		a.focusReq = t.Focus
		t.Focus.Preview = a.MarkdownPreview && isMarkdownPath(n.Path)
		a.revealTab = true
		a.revealRel(n.Rel)
		a.changed()
		a.invalidate()
		return
	}
	p := t.Focus
	if p != nil && p.Term != nil {
		p = t.firstEditorPane()
		if p == nil {
			a.openNoteInTab(n)
			return
		}
	}
	if p == nil {
		p = firstPane(t.Root)
	}
	if p != nil && p.Term != nil {
		// A tab can hold only terminals. A note needs an editor pane,
		// so give it a tab instead of replacing a shell.
		if editor := t.firstEditorPane(); editor != nil {
			p = editor
		} else {
			a.openNoteInTab(n)
			return
		}
	}
	if p == nil {
		p = newPane(n)
		t.Root = &node{Pane: p}
		t.Focus = p
	}
	if p.Note != n {
		p.state = nil
		p.md = nil
		p.Preview = a.MarkdownPreview && isMarkdownPath(n.Path)
	}
	releaseAsset(p)
	p.Note = n
	p.Asset = nil
	p.assetView = nil
	t.Focus = p
	a.focusReq = p
	a.revealRel(n.Rel)
	a.changed()
	a.invalidate()
}

// openRel opens a path from the project: a text file in the editor, or a
// picture and PDF in a viewer.
func (a *app) openRel(rel string) {
	if a.store == nil {
		return
	}
	if n := a.store.ByRelPath(rel); n != nil {
		a.openNote(n)
		return
	}
	if f := a.store.AssetByRel(rel); f != nil {
		a.openAsset(f)
	}
}

// openAssetInTab shows a picture or PDF in a tab of its own.
func (a *app) openAssetInTab(f *assetFile) {
	if !a.loadAsset(f) {
		return
	}
	a.git.Open = false
	t := &tab{ID: nextID("t")}
	p := newAssetPane(f)
	t.Root = &node{Pane: p}
	t.Focus = p
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = p
	a.revealTab = true
	a.treeSel = f.Rel
	a.openPath(f.Rel)
	a.treeReveal = f.Rel
	a.changed()
	a.invalidate()
}

// openAsset shows a picture or PDF in the focused pane, or in a new tab
// when none is open.
func (a *app) openAsset(f *assetFile) {
	if !a.loadAsset(f) {
		return
	}
	a.git.Open = false
	a.treeDraft = nil
	a.treeQuery = ""
	t := a.tab()
	if t == nil {
		t = &tab{ID: nextID("t")}
		p := newAssetPane(f)
		t.Root = &node{Pane: p}
		t.Focus = p
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
		a.focusReq = p
		a.revealTab = true
		a.revealRel(f.Rel)
		a.changed()
		a.invalidate()
		return
	}
	p := t.Focus
	if p != nil && p.Term != nil {
		p = t.firstEditorPane()
	}
	if p == nil {
		p = firstPane(t.Root)
	}
	if p != nil && p.Term != nil {
		// A tab can hold only terminals. A picture or PDF needs a viewer
		// pane, so give it a tab instead of replacing a shell.
		if editor := t.firstEditorPane(); editor != nil {
			p = editor
		} else {
			a.openAssetInTab(f)
			return
		}
	}
	if p == nil {
		p = newAssetPane(f)
		t.Root = &node{Pane: p}
		t.Focus = p
	}
	releaseAsset(p)
	p.Note = nil
	p.state = nil
	p.Asset = f
	p.assetView = &assetView{}
	p.Preview = false
	t.Focus = p
	a.focusReq = p
	a.revealRel(f.Rel)
	a.changed()
	a.invalidate()
}

// closeTab closes a workspace.
func (a *app) closeTab(t *tab) {
	a.dropPaneOverlays(t)
	i := -1
	for j, x := range a.tabs {
		if x == t {
			i = j
			break
		}
	}
	if i < 0 {
		return
	}
	for _, p := range t.panes() {
		a.endTerminalPane(p, true)
		releaseAsset(p)
	}
	a.tabs = append(a.tabs[:i], a.tabs[i+1:]...)
	wasActive := a.active == i
	switch {
	case len(a.tabs) == 0:
		a.active = -1
	case i < a.active:
		a.active--
	case wasActive:
		a.active = min(i, len(a.tabs)-1)
	}
	if a.rename == t {
		a.rename = nil
	}
	if next := a.tab(); next != nil {
		a.focusReq = next.editorPane()
		a.revealTab = true
		if next.Focus != nil {
			if next.Focus.Note != nil {
				a.revealRel(next.Focus.Note.Rel)
			} else if next.Focus.Asset != nil {
				a.revealRel(next.Focus.Asset.Rel)
			}
		}
	} else {
		a.focusReq = nil
	}
	a.reapRemotes()
	a.changed()
	a.invalidate()
}

// closePane closes the pane focused in its tab.
func (a *app) closePane(p *pane) {
	if p == nil || p.closing {
		return
	}
	t := a.tabOf(p)
	if t == nil {
		return
	}
	a.unzoom(t)
	rect := p.lastBounds
	if a.quitting || len(t.panes()) <= 1 || rect.W <= 1 || rect.H <= 1 {
		a.closePaneNow(t, p)
		return
	}
	// Take the pane out of the layout now, so the survivor expands at
	// once, and keep a copy of the card over its old place while it fades
	// and slides back the way it entered. The engine cannot keep an Exit
	// copy because the split node goes with the pane.
	p.closing = true
	p.closeStarted = false
	a.paneOverlays = append(a.paneOverlays, &paneOverlay{p: p, t: t, rect: rect, kind: paneCopyClosing})
	if !t.closePane(p) {
		a.closeTab(t)
		return
	}
	a.focusReq = t.Focus
	a.changed()
	a.invalidate()
}

// tabOf returns the tab holding p.
func (a *app) tabOf(p *pane) *tab {
	for _, t := range a.tabs {
		if t.contains(p) {
			return t
		}
	}
	return nil
}

// finishPaneOverlay drops a pane's copy once its fade has rested. A
// closing pane ends here; a pane a zoom hid stays loaded.
func (a *app) finishPaneOverlay(p *pane, kind paneCopy) {
	if p == nil {
		return
	}
	removed := false
	for i, o := range a.paneOverlays {
		if o.p == p && o.kind == kind {
			a.paneOverlays = append(a.paneOverlays[:i], a.paneOverlays[i+1:]...)
			removed = true
			break
		}
	}
	if kind == paneCopyHidden {
		if !removed {
			return
		}
		p.hideStarted = false
		a.invalidate()
		return
	}
	if !p.closing {
		return
	}
	a.endTerminalPane(p, true)
	releaseAsset(p)
	p.closing = false
	p.closeStarted = false
	a.reapRemotes()
	a.changed()
	a.invalidate()
}

// dropPaneOverlays finishes every copy a tab has, as the tab goes away or
// another one is shown.
func (a *app) dropPaneOverlays(t *tab) {
	for _, o := range append([]*paneOverlay(nil), a.paneOverlays...) {
		if t != nil && o.t != t {
			continue
		}
		a.finishPaneOverlay(o.p, o.kind)
	}
}

// dropHiddenOverlays drops the copies a zoom hid, without touching their
// panes, when the zoom changes direction before the fade rests.
func (a *app) dropHiddenOverlays(t *tab) {
	for _, o := range append([]*paneOverlay(nil), a.paneOverlays...) {
		if o.t == t && o.kind == paneCopyHidden {
			a.finishPaneOverlay(o.p, paneCopyHidden)
		}
	}
}

// closePaneNow removes the pane at once, ending its terminal and closing
// the tab when it was the last pane.
func (a *app) closePaneNow(t *tab, p *pane) {
	if p.closing {
		a.finishPaneOverlay(p, paneCopyClosing)
		return
	}
	a.unzoom(t)
	a.endTerminalPane(p, true)
	releaseAsset(p)
	if !t.closePane(p) {
		a.closeTab(t)
		return
	}
	a.focusReq = t.Focus
	a.reapRemotes()
	a.changed()
	a.invalidate()
}

func (a *app) toggleZoom() {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	if t.Zoom == t.Focus {
		a.unzoom(t)
	} else {
		a.zoom(t, t.Focus)
	}
	a.focusReq = t.Focus
	a.changed()
	a.invalidate()
}

// zoom fills the tab with p. The panes it hides fade over their old
// places, and the zoomed card, note or terminal, flies to the tab's box
// through its Match; a terminal's grid settles at each width on the way.
func (a *app) zoom(t *tab, p *pane) {
	a.dropHiddenOverlays(t)
	for _, other := range t.panes() {
		if other == p {
			continue
		}
		if r := other.lastBounds; r.W > 1 && r.H > 1 {
			a.paneOverlays = append(a.paneOverlays, &paneOverlay{p: other, t: t, rect: r, kind: paneCopyHidden})
		}
	}
	t.Zoom = p
}

// unzoom returns the split and fades in the panes the zoom hid. The
// zoomed card flies back through its Match; its siblings fade in where
// they showed.
func (a *app) unzoom(t *tab) {
	zoomed := t.Zoom
	a.dropHiddenOverlays(t)
	t.Zoom = nil
	for _, p := range t.panes() {
		if p == zoomed {
			continue
		}
		p.enter, p.enterX, p.enterY = true, 0, 0
	}
}

// clearZoom leaves the zoom without a fade, for a layout change that
// moves the panes at once.
func (a *app) clearZoom(t *tab) {
	if t == nil || t.Zoom == nil {
		return
	}
	a.dropHiddenOverlays(t)
	t.Zoom = nil
}

func (a *app) cycleTab(delta int) {
	if len(a.tabs) == 0 {
		return
	}
	a.git.Open = false
	a.active = (a.active + delta + len(a.tabs)) % len(a.tabs)
	a.revealTab = true
	a.treeDraft = nil
	if t := a.tab(); t != nil {
		a.focusReq = t.editorPane()
		if t.Focus != nil {
			a.revealInTree(t.Focus.Note)
		}
	}
	a.changed()
	a.invalidate()
}

// selectTab shows the tab at index i.
func (a *app) selectTab(i int) {
	if i < 0 || i >= len(a.tabs) {
		return
	}
	a.git.Open = false
	a.active = i
	a.revealTab = true
	a.treeDraft = nil
	if t := a.tab(); t != nil {
		a.focusReq = t.editorPane()
		if t.Focus != nil {
			a.revealInTree(t.Focus.Note)
		}
	}
	a.changed()
	a.invalidate()
}

// showTab shows the nth tab, from 1; 9 shows the last.
func (a *app) showTab(n int) {
	if len(a.tabs) == 0 {
		return
	}
	i := n - 1
	if n >= 9 {
		i = len(a.tabs) - 1
	}
	a.selectTab(i)
}

func (a *app) startRename(t *tab) {
	if t == nil {
		return
	}
	for i, x := range a.tabs {
		if x == t {
			a.active = i
			break
		}
	}
	a.rename = t
	a.renameText = t.Name
	a.renameFocus = false
	a.revealTab = true
	a.invalidate()
}

// updateTitle names the window after the focused note.
