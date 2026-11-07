package app

import (
	"testing"

	"mojian/internal/session"

	"github.com/egoist/mygo/ui"
)

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
