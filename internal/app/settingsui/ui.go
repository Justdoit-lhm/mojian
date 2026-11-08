// Package settingsui renders Mojian's Settings window: the searchable
// rail, the table-driven pages, the keyboard recorder and the font picker.
// The hosts page stays in internal/app because it drives host management;
// the app reaches it through Host.SettingsHostsPage.
package settingsui

import (
	"sync"

	"mojian/internal/icons"
	"mojian/internal/platform"
	"mojian/internal/prefs"
	"mojian/internal/project"
	"mojian/internal/remote"
	"mojian/internal/settings"
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
)

// UI is the Settings window's state. It lives beside the app, not in it.
type UI struct {
	Host
	*prefs.Prefs

	Cat        int
	Found      map[int]bool
	Hits       map[string]bool
	Reveal     string
	Revealed   string
	Scroll     ui.ScrollState
	ScrollTop  bool
	Rows       int
	IndexCache []settingsEntry
	IndexDirty bool
	Search     ui.Handle
	Notice     string
	Recording  string
	RecordRef  ui.Handle
	RecordSeen bool
	KeyFilter  string
	ResetAsk   bool
	Query      string

	FontPick    bool
	FontPickFor int
	FontSearch  string
	FontsOnce   sync.Once
	Fonts       []ui.FontFamily
}

// Command is one command the Settings keyboard page lists and records.
type Command struct {
	ID        string
	Title     string
	Accel     string
	Keys      string
	Available bool
}

// UpdateState is the updater's state for the Updates page.
type UpdateState struct {
	Checking       bool
	Installing     bool
	Ready          bool
	Checked        bool
	Err            string
	PendingVersion string
	Summary        string
}

// Host is what Settings asks of the window; *app implements it through a
// small adapter.
type Host interface {
	Invalidate()
	ApplyTheme(*ui.Context)
	RemoteBusyView(*ui.Context, *colors)
	RunPending()
	SaveSettings()
	SaveSettingsSoon()
	Later(func())
	Fail(error)
	RefreshMenu()
	SyncAppearance()

	ProjectPath() string
	RemoteName() string
	SettingsPath() string
	RemotePath() string
	ServerAlias() string
	Store() *project.Store
	ActiveRemote() *remote.Conn

	Shortcuts() map[string]string
	SetShortcut(id, value string) string
	ResetShortcut(id string)
	ResetShortcuts()
	FontSize() float64
	Commands() []Command

	UpdateState() UpdateState
	CheckForUpdates()
	InstallAvailableUpdate()
	RestartForUpdate()

	SettingsHostsPage(*ui.Context, *colors)
}

type colors = theme.Colors

const radius = theme.Radius

var noteFont = theme.NoteFont

func colorsOf(c *ui.Context) *colors { return theme.Of(c) }

func paintBackground(p *ui.Painter, r ui.Rect, k *colors) { theme.PaintBackground(p, r, k) }

func brandMark(c *ui.Context) *ui.SVG {
	if c.Theme().Dark {
		return icons.Brand("mark-reversed")
	}
	return icons.Brand("mark")
}

func nativeKeys(s string) string { return platform.NativeKeys(s) }

func terminalsSupported() bool { return platform.TerminalsSupported() }

// settingsCatAvailable hides the pages of features the platform lacks.
func settingsCatAvailable(cat int) bool {
	if platform.TerminalsSupported() {
		return true
	}
	switch cat {
	case settings.Terminal, settings.Shell, settings.Notifications, settings.Hosts:
		return false
	}
	return true
}

// Lowercase wrappers keep the moved view code unchanged.

func (u *UI) invalidate()                             { u.Invalidate() }
func (u *UI) applyTheme(c *ui.Context)                { u.ApplyTheme(c) }
func (u *UI) remoteBusyView(c *ui.Context, k *colors) { u.RemoteBusyView(c, k) }
func (u *UI) runPending()                             { u.RunPending() }
func (u *UI) saveSettings()                           { u.SaveSettings() }
func (u *UI) saveSettingsSoon()                       { u.SaveSettingsSoon() }
func (u *UI) later(fn func())                         { u.Later(fn) }
func (u *UI) fail(err error)                          { u.Fail(err) }
func (u *UI) refreshMenu()                            { u.RefreshMenu() }
func (u *UI) syncAppearance()                         { u.SyncAppearance() }

func (u *UI) projectPath() string   { return u.ProjectPath() }
func (u *UI) remoteName() string    { return u.RemoteName() }
func (u *UI) settingsPath() string  { return u.SettingsPath() }
func (u *UI) remotePath() string    { return u.RemotePath() }
func (u *UI) serverAlias() string   { return u.ServerAlias() }
func (u *UI) store() *project.Store { return u.Store() }
func (u *UI) activeRemote() *remote.Conn {
	return u.ActiveRemote()
}

func (u *UI) shortcutKeys() map[string]string { return u.Shortcuts() }

func (u *UI) setShortcut(id, value string) string {
	notice := u.SetShortcut(id, value)
	if notice != "" {
		u.Notice = notice
	}
	return notice
}

func (u *UI) resetShortcuts() { u.ResetShortcuts() }

func (u *UI) fontSize() float64 { return u.FontSize() }

func (u *UI) commands() []Command { return u.Commands() }

func (u *UI) updateChecking() bool                       { return u.UpdateState().Checking }
func (u *UI) updateInstalling() bool                     { return u.UpdateState().Installing }
func (u *UI) updateReady() bool                          { return u.UpdateState().Ready }
func (u *UI) updateChecked() bool                        { return u.UpdateState().Checked }
func (u *UI) updateError() string                        { return u.UpdateState().Err }
func (u *UI) pendingVersion() string                     { return u.UpdateState().PendingVersion }
func (u *UI) updateSummary() string                      { return u.UpdateState().Summary }
func (u *UI) checkForUpdates()                           { u.CheckForUpdates() }
func (u *UI) installAvailableUpdate()                    { u.InstallAvailableUpdate() }
func (u *UI) restartForUpdate()                          { u.RestartForUpdate() }
func (u *UI) settingsHostsPage(c *ui.Context, k *colors) { u.SettingsHostsPage(c, k) }

// Section, InfoRow and NoticeLine are the primitives the app's Hosts page
// uses while it stays in internal/app.
func (u *UI) Section(c *ui.Context, k *colors, title string, fn func()) {
	u.settingsSection(c, k, title, fn)
}

func (u *UI) InfoRow(c *ui.Context, k *colors, title, value string) {
	u.settingsInfoRow(c, k, title, value)
}

func (u *UI) NoticeLine(c *ui.Context, k *colors) {
	u.settingsNoticeLine(c, k)
}
