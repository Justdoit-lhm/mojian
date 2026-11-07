package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

// sshHost is a Host alias from ~/.ssh/config. Connections use the system
// ssh, so HostName, ProxyJump, IdentityFile and the agent all keep
// working.
type sshHost struct {
	Alias    string
	HostName string
	User     string
	Port     string
}

func (h sshHost) detail() string {
	var b strings.Builder
	if h.User != "" {
		b.WriteString(h.User)
		b.WriteByte('@')
	}
	if h.HostName != "" {
		b.WriteString(h.HostName)
	} else {
		b.WriteString(h.Alias)
	}
	if h.Port != "" {
		b.WriteString(":")
		b.WriteString(h.Port)
	}
	return b.String()
}

// sshConfigHosts reads the concrete Host aliases in ~/.ssh/config.
func sshConfigHosts() []sshHost {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		return nil
	}
	type block struct {
		aliases          []string
		host, user, port string
	}
	var blocks []block
	var cur *block
	flush := func() {
		if cur != nil && len(cur.aliases) > 0 {
			blocks = append(blocks, *cur)
		}
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		fs := strings.Fields(line)
		if len(fs) == 0 {
			continue
		}
		key := strings.ToLower(fs[0])
		switch key {
		case "host":
			flush()
			cur = &block{}
			for _, alias := range fs[1:] {
				if alias == "" || strings.ContainsAny(alias, "*?!") {
					continue
				}
				cur.aliases = append(cur.aliases, alias)
			}
		case "match":
			flush()
			cur = nil
		case "hostname", "user", "port":
			if cur == nil || len(fs) < 2 {
				continue
			}
			switch key {
			case "hostname":
				cur.host = fs[1]
			case "user":
				cur.user = fs[1]
			case "port":
				cur.port = fs[1]
			}
		}
	}
	flush()
	var out []sshHost
	seen := map[string]bool{}
	for _, b := range blocks {
		for _, alias := range b.aliases {
			if seen[alias] {
				continue
			}
			seen[alias] = true
			out = append(out, sshHost{Alias: alias, HostName: b.host, User: b.user, Port: b.port})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Alias) < strings.ToLower(out[j].Alias)
	})
	return out
}

func (a *app) openSSHPicker() {
	a.sshHosts = sshConfigHosts()
	a.sshQuery = ""
	a.sshSel = 0
	a.sshPick = true
	a.paletteOpen = false
	a.projectPicker = false
	a.invalidate()
}

func (a *app) connectSSH(alias string) {
	if alias == "" {
		return
	}
	a.git.Open = false
	if a.tab() == nil {
		t := &tab{ID: nextID("t")}
		a.tabs = append(a.tabs, t)
		a.active = len(a.tabs) - 1
	}
	tt, err := a.startTerminalSpec(termSpec{Host: alias})
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
	hosts := a.sshHosts
	q := strings.ToLower(strings.TrimSpace(a.sshQuery))
	var filtered []sshHost
	for _, h := range hosts {
		if q == "" || strings.Contains(strings.ToLower(h.Alias+" "+h.HostName+" "+h.User), q) {
			filtered = append(filtered, h)
		}
	}
	a.sshSel = min(max(a.sshSel, 0), max(len(filtered)-1, 0))
	ui.DialogBase(c, &a.sshPick, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop).Justify(ui.Start).Padding(78, 0, 0, 0)
		panel.Width(560).MaxHeight(460).Radius(radius).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).Clip()
		ui.Column(c).Children(func() {
			ui.Row(c).Padding(13, 16).Gap(10).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Box(c).Size(30, 30).Radius(radius).Background(k.Hover).Center().Children(func() {
					ui.Icon(c, icons.Icon("server")).Size(15, 15).TextColor(k.TextMuted)
				})
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "Connect to Server").FontSize(14).FontWeight(650).TextColor(k.Text)
					ui.Text(c, "Hosts from ~/.ssh/config").FontSize(11.5).TextColor(k.TextFaint)
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
				if in.Changed() {
					a.sshSel = 0
				}
				if in.Shortcut(0, ui.KeyDown) {
					a.sshSel = min(a.sshSel+1, len(filtered)-1)
				}
				if in.Shortcut(0, ui.KeyUp) {
					a.sshSel = max(a.sshSel-1, 0)
				}
				if in.Submitted() && len(filtered) > 0 {
					host := filtered[a.sshSel]
					a.sshPick = false
					a.later(func() { a.connectSSH(host.Alias) })
				}
			})
			if len(hosts) == 0 {
				ui.Column(c).Padding(20).Gap(6).Children(func() {
					ui.Text(c, "No hosts found").FontSize(13).FontWeight(600).TextColor(k.Text)
					ui.Text(c, "Add a Host block to ~/.ssh/config and reopen this list.").
						FontSize(11.5).TextColor(k.TextFaint)
				})
				return
			}
			ui.Scroll(c).Height(float32(min(max(len(filtered)*44+12, 80), 340))).Padding(6).Children(func() {
				for i, h := range filtered {
					i, h := i, h
					row := ui.Row(c).Key(h.Alias).Height(40).Padding(0, 10).Gap(9).
						Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).
						Label(h.Alias + " " + h.detail())
					if i == a.sshSel {
						row.Background(k.PanelSel)
					} else if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() {
						a.sshPick = false
						a.later(func() { a.connectSSH(h.Alias) })
					}
					row.Children(func() {
						ui.Icon(c, icons.Icon("server")).Size(14, 14).TextColor(k.TextMuted)
						ui.Text(c, h.Alias).FontSize(12.5).FontWeight(550).TextColor(k.Text).
							Shrink(0)
						ui.Text(c, h.detail()).FontSize(11).TextColor(k.TextFaint).
							SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
						if i == a.sshSel {
							ui.Spacer(c)
							ui.Text(c, "↵").FontSize(10).TextColor(k.TextFaint)
						}
					})
				}
			})
		})
	})
}

// sshArgs connects to a configured Host alias with the system ssh, so
// ssh_config, the agent and ProxyJump keep working.
func sshArgs(host string) []string {
	return []string{"ssh", "-tt",
		"-o", "ServerAliveInterval=15", "-o", "ServerAliveCountMax=3",
		"--", host}
}
