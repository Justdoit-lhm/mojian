package commands

import "testing"

func TestAccelForUsesOtherOffMac(t *testing.T) {
	c := &Command{Accel: "CmdOrCtrl+D", Other: "Ctrl+Shift+O"}
	if got := c.AccelFor("darwin"); got != "CmdOrCtrl+D" {
		t.Fatalf("darwin: %q", got)
	}
	if got := c.AccelFor("linux"); got != "Ctrl+Shift+O" {
		t.Fatalf("linux: %q", got)
	}
	if got := c.AccelFor("js"); got != "Ctrl+Shift+O" {
		t.Fatalf("js: %q", got)
	}
	plain := &Command{Accel: "CmdOrCtrl+W"}
	if got := plain.AccelFor("linux"); got != "CmdOrCtrl+W" {
		t.Fatalf("fallback: %q", got)
	}
}
