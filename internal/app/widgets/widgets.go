// Package widgets holds the list and dialog pieces the command palette,
// quick open, file pickers and the SSH picker share.
package widgets

import (
	"mojian/internal/icons"
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
)

// ListNav applies the keys every filtered list shares: typing resets the
// selection, the arrows move it, and Enter chooses when a row exists.
func ListNav(in ui.Element, sel *int, n int, reveal *bool) bool {
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

// CommandDialog is the search sheet the palette and quick open share.
func CommandDialog(c *ui.Context, k *theme.Colors, open *bool, width, maxH float32, start bool, body func()) {
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		if start {
			backdrop.Justify(ui.Start).Padding(78, 0, 0, 0)
		}
		panel.Width(width).MaxHeight(maxH).Radius(theme.Radius).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.06)).Clip()
		body()
	})
}

// FileRow is one file in a picker: its icon, title and project path, with
// the shared selected and hover faces.
func FileRow(c *ui.Context, k *theme.Colors, key any, title, rel string, selected bool,
	height, padX, gap, titleSize, relSize float32, titleWeight int) (ui.Element, bool) {
	row := ui.Row(c.Key(key)).Height(height).Padding(0, padX).Gap(gap).
		Radius(theme.Radius).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(title)
	fg, muted, icon := k.Text, k.TextFaint, k.TextMuted
	if selected {
		// List paints the accent row; its text has to follow that face.
		fg = c.Theme().AccentText
		muted, icon = fg.Alpha(0.9), fg.Alpha(0.9)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	row.Children(func() {
		ui.Icon(c, icons.FileIcon(rel)).Size(15, 15).TextColor(icon)
		ui.Text(c, title).FontSize(titleSize).FontWeight(titleWeight).TextColor(fg).
			SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
		ui.Text(c, rel).FontSize(relSize).TextColor(muted).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
	})
	return row, row.Clicked()
}
