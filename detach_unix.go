//go:build darwin || linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"mojian/internal/session"

	"github.com/egoist/mygo"
)

// detachFromTerminal re-runs this binary in a new session when it was
// started from a terminal, and reports whether this process should exit.
// The shell gets its prompt back at once, closing the terminal or pressing
// Ctrl+C no longer reaches the window, and the window outlives the shell.
//
// The copy keeps stdout and stderr in app.log under the data directory: a
// window that cannot start has no terminal left to report the error.
//
// mygo dev launches this binary with MYGO_DEV=1 for its live reload, and
// MyGo consumes that variable in its own init; mygo.IsDev is the exported
// check and is also true for a plain go build, which should keep its log
// in the terminal while it is being developed.
func detachFromTerminal() bool {
	if !shouldDetach(stdinIsTerminal(), mygo.IsDev()) {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mojian: cannot detach from the terminal: %v\n", err)
		return false
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mojian: cannot detach from the terminal: %v\n", err)
		return false
	}
	defer devnull.Close()
	out := devnull
	if err := os.MkdirAll(session.Dir(), 0o700); err == nil {
		if f, err := os.OpenFile(filepath.Join(session.Dir(), "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			defer f.Close()
			out = f
		}
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), detachedEnv+"=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, out, out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "mojian: cannot detach from the terminal: %v\n", err)
		return false
	}
	_ = cmd.Process.Release()
	return true
}
