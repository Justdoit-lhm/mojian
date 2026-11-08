package app

import (
	"mojian/internal/app/commands"
	"runtime"

	"mojian/internal/settings"
)

// Settings page indexes, shared by the window, the menu and the palette.
const (
	settingsGeneral       = settings.General
	settingsAppearance    = settings.Appearance
	settingsEditor        = settings.Editor
	settingsTerminal      = settings.Terminal
	settingsShell         = settings.Shell
	settingsNotifications = settings.Notifications
	settingsKeyboard      = settings.Keyboard
	settingsHosts         = settings.Hosts
	settingsGit           = settings.Git
	settingsUpdates       = settings.Updates
)

// commandAvailable hides the commands the platform does not have, as the
// palette does.
func commandAvailable(c *commands.Command) bool {
	if !terminalsSupported() && (c == cmdTerminal || c == cmdSSH) {
		return false
	}
	if runtime.GOOS == "js" && (c == cmdTerminal || c == cmdOpenProject || c == cmdNewFolder) {
		return false
	}
	return true
}
