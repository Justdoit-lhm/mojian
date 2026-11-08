package app

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"mojian/internal/session"
)

// fsKind says which filesystem a project lives on.
type fsKind uint8

const (
	fsMemory fsKind = iota
	fsDirect
	fsServerLocal
	fsServerRemote
)

// fsBackend is where a project's files live: an in-memory store, a local
// folder, or a folder reached through a session server. Paths are relative
// to the project; Abs and LocalPath translate them for the OS and viewers.
type fsBackend interface {
	Kind() fsKind
	Root() string
	List() ([]session.FileEntry, map[string]string, error)
	Read(rel string) ([]byte, error)
	Write(rel string, data []byte) error
	Stat(rel string) (session.FileEntry, error)
	Mkdir(rel string) error
	Rename(oldRel, newRel string) error
	Copy(oldRel, newRel string) error
	Remove(rel string) error
	CopyFrom(srcAbs, destRel string) error
	MoveFrom(srcAbs, destRel string) error
	RemoveAbs(abs string) error
	Abs(rel string) (string, error)
	RelOf(abs string) (string, bool)
	SameFile(oldRel, newRel string) bool
	LocalPath(rel string) (string, bool)
	Client() *session.Client
	CacheDir() string
	Fetch(rel string, size int64, mod time.Time, hash string) (string, error)
	Trash(rel string) error
}

func (s *store) isMemory() bool      { return s.fs.Kind() == fsMemory }
func (s *store) isServer() bool      { k := s.fs.Kind(); return k == fsServerLocal || k == fsServerRemote }
func (s *store) isLocalServer() bool { return s.fs.Kind() == fsServerLocal }
func (s *store) isRemote() bool      { return s.fs.Kind() == fsServerRemote }
func (s *store) client() *session.Client {
	return s.fs.Client()
}

// cleanRel keeps a relative project path inside the project.
func roError() error { return errors.New("files need the desktop app") }

func cleanRel(rel string) (string, error) {
	if rel == "" || rel == "." {
		return "", nil
	}
	clean := filepath.Clean(rel)
	if filepath.IsAbs(clean) || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path is outside the project")
	}
	return clean, nil
}

func joinRoot(root, rel string) (string, error) {
	clean, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	if clean == "" {
		return root, nil
	}
	p := filepath.Join(root, clean)
	r, err := filepath.Rel(root, p)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("path is outside the project")
	}
	return p, nil
}

func relOfRoot(root, p string) (string, bool) {
	if root == "" {
		return "", false
	}
	r, err := filepath.Rel(root, p)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", false
	}
	return r, true
}
