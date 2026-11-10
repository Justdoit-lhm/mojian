package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSSHPickerListsHosts(t *testing.T) {
	if !remoteSupported() {
		t.Skip("SSH is unavailable")
	}
	a := newProjectApp(t)
	a.settingsPath = ""
	a.hostService().SetHosts([]hostConfig{{Name: "Build box", Address: "buildbox"}})
	a.openSSHPicker()
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if !a.sshPick || !tt.HasText("Connect to Server") || !tt.HasText("Build box") || !tt.HasText("buildbox") {
		t.Fatalf("ssh picker: open=%v texts=%q", a.sshPick, tt.Texts())
	}
}

func TestTerminalLayoutRoundTrip(t *testing.T) {
	if !remoteSupported() {
		t.Skip("remote terminals are unavailable")
	}
	dir := t.TempDir()
	a := &app{workspace: &workspace{layoutPath: filepath.Join(dir, "layout.json"), store: &store{}}}
	p := &pane{ID: nextID("p"), Term: &termTab{SID: "sess-1", Spec: termSpec{Host: "buildbox", Server: "buildbox"}}}
	tb := &tab{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}
	a.tabs = []*tab{tb}
	a.active = 0
	a.saveLayout()

	b := &app{workspace: &workspace{layoutPath: a.layoutPath, store: &store{}}}
	if !b.restoreLayout() {
		t.Fatal("restoreLayout failed")
	}
	rp := b.tabs[0].panes()[0]
	if rp.Term == nil || rp.Term.SID != "sess-1" || rp.Term.Spec.Host != "buildbox" || rp.Term.Spec.Server != "buildbox" {
		t.Fatalf("restored term: %+v", rp.Term)
	}
}
