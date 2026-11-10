package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"mojian/internal/icons"
	"mojian/internal/motion"
	"mojian/internal/remote"
	"mojian/internal/session"

	"github.com/egoist/mygo/ui"
)

// provisionRemoteFn is replaceable in tests; the real flow talks SSH.
var provisionRemoteFn = remote.Provision

// openRemotePicker asks the selected server for its home, then shows the
// directories to choose a project from.
func (a *app) openRemotePicker(host string) {
	if !terminalsSupported() {
		a.fail(fmt.Errorf("remote projects need macOS or Linux"))
		return
	}
	if a.remoteBusy {
		return
	}
	a.remoteOpening = false
	if err := a.saveNow(); err != nil {
		a.fail(fmt.Errorf("could not save the open project: %w", err))
		return
	}
	a.closeModals()
	a.initWorkspaces()
	a.remoteList = ui.ListState{}
	cachedHome := a.remoteHomes[host]
	existing := a.remoteForHost(host)
	if existing != nil && cachedHome != "" {
		a.showRemotePicker(host, cachedHome, existing)
		return
	}
	msg := "Connecting to " + a.hostLabel(host) + "…"
	if existing != nil {
		msg = "Reading folders on " + a.hostLabel(host) + "…"
	}
	seq, win, ctx := a.beginRemoteTask(msg)
	if seq == 0 {
		return
	}
	go func() {
		rc := existing
		var err error
		if rc == nil {
			rc, err = remote.DialContext(ctx, host)
		}
		home := cachedHome
		var dirs []session.FileEntry
		if err == nil && home == "" {
			home, err = remote.SSHOutputContext(ctx, host, `printf %s "$HOME"`)
		}
		if err == nil {
			onWindow(win, func() { a.setRemoteBusyMsg(seq, "Reading folders on "+a.hostLabel(host)+"…") })
			dirs, err = rc.Client.ListDirs(home)
		}
		onWindow(win, func() {
			if !a.endRemoteTask(seq) {
				if rc != nil && rc != existing {
					a.detachRemoteConn(rc)
				}
				return
			}
			if err != nil {
				if rc != nil && rc != existing {
					a.detachRemoteConn(rc)
				}
				a.fail(err)
				return
			}
			if home != "" {
				if a.remoteHomes == nil {
					a.remoteHomes = map[string]string{}
				}
				a.remoteHomes[host] = home
			}
			if rc != nil {
				a.installRemoteConn(rc)
				a.remotePickHost = host
			} else {
				a.remotePickHost = ""
			}
			a.remoteHost = host
			a.remoteQuery = home
			a.remoteDirs = dirs
			a.remoteDirsSeq++
			a.remoteDirsBusy = false
			a.remoteErr = ""
			a.remotePicker = true
			a.serverOpen = false
			a.invalidate()
		})
	}()
}

// showRemotePicker opens the folder picker at home on a connected host,
// without the window-wide loading card.
func (a *app) showRemotePicker(host, home string, rc *remote.Conn) {
	if rc == nil {
		return
	}
	a.installRemoteConn(rc)
	a.remotePickHost = host
	a.remoteHost = host
	a.remoteQuery = home
	a.remotePicker = true
	a.serverOpen = false
	a.remoteErr = ""
	a.loadRemoteDirs(home)
}

// pickerRemote is the server whose folders the project picker shows.
func (a *app) pickerRemote() *remote.Conn {
	if a.remotePickHost != "" {
		return a.remoteConn(a.remotePickHost)
	}
	return a.activeRemote()
}

// dropPickerRemote closes a connection made only for the picker. The
// open project's server is left alone.
func (a *app) dropPickerRemote() {
	if a.remoteOpening {
		return
	}
	a.remotePicker = false
	a.remoteDirsBusy = false
	a.remoteDirsSeq++
	host := a.remotePickHost
	a.remotePickHost = ""
	a.releaseRemote(host)
}

func (a *app) remotePickerModal(c *ui.Context, k *colors) {
	if !a.remotePicker {
		return
	}
	dirs := a.remoteDirs
	q := strings.ToLower(strings.TrimSpace(a.remoteQuery))
	ui.DialogBase(c, &a.remotePicker, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Width(620).Height(520).Radius(radius).Background(k.Sheet).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 70, -10, k.ShadowFocused).Clip()
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).Height(60).Padding(0, 18, 0, 12).Gap(11).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Box(c).Size(34, 34).Radius(radius).Background(k.Hover).Center().Children(func() {
					ui.Icon(c, icons.Icon("server")).Size(18, 18).TextColor(k.TextMuted)
				})
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "Open Project on "+a.remoteHost).FontSize(15).
						FontWeight(650).TextColor(k.Text)
					ui.Text(c, "Choose a folder on the server").FontSize(11.5).
						TextColor(k.TextFaint)
				})
				ui.Spacer(c)
				if iconButton(c, k, "x", "Close", 28, 15).Clicked() {
					a.remotePicker = false
				}
			})
			ui.Row(c).Padding(12, 18).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Grow(1).Height(34).Padding(0, 10).Gap(7).Radius(radius).
					Background(k.Hover).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icons.FolderIcon()).Size(14, 14).TextColor(k.TextFaint)
					in := ui.TextInputBase(c, &a.remoteQuery).Grow(1).FontSize(13).
						Placeholder("Server project path…").AutoFocus()
					if in.Submitted() {
						path := strings.TrimSpace(a.remoteQuery)
						if path != "" {
							a.remotePicker = false
							a.later(func() { a.openRemoteProjectAsync(a.remoteHost, path) })
						}
					}
				})
				if ui.PrimaryButton(c, "Open").Clicked() {
					path := strings.TrimSpace(a.remoteQuery)
					if path != "" {
						a.remotePicker = false
						a.remoteOpening = true
						a.later(func() { a.openRemoteProjectAsync(a.remoteHost, path) })
					}
				}
			})
			if a.remoteDirsBusy {
				ui.Row(c).Padding(10, 10).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(14, 14).Label("Reading folders")
					ui.Text(c, "Reading folders…").FontSize(12).TextColor(k.TextFaint)
				})
			} else {
				var shown []session.FileEntry
				for _, d := range dirs {
					if !d.Dir {
						continue
					}
					if q != "" && !strings.Contains(strings.ToLower(d.Rel), q) {
						continue
					}
					shown = append(shown, d)
				}
				openRel := ""
				a.remoteList.Key = func(i int) any { return shown[i].Rel }
				list := ui.List(c, &a.remoteList, len(shown), func(i int) {
					d := shown[i]
					row := ui.Row(c.Key(d.Rel)).Height(34).Padding(0, 10).MarginY(2).
						Gap(8).Radius(radius).Cursor(ui.CursorPointer).
						Label("Open " + d.Rel)
					if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() {
						openRel = d.Rel
					}
					row.Children(func() {
						ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
						ui.Text(c, filepath.Base(d.Rel)).FontSize(12.5).TextColor(k.Text).
							SingleLine().Ellipsis("…")
					})
				}).Grow(1).MinHeight(0).Padding(4, 6)
				if len(shown) == 0 {
					list.Children(func() {
						ui.Text(c, a.remoteErr).FontSize(12).TextColor(k.TextFaint).Padding(14)
					})
				}
				if openRel != "" {
					a.remoteQuery = openRel
					a.loadRemoteDirs(openRel)
				}
			}
		})
	})
	if !a.remotePicker && !a.remoteOpening {
		a.dropPickerRemote()
	}
}

// loadRemoteDirs reads a folder's subfolders away from the UI, with a
// small spinner inside the picker.
func (a *app) loadRemoteDirs(path string) {
	rc := a.pickerRemote()
	if rc == nil || rc.Client == nil {
		a.remoteErr = "The server is not connected."
		a.remoteDirs = nil
		a.invalidate()
		return
	}
	a.remoteDirsSeq++
	seq := a.remoteDirsSeq
	a.remoteDirsBusy = true
	a.remoteDirs = nil
	a.remoteErr = ""
	a.invalidate()
	win := a.win
	go func() {
		entries, err := rc.Client.ListDirs(path)
		onWindow(win, func() {
			if seq != a.remoteDirsSeq {
				return
			}
			a.remoteDirsBusy = false
			if err != nil {
				a.remoteErr = err.Error()
				a.remoteDirs = nil
			} else {
				a.remoteDirs = entries
			}
			a.invalidate()
		})
	}()
}

func (a *app) serverChip(c *ui.Context, k *colors) {
	label := a.remoteName()
	chip := ui.ButtonBase(c).Gap(6).Padding(3, 8, 3, 6).Radius(radius).
		Justify(ui.Start).Cursor(ui.CursorPointer).Label("Switch server: " + label).
		Tooltip("Server  " + nativeKeys("⇧⌘S")).MaxWidth(170).Shrink(0).ClipX().
		Transition(motion.Colors)
	if chip.Hovered() || a.serverOpen {
		chip.Background(k.Hover)
	}
	drop := float32(0)
	if a.serverOpen {
		drop = 1
	}
	drop = chip.AnimateTo("server-menu", drop, motion.PopoverIn)
	if chip.Clicked() {
		a.serverOpen = !a.serverOpen
	}
	chip.Children(func() {
		ui.Icon(c, icons.Icon("server")).Size(16, 16).TextColor(k.TextMuted)
		ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(k.TextMuted).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
	})
	ui.PopoverBase(c, chip, &a.serverOpen, func(panel ui.Element) {
		panel.Width(280).Padding(9).Radius(radius).Background(k.Panel).
			Border(1, k.PanelBorder).Shadow(0, 12, 32, 0, k.ShadowFocused).Gap(1).
			Translate(0, -4*(1-drop)).Opacity(0.45 + 0.55*drop)
		ui.Text(c, "SERVERS").FontSize(10).FontWeight(650).TextColor(k.TextFaint).Padding(4, 8)
		a.serverRow(c, k, "", "Local")
		for _, h := range a.hosts() {
			label := h.Name
			if label == "" {
				label = h.Address
			}
			a.serverRow(c, k, h.Address, label)
		}
		if len(a.hosts()) == 0 {
			ui.Text(c, "Add a server in Settings › Hosts.").FontSize(11).
				TextColor(k.TextFaint).Padding(4, 8)
		}
	})
}

func (a *app) serverRow(c *ui.Context, k *colors, host, label string) {
	selected := a.serverAlias == host
	row := ui.ButtonBase(c.Key("server:"+host)).Height(36).Padding(0, 8).Gap(8).
		Radius(radius).Justify(ui.Start).Cursor(ui.CursorPointer).Label("Server " + label)
	if selected {
		row.Background(k.PanelSel)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		a.serverOpen = false
		if host == "" {
			a.later(func() {
				if a.project != "" {
					a.openProject(a.project)
				}
			})
		} else {
			a.later(func() { a.switchToServer(host) })
		}
	}
	row.Children(func() {
		ui.Icon(c, icons.Icon("server")).Size(14, 14).TextColor(k.TextMuted)
		ui.Text(c, label).FontSize(12.5).TextColor(k.Text)
		if selected {
			ui.Spacer(c)
			ui.Icon(c, icons.Icon("check")).Size(13, 13).TextColor(k.TextMuted)
		}
	})
}

func (a *app) remoteProjectRows(c *ui.Context, k *colors) {
	recent := a.remoteProjects()
	ui.Text(c, "PROJECTS ON "+strings.ToUpper(a.remoteName())).
		FontSize(10).FontWeight(650).TextColor(k.TextFaint).Padding(4, 8)
	if len(recent) == 0 {
		ui.Text(c, "No recent remote projects.").FontSize(11.5).TextColor(k.TextFaint).Padding(4, 8)
	}
	for _, p := range recent {
		if p.Host != a.serverAlias {
			continue
		}
		p := p
		current := p.Path == a.remotePath
		a.projectRowBase(c, k, projectRowSpec{
			key:        "remote:" + p.Host + ":" + p.Path,
			label:      p.Path,
			title:      filepath.Base(p.Path),
			detail:     p.Path,
			closeLabel: p.Path,
			current:    current,
			open: func() {
				a.projectOpen = false
				a.later(func() { a.openRemoteProjectAsync(p.Host, p.Path) })
			},
			close: func() {
				a.projectOpen = false
				a.later(func() { a.closeRemoteProject(p.Host, p.Path) })
			},
		})
	}
}

func (a *app) hostLabel(address string) string {
	for _, h := range a.hosts() {
		if h.Address == address {
			if h.Name != "" {
				return h.Name
			}
			return h.Address
		}
	}
	return address
}

func (a *app) addHost(name, address string) {
	address = strings.TrimSpace(address)
	name = strings.TrimSpace(name)
	if address == "" {
		return
	}
	if name == "" {
		name = address
	}
	for i := range a.hosts() {
		if a.hosts()[i].Address == address {
			a.hosts()[i].Name = name
			a.saveSettings()
			a.invalidate()
			return
		}
	}
	a.hostService().AddHost(name, address)
	a.saveSettings()
	a.invalidate()
}

// setUpHost installs the server on a configured host in the background,
// so the Settings window never waits on SSH. The status belongs to the
// host row that started it.
func (a *app) setUpHost(host string) {
	if host == "" || a.hostSetupBusy {
		return
	}
	existing := a.remoteForHost(host)
	a.hostSetupAddr = host
	a.hostSetupBusy = true
	a.hostSetupMsg = "Connecting to " + a.hostLabel(host) + "…"
	a.hostSetupErr = ""
	a.invalidate()
	if a.win == nil && a.settingsWin == nil {
		setup, err := provisionRemoteFn(host, existing)
		a.finishHostSetup(host, setup, err)
		return
	}
	go func() {
		setup, err := provisionRemoteFn(host, existing)
		a.onUpdateUI(func() { a.finishHostSetup(host, setup, err) })
	}()
}

// finishHostSetup lands a setup result on the thread that owns the
// Settings window.
func (a *app) finishHostSetup(host string, setup *remote.Setup, err error) {
	if a.hostSetupAddr != host {
		if setup != nil && setup.Conn != nil && a.remoteConn(setup.Conn.Host) != setup.Conn {
			a.detachRemoteConn(setup.Conn)
		}
		a.hostSetupBusy = false
		return
	}
	a.hostSetupBusy = false
	if err != nil {
		a.hostSetupErr = err.Error()
		a.hostSetupMsg = ""
		a.invalidate()
		return
	}
	if setup.Home != "" {
		if a.remoteHomes == nil {
			a.remoteHomes = map[string]string{}
		}
		a.remoteHomes[host] = setup.Home
	}
	if setup.Conn != nil {
		if a.remoteConn(setup.Conn.Host) == setup.Conn {
			a.keepRemote(setup.Conn.Host)
		} else if cur := a.remoteForHost(host); cur != nil && cur != setup.Conn {
			remote.Close(setup.Conn)
		} else {
			a.installRemoteConn(setup.Conn)
			a.keepRemote(host)
		}
	}
	if a.hostReady == nil {
		a.hostReady = map[string]bool{}
	}
	a.hostReady[host] = true
	state := "already running"
	if setup.Uploaded {
		state = "uploaded"
	}
	if setup.Path == "" {
		a.hostSetupMsg = "Server ready"
	} else {
		a.hostSetupMsg = "Server " + state + " · " + setup.Path
	}
	a.invalidate()
}

// hostConfig is a server the user added in Settings › Hosts. Address is
// anything the system ssh accepts, such as buildbox, me@devbox or
// ssh://me@devbox:2222 (the last is passed through as ssh accepts it).
