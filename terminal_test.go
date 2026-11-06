package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTerminalCommand(t *testing.T) {
	cmd, err := terminalCommand("darwin", "/vault")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(cmd.Path) != "open" {
		t.Errorf("path: %q", cmd.Path)
	}
	want := []string{"-a", "Terminal", "/vault"}
	if len(cmd.Args) != len(want)+1 {
		t.Fatalf("args: %q", cmd.Args)
	}
	for i, arg := range want {
		if cmd.Args[i+1] != arg {
			t.Errorf("arg %d: %q, want %q", i, cmd.Args[i+1], arg)
		}
	}

	t.Setenv("TERMINAL", "sh -l")
	cmd, err = terminalCommand("linux", "/vault")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(cmd.Path) != "sh" {
		t.Errorf("path: %q", cmd.Path)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "-l" {
		t.Errorf("args: %q", cmd.Args)
	}
	if cmd.Dir != "/vault" {
		t.Errorf("dir: %q", cmd.Dir)
	}
}

func TestOpenTerminalAt(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the macOS open command")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$OUT\"\n"
	if err := os.WriteFile(filepath.Join(bin, "open"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	t.Setenv("OUT", out)
	if err := openTerminalAt("/tmp/vault"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(out)
		if err == nil {
			got := strings.TrimSpace(string(data))
			if got != "-a\nTerminal\n/tmp/vault" {
				t.Fatalf("terminal args: %q", got)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the fake terminal never ran: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
