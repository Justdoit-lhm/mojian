// Package session keeps terminal sessions in a server process of the app
// itself, so that quitting Mojian detaches from them instead of ending
// them.
package session

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"time"
)

// ProtocolVersion changes when the wire protocol does.
const ProtocolVersion = 1

// Request is one line of a control connection.
type Request struct {
	ID     int64          `json:"id"`
	Op     string         `json:"op"`
	SID    string         `json:"sid,omitempty"`
	Cols   int            `json:"cols,omitempty"`
	Rows   int            `json:"rows,omitempty"`
	Create *CreateOptions `json:"create,omitempty"`
}

// Response answers a Request with the same ID.
type Response struct {
	ID    int64           `json:"id"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// CreateOptions starts a session.
type CreateOptions struct {
	Command []string `json:"command,omitempty"`
	Dir     string   `json:"dir,omitempty"`
	Env     []string `json:"env,omitempty"`
	Cols    int      `json:"cols,omitempty"`
	Rows    int      `json:"rows,omitempty"`
}

// SessionInfo describes a session.
type SessionInfo struct {
	ID       string    `json:"id"`
	PID      int       `json:"pid"`
	Shell    string    `json:"shell"`
	Created  time.Time `json:"created"`
	Cols     int       `json:"cols"`
	Rows     int       `json:"rows"`
	Title    string    `json:"title,omitempty"`
	Exited   bool      `json:"exited"`
	ExitCode int       `json:"exitCode"`
}

// Hello is the answer to "hello".
type Hello struct {
	Version int       `json:"version"`
	PID     int       `json:"pid"`
	Started time.Time `json:"started"`
}

// Attach is the first line of a connection attached to a session.
type Attach struct {
	Op   string `json:"op"` // "attach"
	SID  string `json:"sid"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// Dir returns the directory of the server's socket and log.
func Dir() string {
	if d := os.Getenv("MOJIAN_DIR"); d != "" {
		return d
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "Mojian")
}

// SocketPath returns the server's socket, short enough for a Unix path.
func SocketPath() string {
	p := filepath.Join(Dir(), "sessions.sock")
	if len(p) < 100 {
		return p
	}
	h := fnv.New32a()
	h.Write([]byte(p))
	name := fmt.Sprintf("mojian-%x.sock", h.Sum32())
	if uid := os.Getuid(); uid >= 0 {
		name = fmt.Sprintf("mojian-%d-%x.sock", uid, h.Sum32())
	}
	return filepath.Join(os.TempDir(), name)
}
