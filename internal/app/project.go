package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"

	"mojian/internal/icons"
	"mojian/internal/motion"
)

// projectName is the last part of the open project's path.
func (a *app) projectName() string {
	if a.remotePath != "" {
		return filepath.Base(a.remotePath)
	}
	if a.store == nil || a.store.IsMemory() {
		return "Mojian"
	}
	return filepath.Base(a.store.Dir)
}

// projectPath is the open project's path as a person reads it.
func (a *app) projectPath() string {
	if a.remotePath != "" {
		return a.serverAlias + ":" + a.remotePath
	}
	if a.store == nil {
		return ""
	}
	if a.store.IsMemory() {
		return "in memory"
	}
	return shortPath(a.store.Dir)
}

func shortPath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		if rest, ok := strings.CutPrefix(p, home); ok {
			return "~" + rest
		}
	}
	return p
}

// recentProjects lists the projects that still exist, the open one first.
func (a *app) recentProjects() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		seen[p] = true
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			out = append(out, p)
		}
	}
	if a.store != nil && !a.store.IsMemory() {
		add(a.store.Dir)
	}
	add(a.project)
	for _, p := range a.projects {
		add(p)
	}
	return out
}

// openProject makes dir the open project, saving the one before it.
func (a *app) openProject(dir string) {
	if dir == "" || a.remoteBusy {
		return
	}
	if a.harness {
		// Other projects open through the session server, which this
		// binary does not run: keep the endpoint on its fixture.
		a.fail(errors.New("the UI harness keeps its disposable project"))
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		a.fail(err)
		return
	}
	// A project reached through a symlink is stored at its target, so
	// the folder lists its files and compares equal to a real path.
	abs = resolveProjectRoot(abs)
	a.initWorkspaces()
	if ws := a.workspaces[workspaceKeyLocal(abs)]; ws != nil {
		a.switchWorkspace(ws)
		a.rememberProject(abs)
		a.saveSettings()
		return
	}
	if a.remotePath == "" && a.store != nil && !a.store.IsMemory() && a.store.Dir == abs {
		a.projectOpen = false
		a.invalidate()
		return
	}
	// Save while the current server is still connected, and keep its
	// project in the workspace cache: switching back is then a restore.
	if err := a.saveNow(); err != nil {
		a.fail(fmt.Errorf("could not save the open project: %w", err))
		return
	}
	if a.layoutPath != "" {
		a.saveLayout()
	}
	a.layoutDirty = false
	a.dropPickerRemote()
	s, err := a.openLocalStore(abs)
	if err != nil {
		a.fail(err)
		return
	}
	a.stashActiveWorkspace()
	ws := &workspace{
		key:        workspaceKeyLocal(abs),
		project:    abs,
		store:      s,
		layoutPath: a.projectLayoutPath(abs),
		treeOpen:   true,
		active:     -1,
		openDirs:   map[string]bool{},
	}
	a.workspaces[ws.key] = ws
	a.activateWorkspace(ws)
	a.rememberProject(abs)
	if !a.restoreLayout() {
		a.focusFirstNote()
	}
	if t := a.tab(); t != nil {
		a.focusReq = t.editorPane()
	}
	a.revealTab = true
	a.saveLayout()
	a.saveSettings()
	a.updateTitle()
	a.invalidate()
}

// projectChip is the title bar's left end: the open project, and the
// projects to switch to.
func (a *app) projectChip(c *ui.Context, k *colors) {
	if a.store == nil {
		return
	}
	chip := ui.Row(c).Gap(7).Padding(3, 8, 3, 6).Radius(radius).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Switch project: " + a.projectName()).
		Tooltip(a.projectPath() + "  " + nativeKeys("⌘O")).MaxWidth(230).Shrink(0).ClipX().
		Transition(motion.Colors)
	if chip.Hovered() || a.projectOpen {
		chip.Background(k.Hover)
	}
	drop := float32(0)
	if a.projectOpen {
		drop = 1
	}
	drop = chip.AnimateTo("project-menu", drop, motion.PopoverIn)
	if chip.Clicked() && !a.store.IsMemory() {
		a.projectOpen = !a.projectOpen
	}
	chip.Children(func() {
		ui.Icon(c, icons.FolderIcon()).Size(19, 19).TextColor(k.TextMuted)
		ui.Text(c, a.projectName()).FontSize(13.5).FontWeight(700).TextColor(k.Text.Alpha(0.9)).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		ui.Icon(c, icons.Icon("chevron-down")).Size(13, 13).TextColor(k.TextFaint)
	})
	if a.store.IsMemory() {
		return
	}
	ui.PopoverBase(c, chip, &a.projectOpen, func(panel ui.Element) {
		panel.Width(304).Padding(9).Radius(radius).Background(k.Panel).Border(1, k.PanelBorder).
			Shadow(0, 12, 32, 0, k.ShadowFocused).Gap(1).
			Translate(0, -4*(1-drop)).Opacity(0.45 + 0.55*drop)
		if a.remotePath != "" {
			a.remoteProjectRows(c, k)
			ui.Divider(c).Margin(6, 0)
			row := ui.Row(c).Height(34).Padding(0, 8).Gap(8).Radius(radius).
				AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleButton).
				Label("Open Remote Folder…")
			if row.Hovered() {
				row.Background(k.Hover)
			}
			if row.Clicked() {
				a.projectOpen = false
				a.later(func() { a.openRemotePicker(a.serverAlias) })
			}
			row.Children(func() {
				ui.Icon(c, icons.Icon("folder-plus")).Size(15, 15).TextColor(k.TextMuted)
				ui.Text(c, "Open Remote Folder…").FontSize(12.5).TextColor(k.Text)
			})
			return
		}
		ui.Row(c).Padding(4, 8, 5, 8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Projects").FontSize(11).FontWeight(600).TextColor(k.TextFaint)
			ui.Spacer(c)
			ui.Text(c, nativeKeys("⌘O")).FontSize(11).TextColor(k.TextFaint)
		})
		for _, p := range a.recentProjects() {
			a.projectRow(c, k, p)
		}
		if runtime.GOOS != "js" {
			ui.Divider(c).Margin(6, 0)
			row := ui.Row(c).Height(34).Padding(0, 8).Gap(8).Radius(radius).
				AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleButton).
				Label("Open Folder…")
			if row.Hovered() {
				row.Background(k.Hover)
			}
			row.Tooltip(a.tip("Open Folder", cmdOpenProject))
			if row.Clicked() {
				a.projectOpen = false
				a.openProjectPicker()
			}
			row.Children(func() {
				ui.Icon(c, icons.Icon("folder-plus")).Size(15, 15).TextColor(k.TextMuted)
				ui.Text(c, "Open Folder…").FontSize(12.5).TextColor(k.Text)
			})
		}
	})
}

func (a *app) projectRow(c *ui.Context, k *colors, p string) {
	current := a.store != nil && !a.store.IsMemory() && filepath.Clean(a.store.Dir) == filepath.Clean(p)
	a.projectRowBase(c, k, projectRowSpec{
		key:        "project:" + p,
		label:      a.projectLabel(p),
		title:      a.projectLabel(p),
		detail:     shortPath(p),
		closeLabel: p,
		current:    current,
		open: func() {
			a.projectOpen = false
			a.later(func() { a.openProject(p) })
		},
		close: func() {
			a.projectOpen = false
			a.later(func() { a.closeLocalProject(p) })
		},
	})
}

// projectRowSpec is one row of the project switcher. Its trash button
// removes the project from the recent or remembered list.
type projectRowSpec struct {
	key        string
	label      string
	title      string
	detail     string
	closeLabel string
	current    bool
	open       func()
	close      func()
}

func (a *app) projectRowBase(c *ui.Context, k *colors, s projectRowSpec) {
	row := ui.Row(c.Key(s.key)).Height(42).Padding(0, 8).MarginY(2).Gap(9).Radius(radius).
		AlignItems(ui.Center)
	if s.current {
		row.Background(k.PanelSel)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	row.Children(func() {
		info := ui.Row(c).Grow(1).MinWidth(0).Gap(9).AlignItems(ui.Center).
			Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(s.label)
		if info.Clicked() && s.open != nil {
			s.open()
		}
		info.Children(func() {
			ui.Icon(c, icons.FolderIcon()).Size(16, 16).TextColor(k.TextMuted)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, s.title).FontSize(12.5).FontWeight(500).TextColor(k.Text).
					SingleLine().Ellipsis("…")
				ui.Text(c, s.detail).FontSize(11).TextColor(k.TextFaint).
					SingleLine().Ellipsis("…")
			})
		})
		remove := iconButton(c, k, "trash-2", "Remove project "+s.closeLabel, 22, 13).
			Tooltip("Remove Project")
		if remove.Clicked() && s.close != nil {
			s.close()
		}
	})
}

// closeLocalProject forgets one local project from the recent list. When
// it is loaded it is unloaded too, and the open project saves first and
// switches to the next recent project, or leaves the empty workspace.
func (a *app) closeLocalProject(dir string) {
	a.projectOpen = false
	dir = filepath.Clean(dir)
	key := workspaceKeyLocal(dir)
	a.initWorkspaces()
	active := a.remotePath == "" && a.store != nil && !a.store.IsMemory() &&
		filepath.Clean(a.store.Dir) == dir
	ws := a.workspaces[key]
	if active {
		if err := a.saveNow(); err != nil {
			a.fail(fmt.Errorf("could not save the open project: %w", err))
			return
		}
		a.dropPaneOverlays(nil)
	}
	if ws != nil {
		closeWorkspaceTerminals(ws)
		delete(a.workspaces, key)
	}
	kept := a.projects[:0]
	for _, p := range a.projects {
		if filepath.Clean(p) != dir {
			kept = append(kept, p)
		}
	}
	a.projects = kept
	if filepath.Clean(a.project) == dir {
		a.project = ""
	}
	if active {
		a.abandonActiveWorkspace()
		for _, p := range a.projects {
			p = filepath.Clean(p)
			if p == "" || p == "." {
				continue
			}
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				a.project = p
				break
			}
		}
		fallback := a.project
		a.updateTitle()
		a.later(func() {
			if fallback != "" {
				a.openProject(fallback)
			}
		})
	}
	a.saveSettings()
	a.invalidate()
}

func (a *app) projectLabel(p string) string {
	if base := filepath.Base(p); base != "" && base != "." {
		return base
	}
	return p
}
