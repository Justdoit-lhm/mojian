package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"mojian/internal/app"
)

func TestParseStartupFolderAndFile(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "notes.md")
	if err := os.WriteFile(file, []byte("# notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := parseStartup([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != dir || got.File != "" {
		t.Errorf("folder: %+v, want %+v", got, app.Startup{Dir: dir, File: ""})
	}

	got, err = parseStartup([]string{file})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != sub || got.File != file {
		t.Errorf("file: %+v, want %+v", got, app.Startup{Dir: sub, File: file})
	}
}

func TestParseStartupNoPath(t *testing.T) {
	got, err := parseStartup(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != (app.Startup{Dir: "", File: ""}) {
		t.Errorf("no path: %+v", got)
	}
}

func TestParseStartupRelativeAndTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := filepath.Join(home, "abc")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(proj, "README.md")
	if err := os.WriteFile(file, []byte("# abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := parseStartup([]string{"~/abc/README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != proj || got.File != file {
		t.Errorf("tilde: %+v, want %+v", got, app.Startup{Dir: proj, File: file})
	}

	t.Chdir(proj)
	got, err = parseStartup([]string{"README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != proj || got.File != file {
		t.Errorf("relative: %+v, want %+v", got, app.Startup{Dir: proj, File: file})
	}
}

func TestParseStartupErrors(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"missing", []string{filepath.Join(dir, "missing")}},
		{"two paths", []string{dir, dir}},
		{"option", []string{"-x"}},
	} {
		if _, err := parseStartup(tc.args); err == nil {
			t.Errorf("%s: no error", tc.name)
		}
	}
}

func TestParseStartupHelpAndLaunchServices(t *testing.T) {
	if _, err := parseStartup([]string{"--help"}); !errors.Is(err, errHelp) {
		t.Errorf("--help: %v", err)
	}
	dir := t.TempDir()
	got, err := parseStartup([]string{"-psn_0_12345", dir})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != dir {
		t.Errorf("LaunchServices argument: %+v", got)
	}
}

func TestShouldDetach(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stdinTTY bool
		dev      bool
		env      string
		want     bool
	}{
		{"terminal launch", true, false, "", true},
		{"piped or redirected stdin", false, false, "", false},
		{"mygo dev", true, true, "", false},
		{"already detached", true, false, "1", false},
	} {
		t.Setenv(detachedEnv, tc.env)
		if got := shouldDetach(tc.stdinTTY, tc.dev); got != tc.want {
			t.Errorf("%s: shouldDetach(%v, %v) = %v, want %v", tc.name, tc.stdinTTY, tc.dev, got, tc.want)
		}
	}
}
