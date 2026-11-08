// Package commands is the window's command registry: the static metadata
// menus, shortcuts, the palette and Settings share, plus the per-window
// bindings that run each command.
package commands

// Command is one action of the menus, shortcuts and palette.
type Command struct {
	ID      string
	Title   string
	Accel   string // the menu accelerator, e.g. CmdOrCtrl+O
	Display string // an optional key hint, e.g. the tree's ⌘⌫
	Glyph   string // the palette row's icon
}

// Palette is every command the app can run.
var Palette = []*Command{
	{ID: "open-project", Title: "Open Project…", Accel: "CmdOrCtrl+O", Glyph: "folder-open"},
	{ID: "new-file", Title: "New File…", Accel: "CmdOrCtrl+N", Glyph: "file-plus"},
	{ID: "new-folder", Title: "New Folder…", Accel: "CmdOrCtrl+Shift+N", Glyph: "folder-plus"},
	{ID: "rename", Title: "Rename…", Accel: "F2", Glyph: "pencil"},
	{ID: "trash", Title: "Move to Trash", Display: "⌘⌫", Glyph: "trash-2"},
	{ID: "new-tab", Title: "New Tab", Accel: "CmdOrCtrl+T", Glyph: "plus"},
	{ID: "save", Title: "Save", Accel: "CmdOrCtrl+S", Glyph: "save"},
	{ID: "split-left", Title: "Split Left", Accel: "CmdOrCtrl+Alt+Left", Glyph: "columns-2"},
	{ID: "split-right", Title: "Split Right", Accel: "CmdOrCtrl+Shift+D", Glyph: "columns-2"},
	{ID: "split-up", Title: "Split Up", Accel: "CmdOrCtrl+Alt+Up", Glyph: "rows-2"},
	{ID: "split-down", Title: "Split Down", Accel: "CmdOrCtrl+Shift+E", Glyph: "rows-2"},
	{ID: "zoom", Title: "Zoom Pane", Accel: "CmdOrCtrl+Shift+M", Glyph: "maximize-2"},
	{ID: "close-pane", Title: "Close Pane", Accel: "CmdOrCtrl+W", Glyph: "x"},
	{ID: "close-tab", Title: "Close Tab", Accel: "CmdOrCtrl+Shift+W", Glyph: "x"},
	{ID: "rename-tab", Title: "Rename Tab…", Accel: "CmdOrCtrl+Shift+R", Glyph: "pencil"},
	{ID: "next-tab", Title: "Next Tab", Accel: "Ctrl+Tab", Glyph: "chevron-down"},
	{ID: "prev-tab", Title: "Previous Tab", Accel: "Ctrl+Shift+Tab", Glyph: "chevron-down"},
	{ID: "palette", Title: "Command Palette…", Accel: "CmdOrCtrl+Shift+P", Glyph: "command"},
	{ID: "open-file", Title: "Open File…", Accel: "CmdOrCtrl+Shift+O", Glyph: "file-text"},
	{ID: "toggle-tree", Title: "Toggle File Tree", Accel: "CmdOrCtrl+Shift+B", Glyph: "panel-left"},
	{ID: "toggle-terminal", Title: "Toggle Terminal", Accel: "CmdOrCtrl+Shift+T", Glyph: "square-terminal"},
	{ID: "git", Title: "Git Diff", Accel: "CmdOrCtrl+Shift+G", Glyph: "git-branch"},
	{ID: "settings", Title: "Settings…", Accel: "CmdOrCtrl+,", Glyph: "settings"},
	{ID: "ssh", Title: "New SSH Terminal…", Accel: "CmdOrCtrl+Shift+L", Glyph: "server"},
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
