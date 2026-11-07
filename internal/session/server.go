//go:build darwin || linux

package session

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/egoist/mygo/plugins/terminal"
)

const (
	scrollback  = 8 << 20
	resyncDelay = 150 * time.Millisecond
	idleTimeout = 10 * time.Minute
)

// Server owns the sessions and their pseudo-terminals.
type Server struct {
	mu       sync.Mutex
	sessions map[string]*session
	order    []string
	started  time.Time
	ln       net.Listener
	quit     chan struct{}
	once     sync.Once
	controls int
	idleFrom time.Time
}

// Serve runs the session server until it is shut down.
func Serve() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "server.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if err := lockFile(lock); err != nil {
		return errors.New("session: a server already runs")
	}
	defer lock.Close()
	sock := SocketPath()
	_ = os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return err
	}
	_ = os.Chmod(sock, 0o600)
	s := &Server{
		sessions: map[string]*session{}, order: nil, started: time.Now(),
		ln: ln, quit: make(chan struct{}), idleFrom: time.Now(),
	}
	go func() {
		<-s.quit
		_ = ln.Close()
	}()
	go s.watchIdle()
	log.Printf("session: serving on %s (pid %d)", sock, os.Getpid())
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.quit:
				s.mu.Lock()
				all := make([]*session, 0, len(s.sessions))
				for _, ss := range s.sessions {
					all = append(all, ss)
				}
				s.mu.Unlock()
				for _, ss := range all {
					ss.kill()
				}
				_ = os.Remove(sock)
				return nil
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) shutdown() { s.once.Do(func() { close(s.quit) }) }

func (s *Server) watchIdle() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.quit:
			return
		case <-t.C:
		}
		s.mu.Lock()
		idle := len(s.sessions) == 0 && s.controls == 0 && time.Since(s.idleFrom) > idleTimeout
		s.mu.Unlock()
		if idle {
			s.shutdown()
			return
		}
	}
}

func (s *Server) handle(conn net.Conn) {
	r := bufio.NewReaderSize(conn, 64<<10)
	first, err := r.ReadBytes('\n')
	if err != nil {
		_ = conn.Close()
		return
	}
	var a Attach
	if json.Unmarshal(first, &a) == nil && a.Op == "attach" {
		s.mu.Lock()
		ss := s.sessions[a.SID]
		s.mu.Unlock()
		if ss == nil {
			_, _ = conn.Write([]byte(`{"ok":false,"error":"no such session"}` + "\n"))
			_ = conn.Close()
			return
		}
		_, _ = conn.Write([]byte(`{"ok":true}` + "\n"))
		if n := r.Buffered(); n > 0 {
			b, _ := r.Peek(n)
			ss.input(b)
		}
		ss.attach(conn, a.Cols, a.Rows)
		return
	}
	s.control(conn, r, first)
}

func (s *Server) control(conn net.Conn, r *bufio.Reader, first []byte) {
	s.mu.Lock()
	s.controls++
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.controls--
		s.idleFrom = time.Now()
		s.mu.Unlock()
		_ = conn.Close()
	}()
	var wmu sync.Mutex
	reply := func(res Response) {
		b, _ := json.Marshal(res)
		wmu.Lock()
		_, _ = conn.Write(append(b, '\n'))
		wmu.Unlock()
	}
	line := first
	for {
		var req Request
		if json.Unmarshal(line, &req) == nil {
			reply(s.request(req))
		}
		var err error
		line, err = r.ReadBytes('\n')
		if err != nil {
			return
		}
	}
}

func (s *Server) request(req Request) Response {
	data, err := s.do(req)
	res := Response{ID: req.ID}
	if err != nil {
		res.Error = err.Error()
	} else if data != nil {
		res.Data, _ = json.Marshal(data)
	}
	return res
}

func (s *Server) session(id string) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ss := s.sessions[id]; ss != nil {
		return ss, nil
	}
	return nil, fmt.Errorf("no session %q", id)
}

func (s *Server) do(req Request) (any, error) {
	switch req.Op {
	case "hello":
		return Hello{Version: ProtocolVersion, PID: os.Getpid(), Started: s.started}, nil
	case "list":
		s.mu.Lock()
		all := make([]*session, 0, len(s.order))
		for _, id := range s.order {
			all = append(all, s.sessions[id])
		}
		s.mu.Unlock()
		out := make([]SessionInfo, len(all))
		for i, ss := range all {
			out[i] = ss.info()
		}
		return out, nil
	case "create":
		o := CreateOptions{}
		if req.Create != nil {
			o = *req.Create
		}
		id := newID()
		ss, err := newSession(id, o)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.sessions[id] = ss
		s.order = append(s.order, id)
		s.mu.Unlock()
		return ss.info(), nil
	case "kill":
		ss, err := s.session(req.SID)
		if err != nil {
			return nil, nil
		}
		ss.kill()
		s.mu.Lock()
		delete(s.sessions, req.SID)
		s.order = removeString(s.order, req.SID)
		if len(s.sessions) == 0 {
			s.idleFrom = time.Now()
		}
		s.mu.Unlock()
		return nil, nil
	case "resize":
		ss, err := s.session(req.SID)
		if err != nil {
			return nil, err
		}
		ss.resize(req.Cols, req.Rows)
		return nil, nil
	case "shutdown":
		s.shutdown()
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func removeString(xs []string, x string) []string {
	out := xs[:0]
	for _, s := range xs {
		if s != x {
			out = append(out, s)
		}
	}
	return out
}

func newID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type session struct {
	id      string
	shell   string
	created time.Time
	cmd     *exec.Cmd
	pty     *os.File

	mu         sync.Mutex
	vt         *terminal.Terminal
	vtIn       *io.PipeWriter
	clients    map[*attached]struct{}
	cols, rows int
	exited     bool
	code       int
	done       chan struct{}
	output     uint64
	lastOutput time.Time
	resync     *time.Timer
}

type attached struct {
	conn net.Conn
	out  chan []byte
	once sync.Once
}

func (a *attached) send(p []byte) bool {
	select {
	case a.out <- p:
		return true
	default:
		return false
	}
}

func (a *attached) finish() { a.once.Do(func() { close(a.out) }) }

func (a *attached) writer() {
	for p := range a.out {
		if _, err := a.conn.Write(p); err != nil {
			break
		}
	}
	_ = a.conn.Close()
	for range a.out {
	}
}

func newSession(id string, o CreateOptions) (*session, error) {
	argv, name, err := commandArgs(o.Command)
	if err != nil {
		return nil, err
	}
	cols, rows := o.Cols, o.Rows
	if cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	dir := o.Dir
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = sessionEnv(o.Env)
	p, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	vt, err := terminal.New(terminal.Options{Conn: discard{pr}, Scrollback: scrollback})
	if err != nil {
		_ = p.Close()
		_ = cmd.Process.Kill()
		return nil, err
	}
	vt.Resize(cols, rows)
	s := &session{
		id: id, shell: name, created: time.Now(), cmd: cmd, pty: p,
		vt: vt, vtIn: pw, clients: map[*attached]struct{}{},
		cols: cols, rows: rows, done: make(chan struct{}),
	}
	readDone := make(chan struct{})
	go func() {
		s.read()
		close(readDone)
	}()
	go func() {
		code := cmd.Wait()
		select {
		case <-readDone:
		case <-time.After(200 * time.Millisecond):
			_ = p.Close()
			<-readDone
		}
		s.mu.Lock()
		s.exited, s.code = true, exitCode(code)
		for a := range s.clients {
			a.finish()
		}
		s.clients = map[*attached]struct{}{}
		s.mu.Unlock()
		close(s.done)
		_ = pw.Close()
		vt.Close()
	}()
	return s, nil
}

func commandArgs(command []string) ([]string, string, error) {
	if len(command) > 0 {
		path, err := exec.LookPath(command[0])
		if err != nil {
			return nil, "", err
		}
		return append([]string{path}, command[1:]...), filepath.Base(command[0]), nil
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	path, err := exec.LookPath(sh)
	if err != nil {
		return nil, "", err
	}
	return []string{path, "-l"}, filepath.Base(sh), nil
}

func sessionEnv(extra []string) []string {
	var env []string
	hasLang := false
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		switch {
		case k == "TERM", k == "COLORTERM", k == "TERM_PROGRAM", k == "TERM_PROGRAM_VERSION",
			k == "TERM_SESSION_ID", k == "TMUX", k == "TMUX_PANE", k == "STY", k == "WINDOWID",
			k == "COLUMNS", k == "LINES", k == "SHLVL", k == "OLDPWD", k == "PWD", k == "_",
			strings.HasPrefix(k, "ITERM_"), strings.HasPrefix(k, "GHOSTTY_"), strings.HasPrefix(k, "KITTY_"),
			strings.HasPrefix(k, "VSCODE_"), strings.HasPrefix(k, "WEZTERM_"), strings.HasPrefix(k, "ALACRITTY_"),
			strings.HasPrefix(k, "MOJIAN_"):
			continue
		case k == "LANG" || k == "LC_ALL" || k == "LC_CTYPE":
			hasLang = true
		}
		env = append(env, kv)
	}
	env = append(env, "TERM=xterm-256color", "COLORTERM=truecolor", "TERM_PROGRAM=Mojian")
	if !hasLang {
		env = append(env, "LANG=en_US.UTF-8")
	}
	return append(env, extra...)
}

func (s *session) read() {
	buf := make([]byte, 64<<10)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.vt.Feed(data)
			s.output += uint64(n)
			s.lastOutput = time.Now()
			for a := range s.clients {
				if !a.send(data) {
					a.finish()
					delete(s.clients, a)
				}
			}
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (s *session) attach(conn net.Conn, cols, rows int) {
	a := &attached{conn: conn, out: make(chan []byte, 4096)}
	s.mu.Lock()
	if cols > 0 && rows > 0 {
		s.cols, s.rows = cols, rows
		s.vt.Resize(cols, rows)
	}
	snap := s.vt.Snapshot()
	if title := s.vt.Title(); title != "" {
		snap = append([]byte("\x1b]2;"+title+"\x07"), snap...)
	}
	a.out <- snap
	exited := s.exited
	if !exited {
		s.clients[a] = struct{}{}
	}
	s.mu.Unlock()
	go a.writer()
	if exited {
		a.finish()
		_ = conn.Close()
		return
	}
	if cols > 0 && rows > 0 {
		s.resize(cols, rows)
	}
	buf := make([]byte, 32<<10)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			s.input(buf[:n])
		}
		if err != nil {
			break
		}
	}
	s.mu.Lock()
	if _, ok := s.clients[a]; ok {
		delete(s.clients, a)
		a.finish()
	}
	s.mu.Unlock()
}

func (s *session) input(p []byte) {
	s.mu.Lock()
	exited := s.exited
	s.mu.Unlock()
	if !exited {
		_, _ = s.pty.Write(p)
	}
}

func (s *session) resize(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	s.mu.Lock()
	changed := cols != s.cols || rows != s.rows
	s.cols, s.rows = cols, rows
	exited := s.exited
	if changed {
		s.vt.Resize(cols, rows)
		if s.resync == nil {
			s.resync = time.AfterFunc(resyncDelay, s.resyncScreen)
		} else {
			s.resync.Reset(resyncDelay)
		}
	}
	s.mu.Unlock()
	if changed && !exited {
		_ = pty.Setsize(s.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}
}

func (s *session) resyncScreen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.exited || len(s.clients) == 0 {
		return
	}
	snap := s.vt.Snapshot()
	if !bytes.Contains(snap, []byte("\x1b[?1049h")) {
		return
	}
	msg := append([]byte("\x1b[?1049h\x1b[H\x1b[2J"), snap...)
	for a := range s.clients {
		if !a.send(msg) {
			a.finish()
			delete(s.clients, a)
		}
	}
}

func (s *session) kill() {
	s.mu.Lock()
	exited := s.exited
	s.mu.Unlock()
	if !exited {
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Signal(os.Interrupt)
		}
	}
	select {
	case <-s.done:
	case <-time.After(time.Second):
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
	}
}

func (s *session) info() SessionInfo {
	s.mu.Lock()
	in := SessionInfo{
		ID: s.id, Shell: s.shell, Created: s.created,
		Cols: s.cols, Rows: s.rows, Title: s.vt.Title(),
		Exited: s.exited, ExitCode: s.code,
	}
	s.mu.Unlock()
	if s.cmd.Process != nil {
		in.PID = s.cmd.Process.Pid
	}
	return in
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

type discard struct{ r *io.PipeReader }

func (d discard) Read(p []byte) (int, error)  { return d.r.Read(p) }
func (d discard) Write(p []byte) (int, error) { return len(p), nil }
func (d discard) Close() error                { return d.r.Close() }
