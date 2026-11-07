//go:build darwin || linux

package session

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func startOpsServer(t *testing.T) (*Client, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("MOJIAN_DIR", dir)
	done := make(chan error, 1)
	go func() { done <- Serve() }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(SocketPath()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("session server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	c, err := ConnectAt(SocketPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.Shutdown()
		_ = c.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})
	return c, dir
}

func TestExecRunsCommandsWithOutputAndCode(t *testing.T) {
	c, _ := startOpsServer(t)
	dir := t.TempDir()
	res, err := c.Exec(dir, []string{"sh", "-c", "printf out; printf err >&2; exit 3"}, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if res.Stdout != "out" || res.Stderr != "err" || res.Code != 3 {
		t.Fatalf("result: %+v", res)
	}
}

func TestExecDropsInheritedGitVariables(t *testing.T) {
	t.Setenv("GIT_DIR", "/tmp/not-this-repository")
	c, _ := startOpsServer(t)
	res, err := c.Exec(t.TempDir(), []string{"sh", "-c", `printf %s "$GIT_DIR"`}, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if res.Stdout != "" {
		t.Fatalf("GIT_DIR leaked: %q", res.Stdout)
	}
}

func TestExecTimesOut(t *testing.T) {
	c, _ := startOpsServer(t)
	if _, err := c.Exec(t.TempDir(), []string{"sh", "-c", "sleep 5"}, nil, 150*time.Millisecond); err == nil {
		t.Fatal("a timed out command returned no error")
	}
}

func TestWriteFileAtomicKeepsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.sh")
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode: %v", info.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Fatalf("content: %q", data)
	}
}

func TestKillEndsTheWholeProcessGroup(t *testing.T) {
	c, dir := startOpsServer(t)
	pidFile := filepath.Join(dir, "child.pid")
	info, err := c.Create(CreateOptions{
		Command: []string{"sh", "-c", "sleep 60 & echo $! > " + pidFile + "; wait"},
		Dir:     dir,
		Cols:    40, Rows: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	var child int
	deadline := time.Now().Add(3 * time.Second)
	for {
		data, err := os.ReadFile(pidFile)
		if err == nil && strings.TrimSpace(string(data)) != "" {
			child, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the child sleep did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := c.Kill(info.ID); err != nil {
		t.Fatal(err)
	}
	// The child lives in the shell's process group; the group signal must
	// reach it, not only the shell.
	deadline = time.Now().Add(3 * time.Second)
	for {
		err := syscall.Kill(child, 0)
		if err != nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("child process %d survived the session kill", child)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
