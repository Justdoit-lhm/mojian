package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
)

func (a *app) copySelectionTo(rel string, cut bool) {
	if rel == "" || a.store == nil {
		return
	}
	if a.store.isRemote() {
		// A remote path is not a local file, so the clipboard is the
		// app's own; paste works within the same remote project.
		a.fileClip = []string{rel}
		a.fileClipRemote = true
		a.fileClipHost = a.serverAlias
		a.fileClipRoot = a.remotePath
		a.fileCut = cut
		a.invalidate()
		return
	}
	abs, err := a.store.absRel(rel)
	if err != nil {
		a.fail(err)
		return
	}
	a.fileClip = []string{abs}
	a.fileClipRemote = false
	a.fileClipHost = ""
	a.fileClipRoot = ""
	a.fileCut = cut
	if a.win != nil && runtime.GOOS != "js" {
		if err := writeClipboardFiles(abs); err != nil {
			a.fail(err)
		}
	}
	a.invalidate()
}

// The system clipboard is read and written through these seams, so a
// test can say what a Finder copy holds without touching the real one.
var readClipboardFiles = func() ([]string, error) { return mygo.Clipboard.ReadFiles() }
var writeClipboardFiles = func(paths ...string) error { return mygo.Clipboard.WriteFiles(paths...) }

// pasteSource is what a paste will copy or move: paths from the system
// clipboard for a local project, or the app's own remote clipboard.
// pasteSource is what a paste will copy or move: paths from the system
// clipboard for a local project, or the app's own remote clipboard.
type pasteSource struct {
	paths  []string
	move   bool
	remote bool
}

// sameFiles reports whether two file lists name the same set, so a cut
// remains valid only while the system clipboard still holds it.
// sameFiles reports whether two file lists name the same set, so a cut
// remains valid only while the system clipboard still holds it.
func sameFiles(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	count := make(map[string]int, len(a))
	for _, p := range a {
		count[filepath.Clean(p)]++
	}
	for _, p := range b {
		c := filepath.Clean(p)
		if count[c] == 0 {
			return false
		}
		count[c]--
	}
	return true
}

// filePasteSource decides what a paste uses. A local project follows the
// system clipboard; the app's cut marker moves files only while the
// clipboard still names exactly those files. A remote project can only
// paste paths copied inside it, since the system clipboard holds local
// paths.
// filePasteSource decides what a paste uses. A local project follows the
// system clipboard; the app's cut marker moves files only while the
// clipboard still names exactly those files. A remote project can only
// paste paths copied inside it, since the system clipboard holds local
// paths.
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

func (a *app) filePasteSource() (pasteSource, error) {
	if a.store == nil {
		return pasteSource{}, errors.New("no project")
	}
	remoteProject := a.store.isRemote()
	if remoteProject {
		if !a.fileClipRemote || len(a.fileClip) == 0 {
			if len(a.systemClipboardFiles()) > 0 {
				return pasteSource{}, errors.New("a remote project cannot paste Finder files; copy them in the Explorer first")
			}
			return pasteSource{}, errors.New("the clipboard has no files from this remote project")
		}
		if a.fileClipHost != a.serverAlias || a.fileClipRoot != a.remotePath {
			return pasteSource{}, fmt.Errorf("the clipboard belongs to %s:%s", a.fileClipHost, a.fileClipRoot)
		}
		return pasteSource{paths: append([]string(nil), a.fileClip...), move: a.fileCut, remote: true}, nil
	}
	files := a.systemClipboardFiles()
	switch {
	case a.fileCut && len(a.fileClip) > 0 && !a.fileClipRemote && sameFiles(files, a.fileClip):
		// The cut is still the system clipboard's content: move it.
		return pasteSource{paths: append([]string(nil), a.fileClip...), move: true}, nil
	case len(files) > 0:
		return pasteSource{paths: files}, nil
	case len(a.fileClip) > 0 && !a.fileClipRemote:
		// A clipboard that could not be read still pastes what the app
		// copied, as a copy unless the cut is still the app's own.
		return pasteSource{paths: append([]string(nil), a.fileClip...), move: a.fileCut}, nil
	}
	return pasteSource{}, errors.New("the clipboard has no files")
}

// pasteInto copies or moves the clipboard's files into the folder rel.
func (a *app) pasteInto(rel string) {
	if a.store == nil {
		return
	}
	dest := a.destinationFor(rel)
	src, err := a.filePasteSource()
	if err != nil {
		a.fail(err)
		return
	}
	var first string
	var lastErr error
	for _, p := range src.paths {
		var newRel string
		var err error
		switch {
		case src.remote && src.move:
			newRel, err = a.store.moveRel(p, dest)
		case src.remote:
			newRel = a.store.uniqueRel(dest, filepath.Base(p), true)
			err = a.store.fs.Copy(p, newRel)
		case src.move:
			newRel, err = a.store.moveInto(p, dest)
		default:
			newRel, err = a.store.copyInto(p, dest)
		}
		if err != nil {
			lastErr = err
			continue
		}
		if first == "" {
			first = newRel
		}
	}
	if src.move || !src.remote {
		// A move consumed the cut; a local copy also settles a cut that
		// no longer matches the system clipboard.
		a.fileCut = false
	}
	if err := a.store.refresh(); err != nil {
		a.fail(err)
		return
	}
	if lastErr != nil {
		a.fail(lastErr)
	}
	if first != "" {
		a.selectRel(first)
	}
}

func (a *app) copyPath(rel string, absolute bool) {
	if rel == "" || a.store == nil {
		return
	}
	if a.win == nil {
		return
	}
	if absolute {
		if abs, err := a.store.absRel(rel); err == nil {
			mygo.Clipboard.WriteText(abs)
		}
		return
	}
	mygo.Clipboard.WriteText(filepath.ToSlash(rel))
}
