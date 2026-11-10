package app

import (
	"errors"
	"strings"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

// openSSHPicker lists the hosts configured in Settings › Hosts. The app
// does not scan ~/.ssh/config: a host is added once, in Settings, and the
// system ssh still reads its own config for aliases, ProxyJump and keys.
func (a *app) openSSHPicker() {
	if !remoteSupported() {
		a.fail(errors.New("SSH terminals are unavailable in this build"))
		return
	}
	a.sshQuery = ""
	a.sshSel = 0
	a.sshList = ui.ListState{}
	a.closeModals()
	a.sshPick = true
	a.invalidate()
}

func (a *app) connectSSH(address string) {
	if address == "" {
		return
	}
	a.git.Open = false
	if a.tab() == nil {
		t := &tab{ID: nextID("t")}
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
	}
	tt, err := a.startTerminalSpec(termSpec{Host: address})
	if err != nil {
		a.fail(err)
		return
	}
	a.addTerminalPane(tt)
}

func (a *app) sshPicker(c *ui.Context, k *colors) {
	if !a.sshPick {
		return
	}
	q := strings.ToLower(strings.TrimSpace(a.sshQuery))
	var filtered []hostConfig
	for _, h := range a.hosts() {
		if q == "" || strings.Contains(strings.ToLower(h.Name+" "+h.Address), q) {
			filtered = append(filtered, h)
		}
	}
	a.sshSel = min(max(a.sshSel, 0), max(len(filtered)-1, 0))
	connected := ""
	commandDialog(c, k, &a.sshPick, 560, 460, true, func() {
		ui.Column(c).Children(func() {
			reveal := false
			ui.Row(c).Padding(13, 16).Gap(10).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Box(c).Size(30, 30).Radius(radius).Background(k.Hover).Center().Children(func() {
					ui.Icon(c, icons.Icon("server")).Size(15, 15).TextColor(k.TextMuted)
				})
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "Connect to Server").FontSize(14).FontWeight(650).TextColor(k.Text)
					ui.Text(c, "Hosts from Settings › Hosts").FontSize(11.5).TextColor(k.TextFaint)
				})
				ui.Spacer(c)
				if iconButton(c, k, "x", "Close", 24, 13).Clicked() {
					a.sshPick = false
				}
			})
			ui.Row(c).Padding(9, 13).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(14, 14).TextColor(k.TextFaint)
				in := ui.TextInputBase(c, &a.sshQuery).Grow(1).FontSize(13).
					Placeholder("Search hosts…").AutoFocus()
				if listNav(in, &a.sshSel, len(filtered), &reveal) {
					connected = filtered[a.sshSel].Address
				}
			})
			if len(a.hosts()) == 0 {
				ui.Column(c).Padding(20).Gap(10).Children(func() {
					ui.Text(c, "No servers configured").FontSize(13).FontWeight(600).TextColor(k.Text)
					ui.Text(c, "Add a server in Settings › Hosts, then come back here.").
						FontSize(11.5).TextColor(k.TextFaint)
					if ui.Button(c, "Open Settings › Hosts").Clicked() {
						a.sshPick = false
						a.later(func() {
							a.settings.Cat = settingsHosts
							a.openSettingsWindow()
						})
					}
				})
				return
			}
			a.sshList.Selected = &a.sshSel
			a.sshList.Key = func(i int) any { return filtered[i].Address }
			ui.List(c, &a.sshList, len(filtered), func(i int) {
				h := filtered[i]
				label := h.Name
				if label == "" {
					label = h.Address
				}
				row := ui.Row(c.Key(h.Address)).Height(40).Padding(0, 10).Gap(9).
					Radius(radius).Cursor(ui.CursorPointer).
					Label(label + " " + h.Address)
				fg, muted, icon := k.Text, k.TextFaint, k.TextMuted
				if i == a.sshSel {
					fg = c.Theme().AccentText
					muted, icon = fg.Alpha(0.9), fg.Alpha(0.9)
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					connected = h.Address
				}
				row.Children(func() {
					ui.Icon(c, icons.Icon("server")).Size(14, 14).TextColor(icon)
					ui.Text(c, label).FontSize(12.5).FontWeight(550).TextColor(fg).
						Shrink(0)
					ui.Text(c, h.Address).FontSize(11).TextColor(muted).
						SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
					if i == a.sshSel {
						ui.Spacer(c)
						ui.Text(c, "↵").FontSize(10).TextColor(muted)
					}
				})
			}).MaxHeight(340).Padding(6)
			if reveal {
				a.sshList.ScrollIntoView(a.sshSel)
			}
		})
	})
	if connected != "" {
		a.sshPick = false
		a.later(func() { a.connectSSH(connected) })
	}
}

// sshArgs connects to a configured host address with the system ssh, so
// ssh_config, the agent and ProxyJump keep working.
func sshArgs(host string) []string {
	return []string{"ssh", "-tt",
		"-o", "ServerAliveInterval=15", "-o", "ServerAliveCountMax=3",
		"--", host}
}
