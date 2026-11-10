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
	settingsGeneral    = settings.General
	settingsAppearance = settings.Appearance
	settingsEditor     = settings.Editor
	settingsTerminal   = settings.Terminal
	settingsKeyboard   = settings.Keyboard
	settingsHosts      = settings.Hosts
	settingsAdvanced   = settings.Advanced
	settingsAbout      = settings.About
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
	k := u.colorsOf(c)
	c.Root().Background(k.BgBottom)
	u.prepareSettingsSearch()
	if u.Recording == "" {
		c.OnShortcut(ui.Cmd, ui.KeyF, func() { u.Search.Focus(c) })
	}
	ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		paintBackground(p, r, k)
	}).Children(func() {
		u.settingsTitleBar(c, k)
		ui.Row(c).Grow(1).MinHeight(0).Padding(12).Gap(12).
			AlignItems(ui.Stretch).Children(func() {
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
				u.ScrollTop = true
				break
			}
		}
	}
	labels := found[u.Cat]
	if len(labels) > 0 {
		key := q + "\x00" + strconv.Itoa(u.Cat)
		if u.Revealed != key {
			u.Revealed, u.Reveal = key, u.firstFind(u.Cat, labels, q)
		}
	}
	hits := make(map[string]bool, len(labels))
	for _, l := range labels {
		hits[l] = true
	}
	u.Hits = hits
}

// firstFind is the find a search brings into view: the one it names
// exactly when there is one, so "shell" lands on the Shell row rather
// than on the SHELL heading, and the first row otherwise.
func (u *UI) firstFind(cat int, labels []string, query string) string {
	sections := map[string]bool{}
	for _, row := range settings.Rows {
		if row.Cat == cat && row.Section != "" {
			sections[row.Section] = true
		}
	}
	exact, first := "", ""
	for _, label := range labels {
		if sections[label] {
			continue
		}
		if first == "" {
			first = label
		}
		if strings.EqualFold(label, query) {
			exact = label
			break
		}
	}
	switch {
	case exact != "":
		return exact
	case first != "":
		return first
	}
	return labels[0]
}

func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}

// settingsRail is the page list: a frosted card over the window, with the
// one search field above it and a source list of pages below.
func (u *UI) settingsRail(c *ui.Context, k *colors) {
	ui.Column(c).Width(216).Shrink(0).FillHeight().Gap(3).
		Background(k.Card).Border(1, k.CardBorder).Radius(radius).
		Shadow(0, 8, 24, 0, k.Shadow).Clip().Children(func() {
		ui.Box(c).FillWidth().Padding(10, 10, 0, 10).Children(func() {
			search := u.settingsSearchField(c, k, "rail-search", &u.Query,
				"Search settings", "Search settings…", &u.Search)
			search.OnShortcut(0, ui.KeyEscape, func() {
				u.Query = ""
				u.Search.Focus(c)
				u.invalidate()
			})
		})
		if u.Found != nil && len(u.Found) == 0 {
			ui.Text(c, "No settings match.").FontSize(12).TextColor(k.TextFaint).
				Padding(10)
			return
		}
		// A source list carries the pages with one Tab stop, the arrows and
		// type-to-choose, and the tree role screen readers expect.
		chosen := strconv.Itoa(u.Cat)
		rail := ui.Sidebar(c, &chosen, func() {
			for i, cat := range settings.Categories {
				if !settingsCatAvailable(i) {
					continue
				}
				if u.Found != nil {
					if _, ok := u.Found[i]; !ok {
						continue
					}
				}
				ui.SidebarItem(c, strconv.Itoa(i), icons.Icon(cat.Glyph), cat.Title)
			}
		}).Grow(1).FillWidth().MinHeight(0).Background(k.Card)
		if rail.Changed() {
			if i, err := strconv.Atoi(chosen); err == nil && i != u.Cat {
				u.Cat = i
				u.stopRecording()
				u.Notice = ""
				u.ResetAsk = false
				u.ScrollTop = true
				u.invalidate()
			}
		}
	})
}

// settingsContent is the page column: its header, the window's notice and
// the page's own sections, inside one scroll.
func (u *UI) settingsContent(c *ui.Context, k *colors) {
	if u.ScrollTop {
		u.Scroll.Y = 0
		u.ScrollTop = false
	}
	ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).FillHeight().Children(func() {
		ui.Scroll(c).Grow(1).MinHeight(0).TrackScroll(&u.Scroll).Children(func() {
			ui.Column(c).MaxWidth(660).FillWidth().AlignSelf(ui.Center).
				Padding(22, 24, 40, 24).Gap(20).Children(func() {
				cat := u.Cat
				if !settingsCatAvailable(cat) {
					cat = settingsGeneral
				}
				if u.Found != nil && len(u.Found) == 0 {
					u.settingsNoResults(c, k)
					return
				}
				u.settingsPageHeader(c, k, settings.Categories[cat])
				u.settingsNotice(c, k)
				u.settingsPage(c, k, cat)
			})
		})
	})
}

// settingsPageHeader names the page and says what it holds.
func (u *UI) settingsPageHeader(c *ui.Context, k *colors, cat settings.Category) {
	ui.Column(c).Gap(3).Children(func() {
		ui.Text(c, cat.Title).FontSize(20).FontWeight(650).TextColor(k.Text)
		ui.Text(c, cat.Detail).FontSize(12.5).TextColor(k.TextMuted).LineHeight(1.35)
	})
}

// settingsNotice is the window's one line about what just happened; the
// pages never draw their own.
func (u *UI) settingsNotice(c *ui.Context, k *colors) {
	if u.Notice == "" {
		return
	}
	ui.Row(c).Padding(8, 12).Radius(8).Background(k.Hover).Gap(8).
		AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(k.Attention).Shrink(0)
		ui.Text(c, u.Notice).FontSize(12).TextColor(k.Attention).Grow(1).MinWidth(0)
	})
}

func (u *UI) settingsNoResults(c *ui.Context, k *colors) {
	ui.Column(c).Gap(10).Padding(56, 0).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.Icon("search")).Size(22, 22).TextColor(k.TextFaint)
		ui.Text(c, "No settings match “"+u.Query+"”.").FontSize(12.5).
			TextColor(k.TextFaint).TextAlign(ui.Center)
		ui.Text(c, "No other setting uses that word.").FontSize(11.5).
			TextColor(k.TextFaint).TextAlign(ui.Center)
	})
}

func (u *UI) settingsPage(c *ui.Context, k *colors, cat int) {
	switch cat {
	case settingsGeneral, settingsEditor:
		u.settingsGenericPage(c, k, cat)
	case settingsTerminal:
		u.settingsGenericPage(c, k, cat)
		if remoteSupported() {
			u.settingsNote(c, k, "Remote projects use the shell on the server, not this path.")
		}
	case settingsAppearance:
		u.settingsAppearancePage(c, k)
	case settingsKeyboard:
		u.settingsKeyboardPage(c, k)
	case settingsHosts:
		u.settingsHostsPage(c, k)
	case settingsAdvanced:
		u.settingsAdvancedPage(c, k)
	case settingsAbout:
		u.settingsAboutPage(c, k)
	default:
		u.settingsGenericPage(c, k, settingsGeneral)
	}
}

// settingsSearchField draws the one search field Settings uses, binds it to
// handle when non-nil, and returns it for autofocus and Changed. SearchField
// brings the magnifier, the clear button and the search field role; only its
// face follows Settings' card.
func (u *UI) settingsSearchField(c *ui.Context, k *colors, key string, value *string, label, placeholder string, handle *ui.Handle) ui.Element {
	in := ui.SearchField(c.Key(key), value).Grow(1).MinWidth(0).Height(30).
		Padding(0, 9).Radius(8).Background(k.Hover).Border(0, ui.Transparent).
		FontSize(12).Placeholder(placeholder).Label(label)
	if handle != nil {
		in.Bind(handle)
	}
	if in.Changed() {
		u.invalidate()
	}
	return in
}

// settingsSection draws a titled group of rows in one card.
func (u *UI) settingsSection(c *ui.Context, k *colors, title string, fn func()) {
	u.settingsSectionAction(c, k, title, nil, fn)
}

// settingsSectionAction is settingsSection with a control at the end of
// the group's heading, such as Restore Defaults.
func (u *UI) settingsSectionAction(c *ui.Context, k *colors, title string, action func(), fn func()) {
	u.Rows = 0
	ui.Column(c).Gap(8).FillWidth().Children(func() {
		if title != "" {
			head := ui.Row(c.Key("group-" + title)).MinHeight(24).Padding(2).
				AlignItems(ui.Center).Gap(8).Radius(5)
			if u.Hits[title] {
				head.Background(k.PanelSel)
			}
			if u.Reveal == title {
				head.ScrollIntoView()
			}
			head.Children(func() {
				ui.Text(c, title).FontSize(11).FontWeight(650).
					TextColor(k.TextMuted).LetterSpacing(0.5)
				if action != nil {
					ui.Spacer(c)
					action()
				}
			})
		}
		ui.Column(c).FillWidth().Background(k.CardFocused).
			Border(1, k.CardBorderFocused).Radius(radius).
			Shadow(0, 1, 3, 0, k.Shadow).Clip().Children(fn)
	})
}

func (u *UI) settingRow(c *ui.Context, k *colors, title, detail string, control func()) {
	u.settingRowFrame(c, k, title, detail, false, control)
}

// settingRowMono is settingRow with the detail set as a path: one line,
// cut with an ellipsis, and the whole of it in a tooltip.
func (u *UI) settingRowMono(c *ui.Context, k *colors, title, detail string, control func()) {
	u.settingRowFrame(c, k, title, detail, true, control)
}

func (u *UI) settingRowFrame(c *ui.Context, k *colors, title, detail string, mono bool, control func()) {
	u.Rows++
	if u.Rows > 1 {
		ui.Divider(c).Margin(0, 16)
	}
	row := ui.Row(c.Key("row-"+title+"-"+strconv.Itoa(u.Rows))).
		MinHeight(52).Padding(0, 16).Gap(16).AlignItems(ui.Center)
	if u.Hits[title] {
		row.Background(k.PanelSel).Radius(8)
	}
	if u.Reveal == title {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(k.Text)
			switch {
			case detail == "":
			case mono:
				ui.Text(c, detail).FontSize(11.5).TextColor(k.TextMuted).
					Font(noteFont.Family).SingleLine().Ellipsis("…").Tooltip(detail)
			default:
				ui.Text(c, detail).FontSize(11.5).TextColor(k.TextMuted).LineHeight(1.35)
			}
		})
		if control != nil {
			ui.Row(c).Shrink(0).Gap(10).AlignItems(ui.Center).Children(control)
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
func (u *UI) settingsChoiceRow(c *ui.Context, k *colors, key, title, detail string, value func() int, set func(int), labels []string) {
	u.settingRow(c, k, title, detail, func() {
		u.settingsChoiceControl(c, k, key, title, value(), labels, func(n int) {
			set(n)
			u.saveSettings()
			u.invalidate()
		})
	})
}

// settingsChoiceControl is the segmented control a row places at its end;
// the selected index and the labels are read before it is built.
func (u *UI) settingsChoiceControl(c *ui.Context, k *colors, key, label string, current int, labels []string, choose func(int)) {
	seg := ui.SegmentedBase(c, &current, len(labels))
	seg.Track.Padding(3).Radius(8).Background(k.Hover).
		Border(1, k.PanelBorder).Gap(2).Label(label)
	seg.Track.Children(func() {
		for n, text := range labels {
			c.Key(key + "-" + strconv.Itoa(n))
			b := seg.Segment(n).Padding(4, 11).Radius(6).Cursor(ui.CursorPointer)
			switch {
			case n == current:
				b.Background(k.CardFocused)
			case b.Hovered():
				b.Background(k.Pressed)
			}
			b.Children(func() {
				col := k.TextMuted
				if n == current {
					col = k.Text
				}
				ui.Text(c, text).FontSize(11.5).FontWeight(550).TextColor(col)
			})
		}
	})
	// OnChange commits the local the bound segments chose once the input
	// has updated it, which Changed in the same pass cannot promise.
	seg.Track.OnChange(func() { choose(current) })
}

func (u *UI) settingsSliderRow(c *ui.Context, k *colors, title, detail string, value func() float64, set func(float64), lo, hi float64, format string) {
	u.settingRow(c, k, title, detail, func() {
		v := value()
		before := v
		ui.Text(c, fmt.Sprintf(format, v)).FontSize(12).TextColor(k.TextMuted).
			Width(52).TextAlign(ui.End).Shrink(0)
		ui.Slider(c, &v, lo, hi).Width(180).Label(title)
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
	row := ui.Row(c.Key("info-"+title)).MinHeight(46).Padding(0, 16).Gap(16).
		AlignItems(ui.Center)
	if u.Hits[title] {
		row.Background(k.PanelSel).Radius(8)
	}
	if u.Reveal == title {
		row.ScrollIntoView()
	}
	row.Children(func() {
		ui.Text(c, title).FontSize(13).TextColor(k.Text).Shrink(0)
		ui.Spacer(c)
		ui.Text(c, value).FontSize(12).TextColor(k.TextMuted).SingleLine().
			Ellipsis("…").MaxWidth(360).Shrink(1).MinWidth(0).
			TextAlign(ui.End).Tooltip(value)
	})
}

func (u *UI) settingsNote(c *ui.Context, k *colors, text string) {
	ui.Text(c, text).FontSize(11.5).TextColor(k.TextFaint).LineHeight(1.4).Padding(0, 2)
}

func settingsButton(c *ui.Context, label string, primary bool) ui.Element {
	if primary {
		return ui.PrimaryButton(c, label)
	}
	return ui.Button(c, label)
}
