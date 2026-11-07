//go:build darwin || linux

package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func lockFile(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) }

func detached() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }

// Spawn starts the session server as a detached process of this binary.
func Spawn() error {
	exe, err := spawnExecutable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(Dir(), "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	defer devnull.Close()
	cmd := exec.Command(exe, "-server")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, logf, logf
	cmd.SysProcAttr = detached()
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func processAlive(pid int) bool { return syscall.Kill(pid, 0) == nil }
