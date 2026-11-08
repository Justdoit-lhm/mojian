package app

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"mojian/internal/filekind"
	"mojian/internal/session"
)

// directFS is a project opened straight from a local folder. macOS and
// Linux open projects through the session server; the direct backend
// serves the browser's neighbours and the tests.
type directFS struct{ root string }

func (d *directFS) Kind() fsKind            { return fsDirect }
func (d *directFS) Root() string            { return d.root }
func (d *directFS) Client() *session.Client { return nil }
func (d *directFS) CacheDir() string        { return "" }

// List walks the folder and reads every text file, which keeps the
// direct project eager like it was before the server existed.
func (d *directFS) List() ([]session.FileEntry, map[string]string, error) {
	var entries []session.FileEntry
	bodies := map[string]string{}
	err := filepath.WalkDir(d.root, func(path string, de os.DirEntry, err error) error {
		if err != nil {
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

func (d *directFS) Read(rel string) ([]byte, error) {
	var data []byte
	err := at(d, rel, func(p string) error {
		var err error
		data, err = os.ReadFile(p)
		return err
	})
	return data, err
}

func (d *directFS) Write(rel string, data []byte) error {
	return at(d, rel, func(p string) error { return writeFileAtomic(p, data) })
}

func (d *directFS) Stat(rel string) (session.FileEntry, error) {
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

func (d *directFS) Mkdir(rel string) error {
	return at(d, rel, func(p string) error { return os.MkdirAll(p, 0o755) })
}

func (d *directFS) Rename(oldRel, newRel string) error {
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

func (d *directFS) Copy(oldRel, newRel string) error {
	return pair(d, oldRel, newRel, copyPath)
}

func (d *directFS) Remove(rel string) error {
	return at(d, rel, os.RemoveAll)
}

func (d *directFS) CopyFrom(srcAbs, destRel string) error {
	return at(d, destRel, func(dest string) error { return copyPath(srcAbs, dest) })
}

func (d *directFS) MoveFrom(srcAbs, destRel string) error {
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

func (d *directFS) RemoveAbs(abs string) error { return os.RemoveAll(abs) }

func (d *directFS) Abs(rel string) (string, error) { return joinRoot(d.root, rel) }

func (d *directFS) RelOf(abs string) (string, bool) { return relOfRoot(d.root, abs) }

func (d *directFS) SameFile(oldRel, newRel string) bool {
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

func (d *directFS) LocalPath(rel string) (string, bool) {
	p, err := d.Abs(rel)
	return p, err == nil
}

func (d *directFS) Fetch(rel string, _ int64, _ time.Time, _ string) (string, error) {
	return d.Abs(rel)
}

func (d *directFS) Trash(rel string) error {
	return at(d, rel, trashPath)
}
