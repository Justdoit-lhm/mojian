// Package shellcmd keeps the mojian launcher in the user's shell: the
// alias Settings › Advanced installs, in the startup file of zsh or bash.
package shellcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"mojian/internal/fs"
)

// Shell is a shell whose startup file Settings can edit.
type Shell uint8

const (
	Zsh Shell = iota
	Bash
)

const (
	beginMarker = "# >>> Mojian command line >>>"
	endMarker   = "# <<< Mojian command line <<<"
)

// Name is the shell's id in settings.json.
func (s Shell) Name() string {
	if s == Bash {
		return "bash"
	}
	return "zsh"
}

// Parse reads a shell id, reporting whether it is one Settings knows.
func Parse(id string) (Shell, bool) {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "zsh":
		return Zsh, true
	case "bash":
		return Bash, true
	}
	return Zsh, false
}

// Detect is the shell the user logs in with, or the platform's default
// when $SHELL names neither zsh nor bash.
func Detect() Shell {
	if s, ok := Parse(filepath.Base(os.Getenv("SHELL"))); ok {
		return s
	}
	if runtime.GOOS == "darwin" {
		return Zsh
	}
	return Bash
}

// RcFile is the startup file an interactive session of s reads.
func RcFile(s Shell) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return rcFile(home, s, runtime.GOOS), nil
}

func rcFile(home string, s Shell, goos string) string {
	if s != Bash {
		return filepath.Join(home, ".zshrc")
	}
	if goos != "darwin" {
		return filepath.Join(home, ".bashrc")
	}
	// macOS starts login shells. Use a file bash already reads, so adding
	// ours does not shadow .bash_login or .profile.
	for _, name := range []string{".bash_profile", ".bash_login", ".profile"} {
		p := filepath.Join(home, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(home, ".bash_profile")
}

// State is what a shell's startup file holds for mojian.
type State struct {
	Installed bool   // the marked block is present
	Current   bool   // its alias runs executable
	Path      string // the startup file the block belongs in
}

// Executable is this app's binary, the program the alias runs.
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	return exe, nil
}

// Inspect reports the block in s's startup file. A missing file is an
// empty state, not an error.
func Inspect(executable string, s Shell) (State, error) {
	path, err := RcFile(s)
	if err != nil {
		return State{}, err
	}
	st := State{Path: path}
	data, err := os.ReadFile(resolve(path))
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	start, end, found, err := findBlock(string(data))
	if err != nil {
		return st, fmt.Errorf("%s: %w", path, err)
	}
	if !found {
		return st, nil
	}
	st.Installed = true
	st.Current = strings.Contains(string(data[start:end]), "alias mojian="+quote(executable))
	return st, nil
}

// Install adds the marked block to s's startup file, replacing the one
// already there so the alias follows a moved app.
func Install(executable string, s Shell) (State, error) {
	if executable == "" {
		return State{}, errors.New("shellcmd: empty executable")
	}
	path, err := RcFile(s)
	if err != nil {
		return State{}, err
	}
	target := resolve(path)
	data, err := os.ReadFile(target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return State{Path: path}, err
	}
	text := string(data)
	start, end, found, err := findBlock(text)
	if err != nil {
		return State{Path: path}, fmt.Errorf("%s: %w", path, err)
	}
	block := blockText(executable)
	if found {
		text = text[:start] + block + text[end:]
	} else {
		if text != "" {
			if !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			text += "\n"
		}
		text += block
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm()
	}
	if err := fs.WriteFileAtomic(target, []byte(text)); err != nil {
		return State{Path: path}, err
	}
	if err := os.Chmod(target, mode); err != nil {
		return State{Path: path}, err
	}
	return State{Installed: true, Current: true, Path: path}, nil
}

// Remove deletes the marked block from s's startup file, leaving the
// rest of the file alone.
func Remove(s Shell) error {
	path, err := RcFile(s)
	if err != nil {
		return err
	}
	target := resolve(path)
	data, err := os.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	text := string(data)
	start, end, found, err := findBlock(text)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if !found {
		return nil
	}
	stop := end
	if stop < len(text) && text[stop] == '\n' {
		stop++
	}
	// Drop the blank line Install put before the block.
	if start > 1 && text[start-1] == '\n' && text[start-2] == '\n' {
		start--
	}
	return fs.WriteFileAtomic(target, []byte(text[:start]+text[stop:]))
}

func blockText(executable string) string {
	return beginMarker + "\n" +
		"# Added by Mojian (Settings > Advanced).\n" +
		"alias mojian=" + quote(executable) + "\n" +
		endMarker + "\n"
}

// findBlock locates the marked block, refusing a begin marker whose end
// marker was edited away rather than guessing where it ends.
func findBlock(text string) (start, end int, found bool, err error) {
	start = strings.Index(text, beginMarker)
	if start < 0 {
		return 0, 0, false, nil
	}
	rel := strings.Index(text[start:], endMarker)
	if rel < 0 {
		return 0, 0, false, errors.New("the Mojian block has no end marker")
	}
	return start, start + rel + len(endMarker), true, nil
}

// resolve follows a symlinked startup file to the file it names, so a
// dotfile that lives elsewhere stays a symlink.
func resolve(path string) string {
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return path
}

// quote makes s one shell word, including a path that holds a quote.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
