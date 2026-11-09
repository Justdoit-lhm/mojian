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
	rel, err := filepath.Rel(a.store.Dir, a.startup.File)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return
	}
	a.openRel(rel)
}
