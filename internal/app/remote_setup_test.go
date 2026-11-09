//go:build darwin || linux

package app

import (
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"mojian/internal/remote"
	"mojian/internal/session"
)

// TestRemoteServerSetupFedora exercises the real upload, server start,
// forward and file operations against a host named in
// MOJIAN_REMOTE_TEST_HOST. It is skipped unless that variable is set:
//
//	MOJIAN_REMOTE_TEST_HOST=user@host go test ./internal/app -run TestRemoteServerSetupFedora -v -timeout 10m
func TestRemoteServerSetupFedora(t *testing.T) {
	host := os.Getenv("MOJIAN_REMOTE_TEST_HOST")
	if host == "" {
		t.Skip("set MOJIAN_REMOTE_TEST_HOST=user@host to run this integration test")
	}
	first, err := remote.Provision(host, nil)
	if err != nil {
		t.Fatalf("provisioning %s: %v", host, err)
	}
	if first.Protocol != session.ProtocolVersion {
		t.Fatalf("server protocol %d, want %d", first.Protocol, session.ProtocolVersion)
	}
	if first.Conn == nil {
		t.Fatal("provisioning returned no connection")
	}
	defer remote.Close(first.Conn)
	second, err := remote.Provision(host, first.Conn)
	if err != nil {
		t.Fatalf("checking %s twice: %v", host, err)
	}
	if second.Uploaded {
		t.Fatal("the second setup uploaded the binaries again")
	}
	if second.Conn != first.Conn {
		t.Fatal("the second setup did not reuse the live connection")
	}

	rc, err := remote.Dial(host)
	if err != nil {
		t.Fatalf("connecting to %s: %v", host, err)
	}
	defer remote.Close(rc)

	root := "/tmp/mojian-test-" + strconv.Itoa(os.Getpid())
	defer func() { _, _ = rc.Client.Exec("", []string{"rm", "-rf", root}, nil, 10*time.Second) }()
	if err := rc.Client.Mkdir(root); err != nil {
		t.Fatal(err)
	}
	s, err := openServerStore(rc.Client, root, t.TempDir(), false)
	if err != nil {
		t.Fatalf("opening the server project: %v", err)
	}
	n, err := s.NewFileIn("", "fedora.md", "# Fedora\n\nUploaded by the app.\n")
	if err != nil {
		t.Fatalf("creating a file: %v", err)
	}
	n.Buffer().Set("# Fedora\n\nSaved through the server.\n")
	if err := s.Save(n); err != nil {
		t.Fatalf("saving: %v", err)
	}
	data, err := rc.Client.ReadFile(path.Join(root, "fedora.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Fedora\n\nSaved through the server.\n" {
		t.Fatalf("remote file: %q", data)
	}
	if _, _, err := s.Rename("fedora.md", "renamed.md"); err != nil {
		t.Fatalf("renaming: %v", err)
	}
	if s.ByRelPath("renamed.md") == nil {
		t.Fatal("the renamed file is missing from the store")
	}
	if _, err := rc.Client.Stat(path.Join(root, "fedora.md")); err == nil {
		t.Fatal("the old remote file remained")
	}
	if err := s.Refresh(); err != nil {
		t.Fatalf("refreshing: %v", err)
	}
	if s.ByRelPath("renamed.md") == nil {
		t.Fatal("the server listing lost the renamed file")
	}
	pic := []byte("\x89PNG\r\n\x1a\nremote-asset")
	if err := rc.Client.WriteFile(path.Join(root, "pic.png"), pic); err != nil {
		t.Fatal(err)
	}
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	asset := s.AssetByRel("pic.png")
	if asset == nil || asset.Hash == "" {
		t.Fatalf("remote asset listing: %+v", asset)
	}
	local, err := s.MaterializeAsset(asset)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(local); err != nil || string(data) != string(pic) {
		t.Fatalf("remote asset cache: %q, %v", data, err)
	}
}
