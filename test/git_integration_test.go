package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mojian/internal/git"
)

func TestWorkingTreeStatusAndDiff(t *testing.T) {
	dir := t.TempDir()
	WriteVault(t, dir, map[string]string{
		"a.md": "# A\n\nold\n", "b.js": "const b = 1;\n",
		"sub/d.txt": "d\n",
	})
	InitRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\nnew line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.py"), []byte("print('new')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := git.ReadStatus(dir)
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
	diff, err := git.ReadDiff(data.Root, *modified)
	if err != nil || !strings.Contains(diff, "+new line") || !strings.Contains(diff, "-old") {
		t.Fatalf("diff: %q %v", diff, err)
	}
	diff, err = git.ReadDiff(data.Root, *untracked)
	if err != nil || !strings.Contains(diff, "new file") || !strings.Contains(diff, "+print('new')") {
		t.Fatalf("untracked diff: %q %v", diff, err)
	}
}

func TestNestedProjectPaths(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	WriteVault(t, sub, map[string]string{"a.md": "# A\n\nold\n"})
	InitRepo(t, root)
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\n\nnew\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := git.ReadStatus(sub)
	if data.Err != "" || len(data.Changes) != 1 {
		t.Fatalf("status: %+v", data)
	}
	if data.Changes[0].Display != "a.md" || filepath.ToSlash(data.Changes[0].Path) != "sub/a.md" {
		t.Fatalf("paths: %+v", data.Changes[0])
	}
}
