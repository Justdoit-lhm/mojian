// Package fs holds a project's files: the backend interface the app
// stores, and the direct-folder, in-memory and session-server backends.
package fs

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"mojian/internal/session"
)

// Kind says which filesystem a project lives on.
type Kind uint8

const (
	KindMemory Kind = iota
	KindDirect
	KindServerLocal
	KindServerRemote
)

// Backend is where a project's files live: an in-memory store, a local
// folder, or a folder reached through a session server. Paths are relative
// to the project; Abs and LocalPath translate them for the OS and viewers.
type Backend interface {
	Kind() Kind
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

// errReadOnly is what a project without a filesystem reports.
func errReadOnly() error { return errors.New("files need the desktop app") }

// pathOf is the one thing the path helpers need: a backend that resolves
// project paths to absolute ones.
type pathOf interface {
	Abs(rel string) (string, error)
}

// at resolves rel and runs fn on the absolute path. Every backend's file
// operation has this shape.
func at(p pathOf, rel string, fn func(abs string) error) error {
	abs, err := p.Abs(rel)
	if err != nil {
		return err
	}
	return fn(abs)
}

// pair resolves two project paths and runs fn on them.
func pair(p pathOf, oldRel, newRel string, fn func(oldAbs, newAbs string) error) error {
	oldAbs, err := p.Abs(oldRel)
	if err != nil {
		return err
	}
	newAbs, err := p.Abs(newRel)
	if err != nil {
		return err
	}
	return fn(oldAbs, newAbs)
}

// cleanRel keeps a relative project path inside the project.
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
