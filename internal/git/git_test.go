package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeVault(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

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

func TestStatusAndDiff(t *testing.T) {
	dir := t.TempDir()
	writeVault(t, dir, map[string]string{
		"a.md": "# A\n\nold\n", "b.js": "const b = 1;\n",
		"c.py": "print('c')\n", "sub/d.txt": "d\n",
	})
	initRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\nnew line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.py"), []byte("print('new')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := readStatus(dir)
	if data.err != "" || data.branch == "" || data.branch == "detached" {
		t.Fatalf("status: %+v", data)
	}
	if len(data.changes) != 2 {
		t.Fatalf("changes: %+v", data.changes)
	}
	var modified, untracked *Change
	for i := range data.changes {
		switch data.changes[i].Display {
		case "a.md":
			modified = &data.changes[i]
		case "new.py":
			untracked = &data.changes[i]
		}
	}
	if modified == nil || modified.Status != "M" || untracked == nil || untracked.Status != "U" {
		t.Fatalf("changes: %+v", data.changes)
	}
	diff, err := readGitDiff(data.root, *modified)
	if err != nil || !strings.Contains(diff, "+new line") || !strings.Contains(diff, "-old") {
		t.Fatalf("diff: %q %v", diff, err)
	}
	diff, err = readGitDiff(data.root, *untracked)
	if err != nil || !strings.Contains(diff, "new file") || !strings.Contains(diff, "+print('new')") {
		t.Fatalf("untracked diff: %q %v", diff, err)
	}
}

func TestNestedProject(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	writeVault(t, sub, map[string]string{"a.md": "# A\n\nold\n"})
	initRepo(t, root)
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\n\nnew\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := readStatus(sub)
	if data.err != "" || len(data.changes) != 1 {
		t.Fatalf("status: %+v", data)
	}
	if data.changes[0].Display != "a.md" || data.changes[0].Path != filepath.Join("sub", "a.md") {
		t.Fatalf("paths: %+v", data.changes[0])
	}
}

func TestParseDiff(t *testing.T) {
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
	lines := ParseDiff(diff)
	var kinds []byte
	for _, line := range lines {
		kinds = append(kinds, line.kind)
	}
	if string(kinds) != "@ -++ " {
		t.Fatalf("kinds: %q (lines=%+v)", kinds, lines)
	}
	if add, del := DiffStat(diff); add != 2 || del != 1 {
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
