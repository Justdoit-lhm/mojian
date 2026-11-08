package app

import (
	"github.com/egoist/mygo"
	"mojian/internal/app/commands"
)

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
	if a.store == nil || a.store.IsMemory() {
		return
	}
	if err := mygo.Shell.OpenPath(a.store.Dir); err != nil {
		a.fail(err)
	}
}

// menu builds the application menu. Its accelerators are the app's
// shortcuts on every platform.
func (a *app) menu() *mygo.Menu {
	item := func(c *commands.Command) *mygo.MenuItem {
		return &mygo.MenuItem{
			Label:       c.Title,
			Accelerator: a.menuAccel(c),
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.run(c)
			},
		}
	}
	appearance := func(label string, v int) *mygo.MenuItem {
		it := &mygo.MenuItem{
			Label:   label,
			Type:    mygo.MenuItemRadio,
			Checked: a.Appearance == v,
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
	fileItems := []*mygo.MenuItem{
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
	}
	if terminalsSupported() {
		fileItems = append(fileItems, mygo.Separator(), item(cmdSSH))
	}
	fileItems = append(fileItems, mygo.Separator(), &mygo.MenuItem{Label: "Reveal Project", Click: func(*mygo.MenuItem, *mygo.Window) {
		a.revealVault()
	}})
	viewItems := []*mygo.MenuItem{item(cmdTree)}
	if terminalsSupported() {
		viewItems = append(viewItems, item(cmdTerminal))
	}
	viewItems = append(viewItems, item(cmdGit), item(cmdPalette), mygo.Separator(), item(cmdSettings),
		&mygo.MenuItem{Label: "Appearance", Submenu: []*mygo.MenuItem{
			appearance("System", 0),
			appearance("Light", 1),
			appearance("Dark", 2),
		}},
		mygo.Separator(),
		&mygo.MenuItem{Role: mygo.RoleToggleFullScreen},
	)
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: fileItems},
		{Label: "Edit", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleUndo},
			{Role: mygo.RoleRedo},
			mygo.Separator(),
			{Role: mygo.RoleCut},
			{Role: mygo.RoleCopy},
			{Role: mygo.RolePaste},
			{Role: mygo.RoleSelectAll},
		}},
		{Label: "View", Submenu: viewItems},
		{Label: "Tabs", Submenu: append(tabItems,
			mygo.Separator(),
			item(cmdRenameTab),
		)},
		{Label: "Pane", Submenu: []*mygo.MenuItem{
			item(cmdSplitLeft),
			item(cmdSplitRight),
			item(cmdSplitUp),
			item(cmdSplitDown),
			mygo.Separator(),
			item(cmdZoom),
			mygo.Separator(),
			item(cmdClosePane),
		}},
		{Role: mygo.RoleWindowMenu},
	})
}

// run runs a command from a menu or the palette.
func (a *app) run(c *commands.Command) {
	if c == nil {
		return
	}
	if c == cmdClosePane && a.settingsWin != nil && a.settingsWin.IsFocused() {
		a.settingsWin.Close()
		return
	}
	a.commandRegistry().Run(c.ID)
	a.updateTitle()
	a.invalidate()
}
