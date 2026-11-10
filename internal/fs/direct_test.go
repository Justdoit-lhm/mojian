//go:build darwin || linux

package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectListReportsUnreadableRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read an unreadable directory")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(root, 0o755)
	if entries, _, err := NewDirect(root).List(); err == nil {
		t.Fatalf("an unreadable root listed as %d entries", len(entries))
	}
}
