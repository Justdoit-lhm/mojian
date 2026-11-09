//go:build !darwin && !linux

package main

// detachFromTerminal does nothing where Mojian has no native window and no
// session to leave; the browser build keeps its launcher.
func detachFromTerminal() bool { return false }
