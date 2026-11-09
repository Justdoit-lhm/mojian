package settingsui

import (
	"os"
	"path/filepath"
	"strings"

	"mojian/internal/shellcmd"

	"github.com/egoist/mygo/ui"
)

// shellCommandState caches what the chosen shell's startup file holds, so
// the page does not read the file on every frame.
type shellCommandState struct {
	exe     string
	exeErr  error
	exeRead bool
	key     string
	state   shellcmd.State
	err     error
}

// shellCommandRow is Advanced's zsh/bash choice and Install button: the
// button adds the alias to the one startup file the chosen shell reads,
// or takes the marked block out again.
func (u *UI) shellCommandRow(c *ui.Context, k *colors) {
	shell := u.shellChoice()
	exe, exeErr := u.shellExecutable()
	var st shellcmd.State
	var stErr error
	if exeErr == nil {
		st, stErr = u.shellStatus(exe, shell)
	}
	u.settingRow(c, k, "Shell command", shellDetail(st, exeErr, stErr), func() {
		u.settingsChoiceControl(c, k, "shell-command", "Shell", int(shell),
			[]string{"Zsh", "Bash"}, func(i int) {
				u.ShellCommand = shellcmd.Shell(i).Name()
				u.saveSettings()
				u.invalidate()
			})
		installed := st.Installed && st.Current
		label, primary := "Install", true
		switch {
		case installed:
			label, primary = "Remove", false
		case st.Installed:
			label = "Update"
		}
		button := settingsButton(c, label, primary).Disabled(exeErr != nil || stErr != nil)
		if button.Clicked() {
			u.changeShellCommand(exe, shell, st.Path, installed)
		}
	})
}

// changeShellCommand installs, updates or removes the block and says what
// happened in the window's notice.
func (u *UI) changeShellCommand(exe string, shell shellcmd.Shell, path string, remove bool) {
	if remove {
		if err := shellcmd.Remove(shell); err != nil {
			u.Notice = err.Error()
		} else {
			u.Notice = "Removed the mojian command from " + shortPath(path) + "."
		}
	} else if _, err := shellcmd.Install(exe, shell); err != nil {
		u.Notice = err.Error()
	} else {
		u.Notice = "Added the mojian command to " + shortPath(path) + ". Open a new terminal to use it."
	}
	u.shellCmd.key = ""
	u.invalidate()
}

func (u *UI) shellChoice() shellcmd.Shell {
	if s, ok := shellcmd.Parse(u.ShellCommand); ok {
		return s
	}
	return shellcmd.Detect()
}

func (u *UI) shellExecutable() (string, error) {
	if !u.shellCmd.exeRead {
		u.shellCmd.exeRead = true
		u.shellCmd.exe, u.shellCmd.exeErr = u.ShellCommandExecutable()
	}
	return u.shellCmd.exe, u.shellCmd.exeErr
}

func (u *UI) shellStatus(exe string, shell shellcmd.Shell) (shellcmd.State, error) {
	key := exe + "\x00" + shell.Name()
	if u.shellCmd.key != key {
		u.shellCmd.key = key
		u.shellCmd.state, u.shellCmd.err = shellcmd.Inspect(exe, shell)
	}
	return u.shellCmd.state, u.shellCmd.err
}

func shellDetail(st shellcmd.State, exeErr, stErr error) string {
	switch {
	case exeErr != nil:
		return exeErr.Error()
	case stErr != nil:
		return stErr.Error()
	case st.Installed && st.Current:
		return "Installed in " + shortPath(st.Path) + "."
	case st.Installed:
		return "The block in " + shortPath(st.Path) + " runs another Mojian; Update points it here."
	default:
		return "Adds an alias to " + shortPath(st.Path) + "; open a new terminal after installing."
	}
}

// shortPath writes a startup file as ~/.zshrc when it is under the home
// directory.
func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || path == "" {
		return path
	}
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return "~" + string(filepath.Separator) + rel
}
