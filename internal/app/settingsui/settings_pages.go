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
	u.settingsSection(c, k, "SETTINGS FILE", func() {
		path, detail := u.settingsPath(), u.settingsPath()
		if detail == "" {
			detail = "Not saved yet"
		}
		u.settingRowMono(c, k, "Settings file", detail, func() {
			reveal := settingsButton(c, "Reveal", false).Disabled(path == "")
			if reveal.Clicked() {
				u.saveSettings()
				mygo.Shell.ShowItemInFolder(path)
			}
			if settingsButton(c, "Open", false).Disabled(path == "").Clicked() {
				u.openSettingsFile()
			}
		})
	})
	if u.hasAdvancedData() {
		u.settingsSection(c, k, "DATA", func() {
			if terminalsSupported() {
				u.settingsInfoRow(c, k, "Local sessions", session.SocketPath())
			}
			if u.remotePath() != "" {
				u.settingsInfoRow(c, k, "Remote project", u.serverAlias()+":"+u.remotePath())
			}
			if rc := u.activeRemote(); rc != nil {
				u.settingsInfoRow(c, k, "Remote server", rc.Host+" · "+rc.Socket)
			}
		})
	}
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
}

// hasAdvancedData reports whether any path on the DATA card is worth
// showing: a session socket, a remote project, or a live remote server.
func (u *UI) hasAdvancedData() bool {
	return u.remotePath() != "" || u.activeRemote() != nil || terminalsSupported()
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
	u.setAppearance(u.Appearance)
	u.setTheme(u.Theme)
	u.resetShortcuts()
	u.ResetAsk = false
	u.Notice = "Preferences reset."
	u.saveSettings()
	u.refreshMenu()
	u.invalidate()
}

// ResetPreferences restores the defaults, for the app's reset command.
func (u *UI) ResetPreferences() { u.resetPreferences() }

// settingsAboutPage is the app's identity, its updates and the machine it
// runs on.
func (u *UI) settingsAboutPage(c *ui.Context, k *colors) {
	u.settingsSection(c, k, "", func() {
		ui.Column(c).Padding(18).Gap(6).Children(func() {
			wordmark := "wordmark"
			if c.Theme().Dark {
				wordmark = "wordmark-reversed"
			}
			ui.Icon(c, icons.Brand(wordmark)).Size(150, 36).AlignSelf(ui.Start)
			ui.Text(c, "A quiet text editor for plain files.").FontSize(12).
				TextColor(k.TextMuted)
			ui.Text(c, "Version "+appVersion()).FontSize(11.5).TextColor(k.TextFaint)
		})
	})
	u.settingsSection(c, k, "SOFTWARE UPDATE", func() {
		u.settingRow(c, k, "Software Update", "Check GitHub for a newer signed release.", func() {
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
	if msg := u.updateError(); msg != "" {
		ui.Text(c, msg).FontSize(12).TextColor(k.Danger).Padding(0, 4)
	}
	u.settingsSection(c, k, "SYSTEM", func() {
		u.settingsInfoRow(c, k, "System", runtime.GOOS+" · "+runtime.GOARCH)
		project := "No project"
		if u.projectPath() != "" {
			project = u.projectPath()
		}
		u.settingsInfoRow(c, k, "Project", project)
		u.settingsInfoRow(c, k, "Fonts",
			fmt.Sprintf("%d installed families", len(u.installedFonts())))
	})
}

func appVersion() string {
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return "dev"
}
