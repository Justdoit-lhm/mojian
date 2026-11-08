package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mojian/internal/settings"

	"github.com/egoist/mygo/ui"
)

func TestSettingsSearchFindsRowsAndSwitchesPages(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	a.settingsQuery = "line numbers"
	tt.Frame()
	tt.Frame()
	if a.settingsCat != settingsEditor {
		t.Fatalf("page after searching line numbers: %d", a.settingsCat)
	}
	if !a.settingsHits["Line numbers"] {
		t.Fatalf("the row was not found: %v", a.settingsHits)
	}
	if !tt.HasText("Line numbers") {
		t.Fatalf("editor texts: %q", tt.Texts())
	}
	a.settingsQuery = "no such setting"
	tt.Frame()
	tt.Frame()
	if !tt.HasText("No settings match") {
		t.Fatalf("empty search texts: %q", tt.Texts())
	}
	a.settingsQuery = ""
	tt.Frame()
	tt.Frame()
	if a.settingsFound != nil || a.settingsHits != nil {
		t.Fatal("clearing the query kept the results")
	}
}

func TestSettingsTogglesWriteTheSettingBack(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settingsCat = settingsEditor
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	before := a.HideLineNumbers
	if err := tt.Click("Line numbers toggle"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.HideLineNumbers == before {
		t.Fatalf("Line numbers did not change: %v", a.HideLineNumbers)
	}
	before = a.NoActiveLine
	if err := tt.Click("Highlight active line toggle"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.NoActiveLine == before {
		t.Fatalf("Highlight active line did not change: %v", a.NoActiveLine)
	}
	if a.EditorSoftWrap {
		t.Fatal("soft wrap started on")
	}
	if err := tt.Click("Soft wrap code toggle"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.EditorSoftWrap {
		t.Fatal("Soft wrap code did not turn on")
	}
	a.settingsCat = settingsGit
	tt.Frame()
	if a.NoGitAutoRefresh {
		t.Fatal("git refresh started off")
	}
	if err := tt.Click("Refresh when the window returns toggle"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.NoGitAutoRefresh {
		t.Fatal("Refresh when the window returns did not turn off")
	}
}

func TestSettingsShortcutRecording(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settingsCat = settingsKeyboard
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	if err := tt.Click("Shortcut for Save"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Key(ui.Cmd|ui.Shift, ui.KeyS)
	tt.Frame()
	wantAccel, wantKeys := "Ctrl+Shift+S", "Ctrl+Shift+S"
	if runtime.GOOS == "darwin" {
		wantAccel, wantKeys = "CmdOrCtrl+Shift+S", "⇧⌘S"
	}
	if got := a.commandAccel(cmdSave); got != wantAccel {
		t.Fatalf("save shortcut: %q", got)
	}
	if a.settingsRecording != "" {
		t.Fatal("recording did not stop")
	}
	if got := a.commandKeys(cmdSave); got != wantKeys {
		t.Fatalf("save shortcut display: %q", got)
	}
	// The default is not an override.
	a.setShortcut(cmdSave.id, cmdSave.accel)
	if _, ok := a.shortcutKeys[cmdSave.id]; ok {
		t.Fatalf("the default was kept as an override: %v", a.shortcutKeys)
	}
}

func TestSettingsShortcutPersistenceAndConflict(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	a.setShortcut(cmdSave.id, "CmdOrCtrl+Alt+S")
	if got := a.commandAccel(cmdSave); got != "CmdOrCtrl+Alt+S" {
		t.Fatalf("save accel: %q", got)
	}
	data, err := os.ReadFile(a.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved settingsFile
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Shortcuts[cmdSave.id] != "CmdOrCtrl+Alt+S" {
		t.Fatalf("saved shortcuts: %v", saved.Shortcuts)
	}
	loaded := &app{workspace: &workspace{}, settingsPath: a.settingsPath}
	loaded.loadSettings()
	if loaded.commandAccel(cmdSave) != "CmdOrCtrl+Alt+S" {
		t.Fatalf("loaded shortcut: %q", loaded.commandAccel(cmdSave))
	}
	// Binding a command to another's keys unbinds that one.
	a.setShortcut(cmdSave.id, cmdPalette.accel)
	if got := a.commandAccel(cmdPalette); got != "" {
		t.Fatalf("palette kept the taken keys: %q", got)
	}
	if !strings.Contains(a.settingsNotice, "Command Palette") {
		t.Fatalf("conflict notice: %q", a.settingsNotice)
	}
	a.shortcutKeys = nil
}

func TestSettingsHostsAddEditRemove(t *testing.T) {
	a := newProjectApp(t)
	a.hostDraftAddr, a.hostDraftName = "me@devbox", "Dev"
	a.addHostChecked()
	if len(a.hosts) != 1 || a.hosts[0].Name != "Dev" || a.hosts[0].Address != "me@devbox" {
		t.Fatalf("hosts after add: %+v", a.hosts)
	}
	a.hostEditAddr, a.hostEditName, a.hostEditAddress = "me@devbox", "Build", "me@buildbox"
	a.saveHostEdit("me@devbox")
	if len(a.hosts) != 1 || a.hosts[0].Name != "Build" || a.hosts[0].Address != "me@buildbox" {
		t.Fatalf("hosts after edit: %+v", a.hosts)
	}
	a.removeHost("me@buildbox")
	if len(a.hosts) != 0 {
		t.Fatalf("hosts after remove: %+v", a.hosts)
	}
}

func TestSettingsHostRemoveButtons(t *testing.T) {
	if !terminalsSupported() {
		t.Skip("hosts need terminals")
	}
	registerFonts()
	a := newProjectApp(t)
	a.hosts = []hostConfig{{Name: "Dev", Address: "me@devbox"}}
	a.settingsCat = settingsHosts
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	if err := tt.Click("Remove server Dev"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Remove this server?") {
		t.Fatalf("confirmation missing: %q", tt.Texts())
	}
	if err := tt.Click("Remove"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.hosts) != 0 {
		t.Fatalf("host survived remove: %+v", a.hosts)
	}
}

func TestSettingsSearchShortcut(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Frame()
	tt.Type("line numbers")
	tt.Frame()
	tt.Frame()
	if a.settingsCat != settingsEditor {
		t.Fatalf("page after Cmd+F and typing: %d", a.settingsCat)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
	if a.settingsQuery != "" {
		t.Fatalf("Escape kept the query: %q", a.settingsQuery)
	}
}

func TestSettingsResetKeepsProjectsAndServers(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.Appearance = 2
	a.EditorFontSize = 20
	a.HideLineNumbers = true
	a.hosts = []hostConfig{{Name: "Dev", Address: "me@devbox"}}
	a.project = "/tmp/project"
	a.resetPreferences()
	if a.Appearance != 0 || a.fontSize() != 12.5 || a.HideLineNumbers {
		t.Fatalf("preferences after reset: appearance=%d size=%v hide=%v", a.Appearance, a.fontSize(), a.HideLineNumbers)
	}
	if len(a.hosts) != 1 || a.project != "/tmp/project" {
		t.Fatalf("reset dropped projects or servers: hosts=%v project=%q", a.hosts, a.project)
	}
}

func TestHostSetUpButtonInstallsAndShowsStatus(t *testing.T) {
	old := provisionRemoteFn
	provisionRemoteFn = func(host string, existing *remoteConn) (*remoteSetup, error) {
		if host != "fedora" {
			t.Fatalf("set up host: %q", host)
		}
		return &remoteSetup{Uploaded: true, Path: "/home/me/.local/share/mojian/mojian-server"}, nil
	}
	t.Cleanup(func() { provisionRemoteFn = old })

	a := newProjectApp(t)
	a.hosts = []hostConfig{{Name: "Fedora", Address: "fedora"}}
	a.settingsCat = settingsHosts
	tt := ui.NewTester(a.settingsWindowView, 900, 1400)
	tt.Frame()
	if !tt.HasText("Set Up Server") {
		t.Fatalf("the host row has no setup button (cat=%d): %q", a.settingsCat, tt.Texts())
	}
	if err := tt.Click("Set Up Server"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.hostReady["fedora"] {
		t.Fatal("the host was not marked ready")
	}
	if !strings.Contains(a.hostSetupMsg, "uploaded") {
		t.Fatalf("setup message: %q", a.hostSetupMsg)
	}
	if !tt.HasText("Server uploaded · /home/me/.local/share/mojian/mojian-server") {
		t.Fatalf("the setup status is not on the page: %q", tt.Texts())
	}
}

func TestSettingsTableRendersEveryGenericRow(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.settingsWindowView, 900, 1600)
	for _, cat := range []int{
		settingsGeneral, settingsAppearance, settingsEditor,
		settingsTerminal, settingsShell, settingsNotifications, settingsGit,
	} {
		a.settingsCat = cat
		tt.Frame()
		for _, row := range settings.Rows {
			if row.Cat != cat {
				continue
			}
			if !tt.HasText(row.Title) {
				t.Fatalf("page %q is missing row %q: %q", settingsCats[cat].Title, row.Title, tt.Texts())
			}
		}
	}
}

func TestSettingsHostsHidesRecentRemoteProjects(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.remoteProjects = []remoteProject{{Host: "buildbox", Path: "/srv/app"}}
	a.settingsCat = settingsHosts
	tt := ui.NewTester(a.settingsWindowView, 900, 900)
	tt.Frame()
	if tt.HasText("RECENT REMOTE PROJECTS") {
		t.Fatalf("the hosts page still lists recent remote projects: %q", tt.Texts())
	}
}
