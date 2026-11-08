package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"mojian/internal/session"
)

// serverFS is a project's files reached through a session server. The
// app reads and writes them directly: there is no local copy of the
// project. A local project and a remote one use the same path; local says
// whether the server runs on this machine.
type serverFS struct {
	client *session.Client
	root   string
	cache  string // where pictures and PDFs are fetched for their viewers
	local  bool   // the server runs on this machine
}

func (r *serverFS) Kind() fsKind {
	if r.local {
		return fsServerLocal
	}
	return fsServerRemote
}
func (r *serverFS) Root() string            { return r.root }
func (r *serverFS) Client() *session.Client { return r.client }
func (r *serverFS) CacheDir() string        { return r.cache }

func (r *serverFS) Abs(rel string) (string, error) {
	clean, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	if clean == "" {
		return r.root, nil
	}
	return path.Join(r.root, filepath.ToSlash(clean)), nil
}

func (r *serverFS) RelOf(abs string) (string, bool) {
	if !r.local {
		return "", false
	}
	return relOfRoot(r.root, abs)
}

func (r *serverFS) SameFile(string, string) bool { return false }

func (r *serverFS) LocalPath(rel string) (string, bool) {
	if !r.local {
		return "", false
	}
	p, err := r.Abs(rel)
	return p, err == nil
}

func (r *serverFS) List() ([]session.FileEntry, map[string]string, error) {
	entries, err := r.client.ListTree(r.root)
	if err != nil {
		return nil, nil, fmt.Errorf("listing the server project: %w", err)
	}
	return entries, nil, nil
}

func (r *serverFS) Read(rel string) ([]byte, error) {
	var data []byte
	err := at(r, rel, func(p string) error {
		var err error
		data, err = r.client.ReadFile(p)
		return err
	})
	return data, err
}

func (r *serverFS) Write(rel string, data []byte) error {
	return at(r, rel, func(p string) error { return r.client.WriteFile(p, data) })
}

func (r *serverFS) Stat(rel string) (session.FileEntry, error) {
	p, err := r.Abs(rel)
	if err != nil {
		return session.FileEntry{}, err
	}
	e, err := r.client.Stat(p)
	if err != nil {
		return session.FileEntry{}, err
	}
	e.Rel = rel
	return e, nil
}

func (r *serverFS) Mkdir(rel string) error {
	return at(r, rel, r.client.Mkdir)
}

func (r *serverFS) Rename(oldRel, newRel string) error {
	return pair(r, oldRel, newRel, r.client.Rename)
}

func (r *serverFS) Copy(oldRel, newRel string) error {
	return pair(r, oldRel, newRel, r.client.Copy)
}

func (r *serverFS) Remove(rel string) error {
	return at(r, rel, r.client.Remove)
}

func (r *serverFS) CopyFrom(srcAbs, destRel string) error {
	if !r.local {
		return fmt.Errorf("a remote project cannot paste Finder files")
	}
	return at(r, destRel, func(dest string) error { return r.client.Copy(srcAbs, dest) })
}

func (r *serverFS) MoveFrom(srcAbs, destRel string) error {
	if !r.local {
		return fmt.Errorf("a remote project cannot move Finder files")
	}
	return at(r, destRel, func(dest string) error {
		if err := r.client.Rename(srcAbs, dest); err == nil {
			return nil
		}
		if err := r.client.Copy(srcAbs, dest); err != nil {
			return err
		}
		return r.client.Remove(srcAbs)
	})
}

func (r *serverFS) RemoveAbs(abs string) error { return r.client.Remove(abs) }

func (r *serverFS) Fetch(rel string, size int64, mod time.Time, hash string) (string, error) {
	if r.local {
		return r.Abs(rel)
	}
	if hash != "" {
		local := filepath.Join(r.cache, hash+strings.ToLower(filepath.Ext(rel)))
		if info, err := os.Stat(local); err == nil && info.Size() == size {
			return local, nil
		}
		data, err := r.Read(rel)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(r.cache, 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(local, data, 0o644); err != nil {
			return "", err
		}
		return local, nil
	}
	dir := filepath.Join(r.cache, fmt.Sprintf("%d-%d", mod.Unix(), size))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	local := filepath.Join(dir, shortHash(rel)+strings.ToLower(filepath.Ext(rel)))
	if info, err := os.Stat(local); err == nil && info.Size() == size {
		return local, nil
	}
	data, err := r.Read(rel)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(local, data, 0o644); err != nil {
		return "", err
	}
	return local, nil
}

func (r *serverFS) Trash(rel string) error {
	p, err := r.Abs(rel)
	if err != nil {
		return err
	}
	if r.local {
		return trashPath(p)
	}
	dir := path.Join(".mojian-trash", time.Now().Format("20060102-150405"))
	if err := r.client.Mkdir(path.Join(r.root, dir)); err != nil {
		return err
	}
	dest := path.Join(dir, path.Base(filepath.ToSlash(rel)))
	for i := 2; ; i++ {
		if _, err := r.client.Stat(path.Join(r.root, dest)); err != nil {
			break
		}
		stem, ext := splitExt(path.Base(filepath.ToSlash(rel)))
		dest = path.Join(dir, fmt.Sprintf("%s %d%s", stem, i, ext))
	}
	return r.client.Rename(p, path.Join(r.root, dest))
}

// openServerStore lists the project on the server and returns it as the
// open project. File contents are read when a file is opened, not when
// the tree is.
func openServerStore(client *session.Client, root, cache string, local bool) (*store, error) {
	if client == nil || root == "" {
		return nil, fmt.Errorf("server project needs a connection and a folder")
	}
	s := newStore(&serverFS{client: client, root: root, cache: cache, local: local})
	if err := s.refresh(); err != nil {
		return nil, err
	}
	return s, nil
}
