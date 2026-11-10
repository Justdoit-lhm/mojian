package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"mojian/internal/filekind"
	"mojian/internal/session"

	"github.com/egoist/mygo"
)

// Direct is a project opened straight from a local folder. macOS and
// Linux open projects through the session server; the direct backend
// serves the browser's neighbours and the tests.
type Direct struct{ root string }

// NewDirect opens root as a direct project.
func NewDirect(root string) *Direct { return &Direct{root: root} }

func (d *Direct) Kind() Kind              { return KindDirect }
func (d *Direct) Root() string            { return d.root }
func (d *Direct) Client() *session.Client { return nil }
func (d *Direct) CacheDir() string        { return "" }

// List walks the folder and reads every text file, which keeps the
// direct project eager like it was before the server existed.
func (d *Direct) List() ([]session.FileEntry, map[string]string, error) {
	var entries []session.FileEntry
	bodies := map[string]string{}
	err := filepath.WalkDir(d.root, func(path string, de os.DirEntry, err error) error {
		if err != nil {
			if path == d.root {
				// A denied root is an error, not an empty project.
				return err
			}
			return nil // an unreadable folder is skipped, not fatal
		}
		if de.IsDir() {
			if path != d.root && filekind.SkipDir(de.Name()) {
				return filepath.SkipDir
			}
			rel, relErr := filepath.Rel(d.root, path)
			if relErr != nil || rel == "." {
				return nil
			}
			info, infoErr := de.Info()
			if infoErr != nil {
				return nil
			}
			entries = append(entries, session.FileEntry{
				Rel: filepath.ToSlash(rel), Dir: true, Mod: info.ModTime(),
			})
			return nil
		}
		kind, kindErr := filekind.ClassifyFile(path)
		if kindErr != nil || kind == filekind.Other {
			return nil
		}
		rel, relErr := filepath.Rel(d.root, path)
		if relErr != nil {
			return nil
		}
		info, infoErr := de.Info()
		if infoErr != nil {
			return nil
		}
		e := session.FileEntry{
			Rel: filepath.ToSlash(rel), Size: info.Size(), Mod: info.ModTime(), Kind: kind,
		}
		entries = append(entries, e)
		if kind == filekind.Text {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			bodies[filepath.FromSlash(e.Rel)] = string(data)
		}
		return nil
	})
	return entries, bodies, err
}

func (d *Direct) Read(rel string) ([]byte, error) {
	var data []byte
	err := at(d, rel, func(p string) error {
		var err error
		data, err = os.ReadFile(p)
		return err
	})
	return data, err
}

func (d *Direct) Write(rel string, data []byte) error {
	return at(d, rel, func(p string) error { return WriteFileAtomic(p, data) })
}

func (d *Direct) Stat(rel string) (session.FileEntry, error) {
	p, err := d.Abs(rel)
	if err != nil {
		return session.FileEntry{}, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return session.FileEntry{}, err
	}
	e := session.FileEntry{Rel: filepath.ToSlash(rel), Dir: info.IsDir(), Size: info.Size(), Mod: info.ModTime()}
	if !info.IsDir() {
		e.Kind, _ = filekind.ClassifyFile(p)
	}
	return e, nil
}

func (d *Direct) Mkdir(rel string) error {
	return at(d, rel, func(p string) error { return os.MkdirAll(p, 0o755) })
}

func (d *Direct) Rename(oldRel, newRel string) error {
	oldAbs, err := d.Abs(oldRel)
	if err != nil {
		return err
	}
	newAbs, err := d.Abs(newRel)
	if err != nil {
		return err
	}
	info, err := os.Lstat(oldAbs)
	if err != nil {
		return err
	}
	if existing, err := os.Lstat(newAbs); err == nil {
		if os.SameFile(info, existing) {
			// A case-only rename on a case-insensitive volume needs a step
			// aside before the new spelling can take the old name.
			tmp := oldAbs + ".mojian-rename"
			if err := os.Rename(oldAbs, tmp); err != nil {
				return err
			}
			if err := os.Rename(tmp, newAbs); err != nil {
				_ = os.Rename(tmp, oldAbs)
				return err
			}
			return nil
		}
		return errors.New(filepath.Base(newAbs) + " already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(oldAbs, newAbs)
}

func (d *Direct) Copy(oldRel, newRel string) error {
	return pair(d, oldRel, newRel, copyPath)
}

func (d *Direct) Remove(rel string) error {
	return at(d, rel, os.RemoveAll)
}

func (d *Direct) CopyFrom(srcAbs, destRel string) error {
	return at(d, destRel, func(dest string) error { return copyPath(srcAbs, dest) })
}

func (d *Direct) MoveFrom(srcAbs, destRel string) error {
	return at(d, destRel, func(dest string) error {
		if err := os.Rename(srcAbs, dest); err == nil {
			return nil
		}
		if err := copyPath(srcAbs, dest); err != nil {
			return err
		}
		return os.RemoveAll(srcAbs)
	})
}

func (d *Direct) RemoveAbs(abs string) error { return os.RemoveAll(abs) }

func (d *Direct) Abs(rel string) (string, error) { return joinRoot(d.root, rel) }

func (d *Direct) RelOf(abs string) (string, bool) { return relOfRoot(d.root, abs) }

func (d *Direct) SameFile(oldRel, newRel string) bool {
	oldAbs, err := d.Abs(oldRel)
	if err != nil {
		return false
	}
	newAbs, err := d.Abs(newRel)
	if err != nil {
		return false
	}
	oldInfo, err := os.Lstat(oldAbs)
	if err != nil {
		return false
	}
	newInfo, err := os.Lstat(newAbs)
	return err == nil && os.SameFile(oldInfo, newInfo)
}

func (d *Direct) LocalPath(rel string) (string, bool) {
	p, err := d.Abs(rel)
	return p, err == nil
}

func (d *Direct) Fetch(rel string, _ int64, _ time.Time, _ string) (string, error) {
	return d.Abs(rel)
}

func (d *Direct) Trash(rel string) error {
	return at(d, rel, TrashPath)
}

// TrashPath moves a path to the system trash; tests replace it.
var TrashPath = func(path string) error { return mygo.Shell.TrashItem(path) }

// copyPath copies a file, a symlink or a folder.
func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}
