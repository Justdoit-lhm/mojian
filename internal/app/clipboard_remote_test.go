//go:build darwin || linux

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteClipboardBelongsToItsProject(t *testing.T) {
	client, done := startTestSessionServer(t, t.TempDir())
	defer client.Close()
	defer func() { _ = client.Shutdown(); <-done }()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newRemoteApp(t, client, root)
	a.copySelectionTo("a.md", false)
	if !a.fileClipRemote || a.fileClipHost != "buildbox" || a.fileClipRoot != root {
		t.Fatalf("remote clipboard: %+v host=%q root=%q", a.fileClip, a.fileClipHost, a.fileClipRoot)
	}
	a.remotePath = "/srv/other"
	_, err := a.filePasteSource()
	if err == nil || !strings.Contains(err.Error(), "belongs to buildbox") {
		t.Fatalf("pasting another project's clipboard: %v", err)
	}
}
