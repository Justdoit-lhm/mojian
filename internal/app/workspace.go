package app

import (
	"path/filepath"
	"strings"

	gitpage "mojian/internal/git"

	"github.com/egoist/mygo/ui"
)

// workspace is a project that stays loaded while another one is shown,
// so switching back restores its tabs, terminals and remote connection
// instead of reading the project again. The active workspace is embedded
// in app, so its fields are the window's project state and switching does
// not copy them.
type workspace struct {
	key string

	project     string // the last local project, for the server switcher
	store       *store
	serverAlias string
	remotePath  string
	layoutPath  string

	tabs     []*tab
	active   int
	focusReq *pane

	treeOpen   bool
	treeQuery  string
	treeSel    string
	treeReveal string
	openDirs   map[string]bool
	tabScroll  ui.ScrollState

	git       gitpage.Page
	revealTab bool
}

func workspaceKeyLocal(dir string) string {
	return "local:" + filepath.Clean(dir)
}

func workspaceKeyRemote(host, path string) string {
	return "remote:" + host + ":" + strings.TrimRight(path, "/")
}

// initWorkspaces makes sure the active workspace and the caches exist.
func (a *app) initWorkspaces() {
	if a.workspace == nil {
		a.workspace = &workspace{}
	}
	if a.workspaces == nil {
		a.workspaces = map[string]*workspace{}
	}
	if a.remotes == nil {
		a.remotes = map[string]*remoteConn{}
	}
}

// activeWorkspaceKey names the project that holds the window now. The
// workspace keeps its key once opened; the derivation is only a fallback
// for an app built without openProject (tests) or a restored layout.
func (a *app) activeWorkspaceKey() string {
	if a.workspace != nil && a.workspace.key != "" {
		return a.workspace.key
	}
	if a.remotePath != "" && a.serverAlias != "" {
		return workspaceKeyRemote(a.serverAlias, a.remotePath)
	}
	if a.store != nil {
		return workspaceKeyLocal(a.store.dir)
	}
	return ""
}

// stashActiveWorkspace keeps the project on screen for a later switch.
// The active workspace is the object stored in the cache: there is
// nothing to copy.
func (a *app) stashActiveWorkspace() {
	a.initWorkspaces()
	a.dropClosingPanes(nil)
	// A Git refresh from this project must not land in the next one.
	a.git.Seq++
	a.git.DiffSeq++
	a.git.Loading = false
	a.git.DiffLoad = false
	if key := a.activeWorkspaceKey(); key != "" {
		a.workspace.key = key
		a.workspaces[key] = a.workspace
	}
}

// activateWorkspace puts a loaded project on screen without reading it
// again: its tabs, terminals and git page are the objects it kept.
func (a *app) activateWorkspace(ws *workspace) {
	if ws == nil {
		return
	}
	a.initWorkspaces()
	a.workspace = ws
	if ws.key != "" {
		a.workspaces[ws.key] = ws
	}
	if ws.openDirs == nil {
		ws.openDirs = map[string]bool{}
	}
	// A refresh from the project that was on screen must not land here.
	a.git.Seq++
	a.git.DiffSeq++
	a.git.Loading = false
	a.git.DiffLoad = false
	a.closeTransientUI()
	// A cut is a pending one-shot action for the project it was made in.
	a.fileCut = false
	a.updateTitle()
	a.invalidate()
}

// abandonActiveWorkspace closes the terminals of the project on screen
// without keeping it as the active workspace. The cached entry, if any,
// stays for a later reconnect.
func (a *app) abandonActiveWorkspace() {
	if a.workspace == nil {
		return
	}
	closeWorkspaceTerminals(a.workspace)
	a.workspace = &workspace{}
}

// closeModals closes every dialog and popover that belongs to the
// window, so opening one cannot leave another one drawn underneath it.
func (a *app) closeModals() {
	a.dropPickerRemote()
	a.projectOpen = false
	a.serverOpen = false
	a.projectPicker = false
	a.paletteOpen = false
	a.quickOpen = false
	a.sshPick = false
	a.remotePicker = false
}

// closeTransientUI drops the pickers and edits that belong to the
// project that was on screen, not to the one coming next.
func (a *app) closeTransientUI() {
	a.closeModals()
	a.rename = nil
	a.treeDraft = nil
	a.treeSearch.CancelFocus()
	a.err = ""
}

// switchWorkspace saves the project on screen and shows another one
// already loaded.
func (a *app) switchWorkspace(ws *workspace) {
	if ws == nil {
		return
	}
	if ws == a.workspace {
		a.invalidate()
		return
	}
	if err := a.saveNow(); err != nil {
		a.fail(err)
		return
	}
	a.saveLayout()
	a.layoutDirty = false
	a.dropPickerRemote()
	a.stashActiveWorkspace()
	a.activateWorkspace(ws)
	a.saveSettings()
	a.updateTitle()
	a.invalidate()
}

// registerActiveWorkspace records the project loaded at launch or after
// a first open, so leaving and returning to it is a switch, not a read.
func (a *app) registerActiveWorkspace() {
	a.initWorkspaces()
	if key := a.activeWorkspaceKey(); key != "" {
		a.workspace.key = key
		a.workspaces[key] = a.workspace
	}
}

// closeWorkspaces detaches every loaded project's terminals and remote
// connection, as quitting the app does. The servers keep running.
func (a *app) closeWorkspaces() {
	a.initWorkspaces()
	if key := a.activeWorkspaceKey(); key != "" {
		a.workspace.key = key
		a.workspaces[key] = a.workspace
	}
	for _, ws := range a.workspaces {
		closeWorkspaceTerminals(ws)
	}
	for host, rc := range a.remotes {
		closeRemoteConn(rc)
		delete(a.remotes, host)
	}
	a.remotes = nil
	a.remotePickHost = ""
	a.remoteOpening = false
}

func closeWorkspaceTerminals(ws *workspace) {
	if ws == nil {
		return
	}
	for _, t := range ws.tabs {
		for _, p := range t.panes() {
			detachTerminalView(p)
		}
	}
}
