package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func initRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "mojian@example.com")
	run("config", "user.name", "Mojian Test")
	run("add", ".")
	run("commit", "-qm", "initial")
}

func TestGitPage(t *testing.T) {
	registerFonts()
	dir := t.TempDir()
	makeVault(t, dir, map[string]string{"a.md": "# A\n\nold\n", "keep.go": "package x\n"})
	initRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\nnew\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &app{treeOpen: true}
	s, err := openStoreAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.focusFirstNote()
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	a.toggleGit()
	for range 3 {
		tt.Frame()
	}
	if !a.git.Open {
		t.Fatal("the diff page did not open")
	}
	if !tt.HasText("Changes") || !tt.HasText("a.md") {
		t.Fatalf("diff page texts: %q", tt.Texts())
	}
	if a.git.Diff == "" || !strings.Contains(a.git.Diff, "+new") {
		t.Fatalf("diff: %q", a.git.Diff)
	}
}

func TestGitOutsideRepo(t *testing.T) {
	registerFonts()
	dir := t.TempDir()
	makeVault(t, dir, map[string]string{"a.md": "# A\n"})
	a := &app{treeOpen: true}
	s, err := openStoreAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	tt := ui.NewTester(a.view, 1100, 700)
	a.toggleGit()
	for range 3 {
		tt.Frame()
	}
	if !strings.Contains(a.git.Err, "not a Git repository") {
		t.Fatalf("error: %q", a.git.Err)
	}
	if !tt.HasText("This folder is not a Git repository.") {
		t.Fatalf("texts: %q", tt.Texts())
	}
}
