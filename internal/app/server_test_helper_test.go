package app

import (
	"testing"
	"time"

	"mojian/internal/session"
)

// startLocalServerForTest gives a test that calls openProject a session
// server. The server is shut down when the test ends.
func startLocalServerForTest(t *testing.T) {
	t.Helper()
	if !terminalsSupported() {
		return
	}
	t.Setenv("MOJIAN_DIR", t.TempDir())
	done := make(chan error, 1)
	go func() { done <- session.Serve() }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if session.Ready() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("session server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	c, err := session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.Shutdown()
		_ = c.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("session server did not stop")
		}
	})
}
