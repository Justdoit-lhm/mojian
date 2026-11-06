package main

import (
	"errors"
	"fmt"
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
	Path     string
	Title    string
	Saved    string // the text as last read or written
	Mod      time.Time
	lastEdit time.Time

	state *editor.State
}

// text is the note as it stands, including edits not yet saved.
func (n *note) text() string {
	if n.state != nil {
		return n.state.Buffer.String()
	}
	return n.Saved
}

// dirty reports whether the note has edits that are not on disk.
func (n *note) dirty() bool { return n.text() != n.Saved }

// editorState returns the note's editor, making it on first use, so that
// its buffer, carets, folds and undo history outlive a frame.
func (n *note) editorState() *editor.State {
	if n.state == nil {
		buf := editor.NewBuffer(n.Saved)
		buf.SetLanguage(editor.LanguageFor(n.Path))
		n.state = &editor.State{
			Buffer:       buf,
			Font:         noteFont,
			LineNumbers:  editor.LineNumbersOff,
			SoftWrap:     true,
			NoActiveLine: true,
			Placeholder:  "Start writing…",
			Label:        n.Title,
		}
	}
	return n.state
}

// wordCount counts what a reader would call words, near enough: runs of
// text between spaces. Markdown punctuation counts with them.
func wordCount(s string) int { return len(strings.Fields(s)) }

// noteTitle is the first heading of the note, else its file name without
// the extension.
func noteTitle(body, path string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if h := strings.TrimSpace(strings.TrimLeft(line, "#")); h != "" {
				return h
			}
		}
		break
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.TrimSpace(stem) == "" {
		return "Untitled"
	}
	return stem
}

// store is the vault: the Markdown files of the notes, by path.
type store struct {
	dir    string
	memory bool
	notes  []*note
	byPath map[string]*note
	byRel  map[string]*note
	seq    int
}

// openStore opens the vault at $MOJIAN_VAULT, else ~/Documents/Mojian,
// and creates a welcome note in a vault of its own. In a browser there is
// no file system: the notes live in memory for the session.
func openStore() (*store, error) {
	if runtime.GOOS == "js" {
		s := &store{memory: true, byPath: map[string]*note{}, byRel: map[string]*note{}}
		s.add(&note{Path: "Welcome.md", Title: "Welcome to Mojian", Saved: welcomeNote, Mod: time.Now()})
		s.add(&note{Path: "Reading notes.md", Title: "Reading notes", Saved: sampleNote, Mod: time.Now().Add(-time.Minute)})
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
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &store{dir: dir, byPath: map[string]*note{}, byRel: map[string]*note{}}
	if err := s.refresh(); err != nil {
		return nil, err
	}
	if len(s.notes) == 0 {
		path := filepath.Join(dir, "Welcome.md")
		if err := s.write(path, welcomeNote); err != nil {
			return nil, err
		}
		if err := s.refresh(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// refresh adds the Markdown files that appeared in the vault since the
// last look. Notes already open keep their text and their place.
func (s *store) refresh() error {
	if s.memory {
		return nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !isMarkdown(e.Name()) {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		if _, ok := s.byPath[path]; ok {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		n := &note{
			Path:  path,
			Saved: string(data),
			Mod:   info.ModTime(),
		}
		n.Title = noteTitle(n.Saved, path)
		s.add(n)
	}
	sort.SliceStable(s.notes, func(i, j int) bool { return s.notes[i].Mod.After(s.notes[j].Mod) })
	return nil
}

func isMarkdown(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

func (s *store) add(n *note) {
	if n.Path == "" {
		s.seq++
		n.Path = fmt.Sprintf("Untitled %d.md", s.seq)
	}
	n.Rel = relPath(s.dir, n.Path)
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

// newNote writes a new untitled note and returns it.
func (s *store) newNote() (*note, error) {
	title := "Untitled"
	body := "# Untitled\n\n"
	s.seq++
	var name string
	if s.memory {
		name = fmt.Sprintf("Untitled %d.md", s.seq)
	} else {
		name = "Untitled.md"
		for i := 2; ; i++ {
			if _, err := os.Stat(filepath.Join(s.dir, name)); errors.Is(err, os.ErrNotExist) {
				break
			}
			name = fmt.Sprintf("Untitled %d.md", i)
		}
	}
	path := name
	if !s.memory {
		path = filepath.Join(s.dir, name)
		if err := s.write(path, body); err != nil {
			return nil, err
		}
	}
	n := &note{Path: path, Title: title, Saved: body, Mod: time.Now()}
	s.add(n)
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
	n.Mod = time.Now()
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

Mojian is a quiet workspace for Markdown notes, built with [MyGo](https://mygo.egoist.dev)'s native UI.

## Start here

- **⌘N** writes a new note.
- **⌘T** opens a new tab.
- **⇧⌘D** splits the pane, so you can write and preview side by side.
- **⇧⌘V** switches a pane between the editor and the preview.
- **⇧⌘P** opens the command palette.

Notes are ordinary Markdown files in ` + "`~/Documents/Mojian`" + `, so another editor can open
them too.

## A note on ink

> Ink on paper, not noise. A quiet place to think.
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
