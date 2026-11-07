//go:build darwin || linux

package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Client is a control connection to the session server.
type Client struct {
	conn   net.Conn
	socket string // the Unix socket this client talks to

	wmu     sync.Mutex
	mu      sync.Mutex
	nextID  int64
	pending map[int64]chan Response
	err     error
	closed  chan struct{}
}

// Connect connects to the server, starting one in the background when
// none runs.
func Connect() (*Client, error) {
	return connectAt(SocketPath(), true)
}

// ConnectAt connects to a server at an explicit Unix socket, without
// starting one.
func ConnectAt(path string) (*Client, error) {
	return connectAt(path, false)
}

func connectAt(path string, spawn bool) (*Client, error) {
	c, err := dialAt(path, spawn)
	if err != nil {
		return nil, err
	}
	h, helloErr := c.Hello()
	if helloErr == nil && h.Version == ProtocolVersion {
		return c, nil
	}
	// A detached server from an older build (or one whose hello failed)
	// cannot be trusted with this build's protocol. A local server can be
	// replaced; a remote one is left for its caller to restart.
	if spawn {
		// The old server may need a moment to release its lock and socket;
		// ask it to stop and try a fresh one a few times.
		for attempt := 0; attempt < 3; attempt++ {
			_ = c.Shutdown()
			_ = c.Close()
			time.Sleep(100 * time.Millisecond)
			next, err := dialAt(path, true)
			if err != nil {
				if attempt == 2 {
					return nil, err
				}
				continue
			}
			h, helloErr := next.Hello()
			if helloErr == nil && h.Version == ProtocolVersion {
				return next, nil
			}
			c = next
			if helloErr == nil {
				helloErr = &ProtocolError{Server: h.Version, Client: ProtocolVersion}
			}
			if attempt == 2 {
				_ = next.Close()
				return nil, helloErr
			}
		}
	}
	_ = c.Close()
	if helloErr != nil {
		return nil, helloErr
	}
	return nil, &ProtocolError{Server: h.Version, Client: ProtocolVersion}
}

// dialAt connects, starting a local server first when spawn is set.
func dialAt(path string, spawn bool) (*Client, error) {
	conn, err := net.Dial("unix", path)
	if err != nil && spawn {
		deadline := time.Now().Add(8 * time.Second)
		for {
			// An older server may still hold the lock while it exits: ask
			// for another and keep trying, instead of failing on its
			// "already runs" once.
			_ = Spawn()
			for i := 0; i < 10; i++ {
				time.Sleep(30 * time.Millisecond)
				conn, err = net.Dial("unix", path)
				if err == nil {
					goto connected
				}
			}
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("session: connecting to the server: %w", err)
			}
		}
	}
connected:
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn: conn, socket: path,
		pending: map[int64]chan Response{}, closed: make(chan struct{}),
	}
	go c.read()
	return c, nil
}

// Closed closes when the control connection is lost.
func (c *Client) Closed() <-chan struct{} { return c.closed }

func (c *Client) read() {
	r := bufio.NewReaderSize(c.conn, 1<<20)
	for {
		line, err := readLine(r, maxLine)
		if err != nil {
			c.mu.Lock()
			c.err = err
			for id, ch := range c.pending {
				ch <- Response{ID: id, Error: "session server connection lost"}
				delete(c.pending, id)
			}
			c.mu.Unlock()
			close(c.closed)
			return
		}
		var res Response
		if json.Unmarshal(line, &res) != nil {
			continue
		}
		c.mu.Lock()
		ch := c.pending[res.ID]
		delete(c.pending, res.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- res
		}
	}
}

func (c *Client) call(req Request, out any) error {
	ch := make(chan Response, 1)
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return errors.New("session: connection to the server lost")
	}
	c.nextID++
	req.ID = c.nextID
	c.pending[req.ID] = ch
	c.mu.Unlock()
	b, _ := json.Marshal(req)
	c.wmu.Lock()
	_, err := c.conn.Write(append(b, '\n'))
	c.wmu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, req.ID)
		c.mu.Unlock()
		return err
	}
	select {
	case res := <-ch:
		if res.Error != "" {
			return errors.New(res.Error)
		}
		if out != nil && len(res.Data) > 0 {
			return json.Unmarshal(res.Data, out)
		}
		return nil
	case <-time.After(60 * time.Second):
		c.mu.Lock()
		delete(c.pending, req.ID)
		c.mu.Unlock()
		return fmt.Errorf("session: %s timed out", req.Op)
	}
}

func (c *Client) Hello() (h Hello, err error) { err = c.call(Request{Op: "hello"}, &h); return }
func (c *Client) List() (out []SessionInfo, err error) {
	err = c.call(Request{Op: "list"}, &out)
	return
}
func (c *Client) Create(o CreateOptions) (out SessionInfo, err error) {
	err = c.call(Request{Op: "create", Create: &o}, &out)
	return
}
func (c *Client) Kill(sid string) error { return c.call(Request{Op: "kill", SID: sid}, nil) }
func (c *Client) Resize(sid string, cols, rows int) error {
	return c.call(Request{Op: "resize", SID: sid, Cols: cols, Rows: rows}, nil)
}
func (c *Client) Shutdown() error { return c.call(Request{Op: "shutdown"}, nil) }
func (c *Client) Close() error    { return c.conn.Close() }

// Stream is a terminal's connection attached to a session.
type Stream struct {
	c   *Client
	sid string

	mu       sync.Mutex
	cond     *sync.Cond
	conn     net.Conn
	err      error
	closed   bool
	cols     int
	rows     int
	sentCols int
	sentRows int
}

// Stream attaches to the session sid.
func (c *Client) Stream(sid string, cols, rows int) *Stream {
	s := &Stream{c: c, sid: sid, cols: cols, rows: rows}
	s.cond = sync.NewCond(&s.mu)
	go s.attach()
	return s
}

func (s *Stream) attach() {
	s.mu.Lock()
	cols, rows := s.cols, s.rows
	s.mu.Unlock()
	socket := s.c.socket
	if socket == "" {
		socket = SocketPath()
	}
	conn, err := net.Dial("unix", socket)
	if err == nil {
		b, _ := json.Marshal(Attach{Op: "attach", SID: s.sid, Cols: cols, Rows: rows})
		_, err = conn.Write(append(b, '\n'))
		if err == nil {
			err = readOK(conn)
		}
		if err != nil {
			_ = conn.Close()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		if conn != nil {
			_ = conn.Close()
		}
		return
	}
	if err != nil {
		s.err = err
	} else {
		s.conn, s.sentCols, s.sentRows = conn, cols, rows
	}
	s.cond.Broadcast()
}

func readOK(conn net.Conn) error {
	var line []byte
	b := make([]byte, 1)
	for {
		if _, err := conn.Read(b); err != nil {
			return err
		}
		if b[0] == '\n' {
			break
		}
		line = append(line, b[0])
		if len(line) > 4096 {
			return errors.New("session: bad answer")
		}
	}
	var res struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(line, &res); err != nil {
		return err
	}
	if !res.OK {
		return errors.New(res.Error)
	}
	return nil
}

func (s *Stream) wait() (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for s.conn == nil && s.err == nil && !s.closed {
		s.cond.Wait()
	}
	if s.closed {
		return nil, io.EOF
	}
	return s.conn, s.err
}

func (s *Stream) Read(p []byte) (int, error) {
	conn, err := s.wait()
	if err != nil {
		return 0, err
	}
	return conn.Read(p)
}

func (s *Stream) Write(p []byte) (int, error) {
	conn, err := s.wait()
	if err != nil {
		return 0, err
	}
	return conn.Write(p)
}

// Resize tells the session the terminal's size.
func (s *Stream) Resize(cols, rows int) error {
	s.mu.Lock()
	s.cols, s.rows = cols, rows
	conn, sentCols, sentRows := s.conn, s.sentCols, s.sentRows
	s.mu.Unlock()
	if conn == nil {
		return nil
	}
	if cols == sentCols && rows == sentRows {
		return nil
	}
	return s.c.Resize(s.sid, cols, rows)
}

// Close detaches; the session keeps running.
func (s *Stream) Close() error {
	s.mu.Lock()
	s.closed = true
	conn := s.conn
	s.cond.Broadcast()
	s.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *Client) ListTree(root string) (out []FileEntry, err error) {
	err = c.call(Request{Op: "listTree", Path: root}, &out)
	return
}
func (c *Client) ReadFile(path string) (data []byte, err error) {
	err = c.call(Request{Op: "readFile", Path: path}, &data)
	return
}
func (c *Client) WriteFile(path string, data []byte) error {
	return c.call(Request{Op: "writeFile", Path: path, Data: data}, nil)
}
func (c *Client) Mkdir(path string) error { return c.call(Request{Op: "mkdir", Path: path}, nil) }
func (c *Client) Rename(from, to string) error {
	return c.call(Request{Op: "rename", Path: from, To: to}, nil)
}
func (c *Client) Remove(path string) error { return c.call(Request{Op: "remove", Path: path}, nil) }
func (c *Client) Stat(path string) (entry FileEntry, err error) {
	err = c.call(Request{Op: "stat", Path: path}, &entry)
	return
}
func (c *Client) Copy(from, to string) error {
	return c.call(Request{Op: "copy", Path: from, To: to}, nil)
}
func (c *Client) Exec(dir string, args, env []string, timeout time.Duration) (res ExecResult, err error) {
	err = c.call(Request{
		Op: "exec", Dir: dir, Args: args, Env: env,
		Timeout: int(timeout / time.Millisecond),
	}, &res)
	return
}
func (c *Client) ListDirs(path string) (out []FileEntry, err error) {
	err = c.call(Request{Op: "listDirs", Path: path}, &out)
	return
}

// Executable returns the path of the app binary and its modification time.
func Executable() (string, time.Time) {
	exe, err := os.Executable()
	if err != nil {
		return "", time.Time{}
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	st, err := os.Stat(exe)
	if err != nil {
		return exe, time.Time{}
	}
	return exe, st.ModTime()
}

func spawnExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	return exe, nil
}
