package app

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"mojian/internal/fs"
	"mojian/internal/project"
	"mojian/internal/welcome"
)

// defaultStoreDir is the folder of the app's own project: $MOJIAN_VAULT,
// else ~/Documents/Mojian.
func defaultStoreDir() (string, error) {
	if dir := os.Getenv("MOJIAN_VAULT"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents", "Mojian"), nil
}

// openDefaultStore opens the app's own project. In a browser there is no
// file system: the files live in memory for the session. A folder with no
// notes gets the welcome note; a project the user chose stays empty.
func (a *app) openDefaultStore() (*store, error) {
	dir, err := defaultStoreDir()
	if err != nil {
		return nil, err
	}
	s, err := a.openLocalStore(dir)
	if err != nil {
		return nil, err
	}
	if len(s.Notes) == 0 {
		if _, err := s.NewFileIn("", "Welcome.md", welcome.Note); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// openStore opens a project directly, without the session server. It is
// used by the browser and tests; macOS and Linux open projects
// through openLocalStore.
func openStore() (*store, error) {
	if runtime.GOOS == "js" {
		s := project.New(fs.NewMemory())
		s.Add(&note{Path: "Welcome.md", Title: "Welcome to Mojian", Saved: welcome.Note, Mod: time.Now(), Loaded: true})
		s.Add(&note{Path: "Reading notes.md", Title: "Reading notes", Saved: welcome.Sample, Mod: time.Now().Add(-time.Minute), Loaded: true})
		s.BuildTree()
		return s, nil
	}
	dir, err := defaultStoreDir()
	if err != nil {
		return nil, err
	}
	return openStoreAt(dir)
}

// openStoreAt opens dir as a project, creating the folder when it is
// missing. macOS and Linux open projects through the session server; the
// direct store now serves tests and the browser's memory fallback.
func openStoreAt(dir string) (*store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	s := project.New(fs.NewDirect(abs))
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	return s, nil
}
