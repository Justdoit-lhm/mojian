package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	integration "mojian/test"

	"github.com/egoist/mygo/ui"
)

func TestGitPage(t *testing.T) {
	registerFonts()
	dir := t.TempDir()
	integration.WriteVault(t, dir, map[string]string{"a.md": "# A\n\nold\n", "keep.go": "package x\n"})
	integration.InitRepo(t, dir)
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
	integration.WriteVault(t, dir, map[string]string{"a.md": "# A\n"})
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
