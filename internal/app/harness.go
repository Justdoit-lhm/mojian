package app

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo/ui"
)

// Harness views render the app headlessly for cmd/harness, which serves
// them to an MCP client (mygo-mcp). Each fixture gets its own disposable
// project, never the user's data directory or layout: the endpoint is
// local and unauthenticated, so its handlers may only touch test data.

// harnessFonts registers the embedded fonts once for every fixture in
// this process.
var harnessFonts sync.Once

// HarnessWindowView returns the main window's view over a seeded project,
// with the welcome note open and the Explorer showing prose and code.
func HarnessWindowView() (view func(*ui.Context), cleanup func(), err error) {
	a, cleanup, err := newHarnessApp()
	if err != nil {
		return nil, nil, err
	}
	return a.view, cleanup, nil
}

// HarnessSettingsView returns the Settings window's view, with the same
// seeded project behind it so the About page reads like a real one.
func HarnessSettingsView() (view func(*ui.Context), cleanup func(), err error) {
	a, cleanup, err := newHarnessApp()
	if err != nil {
		return nil, nil, err
	}
	return a.settingsWindowView, cleanup, nil
}

// newHarnessApp builds the state a first launch has: preferences at their
// defaults, a project on disk, and no window.
func newHarnessApp() (*app, func(), error) {
	harnessFonts.Do(registerFonts)
	dir, err := os.MkdirTemp("", "mojian-harness-*")
	if err != nil {
		return nil, nil, err
	}
	remove := func() { _ = os.RemoveAll(dir) }
	if err := seedHarnessProject(dir); err != nil {
		remove()
		return nil, nil, err
	}
	s, err := openStoreAt(dir)
	if err != nil {
		remove()
		return nil, nil, err
	}
	a := &app{workspace: &workspace{store: s}, prefs: defaultPrefs(), harness: true}
	a.settingsPath = filepath.Join(dir, "settings.json")
	a.layoutPath = filepath.Join(dir, "layout.json")
	a.treeOpen = true
	a.openDirs = map[string]bool{"Ideas": true, "scripts": true}
	// A terminal would spawn a session server from this binary, which is
	// the harness rather than the app. Keep the endpoint to the views.
	a.shellStarter = func() (*termTab, error) {
		return nil, errors.New("the UI harness does not start terminals")
	}
	a.focusFirstNote()
	if n := s.byRelPath("Welcome.md"); n != nil && a.currentNote() != n {
		a.openNote(n)
	}
	return a, func() {
		a.closeWorkspaces()
		remove()
	}, nil
}

// seedHarnessProject writes the fixture files. Welcome.md comes last so a
// fresh project opens on it, as the app's own folder does.
func seedHarnessProject(dir string) error {
	files := []struct{ name, body string }{
		{"Reading notes.md", sampleNote},
		{"Ideas/Quiet mornings.md", "# Quiet mornings\n\nCoffee, a blank page, and no notifications.\n"},
		{"scripts/tasks.py", "#!/usr/bin/env python3\n\nTASKS = [\"write\", \"read\"]\n\n\ndef open_tasks():\n    return TASKS\n"},
		{"scripts/app.js", "const tasks = [];\n\nfunction openTasks() {\n  return tasks;\n}\n"},
		{"Welcome.md", welcomeNote},
	}
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(f.body), 0o644); err != nil {
			return err
		}
	}
	return nil
}
