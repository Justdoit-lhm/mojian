package app

import "mojian/internal/platform"

// The app spells these helpers without the package name; the
// implementation is internal/platform's, and this file is the only place
// app code reaches it through.

func terminalsSupported() bool { return platform.TerminalsSupported() }

func nativeKeys(s string) string { return platform.NativeKeys(s) }

func revealLabel() string { return platform.RevealLabel() }

func thisComputerName() string { return platform.ThisComputerName() }

// settingsCatAvailable hides the pages of features the platform lacks.
func settingsCatAvailable(cat int) bool {
	if terminalsSupported() {
		return true
	}
	switch cat {
	case settingsTerminal, settingsShell, settingsNotifications, settingsHosts:
		return false
	}
	return true
}
