// Mojian is a quiet text editor in MyGo's native UI. Tabs hold
// workspaces, workspaces hold editors, and every editor shows a file's
// source with syntax highlighting.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go test ./...       the model, the store and the view, without a window
package app

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	gitpage "mojian/internal/git"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type app struct {
	win   *mygo.Window
	store *store

	dataDir      string
	layoutPath   string
	settingsPath string

	project       string
	projects      []string
	projectOpen   bool
	projectPicker bool
	pickerPath    string
	pickerInput   string
	pickerEntries []pickerEntry
	pickerSel     int
	pickerErr     string

	tabs   []*tab
	active int

	focusReq *pane

	splitPick  bool
	splitDown  bool
	splitFrom  *pane
	splitTab   *tab
	splitQuery string
	splitSel   int

	treeOpen        bool
	treeQuery       string
	treeFocusSearch bool
	treeSel         string
	treeReveal      string
	openDirs        map[string]bool
	treeDraft       *treeDraft
	draftSeq        int
	revealTab       bool
	tabScroll       ui.ScrollState

	fileClip []string
	fileCut  bool

	shellStarter func() (*termTab, error) // tests replace this

	git gitpage.Page

	paletteOpen  bool
	paletteQuery string
	paletteSel   int

	rename      *tab
	renameText  string
	renameFocus bool

	appearance      int // 0 follows the system, 1 light, 2 dark
	appearanceItems [3]*mygo.MenuItem

	settingsWin *mygo.Window
	settingsCat int
	fontPick    bool
	fontPickFor int
	fontTarget  int
	fontSearch  string
	fontsOnce   sync.Once
	fonts       []ui.FontFamily

	editorFontSize      float64 // 0 means the default 12.5
	editorFontFamily    string
	editorLineHeight    float64 // 0 means the default 1.55
	editorTabSize       int     // 0 means 4
	editorSoftWrap      bool
	hideLineNumbers     bool
	markdownPreview     bool
	noActiveLine        bool
	noIndentGuides      bool
	noAutoClose         bool
	noScrollPastEnd     bool
	terminalFontSize    float64 // 0 means the default 12.5
	terminalFontFamily  string
	terminalLineHeight  float64 // 0 means 1
	terminalOptionAsAlt bool
	terminalCopySelect  bool
	noCursorBlink       bool
	terminalScrollback  int // MB, 0 means 10
	noRestoreLayout     bool
	noGitAutoRefresh    bool

	err   string
	title string

	layoutDirty bool
	lastChange  time.Time
	layoutTimer *time.Timer
	loaded      bool
	pending     []func()

	edDark  bool
	edTheme *editor.Theme
}

func Run() {
	registerFonts()
	a := &app{
		editorFontSize: 12.5, editorLineHeight: 1.55, editorTabSize: 4,
		terminalFontSize: 12.5, terminalLineHeight: 1, terminalScrollback: 10,
	}
	mygo.App.SetMenu(a.menu())
	mygo.App.WhenReady(a.open)
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows && a.win == nil {
			a.open()
		}
	})
	mygo.App.OnWindowAllClosed(func() {
		if runtime.GOOS != "darwin" {
			mygo.App.Quit()
		}
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.saveNow()
		a.closeTerminals()
		if a.settingsWin != nil {
			a.settingsWin.Close()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// open opens the window, with the tabs of the last launch.
func (a *app) open() {
	if !a.loaded {
		a.loaded = true
		if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
			a.dataDir = dir
			a.settingsPath = filepath.Join(dir, "settings.json")
		}
		a.loadSettings()
		firstProject := a.project == ""
		s, err := a.initialStore()
		if err != nil {
			log.Print(err)
			mygo.Dialog.Message(mygo.MessageOptions{
				Type:    mygo.MessageError,
				Title:   "Mojian could not open the project",
				Message: err.Error(),
			})
			mygo.App.Quit()
			return
		}
		a.store = s
		a.treeOpen = true
		a.layoutPath = a.projectLayoutPath(s.dir)
		if firstProject && !s.memory {
			a.migrateLayout(s.dir)
		}
		a.rememberProject(s.dir)
		a.saveSettings()
		if a.noRestoreLayout || !a.restoreLayout() {
			a.focusFirstNote()
		} else if t := a.tab(); t != nil {
			a.focusReq = t.editorPane()
		}
		if n := a.currentNote(); n != nil {
			a.revealInTree(n)
		}
		a.revealTab = true
		a.saveLayout()
	}
	a.win = a.createWindow("main")
	a.updateTitle()
}

// createWindow opens a window for this app instance.
func (a *app) createWindow(stateKey string) *mygo.Window {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:                "Mojian",
		Width:                1120,
		Height:               720,
		MinWidth:             760,
		MinHeight:            460,
		StateKey:             stateKey,
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 16, Y: 15},
		BackgroundColor:      "light-dark(#efe1e6, #231e27)",
		Content:              ui.View(a.view),
	})
	a.win = win
	win.OnFocus(func() {
		a.refreshProject()
	})
	win.OnClosed(func() {
		a.windowClosed()
	})
	return win
}

// windowClosed clears the window state, keeping the tabs for a reopen.
func (a *app) windowClosed() {
	if a.settingsWin != nil {
		a.settingsWin.Close()
	}
	a.saveNow()
	a.closeTerminals()
	if a.layoutTimer != nil {
		a.layoutTimer.Stop()
		a.layoutTimer = nil
	}
	a.win = nil
	a.focusReq = nil
	a.paletteOpen = false
	a.projectOpen = false
	a.treeDraft = nil
	a.rename = nil
}

// settingsFile is what the app keeps between launches.
type settingsFile struct {
	Appearance          int      `json:"appearance"`
	EditorFontFamily    string   `json:"editorFontFamily,omitempty"`
	EditorFontSize      float64  `json:"editorFontSize,omitempty"`
	EditorLineHeight    float64  `json:"editorLineHeight,omitempty"`
	EditorTabSize       int      `json:"editorTabSize,omitempty"`
	EditorSoftWrap      bool     `json:"editorSoftWrap,omitempty"`
	HideLineNumbers     bool     `json:"hideLineNumbers,omitempty"`
	MarkdownPreview     bool     `json:"markdownPreview,omitempty"`
	NoActiveLine        bool     `json:"noActiveLine,omitempty"`
	NoIndentGuides      bool     `json:"noIndentGuides,omitempty"`
	NoAutoClose         bool     `json:"noAutoClose,omitempty"`
	NoScrollPastEnd     bool     `json:"noScrollPastEnd,omitempty"`
	TerminalFontFamily  string   `json:"terminalFontFamily,omitempty"`
	TerminalFontSize    float64  `json:"terminalFontSize,omitempty"`
	TerminalLineHeight  float64  `json:"terminalLineHeight,omitempty"`
	TerminalOptionAsAlt bool     `json:"terminalOptionAsAlt,omitempty"`
	TerminalCopySelect  bool     `json:"terminalCopySelect,omitempty"`
	NoCursorBlink       bool     `json:"noCursorBlink,omitempty"`
	TerminalScrollback  int      `json:"terminalScrollback,omitempty"`
	NoRestoreLayout     bool     `json:"noRestoreLayout,omitempty"`
	NoGitAutoRefresh    bool     `json:"noGitAutoRefresh,omitempty"`
	Project             string   `json:"project,omitempty"`
	Projects            []string `json:"projects,omitempty"`
}

// loadSettings reads the appearance and the project the app was left in.
func (a *app) loadSettings() {
	if a.settingsPath == "" {
		return
	}
	data, err := os.ReadFile(a.settingsPath)
	if err != nil {
		return
	}
	var s settingsFile
	if json.Unmarshal(data, &s) != nil {
		return
	}
	if s.Appearance >= 0 && s.Appearance <= 2 {
		a.appearance = s.Appearance
		a.syncAppearance()
	}
	if s.EditorFontSize >= 8 && s.EditorFontSize <= 32 {
		a.editorFontSize = s.EditorFontSize
	}
	if s.EditorLineHeight >= 1 && s.EditorLineHeight <= 2.4 {
		a.editorLineHeight = s.EditorLineHeight
	}
	if s.EditorTabSize >= 1 && s.EditorTabSize <= 8 {
		a.editorTabSize = s.EditorTabSize
	}
	if s.TerminalFontSize >= 8 && s.TerminalFontSize <= 32 {
		a.terminalFontSize = s.TerminalFontSize
	}
	if s.TerminalLineHeight >= 0.8 && s.TerminalLineHeight <= 2 {
		a.terminalLineHeight = s.TerminalLineHeight
	}
	if s.TerminalScrollback >= 1 && s.TerminalScrollback <= 1000 {
		a.terminalScrollback = s.TerminalScrollback
	}
	a.editorFontFamily = s.EditorFontFamily
	a.editorSoftWrap = s.EditorSoftWrap
	a.hideLineNumbers = s.HideLineNumbers
	a.markdownPreview = s.MarkdownPreview
	a.noActiveLine = s.NoActiveLine
	a.noIndentGuides = s.NoIndentGuides
	a.noAutoClose = s.NoAutoClose
	a.noScrollPastEnd = s.NoScrollPastEnd
	a.terminalFontFamily = s.TerminalFontFamily
	a.terminalOptionAsAlt = s.TerminalOptionAsAlt
	a.terminalCopySelect = s.TerminalCopySelect
	a.noCursorBlink = s.NoCursorBlink
	a.noRestoreLayout = s.NoRestoreLayout
	a.noGitAutoRefresh = s.NoGitAutoRefresh
	a.project = s.Project
	a.projects = s.Projects
}

func (a *app) saveSettings() {
	if a.settingsPath == "" {
		return
	}
	project := a.project
	if a.store != nil && !a.store.memory {
		project = a.store.dir
	}
	data, _ := json.MarshalIndent(settingsFile{
		Appearance:          a.appearance,
		EditorFontFamily:    a.editorFontFamily,
		EditorFontSize:      a.fontSize(),
		EditorLineHeight:    a.lineHeight(),
		EditorTabSize:       a.tabSize(),
		EditorSoftWrap:      a.editorSoftWrap,
		HideLineNumbers:     a.hideLineNumbers,
		MarkdownPreview:     a.markdownPreview,
		NoActiveLine:        a.noActiveLine,
		NoIndentGuides:      a.noIndentGuides,
		NoAutoClose:         a.noAutoClose,
		NoScrollPastEnd:     a.noScrollPastEnd,
		TerminalFontFamily:  a.terminalFontFamily,
		TerminalFontSize:    a.termFontSize(),
		TerminalLineHeight:  a.termLineHeight(),
		TerminalOptionAsAlt: a.terminalOptionAsAlt,
		TerminalCopySelect:  a.terminalCopySelect,
		NoCursorBlink:       a.noCursorBlink,
		TerminalScrollback:  a.termScrollback(),
		NoRestoreLayout:     a.noRestoreLayout,
		NoGitAutoRefresh:    a.noGitAutoRefresh,
		Project:             project,
		Projects:            a.projects,
	}, "", "  ")
	_ = os.MkdirAll(filepath.Dir(a.settingsPath), 0o755)
	_ = os.WriteFile(a.settingsPath, data, 0o644)
}

// fontSize is the editor's font size, with the default when unset.
func (a *app) fontSize() float64 {
	if a.editorFontSize <= 0 {
		return 12.5
	}
	return a.editorFontSize
}

// termFontSize is the terminal's font size, with the default when unset.
func (a *app) termFontSize() float64 {
	if a.terminalFontSize <= 0 {
		return 12.5
	}
	return a.terminalFontSize
}

// lineHeight, tabSize and the terminal's numbers are the effective
// values, with the defaults when unset.
func (a *app) lineHeight() float64 {
	if a.editorLineHeight <= 0 {
		return 1.55
	}
	return a.editorLineHeight
}

func (a *app) tabSize() int {
	if a.editorTabSize <= 0 {
		return 4
	}
	return a.editorTabSize
}

func (a *app) termLineHeight() float64 {
	if a.terminalLineHeight <= 0 {
		return 1
	}
	return a.terminalLineHeight
}

func (a *app) termScrollback() int {
	if a.terminalScrollback <= 0 {
		return 10
	}
	return a.terminalScrollback
}

// openSettingsWindow shows the preferences in a window of their own, or
// brings it forward.
func (a *app) openSettingsWindow() {
	if a.settingsWin != nil {
		a.settingsWin.Focus()
		return
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:                "Settings",
		Width:                900,
		Height:               640,
		MinWidth:             720,
		MinHeight:            500,
		StateKey:             "settings",
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 16, Y: 15},
		BackgroundColor:      "light-dark(#efe1e6, #231e27)",
		Content:              ui.View(a.settingsWindowView),
	})
	a.settingsWin = win
	win.OnClosed(func() { a.settingsWin = nil })
	win.Focus()
}

// invalidate redraws both windows that show app state.
func (a *app) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
	if a.settingsWin != nil {
		a.settingsWin.Invalidate()
	}
}

// initialStore opens $MOJIAN_VAULT, else the project of the last launch,
// else the default folder.
func (a *app) initialStore() (*store, error) {
	if os.Getenv("MOJIAN_VAULT") != "" {
		return openStore()
	}
	if a.project != "" {
		if info, err := os.Stat(a.project); err == nil && info.IsDir() {
			if s, err := openStoreAt(a.project); err == nil {
				return s, nil
			}
		}
	}
	return openStore()
}

// rememberProject moves dir to the front of the recent projects.
func (a *app) rememberProject(dir string) {
	if dir == "" {
		return
	}
	dir = filepath.Clean(dir)
	a.project = dir
	out := []string{dir}
	for _, p := range a.projects {
		p = filepath.Clean(p)
		if p != dir && p != "." && p != "" {
			out = append(out, p)
		}
	}
	if len(out) > 8 {
		out = out[:8]
	}
	a.projects = out
}

// projectLayoutPath keeps a layout for each project, apart from the
// others.
func (a *app) projectLayoutPath(dir string) string {
	if a.dataDir == "" || dir == "" {
		return ""
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(dir))
	return filepath.Join(a.dataDir, "layouts", fmt.Sprintf("%016x.json", h.Sum64()))
}

// migrateLayout copies the one layout of older versions to the first
// project, so its tabs survive the upgrade.
func (a *app) migrateLayout(dir string) {
	if a.layoutPath == "" {
		return
	}
	if _, err := os.Stat(a.layoutPath); err == nil {
		return
	}
	old := filepath.Join(a.dataDir, "layout.json")
	data, err := os.ReadFile(old)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(a.layoutPath), 0o755)
	_ = os.WriteFile(a.layoutPath, data, 0o644)
}

func (a *app) setAppearance(v int) {
	if v < 0 || v > 2 {
		return
	}
	a.appearance = v
	a.syncAppearance()
	a.saveSettings()
	a.invalidate()
}

func (a *app) syncAppearance() {
	for i, it := range a.appearanceItems {
		if it != nil {
			it.SetChecked(i == a.appearance)
		}
	}
}

// saveNow writes every unsaved note, the layout and the settings.
func (a *app) saveNow() {
	if a.store != nil {
		if err := a.store.saveAll(); err != nil {
			log.Print(err)
		}
	}
	if a.layoutDirty {
		a.saveLayout()
		a.layoutDirty = false
	}
	a.saveSettings()
}

// invalidate asks for a frame after a change made outside the view.
// fail shows an error at the bottom of the window.
func (a *app) fail(err error) {
	if err == nil {
		return
	}
	a.err = err.Error()
	a.invalidate()
}

// later runs fn after the frame that asked for it, so a button does not
// change the tree it is part of while the tree builds.
func (a *app) later(fn func()) {
	a.pending = append(a.pending, fn)
	if a.win != nil {
		a.win.Update(func() {})
	}
}

func (a *app) runPending() {
	if len(a.pending) == 0 {
		return
	}
	fns := a.pending
	a.pending = nil
	for _, fn := range fns {
		fn()
	}
}

// changed marks the layout for saving and schedules the frame that
// persists it after the pointer or keyboard settles.
func (a *app) changed() {
	a.layoutDirty = true
	a.lastChange = time.Now()
	a.scheduleLayoutSave()
}

func (a *app) scheduleLayoutSave() {
	win := a.win
	if win == nil {
		return
	}
	// The timer outlives a single frame. Recreate it for a reopened
	// window rather than waking one that is gone.
	if a.layoutTimer == nil {
		a.layoutTimer = time.AfterFunc(700*time.Millisecond, func() {
			win.Update(func() {})
		})
		return
	}
	a.layoutTimer.Reset(700 * time.Millisecond)
}

// persistLayout writes the window layout after a change settles. File
// contents are written only by an explicit save.
func (a *app) persistLayout(c *ui.Context) {
	if !a.layoutDirty {
		return
	}
	if time.Since(a.lastChange) < 550*time.Millisecond {
		return
	}
	a.saveLayout()
	a.layoutDirty = false
}

// tab returns the active workspace.
func (a *app) tab() *tab {
	if a.active < 0 || a.active >= len(a.tabs) {
		return nil
	}
	return a.tabs[a.active]
}

func (a *app) currentNote() *note {
	if t := a.tab(); t != nil && t.Focus != nil {
		return t.Focus.Note
	}
	return nil
}

// focusFirstNote opens the most recent note, in the first tab.
func (a *app) focusFirstNote() {
	if a.store == nil {
		return
	}
	notes := a.store.sorted()
	if len(notes) == 0 {
		for _, f := range a.store.assets {
			a.tabs = []*tab{{ID: nextID("t")}}
			p := newAssetPane(f)
			a.tabs[0].Root = &node{Pane: p}
			a.tabs[0].Focus = p
			a.active = 0
			a.focusReq = p
			a.revealTab = true
			a.revealRel(f.Rel)
			return
		}
		return
	}
	a.tabs = []*tab{newTab(notes[0])}
	if a.tabs[0].Focus != nil {
		a.tabs[0].Focus.Preview = a.markdownPreview && isMarkdownPath(notes[0].Path)
	}
	a.active = 0
	a.focusReq = a.tabs[0].Focus
	a.revealTab = true
	a.revealInTree(notes[0])
}

// newTab adds an empty workspace to the right of the tabs. It does not
// copy the current file; picking or creating a file fills it.
func (a *app) newTab() {
	a.git.Open = false
	t := &tab{ID: nextID("t")}
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = nil
	a.revealTab = true
	a.treeDraft = nil
	a.changed()
}

// openNoteInTab shows n in a tab of its own.
func (a *app) openNoteInTab(n *note) {
	if n == nil {
		return
	}
	a.git.Open = false
	t := newTab(n)
	if t.Focus != nil {
		t.Focus.Preview = a.markdownPreview && isMarkdownPath(n.Path)
	}
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = t.Focus
	a.revealTab = true
	a.treeSel = n.Rel
	a.openPath(n.Rel)
	a.treeReveal = n.Rel
	a.changed()
}

// openNote shows n in the focused pane, or in a new tab when none is
// open.
func (a *app) openNote(n *note) {
	if n == nil {
		return
	}
	a.git.Open = false
	a.treeDraft = nil
	a.treeQuery = ""
	t := a.tab()
	if t == nil {
		t = newTab(n)
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
		a.focusReq = t.Focus
		t.Focus.Preview = a.markdownPreview && isMarkdownPath(n.Path)
		a.revealTab = true
		a.revealRel(n.Rel)
		a.changed()
		return
	}
	p := t.Focus
	if p != nil && p.Term != nil {
		p = t.firstEditorPane()
		if p == nil {
			a.openNoteInTab(n)
			return
		}
	}
	if p == nil {
		p = firstPane(t.Root)
	}
	if p == nil {
		p = newPane(n)
		t.Root = &node{Pane: p}
		t.Focus = p
	}
	if p.Note != n {
		p.state = nil
		p.md = nil
		p.Preview = a.markdownPreview && isMarkdownPath(n.Path)
	}
	releaseAsset(p)
	p.Note = n
	p.Asset = nil
	p.assetView = nil
	t.Focus = p
	a.focusReq = p
	a.revealRel(n.Rel)
	a.changed()
}

// openRel opens a path from the project: a text file in the editor, or a
// picture and PDF in a viewer.
func (a *app) openRel(rel string) {
	if a.store == nil {
		return
	}
	if n := a.store.byRelPath(rel); n != nil {
		a.openNote(n)
		return
	}
	if f := a.store.assetByRel(rel); f != nil {
		a.openAsset(f)
	}
}

// openAssetInTab shows a picture or PDF in a tab of its own.
func (a *app) openAssetInTab(f *assetFile) {
	if f == nil {
		return
	}
	a.git.Open = false
	t := &tab{ID: nextID("t")}
	p := newAssetPane(f)
	t.Root = &node{Pane: p}
	t.Focus = p
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = p
	a.revealTab = true
	a.treeSel = f.Rel
	a.openPath(f.Rel)
	a.treeReveal = f.Rel
	a.changed()
}

// openAsset shows a picture or PDF in the focused pane, or in a new tab
// when none is open.
func (a *app) openAsset(f *assetFile) {
	if f == nil {
		return
	}
	a.git.Open = false
	a.treeDraft = nil
	a.treeQuery = ""
	t := a.tab()
	if t == nil {
		t = &tab{ID: nextID("t")}
		p := newAssetPane(f)
		t.Root = &node{Pane: p}
		t.Focus = p
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
		a.focusReq = p
		a.revealTab = true
		a.revealRel(f.Rel)
		a.changed()
		return
	}
	p := t.Focus
	if p != nil && p.Term != nil {
		p = t.firstEditorPane()
	}
	if p == nil {
		p = firstPane(t.Root)
	}
	if p == nil {
		p = newAssetPane(f)
		t.Root = &node{Pane: p}
		t.Focus = p
	}
	releaseAsset(p)
	p.Note = nil
	p.state = nil
	p.Asset = f
	p.assetView = &assetView{}
	p.Preview = false
	t.Focus = p
	a.focusReq = p
	a.revealRel(f.Rel)
	a.changed()
}

// closeTab closes a workspace.
func (a *app) closeTab(t *tab) {
	i := -1
	for j, x := range a.tabs {
		if x == t {
			i = j
			break
		}
	}
	if i < 0 {
		return
	}
	for _, p := range t.panes() {
		if p.Term != nil && p.Term.Term != nil {
			_ = p.Term.Term.Close()
		}
		releaseAsset(p)
	}
	a.tabs = append(a.tabs[:i], a.tabs[i+1:]...)
	wasActive := a.active == i
	switch {
	case len(a.tabs) == 0:
		a.active = -1
	case i < a.active:
		a.active--
	case wasActive:
		a.active = min(i, len(a.tabs)-1)
	}
	if a.rename == t {
		a.rename = nil
	}
	if next := a.tab(); next != nil {
		a.focusReq = next.editorPane()
		a.revealTab = true
		if next.Focus != nil {
			if next.Focus.Note != nil {
				a.revealRel(next.Focus.Note.Rel)
			} else if next.Focus.Asset != nil {
				a.revealRel(next.Focus.Asset.Rel)
			}
		}
	} else {
		a.focusReq = nil
	}
	a.changed()
}

// closePane closes the pane focused in its tab.
func (a *app) closePane(p *pane) {
	if p == nil {
		return
	}
	for _, t := range a.tabs {
		if !t.contains(p) {
			continue
		}
		if p.Term != nil && p.Term.Term != nil {
			_ = p.Term.Term.Close()
		}
		releaseAsset(p)
		if !t.closePane(p) {
			a.closeTab(t)
			return
		}
		a.focusReq = t.Focus
		a.changed()
		return
	}
}

func (a *app) toggleZoom() {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	if t.Zoom == t.Focus {
		t.Zoom = nil
	} else {
		t.Zoom = t.Focus
	}
	a.focusReq = t.Focus
	a.changed()
}

func (a *app) equalize() {
	if t := a.tab(); t != nil {
		t.equalize()
		a.changed()
	}
}

func (a *app) cycleTab(delta int) {
	if len(a.tabs) == 0 {
		return
	}
	a.git.Open = false
	a.active = (a.active + delta + len(a.tabs)) % len(a.tabs)
	a.revealTab = true
	a.treeDraft = nil
	if t := a.tab(); t != nil {
		a.focusReq = t.editorPane()
		if t.Focus != nil {
			a.revealInTree(t.Focus.Note)
		}
	}
	a.changed()
}

// showTab shows the nth tab, from 1; 9 shows the last.
func (a *app) showTab(n int) {
	if len(a.tabs) == 0 {
		return
	}
	a.git.Open = false
	i := n - 1
	if n >= 9 {
		i = len(a.tabs) - 1
	}
	a.active = min(max(i, 0), len(a.tabs)-1)
	a.revealTab = true
	a.treeDraft = nil
	if t := a.tab(); t != nil {
		a.focusReq = t.editorPane()
		if t.Focus != nil {
			a.revealInTree(t.Focus.Note)
		}
	}
	a.changed()
}

func (a *app) startRename(t *tab) {
	if t == nil {
		return
	}
	for i, x := range a.tabs {
		if x == t {
			a.active = i
			break
		}
	}
	a.rename = t
	a.renameText = t.Name
	a.renameFocus = false
	a.revealTab = true
	a.invalidate()
}

// updateTitle names the window after the focused note.
func (a *app) updateTitle() {
	if a.win == nil {
		return
	}
	title := "Mojian"
	switch {
	case a.git.Open:
		title = "Changes"
	case a.tab() != nil:
		name, _ := a.tab().label()
		title = name
	}
	if title != a.title {
		a.title = title
		a.win.SetTitle(title)
	}
}

// revealVault shows the vault in the file manager.
func (a *app) revealVault() {
	if a.store == nil || a.store.memory {
		return
	}
	if err := mygo.Shell.OpenPath(a.store.dir); err != nil {
		a.fail(err)
	}
}

// menu builds the application menu. Its accelerators are the app's
// shortcuts on every platform.
func (a *app) menu() *mygo.Menu {
	item := func(c *command) *mygo.MenuItem {
		return &mygo.MenuItem{
			Label:       c.title,
			Accelerator: c.accel,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.run(c)
			},
		}
	}
	appearance := func(label string, v int) *mygo.MenuItem {
		it := &mygo.MenuItem{
			Label:   label,
			Type:    mygo.MenuItemRadio,
			Checked: a.appearance == v,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.setAppearance(v)
			},
		}
		a.appearanceItems[v] = it
		return it
	}
	tabItems := []*mygo.MenuItem{item(cmdNextTab), item(cmdPrevTab), mygo.Separator()}
	for i := 1; i <= 9; i++ {
		i := i
		tabItems = append(tabItems, &mygo.MenuItem{
			Label:       "Show Tab " + string(rune('0'+i)),
			Accelerator: "CmdOrCtrl+" + string(rune('0'+i)),
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.showTab(i)
			},
		})
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			item(cmdOpenProject),
			item(cmdNotes),
			mygo.Separator(),
			item(cmdNewNote),
			item(cmdNewFolder),
			item(cmdNewTab),
			item(cmdSave),
			mygo.Separator(),
			item(cmdRenameNode),
			item(cmdTrashNode),
			mygo.Separator(),
			item(cmdCloseTab),
			mygo.Separator(),
			item(cmdTerminal),
			{Label: "Reveal Project", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.revealVault()
			}},
		}},
		{Label: "Edit", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleUndo},
			{Role: mygo.RoleRedo},
			mygo.Separator(),
			{Role: mygo.RoleCut},
			{Role: mygo.RoleCopy},
			{Role: mygo.RolePaste},
			{Role: mygo.RoleSelectAll},
		}},
		{Label: "View", Submenu: []*mygo.MenuItem{
			item(cmdTree),
			item(cmdTerminal),
			item(cmdGit),
			item(cmdPalette),
			mygo.Separator(),
			item(cmdSettings),
			{Label: "Appearance", Submenu: []*mygo.MenuItem{
				appearance("System", 0),
				appearance("Light", 1),
				appearance("Dark", 2),
			}},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Label: "Tabs", Submenu: append(tabItems,
			mygo.Separator(),
			item(cmdRenameTab),
		)},
		{Label: "Pane", Submenu: []*mygo.MenuItem{
			item(cmdSplitRight),
			item(cmdSplitDown),
			mygo.Separator(),
			item(cmdZoom),
			item(cmdEqualize),
			mygo.Separator(),
			item(cmdClosePane),
		}},
		{Role: mygo.RoleWindowMenu},
	})
}

// run runs a command from a menu or the palette.
func (a *app) run(c *command) {
	if c == nil {
		return
	}
	if c == cmdClosePane && a.settingsWin != nil && a.settingsWin.IsFocused() {
		a.settingsWin.Close()
		return
	}
	c.run(a)
	a.updateTitle()
	a.invalidate()
}

// applyTheme gives the frame the appearance the user chose.
func (a *app) applyTheme(c *ui.Context) {
	switch a.appearance {
	case 1:
		c.SetTheme(ui.LightTheme())
	case 2:
		c.SetTheme(ui.DarkTheme())
	}
}

// editorTheme returns the code theme for the current appearance, kept
// between frames because it is built once per appearance.
func (a *app) editorTheme(c *ui.Context) *editor.Theme {
	dark := c.Theme().Dark
	if a.edTheme == nil || a.edDark != dark {
		a.edTheme = editorThemeOf(c.Theme())
		a.edDark = dark
	}
	return a.edTheme
}
