package app

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"mojian/internal/icons"
	"mojian/internal/session"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	settingsGeneral = iota
	settingsAppearance
	settingsEditor
	settingsFonts
	settingsTerminal
	settingsShell
	settingsNotifications
	settingsKeyboard
	settingsHosts
	settingsGit
	settingsAdvanced
)

type settingsCat struct {
	title, detail, glyph string
}

var settingsCats = []settingsCat{
	{"General", "Startup and session behavior", "settings"},
	{"Appearance", "Theme and window look", "file-text"},
	{"Editor", "Type, writing and Markdown", "pencil"},
	{"Fonts", "Installed families and previews", "file-text"},
	{"Terminal", "Type, input and scrollback", "square-terminal"},
	{"Shell", "The program new local terminals run", "square-terminal"},
	{"Notifications", "When a terminal wants attention", "bell"},
	{"Keyboard", "Shortcuts this app uses", "command"},
	{"Hosts", "Servers, SSH hosts and projects", "server"},
	{"Git", "The working-tree page", "git-branch"},
	{"Advanced", "Data, server and version", "settings"},
}

// settingsWindowView is the whole content of the separate Settings
// window: a rail of categories on the left, the chosen page on the right.
func (a *app) settingsWindowView(c *ui.Context) {
	a.applyTheme(c)
	k := colorsOf(c)
	c.Root().Background(k.BgBottom)
	ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		paintBackground(p, r, k)
	}).Children(func() {
		a.settingsTitleBar(c, k)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.settingsRail(c, k)
			a.settingsBody(c, k)
		})
	})
	a.fontPicker(c, k)
}

func (a *app) settingsTitleBar(c *ui.Context, k *colors) {
	bar := c.TitleBar()
	dragW := bar.Left + 18
	if bar.Left == 0 {
		dragW = 30
	}
	ui.Row(c).Height(44).Padding(0, 14, 0, 0).Gap(12).
		AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Box(c).Width(dragW).Height(44).DragWindow().
			Cursor(ui.CursorGrab).Description("Drag Window")
		ui.Icon(c, brandMark(c)).Size(20, 20)
		ui.Text(c, "Settings").FontSize(13).FontWeight(600).TextColor(k.Text)
	})
}

func (a *app) settingsRail(c *ui.Context, k *colors) {
	ui.Column(c).Width(214).Shrink(0).FillHeight().Padding(10, 12).
		Gap(2).BorderWidth(0, 0, 0, 1).BorderColor(k.PanelBorder).Children(func() {
		ui.Row(c).Height(30).Margin(0, 2, 8, 2).Padding(0, 8).Gap(6).Radius(7).
			Background(k.Hover).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
			ui.TextInputBase(c, &a.settingsQuery).Grow(1).FontSize(12).
				Placeholder("Search settings…")
		})
		q := strings.ToLower(strings.TrimSpace(a.settingsQuery))
		found := false
		for i, cat := range settingsCats {
			if q != "" && !strings.Contains(strings.ToLower(cat.title+" "+cat.detail), q) {
				continue
			}
			found = true
			i, cat := i, cat
			row := ui.Row(c).Height(34).Padding(0, 9).Gap(9).Radius(7).
				Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(cat.title)
			if i == a.settingsCat {
				row.Background(k.PanelSel)
			} else if row.Hovered() {
				row.Background(k.Hover)
			}
			if row.Clicked() {
				a.settingsCat = i
				a.invalidate()
			}
			row.Children(func() {
				col := k.TextMuted
				if i == a.settingsCat {
					col = k.Text
				}
				ui.Icon(c, icons.Icon(cat.glyph)).Size(15, 15).TextColor(col)
				ui.Text(c, cat.title).FontSize(12.5).FontWeight(500).TextColor(col).
					SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
			})
		}
		if !found {
			ui.Text(c, "No settings match.").FontSize(11.5).TextColor(k.TextFaint).
				Padding(8, 10)
		}
	})
}

func (a *app) settingsBody(c *ui.Context, k *colors) {
	ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).FillHeight().Children(func() {
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).MaxWidth(680).FillWidth().AlignSelf(ui.Center).
				Padding(24, 30, 44, 30).Gap(24).Children(func() {
				switch a.settingsCat {
				case settingsGeneral:
					a.settingsGeneralPage(c, k)
				case settingsAppearance:
					a.settingsAppearancePage(c, k)
				case settingsEditor:
					a.settingsEditorPage(c, k)
				case settingsFonts:
					a.settingsFontsPage(c, k)
				case settingsTerminal:
					a.settingsTerminalPage(c, k)
				case settingsShell:
					a.settingsShellPage(c, k)
				case settingsNotifications:
					a.settingsNotificationsPage(c, k)
				case settingsKeyboard:
					a.settingsKeyboardPage(c, k)
				case settingsHosts:
					a.settingsHostsPage(c, k)
				case settingsGit:
					a.settingsGitPage(c, k)
				default:
					a.settingsAdvancedPage(c, k)
				}
			})
		})
	})
}

func (a *app) settingsSection(c *ui.Context, k *colors, title string, fn func()) {
	ui.Column(c).Gap(8).FillWidth().Children(func() {
		ui.Text(c, title).FontSize(10).FontWeight(650).
			TextColor(k.TextFaint).Padding(2, 2)
		panel := ui.Column(c).FillWidth().Background(k.Card).Border(1, k.CardBorder).
			Radius(12).Clip()
		panel.Children(fn)
	})
}

func (a *app) settingRow(c *ui.Context, k *colors, title, detail string, control func()) {
	ui.Row(c).MinHeight(56).Padding(0, 16).Gap(18).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(12.5).FontWeight(550).TextColor(k.Text)
			if detail != "" {
				ui.Text(c, detail).FontSize(11.5).TextColor(k.TextFaint).
					SingleLine().Ellipsis("…")
			}
		})
		ui.Row(c).Shrink(0).AlignItems(ui.Center).Children(control)
	})
}

func (a *app) settingDivider(c *ui.Context, k *colors) {
	ui.Divider(c).Margin(0, 16)
}

func (a *app) switchRow(c *ui.Context, k *colors, title, detail string, on *bool) {
	a.settingRow(c, k, title, detail, func() {
		sw := ui.Switch(c, on).Label(title)
		if sw.Changed() {
			a.saveSettings()
			a.invalidate()
		}
	})
}

func (a *app) settingsHostsPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "ADD HOST", func() {
		a.settingRow(c, k, "Address", "Anything ssh accepts: buildbox, me@devbox, or ssh://me@devbox:2222.", func() {
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				ui.TextInputBase(c, &a.hostDraftAddr).Width(170).FontSize(12).
					Placeholder("me@devbox")
				ui.TextInputBase(c, &a.hostDraftName).Width(90).FontSize(12).
					Placeholder("Name")
				if ui.PrimaryButton(c, "Add").Clicked() {
					a.addHost(a.hostDraftName, a.hostDraftAddr)
					a.hostDraftName, a.hostDraftAddr = "", ""
				}
			})
		})
	})
	a.settingsSection(c, k, "SERVERS", func() {
		a.hostSettingsRow(c, k, "", "Local", "This Mac", a.serverAlias == "")
		for _, h := range a.hosts {
			label := h.Name
			if label == "" {
				label = h.Address
			}
			a.hostSettingsRow(c, k, h.Address, label, h.Address, a.serverAlias == h.Address)
		}
		if len(a.hosts) == 0 {
			ui.Text(c, "No servers yet. Add an SSH host above.").
				FontSize(11.5).TextColor(k.TextFaint).Padding(16)
		}
	})
	if len(a.remoteProjects) > 0 {
		a.settingsSection(c, k, "RECENT REMOTE PROJECTS", func() {
			for i, p := range a.remoteProjects {
				i, p := i, p
				row := ui.Row(c).Height(40).Padding(0, 14).Gap(9).
					Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(p.Path)
				if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					a.later(func() {
						if a.settingsWin != nil {
							a.settingsWin.Close()
						}
						if err := a.openRemoteProject(p.Host, p.Path); err != nil {
							a.fail(err)
						}
					})
				}
				row.Children(func() {
					ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
					ui.Column(c).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, filepath.Base(p.Path)).FontSize(12.5).TextColor(k.Text).
							SingleLine().Ellipsis("…")
						ui.Text(c, a.hostLabel(p.Host)+":"+p.Path).FontSize(10.5).TextColor(k.TextFaint).
							SingleLine().Ellipsis("…")
					})
				})
				if i != len(a.remoteProjects)-1 {
					ui.Divider(c).Margin(0, 14)
				}
			}
		})
	}
	a.settingsSection(c, k, "ABOUT HOSTS", func() {
		ui.Text(c, "Mojian uploads a small per-user server to a host over SSH on first connect. Nothing is installed system-wide and no port is opened. Add hosts here first; only those appear in the server menu.").
			FontSize(11.5).TextColor(k.TextFaint).Padding(16)
	})
}

func (a *app) hostSettingsRow(c *ui.Context, k *colors, host, label, detail string, selected bool) {
	row := ui.Row(c).Height(52).Padding(0, 14).Gap(10).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Server " + label)
	if selected {
		row.Background(k.PanelSel)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		if host == "" {
			a.later(func() {
				a.saveNow()
				a.closeRemote()
				a.serverAlias = ""
				a.remotePath = ""
				a.remoteCache = ""
				a.saveSettings()
				if a.settingsWin != nil {
					a.settingsWin.Close()
				}
				if a.project != "" {
					a.openProject(a.project)
				}
			})
		} else {
			a.later(func() {
				if a.settingsWin != nil {
					a.settingsWin.Close()
				}
				a.openRemotePicker(host)
			})
		}
	}
	row.Children(func() {
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
}

func (a *app) settingsGeneralPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "STARTUP", func() {
		restore := !a.noRestoreLayout
		a.settingRow(c, k, "Restore tabs on launch",
			"Bring back the last project's tabs and panes.", func() {
				sw := ui.Switch(c, &restore).Label("Restore tabs on launch")
				if sw.Changed() {
					a.noRestoreLayout = !restore
					a.saveSettings()
					a.invalidate()
				}
			})
	})
	a.settingsSection(c, k, "PROJECTS", func() {
		a.aboutRow(c, k, "Current project", a.projectPath())
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Server", a.remoteName())
	})
}

func (a *app) settingsAppearancePage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "THEME", func() {
		a.settingRow(c, k, "Appearance", "Follow the system, or keep one look.", func() {
			ui.Row(c).Padding(3).Radius(9).Background(k.Track).Border(1, k.TrackBorder).
				Gap(2).Role(ui.RoleRadioGroup).Label("Theme").Children(func() {
				for i, label := range []string{"System", "Light", "Dark"} {
					i, label := i, label
					b := ui.Row(c).Padding(5, 12).Radius(7).Cursor(ui.CursorPointer).
						Role(ui.RoleRadio).Label(label).Selected(a.appearance == i)
					if a.appearance == i {
						b.Background(k.PanelSel)
					} else if b.Hovered() {
						b.Background(k.Hover)
					}
					if b.Clicked() {
						a.setAppearance(i)
						a.invalidate()
					}
					b.Children(func() {
						col := k.TextMuted
						if a.appearance == i {
							col = k.Text
						}
						ui.Text(c, label).FontSize(11.5).FontWeight(550).TextColor(col)
					})
				}
			})
		})
	})
	a.settingsSection(c, k, "TEXT", func() {
		a.fontSizeRow(c, k, "Editor text size", "The base size of source and prose.",
			&a.editorFontSize, 12.5, 9, 28, ".1f")
		a.settingDivider(c, k)
		a.settingRow(c, k, "Font family", "Choose the face used by editors and previews.", func() {
			b := ui.Row(c).Padding(5, 10).Radius(7).Gap(6).Cursor(ui.CursorPointer).
				Role(ui.RoleButton).Label("Editor font")
			if b.Hovered() {
				b.Background(k.Hover)
			}
			if b.Clicked() {
				a.fontPick, a.fontPickFor, a.fontSearch = true, 0, ""
			}
			label := a.editorFontFamily
			if label == "" {
				label = "JetBrains Mono · default"
			}
			b.Children(func() {
				ui.Text(c, label).FontSize(11.5).TextColor(k.Text).MaxWidth(190)
				ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
			})
		})
	})
}

func (a *app) settingsShellPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "LOGIN SHELL", func() {
		a.settingRow(c, k, "Shell", "Program new local terminals run; empty uses $SHELL.", func() {
			in := ui.TextInputBase(c, &a.shellPath).Width(220).FontSize(12.5).
				Placeholder("$SHELL")
			if in.Changed() {
				a.saveSettings()
				a.invalidate()
			}
		})
		a.settingDivider(c, k)
		ui.Text(c, "Remote projects use the shell on the server, not this path.").
			FontSize(11.5).TextColor(k.TextFaint).Padding(14, 16)
	})
}

func (a *app) settingsNotificationsPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "TERMINAL", func() {
		a.switchRow(c, k, "Notify on a bell",
			"Show a notification when a background terminal rings its bell.", &a.notifyBell)
		a.settingDivider(c, k)
		a.switchRow(c, k, "Notify when a terminal exits",
			"Show a notification when a background terminal finishes.", &a.notifyExit)
	})
}

func (a *app) settingsKeyboardPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "SHORTCUTS", func() {
		cmds := []*command{
			cmdOpenProject, cmdNewNote, cmdNewFolder, cmdNewTab, cmdSave,
			cmdPalette, cmdNotes, cmdTree, cmdTerminal, cmdGit, cmdSSH,
			cmdOverview, cmdSplitLeft, cmdSplitRight, cmdSplitUp, cmdSplitDown,
			cmdZoom, cmdEqualize,
			cmdClosePane, cmdCloseTab, cmdRenameTab, cmdNextTab, cmdPrevTab,
		}
		for i, cmd := range cmds {
			if cmd.keys == "" {
				continue
			}
			ui.Row(c).Height(34).Padding(0, 14).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, cmd.title).FontSize(12.5).TextColor(k.Text).Grow(1)
				ui.Text(c, cmd.keys).FontSize(11.5).TextColor(k.TextFaint).Font(noteFont.Family)
			})
			if i != len(cmds)-1 {
				ui.Divider(c).Margin(0, 14)
			}
		}
	})
	ui.Text(c, "Shortcuts are fixed for now; every command is also in the palette.").
		FontSize(11.5).TextColor(k.TextFaint)
}

func (a *app) settingsAdvancedPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "DATA", func() {
		a.aboutRow(c, k, "Settings", a.settingsPath)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Project cache", a.remoteCache)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Local sessions", session.SocketPath())
		if a.remote != nil {
			a.settingDivider(c, k)
			a.aboutRow(c, k, "Remote server", a.remote.host+" · "+a.remote.socket)
		}
	})
	a.settingsAboutPage(c, k)
}

func (a *app) settingsEditorPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "TYPE", func() {
		a.fontRow(c, k, "Editor font", "The family used by source and prose.", 0, a.editorFontFamily)
		a.settingDivider(c, k)
		a.fontSizeRow(c, k, "Font size", "In points, from 9 to 28.",
			&a.editorFontSize, 12.5, 9, 28, ".1f")
		a.settingDivider(c, k)
		a.fontSizeRow(c, k, "Line height", "Room for each line, as a multiple of the size.",
			&a.editorLineHeight, 1.55, 1.1, 2.2, ".2f")
		a.settingDivider(c, k)
		a.settingRow(c, k, "Tab size", "Columns for a tab and one level of indentation.", func() {
			for _, n := range []int{2, 4, 8} {
				n := n
				b := ui.Row(c).Padding(4, 9).Radius(6).Cursor(ui.CursorPointer).
					Role(ui.RoleRadio).Label(fmt.Sprintf("%d columns", n)).Selected(a.tabSize() == n)
				if a.tabSize() == n {
					b.Background(k.PanelSel)
				} else if b.Hovered() {
					b.Background(k.Hover)
				}
				if b.Clicked() {
					a.editorTabSize = n
					a.saveSettings()
					a.invalidate()
				}
				b.Children(func() {
					ui.Text(c, fmt.Sprintf("%d", n)).FontSize(11.5).TextColor(k.Text)
				})
			}
		})
	})
	a.settingsSection(c, k, "WRITING", func() {
		a.switchRow(c, k, "Line numbers",
			"Number the lines of code files; Markdown stays clean.", boolPtr(!a.hideLineNumbers))
		a.settingDivider(c, k)
		a.switchRow(c, k, "Soft wrap code",
			"Wrap long code lines at the pane's width.", &a.editorSoftWrap)
		a.settingDivider(c, k)
		a.switchRow(c, k, "Preview Markdown by default",
			"Open .md files rendered, with the Edit button to switch back.", &a.markdownPreview)
		a.settingDivider(c, k)
		a.switchRow(c, k, "Highlight active line",
			"Show a quiet tint behind the caret's line.", boolPtr(!a.noActiveLine))
		a.settingDivider(c, k)
		a.switchRow(c, k, "Indent guides",
			"Lines that show each level of indentation.", boolPtr(!a.noIndentGuides))
		a.settingDivider(c, k)
		a.switchRow(c, k, "Auto-close brackets",
			"Close quotes and brackets as you type.", boolPtr(!a.noAutoClose))
		a.settingDivider(c, k)
		a.switchRow(c, k, "Scroll past the last line",
			"Keep a little air below the end of the file.", boolPtr(!a.noScrollPastEnd))
	})
}

func (a *app) settingsFontsPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "CURRENT FONTS", func() {
		a.fontRow(c, k, "Editor font", "The family used by source and prose.", 0, a.editorFontFamily)
		a.settingDivider(c, k)
		a.fontRow(c, k, "Terminal font", "The family used by new shells.", 1, a.terminalFontFamily)
	})
	a.settingsSection(c, k, "INSTALLED FONTS", func() {
		ui.Row(c).Padding(10, 14).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
			for i, label := range []string{"Editor", "Terminal"} {
				i, label := i, label
				b := ui.Row(c).Padding(3, 8).Radius(5).Cursor(ui.CursorPointer).
					Role(ui.RoleRadio).Label("Apply to " + label).Selected(a.fontTarget == i)
				if a.fontTarget == i {
					b.Background(k.PanelSel)
				} else if b.Hovered() {
					b.Background(k.Hover)
				}
				if b.Clicked() {
					a.fontTarget = i
					a.invalidate()
				}
				b.Children(func() {
					ui.Text(c, label).FontSize(11).TextColor(k.Text)
				})
			}
			ui.Spacer(c)
			ui.Text(c, "Click a family to use it").FontSize(10.5).TextColor(k.TextFaint)
		})
		ui.Row(c).Padding(8, 10).Gap(7).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
			ui.TextInputBase(c, &a.fontSearch).Grow(1).FontSize(12).
				Placeholder("Search installed fonts…")
		})
		ui.Scroll(c).Height(240).Padding(6).Children(func() {
			shown := 0
			def := "JetBrains Mono · default"
			if a.fontTarget == 1 {
				def = "System monospace · default"
			}
			row := ui.Row(c).Key("inline-default").Height(33).Padding(0, 10).Gap(8).
				Radius(6).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(def)
			current := a.editorFontFamily
			if a.fontTarget == 1 {
				current = a.terminalFontFamily
			}
			if current == "" {
				row.Background(k.PanelSel)
			} else if row.Hovered() {
				row.Background(k.Hover)
			}
			if row.Clicked() {
				if a.fontTarget == 1 {
					a.terminalFontFamily = ""
				} else {
					a.editorFontFamily = ""
				}
				a.saveSettings()
				a.invalidate()
			}
			row.Children(func() { ui.Text(c, def).FontSize(12).TextColor(k.Text) })
			shown++
			q := strings.ToLower(strings.TrimSpace(a.fontSearch))
			for _, f := range a.installedFonts() {
				f := f
				if q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
					continue
				}
				shown++
				row := ui.Row(c).Key("inline:"+f.Name).Height(33).Padding(0, 10).
					Gap(8).Radius(6).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(f.Name)
				if f.Name == current {
					row.Background(k.PanelSel)
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					if a.fontTarget == 1 {
						a.terminalFontFamily = f.Name
					} else {
						a.editorFontFamily = f.Name
					}
					a.saveSettings()
					a.invalidate()
				}
				row.Children(func() {
					ui.Text(c, f.Name).Font(f.Name).FontSize(12).TextColor(k.Text).
						SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
					if f.Monospaced {
						ui.Text(c, "Mono").FontSize(9).FontWeight(650).
							TextColor(k.TextFaint).Padding(1, 4).Radius(4).
							Background(k.Hover).Shrink(0)
					}
				})
			}
			if shown == 0 {
				ui.Text(c, "No matching font").FontSize(12).TextColor(k.TextFaint).Padding(14)
			}
		})
	})
	a.settingsSection(c, k, "PREVIEW", func() {
		a.settingRow(c, k, "Editor", "The quick brown fox jumps over the lazy dog. 0123456789", func() {
			ui.Text(c, "The quick brown fox jumps over the lazy dog.").
				Font(a.editorFont().Family).FontSize(float32(a.fontSize())).
				TextColor(k.Text).MaxWidth(250)
		})
		a.settingDivider(c, k)
		a.settingRow(c, k, "Terminal", "The quick brown fox jumps over the lazy dog. 0123456789", func() {
			fam := a.terminalFontFamily
			if strings.TrimSpace(fam) == "" {
				fam = "JetBrains Mono, monospace"
			}
			ui.Text(c, "The quick brown fox jumps over the lazy dog.").
				Font(fam).FontSize(float32(a.termFontSize())).
				TextColor(k.Text).MaxWidth(250)
		})
	})
}

func (a *app) settingsTerminalPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "TYPE", func() {
		a.fontRow(c, k, "Terminal font", "The family used by new shells.", 1, a.terminalFontFamily)
		a.settingDivider(c, k)
		a.fontSizeRow(c, k, "Font size", "In points, from 9 to 28.",
			&a.terminalFontSize, 12.5, 9, 28, ".1f")
		a.settingDivider(c, k)
		a.fontSizeRow(c, k, "Line height", "Room for each terminal row.",
			&a.terminalLineHeight, 1, 0.8, 2, ".2f")
	})
	a.settingsSection(c, k, "INPUT", func() {
		a.switchRow(c, k, "Option as Alt",
			"Send Option to programs as Meta instead of typing accented letters.", &a.terminalOptionAsAlt)
		a.settingDivider(c, k)
		a.switchRow(c, k, "Copy on select",
			"Copy a selection to the clipboard as it ends.", &a.terminalCopySelect)
		a.settingDivider(c, k)
		a.switchRow(c, k, "Blink the cursor",
			"Let the terminal's cursor blink when the program allows it.", boolPtr(!a.noCursorBlink))
		a.settingDivider(c, k)
		a.settingRow(c, k, "Cursor shape", "How the terminal draws its caret.", func() {
			for i, label := range []string{"Block", "Bar", "Underline"} {
				i, label := i, label
				b := ui.Row(c).Padding(4, 9).Radius(6).Cursor(ui.CursorPointer).
					Role(ui.RoleRadio).Label(label).Selected(a.terminalCursor == i)
				if a.terminalCursor == i {
					b.Background(k.PanelSel)
				} else if b.Hovered() {
					b.Background(k.Hover)
				}
				if b.Clicked() {
					a.terminalCursor = i
					a.saveSettings()
					a.invalidate()
				}
				b.Children(func() {
					ui.Text(c, label).FontSize(11.5).TextColor(k.Text)
				})
			}
		})
	})
	a.settingsSection(c, k, "SCROLLBACK", func() {
		a.settingRow(c, k, "History size", "Megabytes kept above the screen.", func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("%d MB", a.termScrollback())).FontSize(11.5).
					TextColor(k.TextMuted).Width(56).TextAlign(ui.End).Shrink(0)
				v := float64(a.termScrollback())
				before := v
				ui.Slider(c, &v, 1, 100).Width(160).Label("History size")
				if v != before {
					a.terminalScrollback = int(v)
					a.saveSettings()
					a.invalidate()
				}
			})
		})
	})
}

func (a *app) settingsFilesPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "CURRENT PROJECT", func() {
		a.aboutRow(c, k, "Folder", a.projectPath())
		a.settingDivider(c, k)
		a.settingRow(c, k, "Reveal", "Show the project in Finder.", func() {
			b := ui.Row(c).Padding(4, 9).Radius(6).Cursor(ui.CursorPointer).
				Role(ui.RoleButton).Label("Reveal Project")
			if b.Hovered() {
				b.Background(k.Hover)
			}
			if b.Clicked() {
				a.revealVault()
			}
			b.Children(func() {
				ui.Text(c, "Reveal").FontSize(11.5).FontWeight(550).TextColor(k.Text)
			})
		})
	})
	recent := a.recentProjects()
	a.settingsSection(c, k, "RECENT PROJECTS", func() {
		if len(recent) == 0 {
			ui.Text(c, "No recent projects yet.").FontSize(12).TextColor(k.TextFaint).
				Padding(16)
			return
		}
		for i, p := range recent {
			if i >= 6 {
				break
			}
			p := p
			row := ui.Row(c).Height(38).Padding(0, 14).Gap(8).Cursor(ui.CursorPointer).
				Role(ui.RoleButton).Label("Open " + p)
			if row.Hovered() {
				row.Background(k.Hover)
			}
			if row.Clicked() {
				a.openProject(p)
				a.invalidate()
			}
			row.Children(func() {
				ui.Text(c, shortPath(p)).FontSize(12).TextColor(k.Text).
					SingleLine().Ellipsis("…")
			})
			if i != len(recent)-1 {
				ui.Divider(c).Margin(0, 14)
			}
		}
	})
}

func (a *app) settingsGitPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "CHANGES PAGE", func() {
		a.switchRow(c, k, "Refresh when the window returns",
			"Read the working tree again when the app regains focus.", boolPtr(!a.noGitAutoRefresh))
	})
	a.settingsSection(c, k, "ABOUT THE PAGE", func() {
		a.aboutRow(c, k, "Status", "Working tree against HEAD")
		a.settingDivider(c, k)
		ui.Text(c, "Open it with ⇧⌘G, or the Git button in the title bar.").
			FontSize(11.5).TextColor(k.TextFaint).Padding(14, 16)
	})
}

func (a *app) settingsAboutPage(c *ui.Context, k *colors) {
	version := mygo.App.Version()
	if version == "" {
		version = "0.1.0"
	}
	project := "No project"
	if a.store != nil {
		project = a.store.dir
	}
	a.settingsSection(c, k, "MOJIAN", func() {
		ui.Row(c).Padding(14, 16).AlignItems(ui.Center).Children(func() {
			wm := "wordmark"
			if c.Theme().Dark {
				wm = "wordmark-reversed"
			}
			ui.Icon(c, icons.Brand(wm)).Size(150, 36)
		})
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Version", version)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "System", runtime.GOOS+" · "+runtime.GOARCH)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Project", project)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Settings", a.settingsPath)
		a.settingDivider(c, k)
		a.aboutRow(c, k, "Fonts", fmt.Sprintf("%d installed families", len(a.installedFonts())))
	})
}

func (a *app) aboutRow(c *ui.Context, k *colors, title, value string) {
	ui.Row(c).MinHeight(44).Padding(0, 16).Gap(18).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(12.5).TextColor(k.Text).Grow(1)
		ui.Text(c, value).FontSize(11.5).TextColor(k.TextFaint).
			SingleLine().Ellipsis("…").MaxWidth(430).TextAlign(ui.End)
	})
}

func (a *app) fontRow(c *ui.Context, k *colors, title, detail string, which int, family string) {
	label := strings.TrimSpace(family)
	if label == "" {
		if which == 0 {
			label = "JetBrains Mono · default"
		} else {
			label = "System monospace · default"
		}
	}
	a.settingRow(c, k, title, detail, func() {
		b := ui.Row(c).Padding(5, 10).Radius(7).Gap(7).Cursor(ui.CursorPointer).
			Role(ui.RoleButton).Label(title + ": " + label)
		if b.Hovered() {
			b.Background(k.Hover)
		}
		if b.Clicked() {
			a.fontPick, a.fontPickFor, a.fontSearch = true, which, ""
		}
		b.Children(func() {
			ui.Text(c, label).FontSize(11.5).TextColor(k.Text).
				SingleLine().Ellipsis("…").MaxWidth(190)
			ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
		})
	})
}

func (a *app) fontSizeRow(c *ui.Context, k *colors, title, detail string, value *float64, def, lo, hi float64, format string) {
	if *value <= 0 {
		*value = def
	}
	changed := false
	a.settingRow(c, k, title, detail, func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%"+format, *value)).FontSize(11.5).TextColor(k.TextMuted).
				Width(44).TextAlign(ui.End).Shrink(0)
			before := *value
			ui.Slider(c, value, lo, hi).Width(160).Label(title)
			if *value != before {
				changed = true
			}
		})
	})
	if changed {
		a.saveSettings()
		a.invalidate()
	}
}

// editorFont is the effective editor face: the chosen family with the
// app's monospace fallbacks, at the chosen size.
func (a *app) editorFont() ui.Font {
	f := noteFont
	if fam := strings.TrimSpace(a.editorFontFamily); fam != "" {
		f.Family = fam + ", " + f.Family
	}
	f.Size = float32(a.fontSize())
	return f
}

func boolPtr(v bool) *bool { return &v }

// installedFonts lists the families the system and the app provide,
// loaded once when a font chooser first opens.
func (a *app) installedFonts() []ui.FontFamily {
	a.fontsOnce.Do(func() { a.fonts = ui.FontFamilies() })
	return a.fonts
}

// fontPicker is the searchable list of installed families.
func (a *app) fontPicker(c *ui.Context, k *colors) {
	if !a.fontPick {
		return
	}
	ui.DialogBase(c, &a.fontPick, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Width(500).MaxHeight(480).Radius(16).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).Clip()
		ui.Column(c).Children(func() {
			ui.Row(c).Padding(12, 14).Gap(9).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(15, 15).TextColor(k.TextFaint)
				ui.TextInputBase(c, &a.fontSearch).Grow(1).FontSize(13).
					Placeholder("Search installed fonts…").AutoFocus()
			})
			q := strings.ToLower(strings.TrimSpace(a.fontSearch))
			current := a.editorFontFamily
			if a.fontPickFor == 1 {
				current = a.terminalFontFamily
			}
			ui.Scroll(c).Height(350).Padding(6).Children(func() {
				shown := 0
				def := "JetBrains Mono · default"
				if a.fontPickFor == 1 {
					def = "System monospace · default"
				}
				row := ui.Row(c).Key("default").Height(34).Padding(0, 10).Gap(8).
					Radius(7).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(def)
				if current == "" {
					row.Background(k.PanelSel)
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					if a.fontPickFor == 1 {
						a.terminalFontFamily = ""
					} else {
						a.editorFontFamily = ""
					}
					a.fontPick = false
					a.saveSettings()
					a.invalidate()
				}
				row.Children(func() {
					ui.Text(c, def).FontSize(13).TextColor(k.Text)
				})
				shown++
				for _, f := range a.installedFonts() {
					f := f
					if q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
						continue
					}
					shown++
					row := ui.Row(c).Key(f.Name).Height(34).Padding(0, 10).Gap(8).
						Radius(7).Cursor(ui.CursorPointer).Role(ui.RoleButton).
						Label(f.Name)
					if f.Name == current {
						row.Background(k.PanelSel)
					} else if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() {
						if a.fontPickFor == 1 {
							a.terminalFontFamily = f.Name
						} else {
							a.editorFontFamily = f.Name
						}
						a.fontPick = false
						a.saveSettings()
						a.invalidate()
					}
					row.Children(func() {
						ui.Text(c, f.Name).Font(f.Name).FontSize(13).
							TextColor(k.Text).SingleLine().Ellipsis("…").
							Shrink(1).MinWidth(0)
						if f.Monospaced {
							ui.Text(c, "Mono").FontSize(9).FontWeight(650).
								TextColor(k.TextFaint).Padding(1, 4).Radius(4).
								Background(k.Hover).Shrink(0)
						}
					})
				}
				if shown == 0 {
					ui.Text(c, "No matching font").FontSize(12).TextColor(k.TextFaint).Padding(14)
				}
			})
		})
	})
}
