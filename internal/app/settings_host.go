package app

import (
	"strings"

	"mojian/internal/app/settingsui"
	"mojian/internal/remote"
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
)

// settingsHost adapts the window to settingsui.Host. It is the only place
// the Settings package reaches app state.
type settingsHost struct{ a *app }

func (h settingsHost) Invalidate()                  { h.a.invalidate() }
func (h settingsHost) ApplyTheme(c *ui.Context)     { h.a.applyTheme(c) }
func (h settingsHost) RunPending()                  { h.a.runPending() }
func (h settingsHost) SaveSettings()                { h.a.saveSettings() }
func (h settingsHost) SaveSettingsSoon()            { h.a.saveSettingsSoon() }
func (h settingsHost) Later(fn func())              { h.a.later(fn) }
func (h settingsHost) Fail(err error)               { h.a.fail(err) }
func (h settingsHost) RefreshMenu()                 { h.a.refreshMenu() }
func (h settingsHost) SyncAppearance()              { h.a.syncAppearance() }
func (h settingsHost) ProjectPath() string          { return h.a.projectPath() }
func (h settingsHost) RemoteName() string           { return h.a.remoteName() }
func (h settingsHost) SettingsPath() string         { return h.a.settingsPath }
func (h settingsHost) RemotePath() string           { return h.a.remotePath }
func (h settingsHost) ServerAlias() string          { return h.a.serverAlias }
func (h settingsHost) Store() *store                { return h.a.store }
func (h settingsHost) ActiveRemote() *remote.Conn   { return h.a.activeRemote() }
func (h settingsHost) Shortcuts() map[string]string { return h.a.shortcutKeys }
func (h settingsHost) ShowTabKeys() string          { return showTabKeys() }
func (h settingsHost) FontSize() float64            { return h.a.fontSize() }
func (h settingsHost) CheckForUpdates()             { h.a.checkForUpdates() }
func (h settingsHost) InstallAvailableUpdate()      { h.a.installAvailableUpdate() }
func (h settingsHost) RestartForUpdate()            { h.a.restartForUpdate() }

func (h settingsHost) RemoteBusyView(c *ui.Context, k *theme.Colors) {
	h.a.remoteBusyView(c, k)
}

func (h settingsHost) SetShortcut(id, value string) string {
	return h.a.setShortcutValue(id, value)
}

func (h settingsHost) ResetShortcut(id string) {
	delete(h.a.shortcutKeys, id)
}

func (h settingsHost) ResetShortcuts() { h.a.shortcutKeys = nil }

func (h settingsHost) Commands() []settingsui.Command {
	out := make([]settingsui.Command, 0, len(paletteCommands))
	for _, c := range paletteCommands {
		out = append(out, settingsui.Command{
			ID:        c.ID,
			Title:     c.Title,
			Accel:     c.Default(),
			Keys:      h.a.commandKeys(c),
			Available: commandAvailable(c),
		})
	}
	return out
}

func (h settingsHost) UpdateState() settingsui.UpdateState {
	st := settingsui.UpdateState{
		Checking:   h.a.updateChecking,
		Installing: h.a.updateInstalling,
		Ready:      h.a.updateReady,
		Checked:    h.a.updateChecked,
		Err:        h.a.updateErr,
		Summary:    h.a.updateSummary(),
	}
	if h.a.pendingUpdate != nil {
		st.PendingVersion = h.a.pendingUpdate.Version
	}
	return st
}

func (h settingsHost) SettingsHostsPage(c *ui.Context, k *theme.Colors) {
	h.a.settingsHostsPage(c, k)
}

// settingsWindowView renders Settings with the app as its host.
func (a *app) settingsWindowView(c *ui.Context) {
	a.initSettings()
	a.settings.WindowView(c)
}

// editorFont is the effective editor face, shared by the pane and the
// font row.
func (a *app) editorFont() ui.Font {
	f := noteFont
	if fam := strings.TrimSpace(a.EditorFontFamily); fam != "" {
		f.Family = fam + ", " + f.Family
	}
	f.Size = float32(a.fontSize())
	return f
}

func settingsButton(c *ui.Context, label string, primary bool) ui.Element {
	if primary {
		return ui.PrimaryButton(c, label)
	}
	return ui.Button(c, label)
}

// setShortcut and resetPreferences are the app-facing names tests and
// menus use; the Settings package owns the implementation.
func (a *app) setShortcut(id, accel string) {
	a.settings.Notice = a.setShortcutValue(id, accel)
}

func (a *app) initSettings() {
	a.settings.Host = settingsHost{a}
	a.settings.Prefs = &a.Prefs
}

func (a *app) resetPreferences() {
	a.initSettings()
	a.settings.ResetPreferences()
}
