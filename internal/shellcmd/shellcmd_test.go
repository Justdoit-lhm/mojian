//go:build darwin || linux

package shellcmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallAndRemoveZsh(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	exe := filepath.Join(home, "Applications", "Mojian.app", "Contents", "MacOS", "Mojian")

	st, err := Inspect(exe, Zsh)
	if err != nil {
		t.Fatal(err)
	}
	if st.Installed || st.Path != filepath.Join(home, ".zshrc") {
		t.Fatalf("fresh state: %+v", st)
	}

	st, err = Install(exe, Zsh)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Installed || !st.Current {
		t.Fatalf("state after install: %+v", st)
	}
	rc := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	want := "alias mojian='" + exe + "'"
	if !strings.Contains(string(data), want) {
		t.Fatalf("alias missing: %q", data)
	}

	// Installing again must not add a second block.
	if _, err := Install(exe, Zsh); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), beginMarker); n != 1 {
		t.Fatalf("%d blocks after a second install: %q", n, data)
	}

	// A moved app rewrites the alias in place.
	moved := filepath.Join(home, "elsewhere", "Mojian")
	st, err = Install(moved, Zsh)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Current {
		t.Fatalf("state after update: %+v", st)
	}
	data, err = os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), exe) || !strings.Contains(string(data), moved) {
		t.Fatalf("alias did not follow the app: %q", data)
	}

	st, err = Inspect(moved, Zsh)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Installed || !st.Current {
		t.Fatalf("inspect after update: %+v", st)
	}

	if err := Remove(Zsh); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), beginMarker) {
		t.Fatalf("block left behind: %q", data)
	}
	if st, err := Inspect(moved, Zsh); err != nil || st.Installed {
		t.Fatalf("inspect after remove: %+v, %v", st, err)
	}
}

func TestInstallKeepsExistingStartupFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("export EDITOR=vim"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(filepath.Join(home, "Mojian"), Zsh); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "export EDITOR=vim\n\n"+beginMarker) {
		t.Fatalf("existing line was not kept: %q", data)
	}
	if info, err := os.Stat(rc); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode after install: %v", info.Mode().Perm())
	}

	if err := Remove(Zsh); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "export EDITOR=vim\n" {
		t.Fatalf("after remove: %q", data)
	}
}

func TestRemoveKeepsLinesAfterTheBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rc := filepath.Join(home, ".zshrc")
	text := "before\n\n" + strings.TrimSuffix(blockText("/Applications/Mojian.app/Contents/MacOS/Mojian"), "\n") + "\nafter\n"
	if err := os.WriteFile(rc, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Remove(Zsh); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before\nafter\n" {
		t.Fatalf("after remove: %q", data)
	}
}

func TestIncompleteBlockIsNotTouched(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rc := filepath.Join(home, ".zshrc")
	text := beginMarker + "\nalias mojian='/tmp/mojian'\n"
	if err := os.WriteFile(rc, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install("/tmp/mojian", Zsh); err == nil {
		t.Fatal("install accepted a block without an end marker")
	}
	if err := Remove(Zsh); err == nil {
		t.Fatal("remove accepted a block without an end marker")
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != text {
		t.Fatalf("the file changed: %q", data)
	}
}

func TestRcFile(t *testing.T) {
	home := t.TempDir()
	if got := rcFile(home, Zsh, "darwin"); got != filepath.Join(home, ".zshrc") {
		t.Fatalf("zsh: %s", got)
	}
	if got := rcFile(home, Bash, "linux"); got != filepath.Join(home, ".bashrc") {
		t.Fatalf("bash on linux: %s", got)
	}
	if got := rcFile(home, Bash, "darwin"); got != filepath.Join(home, ".bash_profile") {
		t.Fatalf("bash with no startup file: %s", got)
	}
	profile := filepath.Join(home, ".profile")
	if err := os.WriteFile(profile, []byte("# profile\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := rcFile(home, Bash, "darwin"); got != profile {
		t.Fatalf("bash should reuse .profile: %s", got)
	}
	login := filepath.Join(home, ".bash_login")
	if err := os.WriteFile(login, []byte("# login\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := rcFile(home, Bash, "darwin"); got != login {
		t.Fatalf("bash should reuse .bash_login: %s", got)
	}
}

func TestParseAndDetect(t *testing.T) {
	if s, ok := Parse("Bash"); !ok || s != Bash {
		t.Fatalf("parse bash: %v, %v", s, ok)
	}
	if s, ok := Parse("sh"); ok || s != Zsh {
		t.Fatalf("parse sh: %v, %v", s, ok)
	}
	t.Setenv("SHELL", "/bin/bash")
	if got := Detect(); got != Bash {
		t.Fatalf("detect bash: %v", got)
	}
	t.Setenv("SHELL", "/bin/zsh")
	if got := Detect(); got != Zsh {
		t.Fatalf("detect zsh: %v", got)
	}
	t.Setenv("SHELL", "/usr/bin/fish")
	want := Bash
	if runtime.GOOS == "darwin" {
		want = Zsh
	}
	if got := Detect(); got != want {
		t.Fatalf("detect fallback: %v, want %v", got, want)
	}
}

func TestQuote(t *testing.T) {
	if got := quote("/a b/Mojian"); got != "'/a b/Mojian'" {
		t.Fatalf("quote: %s", got)
	}
	if got := quote("/it's/Mojian"); got != `'/it'\''s/Mojian'` {
		t.Fatalf("quote with a quote: %s", got)
	}
}
