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
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Client is a control connection to the session server.
type Client struct {
	conn net.Conn

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
	conn, err := net.Dial("unix", path)
	if err != nil && spawn {
		if spawnErr := Spawn(); spawnErr != nil {
			return nil, fmt.Errorf("session: starting the server: %w", spawnErr)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			time.Sleep(30 * time.Millisecond)
			conn, err = net.Dial("unix", path)
			if err == nil {
				break
			}
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("session: connecting to the server: %w", err)
			}
		}
	}
	c := &Client{conn: conn, pending: map[int64]chan Response{}, closed: make(chan struct{})}
	go c.read()
	return c, nil
}

// Closed closes when the control connection is lost.
func (c *Client) Closed() <-chan struct{} { return c.closed }

func (c *Client) read() {
	r := bufio.NewReaderSize(c.conn, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
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
	conn, err := net.Dial("unix", SocketPath())
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
func (c *Client) Archive(root string) (data []byte, err error) {
	err = c.call(Request{Op: "archive", Path: root}, &data)
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

var _ = exec.Command
