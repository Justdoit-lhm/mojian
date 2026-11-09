package app

import (
	"errors"
	"strings"
	"testing"

	"mojian/internal/update"

	"github.com/egoist/mygo/ui"
)

func TestSettingsUpdateActions(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settings.Cat = settingsAbout
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	if !tt.HasText("Check for Updates") {
		t.Fatalf("update settings: %q", tt.Texts())
	}
	if !tt.HasText("Restart to Update") {
		t.Fatalf("update settings: %q", tt.Texts())
	}

	a.pendingUpdate = &update.Release{Version: "0.1.2"}
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

func TestFailedUpdateCheckDoesNotClaimTheLatestVersion(t *testing.T) {
	a := &app{workspace: &workspace{}}
	a.updateChecking = true
	a.updateChecked = true
	a.applyUpdateCheck(nil, errors.New("offline"))
	if a.updateChecked {
		t.Fatal("a failed check marked the app as current")
	}
	if a.updateErr == "" {
		t.Fatal("a failed check did not keep its error")
	}
	if got := a.updateSummary(); got != "" {
		t.Fatalf("failed check summary: %q", got)
	}
}

func TestUpdateStateLandsOnTheThread(t *testing.T) {
	a := &app{workspace: &workspace{}}
	a.applyUpdateCheck(&update.Release{Version: "0.3.0"}, nil)
	if !a.updateChecked || a.updateErr != "" || a.pendingUpdate == nil {
		t.Fatalf("check state: %+v", a)
	}
	if got := a.updateSummary(); !strings.Contains(got, "0.3.0") {
		t.Fatalf("available summary: %q", got)
	}
	a.updateInstalling = true
	a.applyUpdateInstall(errors.New("signature"))
	if a.updateInstalling || a.updateErr != "signature" || a.updateReady {
		t.Fatalf("install error state: %+v", a)
	}
	a.applyUpdateInstall(nil)
	if !a.updateReady || a.updateErr != "" {
		t.Fatalf("install ready state: %+v", a)
	}
}
