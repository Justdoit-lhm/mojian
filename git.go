package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	a.gitDiffScroll = ui.ScrollState{}
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

// gitPage draws the working-tree diff in the editor area: changes on
// the left, the selected file's unified diff on the right.
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
	ui.Row(c).Height(headerH).Padding(0, 8, 0, 12).Gap(7).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon("git-branch")).Size(15, 15).TextColor(k.textMuted)
		ui.Text(c, "Changes").FontSize(12.5).FontWeight(600).TextColor(k.text)
		if a.gitBranch != "" {
			ui.Row(c).Padding(2, 7).Radius(5).Background(k.hover).Gap(4).
				AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icon("git-branch")).Size(10, 10).TextColor(k.textFaint)
				ui.Text(c, a.gitBranch).FontSize(11).TextColor(k.textMuted)
			})
		}
		if n := len(a.gitChanges); n > 0 {
			ui.Text(c, gitFileCount(n)).FontSize(10.5).TextColor(k.textFaint)
		}
		ui.Spacer(c)
		if a.gitLoading || a.gitDiffLoad {
			ui.Text(c, "Reading…").FontSize(10.5).TextColor(k.textFaint)
		}
		if iconButton(c, k, "refresh-cw", "Refresh Changes", 25, 14).
			Tooltip("Refresh Changes").Clicked() {
			a.later(func() { a.refreshGit() })
		}
		if iconButton(c, k, "x", "Close Changes", 25, 13).Clicked() {
			a.later(func() { a.gitOpen = false })
		}
	})
}

func gitFileCount(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

func (a *app) gitFiles(c *ui.Context, k *colors) {
	ui.Column(c).Width(248).Shrink(0).FillHeight().Children(func() {
		ui.Row(c).Height(34).Padding(0, 10, 0, 12).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "WORKING TREE").FontSize(10).FontWeight(650).TextColor(k.textFaint)
			ui.Spacer(c)
			if n := len(a.gitChanges); n > 0 {
				ui.Text(c, fmt.Sprintf("%d", n)).FontSize(10).TextColor(k.textFaint).
					Padding(2, 5).Radius(4).Background(k.hover)
			}
		})
		ui.Divider(c)
		if a.gitErr != "" {
			ui.Column(c).Fill().Center().Padding(16).Children(func() {
				ui.Icon(c, icon("git-branch")).Size(22, 22).TextColor(k.textFaint)
				ui.Text(c, "Changes unavailable").FontSize(12).FontWeight(600).
					TextColor(k.text).TextAlign(ui.Center)
				ui.Text(c, a.gitErr).FontSize(11).TextColor(k.textFaint).
					TextAlign(ui.Center).MaxWidth(210)
			})
			return
		}
		if len(a.gitChanges) == 0 {
			ui.Column(c).Fill().Center().Padding(16).Gap(6).Children(func() {
				ui.Icon(c, icon("check")).Size(22, 22).TextColor(k.diffAdd)
				ui.Text(c, "No changes").FontSize(12.5).FontWeight(600).TextColor(k.text)
				if a.gitLoading {
					ui.Text(c, "Reading…").FontSize(11).TextColor(k.textFaint)
				}
			})
			return
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(6, 7).Children(func() {
			for i, ch := range a.gitChanges {
				i, ch := i, ch
				selected := i == a.gitSel
				row := ui.Row(c.Key(fmt.Sprintf("git:%d", i))).Height(38).Padding(0, 7).
					Gap(7).Radius(7).AlignItems(ui.Center).Cursor(ui.CursorPointer).
					Role(ui.RoleButton).Label(ch.Display).Tooltip(ch.Display + " — " + gitStatusLabel(ch.Status)).
					PopoverMenu(func(m *ui.Menu) { a.nodeMenu(m, ch.Display, false) })
				if selected {
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
					ui.Icon(c, fileIcon(ch.Path)).Size(16, 16).TextColor(k.textMuted)
					ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
						ui.Text(c, filepath.Base(ch.Display)).FontSize(12).FontWeight(500).
							TextColor(k.text).SingleLine().Ellipsis("…")
						if dir := filepath.Dir(ch.Display); dir != "." {
							ui.Text(c, dir).FontSize(10.5).TextColor(k.textFaint).
								SingleLine().Ellipsis("…")
						}
					})
					a.gitStatusBadge(c, k, ch.Status)
				})
			}
		})
	})
}

func gitStatusLabel(status string) string {
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

func (a *app) gitChangeStatus() string {
	if a.gitSel >= 0 && a.gitSel < len(a.gitChanges) {
		return a.gitChanges[a.gitSel].Status
	}
	return ""
}

func (a *app) gitStatusBadge(c *ui.Context, k *colors, status string) {
	col := gitStatusColor(status, k)
	ui.Text(c, gitStatusLabel(status)).FontSize(9.5).FontWeight(700).
		TextColor(col).Padding(2, 5).Radius(4).Background(col.Alpha(0.11)).
		Shrink(0).Tooltip(gitStatusLabel(status))
}

func (a *app) gitDiffPane(c *ui.Context, k *colors) {
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
		a.gitDiffHeader(c, k)
		ui.Divider(c)
		a.gitDiffBody(c, k)
	})
}

func (a *app) gitDiffHeader(c *ui.Context, k *colors) {
	ui.Row(c).Height(34).Padding(0, 10, 0, 12).Gap(7).AlignItems(ui.Center).Children(func() {
		path := a.gitDiffPath
		if path == "" {
			ui.Text(c, "No file selected").FontSize(11.5).TextColor(k.textMuted)
			ui.Spacer(c)
			return
		}
		ui.Icon(c, fileIcon(path)).Size(15, 15).TextColor(k.textMuted)
		ui.Text(c, filepath.Base(path)).FontSize(12).FontWeight(550).TextColor(k.text).
			SingleLine().Shrink(0)
		if dir := filepath.Dir(path); dir != "." {
			ui.Text(c, dir).FontSize(10.5).TextColor(k.textFaint).
				SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		}
		ui.Spacer(c)
		if status := a.gitChangeStatus(); status != "" {
			a.gitStatusBadge(c, k, status)
		}
		if add, del := gitDiffStat(a.gitDiff); add > 0 || del > 0 {
			ui.Text(c, fmt.Sprintf("+%d", add)).FontSize(10.5).FontWeight(600).
				TextColor(k.diffAdd).Shrink(0)
			ui.Text(c, fmt.Sprintf("−%d", del)).FontSize(10.5).FontWeight(600).
				TextColor(k.diffDel).Shrink(0)
		}
		if n := a.store.byRelPath(path); n != nil {
			open := ui.Row(c).Padding(4, 9).Radius(6).Cursor(ui.CursorPointer).
				Role(ui.RoleButton).Label("Open File")
			if open.Hovered() {
				open.Background(k.hover)
			}
			if open.Clicked() {
				a.later(func() { a.openNote(n) })
			}
			open.Children(func() {
				ui.Text(c, "Open").FontSize(11.5).FontWeight(550).TextColor(k.text)
			})
		}
	})
}

func (a *app) gitDiffBody(c *ui.Context, k *colors) {
	switch {
	case a.gitErr != "":
		a.gitState(c, k, "git-branch", k.danger, "Changes unavailable", a.gitErr)
	case len(a.gitChanges) == 0 && !a.gitLoading:
		a.gitState(c, k, "check", k.diffAdd, "Working tree clean",
			"There are no changes to review.")
	case a.gitDiffLoad:
		a.gitState(c, k, "git-branch", k.textFaint, "Reading diff…", "")
	case a.gitDiffPath == "":
		a.gitState(c, k, "git-branch", k.textFaint, "Select a change",
			"Pick a changed file on the left to see its diff.")
	case a.gitDiffErr != "":
		a.gitState(c, k, "git-branch", k.danger, "Could not read this diff", a.gitDiffErr)
	case strings.TrimSpace(a.gitDiff) == "":
		a.gitState(c, k, "file-text", k.textFaint, "No text changes",
			"This file has no textual diff.")
	default:
		lines := parseGitDiff(a.gitDiff)
		if len(lines) == 0 {
			a.gitState(c, k, "file-text", k.textFaint, "No text changes",
				"This file has no textual diff.")
			return
		}
		a.gitDiffLines(c, k, lines)
	}
}

func (a *app) gitState(c *ui.Context, k *colors, glyph string, col ui.Color, title, detail string) {
	ui.Column(c).Fill().Center().Padding(20).Gap(8).Children(func() {
		ui.Box(c).Size(42, 42).Radius(12).Background(col.Alpha(0.10)).Center().Children(func() {
			ui.Icon(c, icon(glyph)).Size(21, 21).TextColor(col)
		})
		ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(k.text).
			TextAlign(ui.Center)
		if detail != "" {
			ui.Text(c, detail).FontSize(11.5).TextColor(k.textFaint).
				TextAlign(ui.Center).MaxWidth(460)
		}
	})
}

// gitDiffLine is one parsed line of a unified diff, with the old and new
// line numbers its hunk gives it.
type gitDiffLine struct {
	text string
	old  int
	new  int
	kind byte // ' ', '+', '-', '@', 'm' or '\\'
}

var gitHunkPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func parseGitDiff(diff string) []gitDiffLine {
	var out []gitDiffLine
	oldNo, newNo := 0, 0
	for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		if strings.HasPrefix(line, "\\") {
			out = append(out, gitDiffLine{text: line, kind: '\\'})
			continue
		}
		if strings.HasPrefix(line, "@@") {
			if m := gitHunkPattern.FindStringSubmatch(line); len(m) == 3 {
				oldNo, _ = strconv.Atoi(m[1])
				newNo, _ = strconv.Atoi(m[2])
			}
			out = append(out, gitDiffLine{text: line, kind: '@'})
			continue
		}
		if gitDiffMeta(line) {
			continue
		}
		if line == "" {
			out = append(out, gitDiffLine{old: oldNo, new: newNo})
			oldNo++
			newNo++
			continue
		}
		kind := line[0]
		dl := gitDiffLine{text: line, kind: kind}
		switch kind {
		case '+':
			dl.new = newNo
			newNo++
		case '-':
			dl.old = oldNo
			oldNo++
		default:
			dl.old, dl.new = oldNo, newNo
			oldNo++
			newNo++
		}
		out = append(out, dl)
	}
	return out
}

func gitDiffMeta(line string) bool {
	for _, prefix := range []string{
		"diff --git ", "index ", "new file mode ", "deleted file mode ",
		"similarity index ", "rename from ", "rename to ", "old mode ",
		"new mode ", "--- a/", "+++ b/", "--- /dev/null", "+++ /dev/null",
	} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func gitDiffStat(diff string) (add, del int) {
	for _, line := range strings.Split(diff, "\n") {
		if gitDiffMeta(line) || strings.HasPrefix(line, "@@") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+"):
			add++
		case strings.HasPrefix(line, "-"):
			del++
		}
	}
	return add, del
}

func (a *app) gitDiffLines(c *ui.Context, k *colors, lines []gitDiffLine) {
	const maxLines = 2500
	truncated := len(lines) > maxLines
	if truncated {
		lines = lines[:maxLines]
	}
	scroll := ui.Scroll(c).Grow(1).MinHeight(0).Padding(0, 0, 4, 0).
		TrackScroll(&a.gitDiffScroll)
	scroll.Children(func() {
		for _, line := range lines {
			switch line.kind {
			case '@':
				ui.Row(c).Height(24).FillWidth().Padding(0, 10).AlignItems(ui.Center).
					Background(k.diffHunk.Alpha(0.08)).Children(func() {
					ui.Text(c, diffLineText(line.text)).Font(noteFont.Family).FontSize(11).
						TextColor(k.diffHunk).SingleLine()
				})
			case 'm', '\\':
				ui.Row(c).Height(18).FillWidth().Padding(0, 10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, diffLineText(line.text)).Font(noteFont.Family).FontSize(11).
						TextColor(k.textFaint).SingleLine()
				})
			default:
				row := ui.Row(c).Height(18).FillWidth().ClipX().AlignItems(ui.Center)
				switch line.kind {
				case '+':
					row.Background(k.diffAdd.Alpha(0.10))
				case '-':
					row.Background(k.diffDel.Alpha(0.09))
				}
				row.Children(func() {
					gitGutter(c, k, line.old)
					gitGutter(c, k, line.new)
					ui.Text(c, diffLineText(line.text)).Font(noteFont.Family).FontSize(11.5).
						TextColor(diffLineKindColor(line.kind, k)).SingleLine().Ellipsis("…").
						Shrink(1).MinWidth(0).Padding(0, 0, 0, 8)
				})
			}
		}
		if truncated {
			ui.Text(c, "… diff truncated").FontSize(11).TextColor(k.textFaint).Padding(6, 10)
		}
	})
}

func gitGutter(c *ui.Context, k *colors, n int) {
	text := ""
	if n > 0 {
		text = strconv.Itoa(n)
	}
	ui.Text(c, text).Font(noteFont.Family).FontSize(10).TextColor(k.textFaint).
		Width(38).TextAlign(ui.End).Shrink(0).Padding(0, 5)
}

func diffLineKindColor(kind byte, k *colors) ui.Color {
	switch kind {
	case '+':
		return k.diffAdd
	case '-':
		return k.diffDel
	default:
		return k.textMuted
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
