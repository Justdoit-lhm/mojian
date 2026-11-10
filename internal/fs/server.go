package fs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mojian/internal/session"
)

// Server is a project's files reached through a session server. The app
// reads and writes them directly: there is no local copy of the project.
// A local project and a remote one use the same path; local says whether
// the server runs on this machine.
type Server struct {
	mu     sync.RWMutex
	client *session.Client
	root   string
	cache  string // where pictures and PDFs are fetched for their viewers
	local  bool   // the server runs on this machine
}

// NewServer returns a backend for a folder served by client. cache is
// where remote pictures and PDFs are fetched for their viewers.
func NewServer(client *session.Client, root, cache string, local bool) *Server {
	return &Server{client: client, root: root, cache: cache, local: local}
}

// SetClient points the backend at a new control client after the project
// reconnects.
func (r *Server) SetClient(client *session.Client) {
	r.mu.Lock()
	r.client = client
	r.mu.Unlock()
}

// ctl returns the current control client, safe while a background listing
// runs and the project reconnects.
func (r *Server) ctl() *session.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

func (r *Server) Kind() Kind {
	if r.local {
		return KindServerLocal
	}
	return KindServerRemote
}
func (r *Server) Root() string            { return r.root }
func (r *Server) Client() *session.Client { return r.ctl() }
func (r *Server) CacheDir() string        { return r.cache }

func (r *Server) Abs(rel string) (string, error) {
	clean, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	if clean == "" {
		return r.root, nil
	}
	return path.Join(r.root, filepath.ToSlash(clean)), nil
}

func (r *Server) RelOf(abs string) (string, bool) {
	if !r.local {
		return "", false
	}
	return relOfRoot(r.root, abs)
}

func (r *Server) SameFile(string, string) bool { return false }

func (r *Server) LocalPath(rel string) (string, bool) {
	if !r.local {
		return "", false
	}
	p, err := r.Abs(rel)
	return p, err == nil
}

func (r *Server) List() ([]session.FileEntry, map[string]string, error) {
	entries, err := r.ctl().ListTree(r.root)
	if err != nil {
		return nil, nil, fmt.Errorf("listing the server project: %w", err)
	}
	return entries, nil, nil
}

func (r *Server) Read(rel string) ([]byte, error) {
	var data []byte
	err := at(r, rel, func(p string) error {
		var err error
		data, err = r.ctl().ReadFile(p)
		return err
	})
	return data, err
}

func (r *Server) Write(rel string, data []byte) error {
	return at(r, rel, func(p string) error { return r.ctl().WriteFile(p, data) })
}

func (r *Server) Stat(rel string) (session.FileEntry, error) {
	p, err := r.Abs(rel)
	if err != nil {
		return session.FileEntry{}, err
	}
	e, err := r.ctl().Stat(p)
	if err != nil {
		return session.FileEntry{}, err
	}
	e.Rel = rel
	return e, nil
}

func (r *Server) Mkdir(rel string) error {
	return at(r, rel, r.ctl().Mkdir)
}

func (r *Server) Rename(oldRel, newRel string) error {
	return pair(r, oldRel, newRel, r.ctl().Rename)
}

func (r *Server) Copy(oldRel, newRel string) error {
	return pair(r, oldRel, newRel, r.ctl().Copy)
}

func (r *Server) Remove(rel string) error {
	return at(r, rel, r.ctl().Remove)
}

func (r *Server) CopyFrom(srcAbs, destRel string) error {
	if !r.local {
		return fmt.Errorf("a remote project cannot paste Finder files")
	}
	return at(r, destRel, func(dest string) error { return r.ctl().Copy(srcAbs, dest) })
}

func (r *Server) MoveFrom(srcAbs, destRel string) error {
	if !r.local {
		return fmt.Errorf("a remote project cannot move Finder files")
	}
	return at(r, destRel, func(dest string) error {
		if err := r.ctl().Rename(srcAbs, dest); err == nil {
			return nil
		}
		if err := r.ctl().Copy(srcAbs, dest); err != nil {
			return err
		}
		return r.ctl().Remove(srcAbs)
	})
}

func (r *Server) RemoveAbs(abs string) error { return r.ctl().Remove(abs) }

func (r *Server) Fetch(rel string, size int64, mod time.Time, hash string) (string, error) {
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
	local := filepath.Join(dir, ShortHash(rel)+strings.ToLower(filepath.Ext(rel)))
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

func (r *Server) Trash(rel string) error {
	p, err := r.Abs(rel)
	if err != nil {
		return err
	}
	if r.local {
		return TrashPath(p)
	}
	dir := path.Join(".mojian-trash", time.Now().Format("20060102-150405"))
	if err := r.ctl().Mkdir(path.Join(r.root, dir)); err != nil {
		return err
	}
	dest := path.Join(dir, path.Base(filepath.ToSlash(rel)))
	for i := 2; ; i++ {
		if _, err := r.ctl().Stat(path.Join(r.root, dest)); err != nil {
			break
		}
		stem, ext := SplitExt(path.Base(filepath.ToSlash(rel)))
		dest = path.Join(dir, fmt.Sprintf("%s %d%s", stem, i, ext))
	}
	return r.ctl().Rename(p, path.Join(r.root, dest))
}
