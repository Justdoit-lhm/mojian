package app

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mojian/internal/welcome"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// newProjectApp opens a project with prose and code files, and a tree.
func newProjectApp(t *testing.T) *app {
	t.Helper()
	dir := t.TempDir()
	if err := welcome.WriteFiles(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOJIAN_VAULT", dir)
	s, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	a := &app{workspace: &workspace{store: s, treeOpen: true, openDirs: map[string]bool{"scripts": true}}}
	a.focusFirstNote()
	return a
}

func TestCodeFilesLoad(t *testing.T) {
	a := newProjectApp(t)
	py := a.store.ByRelPath(filepath.Join("scripts", "tasks.py"))
	if py == nil {
		t.Fatal("the Python file was not loaded")
	}
	if py.Title != "tasks.py" {
		t.Errorf("Python title: %q", py.Title)
	}
	if l := editor.LanguageFor(py.Path); l == nil || l.Name != "Python" {
		t.Errorf("Python language: %v", l)
	}
	js := a.store.ByRelPath(filepath.Join("scripts", "app.js"))
	if js == nil || js.Title != "app.js" {
		t.Fatalf("JavaScript file: %+v", js)
	}
	if isMarkdownPath(py.Path) {
		t.Error("a Python file was taken for Markdown")
	}
}

func TestSplitOpensEmptyPaneThenFile(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	py := a.store.ByRelPath(filepath.Join("scripts", "tasks.py"))
	a.openNote(py)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	a.splitFocusedSide(splitRight)
	tt.Frame()
	tb := a.tab()
	panes := tb.panes()
	if len(panes) != 2 {
		t.Fatalf("panes: %d", len(panes))
	}
	if panes[0].Note != py {
		t.Fatal("the split changed the original pane")
	}
	if tb.Focus != panes[1] {
		t.Fatal("the split did not focus the new pane")
	}
	if panes[1].Note != nil || panes[1].Asset != nil || panes[1].Term != nil {
		t.Fatal("the split should start with an empty pane")
	}
	if !tt.HasText("Open a file in this pane") {
		t.Fatalf("empty pane texts: %q", tt.Texts())
	}

	other := a.store.ByRelPath("Reading notes.md")
	a.openNote(other)
	tt.Frame()
	panes = tb.panes()
	if len(panes) != 2 || panes[1].Note != other {
		t.Fatal("the file did not open in the empty pane")
	}
	if tb.Focus != panes[1] {
		t.Fatal("the opened file is not focused")
	}
	if panes[0].state == panes[1].state {
		t.Error("the panes share one caret and scroll state")
	}
	if panes[0].state.Buffer == panes[1].state.Buffer {
		t.Error("different files share one buffer")
	}
}

func TestFocusAfterTabSwitch(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	n1 := a.currentNote()
	a.newTab()
	n2 := a.store.ByRelPath("Reading notes.md")
	a.openNote(n2)
	if a.currentNote() != n2 {
		t.Fatal("the second tab does not show the other file")
	}
	tt.Frame()
	a.showTab(1)
	tt.Frame()
	before := n1.Text()
	tt.Type("X")
	tt.Frame()
	if n1.Text() == before {
		t.Fatal("typing after a tab switch did not reach the editor")
	}
}

func TestCloseTabSelectsNextAndFocuses(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	want := a.currentNote()
	a.newTab()
	a.openNote(a.store.ByRelPath("Reading notes.md"))
	a.closeTab(a.tab())
	if len(a.tabs) != 1 || a.active != 0 {
		t.Fatalf("tabs=%d active=%d", len(a.tabs), a.active)
	}
	if a.currentNote() != want {
		t.Fatal("closing the active tab did not return to the other tab")
	}
	tt.Frame()
	before := want.Text()
	tt.Type("Y")
	tt.Frame()
	if want.Text() == before {
		t.Fatal("typing after closing a tab did not reach the editor")
	}
}

func TestCloseActiveTabPicksNext(t *testing.T) {
	a := newProjectApp(t)
	a.newTab()
	a.newTab()
	if len(a.tabs) != 3 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	a.active = 1
	third := a.tabs[2]
	a.closeTab(a.tabs[1])
	if len(a.tabs) != 2 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	if a.tabs[1] != third {
		t.Fatal("closing a middle tab did not select the next one")
	}
	if a.active != 1 {
		t.Fatalf("active: %d", a.active)
	}
}

func TestEditsWaitForExplicitSave(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	n := a.currentNote()
	if n.Buf == nil {
		t.Fatal("the editor buffer was not made")
	}
	body := "# Saved only on request\n\nBody.\n"
	n.Buf.Set(body)
	if !n.Dirty() {
		t.Fatal("the edited file is not dirty")
	}
	for range 2 {
		tt.Frame()
	}
	data, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == body {
		t.Fatal("editing wrote the file without an explicit save")
	}
	if !n.Dirty() {
		t.Fatal("the edited file is no longer dirty")
	}
	a.saveNow()
	data, err = os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != body {
		t.Fatalf("file after save: %q", data)
	}
	if n.Dirty() {
		t.Error("file still dirty after save")
	}
}

func TestFolderCreation(t *testing.T) {
	a := newProjectApp(t)
	a.treeSel = "Ideas"
	a.startNewFolder()
	if a.treeDraft == nil || a.treeDraft.Kind != draftFolder || a.treeDraft.Parent != "Ideas" {
		t.Fatalf("draft: %+v", a.treeDraft)
	}
	a.treeDraft.Name = "Field Notes"
	a.submitTreeDraft()
	if a.treeDraft != nil {
		t.Fatal("the draft is still open")
	}
	if !a.store.DirExists(filepath.Join("Ideas", "Field Notes")) {
		t.Fatal("the folder was not created")
	}
	if a.treeSel != filepath.Join("Ideas", "Field Notes") {
		t.Errorf("selection: %q", a.treeSel)
	}
	n, err := a.store.NewFileIn(a.treeSel, "note.md", "# Note\n")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(n.Rel) != a.treeSel {
		t.Errorf("new file at %q, want %q", n.Rel, a.treeSel)
	}
}

func TestFolderThroughUI(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.treeSel = ""
	tt := ui.NewTester(a.view, 1100, 700)
	if err := tt.Click("New Folder"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.treeDraft == nil || a.treeDraft.Kind != draftFolder {
		t.Fatalf("clicking New Folder did not start a draft: %+v", a.treeDraft)
	}
	tt.Type("Field Notes")
	tt.Frame()
	if a.treeDraft == nil || a.treeDraft.Name != "Field Notes" {
		t.Fatalf("draft after typing: %+v", a.treeDraft)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.treeDraft != nil {
		t.Fatalf("the draft is still open: %+v", a.treeDraft)
	}
	if !a.store.DirExists("Field Notes") {
		t.Fatalf("the folder was not created (treeSel=%q err=%q)", a.treeSel, a.err)
	}
}

func TestNewFileThroughUI(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.treeSel = "scripts"
	tt := ui.NewTester(a.view, 1100, 700)
	if err := tt.Click("New File"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.treeDraft == nil || a.treeDraft.Kind != draftFile || a.treeDraft.Parent != "scripts" {
		t.Fatalf("draft: %+v", a.treeDraft)
	}
	tt.Type("worker.py")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.treeDraft != nil {
		t.Fatalf("the draft is still open: %+v", a.treeDraft)
	}
	n := a.store.ByRelPath(filepath.Join("scripts", "worker.py"))
	if n == nil {
		t.Fatalf("the file was not created (treeSel=%q err=%q)", a.treeSel, a.err)
	}
	if a.currentNote() != n {
		t.Fatal("the new file did not open")
	}
	if !strings.HasSuffix(n.Title, "worker.py") {
		t.Errorf("title: %q", n.Title)
	}
}

func TestTabStripRevealsActive(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	for i := range 11 {
		a.newTab()
		a.tab().Name = fmt.Sprintf("Workspace %d", i+2)
	}
	tt := ui.NewTester(a.view, 760, 560)
	tt.Frame()
	a.showTab(12)
	a.tab().Name = "Workspace 12"
	a.showTab(12)
	tt.Frame()
	r, ok := tt.Find("Workspace 12")
	if !ok {
		t.Fatal("the active tab has no label")
	}
	if r.W < 40 || r.X < 0 || r.X+r.W > 760 {
		t.Fatalf("the active tab is not revealed: x=%.1f w=%.1f", r.X, r.W)
	}
}

func TestTreeTogglesWithCommand(t *testing.T) {
	a := newProjectApp(t)
	a.treeOpen = false
	a.run(cmdTree)
	if !a.treeOpen {
		t.Fatal("the file tree did not open")
	}
	a.run(cmdTree)
	if a.treeOpen {
		t.Fatal("the file tree did not close")
	}
}

func TestRefreshProject(t *testing.T) {
	a := newProjectApp(t)
	path := filepath.Join(a.store.Dir, "made-in-terminal.py")
	if err := os.WriteFile(path, []byte("print('hi')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if a.store.ByRelPath("made-in-terminal.py") != nil {
		t.Fatal("the file appeared before the refresh")
	}
	a.refreshProject()
	n := a.store.ByRelPath("made-in-terminal.py")
	if n == nil {
		t.Fatal("the refresh did not pick up a file made in a terminal")
	}
	if n.Title != "made-in-terminal.py" {
		t.Errorf("title: %q", n.Title)
	}
}

func TestMenuBuilds(t *testing.T) {
	a := &app{workspace: &workspace{}}
	if m := a.menu(); m == nil {
		t.Fatal("no menu")
	}
}

func TestSettingsPageAndPersistence(t *testing.T) {
	a := newProjectApp(t)
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	if !tt.HasText("General") || !tt.HasText("Appearance") {
		t.Fatalf("settings window texts: %q", tt.Texts())
	}
	if terminalsSupported() && !tt.HasText("Hosts") {
		t.Fatalf("settings window texts: %q", tt.Texts())
	}
	a.EditorFontSize = 15
	a.HideLineNumbers = true
	a.MarkdownPreview = true
	a.TerminalFontSize = 14
	a.saveSettings()
	var saved settingsFile
	data, err := os.ReadFile(a.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.EditorFontSize != 15 || !saved.HideLineNumbers || !saved.MarkdownPreview || saved.TerminalFontSize != 14 {
		t.Fatalf("saved settings: %+v", saved)
	}
}

func TestMarkdownPreviewToggle(t *testing.T) {
	a := newProjectApp(t)
	a.openNote(a.store.ByRelPath("Welcome.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("Preview"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	p := a.tab().Focus
	if !p.Preview {
		t.Fatal("Preview did not turn on")
	}
	if !tt.HasText("Welcome to Mojian") {
		t.Fatalf("rendered Markdown texts: %q", tt.Texts())
	}
	if err := tt.Click("Edit"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if p.Preview {
		t.Fatal("Edit did not return to the source")
	}
}

func TestAssetsAppearAndOpen(t *testing.T) {
	a := newProjectApp(t)
	img := image.NewRGBA(image.Rect(0, 0, 12, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 12; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 20), uint8(y * 20), 180, 255})
		}
	}
	path := filepath.Join(a.store.Dir, "cover.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := a.store.Refresh(); err != nil {
		t.Fatal(err)
	}
	asset := a.store.AssetByRel("cover.png")
	if asset == nil {
		t.Fatal("the image was not indexed")
	}
	a.openAsset(asset)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if p := a.tab().Focus; p == nil || p.Asset != asset {
		t.Fatalf("focused pane does not show the image: %+v", p)
	}
	if !tt.HasText("cover.png") {
		t.Fatalf("asset pane texts: %q", tt.Texts())
	}
}

func TestSettingsAppearanceAndEditor(t *testing.T) {
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	settings := ui.NewTester(a.settingsWindowView, 900, 640)
	settings.Frame()
	if err := settings.Click("Appearance"); err != nil {
		t.Fatal(err)
	}
	settings.Frame()
	if err := settings.Click("Dark"); err != nil {
		t.Fatal(err)
	}
	settings.Frame()
	if a.Appearance != 2 {
		t.Fatalf("appearance after Dark: %d", a.Appearance)
	}
	a.EditorFontSize = 16
	a.HideLineNumbers = true
	a.openNote(a.store.ByRelPath("scripts/tasks.py"))
	tt.Frame()
	p := a.tab().Focus
	if p.state == nil {
		t.Fatal("the editor state was not built")
	}
	if got := p.state.Font.Size; got != 16 {
		t.Fatalf("editor font size: %v", got)
	}
	if p.state.LineNumbers != editor.LineNumbersOff {
		t.Fatalf("line numbers: %v", p.state.LineNumbers)
	}
}

func TestSettingsCategoriesAndFontFamily(t *testing.T) {
	a := newProjectApp(t)
	settings := ui.NewTester(a.settingsWindowView, 900, 640)
	settings.Frame()
	if terminalsSupported() {
		if err := settings.Click("Terminal"); err != nil {
			t.Fatal(err)
		}
		settings.Frame()
		if a.settings.Cat != settingsTerminal {
			t.Fatalf("category after Terminal: %d", a.settings.Cat)
		}
		if !settings.HasText("Option as Alt") {
			t.Fatalf("terminal settings texts: %q", settings.Texts())
		}
	}
	if err := settings.Click("Editor"); err != nil {
		t.Fatal(err)
	}
	settings.Frame()
	a.EditorFontFamily = "Menlo"
	a.EditorFontSize = 15
	a.EditorLineHeight = 1.7
	a.EditorTabSize = 2
	a.NoActiveLine = true
	a.openNote(a.store.ByRelPath("scripts/tasks.py"))
	main := ui.NewTester(a.view, 1100, 700)
	main.Frame()
	p := a.tab().Focus
	if p.state == nil {
		t.Fatal("the editor state was not built")
	}
	if !strings.Contains(p.state.Font.Family, "Menlo") || p.state.Font.Size != 15 {
		t.Fatalf("editor font: %+v", p.state.Font)
	}
	if p.state.LineHeight != 1.7 || p.state.TabSize != 2 || !p.state.NoActiveLine {
		t.Fatalf("editor options: line=%v tab=%d active=%v", p.state.LineHeight, p.state.TabSize, p.state.NoActiveLine)
	}
}

func TestFileTreeButtonSitsAtTheFarLeft(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tree, ok := tt.Find("File Tree")
	if !ok {
		t.Fatal("no File Tree button")
	}
	chip, ok := tt.Find("Switch server: Local")
	if !ok {
		chip, ok = tt.Find("Switch project: " + a.projectName())
	}
	if !ok {
		t.Fatal("no header chip")
	}
	if tree.X > chip.X {
		t.Fatalf("File Tree x=%.1f is not left of the header chip x=%.1f", tree.X, chip.X)
	}
}

func TestMiddleClickClosesTab(t *testing.T) {
	a := newProjectApp(t)
	first := a.tab()
	a.newTab()
	if len(a.tabs) != 2 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	ev := ui.InputEvent{Kind: ui.InputPointerDown, Button: 2}
	if !a.tabMiddleClick(first, ev) {
		t.Fatal("middle click was not taken")
	}
	a.runPending()
	if len(a.tabs) != 1 || a.tabs[0] == first {
		t.Fatalf("middle click did not close the tab: %d tabs", len(a.tabs))
	}
	if a.tabMiddleClick(a.tabs[0], ui.InputEvent{Kind: ui.InputPointerDown}) {
		t.Fatal("a primary click closed the tab")
	}
}

func TestTabCloseButtonIsAlwaysVisible(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if len(a.tabs) != 1 {
		t.Fatalf("tabs: %d", len(a.tabs))
	}
	if !tt.HasText("Close Tab") {
		t.Fatalf("the only tab shows no close button: %q", tt.Texts())
	}
	name, detail := a.tab().label()
	r, ok := tt.Find(name + " " + detail)
	if !ok {
		t.Fatalf("tab label %q not found", name)
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if !tt.HasText("Close Tab") {
		t.Fatalf("the close button disappeared on hover: %q", tt.Texts())
	}
	if err := tt.Click("Close Tab"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if len(a.tabs) != 0 {
		t.Fatalf("the only tab did not close: %d left", len(a.tabs))
	}
}

func TestOpeningTreeReusesRecentListing(t *testing.T) {
	a := newProjectApp(t)
	a.refreshProject()
	if err := os.WriteFile(filepath.Join(a.store.Dir, "later.py"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.treeOpen = false
	a.toggleTree()
	if a.store.ByRelPath("later.py") != nil {
		t.Fatal("opening the tree forced a refresh inside the throttle window")
	}
	a.refreshProject()
	if a.store.ByRelPath("later.py") == nil {
		t.Fatal("the forced refresh did not pick up the new file")
	}
}
