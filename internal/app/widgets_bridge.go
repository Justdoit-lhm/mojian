package app

import (
	"mojian/internal/app/widgets"

	"github.com/egoist/mygo/ui"
)

// The list and dialog implementations live in internal/app/widgets; this
// file keeps the names the app's views use.

func listNav(in ui.Element, sel *int, n int, reveal *bool) bool {
	return widgets.ListNav(in, sel, n, reveal)
}

func commandDialog(c *ui.Context, k *colors, open *bool, width, maxH float32, start bool, body func()) {
	widgets.CommandDialog(c, k, open, width, maxH, start, body)
}

// fileChoiceRow is one file in a picker: its icon, title and project
// path, with the shared selected and hover faces.
func fileChoiceRow(c *ui.Context, k *colors, key any, ch fileChoice, selected bool,
	height, padX, gap, titleSize, relSize float32, titleWeight int) (ui.Element, bool) {
	return widgets.FileRow(c, k, key, ch.title, ch.rel, selected, height, padX, gap, titleSize, relSize, titleWeight)
}
