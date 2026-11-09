package app

import (
	"path/filepath"
	"strings"
)

// Startup is what the command line asked the window to open: a project
// folder, and optionally an absolute file inside it to show.
type Startup struct {
	Dir  string
	File string
}

// openStartupFile shows the file the command line named, once its project
// is loaded, in the pane the restored layout left focused.
func (a *app) openStartupFile() {
	if a.startup.File == "" || a.store == nil {
		return
	}
	// The command line names the file under the folder it named, which
	// may be a symlink the store opened at its target.
	base := a.startup.Dir
	if base == "" {
		base = a.store.Dir
	}
	rel, err := filepath.Rel(base, a.startup.File)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return
	}
	a.openRel(rel)
}
