package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSSHConfigHosts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	config := `Host alpha
  HostName 10.0.0.1
  User alice
  Port 2222

Host beta gamma
  HostName beta.example

Host *
  User ignored
`
	if err := os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	hosts := sshConfigHosts()
	if len(hosts) != 3 {
		t.Fatalf("hosts: %+v", hosts)
	}
	if hosts[0].Alias != "alpha" || hosts[0].HostName != "10.0.0.1" || hosts[0].User != "alice" || hosts[0].Port != "2222" {
		t.Fatalf("alpha: %+v", hosts[0])
	}
	if !argsContain(sshArgs("alpha"), "alpha") {
		t.Fatalf("ssh args: %+v", sshArgs("alpha"))
	}
}

func argsContain(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func TestSSHPickerListsHosts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.MkdirAll(filepath.Join(home, ".ssh"), 0o700)
	_ = os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte("Host buildbox\n  HostName build.example\n  User dev\n"), 0o600)
	a := newProjectApp(t)
	a.settingsPath = ""
	a.openSSHPicker()
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if !a.sshPick || !tt.HasText("Connect to Server") || !tt.HasText("buildbox") {
		t.Fatalf("ssh picker: open=%v texts=%q", a.sshPick, tt.Texts())
	}
}

func TestTerminalLayoutRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := &app{layoutPath: filepath.Join(dir, "layout.json"), store: &store{}}
	p := &pane{ID: nextID("p"), Term: &termTab{SID: "sess-1", Spec: termSpec{Host: "buildbox"}}}
	tb := &tab{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}
	a.tabs = []*tab{tb}
	a.active = 0
	a.saveLayout()

	b := &app{layoutPath: a.layoutPath, store: &store{}}
	if !b.restoreLayout() {
		t.Fatal("restoreLayout failed")
	}
	rp := b.tabs[0].panes()[0]
	if rp.Term == nil || rp.Term.SID != "sess-1" || rp.Term.Spec.Host != "buildbox" {
		t.Fatalf("restored term: %+v", rp.Term)
	}
}
