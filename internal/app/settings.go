package app

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
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
	settingsUpdates
)

type settingsCat struct {
	title, detail, glyph, words string
}

var settingsCats = []settingsCat{
	{"General", "Startup and projects", "settings", "startup restore tabs project server"},
	{"Appearance", "Theme and interface text", "file-text", "theme light dark appearance look color font size"},
	{"Editor", "Type, writing and Markdown", "pencil", "editor code font line height tab wrap markdown active line indent brackets scroll"},
	{"Fonts", "Families and previews", "file-text", "font family typeface mono installed preview"},
	{"Terminal", "Type, input and scrollback", "square-terminal", "terminal shell font cursor blink scrollback clipboard option alt"},
	{"Shell", "The program new terminals run", "square-terminal", "shell login command zsh bash fish"},
	{"Notifications", "When a terminal wants attention", "bell", "notify bell exit attention sound"},
	{"Keyboard", "Shortcuts this app uses", "command", "keyboard shortcut keys hotkeys keybindings commands"},
	{"Hosts", "SSH servers and recent projects", "server", "host server ssh remote project address"},
	{"Git", "The working-tree page", "git-branch", "git changes diff status refresh worktree"},
	{"Advanced", "Data and settings file", "settings", "advanced data settings file reset paths server"},
	{"Updates", "Signed releases and restart", "refresh-cw", "updates update release download install restart"},
}

// settingsEntry is one thing Settings' search finds: the page, the label
// its row or group shows, and words that find it besides.
type settingsEntry struct {
	cat   int
	label string
	words string
}

var settingsEntries = []settingsEntry{
	{settingsGeneral, "SESSIONS", "startup restore layout tabs panes launch"},
	{settingsGeneral, "Restore tabs on launch", "startup layout last project tabs panes restore"},
	{settingsGeneral, "PROJECT", "project folder server current"},
	{settingsGeneral, "Current project", "project folder path current server"},
	{settingsGeneral, "Server", "server ssh local remote host"},

	{settingsAppearance, "THEME", "theme appearance light dark system"},
	{settingsAppearance, "Appearance", "theme appearance light dark system look"},
	{settingsAppearance, "TEXT", "text font size editor interface"},
	{settingsAppearance, "Editor text size", "font size editor text bigger smaller zoom"},
	{settingsAppearance, "Editor font", "font family editor typeface monospace"},

	{settingsEditor, "TYPE", "type font text editor"},
	{settingsEditor, "Editor font", "font family editor typeface monospace"},
	{settingsEditor, "Font size", "font size editor text bigger smaller"},
	{settingsEditor, "Line height", "line height leading spacing editor"},
	{settingsEditor, "Tab size", "tab size indent columns spaces"},
	{settingsEditor, "WRITING", "writing editor markdown wrap"},
	{settingsEditor, "Line numbers", "line numbers gutter code markdown"},
	{settingsEditor, "Soft wrap code", "soft wrap long lines code width"},
	{settingsEditor, "Preview Markdown by default", "markdown preview render md default"},
	{settingsEditor, "Highlight active line", "active line caret highlight tint"},
	{settingsEditor, "Indent guides", "indent guides levels lines"},
	{settingsEditor, "Auto-close brackets", "brackets quotes auto close typing"},
	{settingsEditor, "Scroll past the last line", "scroll past end air bottom line"},

	{settingsFonts, "CURRENT FONTS", "fonts current editor terminal"},
	{settingsFonts, "Editor font", "font family editor typeface monospace"},
	{settingsFonts, "Terminal font", "font family terminal shell typeface monospace"},
	{settingsFonts, "INSTALLED FONTS", "installed fonts families search"},
	{settingsFonts, "Apply to", "apply editor terminal target"},
	{settingsFonts, "Search installed fonts", "search fonts families filter"},
	{settingsFonts, "PREVIEW", "preview fonts sample text"},
	{settingsFonts, "Editor", "editor preview font"},
	{settingsFonts, "Terminal", "terminal preview font"},

	{settingsTerminal, "TYPE", "terminal type font"},
	{settingsTerminal, "Terminal font", "font family terminal shell typeface monospace"},
	{settingsTerminal, "Font size", "font size terminal text"},
	{settingsTerminal, "Line height", "line height terminal rows spacing"},
	{settingsTerminal, "CURSOR", "cursor caret"},
	{settingsTerminal, "Cursor shape", "cursor shape block bar underline caret"},
	{settingsTerminal, "Blink the cursor", "cursor blink caret"},
	{settingsTerminal, "INPUT", "input keyboard mouse clipboard"},
	{settingsTerminal, "Option as Alt", "option alt meta keyboard macos"},
	{settingsTerminal, "Copy on select", "copy select clipboard selection"},
	{settingsTerminal, "SCROLLBACK", "scrollback history buffer memory"},
	{settingsTerminal, "History size", "scrollback history size memory megabytes mb"},

	{settingsShell, "LOGIN SHELL", "shell login terminal command"},
	{settingsShell, "Shell", "shell login program zsh bash fish environment"},

	{settingsNotifications, "TERMINAL", "notifications terminal attention"},
	{settingsNotifications, "Notify on a bell", "bell notify attention sound terminal"},
	{settingsNotifications, "Notify when a terminal exits", "exit notify finish done terminal"},

	{settingsKeyboard, "SHORTCUTS", "shortcuts keys keybindings hotkeys keyboard commands"},
	{settingsKeyboard, "PANE NAVIGATION", "pane navigation focus move keys"},
	{settingsKeyboard, "Move focus", "pane navigation focus arrow keys"},
	{settingsKeyboard, "Move the focused pane", "pane move reorder arrow keys"},
	{settingsKeyboard, "TAB SWITCHING", "tab switching next previous keys"},
	{settingsKeyboard, "Next tab", "tab next forward shortcut"},
	{settingsKeyboard, "Previous tab", "tab previous back shortcut"},
	{settingsKeyboard, "Show tab", "tab number position shortcut"},
	{settingsKeyboard, "Move to Trash", "trash delete remove file tree shortcut"},

	{settingsHosts, "ADD HOST", "add host ssh server address"},
	{settingsHosts, "Address", "host ssh server address"},
	{settingsHosts, "Name", "host name label"},
	{settingsHosts, "Add", "add host server"},
	{settingsHosts, "SERVERS", "servers hosts ssh local remote"},
	{settingsHosts, "Local", "local server this computer"},
	{settingsHosts, "RECENT REMOTE PROJECTS", "recent remote projects ssh"},
	{settingsHosts, "ABOUT HOSTS", "hosts remote server upload ssh"},

	{settingsGit, "CHANGES PAGE", "git changes diff status"},
	{settingsGit, "Refresh when the window returns", "git refresh focus auto worktree"},
	{settingsGit, "ABOUT THE PAGE", "git changes page status"},
	{settingsGit, "Status", "git status working tree head"},

	{settingsAdvanced, "DATA", "data paths settings"},
	{settingsAdvanced, "Settings file", "settings file json path"},
	{settingsAdvanced, "Remote project", "remote project path server"},
	{settingsAdvanced, "Local sessions", "local sessions socket terminal"},
	{settingsAdvanced, "Remote server", "remote server socket ssh"},
	{settingsAdvanced, "SETTINGS FILE", "settings file open reveal json"},
	{settingsAdvanced, "Reveal", "reveal show finder settings file"},
	{settingsAdvanced, "Open", "open settings file json editor"},
	{settingsAdvanced, "Reset all settings", "reset defaults restore settings preferences"},

	{settingsUpdates, "UPDATES", "updates release signed"},
	{settingsUpdates, "Check for Updates", "updates check release download"},
	{settingsUpdates, "Install", "updates install download"},
	{settingsUpdates, "Install Update", "updates install download"},
	{settingsUpdates, "Restart", "updates restart install apply"},
	{settingsUpdates, "Restart to Update", "updates restart install"},
	{settingsUpdates, "ABOUT THE UPDATES", "updates signed github verify"},

	{settingsAdvanced, "MOJIAN", "about mojian version credits"},
	{settingsAdvanced, "Version", "about version build"},
	{settingsAdvanced, "System", "about system os architecture"},
	{settingsAdvanced, "Project", "about project folder"},
	{settingsAdvanced, "Settings", "about settings path"},
	{settingsAdvanced, "Fonts", "about fonts installed"},
}

// settingsIndex is everything the search finds: the rows above and the
// command shortcuts, which the Keyboard page shows.
func (a *app) settingsIndex() []settingsEntry {
	cmds := settingsCommands()
	out := make([]settingsEntry, 0, len(settingsEntries)+len(cmds))
	out = append(out, settingsEntries...)
	for _, cmd := range cmds {
		if cmd.accel == "" || !commandAvailable(cmd) {
			continue
		}
		label := strings.TrimSuffix(cmd.title, "…")
		out = append(out, settingsEntry{settingsKeyboard, label,
			"shortcut keybinding keys " + cmd.id + " " + a.commandKeys(cmd)})
	}
	return out
}

// settingsWindowView is the whole Settings window: a searchable rail of
// pages on the left, the chosen page in a card on the right.
func (a *app) settingsWindowView(c *ui.Context) {
	a.runPending()
	a.applyTheme(c)
	k := colorsOf(c)
	c.Root().Background(k.BgBottom)
	a.prepareSettingsSearch()
	if a.settingsRecording == "" {
		c.OnShortcut(ui.Cmd, ui.KeyF, func() { a.settingsSearch.Focus(c) })
	}
	ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		paintBackground(p, r, k)
	}).Children(func() {
		a.settingsTitleBar(c, k)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.settingsRail(c, k)
			a.settingsContent(c, k)
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

// prepareSettingsSearch runs the query over every page and decides what
// to show: the pages that have finds, the rows to highlight on the page
// shown, and the first find to bring into view.
func (a *app) prepareSettingsSearch() {
	q := strings.TrimSpace(a.settingsQuery)
	a.settingsHits = nil
	a.settingsReveal = ""
	if q == "" {
		a.settingsFound = nil
		a.settingsRevealed = ""
		return
	}
	terms := strings.Fields(strings.ToLower(q))
	match := func(s string) bool {
		s = strings.ToLower(s)
		for _, t := range terms {
			if !strings.Contains(s, t) {
				return false
			}
		}
		return true
	}
	found := map[int][]string{}
	for i, cat := range settingsCats {
		if !settingsCatAvailable(i) {
			continue
		}
		if match(cat.title + " " + cat.detail + " " + cat.words) {
			found[i] = []string{}
		}
	}
	for _, e := range a.settingsIndex() {
		if !settingsCatAvailable(e.cat) || !match(e.label+" "+e.words) {
			continue
		}
		found[e.cat] = appendUnique(found[e.cat], e.label)
	}
	a.settingsFound = found
	if len(found) == 0 {
		return
	}
	if _, ok := found[a.settingsCat]; !ok || !settingsCatAvailable(a.settingsCat) {
		for i := range settingsCats {
			if _, ok := found[i]; ok && settingsCatAvailable(i) {
				a.settingsCat = i
				break
			}
		}
	}
	labels := found[a.settingsCat]
	if len(labels) > 0 {
		key := q + "\x00" + strconv.Itoa(a.settingsCat)
		if a.settingsRevealed != key {
			a.settingsRevealed, a.settingsReveal = key, labels[0]
		}
	}
	hits := make(map[string]bool, len(labels))
	for _, l := range labels {
		hits[l] = true
	}
	a.settingsHits = hits
}

func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}

func (a *app) settingsRail(c *ui.Context, k *colors) {
	ui.Column(c).Width(214).Shrink(0).FillHeight().Padding(10, 12).
		Gap(2).BorderWidth(0, 0, 0, 1).BorderColor(k.PanelBorder).Children(func() {
		ui.Row(c).Height(30).Margin(0, 2, 8, 2).Padding(0, 8).Gap(6).Radius(radius).
			Background(k.Hover).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
			search := ui.TextInputBase(c, &a.settingsQuery).Bind(&a.settingsSearch).Grow(1).
				FontSize(12).Placeholder("Search settings…").Label("Search settings")
			search.OnShortcut(0, ui.KeyEscape, func() {
				a.settingsQuery = ""
				a.settingsSearch.Focus(c)
				a.invalidate()
			})
			if search.Changed() {
				a.invalidate()
			}
		})
		for i, cat := range settingsCats {
			if !settingsCatAvailable(i) {
				continue
			}
			if a.settingsFound != nil {
				if _, ok := a.settingsFound[i]; !ok {
					continue
				}
			}
			a.settingsNavRow(c, k, i, cat)
		}
		if a.settingsFound != nil && len(a.settingsFound) == 0 {
			ui.Text(c, "No settings match.").FontSize(11.5).TextColor(k.TextFaint).
				Padding(8, 10)
		}
	})
}

func (a *app) settingsNavRow(c *ui.Context, k *colors, i int, cat settingsCat) {
	sel := i == a.settingsCat
	row := ui.Row(c).Key("settings-nav-"+strconv.Itoa(i)).Height(34).Padding(0, 9).
		Gap(9).Radius(radius).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(cat.title)
	switch {
	case sel:
		row.Background(k.PanelSel)
	case row.Hovered():
		row.Background(k.Hover)
	}
	if row.Clicked() && !sel {
		a.settingsCat = i
		a.stopRecording()
		a.settingsNotice = ""
		a.settingsResetAsk = false
		a.settingsScrollTop = true
		a.invalidate()
	}
	row.Children(func() {
		col := k.TextMuted
		if sel {
			col = k.Text
		}
		ui.Icon(c, icons.Icon(cat.glyph)).Size(15, 15).TextColor(col)
		ui.Text(c, cat.title).FontSize(12.5).FontWeight(500).TextColor(col).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
	})
}

func (a *app) settingsContent(c *ui.Context, k *colors) {
	if a.settingsScrollTop {
		a.settingsScroll.Y = 0
		a.settingsScrollTop = false
	}
	ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).FillHeight().Children(func() {
		ui.Scroll(c).Grow(1).MinHeight(0).TrackScroll(&a.settingsScroll).Children(func() {
			ui.Column(c).MaxWidth(680).FillWidth().AlignSelf(ui.Center).
				Padding(24, 30, 44, 30).Gap(24).Children(func() {
				cat := a.settingsCat
				if !settingsCatAvailable(cat) {
					cat = settingsGeneral
				}
				if a.settingsFound != nil && len(a.settingsFound) == 0 {
					a.settingsNoResults(c, k)
					return
				}
				a.settingsPage(c, k, cat)
			})
		})
	})
}

func (a *app) settingsNoResults(c *ui.Context, k *colors) {
	ui.Column(c).Gap(10).Padding(24, 0).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.Icon("search")).Size(22, 22).TextColor(k.TextFaint)
		ui.Text(c, "No settings match “"+a.settingsQuery+"”.").FontSize(12.5).
			TextColor(k.TextFaint).TextAlign(ui.Center)
	})
}

func (a *app) settingsPage(c *ui.Context, k *colors, cat int) {
	switch cat {
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
	case settingsUpdates:
		a.settingsUpdatesPage(c, k)
	default:
		a.settingsAdvancedPage(c, k)
	}
}

func (a *app) settingsSection(c *ui.Context, k *colors, title string, fn func()) {
	a.settingsRows = 0
	ui.Column(c).Gap(8).FillWidth().Children(func() {
		if title != "" {
			head := ui.Row(c).Key("group-"+title).Padding(2, 2).Radius(5)
			if a.settingsHits[title] {
				head.Background(k.PanelSel)
			}
			if a.settingsReveal == title {
				head.ScrollIntoView()
			}
			head.Children(func() {
				ui.Text(c, title).FontSize(10).FontWeight(650).TextColor(k.TextFaint)
			})
		}
		panel := ui.Column(c).FillWidth().Background(k.Card).Border(1, k.CardBorder).
			Radius(radius).Clip()
		panel.Children(fn)
	})
}

func (a *app) settingRow(c *ui.Context, k *colors, title, detail string, control func()) {
	a.settingsRows++
	if a.settingsRows > 1 {
		ui.Divider(c).Margin(0, 16)
	}
	row := ui.Row(c).Key("row-"+title+"-"+strconv.Itoa(a.settingsRows)).
		MinHeight(56).Padding(0, 16).Gap(18).AlignItems(ui.Center).Radius(6)
	if a.settingsHits[title] {
		row.Background(k.PanelSel)
	}
	if a.settingsReveal == title {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(12.5).FontWeight(550).TextColor(k.Text)
			if detail != "" {
				ui.Text(c, detail).FontSize(11.5).TextColor(k.TextFaint).LineHeight(1.35)
			}
		})
		if control != nil {
			ui.Row(c).Shrink(0).Gap(8).AlignItems(ui.Center).Children(control)
		}
	})
}

// settingsSwitchRow is a row with a switch, whose on and off write the
// setting behind it: the inverse booleans the file keeps are read and
// written here, not bound to a temporary.
func (a *app) settingsSwitchRow(c *ui.Context, k *colors, title, detail string, get func() bool, set func(bool)) {
	a.settingRow(c, k, title, detail, func() {
		on := get()
		sw := ui.Switch(c, &on).Label(title + " toggle")
		if sw.Changed() {
			set(on)
			a.saveSettings()
			a.invalidate()
		}
	})
}

// settingsChoiceRow is a row with a segmented control, whose index is
// value's and whose choice is set's.
func (a *app) settingsChoiceRow(c *ui.Context, k *colors, key, title string, value func() int, set func(int), labels []string) {
	a.settingRow(c, k, title, "", func() {
		current := value()
		ui.Row(c).Padding(3).Radius(radius).Background(k.Track).Border(1, k.TrackBorder).
			Gap(2).Role(ui.RoleRadioGroup).Label(title).Children(func() {
			for n, label := range labels {
				n, label := n, label
				b := ui.Row(c).Key(key+"-"+strconv.Itoa(n)).Padding(5, 12).
					Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleRadio).
					Label(label).Selected(n == current)
				switch {
				case n == current:
					b.Background(k.PanelSel)
				case b.Hovered():
					b.Background(k.Hover)
				}
				if b.Clicked() {
					set(n)
					a.saveSettings()
					a.invalidate()
				}
				b.Children(func() {
					col := k.TextMuted
					if n == current {
						col = k.Text
					}
					ui.Text(c, label).FontSize(11.5).FontWeight(550).TextColor(col)
				})
			}
		})
	})
}

func (a *app) settingsSliderRow(c *ui.Context, k *colors, title, detail string, value func() float64, set func(float64), lo, hi float64, format string) {
	a.settingRow(c, k, title, detail, func() {
		v := value()
		before := v
		ui.Text(c, fmt.Sprintf(format, v)).FontSize(11.5).TextColor(k.TextMuted).
			Width(44).TextAlign(ui.End).Shrink(0)
		ui.Slider(c, &v, lo, hi).Width(160).Label(title)
		if v != before {
			set(v)
			a.saveSettings()
			a.invalidate()
		}
	})
}

func (a *app) settingsInfoRow(c *ui.Context, k *colors, title, value string) {
	a.settingsRows++
	if a.settingsRows > 1 {
		ui.Divider(c).Margin(0, 16)
	}
	row := ui.Row(c).Key("info-"+title).MinHeight(44).Padding(0, 16).Gap(18).
		AlignItems(ui.Center).Radius(6)
	if a.settingsHits[title] {
		row.Background(k.PanelSel)
	}
	if a.settingsReveal == title {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Text(c, title).FontSize(12.5).TextColor(k.Text).Shrink(0)
		ui.Spacer(c)
		ui.Text(c, value).FontSize(11.5).TextColor(k.TextFaint).SingleLine().
			Ellipsis("…").MaxWidth(280).Shrink(0).TextAlign(ui.End)
	})
}

func (a *app) settingsNote(c *ui.Context, k *colors, text string) {
	ui.Text(c, text).FontSize(11.5).TextColor(k.TextFaint).LineHeight(1.4).Padding(0, 2)
}

func (a *app) settingsNoticeLine(c *ui.Context, k *colors) {
	if a.settingsNotice == "" {
		return
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 2).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).Shrink(0)
		ui.Text(c, a.settingsNotice).FontSize(11.5).TextColor(k.Attention).Grow(1).MinWidth(0)
	})
}

func settingsButton(c *ui.Context, key, label string, primary bool) ui.Element {
	if primary {
		return ui.PrimaryButton(c, label)
	}
	return ui.Button(c, label)
}

func (a *app) settingsGeneralPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "SESSIONS", func() {
		a.settingsSwitchRow(c, k, "Restore tabs on launch",
			"Bring back the last project's tabs and panes.",
			func() bool { return !a.noRestoreLayout },
			func(v bool) { a.noRestoreLayout = !v })
	})
	a.settingsSection(c, k, "PROJECT", func() {
		a.settingsInfoRow(c, k, "Current project", a.projectPath())
		a.settingsInfoRow(c, k, "Server", a.remoteName())
	})
}

func (a *app) settingsAppearancePage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "THEME", func() {
		a.settingsChoiceRow(c, k, "appearance", "Appearance",
			func() int { return a.appearance }, func(i int) { a.setAppearance(i) },
			[]string{"System", "Light", "Dark"})
	})
	a.settingsSection(c, k, "TEXT", func() {
		a.settingsSliderRow(c, k, "Editor text size", "The base size of source and prose.",
			func() float64 { return a.fontSize() }, func(v float64) { a.editorFontSize = v },
			9, 28, "%.1f pt")
		a.settingRow(c, k, "Editor font", "The face used by editors and previews.", func() {
			label := strings.TrimSpace(a.editorFontFamily)
			if label == "" {
				label = "JetBrains Mono · default"
			}
			b := ui.ButtonBase(c.Key("pick-editor-font")).Padding(5, 10).Radius(radius).Gap(7).
				Label("Editor font: " + label)
			if b.Hovered() {
				b.Background(k.Pressed)
			}
			if b.Clicked() {
				a.fontPick, a.fontPickFor, a.fontSearch = true, 0, ""
			}
			b.Children(func() {
				ui.Text(c, label).FontSize(11.5).TextColor(k.Text).SingleLine().
					Ellipsis("…").MaxWidth(190)
				ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
			})
		})
	})
}

func (a *app) settingsEditorPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "TYPE", func() {
		a.fontRow(c, k, "Editor font", "The family used by source and prose.", 0, a.editorFontFamily)
		a.fontSizeRow(c, k, "Font size", "In points, from 9 to 28.",
			&a.editorFontSize, 12.5, 9, 28, ".1f")
		a.fontSizeRow(c, k, "Line height", "Room for each line, as a multiple of the size.",
			&a.editorLineHeight, 1.55, 1.1, 2.2, ".2f")
		a.settingsChoiceRow(c, k, "tab-size", "Tab size",
			func() int {
				switch a.tabSize() {
				case 2:
					return 0
				case 8:
					return 2
				}
				return 1
			}, func(i int) { a.editorTabSize = []int{2, 4, 8}[i] }, []string{"2", "4", "8"})
	})
	a.settingsSection(c, k, "WRITING", func() {
		a.settingsSwitchRow(c, k, "Line numbers",
			"Number the lines of code files; Markdown stays clean.",
			func() bool { return !a.hideLineNumbers }, func(v bool) { a.hideLineNumbers = !v })
		a.settingsSwitchRow(c, k, "Soft wrap code",
			"Wrap long code lines at the pane's width.",
			func() bool { return a.editorSoftWrap }, func(v bool) { a.editorSoftWrap = v })
		a.settingsSwitchRow(c, k, "Preview Markdown by default",
			"Open .md files rendered, with the Edit button to switch back.",
			func() bool { return a.markdownPreview }, func(v bool) { a.markdownPreview = v })
		a.settingsSwitchRow(c, k, "Highlight active line",
			"Show a quiet tint behind the caret's line.",
			func() bool { return !a.noActiveLine }, func(v bool) { a.noActiveLine = !v })
		a.settingsSwitchRow(c, k, "Indent guides",
			"Lines that show each level of indentation.",
			func() bool { return !a.noIndentGuides }, func(v bool) { a.noIndentGuides = !v })
		a.settingsSwitchRow(c, k, "Auto-close brackets",
			"Close quotes and brackets as you type.",
			func() bool { return !a.noAutoClose }, func(v bool) { a.noAutoClose = !v })
		a.settingsSwitchRow(c, k, "Scroll past the last line",
			"Keep a little air below the end of the file.",
			func() bool { return !a.noScrollPastEnd }, func(v bool) { a.noScrollPastEnd = !v })
	})
}

func (a *app) settingsFontsPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "CURRENT FONTS", func() {
		a.fontRow(c, k, "Editor font", "The family used by source and prose.", 0, a.editorFontFamily)
		a.fontRow(c, k, "Terminal font", "The family used by new shells.", 1, a.terminalFontFamily)
	})
	a.settingsSection(c, k, "INSTALLED FONTS", func() {
		ui.Row(c).Padding(8, 10).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Apply to").FontSize(12).TextColor(k.TextMuted).Shrink(0)
			a.settingsFontTargetChoice(c, k)
			if w, _ := c.Size(); w >= 820 {
				ui.Spacer(c)
				ui.Text(c, "Click a family to use it").FontSize(11).TextColor(k.TextFaint)
			}
		})
		ui.Box(c).Height(1).Shrink(0).Background(k.PanelBorder)
		ui.Row(c).Padding(6, 8).Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
			in := ui.TextInputBase(c, &a.fontSearch).Grow(1).FontSize(12).
				Placeholder("Search installed fonts…")
			if in.Changed() {
				a.invalidate()
			}
		})
		ui.Scroll(c).Height(220).Padding(4).Children(func() {
			a.fontsList(c, k)
		})
	})
	a.settingsSection(c, k, "PREVIEW", func() {
		a.settingRow(c, k, "Editor", "The quick brown fox jumps over the lazy dog. 0123456789", func() {
			ui.Text(c, "The quick brown fox jumps over the lazy dog.").
				Font(a.editorFont().Family).FontSize(float32(a.fontSize())).
				TextColor(k.Text).MaxWidth(250)
		})
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

func (a *app) settingsFontTargetChoice(c *ui.Context, k *colors) {
	labels := []string{"Editor", "Terminal"}
	current := a.fontTarget
	ui.Row(c).Padding(3).Radius(radius).Background(k.Track).Border(1, k.TrackBorder).
		Gap(2).Role(ui.RoleRadioGroup).Label("Apply to").Children(func() {
		for n, label := range labels {
			n, label := n, label
			b := ui.Row(c).Key("font-target-"+strconv.Itoa(n)).Padding(3, 8).
				Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleRadio).
				Label(label).Selected(n == current)
			switch {
			case n == current:
				b.Background(k.PanelSel)
			case b.Hovered():
				b.Background(k.Hover)
			}
			if b.Clicked() {
				a.fontTarget = n
				a.invalidate()
			}
			b.Children(func() {
				col := k.TextMuted
				if n == current {
					col = k.Text
				}
				ui.Text(c, label).FontSize(11).FontWeight(550).TextColor(col)
			})
		}
	})
}

func (a *app) fontsList(c *ui.Context, k *colors) {
	q := strings.ToLower(strings.TrimSpace(a.fontSearch))
	shown := 0
	def := "JetBrains Mono · default"
	if a.fontTarget == 1 {
		def = "System monospace · default"
	}
	current := a.editorFontFamily
	if a.fontTarget == 1 {
		current = a.terminalFontFamily
	}
	row := ui.Row(c).Key("font-default").Height(33).Padding(0, 10).Gap(8).
		Radius(6).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(def)
	switch {
	case current == "":
		row.Background(k.PanelSel)
	case row.Hovered():
		row.Background(k.Pressed)
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
	for _, f := range a.installedFonts() {
		if q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
			continue
		}
		f := f
		shown++
		row := ui.Row(c).Key("font-"+f.Name).Height(33).Padding(0, 10).Gap(8).
			Radius(6).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(f.Name)
		switch {
		case f.Name == current:
			row.Background(k.PanelSel)
		case row.Hovered():
			row.Background(k.Pressed)
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
}

func (a *app) settingsTerminalPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "TYPE", func() {
		a.fontRow(c, k, "Terminal font", "The family used by new shells.", 1, a.terminalFontFamily)
		a.fontSizeRow(c, k, "Font size", "In points, from 9 to 28.",
			&a.terminalFontSize, 12.5, 9, 28, ".1f")
		a.fontSizeRow(c, k, "Line height", "Room for each terminal row.",
			&a.terminalLineHeight, 1, 0.8, 2, ".2f")
	})
	a.settingsSection(c, k, "CURSOR", func() {
		a.settingsChoiceRow(c, k, "cursor-shape", "Cursor shape",
			func() int { return a.terminalCursor }, func(i int) { a.terminalCursor = i },
			[]string{"Block", "Bar", "Underline"})
		a.settingsSwitchRow(c, k, "Blink the cursor",
			"Let the terminal's cursor blink when the program allows it.",
			func() bool { return !a.noCursorBlink }, func(v bool) { a.noCursorBlink = !v })
	})
	a.settingsSection(c, k, "INPUT", func() {
		a.settingsSwitchRow(c, k, "Option as Alt",
			"Send Option to programs as Meta instead of typing accented letters.",
			func() bool { return a.terminalOptionAsAlt }, func(v bool) { a.terminalOptionAsAlt = v })
		a.settingsSwitchRow(c, k, "Copy on select",
			"Copy a selection to the clipboard as it ends.",
			func() bool { return a.terminalCopySelect }, func(v bool) { a.terminalCopySelect = v })
	})
	a.settingsSection(c, k, "SCROLLBACK", func() {
		a.settingsSliderRow(c, k, "History size", "Megabytes kept above the screen.",
			func() float64 { return float64(a.termScrollback()) },
			func(v float64) { a.terminalScrollback = int(v) }, 1, 100, "%.0f MB")
	})
}

func (a *app) settingsShellPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "LOGIN SHELL", func() {
		a.settingRow(c, k, "Shell", "Program new local terminals run; empty uses $SHELL.", func() {
			in := ui.TextInputBase(c, &a.shellPath).Width(230).FontSize(12).
				Placeholder("$SHELL")
			if in.Changed() {
				a.saveSettings()
				a.invalidate()
			}
		})
	})
	a.settingsNote(c, k, "Remote projects use the shell on the server, not this path.")
}

func (a *app) settingsNotificationsPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "TERMINAL", func() {
		a.settingsSwitchRow(c, k, "Notify on a bell",
			"Show a notification when a background terminal rings its bell.",
			func() bool { return a.notifyBell }, func(v bool) { a.notifyBell = v })
		a.settingsSwitchRow(c, k, "Notify when a terminal exits",
			"Show a notification when a background terminal finishes.",
			func() bool { return a.notifyExit }, func(v bool) { a.notifyExit = v })
	})
}

func (a *app) settingsKeyboardPage(c *ui.Context, k *colors) {
	a.settingsNoticeLine(c, k)
	q := strings.ToLower(strings.TrimSpace(a.settingsKeyFilter))
	shown := 0
	a.settingsSection(c, k, "SHORTCUTS", func() {
		ui.Row(c).Padding(10, 0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1).Height(30).Padding(0, 8).Gap(6).Radius(radius).
				Background(k.Hover).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
				in := ui.TextInputBase(c, &a.settingsKeyFilter).Grow(1).FontSize(12).
					Placeholder("Filter shortcuts…")
				if in.Changed() {
					a.invalidate()
				}
			})
			if len(a.shortcutKeys) > 0 {
				if settingsButton(c, "reset-shortcuts", "Restore Defaults", false).Clicked() {
					a.stopRecording()
					a.shortcutKeys = nil
					a.settingsNotice = ""
					a.saveSettings()
					a.refreshMenu()
					a.invalidate()
				}
			}
		})
		ui.Divider(c).Margin(0, 16)
		for _, cmd := range settingsCommands() {
			if cmd.accel == "" || !commandAvailable(cmd) {
				continue
			}
			label := strings.TrimSuffix(cmd.title, "…")
			keys := a.commandKeys(cmd)
			if q != "" && !strings.Contains(strings.ToLower(label+" "+keys+" "+cmd.id), q) {
				continue
			}
			if a.settingsHits != nil && !a.settingsHits[label] {
				continue
			}
			a.shortcutRow(c, k, cmd)
			shown++
		}
		if shown == 0 {
			ui.Text(c, "No shortcuts match.").FontSize(12).TextColor(k.TextFaint).
				Padding(14, 0)
		}
	})
	a.settingsSection(c, k, "PANE NAVIGATION", func() {
		a.settingsFixedKeys(c, k, "Move focus", nativeKeys("⌥←  ⌥→  ⌥↑  ⌥↓"),
			"Focus the neighboring pane.")
		a.settingsFixedKeys(c, k, "Move the focused pane", nativeKeys("⌥⇧←  ⌥⇧→  ⌥⇧↑  ⌥⇧↓"),
			"Put it on the neighbor's edge.")
	})
	a.settingsSection(c, k, "TAB SWITCHING", func() {
		a.settingsFixedKeys(c, k, "Next tab", nativeKeys("⌃⇥"), "Go to the tab on the right.")
		a.settingsFixedKeys(c, k, "Previous tab", nativeKeys("⌃⇧⇥"), "Go to the tab on the left.")
		a.settingsFixedKeys(c, k, "Show tab", nativeKeys("⌘1…⌘9"), "Go to a tab by its position.")
		a.settingsFixedKeys(c, k, "Move to Trash", nativeKeys("⌘⌫"), "On a file in the file tree.")
	})
	a.settingsNote(c, k, "A key used twice is taken from the command that had it. Escape cancels; Backspace unbinds.")
}

func (a *app) settingsFixedKeys(c *ui.Context, k *colors, title, keys, detail string) {
	a.settingRow(c, k, title, detail, func() {
		ui.Text(c, keys).FontSize(11.5).TextColor(k.TextFaint).Font(noteFont.Family)
	})
}

// shortcutRow is a command's shortcut: its keys, a reset when they are
// not the default, and a chip that records the keys typed for it.
func (a *app) shortcutRow(c *ui.Context, k *colors, cmd *command) {
	label := strings.TrimSuffix(cmd.title, "…")
	keys := a.commandKeys(cmd)
	custom := false
	if v, ok := a.shortcutKeys[cmd.id]; ok && !strings.EqualFold(v, cmd.accel) {
		custom = true
	}
	recording := a.settingsRecording == cmd.id
	a.settingRow(c, k, label, "", func() {
		if custom {
			back := ui.ButtonBase(c.Key("default-"+cmd.id)).Size(24, 24).Radius(6).
				Center().TextColor(k.TextFaint).Label("Use the default shortcut").
				Tooltip("Use " + nativeKeys(cmd.keys))
			if back.Hovered() {
				back.Background(k.Pressed).TextColor(k.Text)
			}
			back.Children(func() { ui.Icon(c, icons.Icon("refresh-cw")).Size(12, 12) })
			if back.Clicked() {
				delete(a.shortcutKeys, cmd.id)
				if len(a.shortcutKeys) == 0 {
					a.shortcutKeys = nil
				}
				a.settingsNotice = ""
				a.saveSettings()
				a.refreshMenu()
				a.invalidate()
			}
		}
		chip := ui.ButtonBase(c.Key("keys-"+cmd.id)).Height(26).MinWidth(92).
			Padding(0, 10).Radius(radius).Center().Background(k.Hover).
			Label("Shortcut for " + label)
		switch {
		case recording:
			chip.Background(k.PanelSel)
		case chip.Hovered():
			chip.Background(k.Pressed)
		}
		if chip.Clicked() {
			if recording {
				a.stopRecording()
			} else {
				a.settingsRecording = cmd.id
				a.settingsRecordSeen = false
				a.settingsNotice = ""
				a.settingsRecordRef.Focus(c)
				a.refreshMenu()
				a.invalidate()
			}
		}
		if recording {
			if a.settingsRecordRef.Focused(c) {
				a.settingsRecordSeen = true
			} else if a.settingsRecordSeen && !chip.Pressed() {
				a.later(a.stopRecording)
			}
			chip.Bind(&a.settingsRecordRef).HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputText {
					return true
				}
				if ev.Kind != ui.InputKeyDown {
					return false
				}
				return a.recordShortcut(cmd, ev.Mods, ev.Key)
			})
		}
		chip.Children(func() {
			switch {
			case recording:
				ui.Text(c, "Type shortcut…").FontSize(11.5).TextColor(k.Text)
			case keys == "None":
				ui.Text(c, "None").FontSize(11.5).TextColor(k.TextFaint)
			default:
				ui.Text(c, keys).FontSize(11.5).TextColor(k.Text).Font(noteFont.Family).
					SingleLine()
			}
		})
	})
}

func (a *app) settingsHostsPage(c *ui.Context, k *colors) {
	a.settingsNoticeLine(c, k)
	a.settingsSection(c, k, "ADD HOST", func() {
		row := ui.Column(c).Key("host-add").Padding(10, 0).Gap(6).Radius(6)
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
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				addr := ui.TextInputBase(c, &a.hostDraftAddr).Grow(1).MinWidth(0).FontSize(12).
					Placeholder("me@devbox")
				name := ui.TextInputBase(c, &a.hostDraftName).Width(100).FontSize(12).
					Placeholder("Name")
				if addr.Submitted() || name.Submitted() {
					a.addHostChecked()
				}
				if settingsButton(c, "add-host", "Add", true).Clicked() {
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
				FontSize(12).TextColor(k.TextFaint).Padding(14, 0)
		}
	})
	if len(a.remoteProjects) > 0 {
		a.settingsSection(c, k, "RECENT REMOTE PROJECTS", func() {
			for i, p := range a.remoteProjects {
				i, p := i, p
				if i > 0 {
					ui.Box(c).Height(1).Shrink(0).Background(k.PanelBorder)
				}
				row := ui.Row(c).Height(44).Padding(0, 4).Gap(9).
					Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(p.Path)
				if row.Hovered() {
					row.Background(k.Pressed)
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
						ui.Text(c, a.hostLabel(p.Host)+":"+p.Path).FontSize(10.5).
							TextColor(k.TextFaint).SingleLine().Ellipsis("…")
					})
				})
			}
		})
	}
	a.settingsSection(c, k, "ABOUT HOSTS", func() {
		ui.Text(c, "Mojian uploads a small per-user server to a host over SSH on first connect. Nothing is installed system-wide and no port is opened.").
			FontSize(12).TextColor(k.TextFaint).LineHeight(1.4).Padding(12, 0)
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
		if host != "" && a.hostRemoveAddr == host {
			ui.Row(c).Padding(0, 8, 10, 8).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Remove this server?").FontSize(12).TextColor(k.TextMuted).Grow(1)
				if settingsButton(c, "cancel-remove-"+host, "Cancel", false).Clicked() {
					a.hostRemoveAddr = ""
					a.invalidate()
				}
				if settingsButton(c, "confirm-remove-"+host, "Remove", true).Clicked() {
					a.removeHost(host)
				}
			})
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
			if settingsButton(c, "cancel-edit-"+host, "Cancel", false).Clicked() {
				a.hostEditAddr, a.settingsNotice = "", ""
				a.invalidate()
			}
			if settingsButton(c, "save-edit-"+host, "Save", true).Clicked() {
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
		if current {
			a.closeRemote()
			a.serverAlias, a.remotePath = "", ""
			a.saveSettings()
			a.later(func() {
				if a.project != "" {
					a.openProject(a.project)
				}
			})
		} else {
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
			a.closeRemote()
			a.serverAlias = ""
			a.remotePath = ""
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

func (a *app) settingsGitPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "CHANGES PAGE", func() {
		a.settingsSwitchRow(c, k, "Refresh when the window returns",
			"Read the working tree again when the app regains focus.",
			func() bool { return !a.noGitAutoRefresh }, func(v bool) { a.noGitAutoRefresh = !v })
	})
	a.settingsSection(c, k, "ABOUT THE PAGE", func() {
		a.settingsInfoRow(c, k, "Status", "Working tree against HEAD")
	})
	a.settingsNote(c, k, "Open the page with "+nativeKeys("⇧⌘G")+", or the Git button in the title bar.")
}

func (a *app) settingsAdvancedPage(c *ui.Context, k *colors) {
	a.settingsNoticeLine(c, k)
	a.settingsSection(c, k, "DATA", func() {
		a.settingsInfoRow(c, k, "Settings file", a.settingsPath)
		if a.remotePath != "" {
			a.settingsInfoRow(c, k, "Remote project", a.serverAlias+":"+a.remotePath)
		}
		if terminalsSupported() {
			a.settingsInfoRow(c, k, "Local sessions", session.SocketPath())
		}
		if a.remote != nil {
			a.settingsInfoRow(c, k, "Remote server", a.remote.host+" · "+a.remote.socket)
		}
	})
	a.settingsSection(c, k, "SETTINGS FILE", func() {
		a.settingRow(c, k, "Reveal", "Show settings.json in the file manager.", func() {
			if settingsButton(c, "reveal-settings", "Reveal", false).Clicked() {
				a.saveSettings()
				if a.settingsPath != "" {
					mygo.Shell.ShowItemInFolder(a.settingsPath)
				}
			}
		})
		a.settingRow(c, k, "Open", "Open settings.json in the default editor.", func() {
			if settingsButton(c, "open-settings-file", "Open", false).Clicked() {
				a.openSettingsFile()
			}
		})
	})
	a.settingsSection(c, k, "RESET", func() {
		if !a.settingsResetAsk {
			a.settingRow(c, k, "Reset all settings", "Restore the preferences of a new install; projects and servers stay.", func() {
				if settingsButton(c, "reset-settings", "Reset…", false).Clicked() {
					a.settingsResetAsk = true
					a.invalidate()
				}
			})
			return
		}
		a.settingRow(c, k, "Reset all settings", "This restores preferences, not projects or servers.", func() {
			if settingsButton(c, "cancel-reset", "Cancel", false).Clicked() {
				a.settingsResetAsk = false
				a.invalidate()
			}
			if settingsButton(c, "confirm-reset", "Reset", true).Clicked() {
				a.resetPreferences()
			}
		})
	})
	a.settingsAboutPage(c, k)
}

func (a *app) openSettingsFile() {
	if a.settingsPath == "" {
		return
	}
	a.saveSettings()
	if err := mygo.Shell.OpenPath(a.settingsPath); err != nil {
		a.fail(err)
	}
}

func (a *app) resetPreferences() {
	a.appearance = 0
	a.syncAppearance()
	a.editorFontFamily, a.editorFontSize, a.editorLineHeight, a.editorTabSize = "", 12.5, 1.55, 4
	a.editorSoftWrap = false
	a.hideLineNumbers = false
	a.markdownPreview = false
	a.noActiveLine = false
	a.noIndentGuides = false
	a.noAutoClose = false
	a.noScrollPastEnd = false
	a.terminalFontFamily, a.terminalFontSize, a.terminalLineHeight = "", 12.5, 1
	a.shellPath = ""
	a.terminalCursor = 0
	a.notifyBell, a.notifyExit = true, true
	a.terminalOptionAsAlt = false
	a.terminalCopySelect = false
	a.noCursorBlink = false
	a.terminalScrollback = 10
	a.noRestoreLayout = false
	a.noGitAutoRefresh = false
	a.shortcutKeys = nil
	a.settingsResetAsk = false
	a.settingsNotice = "Preferences reset."
	a.saveSettings()
	a.refreshMenu()
	a.invalidate()
}

func (a *app) settingsUpdatesPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "UPDATES", func() {
		a.settingRow(c, k, "Check for Updates", "Check the latest signed release on GitHub.", func() {
			busy := a.updateChecking || a.updateInstalling
			check := ui.Button(c, "Check for Updates").Disabled(busy || a.updateReady)
			if check.Clicked() {
				a.checkForUpdates()
			}
		})
		if a.pendingUpdate != nil && !a.updateReady {
			a.settingRow(c, k, "Install", "Download and install the release it finds.", func() {
				busy := a.updateChecking || a.updateInstalling
				install := ui.PrimaryButton(c, "Install Update").Disabled(busy)
				if install.Clicked() {
					a.installAvailableUpdate()
				}
			})
		}
		a.settingRow(c, k, "Restart", "Restart into the installed update.", func() {
			restart := ui.PrimaryButton(c, "Restart to Update").Disabled(!a.updateReady)
			if restart.Clicked() {
				a.restartForUpdate()
			}
		})
	})
	if msg := a.updateSummary(); msg != "" {
		a.settingsNote(c, k, msg)
	}
	if a.updateErr != "" {
		ui.Text(c, a.updateErr).FontSize(12).TextColor(k.Danger).Padding(0, 4)
	}
}

func (a *app) settingsAboutPage(c *ui.Context, k *colors) {
	a.settingsSection(c, k, "MOJIAN", func() {
		ui.Row(c).Padding(12, 0).AlignItems(ui.Center).Children(func() {
			wm := "wordmark"
			if c.Theme().Dark {
				wm = "wordmark-reversed"
			}
			ui.Icon(c, icons.Brand(wm)).Size(150, 36)
		})
		a.settingsInfoRow(c, k, "Version", appVersion())
		a.settingsInfoRow(c, k, "System", runtime.GOOS+" · "+runtime.GOARCH)
		project := "No project"
		if a.store != nil {
			project = a.store.dir
		}
		a.settingsInfoRow(c, k, "Project", project)
		a.settingsInfoRow(c, k, "Settings", a.settingsPath)
		a.settingsInfoRow(c, k, "Fonts", fmt.Sprintf("%d installed families", len(a.installedFonts())))
	})
}

// commandAvailable hides the commands the platform does not have, as the
// palette does.
func commandAvailable(c *command) bool {
	if !terminalsSupported() && (c == cmdTerminal || c == cmdSSH) {
		return false
	}
	if runtime.GOOS == "js" && (c == cmdTerminal || c == cmdOpenProject || c == cmdNewFolder) {
		return false
	}
	return true
}

func appVersion() string {
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return "dev"
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
		b := ui.ButtonBase(c.Key("font-"+strconv.Itoa(which))).Padding(5, 10).Radius(radius).
			Gap(7).Label(title + ": " + label)
		if b.Hovered() {
			b.Background(k.Pressed)
		}
		if b.Clicked() {
			a.fontPick, a.fontPickFor, a.fontSearch = true, which, ""
		}
		b.Children(func() {
			ui.Text(c, label).FontSize(11.5).TextColor(k.Text).SingleLine().
				Ellipsis("…").MaxWidth(190)
			ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
		})
	})
}

func (a *app) fontSizeRow(c *ui.Context, k *colors, title, detail string, value *float64, def, lo, hi float64, format string) {
	if *value <= 0 {
		*value = def
	}
	a.settingRow(c, k, title, detail, func() {
		before := *value
		ui.Text(c, fmt.Sprintf("%"+format, *value)).FontSize(11.5).TextColor(k.TextMuted).
			Width(44).TextAlign(ui.End).Shrink(0)
		ui.Slider(c, value, lo, hi).Width(160).Label(title)
		if *value != before {
			a.saveSettings()
			a.invalidate()
		}
	})
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

// installedFonts lists the families the system and the app provide,
// loaded once when a font chooser first opens.
func (a *app) installedFonts() []ui.FontFamily {
	a.fontsOnce.Do(func() { a.fonts = ui.FontFamilies() })
	return a.fonts
}

// fontPicker is the searchable list of installed families, opened by the
// font rows.
func (a *app) fontPicker(c *ui.Context, k *colors) {
	if !a.fontPick {
		return
	}
	ui.DialogBase(c, &a.fontPick, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Width(500).MaxHeight(480).Radius(radius).Background(k.Panel).
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
				row := ui.Row(c).Key("picker-default").Height(34).Padding(0, 10).Gap(8).
					Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(def)
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
					row := ui.Row(c).Key("picker-"+f.Name).Height(34).Padding(0, 10).Gap(8).
						Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).
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
								TextColor(k.TextFaint).Padding(1, 4).Radius(radius).
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
