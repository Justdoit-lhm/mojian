package app

import (
	"errors"
	"log"
	"runtime"

	"mojian/internal/assets"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// setDockIcon gives the running app the new ink icon as soon as it starts.
// The bundle carries the same icon, but the Dock keeps the icon a process
// launched with until it returns; setting it here also covers a bare
// executable run outside the .app.
func (a *app) setDockIcon() {
	if runtime.GOOS != "darwin" {
		return
	}
	png, err := assets.Files.ReadFile("brand/mojian-app-icon.png")
	if err != nil {
		log.Print(err)
		return
	}
	if err := mygo.App.Dock.SetIcon(png); err != nil {
		log.Print(err)
	}
}

// installTray adds Mojian's mark to the macOS menu bar. It stays after the
// last window closes, like Docker's menu bar item, and brings the window
// back without going through the Dock.
func (a *app) installTray() {
	if runtime.GOOS != "darwin" || a.tray != nil {
		return
	}
	icon, err := assets.Files.ReadFile("brand/mojian-tray.png")
	if err != nil {
		log.Print(err)
		return
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Show Mojian", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.showFromTray()
		}},
		{Label: "Settings…", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.openSettingsWindow()
		}},
		mygo.Separator(),
		{Label: "Quit Mojian", Click: func(*mygo.MenuItem, *mygo.Window) {
			mygo.App.Quit()
		}},
	})
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:             "mojian",
		Icon:           icon,
		IconIsTemplate: true,
		ToolTip:        "Mojian",
		Menu:           menu,
	})
	if err != nil {
		log.Print(err)
		return
	}
	a.tray = tray
}

// showFromTray brings the main window back, opening it if it was closed.
func (a *app) showFromTray() {
	if a.quitting {
		return
	}
	if a.win == nil {
		a.open()
		return
	}
	a.win.Restore()
	a.win.Show()
	a.win.Focus()
	mygo.App.Focus()
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
	win.OnClose(func(e *mygo.CloseEvent) {
		if a.quitting || a.saveSkipped {
			return
		}
		if err := a.saveNow(); err != nil {
			e.PreventDefault()
			a.askQuit(err, func() {
				a.saveSkipped = true
				win.Update(func() { win.Close() })
			})
		}
	})
	win.OnClosed(func() {
		a.windowClosed()
	})
	return win
}

// windowClosed clears the window state, keeping the tabs for a reopen.
func (a *app) windowClosed() {
	a.quitting = true
	if a.settingsWin != nil {
		a.settingsWin.Close()
	}
	if !a.saveSkipped {
		_ = a.saveNow()
	}
	a.closeWorkspaces()
	a.workspaces = nil
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
// openSettingsWindow shows the preferences in a window of their own, or
// brings it forward.
func (a *app) openSettingsWindow() {
	if a.harness {
		// A native window needs mygo.App's event loop, which a harness
		// never starts; its Settings view has a fixture of its own.
		a.fail(errors.New("Settings is the Mojian Settings fixture"))
		return
	}
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
	win.OnClosed(func() {
		a.settingsWin = nil
		a.stopRecording()
		a.settingsQuery = ""
		a.settingsFound, a.settingsHits, a.settingsReveal = nil, nil, ""
		a.settingsKeyFilter = ""
		a.hostEditAddr, a.hostRemoveAddr, a.settingsNotice = "", "", ""
	})
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
// else the default folder. The app's own project gets a welcome note; a
// project the user chose does not.
// applyTheme gives the frame the appearance the user chose.
func (a *app) applyTheme(c *ui.Context) {
	switch a.Appearance {
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
