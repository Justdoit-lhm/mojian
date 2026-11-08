package app

import (
	"fmt"
	"runtime"

	"mojian/internal/icons"
	"mojian/internal/session"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) settingsAdvancedPage(c *ui.Context, k *colors) {
	a.settingsNoticeLine(c, k)
	a.settingsSection(c, k, "DATA", func() {
		a.settingsInfoRow(c, k, "Settings file", a.settingsPath)
		if a.remotePath != "" {
			a.settingsInfoRow(c, k, "Remote project", a.serverAlias+":"+a.remotePath)
		}
		if terminalsSupported() {
			a.settingsInfoRow(c, k, "Local sessions", session.SocketPath())
		}
		if rc := a.activeRemote(); rc != nil {
			a.settingsInfoRow(c, k, "Remote server", rc.host+" · "+rc.socket)
		}
	})
	a.settingsSection(c, k, "SETTINGS FILE", func() {
		a.settingRow(c, k, "Reveal", "Show settings.json in the file manager.", func() {
			if settingsButton(c, "Reveal", false).Clicked() {
				a.saveSettings()
				if a.settingsPath != "" {
					mygo.Shell.ShowItemInFolder(a.settingsPath)
				}
			}
		})
		a.settingRow(c, k, "Open", "Open settings.json in the default editor.", func() {
			if settingsButton(c, "Open", false).Clicked() {
				a.openSettingsFile()
			}
		})
	})
	a.settingsSection(c, k, "RESET", func() {
		if !a.settingsResetAsk {
			a.settingRow(c, k, "Reset all settings", "Restore the preferences of a new install; projects and servers stay.", func() {
				if settingsButton(c, "Reset…", false).Clicked() {
					a.settingsResetAsk = true
					a.invalidate()
				}
			})
			return
		}
		a.settingRow(c, k, "Reset all settings", "This restores preferences, not projects or servers.", func() {
			if settingsButton(c, "Cancel", false).Clicked() {
				a.settingsResetAsk = false
				a.invalidate()
			}
			if settingsButton(c, "Reset", true).Clicked() {
				a.resetPreferences()
			}
		})
	})
	a.settingsAboutPage(c, k)
}

func (a *app) openSettingsFile() {
	if a.settingsPath == "" {
		return
	}
	a.saveSettings()
	if err := mygo.Shell.OpenPath(a.settingsPath); err != nil {
		a.fail(err)
	}
}

func (a *app) resetPreferences() {
	a.prefs = defaultPrefs()
	a.syncAppearance()
	a.shortcutKeys = nil
	a.settingsResetAsk = false
	a.settingsNotice = "Preferences reset."
	a.saveSettings()
	a.refreshMenu()
	a.invalidate()
}

func (a *app) settingsUpdatesPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "UPDATES", func() {
		a.settingRow(c, k, "Check for Updates", "Check the latest signed release on GitHub.", func() {
			busy := a.updateChecking || a.updateInstalling
			check := ui.Button(c, "Check for Updates").Disabled(busy || a.updateReady)
			if check.Clicked() {
				a.checkForUpdates()
			}
		})
		if a.pendingUpdate != nil && !a.updateReady {
			a.settingRow(c, k, "Install", "Download and install the release it finds.", func() {
				busy := a.updateChecking || a.updateInstalling
				install := ui.PrimaryButton(c, "Install Update").Disabled(busy)
				if install.Clicked() {
					a.installAvailableUpdate()
				}
			})
		}
		a.settingRow(c, k, "Restart", "Restart into the installed update.", func() {
			restart := ui.PrimaryButton(c, "Restart to Update").Disabled(!a.updateReady)
			if restart.Clicked() {
				a.restartForUpdate()
			}
		})
	})
	if msg := a.updateSummary(); msg != "" {
		a.settingsNote(c, k, msg)
	}
	if a.updateErr != "" {
		ui.Text(c, a.updateErr).FontSize(12).TextColor(k.Danger).Padding(0, 4)
	}
}

func (a *app) settingsAboutPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "MOJIAN", func() {
		ui.Row(c).Padding(12, 16).AlignItems(ui.Center).Children(func() {
			wm := "wordmark"
			if c.Theme().Dark {
				wm = "wordmark-reversed"
			}
			ui.Icon(c, icons.Brand(wm)).Size(150, 36)
		})
		a.settingsInfoRow(c, k, "Version", appVersion())
		a.settingsInfoRow(c, k, "System", runtime.GOOS+" · "+runtime.GOARCH)
		project := "No project"
		if a.store != nil {
			project = a.store.dir
		}
		a.settingsInfoRow(c, k, "Project", project)
		a.settingsInfoRow(c, k, "Fonts", fmt.Sprintf("%d installed families", len(a.installedFonts())))
	})
}

// commandAvailable hides the commands the platform does not have, as the
// palette does.
func commandAvailable(c *command) bool {
	if !terminalsSupported() && (c == cmdTerminal || c == cmdSSH) {
		return false
	}
	if runtime.GOOS == "js" && (c == cmdTerminal || c == cmdOpenProject || c == cmdNewFolder) {
		return false
	}
	return true
}

func appVersion() string {
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return "dev"
}
