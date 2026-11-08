package app

import (
	"fmt"
	"strconv"
	"strings"

	"mojian/internal/icons"
	"mojian/internal/settings"

	"github.com/egoist/mygo/ui"
)

const (
	settingsGeneral       = settings.General
	settingsAppearance    = settings.Appearance
	settingsEditor        = settings.Editor
	settingsTerminal      = settings.Terminal
	settingsShell         = settings.Shell
	settingsNotifications = settings.Notifications
	settingsKeyboard      = settings.Keyboard
	settingsHosts         = settings.Hosts
	settingsGit           = settings.Git
	settingsAdvanced      = settings.Advanced
	settingsUpdates       = settings.Updates
)

var settingsCats = settings.Categories

// settingsEntry is one thing Settings' search finds: the page, the label
// its row or group shows, and words that find it besides.
type settingsEntry struct {
	cat   int
	label string
	words string
}

// settingsIndex is everything the search finds: the page table's rows
// and section heads, then the command shortcuts the Keyboard page shows.
func (a *app) settingsIndex() []settingsEntry {
	if a.settingsIndexCache != nil && !a.settingsIndexDirty {
		return a.settingsIndexCache
	}
	out := make([]settingsEntry, 0, len(settings.Rows)+len(paletteCommands))
	seenSection := map[string]bool{}
	for _, row := range settings.Rows {
		if row.Section != "" {
			key := strconv.Itoa(row.Cat) + "\x00" + row.Section
			if !seenSection[key] {
				seenSection[key] = true
				out = append(out, settingsEntry{row.Cat, row.Section, ""})
			}
		}
		out = append(out, settingsEntry{row.Cat, row.Title, row.Words})
	}
	for _, cmd := range paletteCommands {
		if cmd.accel == "" || !commandAvailable(cmd) {
			continue
		}
		label := strings.TrimSuffix(cmd.title, "…")
		out = append(out, settingsEntry{settingsKeyboard, label,
			"shortcut keybinding keys " + cmd.id + " " + a.commandKeys(cmd)})
	}
	a.settingsIndexCache = out
	a.settingsIndexDirty = false
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
	a.remoteBusyView(c, k)
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
		if match(cat.Title + " " + cat.Detail + " " + cat.Words) {
			found[i] = []string{}
		}
	}
	for _, e := range a.settingsIndex() {
		if !settingsCatAvailable(e.cat) || !match(e.label+" "+e.words) {
			continue
		}
		found[e.cat] = appendUnique(found[e.cat], e.label)
	}
	a.settingsFound = make(map[int]bool, len(found))
	for i := range found {
		a.settingsFound[i] = true
	}
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

func (a *app) settingsNavRow(c *ui.Context, k *colors, i int, cat settings.Category) {
	sel := i == a.settingsCat
	row := ui.Row(c).Key("settings-nav-"+strconv.Itoa(i)).Height(34).Padding(0, 9).
		Gap(9).Radius(radius).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(cat.Title)
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
		ui.Icon(c, icons.Icon(cat.Glyph)).Size(15, 15).TextColor(col)
		ui.Text(c, cat.Title).FontSize(12.5).FontWeight(500).TextColor(col).
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
	case settingsGeneral, settingsAppearance, settingsEditor, settingsTerminal, settingsNotifications:
		a.settingsGenericPage(c, k, cat)
	case settingsShell:
		a.settingsGenericPage(c, k, cat)
		a.settingsNote(c, k, "Remote projects use the shell on the server, not this path.")
	case settingsKeyboard:
		a.settingsKeyboardPage(c, k)
	case settingsHosts:
		a.settingsHostsPage(c, k)
	case settingsGit:
		a.settingsGenericPage(c, k, cat)
		a.settingsNote(c, k, "Open the page with "+nativeKeys("⇧⌘G")+", or the Git button in the title bar.")
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
			a.saveSettingsSoon()
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

func settingsButton(c *ui.Context, label string, primary bool) ui.Element {
	if primary {
		return ui.PrimaryButton(c, label)
	}
	return ui.Button(c, label)
}
