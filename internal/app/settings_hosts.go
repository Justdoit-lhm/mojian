package app

import (
	"context"
	"strings"
	"time"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

func (a *app) settingsHostsPage(c *ui.Context, k *colors) {
	if remoteSupported() {
		a.settings.Section(c, k, "ADD HOST", func() {
			ui.Column(c).Padding(14, 16).Gap(8).Children(func() {
				var addr, name ui.Element
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					addrField := ui.Row(c).Grow(1).MinWidth(0).Height(30).Padding(0, 9).
						Gap(7).Radius(radius).Background(k.Hover).AlignItems(ui.Center)
					addrField.Children(func() {
						ui.Icon(c, icons.Icon("server")).Size(13, 13).TextColor(k.TextFaint)
						addr = ui.TextInputBase(c, &a.hostDraftAddr).Grow(1).MinWidth(0).
							FontSize(12).Placeholder("me@devbox").Label("Host address")
					})
					nameField := ui.Row(c).Width(140).Height(30).Padding(0, 9).
						Radius(radius).Background(k.Hover).AlignItems(ui.Center)
					nameField.Children(func() {
						name = ui.TextInputBase(c, &a.hostDraftName).Grow(1).FontSize(12).
							Placeholder("Name").Label("Host name")
					})
					if addr.Submitted() || name.Submitted() {
						a.addHostChecked()
					}
					if settingsButton(c, "Add", true).Clicked() {
						a.addHostChecked()
					}
				})
				ui.Text(c, "Anything ssh accepts: buildbox, me@devbox, or ssh://me@devbox:2222.").
					FontSize(11.5).TextColor(k.TextFaint)
			})
		})
	}
	a.settings.Section(c, k, "SERVERS", func() {
		ui.Column(c).Padding(6).Children(func() {
			a.hostSettingsRow(c, k, "", "Local", thisComputerName(), a.serverAlias == "")
			if remoteSupported() {
				for _, h := range a.hosts() {
					ui.Divider(c).Margin(0, 10)
					label := h.Name
					if label == "" {
						label = h.Address
					}
					a.hostSettingsRow(c, k, h.Address, label, h.Address, a.serverAlias == h.Address)
				}
				if len(a.hosts()) == 0 {
					ui.Text(c, "No SSH servers yet. Add one above.").
						FontSize(12).TextColor(k.TextFaint).Padding(12, 10)
				}
			}
		})
	})
	if remoteSupported() {
		a.settings.Note(c, k, "Set Up Server uploads mojian-server and libghostty-vt to ~/.local/share/mojian on the host, using your SSH key. No root, no service and no open port: the app reaches it through an SSH forward. A server keeps running when Mojian or the SSH connection disconnects and never stops on its own: Stop Server ends it and its terminals, and Start Server brings it back. The Local row controls this machine's server the same way.")
	} else {
		a.settings.Note(c, k, "The Local row controls this machine's session server.")
	}
}

func (a *app) addHostChecked() {
	if !remoteSupported() {
		return
	}
	addr := strings.TrimSpace(a.hostDraftAddr)
	name := strings.TrimSpace(a.hostDraftName)
	if addr == "" {
		a.settings.Notice = "An address is required."
		a.invalidate()
		return
	}
	for _, h := range a.hosts() {
		if h.Address == addr {
			a.settings.Notice = "That server is already in the list."
			a.invalidate()
			return
		}
	}
	a.addHost(name, addr)
	a.hostDraftAddr, a.hostDraftName = "", ""
	a.settings.Notice = ""
	a.invalidate()
}

func (a *app) hostSettingsRow(c *ui.Context, k *colors, host, label, detail string, selected bool) {
	if host != "" && a.hostEditAddr == host {
		a.hostEditRow(c, k, host, label)
		return
	}
	outer := ui.Column(c.Key("server-" + host + "-" + label))
	if selected {
		outer.Background(k.PanelSel).Radius(8)
	}
	outer.Children(func() {
		row := ui.Row(c).MinHeight(48).Padding(0, 6).Gap(10).AlignItems(ui.Center)
		row.Children(func() {
			info := ui.ButtonBase(c).Grow(1).MinWidth(0).Gap(10).Justify(ui.Start).
				Cursor(ui.CursorPointer).Label("Server " + label)
			info.Children(func() {
				ui.Icon(c, icons.Icon("server")).Size(16, 16).TextColor(k.TextMuted)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(k.Text)
					ui.Text(c, detail).FontSize(11.5).TextColor(k.TextFaint).
						SingleLine().Ellipsis("…")
				})
				if selected {
					ui.Icon(c, icons.Icon("check")).Size(14, 14).TextColor(k.TextMuted)
				}
			})
			if info.Clicked() {
				a.selectServer(host)
			}
			if host != "" {
				edit := iconButton(c, k, "pencil", "Edit server "+label, 24, 14)
				if edit.Clicked() {
					a.hostEditAddr, a.hostEditName, a.hostEditAddress = host, label, host
					a.hostRemoveAddr, a.settings.Notice = "", ""
					a.invalidate()
				}
				remove := iconButton(c, k, "x", "Remove server "+label, 24, 14)
				if remove.Clicked() {
					a.hostRemoveAddr = host
					a.settings.Notice = ""
					a.invalidate()
				}
			}
		})
		if host != "" {
			a.hostSetupRow(c, k, host)
		} else {
			a.localServerRow(c, k)
		}
		if host != "" && a.hostRemoveAddr == host {
			ui.Row(c).Padding(0, 6, 10, 6).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Remove this server?").FontSize(12).TextColor(k.TextMuted).Grow(1)
				if settingsButton(c, "Cancel", false).Clicked() {
					a.hostRemoveAddr = ""
					a.invalidate()
				}
				if settingsButton(c, "Remove", true).TextColor(k.Danger).Clicked() {
					a.removeHost(host)
				}
			})
		}
	})
}

// hostState is what the app knows about one host's server: whether this
// session set it up or stopped it, and the last status probe.
type hostState struct {
	ready    bool
	stopped  bool
	running  bool
	checked  bool
	checking bool
	checkSeq int
}

// hostStateFor returns host's state, creating it on first use.
func (a *app) hostStateFor(host string) *hostState {
	if a.hostStates == nil {
		a.hostStates = map[string]*hostState{}
	}
	st := a.hostStates[host]
	if st == nil {
		st = &hostState{}
		a.hostStates[host] = st
	}
	return st
}

// serverRowSpec is one server's status line, its primary action and the
// stop confirmation beneath it.
type serverRowSpec struct {
	busy    bool
	status  string
	color   ui.Color
	action  string
	stopAsk bool
	search  bool
	act     func()
	stop    func()
	cancel  func()
}

// serverControlRow draws the status line used by the Local row and every
// remote host row.
func (a *app) serverControlRow(c *ui.Context, k *colors, s serverRowSpec) {
	ui.Column(c).Gap(4).Children(func() {
		row := ui.Row(c).Padding(0, 6, 6, 6).Gap(8).AlignItems(ui.Center)
		if s.search {
			if a.settings.Hits["Set Up Server"] {
				row.Background(k.PanelSel)
			}
			if a.settings.Reveal == "Set Up Server" {
				row.ScrollIntoView()
			}
		}
		row.Children(func() {
			if s.busy {
				ui.Spinner(c).Size(12, 12).Label("Server state")
			}
			ui.Text(c, s.status).FontSize(11).TextColor(s.color).Grow(1).SingleLine().Ellipsis("…")
			if !s.busy && s.action != "" {
				if settingsButton(c, s.action, false).Clicked() {
					s.act()
				}
			}
		})
		if s.stopAsk {
			ui.Row(c).Padding(0, 2, 10, 8).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Stop this server? Its terminals and the processes in them end.").
					FontSize(11.5).TextColor(k.TextMuted).Grow(1).LineHeight(1.35)
				if settingsButton(c, "Cancel", false).Clicked() {
					s.cancel()
				}
				if settingsButton(c, "Stop", true).TextColor(k.Danger).Clicked() {
					s.stop()
				}
			})
		}
	})
}

func (a *app) hostSetupRow(c *ui.Context, k *colors, host string) {
	st := a.hostStateFor(host)
	busy := a.hostSetupBusy && a.hostSetupAddr == host
	running := a.hostServerRunning(host)
	if !busy && !running && !st.checked {
		a.later(func() { a.checkHostServer(host) })
	}
	msg := "Server not set up yet"
	col := k.TextFaint
	switch {
	case busy:
		msg = a.hostSetupMsg
		col = k.TextMuted
	case a.hostSetupAddr == host && a.hostSetupErr != "":
		msg = a.hostSetupErr
		col = k.Danger
	case a.hostSetupAddr == host && a.hostSetupMsg != "":
		msg = a.hostSetupMsg
		col = k.TextMuted
	case st.checking:
		msg = "Checking server…"
		col = k.TextMuted
	case running:
		msg = "Server running"
		col = k.TextMuted
	}
	action := ""
	if !busy {
		switch {
		case running:
			action = "Stop Server"
		case st.stopped:
			action = "Start Server"
		default:
			action = "Set Up Server"
		}
	}
	a.serverControlRow(c, k, serverRowSpec{
		busy:    busy,
		status:  msg,
		color:   col,
		action:  action,
		stopAsk: a.hostStopAddr == host,
		search:  true,
		act: func() {
			if action == "Stop Server" {
				a.hostStopAddr = host
				a.invalidate()
				return
			}
			a.setUpHost(host)
		},
		stop: func() {
			a.hostStopAddr = ""
			a.stopHostServer(host)
		},
		cancel: func() {
			a.hostStopAddr = ""
			a.invalidate()
		},
	})
}

// localServerRow shows this machine's session server and controls it.
func (a *app) localServerRow(c *ui.Context, k *colors) {
	running := a.localServerRunning()
	msg := "Server not started"
	col := k.TextFaint
	switch {
	case a.localServerErr != "":
		msg = a.localServerErr
		col = k.Danger
	case running:
		msg = "Server running"
		col = k.TextMuted
	case a.localServerStopped:
		msg = "Server stopped"
		col = k.TextMuted
	}
	action := "Start Server"
	if running {
		action = "Stop Server"
	}
	a.serverControlRow(c, k, serverRowSpec{
		status:  msg,
		color:   col,
		action:  action,
		stopAsk: a.localStopAsk,
		act: func() {
			if running {
				a.localStopAsk = true
				a.invalidate()
				return
			}
			a.startLocalServer()
		},
		stop: a.stopLocalServer,
		cancel: func() {
			a.localStopAsk = false
			a.invalidate()
		},
	})
}

// hostServerRunning reports whether the host's server is known to be up:
// set up in this session, found by the status probe, or reached by a live
// pooled connection.
func (a *app) hostServerRunning(host string) bool {
	if st := a.hostStates[host]; st != nil && (st.ready || st.running) {
		return true
	}
	rc := a.remoteConn(host)
	return rc != nil && rc.Healthy()
}

// checkHosts asks every configured host about its server in the
// background, once per launch, so a server left running is known before
// Settings is opened. checkHostServer skips a host already checked.
func (a *app) checkHosts() {
	if !remoteSupported() {
		return
	}
	for _, h := range a.hosts() {
		if h.Address != "" {
			a.checkHostServer(h.Address)
		}
	}
}

// checkHostServer asks the host once whether its server still has a
// socket, so a server left running from the last launch shows Stop
// Server instead of Set Up Server.
func (a *app) checkHostServer(host string) {
	if host == "" {
		return
	}
	st := a.hostStateFor(host)
	if st.checked {
		return
	}
	st.checked, st.checking = true, true
	st.checkSeq++
	seq := st.checkSeq
	if a.win == nil && a.settingsWin == nil {
		running, _ := hostServerRunningFn(context.Background(), host)
		a.finishHostCheck(host, seq, running)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		running, _ := hostServerRunningFn(ctx, host)
		a.onUpdateUI(func() { a.finishHostCheck(host, seq, running) })
	}()
}

func (a *app) finishHostCheck(host string, seq int, running bool) {
	st := a.hostStates[host]
	if st == nil || st.checkSeq != seq {
		return
	}
	st.running = running
	st.checking = false
	a.invalidate()
}

// setHostRunning records the result of an explicit Start or Stop and
// cancels a status probe that was still in flight.
func (a *app) setHostRunning(host string, running bool) {
	st := a.hostStateFor(host)
	st.checkSeq++
	st.running = running
	st.checked = true
	st.checking = false
}

func (a *app) hostEditRow(c *ui.Context, k *colors, host, label string) {
	ui.Column(c).Padding(10).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			addrField := ui.Row(c).Grow(1).MinWidth(0).Height(30).Padding(0, 9).
				Gap(7).Radius(radius).Background(k.Hover).AlignItems(ui.Center)
			addrField.Children(func() {
				ui.Icon(c, icons.Icon("server")).Size(13, 13).TextColor(k.TextFaint)
				addr := ui.TextInputBase(c, &a.hostEditAddress).Grow(1).MinWidth(0).
					FontSize(12).Placeholder("me@devbox").Label("Host address")
				if addr.Submitted() {
					a.saveHostEdit(host)
				}
			})
			nameField := ui.Row(c).Width(140).Height(30).Padding(0, 9).
				Radius(radius).Background(k.Hover).AlignItems(ui.Center)
			nameField.Children(func() {
				name := ui.TextInputBase(c, &a.hostEditName).Grow(1).FontSize(12).
					Placeholder("Name").Label("Host name")
				if name.Submitted() {
					a.saveHostEdit(host)
				}
			})
			if settingsButton(c, "Cancel", false).Clicked() {
				a.hostEditAddr, a.settings.Notice = "", ""
				a.invalidate()
			}
			if settingsButton(c, "Save", true).Clicked() {
				a.saveHostEdit(host)
			}
		})
		ui.Text(c, "Editing "+label+". A new address takes effect the next time it connects.").
			FontSize(11.5).TextColor(k.TextFaint)
	})
}

func (a *app) saveHostEdit(host string) {
	addr := strings.TrimSpace(a.hostEditAddress)
	name := strings.TrimSpace(a.hostEditName)
	if addr == "" {
		a.settings.Notice = "An address is required."
		a.invalidate()
		return
	}
	for _, h := range a.hosts() {
		if h.Address == addr && h.Address != host {
			a.settings.Notice = "That address is already in the list."
			a.invalidate()
			return
		}
	}
	for i := range a.hosts() {
		if a.hosts()[i].Address != host {
			continue
		}
		if name == "" {
			name = addr
		}
		a.hosts()[i].Name, a.hosts()[i].Address = name, addr
		if a.serverAlias == host {
			a.serverAlias = addr
		}
		if st := a.hostStates[host]; st != nil {
			delete(a.hostStates, host)
			a.hostStates[addr] = st
		}
		if a.hostStopAddr == host {
			a.hostStopAddr = addr
		}
		if a.hostSetupAddr == host {
			a.hostSetupAddr = addr
		}
		delete(a.remoteHomes, host)
		delete(a.remoteHomes, addr)
		delete(a.remoteKeep, host)
		for j := range a.remoteProjects() {
			if a.remoteProjects()[j].Host == host {
				a.remoteProjects()[j].Host = addr
			}
		}
		a.hostEditAddr, a.settings.Notice = "", ""
		a.saveSettings()
		a.invalidate()
		return
	}
	a.hostEditAddr, a.settings.Notice = "", ""
	a.invalidate()
}

func (a *app) removeHost(address string) {
	for _, h := range a.hosts() {
		if h.Address != address {
			continue
		}
		current := a.serverAlias == address
		if current {
			if err := a.saveNow(); err != nil {
				a.fail(err)
				return
			}
		}
		a.hostService().RemoveHost(address)
		a.hostService().RemoveProjects(address)
		a.hostRemoveAddr = ""
		a.hostStopAddr = ""
		delete(a.hostStates, address)
		delete(a.remoteKeep, address)
		delete(a.remoteHomes, address)
		if a.hostSetupAddr == address {
			a.hostSetupAddr, a.hostSetupBusy = "", false
			a.hostSetupMsg, a.hostSetupErr = "", ""
		}
		for key, ws := range a.workspaces {
			if ws.serverAlias != address {
				continue
			}
			closeWorkspaceTerminals(ws)
			delete(a.workspaces, key)
		}
		if current {
			host := a.serverAlias
			a.dropPickerRemote()
			a.serverAlias, a.remotePath = "", ""
			a.abandonActiveWorkspace()
			a.releaseRemote(host)
			a.saveSettings()
			a.later(func() {
				if a.project != "" {
					a.openProject(a.project)
				}
			})
		} else {
			a.releaseRemote(address)
			a.saveSettings()
		}
		a.invalidate()
		return
	}
}

func (a *app) selectServer(host string) {
	if host == "" {
		a.later(func() {
			if err := a.saveNow(); err != nil {
				a.fail(err)
				return
			}
			host := a.serverAlias
			a.dropPickerRemote()
			a.serverAlias = ""
			a.remotePath = ""
			a.abandonActiveWorkspace()
			a.releaseRemote(host)
			a.saveSettings()
			if a.settingsWin != nil {
				a.settingsWin.Close()
			}
			if a.project != "" {
				a.openProject(a.project)
			}
		})
		return
	}
	a.later(func() {
		if a.settingsWin != nil {
			a.settingsWin.Close()
		}
		a.switchToServer(host)
	})
}
