//go:build windows

package session

import "errors"

func execCommand(Request) (ExecResult, error) {
	return ExecResult{}, errors.New("session: exec is unavailable on Windows")
}
