# Mojian

A quiet text editor for plain files, built with [MyGo](https://mygo.egoist.dev)'s
native UI: no webview, no cgo, one small Go binary. Tabs hold workspaces,
workspaces hold panes, and every pane edits a file with syntax highlighting.

![Mojian](docs/screenshot.png)

## What it does

- **Projects, not a fixed vault.** The chip at the top left names the open
  project and switches to a recent one, or opens any folder with `⌘O`. Each
  project keeps its own tabs and layout, so switching does not mix them.
- **A file tree.** `⇧⌘B` shows the project's folders and text files. Create a
  folder with the button at the top right (`⇧⌘N`); search files from the
  sidebar or with `⇧⌘O`; open them with a click. The tree reloads when the
  window regains focus, so a file made in the terminal appears.
- **Code, not only Markdown.** Python, JavaScript, TypeScript, Go, Rust, C, C++,
  Swift, Kotlin, JSON, YAML, TOML, SQL, HTML, CSS, Markdown and the rest of
  MyGo's languages are highlighted. Code files get line numbers; a Markdown
  note gets a word count instead.
- **Tabs and split panes.** A tab is a workspace; each pane is an editor.
  Split right or down, drag the divider, double-click it for half and half,
  zoom a pane, or close it. Two panes of one file share its buffer, each with
  its own caret and scroll.
- **A terminal one shortcut away.** `⇧⌘T` opens the system terminal in the
  selected folder, or the folder of the file in view, from the File menu or the
  button at the bottom of the file tree. `MOJIAN_TERMINAL` names another
  terminal on Linux.
- **Plain files, saved automatically.** Saving is atomic and automatic; the
  layout comes back after a restart.
- **A command palette.** `⇧⌘P` runs every command and jumps to any file or tab.
- **Light, dark, or the system.** Both palettes are hand-tuned; the window
  follows the system by default, and View ▸ Appearance overrides it.

## Shortcuts

| macOS | Action |
|---|---|
| `⌘O` | Open project folder |
| `⇧⌘O` | Find a file |
| `⌘N` | New note |
| `⇧⌘N` | New folder |
| `⌘T` | New tab |
| `⇧⌘B` | Show or hide the file tree |
| `⇧⌘T` | Open a terminal here |
| `⇧⌘D` / `⇧⌘E` | Split right / down |
| `⌘W` / `⇧⌘W` | Close pane / tab |
| `⇧⌘M` | Zoom the pane |
| `⌃⌘=` | Equalize panes |
| `⌘1…⌘9` | Show tab |
| `⌃⇥` / `⌃⇧⇥` | Next / previous tab |
| `⇧⌘R` | Rename tab |
| `⇧⌘P` | Command palette |
| `⌘S` | Save now |

Double-click a tab to rename it; double-click a pane's header to zoom it.

## Develop

Mojian needs Go 1.27 or later. MyGo is pinned in `go.mod` as a tool, so no
global install and no Bun are needed:

```sh
go tool mygo dev     # Mojian Dev, rebuilt as the code changes
go test ./...        # the model, the store and the view, without a window
go tool mygo build   # build/darwin-arm64/Mojian.app and a .dmg
```

`mygo dev` uses the same default project (`~/Documents/Mojian`), but its window
state and preferences live in `Mojian Dev`, apart from the installed app.
`MOJIAN_VAULT` names another project at launch. The last project and the recent
projects are kept in the app's settings, and each project's layout in
`layouts/` there.

The native UI can also be developed in a browser:

```sh
go tool mygo dev -web
```

There, the files live in memory for the session and the file system is not
touched; projects and terminals need the desktop app.

## Layout

| | |
|---|---|
| `main.go` | the app, its window, menus, commands and lifecycle |
| `store.go` | a project folder: text files, new notes, atomic saves, autosave |
| `tree.go` | the file tree, folder creation and file search |
| `project.go` | the project switcher and recent projects |
| `terminal.go` | opening the system terminal in a folder |
| `model.go` | files, tabs, panes, the split tree and the saved layout |
| `view.go` | the title bar, pane cards and splitters |
| `tabs.go` | the tab strip and tab menu |
| `palette.go` | the command palette |
| `style.go` | palettes, the window gradient, fonts and icons |
| `assets/` | Lucide icons (ISC) and JetBrains Mono (OFL) |

## Credits

Icons: [Lucide](https://lucide.dev), ISC (`assets/icons/LICENSE`).
Font: [JetBrains Mono](https://www.jetbrains.com/lp/mono/), OFL
(`assets/fonts/OFL.txt`).
The window's shape and interaction follow [GoRex](https://github.com/egoist/gorex),
also built with MyGo.
