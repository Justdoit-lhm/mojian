//go:build !darwin && !linux

package session

import "io"

// Client is unavailable on this platform.
type Client struct{}
type Stream struct{}

func (c *Client) Hello() (Hello, error)                     { return Hello{}, errUnsupported }
func (c *Client) List() ([]SessionInfo, error)              { return nil, errUnsupported }
func (c *Client) Create(CreateOptions) (SessionInfo, error) { return SessionInfo{}, errUnsupported }
func (c *Client) Kill(string) error                         { return errUnsupported }
func (c *Client) Resize(string, int, int) error             { return errUnsupported }
func (c *Client) Shutdown() error                           { return errUnsupported }
func (c *Client) Close() error                              { return nil }
func (c *Client) Closed() <-chan struct{}                   { return nil }
func (c *Client) Stream(string, int, int) *Stream           { return &Stream{} }
func (s *Stream) Read([]byte) (int, error)                  { return 0, io.EOF }
func (s *Stream) Write([]byte) (int, error)                 { return 0, io.EOF }
func (s *Stream) Close() error                              { return nil }
func (s *Stream) Resize(int, int) error                     { return nil }
