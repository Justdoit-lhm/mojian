//go:build windows

package platform

import (
	"os/exec"
	"syscall"
)

// HideConsole keeps a console helper such as git or pdftoppm from
// flashing a window in a GUI build.
func HideConsole(v any) {
	cmd, ok := v.(*exec.Cmd)
	if !ok {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
