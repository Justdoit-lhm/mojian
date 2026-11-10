package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"mojian/internal/gitenv"
	"mojian/internal/platform"
)

// ReadStatus reads the repository's branch and changed files under
// the project.
func ReadStatus(r Runner, project string) StatusData {
	if r == nil {
		r = LocalRunner{}
	}
	if resolved, err := r.RealPath(project); err == nil && resolved != "" {
		project = resolved
	}
	d := StatusData{Root: project}
	out, err := r.Run(project, "rev-parse", "--show-toplevel")
	if err != nil {
		d.Err = "This folder is not a Git repository."
		return d
	}
	d.Root = strings.TrimSpace(out)
	if r, err := filepath.Rel(d.Root, project); err == nil && r != "." && r != "" {
		d.Prefix = filepath.ToSlash(r)
	}
	if branch, err := r.Run(d.Root, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		d.Branch = strings.TrimSpace(branch)
	} else if branch, err := r.Run(d.Root, "symbolic-ref", "--short", "HEAD"); err == nil {
		d.Branch = strings.TrimSpace(branch)
	}
	if d.Branch == "" || d.Branch == "HEAD" {
		d.Branch = "detached"
	}
	pathspec := "."
	if d.Prefix != "" {
		pathspec = d.Prefix
	}
	status, err := r.Run(d.Root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", pathspec)
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
		oldPath := ""
		if code[0] == 'R' || code[0] == 'C' {
			// The old path follows: keep it so git can pair the rename
			// instead of reporting the whole file as new.
			if i+1 < len(parts) {
				oldPath = parts[i+1]
			}
			i++
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
			OldPath: oldPath,
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
func ReadDiff(r Runner, root string, c Change) (string, error) {
	if r == nil {
		r = LocalRunner{}
	}
	path := c.Path
	if c.Status == "U" {
		return untrackedDiff(r, root, path, c.Display)
	}
	paths := []string{path}
	if c.OldPath != "" && c.OldPath != path {
		paths = append([]string{c.OldPath}, paths...)
	}
	out, err := r.Run(root, append([]string{"diff", "--no-color", "--no-ext-diff", "HEAD", "--"}, paths...)...)
	if err != nil && (strings.Contains(err.Error(), "unknown revision") ||
		strings.Contains(err.Error(), "bad revision") ||
		strings.Contains(err.Error(), "ambiguous argument")) {
		out, err = r.Run(root, append([]string{"diff", "--no-color", "--no-ext-diff", "--cached", "--"}, paths...)...)
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		out, err = r.Run(root, append([]string{"diff", "--no-color", "--no-ext-diff", "--cached", "--"}, paths...)...)
		if err != nil {
			return "", err
		}
	}
	return out, nil
}

// untrackedDiff makes a new-file diff for a file Git does not track.
func untrackedDiff(r Runner, root, path, display string) (string, error) {
	data, err := r.ReadFile(root, path, 512*1024)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "is a directory") {
			return "This is a folder Git does not track.\n", nil
		}
		return "", err
	}
	if len(data) > 512*1024 {
		return "File is too large to show here.\n", nil
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return "Binary file not shown.\n", nil
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
	platform.HideConsole(cmd)
	cmd.Env = append(gitEnv(), "GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
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

// gitEnv is the inherited environment without the variables a shell
// could set to point Git at another repository.
func gitEnv() []string { return gitenv.Env() }

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

var gitHunkPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// gitDiffLine is one parsed line of a unified diff, with the old and new
// line numbers its hunk gives it.
type gitDiffLine struct {
	text string
	old  int
	new  int
	kind byte // ' ', '+', '-', '@', 'm' or '\\'
}

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
