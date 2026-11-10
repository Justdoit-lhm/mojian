package session

import "io"

// ptyProcess is a program running in a pseudo-terminal. Unix uses
// creack/pty; Windows uses ConPTY through a copied MyGo implementation.
type ptyProcess interface {
	io.ReadWriteCloser
	Resize(cols, rows int) error
	Pid() int
	Wait() (int, error)
	Interrupt() error
	Kill() error
}
