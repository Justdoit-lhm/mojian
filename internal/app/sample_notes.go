package app

import "strings"

const welcomeNoteText = `# Welcome to Mojian

Mojian is a quiet text editor for plain files, built with [MyGo](https://mygo.egoist.dev)'s native UI.

## Start here

- **⌘O** opens a project folder, and the name at the top left switches between them.
- **⇧⌘B** shows the file tree.
- **⌘N** names a new file; **⇧⌘N** makes a folder.
- **⇧⌘D** splits the editor, so you can work in two places at once.
- **⇧⌘O** opens any file of the project.
- **⇧⌘T** opens a terminal on the right, in the folder you are in.
- **⇧⌘P** opens the command palette.

Mojian edits text and code, with syntax highlighting for Python, JavaScript,
TypeScript, Go, Rust, Swift, Markdown and more. The Explorer copies, cuts,
pastes and renames files like a code editor; files stay ordinary files in
` + "`~/Documents/Mojian`" + `, or wherever the project lives.
`

var welcomeNote = platformWelcomeNote()

func platformWelcomeNote() string {
	note := welcomeNoteText
	if !terminalsSupported() {
		note = strings.Replace(note, "- **⇧⌘T** opens a terminal on the right, in the folder you are in.\n", "", 1)
	}
	return nativeKeys(note)
}

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
