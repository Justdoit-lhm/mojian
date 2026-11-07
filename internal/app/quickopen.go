package app

import (
	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

// openQuickOpen shows the file picker over the window: every file of the
// project is listed before typing, the field filters them, and Enter or a
// click opens one in the active tab. The welcome page and ⇧⌘O both lead
// here, so the action is visible even when the Explorer is already open.
func (a *app) openQuickOpen() {
	a.paletteOpen = false
	a.projectPicker = false
	a.quickOpen = true
	a.quickQuery = ""
	a.quickSel = 0
	a.invalidate()
}

// quickOpenChoice opens the file chosen in the picker.
func (a *app) quickOpenChoice(ch fileChoice) {
	switch {
	case ch.note != nil:
		a.openNote(ch.note)
	case ch.asset != nil:
		a.openAsset(ch.asset)
	}
}

// quickOpenModal draws the file picker over the window.
func (a *app) quickOpenModal(c *ui.Context, k *colors) {
	if !a.quickOpen {
		return
	}
	choices := a.paneChoices(a.quickQuery)
	a.quickSel = min(max(a.quickSel, 0), max(len(choices)-1, 0))
	was := a.quickOpen
	reveal := false
	ui.DialogBase(c, &a.quickOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Width(560).MaxHeight(440).Radius(radius).Background(k.Panel).Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.06)).Clip()
		ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(17, 17).TextColor(k.TextFaint)
			in := ui.TextInputBase(c, &a.quickQuery).Grow(1).FontSize(15).
				Placeholder("Open a file…").Label("Open a file").AutoFocus()
			if in.Changed() {
				a.quickSel, reveal = 0, true
			}
			if in.Shortcut(0, ui.KeyDown) {
				a.quickSel, reveal = min(a.quickSel+1, len(choices)-1), true
			}
			if in.Shortcut(0, ui.KeyUp) {
				a.quickSel, reveal = max(a.quickSel-1, 0), true
			}
			if in.Submitted() && len(choices) > 0 {
				a.quickOpen = false
				ch := choices[a.quickSel]
				a.later(func() { a.quickOpenChoice(ch) })
			}
		})
		listH := float32(min(max(len(choices), 1)*34+12, 384))
		ui.Scroll(c).Height(listH).Padding(6).Children(func() {
			if len(choices) == 0 {
				ui.Text(c, "No files match").FontSize(13).TextColor(k.TextFaint).Padding(14)
			}
			for i, ch := range choices {
				i, ch := i, ch
				row := ui.Row(c.Key(i)).Height(34).Padding(0, 10).Gap(10).Radius(radius).
					AlignItems(ui.Center).Cursor(ui.CursorPointer).
					Role(ui.RoleButton).Label(ch.title)
				if i == a.quickSel {
					row.Background(k.PanelSel)
					if reveal {
						row.ScrollIntoView()
					}
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					a.quickOpen = false
					a.later(func() { a.quickOpenChoice(ch) })
				}
				row.Children(func() {
					ui.Icon(c, icons.FileIcon(ch.rel)).Size(15, 15).TextColor(k.TextMuted)
					ui.Text(c, ch.title).FontSize(13.5).FontWeight(500).TextColor(k.Text).
						SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
					ui.Text(c, ch.rel).FontSize(13).TextColor(k.TextFaint).
						SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
				})
			}
		})
	})
	if was && !a.quickOpen {
		if t := a.tab(); t != nil && a.focusReq == nil {
			a.focusReq = t.editorPane()
		}
	}
}
