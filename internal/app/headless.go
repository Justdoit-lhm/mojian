package app

import (
	"errors"
	"sync"

	"mojian/internal/prefs"
	"mojian/internal/project"

	"github.com/egoist/mygo/ui"
)

// Headless serves the real views to cmd/harness in a disposable project,
// with no native window or session server. It is a development tool and
// is never imported by the app itself.

var headlessFonts sync.Once

// Headless is an app state whose views an MCP client can render.
type Headless struct{ a *app }

// OpenStoreAt opens dir as a project for a headless fixture.
func OpenStoreAt(dir string) (*project.Store, error) { return openStoreAt(dir) }

// NewHeadless builds the state a first launch has: preferences at their
// defaults, a project on disk, and no window.
func NewHeadless(s *project.Store, settingsPath, layoutPath string) *Headless {
	headlessFonts.Do(registerFonts)
	a := &app{workspace: &workspace{store: s}, Prefs: prefs.Default(), harness: true}
	a.settingsPath = settingsPath
	a.layoutPath = layoutPath
	a.treeOpen = true
	a.openDirs = map[string]bool{"Ideas": true, "scripts": true}
	// A terminal would spawn a session server from this binary, which is
	// the harness rather than the app. Keep the endpoint to the views.
	a.shellStarter = func() (*termTab, error) {
		return nil, errors.New("the UI harness does not start terminals")
	}
	a.focusFirstNote()
	if n := s.ByRelPath("Welcome.md"); n != nil && a.currentNote() != n {
		a.openNote(n)
	}
	return &Headless{a: a}
}

// WindowView renders the main window.
func (h *Headless) WindowView(c *ui.Context) { h.a.view(c) }

// SettingsView renders the Settings window.
func (h *Headless) SettingsView(c *ui.Context) { h.a.settingsWindowView(c) }

// Close detaches workspaces and releases the fixture's state.
func (h *Headless) Close() { h.a.closeWorkspaces() }
