package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestSettingsUpdateActions(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settingsCat = settingsUpdates
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	if !tt.HasText("Check for Updates") {
		t.Fatalf("update settings: %q", tt.Texts())
	}
	if !tt.HasText("Restart to Update") {
		t.Fatalf("update settings: %q", tt.Texts())
	}

	a.pendingUpdate = &updateRelease{Version: "0.1.2"}
	tt.Frame()
	if !tt.HasText("Install Update") {
		t.Fatalf("available update settings: %q", tt.Texts())
	}
	if !tt.HasText("Mojian 0.1.2 is available.") {
		t.Fatalf("available update status: %q", tt.Texts())
	}

	a.updateReady = true
	tt.Frame()
	if !tt.HasText("Mojian 0.1.2 is ready. Restart to apply it.") {
		t.Fatalf("ready update status: %q", tt.Texts())
	}
	if tt.HasText("Install Update") {
		t.Fatalf("the install button survived a staged update: %q", tt.Texts())
	}

	relaunched := false
	a.updateRelaunchFn = func() { relaunched = true }
	if err := tt.Click("Restart to Update"); err != nil {
		t.Fatal(err)
	}
	if !relaunched {
		t.Fatal("Restart to Update did not relaunch the app")
	}
}

func TestUpdateDisabledMessage(t *testing.T) {
	got := updateErrorMessage(mygo.ErrUpdatesDisabled)
	if !strings.Contains(got, "cannot update itself") {
		t.Fatalf("disabled update message: %q", got)
	}
}
