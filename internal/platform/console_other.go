//go:build !windows

package platform

// HideConsole does nothing outside Windows.
func HideConsole(any) {}
