package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// startTreeDraft opens a name field in the tree.
func (a *app) startTreeDraft(kind draftKind, parent, target, name string) {
	if a.store == nil {
		return
	}
	if a.store.isMemory() && kind != draftRename {
		a.fail(errors.New("files need the desktop app"))
		return
	}
	a.treeOpen = true
	a.treeQuery = ""
	a.treeSearch.CancelFocus()
	if kind == draftRename {
		parent = filepath.Dir(target)
		if parent == "." {
			parent = ""
		}
	} else {
		a.treeSel = parent
	}
	a.openPath(parent)
	a.draftSeq++
	a.treeDraft = &treeDraft{kind: kind, parent: parent, target: target, name: name, gen: a.draftSeq}
	a.treeReveal = target
	a.invalidate()
}

func (a *app) startNewFile() {
	a.startTreeDraft(draftFile, a.selectedDir(), "", "")
}

func (a *app) startNewFileIn(dir string) {
	a.startTreeDraft(draftFile, dir, "", "")
}

func (a *app) startNewFolder() {
	a.startTreeDraft(draftFolder, a.selectedDir(), "", "")
}

func (a *app) startRenameNode(rel string) {
	if rel == "" || a.store == nil {
		return
	}
	abs, err := a.store.absRel(rel)
	if err != nil {
		a.fail(err)
		return
	}
	a.startTreeDraft(draftRename, "", rel, filepath.Base(abs))
}

func (a *app) cancelTreeDraft() {
	a.treeDraft = nil
	a.invalidate()
}

// submitTreeDraft creates the named file or folder, or renames the target.
// submitTreeDraft creates the named file or folder, or renames the target.
func (a *app) submitTreeDraft() {
	d := a.treeDraft
	if d == nil || a.store == nil {
		return
	}
	name := strings.TrimSpace(d.name)
	if d.kind == draftRename && name == "" {
		a.treeDraft = nil
		return
	}
	var (
		rel     string
		n       *note
		err     error
		changed []*note
	)
	before := map[*note]string{}
	for _, n := range a.store.notes {
		before[n] = editorShape(n.Path)
	}
	switch d.kind {
	case draftFile:
		if name == "" {
			name = "untitled.txt"
		}
		n, err = a.store.newFileIn(d.parent, name, "")
		if n != nil {
			rel = n.Rel
		}
	case draftFolder:
		if name == "" {
			name = "New Folder"
		}
		rel, err = a.store.newFolder(d.parent, name)
	case draftRename:
		rel, changed, err = a.store.rename(d.target, name)
	}
	if err != nil {
		a.fail(err)
		return
	}
	a.treeDraft = nil
	a.treeSel = rel
	if d.kind == draftFolder {
		if a.openDirs == nil {
			a.openDirs = map[string]bool{}
		}
		a.openDirs[rel] = true
	}
	if d.kind == draftRename {
		a.remapOpenDirs(d.target, rel)
	}
	a.openPath(filepath.Dir(rel))
	a.treeReveal = rel
	if d.kind == draftFile && n != nil {
		a.openNote(n)
	}
	if d.kind == draftRename {
		a.resetRenamedStates(changed, before)
	}
	a.changed()
	a.invalidate()
}

// editorShape is what decides an editor's line numbers and wrapping.
// editorShape is what decides an editor's line numbers and wrapping.
func editorShape(path string) string {
	lang := ""
	if l := editorLanguage(path); l != nil {
		lang = l.Name
	}
	return fmt.Sprintf("%s/%t", lang, isMarkdownPath(path))
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
			// Markdown previews resolve images relative to the note's
			// folder, which a rename or move may have changed.
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

// selectedDir is the folder new files and folders go into: the selected
// folder, the parent of the selected file, or the project root.
