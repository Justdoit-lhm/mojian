// Package session keeps terminal sessions in a server process of the app
// itself, so that quitting Mojian detaches from them instead of ending
// them.
package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"time"

	"mojian/internal/filekind"
)

// readLine reads one newline-terminated frame, refusing one longer than
// max bytes, so a peer cannot make the server or the app allocate without
// bound.
func readLine(r *bufio.Reader, max int) ([]byte, error) {
	var out []byte
	for {
		frag, err := r.ReadSlice('\n')
		if len(out)+len(frag) > max {
			return nil, errors.New("session: frame too long")
		}
		out = append(out, frag...)
		if err == nil {
			return out, nil
		}
		if err != bufio.ErrBufferFull {
			return out, err
		}
	}
}

// ProtocolVersion changes when the wire protocol does.
const ProtocolVersion = 3

// ProtocolError says a running server speaks another protocol version.
type ProtocolError struct{ Server, Client int }

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("session: server protocol %d, app needs %d", e.Server, e.Client)
}

// Request is one line of a control connection.
type Request struct {
	ID     int64          `json:"id"`
	Op     string         `json:"op"`
	SID    string         `json:"sid,omitempty"`
	Cols   int            `json:"cols,omitempty"`
	Rows   int            `json:"rows,omitempty"`
	Create *CreateOptions `json:"create,omitempty"`
	Path   string         `json:"path,omitempty"`
	To     string         `json:"to,omitempty"`
	// Fresh asks listTree to bypass the server's listing cache.
	Fresh bool     `json:"fresh,omitempty"`
	Data  []byte   `json:"data,omitempty"`
	Dir   string   `json:"dir,omitempty"`
	Args  []string `json:"args,omitempty"`
	Env   []string `json:"env,omitempty"`
	// Timeout is how long a command may run, in milliseconds.
	Timeout int `json:"timeout,omitempty"`
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

// FileEntry describes a path in a remote project.
type FileEntry struct {
	Rel  string        `json:"rel"`
	Dir  bool          `json:"dir,omitempty"`
	Size int64         `json:"size,omitempty"`
	Mod  time.Time     `json:"mod,omitempty"`
	Kind filekind.Kind `json:"kind,omitempty"`
	// Hash is the content hash of an asset, so a viewer cache is keyed by
	// what the file contains rather than by its size and timestamp.
	Hash string `json:"hash,omitempty"`
}

// ExecResult is what a command run on the server left.
type ExecResult struct {
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
	Code   int    `json:"code,omitempty"`
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

// RuntimeDir is where the server keeps its socket and lock. A remote
// server sets MOJIAN_RUNTIME_DIR to a path sshd may reach with stream
// local forwarding, such as one under XDG_RUNTIME_DIR; locally it is the
// data directory.
func RuntimeDir() string {
	if d := os.Getenv("MOJIAN_RUNTIME_DIR"); d != "" {
		return d
	}
	return Dir()
}

// SocketPath returns the server's socket, short enough for a Unix path.
func SocketPath() string { return SocketPathIn(RuntimeDir()) }

// SocketPathIn returns the socket a server started with dir would bind,
// so a forwarded socket and the server agree even when the path is long.
func SocketPathIn(dir string, uid ...int) string {
	p := filepath.Join(dir, "sessions.sock")
	if len(p) < 100 {
		return p
	}
	id := os.Getuid()
	if len(uid) > 0 {
		id = uid[0]
	}
	h := fnv.New32a()
	h.Write([]byte(p))
	name := fmt.Sprintf("mojian-%x.sock", h.Sum32())
	if id >= 0 {
		name = fmt.Sprintf("mojian-%d-%x.sock", id, h.Sum32())
	}
	return filepath.Join(os.TempDir(), name)
}
