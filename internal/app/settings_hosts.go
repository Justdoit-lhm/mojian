package app

import (
	"strings"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

func (a *app) settingsHostsPage(c *ui.Context, k *colors) {
	a.settingsNoticeLine(c, k)
	a.settingsSection(c, k, "ADD HOST", func() {
		row := ui.Column(c).Key("host-add").Padding(10, 16).Gap(6).Radius(6)
		if a.settingsHits["Address"] {
			row.Background(k.PanelSel)
		}
		if a.settingsReveal == "Address" {
			row.ScrollIntoView()
		}
		row.Children(func() {
			ui.Text(c, "Address").FontSize(12.5).FontWeight(550).TextColor(k.Text)
			ui.Text(c, "Anything ssh accepts: buildbox, me@devbox, or ssh://me@devbox:2222.").
				FontSize(11.5).TextColor(k.TextFaint).LineHeight(1.35)
			var addr, name ui.Element
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				addrField := ui.Row(c).Grow(1).MinWidth(0).Height(30).Padding(0, 9).
					Gap(7).Radius(radius).Background(k.Hover).AlignItems(ui.Center)
				addrField.Children(func() {
					ui.Icon(c, icons.Icon("server")).Size(13, 13).TextColor(k.TextFaint)
					addr = ui.TextInputBase(c, &a.hostDraftAddr).Grow(1).MinWidth(0).
						FontSize(12).Placeholder("me@devbox")
				})
				nameField := ui.Row(c).Width(120).Height(30).Padding(0, 9).
					Radius(radius).Background(k.Hover).AlignItems(ui.Center)
				nameField.Children(func() {
					name = ui.TextInputBase(c, &a.hostDraftName).Grow(1).FontSize(12).
						Placeholder("Name")
				})
				if addr.Submitted() || name.Submitted() {
					a.addHostChecked()
				}
				if settingsButton(c, "Add", true).Clicked() {
					a.addHostChecked()
				}
			})
		})
	})
	a.settingsSection(c, k, "SERVERS", func() {
		a.hostSettingsRow(c, k, "", "Local", thisComputerName(), a.serverAlias == "")
		for _, h := range a.hosts {
			ui.Box(c).Height(1).Shrink(0).Background(k.PanelBorder)
			label := h.Name
			if label == "" {
				label = h.Address
			}
			a.hostSettingsRow(c, k, h.Address, label, h.Address, a.serverAlias == h.Address)
		}
		if len(a.hosts) == 0 {
			ui.Text(c, "No servers yet. Add an SSH host above.").
				FontSize(12).TextColor(k.TextFaint).Padding(14, 16)
		}
	})
	a.settingsSection(c, k, "ABOUT HOSTS", func() {
		ui.Text(c, "Set Up Server uploads mojian-server and libghostty-vt to ~/.local/share/mojian on the host, using your SSH key. No root, no service and no open port: the app reaches it through an SSH forward.").
			FontSize(12).TextColor(k.TextFaint).LineHeight(1.4).Padding(12, 16)
	})
}

func (a *app) addHostChecked() {
	addr := strings.TrimSpace(a.hostDraftAddr)
	name := strings.TrimSpace(a.hostDraftName)
	if addr == "" {
		a.settingsNotice = "An address is required."
		a.invalidate()
		return
	}
	for _, h := range a.hosts {
		if h.Address == addr {
			a.settingsNotice = "That server is already in the list."
			a.invalidate()
			return
		}
	}
	a.addHost(name, addr)
	a.hostDraftAddr, a.hostDraftName = "", ""
	a.settingsNotice = ""
	a.invalidate()
}

func (a *app) hostSettingsRow(c *ui.Context, k *colors, host, label, detail string, selected bool) {
	if host != "" && a.hostEditAddr == host {
		a.hostEditRow(c, k, host, label)
		return
	}
	outer := ui.Column(c).Key("server-" + host + "-" + label)
	if selected {
		outer.Background(k.PanelSel).Radius(6)
	}
	outer.Children(func() {
		row := ui.Row(c).MinHeight(52).Padding(0, 8).Gap(8).AlignItems(ui.Center)
		row.Children(func() {
			info := ui.Row(c).Grow(1).MinWidth(0).Gap(10).AlignItems(ui.Center).
				Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Server " + label)
			info.Children(func() {
				ui.Icon(c, icons.Icon("server")).Size(16, 16).TextColor(k.TextMuted)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, label).FontSize(12.5).FontWeight(550).TextColor(k.Text)
					ui.Text(c, detail).FontSize(11).TextColor(k.TextFaint).
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
					a.hostRemoveAddr, a.settingsNotice = "", ""
					a.invalidate()
				}
				remove := iconButton(c, k, "x", "Remove server "+label, 24, 14)
				if remove.Clicked() {
					a.hostRemoveAddr = host
					a.settingsNotice = ""
					a.invalidate()
				}
			}
		})
		if host != "" {
			a.hostSetupRow(c, k, host)
		}
		if host != "" && a.hostRemoveAddr == host {
			ui.Row(c).Padding(0, 8, 10, 8).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Remove this server?").FontSize(12).TextColor(k.TextMuted).Grow(1)
				if settingsButton(c, "Cancel", false).Clicked() {
					a.hostRemoveAddr = ""
					a.invalidate()
				}
				if settingsButton(c, "Remove", true).Clicked() {
					a.removeHost(host)
				}
			})
		}
	})
}

// hostSetupRow shows the server state and installs it on demand.
func (a *app) hostSetupRow(c *ui.Context, k *colors, host string) {
	busy := a.hostSetupBusy && a.hostSetupAddr == host
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
	case a.hostReady[host]:
		msg = "Server ready"
		col = k.TextMuted
	}
	row := ui.Row(c).Padding(0, 8, 6, 8).Gap(8).AlignItems(ui.Center)
	if a.settingsHits["Set Up Server"] {
		row.Background(k.PanelSel)
	}
	if a.settingsReveal == "Set Up Server" {
		row.ScrollIntoView()
	}
	row.Children(func() {
		if busy {
			ui.Spinner(c).Size(12, 12).Label("Setting up server")
		}
		ui.Text(c, msg).FontSize(11).TextColor(col).Grow(1).SingleLine().Ellipsis("…")
		if !busy {
			if settingsButton(c, "Set Up Server", false).Clicked() {
				a.setUpHost(host)
			}
		}
	})
}

func (a *app) hostEditRow(c *ui.Context, k *colors, host, label string) {
	ui.Column(c).Padding(8, 10).Gap(7).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			addr := ui.TextInputBase(c, &a.hostEditAddress).Grow(1).FontSize(12).
				Placeholder("me@devbox")
			name := ui.TextInputBase(c, &a.hostEditName).Width(100).FontSize(12).
				Placeholder("Name")
			if addr.Submitted() || name.Submitted() {
				a.saveHostEdit(host)
			}
			if settingsButton(c, "Cancel", false).Clicked() {
				a.hostEditAddr, a.settingsNotice = "", ""
				a.invalidate()
			}
			if settingsButton(c, "Save", true).Clicked() {
				a.saveHostEdit(host)
			}
		})
		ui.Text(c, "Editing "+label+". A new address takes effect the next time it connects.").
			FontSize(11).TextColor(k.TextFaint)
	})
}

func (a *app) saveHostEdit(host string) {
	addr := strings.TrimSpace(a.hostEditAddress)
	name := strings.TrimSpace(a.hostEditName)
	if addr == "" {
		a.settingsNotice = "An address is required."
		a.invalidate()
		return
	}
	for _, h := range a.hosts {
		if h.Address == addr && h.Address != host {
			a.settingsNotice = "That address is already in the list."
			a.invalidate()
			return
		}
	}
	for i := range a.hosts {
		if a.hosts[i].Address != host {
			continue
		}
		if name == "" {
			name = addr
		}
		a.hosts[i].Name, a.hosts[i].Address = name, addr
		if a.serverAlias == host {
			a.serverAlias = addr
		}
		if a.hostReady[host] {
			delete(a.hostReady, host)
			a.hostReady[addr] = true
		}
		if a.hostSetupAddr == host {
			a.hostSetupAddr = addr
		}
		for j := range a.remoteProjects {
			if a.remoteProjects[j].Host == host {
				a.remoteProjects[j].Host = addr
			}
		}
		a.hostEditAddr, a.settingsNotice = "", ""
		a.saveSettings()
		a.invalidate()
		return
	}
	a.hostEditAddr, a.settingsNotice = "", ""
	a.invalidate()
}

func (a *app) removeHost(address string) {
	for i, h := range a.hosts {
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
		a.hosts = append(a.hosts[:i], a.hosts[i+1:]...)
		projects := a.remoteProjects[:0]
		for _, p := range a.remoteProjects {
			if p.Host != address {
				projects = append(projects, p)
			}
		}
		a.remoteProjects = projects
		a.hostRemoveAddr = ""
		delete(a.hostReady, address)
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
		a.openRemotePicker(host)
	})
}
