package main

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// openTerminal opens the system terminal in the folder of the note in
// view, or in the vault's root.
func (a *app) openTerminal() {
	if a.store == nil {
		return
	}
	if a.store.memory {
		a.fail(errors.New("the terminal needs the desktop app"))
		return
	}
	dir := a.store.dir
	if rel := a.selectedDir(); rel != "" {
		if p, err := a.store.dirPath(rel); err == nil {
			dir = p
		}
	}
	if err := openTerminalAt(dir); err != nil {
		a.fail(err)
	}
}

// openTerminalAt starts a terminal in dir without waiting for it to close.
func openTerminalAt(dir string) error {
	cmd, err := terminalCommand(runtime.GOOS, dir)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// terminalCommand names the command that opens a terminal in dir, for the
// system's own terminal.
func terminalCommand(goos, dir string) (*exec.Cmd, error) {
	switch goos {
	case "darwin":
		if _, err := exec.LookPath("open"); err != nil {
			return nil, err
		}
		return exec.Command("open", "-a", "Terminal", dir), nil
	case "windows":
		if _, err := exec.LookPath("cmd"); err != nil {
			return nil, err
		}
		return exec.Command("cmd", "/c", "start", "", "cmd.exe", "/K", "cd", "/d", dir), nil
	default:
		if term := strings.TrimSpace(os.Getenv("TERMINAL")); term != "" {
			fields := strings.Fields(term)
			if len(fields) > 0 {
				cmd := exec.Command(fields[0], fields[1:]...)
				cmd.Dir = dir
				return cmd, nil
			}
		}
		for _, name := range []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal", "kitty", "alacritty"} {
			path, err := exec.LookPath(name)
			if err != nil {
				continue
			}
			cmd := exec.Command(path)
			cmd.Dir = dir
			return cmd, nil
		}
		return nil, errors.New("no terminal found; set $TERMINAL")
	}
}
