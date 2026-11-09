package app

import (
	"testing"

	"mojian/internal/remote"
	"mojian/internal/session"

	"github.com/egoist/mygo/ui"
)

func TestRemotePickerReusesConnectionAndHome(t *testing.T) {
	if !terminalsSupported() {
		t.Skip("remote projects need terminals")
	}
	startLocalServerForTest(t)
	a := newProjectApp(t)
	client, err := session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	a.setRemoteConn(&remote.Conn{Host: "buildbox", Client: client})
	a.keepRemote("buildbox")
	a.remoteHomes = map[string]string{"buildbox": t.TempDir()}

	a.openRemotePicker("buildbox")
	if a.remoteBusy {
		t.Fatal("a kept connection still used the loading card")
	}
	if !a.remotePicker || a.remotePickHost != "buildbox" || a.remoteHost != "buildbox" {
		t.Fatalf("picker: open=%v host=%q shown=%q", a.remotePicker, a.remotePickHost, a.remoteHost)
	}
}

func TestRemotePickerScrollsItsFolders(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.remotePicker = true
	a.remoteHost = "Buildbox"
	a.remoteQuery = "/home/me"
	for i := 0; i < 30; i++ {
		name := "Folder" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		a.remoteDirs = append(a.remoteDirs, session.FileEntry{Rel: "/home/me/" + name, Dir: true})
	}
	tt := ui.NewTester(a.view, 900, 640)
	tt.Frame()
	before, ok := tt.Find("Open /home/me/FolderF0")
	if !ok {
		t.Fatal("no folder row")
	}
	tt.Scroll(450, 400, 0, 240)
	tt.Frame()
	after, ok := tt.Find("Open /home/me/FolderF0")
	if !ok {
		t.Fatal("the folder row disappeared")
	}
	if after.Y >= before.Y {
		t.Fatalf("the list did not scroll: before y=%.1f after y=%.1f", before.Y, after.Y)
	}
}
