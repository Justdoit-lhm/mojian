// Package welcome holds the notes a new project and the browser start
// with, in the platform's own key spelling.
package welcome

import (
	"os"
	"path/filepath"
	"strings"

	"mojian/internal/platform"
)

const text = `# Welcome to Mojian

Mojian is a quiet text editor for plain files, built with [MyGo](https://mygo.egoist.dev)'s native UI.

## Start here

- **⌘O** opens a project folder, and the name at the top left switches between them.
- **⌘B** shows the file tree.
- **⌘N** names a new file; **⇧⌘N** makes a folder.
- **Pane › Split Right** splits the editor to the right, so you can work in two places at once.
- **⌘P** opens any file of the project.
- **⌃ + backtick** toggles a terminal on the right, in the folder you are in.
- **⇧⌘P** opens the command palette.

Mojian edits text and code, with syntax highlighting for Python, JavaScript,
TypeScript, Go, Rust, Swift, Markdown and more. The Explorer copies, cuts,
pastes and renames files like a code editor; files stay ordinary files in
` + "`~/Documents/Mojian`" + `, or wherever the project lives.
`

// Note is the welcome note for a new project.
var Note = platformNote()

func platformNote() string {
	note := text
	if !platform.TerminalsSupported() {
		note = strings.Replace(note, "- **⌃ + backtick** toggles a terminal on the right, in the folder you are in.\n", "", 1)
	}
	return platform.NativeKeys(note)
}

// Sample is the reading note the browser's sample project opens with.
const Sample = `# Reading notes

## Today

- [x] Sketch the shell
- [ ] Wire the editor
- [ ] Drag a divider

The pane keeps the note's caret and scroll while the tab is out of
sight.

## A quiet place

> Ink on paper, not noise.
`

// Fixture is one file of the sample project the harness and the app's
// tests open.
type Fixture struct {
	Name string
	Body string
}

// Files is the sample project: prose, code and the welcome note last, so
// a fresh project opens on it.
func Files() []Fixture {
	return []Fixture{
		{"Reading notes.md", Sample},
		{"Ideas/Quiet mornings.md", "# Quiet mornings\n\nCoffee, a blank page, and no notifications.\n"},
		{"scripts/tasks.py", "#!/usr/bin/env python3\n\nTASKS = [\"write\", \"read\"]\n\n\ndef open_tasks():\n    return TASKS\n"},
		{"scripts/app.js", "const tasks = [];\n\nfunction openTasks() {\n  return tasks;\n}\n"},
		{"Welcome.md", Note},
	}
}

// WriteFiles creates the sample project under dir.
func WriteFiles(dir string) error {
	for _, f := range Files() {
		path := filepath.Join(dir, f.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(f.Body), 0o644); err != nil {
			return err
		}
	}
	return nil
}
