// Package git reads and draws the working tree's status and diff: the
// runner interface that reaches local or server-side Git, the parser for
// its output, the page state, and the view the window shows.
package git

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const headerH = 33

type Change struct {
	Path    string // relative to the repository root, for git
	OldPath string // where a rename or copy came from, for git
	Display string // relative to the project, for people
	Status  string // M, A, D, R, U, C
}

// Runner runs Git and reads files where the project lives: locally, or
// on an SSH host through the session server.
type Runner interface {
	Run(dir string, args ...string) (stdout string, err error)
	// ReadFile reads at most limit bytes of rel under dir; more than
	// limit means the caller should treat the file as too large. The
	// returned slice may hold limit+1 bytes to say so.
	ReadFile(dir, rel string, limit int) ([]byte, error)
	// RealPath resolves symlinks where the project lives. A remote path
	// must be resolved on its host, not here.
	RealPath(path string) (string, error)
}

// LocalRunner runs Git in the local project.
type LocalRunner struct{}

func (LocalRunner) Run(dir string, args ...string) (string, error) { return runGit(dir, args...) }

func (LocalRunner) RealPath(path string) (string, error) { return filepath.EvalSymlinks(path) }

func (LocalRunner) ReadFile(dir, rel string, limit int) ([]byte, error) {
	f, err := os.Open(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, int64(limit)+1))
}

// Host is what the page needs from the window around it.
type Host interface {
	ProjectDir() (string, bool)
	GitRunner() Runner
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

	h      Host
	runner Runner
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
	p.runner = h.GitRunner()
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
	p.runner = h.GitRunner()
	p.Loading = true
	p.Err = ""
	p.Seq++
	seq := p.Seq
	root := mustDir(h)
	if h.Window() == nil {
		p.applyStatus(seq, ReadStatus(p.runner, root))
		return
	}
	win := h.Window()
	go func() {
		data := ReadStatus(p.runner, root)
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
	sel := min(max(p.Sel, 0), len(data.Changes)-1)
	if sel != p.Sel || p.DiffPath != data.Changes[sel].Display ||
		p.Diff == "" && p.DiffErr == "" && !p.DiffLoad {
		p.Sel = sel
		p.loadDiff(sel)
		return
	}
	p.Sel = sel
}

// Reset drops the page's in-flight reads, for a project switch.
func (p *Page) Reset() {
	p.Loading = false
	p.DiffLoad = false
	p.Err = ""
	p.DiffErr = ""
	p.Seq++
	p.DiffSeq++
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
		diff, err := ReadDiff(p.runner, root, change)
		p.applyDiff(seq, diff, err)
		return
	}
	win := h.Window()
	go func() {
		diff, err := ReadDiff(p.runner, root, change)
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
