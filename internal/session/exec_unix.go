//go:build darwin || linux

package session

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"

	"mojian/internal/gitenv"
)

const (
	execOutLimit = 2 << 20
	execErrLimit = 64 << 10
)

// execCommand runs a command for the app (Git, mostly) with a deadline
// and a cap on what it may print.
func execCommand(req Request) (ExecResult, error) {
	if len(req.Args) == 0 {
		return ExecResult{}, errors.New("exec needs a command")
	}
	timeout := time.Duration(req.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, req.Args[0], req.Args[1:]...)
	cmd.Dir = req.Dir
	cmd.Env = execEnv(req.Env)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 2 * time.Second
	var out, errb limitedWriter
	out.limit, errb.limit = execOutLimit, execErrLimit
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	res := ExecResult{Stdout: out.String(), Stderr: errb.String()}
	if ee, ok := err.(*exec.ExitError); ok {
		res.Code = ee.ExitCode()
	}
	if ctx.Err() != nil {
		return res, errors.New("command timed out")
	}
	if err != nil && res.Code == 0 {
		return res, err
	}
	return res, nil
}

// execEnv starts from the server's environment, without the Git
// variables a launching shell may have set, then adds the command's.
func execEnv(extra []string) []string { return gitenv.Env(extra...) }

// limitedWriter keeps the first limit bytes and drops the rest.
type limitedWriter struct {
	bytes.Buffer
	limit int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if left := w.limit - w.Len(); left > 0 {
		if len(p) > left {
			w.Buffer.Write(p[:left])
		} else {
			w.Buffer.Write(p)
		}
	}
	return len(p), nil
}
