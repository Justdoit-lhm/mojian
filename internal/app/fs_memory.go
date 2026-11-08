package app

import (
	"time"

	"mojian/internal/session"
)

// memoryFS is the browser's project: its notes live in the store, and
// there is no path on disk to read, copy or reveal.
type memoryFS struct{}

func (m *memoryFS) Kind() fsKind            { return fsMemory }
func (m *memoryFS) Root() string            { return "" }
func (m *memoryFS) Client() *session.Client { return nil }
func (m *memoryFS) CacheDir() string        { return "" }

func (m *memoryFS) List() ([]session.FileEntry, map[string]string, error) {
	return nil, nil, nil
}
func (m *memoryFS) Read(string) ([]byte, error) { return nil, roError() }
func (m *memoryFS) Write(string, []byte) error  { return nil }
func (m *memoryFS) Stat(string) (session.FileEntry, error) {
	return session.FileEntry{}, roError()
}
func (m *memoryFS) Mkdir(string) error              { return roError() }
func (m *memoryFS) Rename(string, string) error     { return roError() }
func (m *memoryFS) Copy(string, string) error       { return roError() }
func (m *memoryFS) Remove(string) error             { return roError() }
func (m *memoryFS) CopyFrom(string, string) error   { return roError() }
func (m *memoryFS) MoveFrom(string, string) error   { return roError() }
func (m *memoryFS) RemoveAbs(string) error          { return roError() }
func (m *memoryFS) Abs(string) (string, error)      { return "", roError() }
func (m *memoryFS) RelOf(string) (string, bool)     { return "", false }
func (m *memoryFS) SameFile(string, string) bool    { return false }
func (m *memoryFS) LocalPath(string) (string, bool) { return "", false }
func (m *memoryFS) Fetch(string, int64, time.Time, string) (string, error) {
	return "", roError()
}
func (m *memoryFS) Trash(string) error { return roError() }
