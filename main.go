// Mojian is a quiet workspace for Markdown notes, in MyGo's native UI.
// Tabs hold workspaces, workspaces hold panes, and every pane shows a note
// as source or as its rendered preview.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go test ./...       the model, the store and the view, without a window
package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

type app struct {
	win   *mygo.Window
	store *store

	dataDir      string
	layoutPath   string
	settingsPath string

	tabs   []*tab
	active int

	focusReq *pane

	notesOpen    bool
	noteQuery    string
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
	loaded      bool
	pending     []func()

	md      markdown.Options
	mdDark  bool
	mdTheme *markdown.Theme
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
			a.layoutPath = filepath.Join(dir, "layout.json")
			a.settingsPath = filepath.Join(dir, "settings.json")
		}
		s, err := openStore()
		if err != nil {
			log.Print(err)
			mygo.Dialog.Message(mygo.MessageOptions{
				Type:    mygo.MessageError,
				Title:   "Mojian could not open its notes",
				Message: err.Error(),
			})
			mygo.App.Quit()
			return
		}
		a.store = s
		a.loadSettings()
		if !a.restoreLayout() {
			a.focusFirstNote()
		}
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
	a.updateTitle()
	win.OnClosed(func() {
		a.saveNow()
		if a.saveTimer != nil {
			a.saveTimer.Stop()
		}
		a.win = nil
		a.focusReq = nil
	})
}

// loadSettings reads the appearance the app was left in.
func (a *app) loadSettings() {
	if a.settingsPath == "" {
		return
	}
	data, err := os.ReadFile(a.settingsPath)
	if err != nil {
		return
	}
	var s struct {
		Appearance int `json:"appearance"`
	}
	if json.Unmarshal(data, &s) == nil && s.Appearance >= 0 && s.Appearance <= 2 {
		a.appearance = s.Appearance
		a.syncAppearance()
	}
}

func (a *app) saveSettings() {
	if a.settingsPath == "" {
		return
	}
	data, _ := json.MarshalIndent(struct {
		Appearance int `json:"appearance"`
	}{a.appearance}, "", "  ")
	_ = os.MkdirAll(filepath.Dir(a.settingsPath), 0o755)
	_ = os.WriteFile(a.settingsPath, data, 0o644)
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
	if a.saveTimer == nil {
		a.saveTimer = time.AfterFunc(650*time.Millisecond, func() {
			win.Update(func() {})
		})
		return
	}
	a.saveTimer.Reset(650 * time.Millisecond)
}

// autosave writes the notes and the layout once editing has paused.
func (a *app) autosave(c *ui.Context) {
	now := time.Now()
	saved := false
	if a.store != nil {
		for _, n := range a.store.notes {
			if !n.dirty() || now.Sub(n.lastEdit) < 700*time.Millisecond {
				continue
			}
			if err := a.store.save(n); err != nil {
				a.err = err.Error()
			} else {
				saved = true
			}
		}
	}
	if a.layoutDirty && now.Sub(a.lastChange) > 550*time.Millisecond {
		a.saveLayout()
		a.layoutDirty = false
	}
	if saved {
		a.invalidate()
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
}

// newNote writes a note and shows it in the focused pane, or a new tab
// when none is open.
func (a *app) newNote() {
	if a.store == nil {
		return
	}
	n, err := a.store.newNote()
	if err != nil {
		a.fail(err)
		return
	}
	a.openNote(n)
	a.changed()
}

// newTab opens a workspace with the focused note.
func (a *app) newTab() {
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
	if a.noteHasEditor(n, nil) {
		t.Focus.Preview = true
	}
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.focusReq = t.Focus
	a.changed()
}

// openNote shows n in the focused pane, or in a new tab when none is
// open.
func (a *app) openNote(n *note) {
	if n == nil {
		return
	}
	t := a.tab()
	if t == nil {
		t = newTab(n)
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
		a.focusReq = t.Focus
		a.changed()
		return
	}
	p := t.Focus
	if p == nil {
		p = firstPane(t.Root)
	}
	if p == nil {
		p = newPane(n, false)
		t.Root = &node{Pane: p}
		t.Focus = p
	}
	p.Note = n
	p.Preview = a.noteHasEditor(n, p)
	t.Focus = p
	a.focusReq = p
	a.notesOpen = false
	a.changed()
}

// noteHasEditor reports whether a pane other than except shows the note
// in the editor. A note has one editor at a time; opening it elsewhere
// shows the preview instead, so two views never edit the same state.
func (a *app) noteHasEditor(n *note, except *pane) bool {
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p != except && p.Note == n && !p.Preview {
				return true
			}
		}
	}
	return false
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
	if len(a.tabs) == 0 {
		a.active = -1
	} else if a.active >= i {
		a.active = min(max(a.active-1, 0), len(a.tabs)-1)
	}
	a.focusReq = nil
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

// split adds a pane beside or below the focused one, showing the note's
// other face: the preview beside the editor, or the editor beside the
// preview.
func (a *app) split(down bool) {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	p := t.Focus
	with := newPane(p.Note, !p.Preview)
	if !with.Preview && a.noteHasEditor(p.Note, p) {
		with.Preview = true
	}
	if !t.split(p, with, down) {
		return
	}
	a.focusReq = with
	a.changed()
}

func (a *app) togglePreview() {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	t.Focus.Preview = !t.Focus.Preview
	a.focusReq = t.Focus
	a.changed()
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
	a.active = (a.active + delta + len(a.tabs)) % len(a.tabs)
	a.focusReq = nil
	a.changed()
}

// showTab shows the nth tab, from 1; 9 shows the last.
func (a *app) showTab(n int) {
	if len(a.tabs) == 0 {
		return
	}
	i := n - 1
	if n >= 9 {
		i = len(a.tabs) - 1
	}
	a.active = min(max(i, 0), len(a.tabs)-1)
	a.focusReq = nil
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
	a.invalidate()
}

func (a *app) finishRename(keep bool) {
	if a.rename == nil {
		return
	}
	if keep {
		a.rename.Name = a.renameText
		a.changed()
	}
	a.rename = nil
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
			item(cmdNewNote),
			item(cmdNewTab),
			item(cmdSave),
			mygo.Separator(),
			item(cmdCloseTab),
			{Label: "Reveal Vault", Click: func(*mygo.MenuItem, *mygo.Window) { a.revealVault() }},
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
			item(cmdPalette),
			item(cmdNotes),
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
			item(cmdPreview),
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

// markdownTheme returns the preview theme for the current appearance.
func (a *app) markdownTheme(c *ui.Context) *markdown.Theme {
	dark := c.Theme().Dark
	if a.mdTheme == nil || a.mdDark != dark {
		th := markdown.DefaultTheme(c.Theme())
		th.FontSize = 14.5
		th.LineHeight = 1.65
		th.CodeFont = "JetBrains Mono"
		a.mdTheme = th
		a.mdDark = dark
	}
	return a.mdTheme
}
