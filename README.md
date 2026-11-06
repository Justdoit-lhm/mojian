# Mojian

A quiet text editor for plain files, built with [MyGo](https://mygo.egoist.dev)'s
native UI: no webview, no cgo, one small Go binary. Tabs hold workspaces,
workspaces hold panes, and every pane edits a file with syntax highlighting.

![Mojian](docs/screenshot.png)

## What it does

- **Projects, not a fixed vault.** The chip at the top left names the open
  project and switches to a recent one, or opens any folder with `⌘O`. The
  folder chooser is part of the app, with the same colors and controls as the
  rest of the window. Each project keeps its own tabs and layout.
- **An Explorer, not a file list.** `⇧⌘B` shows the project's folders and text
  files. New File and New Folder live in its header; right-click an entry for
  Open in Current Window, Open in New Tab, Rename, Duplicate, Copy, Cut,
  Paste, Copy Path, Reveal in Finder, and Move to Trash. The tree reloads when the window
  regains focus, so a file made in the terminal appears.
- **Code, not only Markdown.** Python, JavaScript, TypeScript, Go, Rust, C, C++,
  Swift, Kotlin, JSON, YAML, TOML, SQL, HTML, CSS, Markdown and the rest of
  MyGo's languages are highlighted. Code files get line numbers; prose wraps
  instead of scrolling sideways. File rows use Material Icon Theme's
  per-language icons.
- **Tabs and split panes.** A tab is a workspace; each pane is an editor.
  The `+` beside the tabs opens an empty workspace, not a copy of the current
  one. Split right or down and choose the file for the new pane; a split never
  opens the same file twice. A tab holds at most four panes. Drag the divider,
  double-click it for half and half, zoom a pane, or close it.
- **A terminal inside the window.** `⇧⌘T` opens a terminal pane on the right,
  in the selected folder. It shares the tab's four-pane limit and the same
  split, zoom and close controls as a file pane; `⇧⌘T` again closes it. It is a
  real terminal (the MyGo terminal plugin, libghostty-vt) with colors,
  selection and clipboard.
- **A Git diff page.** `⇧⌘G` shows the working tree against `HEAD`: changed
  files on the left, and a line-numbered diff with tinted additions and
  deletions on the right. The header shows the branch, file count, status,
  change stats and an Open action; double-click a changed file to open it.
- **Plain files, saved when you ask.** `⌘S` writes atomically; edits stay in
  memory until you save. The layout comes back after a restart.
- **A command palette.** `⇧⌘P` runs every command and jumps to any file or tab.
- **Light, dark, or the system.** Both palettes are hand-tuned; the window
  follows the system by default, and View ▸ Appearance overrides it.

## Shortcuts

| macOS | Action |
|---|---|
| `⌘O` | Open project folder |
| `⇧⌘O` | Find a file |
| `⌘N` | New file |
| `⇧⌘N` | New folder |
| `⌘T` | New tab |
| `⇧⌘B` | Show or hide the Explorer |
| `⇧⌘T` | Open or close the terminal pane |
| `⇧⌘G` | Show the Git diff page |
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
touched; projects, Git and terminals need the desktop app.

## Layout

| | |
|---|---|
| `main.go` | the app, its window, menus, commands and lifecycle |
| `store.go` | a project folder: text files, new notes and atomic saves |
| `tree.go` | the file tree, folder creation and file search |
| `project.go` | the project switcher and recent projects |
| `term.go` | the terminal pane, via MyGo's terminal plugin |
| `git.go` | the working-tree diff page |
| `files.go` | file operations: rename, copy, move, trash and the context menu |
| `picker.go` | the in-window project folder chooser |
| `split.go` | the split file chooser and the four-pane limit |
| `model.go` | files, tabs, panes, the split tree and the saved layout |
| `view.go` | the title bar, pane cards and splitters |
| `tabs.go` | the tab strip and tab menu |
| `palette.go` | the command palette |
| `style.go` | palettes, the window gradient, fonts and icons |
| `assets/` | Lucide action icons (ISC), Material Icon Theme file icons (MIT), and JetBrains Mono (OFL) |

## Credits

Action icons: [Lucide](https://lucide.dev), ISC (`assets/icons/LICENSE`).
File and folder icons: [Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme), MIT (`assets/icons/material/LICENSE`).
Font: [JetBrains Mono](https://www.jetbrains.com/lp/mono/), OFL
(`assets/fonts/OFL.txt`).
The window's shape and interaction follow [GoRex](https://github.com/egoist/gorex),
also built with MyGo.
