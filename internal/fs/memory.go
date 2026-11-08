package fs

import (
	"time"

	"mojian/internal/session"
)

// Memory is the browser's project: its notes live in the store, and
// there is no path on disk to read, copy or reveal.
type Memory struct{}

// NewMemory returns the browser's project backend.
func NewMemory() *Memory { return &Memory{} }

func (m *Memory) Kind() Kind              { return KindMemory }
func (m *Memory) Root() string            { return "" }
func (m *Memory) Client() *session.Client { return nil }
func (m *Memory) CacheDir() string        { return "" }

func (m *Memory) List() ([]session.FileEntry, map[string]string, error) {
	return nil, nil, nil
}
func (m *Memory) Read(string) ([]byte, error) { return nil, errReadOnly() }
func (m *Memory) Write(string, []byte) error  { return nil }
func (m *Memory) Stat(string) (session.FileEntry, error) {
	return session.FileEntry{}, errReadOnly()
}
func (m *Memory) Mkdir(string) error              { return errReadOnly() }
func (m *Memory) Rename(string, string) error     { return errReadOnly() }
func (m *Memory) Copy(string, string) error       { return errReadOnly() }
func (m *Memory) Remove(string) error             { return errReadOnly() }
func (m *Memory) CopyFrom(string, string) error   { return errReadOnly() }
func (m *Memory) MoveFrom(string, string) error   { return errReadOnly() }
func (m *Memory) RemoveAbs(string) error          { return errReadOnly() }
func (m *Memory) Abs(string) (string, error)      { return "", errReadOnly() }
func (m *Memory) RelOf(string) (string, bool)     { return "", false }
func (m *Memory) SameFile(string, string) bool    { return false }
func (m *Memory) LocalPath(string) (string, bool) { return "", false }
func (m *Memory) Fetch(string, int64, time.Time, string) (string, error) {
	return "", errReadOnly()
}
func (m *Memory) Trash(string) error { return errReadOnly() }
