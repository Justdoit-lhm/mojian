//go:build darwin || linux

package remote_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mojian/internal/remote"
)

// A cancelled connection must kill the ssh process it is waiting on
// instead of letting the loading card's work run to completion.
func TestSSHOutputContextStopsTheCommand(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	script := "#!/bin/sh\nprintf started > \"$MOJIAN_TEST_SSH_MARKER\"\nexec sleep 30\n"
	if err := os.WriteFile(filepath.Join(dir, "ssh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOJIAN_TEST_SSH_MARKER", marker)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := remote.SSHOutputContext(ctx, "test-host", "true")
		done <- err
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the fake ssh command did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled ssh returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the ssh command outlived its context")
	}
}
