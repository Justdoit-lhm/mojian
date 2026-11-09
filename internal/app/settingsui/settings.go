package settingsui

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
	settingsUpdates       = settings.Updates
)

// settingsEntry is one thing Settings' search finds: the page, the label
// its row or group shows, and words that find it besides.
type settingsEntry struct {
	cat   int
	label string
	words string
}

// settingsIndex is everything the search finds: the page table's rows
// and section heads, then the command shortcuts the Keyboard page shows.
func (u *UI) settingsIndex() []settingsEntry {
	if u.IndexCache != nil && !u.IndexDirty {
		return u.IndexCache
	}
	out := make([]settingsEntry, 0, len(settings.Rows)+len(u.commands()))
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
	for _, cmd := range u.commands() {
		if !cmd.Available {
			continue
		}
		label := strings.TrimSuffix(cmd.Title, "…")
		out = append(out, settingsEntry{settingsKeyboard, label,
			"shortcut keybinding keys " + cmd.ID + " " + cmd.Keys})
	}
	u.IndexCache = out
	u.IndexDirty = false
	return out
}

// settingsWindowView is the whole Settings window: a searchable rail of
// pages on the left, the chosen page in a card on the right.
func (u *UI) WindowView(c *ui.Context) {
	u.runPending()
	u.applyTheme(c)
	k := colorsOf(c)
	c.Root().Background(k.BgBottom)
	u.prepareSettingsSearch()
	if u.Recording == "" {
		c.OnShortcut(ui.Cmd, ui.KeyF, func() { u.Search.Focus(c) })
	}
	ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		paintBackground(p, r, k)
	}).Children(func() {
		u.settingsTitleBar(c, k)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			u.settingsRail(c, k)
			u.settingsContent(c, k)
		})
	})
	u.fontPicker(c, k)
	u.remoteBusyView(c, k)
}

func (u *UI) settingsTitleBar(c *ui.Context, k *colors) {
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
func (u *UI) prepareSettingsSearch() {
	q := strings.TrimSpace(u.Query)
	u.Hits = nil
	u.Reveal = ""
	if q == "" {
		u.Found = nil
		u.Revealed = ""
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
	for i, cat := range settings.Categories {
		if !settingsCatAvailable(i) {
			continue
		}
		if match(cat.Title + " " + cat.Detail + " " + cat.Words) {
			found[i] = []string{}
		}
	}
	for _, e := range u.settingsIndex() {
		if !settingsCatAvailable(e.cat) || !match(e.label+" "+e.words) {
			continue
		}
		found[e.cat] = appendUnique(found[e.cat], e.label)
	}
	u.Found = make(map[int]bool, len(found))
	for i := range found {
		u.Found[i] = true
	}
	if len(found) == 0 {
		return
	}
	if _, ok := found[u.Cat]; !ok || !settingsCatAvailable(u.Cat) {
		for i := range settings.Categories {
			if _, ok := found[i]; ok && settingsCatAvailable(i) {
				u.Cat = i
				break
			}
		}
	}
	labels := found[u.Cat]
	if len(labels) > 0 {
		key := q + "\x00" + strconv.Itoa(u.Cat)
		if u.Revealed != key {
			u.Revealed, u.Reveal = key, labels[0]
		}
	}
	hits := make(map[string]bool, len(labels))
	for _, l := range labels {
		hits[l] = true
	}
	u.Hits = hits
}

func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}

func (u *UI) settingsRail(c *ui.Context, k *colors) {
	ui.Column(c).Width(214).Shrink(0).FillHeight().Padding(10, 12).
		Gap(2).BorderWidth(0, 0, 0, 1).BorderColor(k.PanelBorder).Children(func() {
		ui.Row(c).Height(30).Margin(0, 2, 8, 2).Padding(0, 8).Gap(6).Radius(radius).
			Background(k.Hover).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(13, 13).TextColor(k.TextFaint)
			search := ui.TextInputBase(c, &u.Query).Bind(&u.Search).Grow(1).
				FontSize(12).Placeholder("Search settings…").Label("Search settings")
			search.OnShortcut(0, ui.KeyEscape, func() {
				u.Query = ""
				u.Search.Focus(c)
				u.invalidate()
			})
			if search.Changed() {
				u.invalidate()
			}
		})
		for i, cat := range settings.Categories {
			if !settingsCatAvailable(i) {
				continue
			}
			if u.Found != nil {
				if _, ok := u.Found[i]; !ok {
					continue
				}
			}
			u.settingsNavRow(c, k, i, cat)
		}
		if u.Found != nil && len(u.Found) == 0 {
			ui.Text(c, "No settings match.").FontSize(11.5).TextColor(k.TextFaint).
				Padding(8, 10)
		}
	})
}

func (u *UI) settingsNavRow(c *ui.Context, k *colors, i int, cat settings.Category) {
	sel := i == u.Cat
	row := ui.Row(c.Key("settings-nav-"+strconv.Itoa(i))).Height(34).Padding(0, 9).
		Gap(9).Radius(radius).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(cat.Title)
	switch {
	case sel:
		row.Background(k.PanelSel)
	case row.Hovered():
		row.Background(k.Hover)
	}
	if row.Clicked() && !sel {
		u.Cat = i
		u.stopRecording()
		u.Notice = ""
		u.ResetAsk = false
		u.ScrollTop = true
		u.invalidate()
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

func (u *UI) settingsContent(c *ui.Context, k *colors) {
	if u.ScrollTop {
		u.Scroll.Y = 0
		u.ScrollTop = false
	}
	ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).FillHeight().Children(func() {
		ui.Scroll(c).Grow(1).MinHeight(0).TrackScroll(&u.Scroll).Children(func() {
			ui.Column(c).MaxWidth(680).FillWidth().AlignSelf(ui.Center).
				Padding(24, 30, 44, 30).Gap(24).Children(func() {
				cat := u.Cat
				if !settingsCatAvailable(cat) {
					cat = settingsGeneral
				}
				if u.Found != nil && len(u.Found) == 0 {
					u.settingsNoResults(c, k)
					return
				}
				u.settingsPage(c, k, cat)
			})
		})
	})
}

func (u *UI) settingsNoResults(c *ui.Context, k *colors) {
	ui.Column(c).Gap(10).Padding(24, 0).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.Icon("search")).Size(22, 22).TextColor(k.TextFaint)
		ui.Text(c, "No settings match “"+u.Query+"”.").FontSize(12.5).
			TextColor(k.TextFaint).TextAlign(ui.Center)
	})
}

func (u *UI) settingsPage(c *ui.Context, k *colors, cat int) {
	switch cat {
	case settingsGeneral, settingsAppearance, settingsEditor, settingsTerminal, settingsNotifications:
		u.settingsGenericPage(c, k, cat)
	case settingsShell:
		u.settingsGenericPage(c, k, cat)
		u.settingsNote(c, k, "Remote projects use the shell on the server, not this path.")
	case settingsKeyboard:
		u.settingsKeyboardPage(c, k)
	case settingsHosts:
		u.settingsHostsPage(c, k)
	case settingsGit:
		u.settingsGenericPage(c, k, cat)
		u.settingsNote(c, k, "Open the page with "+nativeKeys("⌃⇧G")+", or the Git button in the title bar.")
	case settingsUpdates:
		u.settingsUpdatesPage(c, k)
	default:
		u.settingsAdvancedPage(c, k)
	}
}

func (u *UI) settingsSection(c *ui.Context, k *colors, title string, fn func()) {
	u.Rows = 0
	ui.Column(c).Gap(8).FillWidth().Children(func() {
		if title != "" {
			head := ui.Row(c.Key("group-"+title)).Padding(2, 2).Radius(5)
			if u.Hits[title] {
				head.Background(k.PanelSel)
			}
			if u.Reveal == title {
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

func (u *UI) settingRow(c *ui.Context, k *colors, title, detail string, control func()) {
	u.Rows++
	if u.Rows > 1 {
		ui.Divider(c).Margin(0, 16)
	}
	row := ui.Row(c.Key("row-"+title+"-"+strconv.Itoa(u.Rows))).
		MinHeight(56).Padding(0, 16).Gap(18).AlignItems(ui.Center).Radius(6)
	if u.Hits[title] {
		row.Background(k.PanelSel)
	}
	if u.Reveal == title {
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
func (u *UI) settingsSwitchRow(c *ui.Context, k *colors, title, detail string, get func() bool, set func(bool)) {
	u.settingRow(c, k, title, detail, func() {
		on := get()
		sw := ui.Switch(c, &on).Label(title + " toggle")
		if sw.Changed() {
			set(on)
			u.saveSettings()
			u.invalidate()
		}
	})
}

// settingsChoiceRow is a row with a segmented control, whose index is
// value's and whose choice is set's.
func (u *UI) settingsChoiceRow(c *ui.Context, k *colors, key, title string, value func() int, set func(int), labels []string) {
	u.settingRow(c, k, title, "", func() {
		current := value()
		ui.Row(c).Padding(3).Radius(radius).Background(k.Track).Border(1, k.TrackBorder).
			Gap(2).Role(ui.RoleRadioGroup).Label(title).Children(func() {
			for n, label := range labels {
				n, label := n, label
				b := ui.Row(c.Key(key+"-"+strconv.Itoa(n))).Padding(5, 12).
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
					u.saveSettings()
					u.invalidate()
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

func (u *UI) settingsSliderRow(c *ui.Context, k *colors, title, detail string, value func() float64, set func(float64), lo, hi float64, format string) {
	u.settingRow(c, k, title, detail, func() {
		v := value()
		before := v
		ui.Text(c, fmt.Sprintf(format, v)).FontSize(11.5).TextColor(k.TextMuted).
			Width(44).TextAlign(ui.End).Shrink(0)
		ui.Slider(c, &v, lo, hi).Width(160).Label(title)
		if v != before {
			set(v)
			u.saveSettingsSoon()
			u.invalidate()
		}
	})
}

func (u *UI) settingsInfoRow(c *ui.Context, k *colors, title, value string) {
	u.Rows++
	if u.Rows > 1 {
		ui.Divider(c).Margin(0, 16)
	}
	row := ui.Row(c.Key("info-"+title)).MinHeight(44).Padding(0, 16).Gap(18).
		AlignItems(ui.Center).Radius(6)
	if u.Hits[title] {
		row.Background(k.PanelSel)
	}
	if u.Reveal == title {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Text(c, title).FontSize(12.5).TextColor(k.Text).Shrink(0)
		ui.Spacer(c)
		ui.Text(c, value).FontSize(11.5).TextColor(k.TextFaint).SingleLine().
			Ellipsis("…").MaxWidth(280).Shrink(0).TextAlign(ui.End)
	})
}

func (u *UI) settingsNote(c *ui.Context, k *colors, text string) {
	ui.Text(c, text).FontSize(11.5).TextColor(k.TextFaint).LineHeight(1.4).Padding(0, 2)
}

func (u *UI) settingsNoticeLine(c *ui.Context, k *colors) {
	if u.Notice == "" {
		return
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 2).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).Shrink(0)
		ui.Text(c, u.Notice).FontSize(11.5).TextColor(k.Attention).Grow(1).MinWidth(0)
	})
}

func settingsButton(c *ui.Context, label string, primary bool) ui.Element {
	if primary {
		return ui.PrimaryButton(c, label)
	}
	return ui.Button(c, label)
}
