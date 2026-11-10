package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mojian/internal/remote"
	"mojian/internal/session"
	"mojian/internal/settings"
	"mojian/internal/shellcmd"
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
)

func TestSettingsSearchFindsRowsAndSwitchesPages(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	a.settings.Query = "line numbers"
	tt.Frame()
	tt.Frame()
	if a.settings.Cat != settingsEditor {
		t.Fatalf("page after searching line numbers: %d", a.settings.Cat)
	}
	if !a.settings.Hits["Line numbers"] {
		t.Fatalf("the row was not found: %v", a.settings.Hits)
	}
	if !tt.HasText("Line numbers") {
		t.Fatalf("editor texts: %q", tt.Texts())
	}
	a.settings.Query = "no such setting"
	tt.Frame()
	tt.Frame()
	if !tt.HasText("No settings match") {
		t.Fatalf("empty search texts: %q", tt.Texts())
	}
	a.settings.Query = ""
	tt.Frame()
	tt.Frame()
	if a.settings.Found != nil || a.settings.Hits != nil {
		t.Fatal("clearing the query kept the results")
	}
}

func TestSettingsTogglesWriteTheSettingBack(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settings.Cat = settingsEditor
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
	a.settings.Cat = settingsGeneral
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
	a.settings.Cat = settingsKeyboard
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
	if a.settings.Recording != "" {
		t.Fatal("recording did not stop")
	}
	if got := a.commandKeys(cmdSave); got != wantKeys {
		t.Fatalf("save shortcut display: %q", got)
	}
	// The default is not an override.
	a.setShortcut(cmdSave.ID, cmdSave.Default())
	if _, ok := a.shortcutKeys[cmdSave.ID]; ok {
		t.Fatalf("the default was kept as an override: %v", a.shortcutKeys)
	}
}

func TestShortcutDefaultsParseAndDoNotCollide(t *testing.T) {
	seen := map[string]string{}
	for _, c := range paletteCommands {
		accel := c.Default()
		if accel == "" {
			continue
		}
		mods, key, ok := parseAccel(accel)
		if !ok {
			t.Fatalf("%s: %q does not parse", c.ID, accel)
		}
		sig := fmt.Sprintf("%d/%d", mods, key)
		if other, ok := seen[sig]; ok {
			t.Fatalf("%s and %s both use %s", other, c.ID, accel)
		}
		seen[sig] = c.ID
	}
}

func TestSettingsShortcutPersistenceAndConflict(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	a.setShortcut(cmdSave.ID, "CmdOrCtrl+Alt+S")
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
	if saved.Shortcuts[cmdSave.ID] != "CmdOrCtrl+Alt+S" {
		t.Fatalf("saved shortcuts: %v", saved.Shortcuts)
	}
	loaded := &app{workspace: &workspace{}, settingsPath: a.settingsPath}
	loaded.loadSettings()
	if loaded.commandAccel(cmdSave) != "CmdOrCtrl+Alt+S" {
		t.Fatalf("loaded shortcut: %q", loaded.commandAccel(cmdSave))
	}
	// Binding a command to another's keys unbinds that one.
	a.setShortcut(cmdSave.ID, cmdPalette.Default())
	if got := a.commandAccel(cmdPalette); got != "" {
		t.Fatalf("palette kept the taken keys: %q", got)
	}
	if !strings.Contains(a.settings.Notice, "Command Palette") {
		t.Fatalf("conflict notice: %q", a.settings.Notice)
	}
	a.shortcutKeys = nil
}

func TestSettingsHostsAddEditRemove(t *testing.T) {
	if !remoteSupported() {
		t.Skip("remote projects are unavailable")
	}
	a := newProjectApp(t)
	a.hostDraftAddr, a.hostDraftName = "me@devbox", "Dev"
	a.addHostChecked()
	if len(a.hosts()) != 1 || a.hosts()[0].Name != "Dev" || a.hosts()[0].Address != "me@devbox" {
		t.Fatalf("hosts after add: %+v", a.hosts())
	}
	a.hostEditAddr, a.hostEditName, a.hostEditAddress = "me@devbox", "Build", "me@buildbox"
	a.saveHostEdit("me@devbox")
	if len(a.hosts()) != 1 || a.hosts()[0].Name != "Build" || a.hosts()[0].Address != "me@buildbox" {
		t.Fatalf("hosts after edit: %+v", a.hosts())
	}
	a.removeHost("me@buildbox")
	if len(a.hosts()) != 0 {
		t.Fatalf("hosts after remove: %+v", a.hosts())
	}
}

func TestSettingsHostRemoveButtons(t *testing.T) {
	if !remoteSupported() {
		t.Skip("remote projects are unavailable")
	}
	registerFonts()
	a := newProjectApp(t)
	a.hostService().SetHosts([]hostConfig{{Name: "Dev", Address: "me@devbox"}})
	a.settings.Cat = settingsHosts
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
	if len(a.hosts()) != 0 {
		t.Fatalf("host survived remove: %+v", a.hosts())
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
	if a.settings.Cat != settingsEditor {
		t.Fatalf("page after Cmd+F and typing: %d", a.settings.Cat)
	}
	if err := tt.Click("Clear"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if a.settings.Query != "" {
		t.Fatalf("the clear button kept the query: %q", a.settings.Query)
	}
	tt.Type("line numbers")
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
	if a.settings.Query != "" {
		t.Fatalf("Escape kept the query: %q", a.settings.Query)
	}
}

func TestSettingsSearchFindsTheme(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	a.settings.Query = "meadow"
	tt.Frame()
	tt.Frame()
	if !a.settings.Found[settingsAppearance] {
		t.Fatalf("searching meadow did not find the theme row: %q", tt.Texts())
	}
	if a.settings.Cat != settingsAppearance {
		t.Fatalf("page after searching meadow: %d", a.settings.Cat)
	}
}

func TestSettingsResetKeepsProjectsAndServers(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.Appearance = 2
	a.EditorFontSize = 20
	a.HideLineNumbers = true
	a.Theme = "dusk"
	a.hostService().SetHosts([]hostConfig{{Name: "Dev", Address: "me@devbox"}})
	a.project = "/tmp/project"
	a.resetPreferences()
	if a.Appearance != 0 || a.fontSize() != 12.5 || a.HideLineNumbers {
		t.Fatalf("preferences after reset: appearance=%d size=%v hide=%v", a.Appearance, a.fontSize(), a.HideLineNumbers)
	}
	if a.Theme != theme.DefaultID {
		t.Fatalf("theme after reset: %q", a.Theme)
	}
	if len(a.hosts()) != 1 || a.project != "/tmp/project" {
		t.Fatalf("reset dropped projects or servers: hosts=%v project=%q", a.hosts(), a.project)
	}
}

func TestHostSetUpButtonInstallsAndShowsStatus(t *testing.T) {
	old := provisionRemoteFn
	provisionRemoteFn = func(host string, existing *remote.Conn) (*remote.Setup, error) {
		if host != "fedora" {
			t.Fatalf("set up host: %q", host)
		}
		return &remote.Setup{Uploaded: true, Path: "/home/me/.local/share/mojian/mojian-server"}, nil
	}
	t.Cleanup(func() { provisionRemoteFn = old })

	a := newProjectApp(t)
	a.hostService().SetHosts([]hostConfig{{Name: "Fedora", Address: "fedora"}})
	a.settings.Cat = settingsHosts
	tt := ui.NewTester(a.settingsWindowView, 900, 1400)
	tt.Frame()
	if !tt.HasText("Set Up Server") {
		t.Fatalf("the host row has no setup button (cat=%d): %q", a.settings.Cat, tt.Texts())
	}
	if err := tt.Click("Set Up Server"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.hostStateFor("fedora").ready {
		t.Fatal("the host was not marked ready")
	}
	if !strings.Contains(a.hostSetupMsg, "uploaded") {
		t.Fatalf("setup message: %q", a.hostSetupMsg)
	}
	if !tt.HasText("Server uploaded · /home/me/.local/share/mojian/mojian-server") {
		t.Fatalf("the setup status is not on the page: %q", tt.Texts())
	}
}

func TestHostStopServerButton(t *testing.T) {
	oldProvision, oldStop := provisionRemoteFn, stopRemoteFn
	provisionRemoteFn = func(host string, existing *remote.Conn) (*remote.Setup, error) {
		return &remote.Setup{Path: "/home/me/.local/share/mojian/mojian-server"}, nil
	}
	stopped := ""
	stopRemoteFn = func(host string) error {
		stopped = host
		return nil
	}
	t.Cleanup(func() { provisionRemoteFn, stopRemoteFn = oldProvision, oldStop })

	a := newProjectApp(t)
	a.hostService().SetHosts([]hostConfig{{Name: "Fedora", Address: "fedora"}})
	a.settings.Cat = settingsHosts
	tt := ui.NewTester(a.settingsWindowView, 900, 1400)
	tt.Frame()
	if err := tt.Click("Set Up Server"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Stop Server") {
		t.Fatalf("a running server has no stop button: %q", tt.Texts())
	}
	if err := tt.Click("Stop Server"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Stop this server? Its terminals and the processes in them end.") {
		t.Fatalf("stopping has no confirmation: %q", tt.Texts())
	}
	if err := tt.Click("Stop"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if stopped != "fedora" {
		t.Fatalf("stopped host: %q", stopped)
	}
	if a.hostStateFor("fedora").ready {
		t.Fatal("the stopped server stayed ready")
	}
	if !a.hostStateFor("fedora").stopped {
		t.Fatal("the stopped server is not remembered as stopped")
	}
	if !tt.HasText("Start Server") {
		t.Fatalf("a stopped server has no start button: %q", tt.Texts())
	}
	if !tt.HasText("Server stopped") {
		t.Fatalf("the stop status is not on the page: %q", tt.Texts())
	}
}

func TestRunningServerShowsStopAfterRestart(t *testing.T) {
	old := hostServerRunningFn
	hostServerRunningFn = func(ctx context.Context, host string) (bool, error) {
		if host != "fedora" {
			t.Fatalf("status host: %q", host)
		}
		return true, nil
	}
	t.Cleanup(func() { hostServerRunningFn = old })

	registerFonts()
	a := newProjectApp(t)
	a.hostService().SetHosts([]hostConfig{{Name: "Fedora", Address: "fedora"}})
	a.settings.Cat = settingsHosts
	tt := ui.NewTester(a.settingsWindowView, 900, 1400)
	tt.Frame()
	tt.Frame()
	if !a.hostStateFor("fedora").running {
		t.Fatalf("the probe did not find the running server: %+v", a.hostStates)
	}
	if _, ok := tt.Find("Stop Server"); !ok {
		t.Fatalf("a server left running has no stop button: %q", tt.Texts())
	}
	if _, ok := tt.Find("Set Up Server"); ok {
		t.Fatalf("a running server still offers setup: %q", tt.Texts())
	}
}

func TestCheckHostsProbesEveryHostOnLaunch(t *testing.T) {
	old := hostServerRunningFn
	seen := map[string]bool{}
	hostServerRunningFn = func(ctx context.Context, host string) (bool, error) {
		seen[host] = true
		return host == "fedora", nil
	}
	t.Cleanup(func() { hostServerRunningFn = old })

	a := newProjectApp(t)
	a.hostService().SetHosts([]hostConfig{
		{Name: "Fedora", Address: "fedora"},
		{Name: "Dev", Address: "me@devbox"},
	})
	a.checkHosts()
	if !seen["fedora"] || !seen["me@devbox"] {
		t.Fatalf("launch checked %v", seen)
	}
	if !a.hostStateFor("fedora").running || a.hostStateFor("me@devbox").running {
		t.Fatalf("launch status: %+v", a.hostStates)
	}
}

func TestSettingsTableRendersEveryGenericRow(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.settingsWindowView, 900, 1600)
	for _, cat := range []int{settingsGeneral, settingsEditor, settingsTerminal} {
		a.settings.Cat = cat
		tt.Frame()
		for _, row := range settings.Rows {
			if row.Cat != cat {
				continue
			}
			if !tt.HasText(row.Title) {
				t.Fatalf("page %q is missing row %q: %q", settings.Categories[cat].Title, row.Title, tt.Texts())
			}
		}
	}
}

func TestSettingsAppearanceThemes(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.settings.Cat = settingsAppearance
	tt := ui.NewTester(a.settingsWindowView, 900, 720)
	tt.Frame()
	for _, label := range []string{"System", "Light", "Dark"} {
		if !tt.HasText(label) {
			t.Fatalf("the appearance page is missing %q: %q", label, tt.Texts())
		}
	}
	for _, ch := range theme.Choices {
		if !tt.HasText(ch.Name) {
			t.Fatalf("the appearance page is missing the %s theme: %q", ch.ID, tt.Texts())
		}
	}
	if err := tt.Click("Dark"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Appearance != 2 {
		t.Fatalf("appearance after Dark: %d", a.Appearance)
	}
	if err := tt.Click("System"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Appearance != 0 {
		t.Fatalf("appearance after System: %d", a.Appearance)
	}
	if err := tt.Click("Mist theme"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Theme != "mist" {
		t.Fatalf("theme after the mist tile: %q", a.Theme)
	}
	if err := tt.Click("Petal theme"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Theme != theme.DefaultID {
		t.Fatalf("theme after the petal tile: %q", a.Theme)
	}
}

// TestSettingsPagesHaveHeaders checks that every page names itself, which
// is what the content shell draws above the page's own sections.
func TestSettingsPagesHaveHeaders(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	for cat, category := range settings.Categories {
		if !terminalsSupported() && (cat == settingsTerminal || cat == settingsHosts) {
			continue
		}
		a.settings.Cat = cat
		tt.Frame()
		if !tt.HasText(category.Detail) {
			t.Fatalf("%s has no header: %q", category.Title, tt.Texts())
		}
	}
}

// TestSettingsSearchMergedPages checks the pages the shell and
// notifications moved to: one search finds the Editor and Terminal faces
// and leaves Appearance with the theme alone. About also matches, on the
// Fonts row of its SYSTEM card.
func TestSettingsSearchMergedPages(t *testing.T) {
	if !terminalsSupported() {
		t.Skip("terminal page is unavailable")
	}
	registerFonts()
	a := newProjectApp(t)
	a.settings.Cat = settingsAppearance
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	a.settings.Query = "shell"
	tt.Frame()
	tt.Frame()
	if a.settings.Cat != settingsTerminal {
		t.Fatalf("page after searching shell: %d", a.settings.Cat)
	}
	if !tt.HasText("Shell") {
		t.Fatalf("terminal page after searching shell: %q", tt.Texts())
	}
	a.settings.Query = ""
	tt.Frame()
	tt.Frame()
	a.settings.Query = "font"
	tt.Frame()
	tt.Frame()
	if !a.settings.Found[settingsEditor] || !a.settings.Found[settingsTerminal] {
		t.Fatalf("font did not find both faces: %v", a.settings.Found)
	}
	if a.settings.Found[settingsAppearance] {
		t.Fatalf("appearance still has a font row: %v", a.settings.Found)
	}
}

func TestSettingsHostsHidesRecentRemoteProjects(t *testing.T) {
	if !remoteSupported() {
		t.Skip("remote projects are unavailable")
	}
	registerFonts()
	a := newProjectApp(t)
	a.hostService().SetProjects([]remoteProject{{Host: "buildbox", Path: "/srv/app"}})
	a.settings.Cat = settingsHosts
	tt := ui.NewTester(a.settingsWindowView, 900, 900)
	tt.Frame()
	if tt.HasText("RECENT REMOTE PROJECTS") {
		t.Fatalf("the hosts page still lists recent remote projects: %q", tt.Texts())
	}
}

func TestFinishHostSetupKeepsConnection(t *testing.T) {
	if !remoteSupported() {
		t.Skip("remote projects are unavailable")
	}
	startLocalServerForTest(t)
	a := newProjectApp(t)
	client, err := session.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	rc := &remote.Conn{Host: "fedora", Client: client}
	a.hostSetupAddr = "fedora"
	a.finishHostSetup("fedora", &remote.Setup{Home: "/home/me", Conn: rc}, nil)
	if a.remoteConn("fedora") != rc {
		t.Fatal("the setup connection was not pooled")
	}
	if !a.remoteKeep["fedora"] {
		t.Fatal("the setup connection was not kept")
	}
	if a.remoteHomes["fedora"] != "/home/me" {
		t.Fatalf("cached home: %q", a.remoteHomes["fedora"])
	}
}

func TestSettingsShellCommandInstallsAndRemoves(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("zsh/bash only")
	}
	registerFonts()
	a := newProjectApp(t)
	a.ShellCommand = "zsh"
	a.settings.Cat = settingsAdvanced
	tt := ui.NewTester(a.settingsWindowView, 900, 1400)
	tt.Frame()
	if !tt.HasText("Shell command") {
		t.Fatalf("the command line section is missing: %q", tt.Texts())
	}
	if err := tt.Click("Bash"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.ShellCommand != "bash" {
		t.Fatalf("shell after choosing Bash: %q", a.ShellCommand)
	}
	if err := tt.Click("Zsh"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.ShellCommand != "zsh" {
		t.Fatalf("shell after choosing Zsh: %q", a.ShellCommand)
	}
	rc, err := shellcmd.RcFile(shellcmd.Zsh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rc); err == nil {
		t.Fatalf("the test home already has %s", rc)
	}
	if err := tt.Click("Install"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "alias mojian=") {
		t.Fatalf("the alias is missing: %q", data)
	}
	if !tt.HasText("Remove") {
		t.Fatalf("the install button did not become Remove: %q", tt.Texts())
	}
	if !strings.Contains(a.settings.Notice, "Added the mojian command") {
		t.Fatalf("notice after install: %q", a.settings.Notice)
	}
	if err := tt.Click("Remove"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	data, err = os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Mojian command line") {
		t.Fatalf("the block is still there: %q", data)
	}
	if !strings.Contains(a.settings.Notice, "Removed the mojian command") {
		t.Fatalf("notice after remove: %q", a.settings.Notice)
	}
}
