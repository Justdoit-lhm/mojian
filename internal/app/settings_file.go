package app

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"mojian/internal/fs"
	"mojian/internal/prefs"
)

// settingsFile is what the app keeps between launches.
type settingsFile struct {
	prefs.Prefs
	Shortcuts      map[string]string `json:"shortcuts,omitempty"`
	Project        string            `json:"project,omitempty"`
	Projects       []string          `json:"projects,omitempty"`
	Server         string            `json:"server,omitempty"`
	RemotePath     string            `json:"remotePath,omitempty"`
	RemoteProjects []remoteProject   `json:"remoteProjects,omitempty"`
	Hosts          []hostConfig      `json:"hosts,omitempty"`
}

// loadSettings reads the preferences and the project the app was left in.
// Unmarshalling starts from the defaults, so a file written by an older
// version keeps sane values for fields it did not know.
func (a *app) loadSettings() {
	if a.settingsPath == "" {
		return
	}
	data, err := os.ReadFile(a.settingsPath)
	if err != nil {
		return
	}
	s := settingsFile{Prefs: prefs.Default()}
	if json.Unmarshal(data, &s) != nil {
		return
	}
	if !remoteSupported() {
		s.Server, s.RemotePath = "", ""
		s.RemoteProjects, s.Hosts = nil, nil
	}
	a.Prefs = s.Prefs
	a.Prefs.Sanitize()
	a.Theme = themeName(a.Theme)
	a.syncAppearance()
	a.shortcutKeys = s.Shortcuts
	a.settings.IndexCache = nil
	a.settings.IndexDirty = true
	a.project = s.Project
	a.projects = s.Projects
	a.serverAlias = s.Server
	a.remotePath = s.RemotePath
	a.hostService().SetProjects(s.RemoteProjects)
	a.hostService().SetHosts(s.Hosts)
}

// saveSettingsSoon debounces a write while a slider is dragged: the
// value changes on every pointer frame, but the file needs to land once
// the drag settles.
func (a *app) saveSettingsSoon() {
	if a.win == nil && a.settingsWin == nil {
		a.saveSettings()
		return
	}
	if a.settingsTimer == nil {
		a.settingsTimer = time.AfterFunc(400*time.Millisecond, func() {
			a.onUpdateUI(func() {
				a.saveSettings()
				a.invalidate()
			})
		})
		return
	}
	a.settingsTimer.Reset(400 * time.Millisecond)
}

func (a *app) saveSettings() {
	if a.settingsPath == "" {
		return
	}
	project := a.project
	if a.remotePath == "" && a.store != nil && !a.store.IsMemory() {
		project = a.store.Dir
	}
	p := a.Prefs
	p.Sanitize()
	server, remotePath := a.serverAlias, a.remotePath
	remoteProjects, hosts := a.remoteProjects(), a.hosts()
	if !remoteSupported() {
		server, remotePath, remoteProjects, hosts = "", "", nil, nil
	}
	data, _ := json.MarshalIndent(settingsFile{
		Prefs:          p,
		Shortcuts:      a.shortcutKeys,
		Project:        project,
		Projects:       a.projects,
		Server:         server,
		RemotePath:     remotePath,
		RemoteProjects: remoteProjects,
		Hosts:          hosts,
	}, "", "  ")
	a.settings.IndexDirty = true
	if err := fs.WriteFileAtomic(a.settingsPath, data); err != nil {
		log.Print(err)
	}
}

// fontSize and the other getters are the effective values: what the
// editor and terminal use when a preference is unset or zero.
func (a *app) fontSize() float64 {
	if a.EditorFontSize <= 0 {
		return prefs.Default().EditorFontSize
	}
	return a.EditorFontSize
}

func (a *app) termFontSize() float64 {
	if a.TerminalFontSize <= 0 {
		return prefs.Default().TerminalFontSize
	}
	return a.TerminalFontSize
}

func (a *app) lineHeight() float64 {
	if a.EditorLineHeight <= 0 {
		return prefs.Default().EditorLineHeight
	}
	return a.EditorLineHeight
}

func (a *app) tabSize() int {
	if a.EditorTabSize <= 0 {
		return prefs.Default().EditorTabSize
	}
	return a.EditorTabSize
}

func (a *app) termLineHeight() float64 {
	if a.TerminalLineHeight <= 0 {
		return prefs.Default().TerminalLineHeight
	}
	return a.TerminalLineHeight
}

func (a *app) termScrollback() int {
	if a.TerminalScrollback <= 0 {
		return prefs.Default().TerminalScrollback
	}
	return a.TerminalScrollback
}

// applyAppearance applies an appearance without saving it: the native
// chrome follows it, the window's background follows it, and the menu's
// checks follow it. Its callers decide whether the file changes.
func (a *app) applyAppearance(v int) {
	if v < 0 || v > 2 {
		return
	}
	a.Appearance = v
	a.syncAppearance()
	a.applyAppearanceSource()
	a.updateWindowBackground()
}

func (a *app) setAppearance(v int) {
	if v < 0 || v > 2 {
		return
	}
	a.applyAppearance(v)
	a.saveSettings()
	a.invalidate()
}

// setTheme applies a theme without saving it: every window, pane, terminal
// and open note takes its colors from it.
func (a *app) setTheme(id string) {
	id = themeName(id)
	a.Theme = id
	a.edTheme = nil
	a.rethemeTerminals()
	a.updateWindowBackground()
}

func (a *app) syncAppearance() {
	for i, it := range a.appearanceItems {
		if it != nil {
			it.SetChecked(i == a.Appearance)
		}
	}
}

// askQuit asks whether to quit after a save failed. The dialog runs off
// the UI thread, as MyGo's dialogs may; proceed runs from the UI thread.
