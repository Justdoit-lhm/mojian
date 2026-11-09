package app

import (
	"runtime"

	"mojian/internal/app/commands"
	"mojian/internal/settings"
)

// Settings page indexes, shared by the window, the menu and the palette.
const (
	settingsGeneral    = settings.General
	settingsAppearance = settings.Appearance
	settingsEditor     = settings.Editor
	settingsTerminal   = settings.Terminal
	settingsKeyboard   = settings.Keyboard
	settingsHosts      = settings.Hosts
	settingsAdvanced   = settings.Advanced
	settingsAbout      = settings.About
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
