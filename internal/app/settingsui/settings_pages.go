package settingsui

import (
	"fmt"
	"runtime"

	"mojian/internal/icons"
	"mojian/internal/prefs"
	"mojian/internal/session"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (u *UI) settingsAdvancedPage(c *ui.Context, k *colors) {
	u.settingsNoticeLine(c, k)
	u.settingsSection(c, k, "DATA", func() {
		u.settingsInfoRow(c, k, "Settings file", u.settingsPath())
		if u.remotePath() != "" {
			u.settingsInfoRow(c, k, "Remote project", u.serverAlias()+":"+u.remotePath())
		}
		if terminalsSupported() {
			u.settingsInfoRow(c, k, "Local sessions", session.SocketPath())
		}
		if rc := u.activeRemote(); rc != nil {
			u.settingsInfoRow(c, k, "Remote server", rc.Host+" · "+rc.Socket)
		}
	})
	u.settingsSection(c, k, "SETTINGS FILE", func() {
		u.settingRow(c, k, "Reveal", "Show settings.json in the file manager.", func() {
			if settingsButton(c, "Reveal", false).Clicked() {
				u.saveSettings()
				if u.settingsPath() != "" {
					mygo.Shell.ShowItemInFolder(u.settingsPath())
				}
			}
		})
		u.settingRow(c, k, "Open", "Open settings.json in the default editor.", func() {
			if settingsButton(c, "Open", false).Clicked() {
				u.openSettingsFile()
			}
		})
	})
	u.settingsSection(c, k, "RESET", func() {
		if !u.ResetAsk {
			u.settingRow(c, k, "Reset all settings", "Restore the preferences of a new install; projects and servers stay.", func() {
				if settingsButton(c, "Reset…", false).Clicked() {
					u.ResetAsk = true
					u.invalidate()
				}
			})
			return
		}
		u.settingRow(c, k, "Reset all settings", "This restores preferences, not projects or servers.", func() {
			if settingsButton(c, "Cancel", false).Clicked() {
				u.ResetAsk = false
				u.invalidate()
			}
			if settingsButton(c, "Reset", true).Clicked() {
				u.resetPreferences()
			}
		})
	})
	u.settingsAboutPage(c, k)
}

func (u *UI) openSettingsFile() {
	if u.settingsPath() == "" {
		return
	}
	u.saveSettings()
	if err := mygo.Shell.OpenPath(u.settingsPath()); err != nil {
		u.fail(err)
	}
}

func (u *UI) resetPreferences() {
	*u.Prefs = prefs.Default()
	u.syncAppearance()
	u.resetShortcuts()
	u.ResetAsk = false
	u.Notice = "Preferences reset."
	u.saveSettings()
	u.refreshMenu()
	u.invalidate()
}

// ResetPreferences restores the defaults, for the app's reset command.
func (u *UI) ResetPreferences() { u.resetPreferences() }

func (u *UI) settingsUpdatesPage(c *ui.Context, k *colors) {
	u.settingsSection(c, k, "UPDATES", func() {
		u.settingRow(c, k, "Check for Updates", "Check the latest signed release on GitHub.", func() {
			busy := u.updateChecking() || u.updateInstalling()
			check := ui.Button(c, "Check for Updates").Disabled(busy || u.updateReady())
			if check.Clicked() {
				u.checkForUpdates()
			}
		})
		if u.pendingVersion() != "" && !u.updateReady() {
			u.settingRow(c, k, "Install", "Download and install the release it finds.", func() {
				busy := u.updateChecking() || u.updateInstalling()
				install := ui.PrimaryButton(c, "Install Update").Disabled(busy)
				if install.Clicked() {
					u.installAvailableUpdate()
				}
			})
		}
		u.settingRow(c, k, "Restart", "Restart into the installed update.", func() {
			restart := ui.PrimaryButton(c, "Restart to Update").Disabled(!u.updateReady())
			if restart.Clicked() {
				u.restartForUpdate()
			}
		})
	})
	if msg := u.updateSummary(); msg != "" {
		u.settingsNote(c, k, msg)
	}
	if u.updateError() != "" {
		ui.Text(c, u.updateError()).FontSize(12).TextColor(k.Danger).Padding(0, 4)
	}
}

func (u *UI) settingsAboutPage(c *ui.Context, k *colors) {
	u.settingsSection(c, k, "MOJIAN", func() {
		ui.Row(c).Padding(12, 16).AlignItems(ui.Center).Children(func() {
			wm := "wordmark"
			if c.Theme().Dark {
				wm = "wordmark-reversed"
			}
			ui.Icon(c, icons.Brand(wm)).Size(150, 36)
		})
		u.settingsInfoRow(c, k, "Version", appVersion())
		u.settingsInfoRow(c, k, "System", runtime.GOOS+" · "+runtime.GOARCH)
		project := "No project"
		if u.store() != nil {
			project = u.store().Dir
		}
		u.settingsInfoRow(c, k, "Project", project)
		u.settingsInfoRow(c, k, "Fonts", fmt.Sprintf("%d installed families", len(u.installedFonts())))
	})
}

func appVersion() string {
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return "dev"
}
