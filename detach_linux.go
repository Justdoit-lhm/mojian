//go:build linux

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// stdinIsTerminal reports whether standard input is a terminal, so a piped
// or redirected launch stays in the foreground.
func stdinIsTerminal() bool {
	_, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	return err == nil
}
