//go:build darwin || linux

package session

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func lockFile(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) }

func detached() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }
