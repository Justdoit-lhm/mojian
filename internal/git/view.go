package git

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"mojian/internal/icons"
	"mojian/internal/motion"
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
)

// View draws the working-tree diff in the editor area: changes on
// the left, the selected file's unified diff on the right.
func (p *Page) View(c *ui.Context, k *theme.Colors, h Host) {
	p.h = h
	card := ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Radius(theme.Radius).Clip().
		Background(k.CardFocused).Border(1, k.CardBorder).
		Shadow(0, 1, 2, 0, k.Shadow)
	card.Children(func() {
		p.header(c, k)
		ui.Divider(c)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			p.files(c, k)
			ui.Box(c).Width(1).FillHeight().Background(k.PanelBorder)
			p.diffPane(c, k)
		})
	})
}

func (p *Page) header(c *ui.Context, k *theme.Colors) {
	ui.Row(c).Height(headerH).Padding(0, 8, 0, 12).Gap(7).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.Icon("git-branch")).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, "Changes").FontSize(12.5).FontWeight(600).TextColor(k.Text)
		if p.Branch != "" {
			ui.Row(c).Padding(2, 7).Radius(theme.Radius).Background(k.Hover).Gap(4).
				AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Icon("git-branch")).Size(10, 10).TextColor(k.TextFaint)
				ui.Text(c, p.Branch).FontSize(11).TextColor(k.TextMuted)
			})
		}
		if n := len(p.Changes); n > 0 {
			ui.Text(c, fileCount(n)).FontSize(10.5).TextColor(k.TextFaint)
		}
		ui.Spacer(c)
		if p.Loading || p.DiffLoad {
			ui.Text(c, "Reading…").FontSize(10.5).TextColor(k.TextFaint)
		}
		if iconButton(c, k, "refresh-cw", "Refresh Changes", 25, 14).
			Tooltip("Refresh Changes").Clicked() {
			p.Refresh()
		}
		if iconButton(c, k, "x", "Close Changes", 25, 13).Clicked() {
			p.Open = false
			p.h.Invalidate()
		}
	})
}

func fileCount(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

func (p *Page) files(c *ui.Context, k *theme.Colors) {
	ui.Column(c).Width(248).Shrink(0).FillHeight().Children(func() {
		ui.Row(c).Height(34).Padding(0, 10, 0, 12).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "WORKING TREE").FontSize(10).FontWeight(650).TextColor(k.TextFaint)
			ui.Spacer(c)
			if n := len(p.Changes); n > 0 {
				ui.Text(c, fmt.Sprintf("%d", n)).FontSize(10).TextColor(k.TextFaint).
					Padding(2, 5).Radius(theme.Radius).Background(k.Hover)
			}
		})
		ui.Divider(c)
		if p.Err != "" {
			ui.Column(c).Fill().Center().Padding(16).Children(func() {
				ui.Icon(c, icons.Icon("git-branch")).Size(22, 22).TextColor(k.TextFaint)
				ui.Text(c, "Changes unavailable").FontSize(12).FontWeight(600).
					TextColor(k.Text).TextAlign(ui.Center)
				ui.Text(c, p.Err).FontSize(11).TextColor(k.TextFaint).
					TextAlign(ui.Center).MaxWidth(210)
			})
			return
		}
		if len(p.Changes) == 0 {
			ui.Column(c).Fill().Center().Padding(16).Gap(6).Children(func() {
				ui.Icon(c, icons.Icon("check")).Size(22, 22).TextColor(k.DiffAdd)
				ui.Text(c, "No changes").FontSize(12.5).FontWeight(600).TextColor(k.Text)
				if p.Loading {
					ui.Text(c, "Reading…").FontSize(11).TextColor(k.TextFaint)
				}
			})
			return
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(6, 7).Children(func() {
			for i, ch := range p.Changes {
				i, ch := i, ch
				selected := i == p.Sel
				row := ui.Row(c.Key(fmt.Sprintf("git:%d", i))).Height(38).Padding(0, 7).
					Gap(7).Radius(theme.Radius).AlignItems(ui.Center).Cursor(ui.CursorPointer).
					Role(ui.RoleButton).Label(ch.Display).Tooltip(ch.Display + " — " + statusLabel(ch.Status)).
					PopoverMenu(func(m *ui.Menu) { p.h.FileMenu(m, ch.Display) })
				if selected {
					row.Background(k.PanelSel)
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					p.Select(i)
				}
				if row.DoubleClicked() && p.h.HasPath(ch.Display) {
					p.h.OpenPath(ch.Display)
				}
				row.Children(func() {
					ui.Icon(c, icons.FileIcon(ch.Path)).Size(16, 16).TextColor(k.TextMuted)
					ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
						ui.Text(c, filepath.Base(ch.Display)).FontSize(12).FontWeight(500).
							TextColor(k.Text).SingleLine().Ellipsis("…")
						if dir := filepath.Dir(ch.Display); dir != "." {
							ui.Text(c, dir).FontSize(10.5).TextColor(k.TextFaint).
								SingleLine().Ellipsis("…")
						}
					})
					p.statusBadge(c, k, ch.Status)
				})
			}
		})
	})
}

func statusLabel(status string) string {
	switch status {
	case "A":
		return "Added"
	case "D":
		return "Deleted"
	case "R":
		return "Renamed"
	case "C":
		return "Conflict"
	case "U":
		return "Untracked"
	default:
		return "Modified"
	}
}

func statusColor(status string, k *theme.Colors) ui.Color {
	switch status {
	case "A", "U":
		return k.DiffAdd
	case "D":
		return k.DiffDel
	case "R", "C":
		return k.DiffHunk
	default:
		return k.Attention
	}
}

func (p *Page) changeStatus() string {
	if p.Sel >= 0 && p.Sel < len(p.Changes) {
		return p.Changes[p.Sel].Status
	}
	return ""
}

func (p *Page) statusBadge(c *ui.Context, k *theme.Colors, status string) {
	col := statusColor(status, k)
	ui.Text(c, statusLabel(status)).FontSize(9.5).FontWeight(700).
		TextColor(col).Padding(2, 5).Radius(theme.Radius).Background(col.Alpha(0.11)).
		Shrink(0).Tooltip(statusLabel(status))
}

func (p *Page) diffPane(c *ui.Context, k *theme.Colors) {
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
		p.diffHeader(c, k)
		ui.Divider(c)
		p.diffBody(c, k)
	})
}

func (p *Page) diffHeader(c *ui.Context, k *theme.Colors) {
	ui.Row(c).Height(34).Padding(0, 10, 0, 12).Gap(7).AlignItems(ui.Center).Children(func() {
		path := p.DiffPath
		if path == "" {
			ui.Text(c, "No file selected").FontSize(11.5).TextColor(k.TextMuted)
			ui.Spacer(c)
			return
		}
		ui.Icon(c, icons.FileIcon(path)).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, filepath.Base(path)).FontSize(12).FontWeight(550).TextColor(k.Text).
			SingleLine().Shrink(0)
		if dir := filepath.Dir(path); dir != "." {
			ui.Text(c, dir).FontSize(10.5).TextColor(k.TextFaint).
				SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		}
		ui.Spacer(c)
		if status := p.changeStatus(); status != "" {
			p.statusBadge(c, k, status)
		}
		if add, del := DiffStat(p.Diff); add > 0 || del > 0 {
			ui.Text(c, fmt.Sprintf("+%d", add)).FontSize(10.5).FontWeight(600).
				TextColor(k.DiffAdd).Shrink(0)
			ui.Text(c, fmt.Sprintf("−%d", del)).FontSize(10.5).FontWeight(600).
				TextColor(k.DiffDel).Shrink(0)
		}
		if p.h.HasPath(path) {
			open := ui.Row(c).Padding(4, 9).Radius(theme.Radius).Cursor(ui.CursorPointer).
				Role(ui.RoleButton).Label("Open File")
			if open.Hovered() {
				open.Background(k.Hover)
			}
			if open.Clicked() {
				p.h.OpenPath(path)
			}
			open.Children(func() {
				ui.Text(c, "Open").FontSize(11.5).FontWeight(550).TextColor(k.Text)
			})
		}
	})
}

func (p *Page) diffBody(c *ui.Context, k *theme.Colors) {
	ui.Box(c.Key("diff:" + p.DiffPath)).Grow(1).MinWidth(0).MinHeight(0).Clip().
		Transition(motion.Page).Children(func() { p.diffState(c, k) })
}

func (p *Page) diffState(c *ui.Context, k *theme.Colors) {
	switch {
	case p.Err != "":
		p.state(c, k, "git-branch", k.Danger, "Changes unavailable", p.Err)
	case len(p.Changes) == 0 && !p.Loading:
		p.state(c, k, "check", k.DiffAdd, "Working tree clean",
			"There are no changes to review.")
	case p.DiffLoad:
		p.state(c, k, "git-branch", k.TextFaint, "Reading diff…", "")
	case p.DiffPath == "":
		p.state(c, k, "git-branch", k.TextFaint, "Select a change",
			"Pick a changed file on the left to see its diff.")
	case p.DiffErr != "":
		p.state(c, k, "git-branch", k.Danger, "Could not read this diff", p.DiffErr)
	case strings.TrimSpace(p.Diff) == "":
		p.state(c, k, "file-text", k.TextFaint, "No text changes",
			"This file has no textual diff.")
	default:
		lines := ParseDiff(p.Diff)
		if len(lines) == 0 {
			p.state(c, k, "file-text", k.TextFaint, "No text changes",
				"This file has no textual diff.")
			return
		}
		p.diffLines(c, k, lines)
	}
}

func (p *Page) state(c *ui.Context, k *theme.Colors, glyph string, col ui.Color, title, detail string) {
	ui.Column(c).Fill().Center().Padding(20).Gap(8).Children(func() {
		ui.Box(c).Size(42, 42).Radius(theme.Radius).Background(col.Alpha(0.10)).Center().Children(func() {
			ui.Icon(c, icons.Icon(glyph)).Size(21, 21).TextColor(col)
		})
		ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(k.Text).
			TextAlign(ui.Center)
		if detail != "" {
			ui.Text(c, detail).FontSize(11.5).TextColor(k.TextFaint).
				TextAlign(ui.Center).MaxWidth(460)
		}
	})
}

// gitDiffLine is one parsed line of a unified diff, with the old and new
// line numbers its hunk gives it.
func (p *Page) diffLines(c *ui.Context, k *theme.Colors, lines []gitDiffLine) {
	const maxLines = 2500
	truncated := len(lines) > maxLines
	if truncated {
		lines = lines[:maxLines]
	}
	scroll := ui.Scroll(c).Grow(1).MinHeight(0).Padding(0, 0, 4, 0).
		TrackScroll(&p.DiffScroll)
	scroll.Children(func() {
		for _, line := range lines {
			switch line.kind {
			case '@':
				ui.Row(c).Height(24).FillWidth().Padding(0, 10).AlignItems(ui.Center).
					Background(k.DiffHunk.Alpha(0.08)).Children(func() {
					ui.Text(c, diffLineText(line.text)).Font(theme.NoteFont.Family).FontSize(11).
						TextColor(k.DiffHunk).SingleLine()
				})
			case 'm', '\\':
				ui.Row(c).Height(18).FillWidth().Padding(0, 10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, diffLineText(line.text)).Font(theme.NoteFont.Family).FontSize(11).
						TextColor(k.TextFaint).SingleLine()
				})
			default:
				row := ui.Row(c).Height(18).FillWidth().ClipX().AlignItems(ui.Center)
				switch line.kind {
				case '+':
					row.Background(k.DiffAdd.Alpha(0.10))
				case '-':
					row.Background(k.DiffDel.Alpha(0.09))
				}
				row.Children(func() {
					gitGutter(c, k, line.old)
					gitGutter(c, k, line.new)
					ui.Text(c, diffLineText(line.text)).Font(theme.NoteFont.Family).FontSize(11.5).
						TextColor(diffLineKindColor(line.kind, k)).SingleLine().Ellipsis("…").
						Shrink(1).MinWidth(0).Padding(0, 0, 0, 8)
				})
			}
		}
		if truncated {
			ui.Text(c, "… diff truncated").FontSize(11).TextColor(k.TextFaint).Padding(6, 10)
		}
	})
}

func gitGutter(c *ui.Context, k *theme.Colors, n int) {
	text := ""
	if n > 0 {
		text = strconv.Itoa(n)
	}
	ui.Text(c, text).Font(theme.NoteFont.Family).FontSize(10).TextColor(k.TextFaint).
		Width(38).TextAlign(ui.End).Shrink(0).Padding(0, 5)
}

func diffLineKindColor(kind byte, k *theme.Colors) ui.Color {
	switch kind {
	case '+':
		return k.DiffAdd
	case '-':
		return k.DiffDel
	default:
		return k.TextMuted
	}
}

func diffLineText(line string) string {
	line = strings.ReplaceAll(line, "\t", "    ")
	line = strings.ReplaceAll(line, " ", "\u00a0")
	if line == "" {
		return "\u00a0"
	}
	return line
}

// hasProject reports whether the host has a project on disk.
func hasProject(h Host) bool {
	_, ok := h.ProjectDir()
	return ok
}

// mustDir returns the project directory, or "".
func mustDir(h Host) string {
	dir, _ := h.ProjectDir()
	return dir
}

// iconButton is a borderless icon button of the page header.
func iconButton(c *ui.Context, k *theme.Colors, name, label string, size, iconSize float32) ui.Element {
	b := ui.Box(c).Size(size, size).Center().Radius(theme.Radius).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if b.Pressed() {
		b.Background(k.Pressed)
	} else if b.Hovered() {
		b.Background(k.Hover)
	}
	b.Children(func() {
		ui.Icon(c, icons.Icon(name)).Size(iconSize, iconSize).TextColor(k.IconMuted)
	})
	return b
}
