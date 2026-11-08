package app

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

// settingsFile is what the app keeps between launches.
type settingsFile struct {
	prefs
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
	s := settingsFile{prefs: defaultPrefs()}
	if json.Unmarshal(data, &s) != nil {
		return
	}
	a.prefs = s.prefs
	a.prefs.sanitize()
	a.syncAppearance()
	a.shortcutKeys = s.Shortcuts
	a.settingsIndexCache = nil
	a.settingsIndexDirty = true
	a.project = s.Project
	a.projects = s.Projects
	a.serverAlias = s.Server
	a.remotePath = s.RemotePath
	a.remoteProjects = s.RemoteProjects
	a.hosts = s.Hosts
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
	if a.remotePath == "" && a.store != nil && !a.store.isMemory() {
		project = a.store.dir
	}
	p := a.prefs
	p.sanitize()
	data, _ := json.MarshalIndent(settingsFile{
		prefs:          p,
		Shortcuts:      a.shortcutKeys,
		Project:        project,
		Projects:       a.projects,
		Server:         a.serverAlias,
		RemotePath:     a.remotePath,
		RemoteProjects: a.remoteProjects,
		Hosts:          a.hosts,
	}, "", "  ")
	a.settingsIndexDirty = true
	if err := writeFileAtomic(a.settingsPath, data); err != nil {
		log.Print(err)
	}
}

// writeFileAtomic replaces path with data, so a crash leaves the old
// file or the new one, never half of each.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// fontSize and the other getters are the effective values: what the
// editor and terminal use when a preference is unset or zero.
func (a *app) fontSize() float64 {
	if a.EditorFontSize <= 0 {
		return prefsDefaults.EditorFontSize
	}
	return a.EditorFontSize
}

func (a *app) termFontSize() float64 {
	if a.TerminalFontSize <= 0 {
		return prefsDefaults.TerminalFontSize
	}
	return a.TerminalFontSize
}

func (a *app) lineHeight() float64 {
	if a.EditorLineHeight <= 0 {
		return prefsDefaults.EditorLineHeight
	}
	return a.EditorLineHeight
}

func (a *app) tabSize() int {
	if a.EditorTabSize <= 0 {
		return prefsDefaults.EditorTabSize
	}
	return a.EditorTabSize
}

func (a *app) termLineHeight() float64 {
	if a.TerminalLineHeight <= 0 {
		return prefsDefaults.TerminalLineHeight
	}
	return a.TerminalLineHeight
}

func (a *app) termScrollback() int {
	if a.TerminalScrollback <= 0 {
		return prefsDefaults.TerminalScrollback
	}
	return a.TerminalScrollback
}

func (a *app) setAppearance(v int) {
	if v < 0 || v > 2 {
		return
	}
	a.Appearance = v
	a.syncAppearance()
	a.saveSettings()
	a.invalidate()
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
