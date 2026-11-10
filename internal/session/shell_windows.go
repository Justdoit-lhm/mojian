//go:build windows

package session

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

func defaultShell() ([]string, string, error) {
	for _, name := range []string{"pwsh.exe", "powershell.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return []string{path, "-NoLogo"}, filepath.Base(path), nil
		}
	}
	if path, err := exec.LookPath("cmd.exe"); err == nil {
		return []string{path}, filepath.Base(path), nil
	}
	if path := os.Getenv("COMSPEC"); path != "" {
		return []string{path}, filepath.Base(path), nil
	}
	return nil, "", errors.New("session: no Windows shell found")
}
