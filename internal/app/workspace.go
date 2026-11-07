package app

import (
	"path/filepath"
	"strings"

	gitpage "mojian/internal/git"

	"github.com/egoist/mygo/ui"
)

// workspace is a project that stays loaded while another one is shown,
// so switching back restores its tabs, terminals and remote connection
// instead of reading the project again.
type workspace struct {
	key string

	project     string // the last local project, for the server switcher
	store       *store
	serverAlias string
	remotePath  string
	remote      *remoteConn
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

	fileClip       []string
	fileCut        bool
	fileClipRemote bool

	git       gitpage.Page
	revealTab bool
}

func workspaceKeyLocal(dir string) string {
	return "local:" + filepath.Clean(dir)
}

func workspaceKeyRemote(host, path string) string {
	return "remote:" + host + ":" + strings.TrimRight(path, "/")
}

// initWorkspaces prepares the caches at their first use.
func (a *app) initWorkspaces() {
	if a.workspaces == nil {
		a.workspaces = map[string]*workspace{}
	}
	if a.remotes == nil {
		a.remotes = map[string]*remoteConn{}
	}
}

// activeWorkspaceKey names the project that holds the window now.
func (a *app) activeWorkspaceKey() string {
	if a.remotePath != "" && a.serverAlias != "" {
		return workspaceKeyRemote(a.serverAlias, a.remotePath)
	}
	if a.store != nil {
		return workspaceKeyLocal(a.store.dir)
	}
	return ""
}

// captureWorkspace copies the loaded project out of the app's active
// fields.
func (a *app) captureWorkspace(key string) *workspace {
	ws := &workspace{
		key:            key,
		project:        a.project,
		store:          a.store,
		serverAlias:    a.serverAlias,
		remotePath:     a.remotePath,
		remote:         a.remote,
		layoutPath:     a.layoutPath,
		tabs:           a.tabs,
		active:         a.active,
		focusReq:       a.focusReq,
		treeOpen:       a.treeOpen,
		treeQuery:      a.treeQuery,
		treeSel:        a.treeSel,
		treeReveal:     a.treeReveal,
		openDirs:       a.openDirs,
		tabScroll:      a.tabScroll,
		fileCut:        a.fileCut,
		fileClipRemote: a.fileClipRemote,
		git:            a.git,
		revealTab:      a.revealTab,
	}
	if len(a.fileClip) > 0 {
		ws.fileClip = append([]string(nil), a.fileClip...)
	}
	return ws
}

// stashActiveWorkspace keeps the project on screen for a later switch.
func (a *app) stashActiveWorkspace() {
	a.initWorkspaces()
	a.dropClosingPanes(nil)
	// A Git refresh from this project must not land in the next one.
	a.git.Seq++
	a.git.DiffSeq++
	a.git.Loading = false
	a.git.DiffLoad = false
	key := a.activeWorkspaceKey()
	if key == "" {
		return
	}
	a.workspaces[key] = a.captureWorkspace(key)
	a.activeWS = key
}

// activateWorkspace puts a loaded project on screen without reading it
// again: its tabs, terminals and git page are the objects it kept.
func (a *app) activateWorkspace(ws *workspace) {
	if ws == nil {
		return
	}
	a.initWorkspaces()
	a.activeWS = ws.key
	a.workspaces[ws.key] = ws
	a.project = ws.project
	a.store = ws.store
	a.serverAlias = ws.serverAlias
	a.remotePath = ws.remotePath
	a.remote = ws.remote
	if ws.remote != nil {
		a.remotes[ws.remote.host] = ws.remote
	}
	a.layoutPath = ws.layoutPath
	a.tabs = ws.tabs
	a.active = ws.active
	a.focusReq = ws.focusReq
	a.treeOpen = ws.treeOpen
	a.treeQuery = ws.treeQuery
	a.treeSel = ws.treeSel
	a.treeReveal = ws.treeReveal
	a.openDirs = ws.openDirs
	if a.openDirs == nil {
		a.openDirs = map[string]bool{}
	}
	a.tabScroll = ws.tabScroll
	a.fileClip = append([]string(nil), ws.fileClip...)
	a.fileCut = ws.fileCut
	a.fileClipRemote = ws.fileClipRemote
	a.git = ws.git
	a.git.Seq++
	a.git.DiffSeq++
	a.git.Loading = false
	a.git.DiffLoad = false
	a.revealTab = ws.revealTab
	a.closeTransientUI()
	a.updateTitle()
	a.invalidate()
}

// closeTransientUI drops the pickers and edits that belong to the
// project that was on screen, not to the one coming next.
func (a *app) closeTransientUI() {
	a.dropPickerRemote()
	a.serverOpen = false
	a.projectOpen = false
	a.projectPicker = false
	a.paletteOpen = false
	a.quickOpen = false
	a.sshPick = false
	a.rename = nil
	a.treeDraft = nil
	a.treeSearch.CancelFocus()
	a.err = ""
}

// switchWorkspace saves the project on screen and shows another one
// already loaded.
func (a *app) switchWorkspace(ws *workspace) {
	if ws == nil || ws.key == a.activeWS {
		if ws != nil {
			a.invalidate()
		}
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
	key := a.activeWorkspaceKey()
	if key == "" {
		return
	}
	a.initWorkspaces()
	a.activeWS = key
	a.workspaces[key] = a.captureWorkspace(key)
}

// closeWorkspaces detaches every loaded project's terminals and ends its
// remote server, as quitting the app does.
func (a *app) closeWorkspaces() {
	a.initWorkspaces()
	if key := a.activeWorkspaceKey(); key != "" {
		a.workspaces[key] = a.captureWorkspace(key)
	}
	for _, ws := range a.workspaces {
		closeWorkspaceTerminals(ws)
	}
	active := a.remote
	for host, rc := range a.remotes {
		a.closeRemoteConn(rc)
		delete(a.remotes, host)
	}
	if active != nil {
		a.closeRemoteConn(active)
	}
	a.remotes = nil
	a.remote = nil
}

func closeWorkspaceTerminals(ws *workspace) {
	if ws == nil {
		return
	}
	for _, t := range ws.tabs {
		for _, p := range t.panes() {
			if p.Term == nil {
				continue
			}
			if p.Term.Term != nil {
				_ = p.Term.Term.Close()
				p.Term.Term = nil
			}
			if p.Term.stream != nil {
				_ = p.Term.stream.Close()
				p.Term.stream = nil
			}
		}
	}
}
