package app

import (
	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

// openQuickOpen shows the file picker over the window: every file of the
// project is listed before typing, the field filters them, and Enter or a
// click opens one in the active tab. The welcome page and ⌘P both lead
// here, so the action is visible even when the Explorer is already open.
func (a *app) openQuickOpen() {
	a.closeModals()
	a.quickOpen = true
	a.quickQuery = ""
	a.quickSel = 0
	a.quickList = ui.ListState{}
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
	chosen := -1
	commandDialog(c, k, &a.quickOpen, 560, 440, false, func() {
		ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
			ui.Icon(c, icons.Icon("search")).Size(17, 17).TextColor(k.TextFaint)
			in := ui.TextInputBase(c, &a.quickQuery).Grow(1).FontSize(15).
				Placeholder("Open a file…").Label("Open a file").AutoFocus()
			if listNav(in, &a.quickSel, len(choices), &reveal) {
				chosen = a.quickSel
			}
		})
		a.quickList.Selected = &a.quickSel
		a.quickList.Key = func(i int) any { return choices[i].rel }
		list := ui.List(c, &a.quickList, len(choices), func(i int) {
			ch := choices[i]
			_, clicked := fileChoiceRow(c, k, i, ch, i == a.quickSel, 34, 10, 10, 13.5, 13, 500)
			if clicked {
				chosen = i
			}
		}).MaxHeight(384).Padding(6)
		if len(choices) == 0 {
			list.Children(func() {
				ui.Text(c, "No files match").FontSize(13).TextColor(k.TextFaint).Padding(14)
			})
		}
		if reveal {
			a.quickList.ScrollIntoView(a.quickSel)
		}
	})
	if chosen >= 0 && chosen < len(choices) {
		a.quickOpen = false
		ch := choices[chosen]
		a.later(func() { a.quickOpenChoice(ch) })
	}
	if was && !a.quickOpen {
		if t := a.tab(); t != nil && a.focusReq == nil {
			a.focusReq = t.editorPane()
		}
	}
}
