package app

import (
	"os"
	"path/filepath"
	"testing"

	integration "mojian/test"
)

func TestPickerListsSymlinksAndHiddenFolders(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Visible", ".hidden"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "Visible"), filepath.Join(root, "Link")); err != nil {
		t.Skipf("symlinks need a Unix file system: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "Broken")); err != nil {
		t.Fatal(err)
	}

	names := func(a *app) map[string]bool {
		got := map[string]bool{}
		for _, e := range a.pickerEntries {
			got[e.Name] = true
		}
		return got
	}
	a := &app{}
	a.setPickerPath(root)
	if a.pickerErr != "" {
		t.Fatalf("picker: %s", a.pickerErr)
	}
	got := names(a)
	if !got["Visible"] || !got["Link"] {
		t.Errorf("regular and linked folders: %v", got)
	}
	if got[".hidden"] || got["note.md"] || got["Broken"] {
		t.Errorf("unexpected entries: %v", got)
	}

	a.pickerHidden = true
	a.setPickerPath(root)
	got = names(a)
	if !got[".hidden"] {
		t.Errorf("hidden folder missing: %v", got)
	}
	if !got["Link"] {
		t.Errorf("linked folder missing: %v", got)
	}
}

func TestProjectOpenedThroughASymlinkListsItsFiles(t *testing.T) {
	startLocalServerForTest(t)
	root := t.TempDir()
	real := filepath.Join(root, "Real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	integration.WriteVault(t, real, map[string]string{"a.md": "# A\n"})
	link := filepath.Join(root, "Link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks need a Unix file system: %v", err)
	}

	a := &app{workspace: &workspace{}, dataDir: t.TempDir()}
	a.openProject(link)
	if a.err != "" {
		t.Fatal(a.err)
	}
	target, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	if a.store == nil || a.store.Dir != target {
		t.Fatalf("project: %+v, want %s", a.store, target)
	}
	if n := a.store.ByRelPath("a.md"); n == nil {
		t.Fatalf("the linked project is empty: %+v", a.store.Entries)
	}
}
