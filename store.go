package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

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
	words int

	buf *editor.Buffer
}

// text is the note as it stands, including edits not yet saved.
func (n *note) text() string {
	if n.buf != nil {
		return n.buf.String()
	}
	return n.Saved
}

// dirty reports whether the note has edits that are not on disk.
func (n *note) dirty() bool { return n.buf != nil && n.buf.Modified() }

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

// wordCount counts what a reader would call words, near enough: runs of
// text between spaces. Markdown punctuation counts with them.
func wordCount(s string) int { return len(strings.Fields(s)) }

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

// store is a project: the text files it holds, by path.
type store struct {
	dir    string
	memory bool
	notes  []*note
	byPath map[string]*note
	byRel  map[string]*note
	seq    int
	tree   *fileNode
}

// openStore opens the folder at $MOJIAN_VAULT, else ~/Documents/Mojian,
// and creates a welcome note in a folder of its own. In a browser there
// is no file system: the files live in memory for the session.
func openStore() (*store, error) {
	if runtime.GOOS == "js" {
		s := &store{memory: true, byPath: map[string]*note{}, byRel: map[string]*note{}}
		s.add(&note{Path: "Welcome.md", Title: "Welcome to Mojian", Saved: welcomeNote, Mod: time.Now()})
		s.add(&note{Path: "Reading notes.md", Title: "Reading notes", Saved: sampleNote, Mod: time.Now().Add(-time.Minute)})
		s.buildTree()
		return s, nil
	}
	dir := os.Getenv("MOJIAN_VAULT")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, "Documents", "Mojian")
	}
	s, err := openStoreAt(dir)
	if err != nil {
		return nil, err
	}
	if len(s.notes) == 0 {
		path := filepath.Join(s.dir, "Welcome.md")
		if err := s.write(path, welcomeNote); err != nil {
			return nil, err
		}
		if err := s.refresh(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// openStoreAt opens dir as a project, creating the folder when it is
// missing. It leaves an empty folder empty: only the app's own default
// project gets a welcome note.
func openStoreAt(dir string) (*store, error) {
	if runtime.GOOS == "js" {
		return nil, errors.New("folders need the desktop app")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	s := &store{dir: abs, byPath: map[string]*note{}, byRel: map[string]*note{}}
	if err := s.refresh(); err != nil {
		return nil, err
	}
	return s, nil
}

// refresh adds the text files that appeared in the vault since the last
// look, in every folder. Files already open keep their text and their
// place.
func (s *store) refresh() error {
	if s.memory {
		s.buildTree()
		return nil
	}
	err := filepath.WalkDir(s.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable folder is skipped, not fatal
		}
		if d.IsDir() {
			if path != s.dir && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !isTextFile(path) {
			return nil
		}
		if _, ok := s.byPath[path]; ok {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		n := &note{
			Path:  path,
			Saved: string(data),
			Mod:   info.ModTime(),
		}
		if isMarkdownPath(path) {
			n.words = wordCount(n.Saved)
		}
		n.Title = noteTitle(n.Saved, path)
		s.add(n)
		return nil
	})
	if err != nil {
		return err
	}
	sort.SliceStable(s.notes, func(i, j int) bool { return s.notes[i].Mod.After(s.notes[j].Mod) })
	s.buildTree()
	return nil
}

// skipDir keeps generated trees and version control out of the vault
// view; a code project's own sources still show.
func skipDir(name string) bool {
	switch name {
	case ".git", ".svn", ".hg", "node_modules", "__pycache__", ".venv", "venv",
		"dist", "build", ".next", ".cache", ".turbo", "target", "vendor",
		".idea", ".gradle", "Pods", "DerivedData":
		return true
	}
	return false
}

// isMarkdownPath reports whether a file is Markdown: prose, with soft
// wrap and a word count, rather than source code.
func isMarkdownPath(path string) bool {
	l := editor.LanguageFor(path)
	return l != nil && l.Name == "Markdown"
}

// textExts are text files MyGo has no highlighter for, or the editor
// shows as plain text.
var textExts = map[string]bool{
	".txt": true, ".text": true, ".log": true, ".csv": true, ".tsv": true,
	".rst": true, ".org": true, ".tex": true, ".bib": true, ".nfo": true,
}

// binaryExts are never read as text, even if a prefix could pass.
var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".ico": true, ".icns": true, ".pdf": true, ".zip": true, ".gz": true,
	".tar": true, ".7z": true, ".rar": true, ".dmg": true, ".app": true,
	".mp3": true, ".mp4": true, ".mov": true, ".woff": true, ".woff2": true,
	".ttf": true, ".otf": true, ".exe": true, ".dylib": true, ".so": true,
	".a": true, ".o": true, ".class": true, ".jar": true, ".wasm": true,
	".sqlite": true, ".db": true,
}

// isTextFile reports whether a file in the vault opens in the editor: a
// language MyGo highlights, a plain text kind, or a file whose start
// looks like text.
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if binaryExts[ext] {
		return false
	}
	if editor.LanguageFor(path) != nil || textExts[ext] {
		return true
	}
	return looksText(path)
}

// looksText reads the start of a file for the signs of text: no NUL
// bytes, and few control characters.
func looksText(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	prefix := make([]byte, 8192)
	n, _ := f.Read(prefix)
	prefix = prefix[:n]
	if len(prefix) == 0 {
		return true
	}
	if bytes.IndexByte(prefix, 0) >= 0 {
		return false
	}
	controls := 0
	for _, b := range prefix {
		if b < 0x09 || b > 0x0d && b < 0x20 {
			controls++
		}
	}
	return controls*100 < len(prefix)
}

func (s *store) add(n *note) {
	if n.Path == "" {
		s.seq++
		n.Path = fmt.Sprintf("Untitled %d.md", s.seq)
	}
	n.Rel = relPath(s.dir, n.Path)
	if n.words == 0 && n.Saved != "" && isMarkdownPath(n.Path) {
		n.words = wordCount(n.Saved)
	}
	s.notes = append(s.notes, n)
	s.byPath[n.Path] = n
	s.byRel[n.Rel] = n
}

// relPath is p within dir, for a layout the app can move with its vault.
func relPath(dir, p string) string {
	if dir == "" {
		return filepath.Base(p)
	}
	if rel, err := filepath.Rel(dir, p); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return filepath.Base(p)
}

// newNote writes a new untitled note in the root of the vault.
func (s *store) newNote() (*note, error) { return s.newNoteIn("") }

// newNoteIn writes a new untitled note in the folder rel, or at the root
// for an empty rel, and returns it.
func (s *store) newNoteIn(rel string) (*note, error) {
	title := "Untitled"
	body := "# Untitled\n\n"
	dir := s.dir
	if s.memory {
		rel = ""
	}
	if !s.memory && rel != "" && rel != "." {
		var err error
		dir, err = s.dirPath(rel)
		if err != nil {
			return nil, err
		}
	}
	s.seq++
	var name string
	if s.memory {
		name = fmt.Sprintf("Untitled %d.md", s.seq)
	} else {
		name = "Untitled.md"
		for i := 2; ; i++ {
			if _, err := os.Stat(filepath.Join(dir, name)); errors.Is(err, os.ErrNotExist) {
				break
			}
			name = fmt.Sprintf("Untitled %d.md", i)
		}
	}
	path := name
	if !s.memory {
		path = filepath.Join(dir, name)
		if err := s.write(path, body); err != nil {
			return nil, err
		}
	}
	n := &note{Path: path, Title: title, Saved: body, Mod: time.Now(), words: wordCount(body)}
	s.add(n)
	s.buildTree()
	return n, nil
}

// save writes the note if it changed, atomically, so that a crash leaves
// the old text or the new, never half of each.
func (s *store) save(n *note) error {
	body := n.text()
	if body == n.Saved {
		return nil
	}
	if err := s.write(n.Path, body); err != nil {
		return err
	}
	n.Saved = body
	n.Title = noteTitle(body, n.Path)
	n.words = 0
	if isMarkdownPath(n.Path) {
		n.words = wordCount(body)
	}
	n.Mod = time.Now()
	if n.buf != nil {
		n.buf.MarkSaved()
	}
	return nil
}

// saveAll writes every note with unsaved edits.
func (s *store) saveAll() error {
	var first error
	for _, n := range s.notes {
		if !n.dirty() {
			continue
		}
		if err := s.save(n); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *store) write(path, body string) error {
	if s.memory {
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Windows will not rename over an existing file.
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			os.Remove(tmp)
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			os.Remove(tmp)
			return err
		}
	}
	return nil
}

// sorted returns the notes, most recently touched first.
func (s *store) sorted() []*note {
	out := append([]*note(nil), s.notes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mod.Equal(out[j].Mod) {
			return out[i].Title < out[j].Title
		}
		return out[i].Mod.After(out[j].Mod)
	})
	return out
}

// byRel returns the note at a path within the vault, if it is in it.
func (s *store) byRelPath(rel string) *note {
	if n, ok := s.byRel[rel]; ok {
		return n
	}
	if n, ok := s.byPath[rel]; ok {
		return n
	}
	if !s.memory && !filepath.IsAbs(rel) {
		return s.byPath[filepath.Join(s.dir, rel)]
	}
	return nil
}

// displayDir is the vault's path as the title bar shows it.
func (s *store) displayDir() string {
	if s.memory {
		return "in memory"
	}
	home, err := os.UserHomeDir()
	if err == nil {
		if rest, ok := strings.CutPrefix(s.dir, home); ok {
			return "~" + rest
		}
	}
	return s.dir
}

const welcomeNote = `# Welcome to Mojian

Mojian is a quiet text editor for plain files, built with [MyGo](https://mygo.egoist.dev)'s native UI.

## Start here

- **⌘O** opens a project folder, and the name at the top left switches between them.
- **⇧⌘B** shows the file tree.
- **⌘N** writes a new note; **⇧⌘N** makes a folder.
- **⇧⌘D** splits the editor, so you can work in two places at once.
- **⇧⌘O** searches the files of the project.
- **⇧⌘T** opens a terminal in the folder you are in.
- **⇧⌘P** opens the command palette.

Mojian edits text and code, with syntax highlighting for Python, JavaScript,
TypeScript, Go, Rust, Swift, Markdown and more. Files stay ordinary files in
` + "`~/Documents/Mojian`" + `, or wherever the project lives.
`

const sampleNote = `# Reading notes

## Today

- [x] Sketch the shell
- [ ] Wire the editor
- [ ] Drag a divider

The pane keeps the note's caret and scroll while the tab is out of
sight.

## A quiet place

> Ink on paper, not noise.
`
