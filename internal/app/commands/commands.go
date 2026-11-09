// Package commands is the window's command registry: the static metadata
// menus, shortcuts, the palette and Settings share, plus the per-window
// bindings that run each command.
package commands

import "runtime"

// Command is one action of the menus, shortcuts and palette.
type Command struct {
	ID      string
	Title   string
	Accel   string // the menu accelerator, e.g. CmdOrCtrl+O
	Other   string // the accelerator elsewhere than macOS, when it differs
	Display string // an optional key hint, e.g. the tree's ⌘⌫
	Glyph   string // the palette row's icon
}

// AccelFor is the default accelerator on goos: Accel on macOS, Other
// elsewhere when the platform keys differ, as Ghostty's do.
func (c *Command) AccelFor(goos string) string {
	if goos != "darwin" && c.Other != "" {
		return c.Other
	}
	return c.Accel
}

// Default is the default accelerator on this platform.
func (c *Command) Default() string { return c.AccelFor(runtime.GOOS) }

// Palette is every command the app can run.
//
// The defaults follow Ghostty where Mojian has the same idea: Command
// alone is a window or tab, Command and Option move through panes, and
// Shift adds a split, zooms it, or steps to the next tab. Linux keeps
// Control for the shell, so its tab and split defaults use Control with
// Shift, as Ghostty's do; a pane move has no default on either platform
// because dragging is the gesture. The editor owns several of these keys
// too, so its bindings hand the pane keys back to the window.
var Palette = []*Command{
	// Files and tabs.
	{ID: "open-project", Title: "Open Project…", Accel: "CmdOrCtrl+O", Glyph: "folder-open"},
	{ID: "open-file", Title: "Open File…", Accel: "CmdOrCtrl+P", Glyph: "file-text"},
	{ID: "new-file", Title: "New File…", Accel: "CmdOrCtrl+N", Glyph: "file-plus"},
	{ID: "new-folder", Title: "New Folder…", Accel: "CmdOrCtrl+Shift+N", Glyph: "folder-plus"},
	{ID: "save", Title: "Save", Accel: "CmdOrCtrl+S", Glyph: "save"},
	{ID: "rename", Title: "Rename…", Accel: "F2", Glyph: "pencil"},
	{ID: "trash", Title: "Move to Trash", Display: "⌘⌫", Glyph: "trash-2"},
	{ID: "new-tab", Title: "New Tab", Accel: "CmdOrCtrl+T", Other: "Ctrl+Shift+T", Glyph: "plus"},
	{ID: "close-tab", Title: "Close Tab", Accel: "CmdOrCtrl+Alt+W", Other: "Ctrl+Alt+W", Glyph: "x"},
	{ID: "rename-tab", Title: "Rename Tab…", Accel: "CmdOrCtrl+Shift+R", Glyph: "pencil"},
	{ID: "next-tab", Title: "Next Tab", Accel: "CmdOrCtrl+Shift+]", Other: "Ctrl+Tab", Glyph: "chevron-right"},
	{ID: "prev-tab", Title: "Previous Tab", Accel: "CmdOrCtrl+Shift+[", Other: "Ctrl+Shift+Tab", Glyph: "chevron-left"},

	// Panes. Ghostty binds a split to the right and down; left and up
	// have no defaults, as in Ghostty. Focus is Command+Option and an
	// arrow, and a zoom is Shift added to the platform's Enter.
	{ID: "focus-left", Title: "Focus Pane Left", Accel: "CmdOrCtrl+Alt+Left", Glyph: "columns-2"},
	{ID: "focus-right", Title: "Focus Pane Right", Accel: "CmdOrCtrl+Alt+Right", Glyph: "columns-2"},
	{ID: "focus-up", Title: "Focus Pane Above", Accel: "CmdOrCtrl+Alt+Up", Glyph: "rows-2"},
	{ID: "focus-down", Title: "Focus Pane Below", Accel: "CmdOrCtrl+Alt+Down", Glyph: "rows-2"},
	{ID: "move-pane-left", Title: "Move Pane Left", Glyph: "columns-2"},
	{ID: "move-pane-right", Title: "Move Pane Right", Glyph: "columns-2"},
	{ID: "move-pane-up", Title: "Move Pane Above", Glyph: "rows-2"},
	{ID: "move-pane-down", Title: "Move Pane Below", Glyph: "rows-2"},
	{ID: "split-left", Title: "Split Left", Glyph: "columns-2"},
	{ID: "split-right", Title: "Split Right", Accel: "CmdOrCtrl+D", Other: "Ctrl+Shift+O", Glyph: "columns-2"},
	{ID: "split-up", Title: "Split Up", Glyph: "rows-2"},
	{ID: "split-down", Title: "Split Down", Accel: "CmdOrCtrl+Shift+D", Other: "Ctrl+Shift+E", Glyph: "rows-2"},
	{ID: "zoom", Title: "Zoom Pane", Accel: "CmdOrCtrl+Shift+Enter", Glyph: "maximize-2"},
	{ID: "close-pane", Title: "Close Pane", Accel: "CmdOrCtrl+W", Other: "Ctrl+Shift+W", Glyph: "x"},

	// Views and tools.
	{ID: "palette", Title: "Command Palette…", Accel: "CmdOrCtrl+Shift+P", Glyph: "command"},
	{ID: "toggle-tree", Title: "Toggle File Tree", Accel: "CmdOrCtrl+B", Glyph: "panel-left"},
	{ID: "toggle-terminal", Title: "Toggle Terminal", Accel: "Ctrl+`", Glyph: "square-terminal"},
	{ID: "git", Title: "Git Diff", Accel: "Ctrl+Shift+G", Glyph: "git-branch"},
	{ID: "settings", Title: "Settings…", Accel: "CmdOrCtrl+,", Glyph: "settings"},
	{ID: "ssh", Title: "New SSH Terminal…", Accel: "CmdOrCtrl+Shift+S", Glyph: "server"},
}

var byID = func() map[string]*Command {
	m := make(map[string]*Command, len(Palette))
	for _, c := range Palette {
		m[c.ID] = c
	}
	return m
}()

// ByID returns the command with id, or nil.
func ByID(id string) *Command { return byID[id] }

// Registry holds the window's run functions for the static palette.
type Registry struct {
	runs map[string]func()
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{runs: map[string]func(){}} }

// Bind gives id its action.
func (r *Registry) Bind(id string, run func()) {
	if id != "" && run != nil {
		r.runs[id] = run
	}
}

// Run runs the command bound to id.
func (r *Registry) Run(id string) bool {
	run := r.runs[id]
	if run == nil {
		return false
	}
	run()
	return true
}
