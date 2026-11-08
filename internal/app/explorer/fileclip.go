package explorer

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
)

func (u *UI) CopySelectionTo(rel string, cut bool) {
	if rel == "" || u.store() == nil {
		return
	}
	if u.store().IsRemote() {
		// A remote path is not a local file, so the clipboard is the
		// app's own; paste works within the same remote project.
		u.SetClip(ClipState{
			Paths: []string{rel}, Cut: cut, Remote: true,
			Host: u.serverAlias(), Root: u.remotePath(),
		})
		u.invalidate()
		return
	}
	abs, err := u.store().AbsRel(rel)
	if err != nil {
		u.fail(err)
		return
	}
	u.SetClip(ClipState{Paths: []string{abs}, Cut: cut})
	if u.nativeWindow() && runtime.GOOS != "js" {
		if err := u.WriteClipboardFiles(abs); err != nil {
			u.fail(err)
		}
	}
	u.invalidate()
}

// PasteSource is what a paste will copy or move: paths from the system
// clipboard for a local project, or the app's own remote clipboard.
type PasteSource struct {
	paths  []string
	move   bool
	remote bool
}

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

// FilePasteSource decides what a paste uses. A local project follows the
// system clipboard; the app's cut marker moves files only while the
// clipboard still names exactly those files. A remote project can only
// paste paths copied inside it, since the system clipboard holds local
// paths.
func (u *UI) FilePasteSource() (PasteSource, error) {
	if u.store() == nil {
		return PasteSource{}, errors.New("no project")
	}
	clip := u.Clip()
	remoteProject := u.store().IsRemote()
	if remoteProject {
		if !clip.Remote || len(clip.Paths) == 0 {
			if len(u.SystemClipboardFiles()) > 0 {
				return PasteSource{}, errors.New("a remote project cannot paste Finder files; copy them in the Explorer first")
			}
			return PasteSource{}, errors.New("the clipboard has no files from this remote project")
		}
		if clip.Host != u.serverAlias() || clip.Root != u.remotePath() {
			return PasteSource{}, fmt.Errorf("the clipboard belongs to %s:%s", clip.Host, clip.Root)
		}
		return PasteSource{paths: append([]string(nil), clip.Paths...), move: clip.Cut, remote: true}, nil
	}
	files := u.SystemClipboardFiles()
	switch {
	case clip.Cut && len(clip.Paths) > 0 && !clip.Remote && sameFiles(files, clip.Paths):
		// The cut is still the system clipboard's content: move it.
		return PasteSource{paths: append([]string(nil), clip.Paths...), move: true}, nil
	case len(files) > 0:
		return PasteSource{paths: files}, nil
	case len(clip.Paths) > 0 && !clip.Remote:
		// A clipboard that could not be read still pastes what the app
		// copied, as a copy unless the cut is still the app's own.
		return PasteSource{paths: append([]string(nil), clip.Paths...), move: clip.Cut}, nil
	}
	return PasteSource{}, errors.New("the clipboard has no files")
}

// PasteInto copies or moves the clipboard's files into the folder rel.
func (u *UI) PasteInto(rel string) {
	if u.store() == nil {
		return
	}
	dest := u.destinationFor(rel)
	src, err := u.FilePasteSource()
	if err != nil {
		u.fail(err)
		return
	}
	var first string
	var lastErr error
	for _, p := range src.paths {
		var newRel string
		var err error
		switch {
		case src.remote && src.move:
			newRel, err = u.store().MoveRel(p, dest)
		case src.remote:
			newRel = u.store().UniqueRel(dest, filepath.Base(p), true)
			err = u.store().FS.Copy(p, newRel)
		case src.move:
			newRel, err = u.store().MoveInto(p, dest)
		default:
			newRel, err = u.store().CopyInto(p, dest)
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
		clip := u.Clip()
		clip.Cut = false
		u.SetClip(clip)
	}
	if err := u.store().Refresh(); err != nil {
		u.fail(err)
		return
	}
	if lastErr != nil {
		u.fail(lastErr)
	}
	if first != "" {
		u.SelectRel(first)
	}
}

func (u *UI) CopyPath(rel string, absolute bool) {
	if rel == "" || u.store() == nil {
		return
	}
	if !u.nativeWindow() {
		return
	}
	if absolute {
		if abs, err := u.store().AbsRel(rel); err == nil {
			mygo.Clipboard.WriteText(abs)
		}
		return
	}
	mygo.Clipboard.WriteText(filepath.ToSlash(rel))
}
