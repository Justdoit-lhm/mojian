//go:build !darwin && !linux && !windows

package session

import (
	"io"
	"time"
)

// Client is unavailable on this platform.
type Client struct{}
type Stream struct{}

var closedChan = func() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

func (c *Client) Hello() (Hello, error)                     { return Hello{}, errUnsupported }
func (c *Client) List() ([]SessionInfo, error)              { return nil, errUnsupported }
func (c *Client) Create(CreateOptions) (SessionInfo, error) { return SessionInfo{}, errUnsupported }
func (c *Client) Kill(string) error                         { return errUnsupported }
func (c *Client) Resize(string, int, int) error             { return errUnsupported }
func (c *Client) Shutdown() error                           { return errUnsupported }
func Stop() error                                           { return errUnsupported }
func Ready() bool                                           { return false }
func (c *Client) Close() error                              { return nil }
func (c *Client) Closed() <-chan struct{}                   { return closedChan }
func (c *Client) Stream(string, int, int) *Stream           { return &Stream{} }

// ConnectAt is unavailable where the session server does not run.
func ConnectAt(string) (*Client, error) { return nil, errUnsupported }

func (c *Client) ListTree(string) ([]FileEntry, error)      { return nil, errUnsupported }
func (c *Client) ListTreeFresh(string) ([]FileEntry, error) { return nil, errUnsupported }
func (c *Client) ListDirs(string) ([]FileEntry, error)      { return nil, errUnsupported }
func (c *Client) ReadFile(string) ([]byte, error)           { return nil, errUnsupported }
func (c *Client) WriteFile(string, []byte) error            { return errUnsupported }
func (c *Client) Stat(string) (FileEntry, error)            { return FileEntry{}, errUnsupported }
func (c *Client) Copy(string, string) error                 { return errUnsupported }
func (c *Client) Mkdir(string) error                        { return errUnsupported }
func (c *Client) Rename(string, string) error               { return errUnsupported }
func (c *Client) Remove(string) error                       { return errUnsupported }
func (c *Client) Exec(string, []string, []string, time.Duration) (ExecResult, error) {
	return ExecResult{}, errUnsupported
}

func (s *Stream) Read([]byte) (int, error)  { return 0, io.EOF }
func (s *Stream) Write([]byte) (int, error) { return 0, io.EOF }
func (s *Stream) Close() error              { return nil }
func (s *Stream) Resize(int, int) error     { return nil }
