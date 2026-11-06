// Mojian is a quiet text editor in MyGo's native UI. Tabs hold
// workspaces, workspaces hold editors, and every editor shows a file's
// source with syntax highlighting.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go test ./...       the model, the store and the view, without a window
package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

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

	terms      []*termTab
	activeTerm int
	termOpen   bool
	termFocus  bool
	termSeq    int

	gitOpen       bool
	gitLoading    bool
	gitErr        string
	gitRoot       string
	gitPrefix     string
	gitBranch     string
	gitChanges    []gitChange
	gitSel        int
	gitDiff       string
	gitDiffPath   string
	gitDiffErr    string
	gitDiffLoad   bool
	gitSeq        int
	gitDiffSeq    int
	gitDiffScroll ui.ScrollState

	paletteOpen  bool
	paletteQuery string
	paletteSel   int

	rename      *tab
	renameText  string
	renameFocus bool

	appearance      int // 0 follows the system, 1 light, 2 dark
	appearanceItems [3]*mygo.MenuItem

	err   string
	title string

	layoutDirty bool
	lastChange  time.Time
	saveTimer   *time.Timer
	saveTick    atomic.Bool
	loaded      bool
	pending     []func()

	edDark  bool
	edTheme *editor.Theme
}

func main() {
	registerFonts()
	a := &app{}
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
		if !a.restoreLayout() {
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
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:                "Mojian",
		Width:                1120,
		Height:               720,
		MinWidth:             760,
		MinHeight:            460,
		StateKey:             "main",
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 16, Y: 15},
		BackgroundColor:      "light-dark(#efe1e6, #231e27)",
		Content:              ui.View(a.view),
	})
	a.win = win
	win.OnFocus(func() {
		a.refreshProject()
	})
	a.updateTitle()
	win.OnClosed(func() {
		a.saveNow()
		a.closeTerminals()
		if a.saveTimer != nil {
			a.saveTimer.Stop()
			a.saveTimer = nil
		}
		a.win = nil
		a.focusReq = nil
		a.paletteOpen = false
		a.projectOpen = false
		a.treeDraft = nil
		a.rename = nil
	})
}

// settingsFile is what the app keeps between launches.
type settingsFile struct {
	Appearance int      `json:"appearance"`
	Project    string   `json:"project,omitempty"`
	Projects   []string `json:"projects,omitempty"`
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
		Appearance: a.appearance,
		Project:    project,
		Projects:   a.projects,
	}, "", "  ")
	_ = os.MkdirAll(filepath.Dir(a.settingsPath), 0o755)
	_ = os.WriteFile(a.settingsPath, data, 0o644)
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
	a.saveTick.Store(false)
	if a.layoutDirty {
		a.saveLayout()
		a.layoutDirty = false
	}
	a.saveSettings()
}

// invalidate asks for a frame after a change made outside the view.
func (a *app) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

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

// changed marks the layout for saving and schedules the redraw that lets
// the autosave run after a pause in typing.
func (a *app) changed() {
	a.layoutDirty = true
	a.lastChange = time.Now()
	a.scheduleSave()
}

func (a *app) scheduleSave() {
	win := a.win
	if win == nil {
		return
	}
	// The timer outlives a single frame. Recreate it for a reopened
	// window rather than waking one that is gone.
	if a.saveTimer == nil {
		a.saveTimer = time.AfterFunc(800*time.Millisecond, func() {
			a.saveTick.Store(true)
			win.Update(func() {})
		})
		return
	}
	a.saveTimer.Reset(800 * time.Millisecond)
}

// autosave writes the files and the layout once editing has paused. The
// timer that wakes the frame is the only path into the vault scan, so a
// frame that merely animates or hovers does not read every note.
func (a *app) autosave(c *ui.Context) {
	now := time.Now()
	if a.saveTick.Swap(false) && a.store != nil {
		saved := false
		for _, n := range a.store.notes {
			if !n.dirty() {
				continue
			}
			if err := a.store.save(n); err != nil {
				a.err = err.Error()
				a.invalidate()
				continue
			}
			saved = true
		}
		if saved {
			a.invalidate()
		}
	}
	if a.layoutDirty && now.Sub(a.lastChange) > 550*time.Millisecond {
		a.saveLayout()
		a.layoutDirty = false
	}
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
		return
	}
	a.tabs = []*tab{newTab(notes[0])}
	a.active = 0
	a.focusReq = a.tabs[0].Focus
	a.revealTab = true
	a.revealInTree(notes[0])
}

// newTab opens a workspace with the focused note.
func (a *app) newTab() {
	a.gitOpen = false
	n := a.currentNote()
	if n == nil {
		if a.store == nil {
			return
		}
		var err error
		if n, err = a.store.newNote(); err != nil {
			a.fail(err)
			return
		}
	}
	t := newTab(n)
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = t.editorPane()
	a.revealTab = true
	a.treeDraft = nil
	if t.Focus != nil {
		a.revealInTree(t.Focus.Note)
	}
	a.changed()
}

// openNoteInTab shows n in a tab of its own.
func (a *app) openNoteInTab(n *note) {
	if n == nil {
		return
	}
	a.gitOpen = false
	t := newTab(n)
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
	a.gitOpen = false
	a.treeDraft = nil
	a.treeQuery = ""
	t := a.tab()
	if t == nil {
		t = newTab(n)
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
		a.focusReq = t.Focus
		a.revealTab = true
		a.revealInTree(n)
		a.changed()
		return
	}
	p := t.Focus
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
	}
	p.Note = n
	t.Focus = p
	a.focusReq = p
	a.revealInTree(n)
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
		if p.Note != nil && p.Note.dirty() {
			if err := a.store.save(p.Note); err != nil {
				a.fail(err)
			}
		}
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
			a.revealInTree(next.Focus.Note)
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
		if p.Note != nil && p.Note.dirty() {
			if err := a.store.save(p.Note); err != nil {
				a.fail(err)
			}
		}
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
	a.gitOpen = false
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
	a.gitOpen = false
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
	if t := a.tab(); t != nil {
		name, _ := t.label()
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
			{Label: "Reveal Project", Click: func(*mygo.MenuItem, *mygo.Window) { a.revealVault() }},
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
