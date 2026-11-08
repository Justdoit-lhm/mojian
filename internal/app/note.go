package app

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// note is one Markdown file in the vault: its text, its editor state, and
// what was last written to disk.
type note struct {
	Rel string // the path within the vault, which the layout stores
	// Path is the file on disk, or its name in a browser, where no file
	// system exists.
	Path  string
	Title string
	Saved string // the text as last read or written
	Mod   time.Time
	// loaded reports whether Saved holds the file. A server project lists
	// its notes without their text and reads one the first time it opens.
	loaded bool

	buf *editor.Buffer
}

// assetFile is a picture or a PDF in the project: a file shown by a
// viewer rather than edited as text.
// assetFile is a picture or a PDF in the project: a file shown by a
// viewer rather than edited as text.
type assetFile struct {
	Rel  string
	Path string
	Name string
	Size int64
	Mod  time.Time
	// Hash is the server's content hash for a remote asset.
	Hash string
	// local is where a remote picture or PDF was fetched for its viewer;
	// empty for a local project, where Path is the file.
	local string

	once sync.Once
	img  ui.ImageSource
	err  error
}

// localPath is the file the viewer reads: the fetched copy for a remote
// project, the project file otherwise.
// localPath is the file the viewer reads: the fetched copy for a remote
// project, the project file otherwise.
func (f *assetFile) localPath() string {
	if f.local != "" {
		return f.local
	}
	return f.Path
}

// image decodes the picture once, on the frame that first shows it.
// image decodes the picture once, on the frame that first shows it.
func (f *assetFile) image() (ui.ImageSource, error) {
	f.once.Do(func() {
		data, err := os.ReadFile(f.localPath())
		if err != nil {
			f.err = err
			return
		}
		f.img, f.err = ui.DecodeImage(data)
	})
	return f.img, f.err
}

// text is the note as it stands, including edits not yet saved.
// text is the note as it stands, including edits not yet saved.
func (n *note) text() string {
	if n.buf != nil {
		return n.buf.String()
	}
	return n.Saved
}

// dirty reports whether the note has edits that are not on disk.
// dirty reports whether the note has edits that are not on disk.
func (n *note) dirty() bool { return n.buf != nil && n.buf.Modified() }

// buffer is the note's one buffer, shared by every pane showing it, so
// an edit in one shows in the others.
// buffer is the note's one buffer, shared by every pane showing it, so
// an edit in one shows in the others.
func (n *note) buffer() *editor.Buffer {
	if n.buf == nil {
		n.buf = editor.NewBuffer(n.Saved)
		n.buf.SetLanguage(editor.LanguageFor(n.Path))
	}
	return n.buf
}

// newState is one pane's view of the note's buffer, with its own caret,
// selection and scroll.
// newState is one pane's view of the note's buffer, with its own caret,
// selection and scroll.
func (n *note) newState() *editor.State {
	return &editor.State{
		Buffer:       n.buffer(),
		Font:         noteFont,
		LineNumbers:  lineNumbersFor(n.Path),
		SoftWrap:     wrapFor(n.Path),
		NoActiveLine: true,
		Placeholder:  "Start writing…",
		Label:        n.Title,
	}
}

func lineNumbersFor(path string) editor.LineNumbers {
	if isMarkdownPath(path) {
		return editor.LineNumbersOff
	}
	return editor.LineNumbersOn
}

// wrapFor wraps prose, and scrolls code sideways rather than folding a
// line onto the next.
// wrapFor wraps prose, and scrolls code sideways rather than folding a
// line onto the next.
func wrapFor(path string) bool {
	if isMarkdownPath(path) {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".text", ".rst", ".org", ".tex", ".nfo":
		return true
	}
	return false
}

// noteTitle is the first heading of the note, else its file name without
// the extension.
// noteTitle is the first heading of the note, else its file name without
// the extension.
func noteTitle(body, path string) string {
	if !isMarkdownPath(path) {
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

// store is a project: the text files it holds, by path. Its backend is a
// local folder, a session server, or the browser's memory.
// isMarkdownPath reports whether a file is Markdown: prose, with soft
// wrap, rather than source code.
func isMarkdownPath(path string) bool {
	l := editor.LanguageFor(path)
	return l != nil && l.Name == "Markdown"
}
