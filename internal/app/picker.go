package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
)

type pickerEntry struct {
	Name string
	Path string
}

// openProjectPicker shows the folder picker in the window, with the
// app's own colors and controls.
func (a *app) openProjectPicker() {
	if runtime.GOOS == "js" {
		a.fail(fmt.Errorf("projects need the desktop app"))
		return
	}
	a.projectOpen = false
	a.paletteOpen = false
	a.projectPicker = true
	a.pickerSel = -1
	a.pickerErr = ""
	start := ""
	if a.store != nil && !a.store.memory {
		start = a.store.dir
	}
	if start == "" {
		if home, err := os.UserHomeDir(); err == nil {
			start = home
		}
	}
	a.setPickerPath(start)
	a.invalidate()
}

// setPickerPath moves the picker to a folder and lists its subfolders.
func (a *app) setPickerPath(p string) {
	p = strings.TrimSpace(p)
	if p == "" {
		return
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		}
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		a.pickerErr = err.Error()
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		a.pickerErr = err.Error()
		return
	}
	if !info.IsDir() {
		a.pickerErr = abs + " is not a folder"
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		a.pickerErr = err.Error()
		return
	}
	var dirs []pickerEntry
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirs = append(dirs, pickerEntry{Name: e.Name(), Path: filepath.Join(abs, e.Name())})
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	a.pickerPath = abs
	a.pickerInput = abs
	a.pickerEntries = dirs
	a.pickerSel = -1
	a.pickerErr = ""
	a.invalidate()
}

func (a *app) pickerTarget() string {
	if a.pickerSel >= 0 && a.pickerSel < len(a.pickerEntries) {
		return a.pickerEntries[a.pickerSel].Path
	}
	return a.pickerPath
}

func (a *app) pickerUp() {
	parent := filepath.Dir(a.pickerPath)
	if parent != "" && parent != a.pickerPath {
		a.setPickerPath(parent)
	}
}

func (a *app) pickerOpen() {
	target := a.pickerTarget()
	if target == "" {
		return
	}
	a.projectPicker = false
	a.openProject(target)
	a.invalidate()
}

// projectPickerModal draws the folder picker over the window.
func (a *app) projectPickerModal(c *ui.Context, k *colors) {
	if !a.projectPicker {
		return
	}
	ui.DialogBase(c, &a.projectPicker, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Fill().Margin(18).Radius(radius).Background(k.Sheet).Border(1, k.PanelBorder).
			Shadow(0, 24, 70, -10, k.ShadowFocused).Clip()
		ui.Column(c).Fill().Children(func() {
			a.pickerHeader(c, k)
			ui.Divider(c)
			ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
				a.pickerRail(c, k)
				ui.Box(c).Width(1).FillHeight().Background(k.PanelBorder)
				a.pickerMain(c, k)
			})
			ui.Divider(c)
			a.pickerFooter(c, k)
		})
	})
}

func (a *app) pickerHeader(c *ui.Context, k *colors) {
	ui.Row(c).Height(60).Padding(0, 18, 0, 12).Gap(11).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(34, 34).Radius(radius).Background(k.Hover).Center().Children(func() {
			ui.Icon(c, icons.FolderIcon()).Size(18, 18).TextColor(k.TextMuted)
		})
		ui.Column(c).Gap(1).Children(func() {
			ui.Text(c, "Open Folder").FontSize(15).FontWeight(650).TextColor(k.Text)
			ui.Text(c, "Choose a folder to open as the project").
				FontSize(11.5).TextColor(k.TextFaint)
		})
		ui.Spacer(c)
		if iconButton(c, k, "x", "Close", 28, 15).Clicked() {
			a.projectPicker = false
		}
	})
}

func (a *app) pickerRail(c *ui.Context, k *colors) {
	ui.Column(c).Width(190).Shrink(0).FillHeight().Padding(12, 10).Gap(2).Children(func() {
		ui.Text(c, "PLACES").FontSize(10).FontWeight(650).TextColor(k.TextFaint).
			Padding(2, 8, 4, 8)
		home, _ := os.UserHomeDir()
		for _, place := range []struct{ label, path string }{
			{"Home", home},
			{"Desktop", filepath.Join(home, "Desktop")},
			{"Documents", filepath.Join(home, "Documents")},
			{"Downloads", filepath.Join(home, "Downloads")},
		} {
			a.pickerPlace(c, k, place.label, place.path)
		}
		recent := a.recentProjects()
		if len(recent) > 0 {
			ui.Divider(c).Margin(6, 4)
			ui.Text(c, "RECENT").FontSize(10).FontWeight(650).TextColor(k.TextFaint).
				Padding(2, 8, 4, 8)
			for i, p := range recent {
				if i >= 5 {
					break
				}
				a.pickerRecent(c, k, p)
			}
		}
	})
}

func (a *app) pickerPlace(c *ui.Context, k *colors, label, path string) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return
	}
	current := filepath.Clean(path) == filepath.Clean(a.pickerPath)
	row := ui.Row(c.Key("place:"+path)).Height(30).Padding(0, 8).Gap(8).Radius(radius).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if current {
		row.Background(k.PanelSel)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		a.later(func() { a.setPickerPath(path) })
	}
	row.Children(func() {
		ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, label).FontSize(12.5).TextColor(k.Text)
	})
}

func (a *app) pickerRecent(c *ui.Context, k *colors, path string) {
	row := ui.Row(c.Key("recent:"+path)).Height(30).Padding(0, 8).Gap(8).Radius(radius).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleButton).
		Label(a.projectLabel(path)).Tooltip(shortPath(path))
	if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		a.projectPicker = false
		a.later(func() { a.openProject(path) })
	}
	row.Children(func() {
		ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, a.projectLabel(path)).FontSize(12.5).TextColor(k.Text).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
	})
}

func (a *app) pickerMain(c *ui.Context, k *colors) {
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Padding(14, 12).Gap(8).Children(func() {
		ui.Row(c).Height(34).Gap(6).AlignItems(ui.Center).Children(func() {
			if b := iconButton(c, k, "arrow-up", "Parent Folder", 28, 14).
				Tooltip("Parent Folder"); b.Clicked() {
				a.later(func() { a.pickerUp() })
			}
			field := ui.Row(c).Grow(1).Height(34).Padding(0, 10).Gap(7).Radius(radius).
				Background(k.Hover).AlignItems(ui.Center)
			field.Children(func() {
				ui.Icon(c, icons.FolderIcon()).Size(14, 14).TextColor(k.TextFaint)
				in := ui.TextInputBase(c, &a.pickerInput).Grow(1).FontSize(12.5).
					Placeholder("Folder path").AutoFocus()
				if in.Submitted() {
					a.later(func() { a.setPickerPath(a.pickerInput) })
				}
			})
		})
		if a.pickerErr != "" {
			ui.Text(c, a.pickerErr).FontSize(11.5).TextColor(k.Danger).
				SingleLine().Ellipsis("…")
		}
		list := ui.Scroll(c).Grow(1).MinHeight(0).Padding(0, 2)
		list.Children(func() {
			if len(a.pickerEntries) == 0 {
				ui.Text(c, "No subfolders").FontSize(12.5).TextColor(k.TextFaint).
					Padding(10, 8)
				return
			}
			for i, e := range a.pickerEntries {
				i, e := i, e
				selected := i == a.pickerSel
				row := ui.Row(c.Key("pick:"+e.Path)).Height(32).Padding(0, 9).Gap(9).
					Radius(radius).AlignItems(ui.Center).Cursor(ui.CursorPointer).
					Role(ui.RoleButton).Focusable().FocusRing(false).Label(e.Name)
				if selected {
					row.Background(k.PanelSel)
				} else if row.Hovered() {
					row.Background(k.Hover)
				}
				if row.Clicked() {
					a.pickerSel = i
				}
				if row.DoubleClicked() || row.Shortcut(0, ui.KeyEnter) {
					a.later(func() { a.setPickerPath(e.Path) })
				}
				if selected {
					row.ScrollIntoView()
				}
				row.Children(func() {
					ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
					ui.Text(c, e.Name).FontSize(12.5).TextColor(k.Text).
						SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
					ui.Spacer(c)
					if row.Hovered() {
						ui.Icon(c, icons.Icon("chevron-right")).Size(13, 13).TextColor(k.TextFaint)
					}
				})
			}
		})
	})
}

func (a *app) pickerFooter(c *ui.Context, k *colors) {
	ui.Row(c).Height(58).Padding(0, 18).Gap(8).AlignItems(ui.Center).Children(func() {
		target := a.pickerTarget()
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, "Open").FontSize(10.5).FontWeight(600).TextColor(k.TextFaint)
			ui.Text(c, shortPath(target)).FontSize(12).TextColor(k.TextMuted).
				SingleLine().Ellipsis("…")
		})
		a.pickerButton(c, k, "Cancel", false, func() {
			a.projectPicker = false
		})
		a.pickerButton(c, k, "Open", true, func() {
			a.later(func() { a.pickerOpen() })
		})
	})
}

func (a *app) pickerButton(c *ui.Context, k *colors, label string, primary bool, fn func()) {
	b := ui.Row(c).Padding(6, 14).Radius(radius).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(label)
	if primary {
		b.Background(k.Text)
		if b.Hovered() {
			b.Background(k.Text.Alpha(0.86))
		}
	} else if b.Hovered() {
		b.Background(k.Hover)
	}
	if b.Clicked() {
		fn()
	}
	b.Children(func() {
		col := k.Text
		if primary {
			col = k.Panel
		}
		ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(col)
	})
}
