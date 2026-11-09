package app

import (
	"errors"
	"fmt"
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
	// The native appearance has to be set before the window exists, or
	// the first window opens in the system's and then flips.
	if !a.harness {
		mygo.Theme.SetSource(a.themeSource())
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:                "Mojian",
		Width:                1120,
		Height:               720,
		MinWidth:             760,
		MinHeight:            460,
		StateKey:             stateKey,
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 16, Y: 15},
		BackgroundColor:      a.windowBackground(),
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

// openSettingsWindow shows the preferences in a window of their own, or
// brings it forward.
func (a *app) openSettingsWindow() {
	if a.harness {
		// A native window needs mygo.App's event loop, which a harness
		// never starts; its Settings view has a fixture of its own.
		a.fail(errors.New("the Settings window is the Mojian Settings fixture"))
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
		BackgroundColor:      a.windowBackground(),
		Content:              ui.View(a.settingsWindowView),
	})
	a.settingsWin = win
	win.OnClosed(func() {
		a.settingsWin = nil
		a.settings.StopRecording()
		a.settings.Query = ""
		a.settings.Found, a.settings.Hits, a.settings.Reveal = nil, nil, ""
		a.settings.KeyFilter = ""
		a.hostEditAddr, a.hostRemoveAddr, a.settings.Notice = "", "", ""
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

// applyTheme gives the frame the appearance and the palette the user
// chose.
func (a *app) applyTheme(c *ui.Context) {
	dark := a.dark(c)
	c.SetTheme(windowTheme(c, a.colorsOf(c), dark))
}

// editorTheme returns the code theme for the current appearance and theme,
// kept between frames because it is built once per pair.
func (a *app) editorTheme(c *ui.Context) *editor.Theme {
	id, dark := themeName(a.Theme), a.dark(c)
	if a.edTheme == nil || a.edID != id || a.edDark != dark {
		a.edTheme = codeTheme(c.Theme(), id, dark)
		a.edID, a.edDark = id, dark
	}
	return a.edTheme
}

// themeSource is the appearance MyGo's native chrome draws in: the
// window's buttons, its menus and the accessibility preferences a page
// reads all follow it.
func (a *app) themeSource() mygo.ThemeSource {
	switch a.Appearance {
	case 1:
		return mygo.ThemeLight
	case 2:
		return mygo.ThemeDark
	}
	return mygo.ThemeSystem
}

// applyAppearanceSource hands the appearance to the native side, which the
// frame's colors alone cannot do: a forced dark window needs dark traffic
// lights and dark system panels too.
func (a *app) applyAppearanceSource() {
	if a.harness || a.win == nil {
		return
	}
	mygo.Theme.SetSource(a.themeSource())
}

// windowBackground is the color a window paints before its view draws, in
// both appearances, so resizing or a system appearance change never flashes
// the wrong one.
func (a *app) windowBackground() string {
	l, d := themeBgMid(a.Theme, false), themeBgMid(a.Theme, true)
	return fmt.Sprintf("light-dark(#%02x%02x%02x, #%02x%02x%02x)", l.R, l.G, l.B, d.R, d.G, d.B)
}

// updateWindowBackground repaints the two windows' own backgrounds after a
// theme or appearance change.
func (a *app) updateWindowBackground() {
	bg := a.windowBackground()
	if a.win != nil {
		if err := a.win.SetBackgroundColor(bg); err != nil {
			log.Print(err)
		}
	}
	if a.settingsWin != nil {
		if err := a.settingsWin.SetBackgroundColor(bg); err != nil {
			log.Print(err)
		}
	}
}
