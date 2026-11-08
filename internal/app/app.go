// Mojian is a quiet text editor in MyGo's native UI. Tabs hold
// workspaces, workspaces hold editors, and every editor shows a file's
// source with syntax highlighting.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go test ./...       the model, the store and the view, without a window
package app

import (
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"mojian/internal/app/commands"
	"mojian/internal/app/explorer"
	"mojian/internal/app/settingsui"
	"mojian/internal/fs"
	"mojian/internal/prefs"
	"mojian/internal/remotehost"
	"mojian/internal/session"
	"mojian/internal/update"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type app struct {
	// workspace is the project on screen. It is also in workspaces, so
	// switching stores and restores the same object instead of copying
	// its fields into the app.
	*workspace
	prefs.Prefs

	win  *mygo.Window
	tray *mygo.Tray

	dataDir      string
	settingsPath string

	projects       []string
	workspaces     map[string]*workspace
	remoteHosts    *remotehost.Service
	remoteBusy     bool
	remoteBusyMsg  string
	remoteTaskSeq  int
	remoteDirsSeq  int
	remoteDirsBusy bool
	projectOpen    bool
	projectPicker  bool
	remotePickHost string // the host a picker connection was made for
	remoteOpening  bool   // a picked project is about to open
	// fileClip is the app's own clipboard: absolute local paths, or paths
	// relative to a remote project (fileClipRemote). A cut is a one-shot
	// intent and is cleared when the project changes.
	fileClip       []string
	fileCut        bool
	fileClipRemote bool
	fileClipHost   string
	fileClipRoot   string
	hostDraftName  string
	hostDraftAddr  string
	serverOpen     bool
	remotePicker   bool
	remoteHost     string
	remoteQuery    string
	remoteDirs     []session.FileEntry
	remoteErr      string
	pickerPath     string
	pickerInput    string
	pickerEntries  []pickerEntry
	pickerSel      int
	pickerErr      string

	closingPanes []*closingPane

	sshPick  bool
	sshQuery string
	sshSel   int

	treeSearch ui.Handle
	treeDraft  *treeDraft
	draftSeq   int

	updateRelaunchFn func()
	updateChecking   bool
	updateInstalling bool
	updateReady      bool
	updateChecked    bool
	pendingUpdate    *update.Release
	updateErr        string

	shellStarter  func() (*termTab, error) // tests replace this
	sessionClient *session.Client
	// harness marks the headless fixtures of cmd/harness: no native
	// windows or session servers, only the fixture's disposable project.
	harness bool

	paletteOpen  bool
	paletteQuery string
	paletteSel   int

	quickOpen  bool
	quickQuery string
	quickSel   int

	rename      *tab
	renameText  string
	renameFocus bool

	appearanceItems [3]*mygo.MenuItem

	settingsWin     *mygo.Window
	settings        settingsui.UI
	explorer        explorer.UI
	hostEditAddr    string
	hostEditName    string
	hostEditAddress string
	hostRemoveAddr  string
	hostSetupAddr   string
	hostSetupBusy   bool
	hostSetupMsg    string
	hostSetupErr    string
	hostReady       map[string]bool
	shortcutKeys    map[string]string
	commands        *commands.Registry

	err   string
	title string

	layoutDirty   bool
	lastChange    time.Time
	layoutTimer   *time.Timer
	settingsTimer *time.Timer
	loaded        bool
	quitting      bool
	saveSkipped   bool
	pending       []func()

	edDark  bool
	edTheme *editor.Theme
}

func Run() {
	registerFonts()
	a := &app{
		workspace: &workspace{},
		Prefs:     prefs.Default(),
	}
	go warmSessionServer()
	update.UseGitHub()
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
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		a.quitting = true
		if !a.saveSkipped {
			if err := a.saveNow(); err != nil {
				// Keep the window and its unsaved text: a lost remote
				// connection or a full disk must not discard edits.
				e.PreventDefault()
				a.quitting = false
				a.askQuit(err, func() {
					a.saveSkipped = true
					mygo.App.Quit()
				})
				return
			}
		}
		a.closeWorkspaces()
		if a.sessionClient != nil {
			_ = a.sessionClient.Close()
			a.sessionClient = nil
		}
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
	a.setDockIcon()
	a.installTray()
	if !a.loaded {
		a.loaded = true
		if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
			a.dataDir = dir
			a.settingsPath = filepath.Join(dir, "settings.json")
		}
		a.loadSettings()
		if len(a.shortcutKeys) > 0 {
			mygo.App.SetMenu(a.menu())
		}
		if a.serverAlias != "" && a.remotePath != "" {
			// Create the window first and connect behind the loading card,
			// so an unreachable host cannot delay the launch.
			a.treeOpen = true
			a.win = a.createWindow("main")
			a.updateTitle()
			a.openRemoteProjectAsync(a.serverAlias, a.remotePath)
			return
		}
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
		a.layoutPath = a.projectLayoutPath(s.Dir)
		if firstProject && !s.IsMemory() {
			a.migrateLayout(s.Dir)
		}
		a.rememberProject(s.Dir)
		a.saveSettings()
		if a.NoRestoreLayout || !a.restoreLayout() {
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
	a.quitting = false
	a.saveSkipped = false
	if a.loaded && a.serverAlias != "" && a.remotePath != "" {
		// Reopening a window on a remote project reconnects behind the
		// loading card instead of holding the window back.
		a.win = a.createWindow("main")
		a.registerActiveWorkspace()
		a.updateTitle()
		a.reconnectActiveRemoteAsync()
		return
	}
	a.win = a.createWindow("main")
	a.registerActiveWorkspace()
	a.updateTitle()
}

// initialStore opens $MOJIAN_VAULT, else the project of the last launch,
// else the default folder. The app's own project gets a welcome note; a
// project the user chose does not.
func (a *app) initialStore() (*store, error) {
	if os.Getenv("MOJIAN_VAULT") != "" {
		return a.openDefaultStore()
	}
	if a.project != "" {
		if info, err := os.Stat(a.project); err == nil && info.IsDir() {
			return a.openLocalStore(a.project)
		}
	}
	return a.openDefaultStore()
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
	if err := fs.WriteFileAtomic(a.layoutPath, data); err != nil {
		log.Print(err)
	}
}

func (a *app) askQuit(err error, proceed func()) {
	parent := a.win
	go func() {
		res, _ := mygo.Dialog.Message(mygo.MessageOptions{
			Parent: parent, Type: mygo.MessageError,
			Title:   "Mojian could not save",
			Message: err.Error(),
			Detail:  "Your edits are still in memory. Quit without saving?",
			Buttons: []string{"Keep Editing", "Quit Without Saving"},
		})
		if res.Button == 1 {
			proceed()
		}
	}()
}

// saveNow writes every unsaved note, the layout and the settings. It
// returns the first note that could not be written, so a project switch
// can stop before dropping unsaved work.
func (a *app) saveNow() error {
	var first error
	if a.store != nil {
		if err := a.store.SaveAll(); err != nil {
			log.Print(err)
			first = err
		}
	}
	if a.layoutDirty {
		a.saveLayout()
		a.layoutDirty = false
	}
	a.saveSettings()
	return first
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
	switch {
	case a.win != nil:
		a.win.Update(func() {})
	case a.settingsWin != nil:
		a.settingsWin.Update(func() {})
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
