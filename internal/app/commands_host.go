package app

import "mojian/internal/app/commands"

// initCommands binds the static command table to this window.
func (a *app) initCommands() {
	r := commands.NewRegistry()
	r.Bind("open-project", func() { a.openProjectPicker() })
	r.Bind("new-file", func() { a.startNewFile() })
	r.Bind("new-folder", func() { a.startNewFolder() })
	r.Bind("rename", func() { a.renameSelection() })
	r.Bind("trash", func() { a.trashSelection() })
	r.Bind("new-tab", func() { a.newTab() })
	r.Bind("save", func() {
		if err := a.saveNow(); err != nil {
			a.fail(err)
		}
	})
	r.Bind("focus-left", func() { a.focusPaneDirection(paneLeft) })
	r.Bind("focus-right", func() { a.focusPaneDirection(paneRight) })
	r.Bind("focus-up", func() { a.focusPaneDirection(paneUp) })
	r.Bind("focus-down", func() { a.focusPaneDirection(paneDown) })
	r.Bind("move-pane-left", func() { a.movePaneDirection(paneLeft) })
	r.Bind("move-pane-right", func() { a.movePaneDirection(paneRight) })
	r.Bind("move-pane-up", func() { a.movePaneDirection(paneUp) })
	r.Bind("move-pane-down", func() { a.movePaneDirection(paneDown) })
	r.Bind("split-left", func() { a.splitFocusedSide(splitLeft) })
	r.Bind("split-right", func() { a.splitFocusedSide(splitRight) })
	r.Bind("split-up", func() { a.splitFocusedSide(splitUp) })
	r.Bind("split-down", func() { a.splitFocusedSide(splitDown) })
	r.Bind("zoom", func() { a.toggleZoom() })
	r.Bind("close-pane", func() { a.closePane(a.focusPane()) })
	r.Bind("close-tab", func() { a.closeTab(a.tab()) })
	r.Bind("rename-tab", func() { a.startRename(a.tab()) })
	r.Bind("next-tab", func() { a.cycleTab(1) })
	r.Bind("prev-tab", func() { a.cycleTab(-1) })
	r.Bind("palette", func() { a.openPalette() })
	r.Bind("open-file", func() { a.openQuickOpen() })
	r.Bind("toggle-tree", func() { a.toggleTree() })
	r.Bind("toggle-terminal", func() { a.toggleTerminal() })
	r.Bind("git", func() { a.toggleGit() })
	r.Bind("settings", func() { openSettingsCommand(a) })
	r.Bind("ssh", func() { a.openSSHPicker() })
	a.commands = r
}

func (a *app) commandRegistry() *commands.Registry {
	if a.commands == nil {
		a.initCommands()
	}
	return a.commands
}
