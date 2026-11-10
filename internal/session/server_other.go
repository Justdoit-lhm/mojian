//go:build !darwin && !linux && !windows

package session

import "errors"

var errUnsupported = errors.New("session: the server runs on macOS and Linux")

func Serve() error              { return errUnsupported }
func Spawn() error              { return errUnsupported }
func Connect() (*Client, error) { return nil, errUnsupported }
