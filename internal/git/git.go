package git

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

	"mojian/internal/icons"
	"mojian/internal/motion"
	"mojian/internal/theme"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const headerH = 33

type Change struct {
	Path    string // relative to the repository root, for git
	Display string // relative to the project, for people
	Status  string // M, A, D, R, U, C
}

// Host is what the page needs from the window around it.
type Host interface {
	ProjectDir() (string, bool)
	Window() *mygo.Window
	Invalidate()
	Fail(error)
	HasPath(string) bool
	OpenPath(string)
	FileMenu(*ui.Menu, string)
}

// Page is the working-tree diff: its state and its view.
type Page struct {
	Open       bool
	Loading    bool
	Err        string
	Root       string
	Prefix     string
	Branch     string
	Changes    []Change
	Sel        int
	Diff       string
	DiffPath   string
	DiffErr    string
	DiffLoad   bool
	Seq        int
	DiffSeq    int
	DiffScroll ui.ScrollState

	h Host
}

type StatusData struct {
	Root    string
	Prefix  string
	Branch  string
	Changes []Change
	Err     string
}

func (p *Page) Toggle(h Host) {
	p.h = h
	if p.Open {
		p.Open = false
		h.Invalidate()
		return
	}
	if !hasProject(h) {
		h.Fail(errors.New("the diff needs a project on disk"))
		return
	}
	p.Open = true
	p.Changes = nil
	p.Sel = -1
	p.Diff = ""
	p.DiffPath = ""
	p.Err = ""
	p.Loading = true
	p.Refresh()
	h.Invalidate()
}

func (p *Page) Refresh() {
	h := p.h
	if !hasProject(h) {
		return
	}
	p.Loading = true
	p.Err = ""
	p.Seq++
	seq := p.Seq
	root := mustDir(h)
	if h.Window() == nil {
		p.applyStatus(seq, ReadStatus(root))
		return
	}
	win := h.Window()
	go func() {
		data := ReadStatus(root)
		win.Update(func() {
			if seq == p.Seq {
				p.applyStatus(seq, data)
				h.Invalidate()
			}
		})
	}()
}

func (p *Page) applyStatus(seq int, data StatusData) {
	if seq != p.Seq {
		return
	}
	p.Loading = false
	p.Root = data.Root
	p.Prefix = data.Prefix
	p.Branch = data.Branch
	p.Changes = data.Changes
	p.Err = data.Err
	if len(data.Changes) == 0 {
		p.Sel = -1
		p.Diff = ""
		p.DiffPath = ""
		return
	}
	p.Sel = min(max(p.Sel, 0), len(data.Changes)-1)
	p.loadDiff(p.Sel)
}

func (p *Page) Select(i int) {
	h := p.h
	if i < 0 || i >= len(p.Changes) {
		return
	}
	p.Sel = i
	p.loadDiff(i)
	h.Invalidate()
}

func (p *Page) loadDiff(i int) {
	h := p.h
	if i < 0 || i >= len(p.Changes) {
		return
	}
	change := p.Changes[i]
	p.DiffPath = change.Display
	p.Diff = ""
	p.DiffErr = ""
	p.DiffLoad = true
	p.DiffScroll = ui.ScrollState{}
	p.DiffSeq++
	seq := p.DiffSeq
	root := p.Root
	if h.Window() == nil {
		diff, err := ReadDiff(root, change)
		p.applyDiff(seq, diff, err)
		return
	}
	win := h.Window()
	go func() {
		diff, err := ReadDiff(root, change)
		win.Update(func() {
			if seq == p.DiffSeq {
				p.applyDiff(seq, diff, err)
				h.Invalidate()
			}
		})
	}()
}

func (p *Page) applyDiff(seq int, diff string, err error) {
	if seq != p.DiffSeq {
		return
	}
	p.DiffLoad = false
	p.Diff = diff
	if err != nil {
		p.DiffErr = err.Error()
	}
}

// ReadStatus reads the repository's branch and changed files under
// the project.
func ReadStatus(project string) StatusData {
	if resolved, err := filepath.EvalSymlinks(project); err == nil {
		project = resolved
	}
	d := StatusData{Root: project}
	out, err := runGit(project, "rev-parse", "--show-toplevel")
	if err != nil {
		d.Err = "This folder is not a Git repository."
		return d
	}
	d.Root = strings.TrimSpace(out)
	if r, err := filepath.Rel(d.Root, project); err == nil && r != "." && r != "" {
		d.Prefix = filepath.ToSlash(r)
	}
	if branch, err := runGit(d.Root, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		d.Branch = strings.TrimSpace(branch)
	} else if branch, err := runGit(d.Root, "symbolic-ref", "--short", "HEAD"); err == nil {
		d.Branch = strings.TrimSpace(branch)
	}
	if d.Branch == "" || d.Branch == "HEAD" {
		d.Branch = "detached"
	}
	pathspec := "."
	if d.Prefix != "" {
		pathspec = d.Prefix
	}
	status, err := runGit(d.Root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", pathspec)
	if err != nil {
		d.Err = err.Error()
		return d
	}
	d.Changes = parseStatus(status, d.Prefix)
	return d
}

// parseStatus turns porcelain -z output into changes, scoped to prefix.
func parseStatus(out, prefix string) []Change {
	var changes []Change
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
		status := statusLetter(code)
		changes = append(changes, Change{
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

func statusLetter(code string) string {
	switch {
	case code == "??":
		return "U"
	case code == "AA" || code == "DD" || strings.Contains(code, "U"):
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

// ReadDiff reads one file's diff against HEAD, including staged and
// unstaged work. Untracked files are shown as additions.
func ReadDiff(root string, c Change) (string, error) {
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
type gitDiffLine struct {
	text string
	old  int
	new  int
	kind byte // ' ', '+', '-', '@', 'm' or '\\'
}

var gitHunkPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func ParseDiff(diff string) []gitDiffLine {
	var out []gitDiffLine
	oldNo, newNo := 0, 0
	inHunk := false
	for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			inHunk = false
			continue
		}
		// File headers only appear before a hunk. Inside one, "--- a/x"
		// is a deleted line whose text starts with "--", not metadata.
		if !inHunk && diffMeta(line) {
			continue
		}
		if strings.HasPrefix(line, "\\") {
			out = append(out, gitDiffLine{text: line, kind: '\\'})
			continue
		}
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			if m := gitHunkPattern.FindStringSubmatch(line); len(m) == 3 {
				oldNo, _ = strconv.Atoi(m[1])
				newNo, _ = strconv.Atoi(m[2])
			}
			out = append(out, gitDiffLine{text: line, kind: '@'})
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

func diffMeta(line string) bool {
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

func DiffStat(diff string) (add, del int) {
	inHunk := false
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			inHunk = false
			continue
		}
		if !inHunk {
			if strings.HasPrefix(line, "@@") {
				inHunk = true
			}
			continue
		}
		if strings.HasPrefix(line, "\\") {
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
