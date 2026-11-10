//go:build windows

package session

import (
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// SocketPath is a per-user named pipe; a pipe has no filesystem path.
func SocketPath() string {
	sid := ""
	token := windows.GetCurrentProcessToken()
	if user, err := token.GetTokenUser(); err == nil {
		sid = user.User.Sid.String()
	}
	if sid == "" {
		sid = os.Getenv("USERNAME")
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(Dir()))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(sid))
	return fmt.Sprintf(`\\.\pipe\mojian-%x`, h.Sum64())
}

func listen(sock string) (net.Listener, error) {
	return winio.ListenPipe(sock, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;OW)",
	})
}

func dial(sock string) (net.Conn, error) { return winio.DialPipe(sock, nil) }

func dialTimeout(sock string, d time.Duration) (net.Conn, error) {
	return winio.DialPipe(sock, &d)
}

func removeSocket(string) {}
