package app

import (
	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

// listNav applies the keys every filtered list shares: typing resets the
// selection, the arrows move it, and Enter chooses when a row exists.
func listNav(in ui.Element, sel *int, n int, reveal *bool) bool {
	if in.Changed() {
		*sel = 0
		*reveal = true
		return false
	}
	if n == 0 {
		return false
	}
	switch {
	case in.Shortcut(0, ui.KeyDown):
		*sel = min(*sel+1, n-1)
		*reveal = true
	case in.Shortcut(0, ui.KeyUp):
		*sel = max(*sel-1, 0)
		*reveal = true
	}
	return in.Submitted()
}

// commandDialog is the search sheet the palette and quick open share.
func commandDialog(c *ui.Context, k *colors, open *bool, width, maxH float32, start bool, body func()) {
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		if start {
			backdrop.Justify(ui.Start).Padding(78, 0, 0, 0)
		}
		panel.Width(width).MaxHeight(maxH).Radius(radius).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.06)).Clip()
		body()
	})
}

// fileChoiceRow is one file in a picker: its icon, title and project
// path, with the shared selected and hover faces.
func fileChoiceRow(c *ui.Context, k *colors, key any, ch fileChoice, selected bool,
	height, padX, gap, titleSize, relSize float32, titleWeight int) (ui.Element, bool) {
	row := ui.Row(c.Key(key)).Height(height).Padding(0, padX).Gap(gap).
		Radius(radius).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(ch.title)
	if selected {
		row.Background(k.PanelSel)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	row.Children(func() {
		ui.Icon(c, icons.FileIcon(ch.rel)).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, ch.title).FontSize(titleSize).FontWeight(titleWeight).TextColor(k.Text).
			SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
		ui.Text(c, ch.rel).FontSize(relSize).TextColor(k.TextFaint).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
	})
	return row, row.Clicked()
}
