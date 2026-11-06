package main

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

func TestGitStatusAndDiff(t *testing.T) {
	dir := t.TempDir()
	makeVault(t, dir, map[string]string{
		"a.md":      "# A\n\nold\n",
		"b.js":      "const b = 1;\n",
		"c.py":      "print('c')\n",
		"sub/d.txt": "d\n",
	})
	initRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\nnew line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.py"), []byte("print('new')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := readGitStatus(dir)
	if data.err != "" {
		t.Fatalf("status: %s", data.err)
	}
	if data.branch == "" || data.branch == "detached" {
		t.Fatalf("branch: %q", data.branch)
	}
	if len(data.changes) != 2 {
		t.Fatalf("changes: %+v", data.changes)
	}
	var modified, untracked *gitChange
	for i := range data.changes {
		switch data.changes[i].Display {
		case "a.md":
			modified = &data.changes[i]
		case "new.py":
			untracked = &data.changes[i]
		}
	}
	if modified == nil || modified.Status != "M" {
		t.Fatalf("modified: %+v", modified)
	}
	if untracked == nil || untracked.Status != "U" {
		t.Fatalf("untracked: %+v", untracked)
	}
	diff, err := readGitDiff(data.root, *modified)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "+new line") || !strings.Contains(diff, "-old") {
		t.Fatalf("diff:\n%s", diff)
	}
	diff, err = readGitDiff(data.root, *untracked)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "new file") || !strings.Contains(diff, "+print('new')") {
		t.Fatalf("untracked diff:\n%s", diff)
	}
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
	if !a.gitOpen {
		t.Fatal("the diff page did not open")
	}
	if !tt.HasText("Changes") || !tt.HasText("a.md") {
		t.Fatalf("diff page texts: %q", tt.Texts())
	}
	if a.gitDiff == "" || !strings.Contains(a.gitDiff, "+new") {
		t.Fatalf("diff: %q", a.gitDiff)
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
	if !strings.Contains(a.gitErr, "not a Git repository") {
		t.Fatalf("error: %q", a.gitErr)
	}
	if !tt.HasText("This folder is not a Git repository.") {
		t.Fatalf("texts: %q", tt.Texts())
	}
}

func TestGitNestedProject(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	makeVault(t, sub, map[string]string{"a.md": "# A\n\nold\n"})
	initRepo(t, root)
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\n\nnew\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := readGitStatus(sub)
	if data.err != "" {
		t.Fatal(data.err)
	}
	if len(data.changes) != 1 {
		t.Fatalf("changes: %+v", data.changes)
	}
	if data.changes[0].Display != "a.md" || data.changes[0].Path != filepath.Join("sub", "a.md") {
		t.Fatalf("paths: %+v", data.changes[0])
	}
	diff, err := readGitDiff(data.root, data.changes[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "+new") {
		t.Fatalf("diff:\n%s", diff)
	}
}

func TestParseGitDiff(t *testing.T) {
	diff := `diff --git a/x.py b/x.py
index 111..222 100644
--- a/x.py
+++ b/x.py
@@ -1,3 +1,4 @@
 old
-removed
+added
+more
 context
`
	lines := parseGitDiff(diff)
	var kinds []byte
	for _, line := range lines {
		kinds = append(kinds, line.kind)
	}
	if string(kinds) != "@ -++ " {
		t.Fatalf("kinds: %q (lines=%+v)", kinds, lines)
	}
	if add, del := gitDiffStat(diff); add != 2 || del != 1 {
		t.Fatalf("stat: +%d -%d", add, del)
	}
	if lines[2].old != 2 || lines[2].new != 0 {
		t.Fatalf("deleted numbers: %+v", lines[2])
	}
	if lines[3].old != 0 || lines[3].new != 2 {
		t.Fatalf("added numbers: %+v", lines[3])
	}
	if lines[5].old != 3 || lines[5].new != 4 {
		t.Fatalf("context numbers: %+v", lines[5])
	}
}
