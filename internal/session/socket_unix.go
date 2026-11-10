//go:build darwin || linux

package session

import (
	"fmt"
	"net"
	"os"
	"time"
)

// SocketPath returns the server's socket, short enough for a Unix path.
func SocketPath() string { return SocketPathIn(RuntimeDir()) }

func listen(sock string) (net.Listener, error) {
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(sock, 0o600); err != nil {
		_ = ln.Close()
		_ = os.Remove(sock)
		return nil, fmt.Errorf("session: securing the socket: %w", err)
	}
	return ln, nil
}

func dial(sock string) (net.Conn, error) { return net.Dial("unix", sock) }

func dialTimeout(sock string, d time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", sock, d)
}

func removeSocket(sock string) { _ = os.Remove(sock) }
