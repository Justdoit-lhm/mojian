package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func makeVault(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProjectSwitch(t *testing.T) {
	root := t.TempDir()
	alpha := filepath.Join(root, "Alpha")
	beta := filepath.Join(root, "Beta")
	makeVault(t, alpha, map[string]string{"alpha.md": "# Alpha\n"})
	makeVault(t, beta, map[string]string{"beta.py": "print('beta')\n"})

	data := t.TempDir()
	a := &app{
		dataDir:      data,
		settingsPath: filepath.Join(data, "settings.json"),
		treeOpen:     true,
	}
	s, err := openStoreAt(alpha)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.dir)
	a.rememberProject(s.dir)
	a.focusFirstNote()
	a.saveLayout()
	a.saveSettings()
	alphaLayout := a.layoutPath

	a.openProject(beta)
	if a.store.dir != beta {
		t.Fatalf("store dir: %q", a.store.dir)
	}
	if a.currentNote() == nil || a.currentNote().Rel != "beta.py" {
		t.Fatalf("current note: %+v", a.currentNote())
	}
	if a.layoutPath == alphaLayout {
		t.Fatal("projects share a layout path")
	}
	if got := a.projectName(); got != "Beta" {
		t.Errorf("project name: %q", got)
	}
	if len(a.projects) < 2 || a.projects[0] != beta {
		t.Errorf("recent projects: %q", a.projects)
	}

	var saved settingsFile
	dataJSON, err := os.ReadFile(a.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(dataJSON, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Project != beta {
		t.Errorf("saved project: %q", saved.Project)
	}

	// Switching back restores the first project and its layout.
	a.openProject(alpha)
	if a.store.dir != alpha {
		t.Fatalf("store dir: %q", a.store.dir)
	}
	if len(a.tabs) != 1 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	if a.currentNote() == nil || a.currentNote().Rel != "alpha.md" {
		t.Fatalf("restored note: %+v", a.currentNote())
	}
	if a.projects[0] != alpha || len(a.projects) != 2 {
		t.Errorf("recent projects: %q", a.projects)
	}
}

func TestProjectChip(t *testing.T) {
	registerFonts()
	dir := filepath.Join(t.TempDir(), "Quiet")
	makeVault(t, dir, map[string]string{"note.md": "# A note\n", "app.js": "console.log(1);\n"})
	a := &app{dataDir: t.TempDir(), treeOpen: true}
	s, err := openStoreAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	a.layoutPath = a.projectLayoutPath(s.dir)
	a.rememberProject(s.dir)
	a.focusFirstNote()
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()

	if !tt.HasText("Quiet") {
		t.Fatalf("texts: %q", tt.Texts())
	}
	if tt.HasText("2 files") || tt.HasText("1 file") {
		t.Fatal("the project chip still shows a file count")
	}
	if err := tt.Click("Switch project: Quiet"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Projects") {
		t.Fatalf("no project popover: %q", tt.Texts())
	}
	if !tt.HasText("Open Folder…") {
		t.Fatal("the project popover has no Open Folder")
	}
}
