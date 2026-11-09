package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mojian/internal/git"
	"mojian/internal/testutil"
)

func TestWorkingTreeStatusAndDiff(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteVault(t, dir, map[string]string{
		"a.md": "# A\n\nold\n", "b.js": "const b = 1;\n",
		"sub/d.txt": "d\n",
	})
	testutil.InitRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\nnew line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.py"), []byte("print('new')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := git.ReadStatus(git.LocalRunner{}, dir)
	if data.Err != "" || data.Branch == "" || data.Branch == "detached" {
		t.Fatalf("status: %+v", data)
	}
	if len(data.Changes) != 2 {
		t.Fatalf("changes: %+v", data.Changes)
	}
	var modified, untracked *git.Change
	for i := range data.Changes {
		switch data.Changes[i].Display {
		case "a.md":
			modified = &data.Changes[i]
		case "new.py":
			untracked = &data.Changes[i]
		}
	}
	if modified == nil || modified.Status != "M" || untracked == nil || untracked.Status != "U" {
		t.Fatalf("changes: %+v", data.Changes)
	}
	diff, err := git.ReadDiff(git.LocalRunner{}, data.Root, *modified)
	if err != nil || !strings.Contains(diff, "+new line") || !strings.Contains(diff, "-old") {
		t.Fatalf("diff: %q %v", diff, err)
	}
	diff, err = git.ReadDiff(git.LocalRunner{}, data.Root, *untracked)
	if err != nil || !strings.Contains(diff, "new file") || !strings.Contains(diff, "+print('new')") {
		t.Fatalf("untracked diff: %q %v", diff, err)
	}
}

func TestNestedProjectPaths(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	testutil.WriteVault(t, sub, map[string]string{"a.md": "# A\n\nold\n"})
	testutil.InitRepo(t, root)
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\n\nnew\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := git.ReadStatus(git.LocalRunner{}, sub)
	if data.Err != "" || len(data.Changes) != 1 {
		t.Fatalf("status: %+v", data)
	}
	if data.Changes[0].Display != "a.md" || filepath.ToSlash(data.Changes[0].Path) != "sub/a.md" {
		t.Fatalf("paths: %+v", data.Changes[0])
	}
}

func TestRenameDiffKeepsTheRename(t *testing.T) {
	dir := t.TempDir()
	var body strings.Builder
	for i := 1; i <= 40; i++ {
		body.WriteString("line ")
		body.WriteString(strings.Repeat("x", i%3))
		body.WriteString("\n")
	}
	testutil.WriteVault(t, dir, map[string]string{"pack.go": body.String()})
	testutil.InitRepo(t, dir)
	if err := os.Rename(filepath.Join(dir, "pack.go"), filepath.Join(dir, "moved.go")); err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(body.String(), "line 1\n", "line one\n", 1)
	if err := os.WriteFile(filepath.Join(dir, "moved.go"), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", "-A")
	data := git.ReadStatus(git.LocalRunner{}, dir)
	var renamed *git.Change
	for i := range data.Changes {
		if data.Changes[i].Status == "R" {
			renamed = &data.Changes[i]
		}
	}
	if renamed == nil {
		t.Fatalf("status: %+v", data.Changes)
	}
	if renamed.OldPath != "pack.go" {
		t.Fatalf("rename source: %q", renamed.OldPath)
	}
	diff, err := git.ReadDiff(git.LocalRunner{}, data.Root, *renamed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "rename from pack.go") || !strings.Contains(diff, "rename to moved.go") {
		t.Fatalf("rename diff: %q", diff)
	}
}
