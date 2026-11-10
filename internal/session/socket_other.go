//go:build !darwin && !linux && !windows

package session

func SocketPath() string { return "" }
