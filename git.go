package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

type gitChange struct {
	Path    string // relative to the repository root, for git
	Display string // relative to the project, for people
	Status  string // M, A, D, R, U, C
}

type gitStatusData struct {
	root    string
	prefix  string
	branch  string
	changes []gitChange
	err     string
}

func (a *app) toggleGit() {
	if a.gitOpen {
		a.gitOpen = false
		a.invalidate()
		return
	}
	if a.store == nil || a.store.memory {
		a.fail(errors.New("the diff needs a project on disk"))
		return
	}
	a.gitOpen = true
	a.gitChanges = nil
	a.gitSel = -1
	a.gitDiff = ""
	a.gitDiffPath = ""
	a.gitErr = ""
	a.gitLoading = true
	a.refreshGit()
	a.invalidate()
}

func (a *app) refreshGit() {
	if a.store == nil || a.store.memory {
		return
	}
	a.gitLoading = true
	a.gitErr = ""
	a.gitSeq++
	seq := a.gitSeq
	root := a.store.dir
	if a.win == nil {
		a.applyGitStatus(seq, readGitStatus(root))
		return
	}
	win := a.win
	go func() {
		data := readGitStatus(root)
		win.Update(func() {
			if seq == a.gitSeq {
				a.applyGitStatus(seq, data)
				a.invalidate()
			}
		})
	}()
}

func (a *app) applyGitStatus(seq int, data gitStatusData) {
	if seq != a.gitSeq {
		return
	}
	a.gitLoading = false
	a.gitRoot = data.root
	a.gitPrefix = data.prefix
	a.gitBranch = data.branch
	a.gitChanges = data.changes
	a.gitErr = data.err
	if len(data.changes) == 0 {
		a.gitSel = -1
		a.gitDiff = ""
		a.gitDiffPath = ""
		return
	}
	a.gitSel = min(max(a.gitSel, 0), len(data.changes)-1)
	a.loadGitDiff(a.gitSel)
}

func (a *app) selectGitFile(i int) {
	if i < 0 || i >= len(a.gitChanges) {
		return
	}
	a.gitSel = i
	a.loadGitDiff(i)
	a.invalidate()
}

func (a *app) loadGitDiff(i int) {
	if i < 0 || i >= len(a.gitChanges) {
		return
	}
	change := a.gitChanges[i]
	a.gitDiffPath = change.Display
	a.gitDiff = ""
	a.gitDiffErr = ""
	a.gitDiffLoad = true
	a.gitDiffSeq++
	seq := a.gitDiffSeq
	root := a.gitRoot
	if a.win == nil {
		diff, err := readGitDiff(root, change)
		a.applyGitDiff(seq, diff, err)
		return
	}
	win := a.win
	go func() {
		diff, err := readGitDiff(root, change)
		win.Update(func() {
			if seq == a.gitDiffSeq {
				a.applyGitDiff(seq, diff, err)
				a.invalidate()
			}
		})
	}()
}

func (a *app) applyGitDiff(seq int, diff string, err error) {
	if seq != a.gitDiffSeq {
		return
	}
	a.gitDiffLoad = false
	a.gitDiff = diff
	if err != nil {
		a.gitDiffErr = err.Error()
	}
}

// readGitStatus reads the repository's branch and changed files under
// the project.
func readGitStatus(project string) gitStatusData {
	if resolved, err := filepath.EvalSymlinks(project); err == nil {
		project = resolved
	}
	d := gitStatusData{root: project}
	out, err := runGit(project, "rev-parse", "--show-toplevel")
	if err != nil {
		d.err = "This folder is not a Git repository."
		return d
	}
	d.root = strings.TrimSpace(out)
	if r, err := filepath.Rel(d.root, project); err == nil && r != "." && r != "" {
		d.prefix = filepath.ToSlash(r)
	}
	if branch, err := runGit(d.root, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		d.branch = strings.TrimSpace(branch)
	} else if branch, err := runGit(d.root, "symbolic-ref", "--short", "HEAD"); err == nil {
		d.branch = strings.TrimSpace(branch)
	}
	if d.branch == "" || d.branch == "HEAD" {
		d.branch = "detached"
	}
	pathspec := "."
	if d.prefix != "" {
		pathspec = d.prefix
	}
	status, err := runGit(d.root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", pathspec)
	if err != nil {
		d.err = err.Error()
		return d
	}
	d.changes = parseGitStatus(status, d.prefix)
	return d
}

// parseGitStatus turns porcelain -z output into changes, scoped to prefix.
func parseGitStatus(out, prefix string) []gitChange {
	var changes []gitChange
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		entry := parts[i]
		if len(entry) < 4 {
			continue
		}
		code := entry[:2]
		path := entry[3:]
		if code[0] == 'R' || code[0] == 'C' {
			i++ // the old path follows
		}
		rootPath := path
		display := path
		if prefix != "" {
			rest, ok := strings.CutPrefix(path, prefix+"/")
			if !ok {
				continue
			}
			display = rest
		}
		status := gitStatusLetter(code)
		changes = append(changes, gitChange{
			Path:    rootPath,
			Display: display,
			Status:  status,
		})
	}
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Status == "U" && changes[j].Status != "U" {
			return false
		}
		if changes[j].Status == "U" && changes[i].Status != "U" {
			return true
		}
		return strings.ToLower(changes[i].Display) < strings.ToLower(changes[j].Display)
	})
	return changes
}

func gitStatusLetter(code string) string {
	switch {
	case code == "??":
		return "U"
	case strings.Contains(code, "U"):
		return "C"
	case strings.Contains(code, "D"):
		return "D"
	case strings.Contains(code, "R"):
		return "R"
	case strings.Contains(code, "A"):
		return "A"
	default:
		return "M"
	}
}

// readGitDiff reads one file's diff against HEAD, including staged and
// unstaged work. Untracked files are shown as additions.
func readGitDiff(root string, c gitChange) (string, error) {
	path := c.Path
	if c.Status == "U" {
		return untrackedDiff(root, path, c.Display)
	}
	out, err := runGit(root, "diff", "--no-color", "--no-ext-diff", "HEAD", "--", path)
	if err != nil && (strings.Contains(err.Error(), "unknown revision") ||
		strings.Contains(err.Error(), "bad revision") ||
		strings.Contains(err.Error(), "ambiguous argument")) {
		out, err = runGit(root, "diff", "--no-color", "--no-ext-diff", "--cached", "--", path)
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		out, err = runGit(root, "diff", "--no-color", "--no-ext-diff", "--cached", "--", path)
		if err != nil {
			return "", err
		}
	}
	return out, nil
}

// untrackedDiff makes a new-file diff for a file Git does not track.
func untrackedDiff(root, path, display string) (string, error) {
	abs := filepath.Join(root, filepath.FromSlash(path))
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return "Binary file not shown.\n", nil
	}
	if len(data) > 512*1024 {
		return "File is too large to show here.\n", nil
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\nnew file mode 100644\n--- /dev/null\n+++ b/%s\n", display, display, display)
	fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, line := range lines {
		b.WriteByte('+')
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// runGit runs Git in a folder, with a deadline and a cap on its output.
func runGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	var out, errb cappedBuffer
	out.limit = 2 << 20
	errb.limit = 64 << 10
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", errors.New("git timed out")
	}
	if err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), errors.New(msg)
	}
	return out.String(), nil
}

type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.Len() < b.limit {
		n := min(len(p), b.limit-b.Len())
		_, _ = b.Buffer.Write(p[:n])
	}
	return len(p), nil
}

// gitPage draws the working-tree diff in the editor area.
func (a *app) gitPage(c *ui.Context, k *colors) {
	card := ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Radius(cardR).Clip().
		Background(k.cardFocused).Border(1, k.cardBorder).
		Shadow(0, 1, 2, 0, k.shadow)
	card.Children(func() {
		a.gitHeader(c, k)
		ui.Divider(c)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.gitFiles(c, k)
			ui.Box(c).Width(1).FillHeight().Background(k.panelBorder)
			a.gitDiffPane(c, k)
		})
	})
}

func (a *app) gitHeader(c *ui.Context, k *colors) {
	ui.Row(c).Height(42).Padding(0, 10, 0, 14).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon("git-branch")).Size(16, 16).TextColor(k.textMuted)
		ui.Text(c, "Changes").FontSize(13.5).FontWeight(650).TextColor(k.text)
		if a.gitBranch != "" {
			ui.Row(c).Padding(2, 7).Radius(5).Background(k.hover).Gap(5).AlignItems(ui.Center).
				Children(func() {
					ui.Icon(c, icon("git-branch")).Size(11, 11).TextColor(k.textFaint)
					ui.Text(c, a.gitBranch).FontSize(11).TextColor(k.textMuted)
				})
		}
		if n := len(a.gitChanges); n > 0 {
			ui.Text(c, fmt.Sprintf("%d changes", n)).FontSize(11).TextColor(k.textFaint)
		}
		ui.Spacer(c)
		if a.gitLoading || a.gitDiffLoad {
			ui.Text(c, "Reading…").FontSize(11).TextColor(k.textFaint)
		}
		if iconButton(c, k, "refresh-cw", "Refresh Changes", 26, 14).
			Tooltip("Refresh Changes").Clicked() {
			a.later(func() { a.refreshGit() })
		}
		if iconButton(c, k, "x", "Close Changes", 26, 13).Clicked() {
			a.later(func() { a.gitOpen = false })
		}
	})
}

func (a *app) gitFiles(c *ui.Context, k *colors) {
	ui.Column(c).Width(218).Shrink(0).FillHeight().Padding(8, 8).Gap(2).Children(func() {
		ui.Text(c, "WORKING TREE").FontSize(10).FontWeight(650).TextColor(k.textFaint).
			Padding(2, 7, 5, 7)
		if a.gitErr != "" {
			ui.Text(c, a.gitErr).FontSize(11.5).TextColor(k.danger).Padding(8, 7)
		}
		if len(a.gitChanges) == 0 && !a.gitLoading && a.gitErr == "" {
			ui.Text(c, "No changes").FontSize(12.5).TextColor(k.textFaint).Padding(8, 7)
			return
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			for i, ch := range a.gitChanges {
				i, ch := i, ch
				row := ui.Row(c.Key(fmt.Sprintf("git:%d", i))).Height(28).Padding(0, 7).
					Gap(7).Radius(6).AlignItems(ui.Center).Cursor(ui.CursorPointer).
					Role(ui.RoleButton).Label(ch.Display).Tooltip(ch.Display).
					PopoverMenu(func(m *ui.Menu) { a.nodeMenu(m, ch.Display, false) })
				if i == a.gitSel {
					row.Background(k.panelSel)
				} else if row.Hovered() {
					row.Background(k.hover)
				}
				if row.Clicked() {
					a.selectGitFile(i)
				}
				if row.DoubleClicked() {
					if n := a.store.byRelPath(ch.Display); n != nil {
						a.later(func() { a.openNote(n) })
					}
				}
				row.Children(func() {
					ui.Box(c).Size(17, 17).Radius(4).Center().
						Background(k.hover).Children(func() {
						ui.Text(c, ch.Status).FontSize(10).FontWeight(700).
							TextColor(gitStatusColor(ch.Status, k))
					})
					ui.Column(c).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, filepath.Base(ch.Display)).FontSize(12).FontWeight(500).
							TextColor(k.text).SingleLine().Ellipsis("…")
						if dir := filepath.Dir(ch.Display); dir != "." {
							ui.Text(c, dir).FontSize(10).TextColor(k.textFaint).
								SingleLine().Ellipsis("…")
						}
					})
				})
			}
		})
	})
}

func gitStatusColor(status string, k *colors) ui.Color {
	switch status {
	case "A", "U":
		return k.diffAdd
	case "D":
		return k.diffDel
	case "R", "C":
		return k.diffHunk
	default:
		return k.attention
	}
}

func (a *app) gitDiffPane(c *ui.Context, k *colors) {
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
		ui.Row(c).Height(36).Padding(0, 10, 0, 14).Gap(8).AlignItems(ui.Center).Children(func() {
			path := a.gitDiffPath
			if path == "" {
				path = "No file selected"
			}
			ui.Text(c, path).FontSize(11.5).FontWeight(550).TextColor(k.textMuted).
				SingleLine().Ellipsis("…")
			ui.Spacer(c)
			if a.gitDiffPath != "" {
				if s := a.gitChangeStatus(); s != "" {
					ui.Text(c, s).FontSize(10).FontWeight(700).TextColor(gitStatusColor(s, k))
				}
			}
		})
		ui.Divider(c)
		if a.gitErr != "" {
			ui.Column(c).Fill().Center().Padding(20).Children(func() {
				ui.Icon(c, icon("git-branch")).Size(24, 24).TextColor(k.textFaint)
				ui.Text(c, a.gitErr).FontSize(12.5).TextColor(k.textMuted)
			})
			return
		}
		if len(a.gitChanges) == 0 && !a.gitLoading {
			ui.Column(c).Fill().Center().Gap(7).Children(func() {
				ui.Icon(c, icon("check")).Size(24, 24).TextColor(k.diffAdd)
				ui.Text(c, "Working tree clean").FontSize(13).FontWeight(600).TextColor(k.text)
				ui.Text(c, "There are no changes to review.").FontSize(11.5).TextColor(k.textFaint)
			})
			return
		}
		if a.gitDiffLoad {
			ui.Column(c).Fill().Center().Children(func() {
				ui.Text(c, "Reading diff…").FontSize(12.5).TextColor(k.textFaint)
			})
			return
		}
		if a.gitDiffErr != "" {
			ui.Text(c, a.gitDiffErr).FontSize(12).TextColor(k.danger).Padding(12, 10)
			return
		}
		if strings.TrimSpace(a.gitDiff) == "" {
			ui.Column(c).Fill().Center().Children(func() {
				ui.Text(c, "No changes in this file").FontSize(12.5).TextColor(k.textFaint)
			})
			return
		}
		a.gitDiffLines(c, k)
	})
}

func (a *app) gitChangeStatus() string {
	if a.gitSel >= 0 && a.gitSel < len(a.gitChanges) {
		return a.gitChanges[a.gitSel].Status
	}
	return ""
}

func (a *app) gitDiffLines(c *ui.Context, k *colors) {
	lines := strings.Split(strings.TrimRight(a.gitDiff, "\n"), "\n")
	const maxLines = 3000
	truncated := len(lines) > maxLines
	if truncated {
		lines = lines[:maxLines]
	}
	scroll := ui.Scroll(c).Grow(1).MinHeight(0).Padding(8, 12).TrackScroll(&a.gitDiffScroll)
	scroll.Children(func() {
		for _, line := range lines {
			ui.Text(c, diffLineText(line)).Font(noteFont.Family).FontSize(11.5).
				TextColor(diffLineColor(line, k))
		}
		if truncated {
			ui.Text(c, "… diff truncated").FontSize(11).TextColor(k.textFaint).Padding(6, 0)
		}
	})
}

func diffLineText(line string) string {
	line = strings.ReplaceAll(line, "\t", "    ")
	line = strings.ReplaceAll(line, " ", "\u00a0")
	if line == "" {
		return "\u00a0"
	}
	return line
}

func diffLineColor(line string, k *colors) ui.Color {
	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return k.textFaint
	case strings.HasPrefix(line, "@@"):
		return k.diffHunk
	case strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index "):
		return k.textMuted
	case strings.HasPrefix(line, "+"):
		return k.diffAdd
	case strings.HasPrefix(line, "-"):
		return k.diffDel
	default:
		return k.textMuted
	}
}
