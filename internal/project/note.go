// Package project holds a project's files as the app edits them: notes
// with their shared editor buffers, viewer assets, and the tree that
// lists them. It has no dependency on internal/app.
package project

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Note is one Markdown file in the vault: its text, its editor state, and
// what was last written to disk.
type Note struct {
	Rel string // the path within the vault, which the layout stores
	// Path is the file on disk, or its name in a browser, where no file
	// system exists.
	Path  string
	Title string
	Saved string // the text as last read or written
	Mod   time.Time
	// Loaded reports whether Saved holds the file. A server project lists
	// its notes without their text and reads one the first time it opens.
	Loaded bool

	Buf *editor.Buffer
}

// Asset is a picture or a PDF in the project: a file shown by a viewer
// rather than edited as text.
type Asset struct {
	Rel  string
	Path string
	Name string
	Size int64
	Mod  time.Time
	// Hash is the server's content hash for a remote asset.
	Hash string
	// Local is where a remote picture or PDF was fetched for its viewer;
	// empty for a local project, where Path is the file.
	Local string

	Once sync.Once
	Img  ui.ImageSource
	Err  error
}

// LocalPath is the file the viewer reads: the fetched copy for a remote
// project, the project file otherwise.
func (f *Asset) LocalPath() string {
	if f.Local != "" {
		return f.Local
	}
	return f.Path
}

// Image decodes the picture once, on the frame that first shows it.
func (f *Asset) Image() (ui.ImageSource, error) {
	f.Once.Do(func() {
		data, err := os.ReadFile(f.LocalPath())
		if err != nil {
			f.Err = err
			return
		}
		f.Img, f.Err = ui.DecodeImage(data)
	})
	return f.Img, f.Err
}

// Text is the note as it stands, including edits not yet saved.
func (n *Note) Text() string {
	if n.Buf != nil {
		return n.Buf.String()
	}
	return n.Saved
}

// Dirty reports whether the note has edits that are not on disk.
func (n *Note) Dirty() bool { return n.Buf != nil && n.Buf.Modified() }

// Buffer is the note's one buffer, shared by every pane showing it, so
// an edit in one shows in the others.
func (n *Note) Buffer() *editor.Buffer {
	if n.Buf == nil {
		n.Buf = editor.NewBuffer(n.Saved)
		n.Buf.SetLanguage(editor.LanguageFor(n.Path))
	}
	return n.Buf
}

// NewState is one pane's view of the note's buffer, with its own caret,
// selection and scroll.
func (n *Note) NewState() *editor.State {
	return &editor.State{
		Buffer:       n.Buffer(),
		Font:         theme.NoteFont,
		LineNumbers:  LineNumbersFor(n.Path),
		SoftWrap:     WrapFor(n.Path),
		NoActiveLine: true,
		Placeholder:  "Start writing…",
		Label:        n.Title,
		Bindings:     editorBindings(),
	}
}

// editorBindings keeps the window's pane family working in the browser
// and the harness, and keeps the editor commands those keys replace on a
// key of their own. The menu bar takes Ghostty's keys before the editor
// on macOS, but a browser build has no menu, so the editor would take
// them and the pane commands would never see them. A NoCommand binding
// hands the key back to the window.
func editorBindings() []editor.Binding {
	add := ui.Ctrl | ui.Super
	if runtime.GOOS != "darwin" {
		add = ui.Ctrl | ui.Alt | ui.Shift
	}
	b := []editor.Binding{
		// Pane focus owns Command+Option and every arrow.
		{Mods: ui.Cmd | ui.Alt, Key: ui.KeyLeft, Command: editor.NoCommand},
		{Mods: ui.Cmd | ui.Alt, Key: ui.KeyRight, Command: editor.NoCommand},
		{Mods: ui.Cmd | ui.Alt, Key: ui.KeyUp, Command: editor.NoCommand},
		{Mods: ui.Cmd | ui.Alt, Key: ui.KeyDown, Command: editor.NoCommand},
		// Multi-cursor moves aside.
		{Mods: add, Key: ui.KeyUp, Command: editor.AddCursorAbove},
		{Mods: add, Key: ui.KeyDown, Command: editor.AddCursorBelow},
		// Split Right owns Command+D, and Zoom Pane owns the platform's
		// Shift+Enter; Select Next and New Line Above move aside.
		{Mods: ui.Ctrl | ui.Shift, Key: ui.KeyD, Command: editor.SelectNext},
		{Mods: ui.Alt | ui.Shift, Key: ui.KeyEnter, Command: editor.NewlineAbove},
	}
	if runtime.GOOS == "darwin" {
		b = append(b,
			editor.Binding{Mods: ui.Super, Key: ui.KeyD, Command: editor.NoCommand},
			editor.Binding{Mods: ui.Super | ui.Shift, Key: ui.KeyEnter, Command: editor.NoCommand},
		)
	} else {
		b = append(b,
			editor.Binding{Mods: ui.Ctrl | ui.Shift, Key: ui.KeyEnter, Command: editor.NoCommand},
		)
	}
	return b
}

// LineNumbersFor keeps line numbers off prose and on source code.
func LineNumbersFor(path string) editor.LineNumbers {
	if IsMarkdownPath(path) {
		return editor.LineNumbersOff
	}
	return editor.LineNumbersOn
}

// WrapFor wraps prose, and scrolls code sideways rather than folding a
// line onto the next.
func WrapFor(path string) bool {
	if IsMarkdownPath(path) {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".text", ".rst", ".org", ".tex", ".nfo":
		return true
	}
	return false
}

// NoteTitle is the first heading of the note, else its file name without
// the extension.
func NoteTitle(body, path string) string {
	if !IsMarkdownPath(path) {
		return filepath.Base(path)
	}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		if h := strings.TrimSpace(strings.TrimLeft(line, "#")); h != "" {
			return h
		}
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.TrimSpace(stem) == "" {
		return "Untitled"
	}
	return stem
}

// IsMarkdownPath reports whether a file is Markdown: prose, with soft
// wrap, rather than source code.
func IsMarkdownPath(path string) bool {
	l := editor.LanguageFor(path)
	return l != nil && l.Name == "Markdown"
}
