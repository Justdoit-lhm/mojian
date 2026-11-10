//go:build darwin || linux

package session

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

type unixPTY struct {
	cmd  *exec.Cmd
	file *os.File
}

func startPTY(argv []string, dir string, env []string, cols, rows int) (ptyProcess, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	return &unixPTY{cmd: cmd, file: file}, nil
}

func (p *unixPTY) Read(b []byte) (int, error)  { return p.file.Read(b) }
func (p *unixPTY) Write(b []byte) (int, error) { return p.file.Write(b) }
func (p *unixPTY) Close() error                { return p.file.Close() }

func (p *unixPTY) Resize(cols, rows int) error {
	return pty.Setsize(p.file, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (p *unixPTY) Pid() int { return p.cmd.Process.Pid }

func (p *unixPTY) Wait() (int, error) {
	err := p.cmd.Wait()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return -1, err
}

func (p *unixPTY) Interrupt() error {
	if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT); err == nil {
		return nil
	}
	return p.cmd.Process.Signal(os.Interrupt)
}

func (p *unixPTY) Kill() error {
	if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}
