//go:build darwin || linux

package session

import (
	"os"
	"os/exec"
	"path/filepath"
)

func defaultShell() ([]string, string, error) {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	path, err := exec.LookPath(sh)
	if err != nil {
		return nil, "", err
	}
	return []string{path, "-l"}, filepath.Base(sh), nil
}
