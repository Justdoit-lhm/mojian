# Mojian

<p align="center"><img src="logos/png/mojian-logo-horizontal-1600.png" alt="Mojian" width="420"></p>

A quiet text editor for plain files. Mojian is built with
[MyGo](https://mygo.egoist.dev)'s native UI: no webview, no cgo, no JavaScript
build — one Go binary that draws its own window, through Metal on macOS.

![The Mojian window](docs/screenshot.png)

## What it does

- **A project is a folder.** Mojian opens any folder; nothing is imported or
  indexed. `⌘O` opens an in-window folder picker with places and recent
  projects, and the chips at the top left switch between projects and servers.
  Each project keeps its own tabs and pane layout. The app's own folder,
  `~/Documents/Mojian`, is created on first launch with a welcome note.
- **Plain files, written when you ask.** `⌘S` saves atomically — temp file,
  fsync, rename — and edits stay in memory until then; closing the window
  writes them. Syntax highlighting covers the languages MyGo's editor knows:
  Python, JavaScript, TypeScript, Go, Rust, C, C++, Swift, Kotlin, JSON, YAML,
  TOML, SQL, HTML, CSS, Markdown and more. Code files get line numbers; prose
  wraps instead of scrolling sideways.
- **An Explorer, not a file list.** `⇧⌘B` shows the project tree. Type in its
  search field to filter files, or press `⇧⌘O` to jump there. The header
  creates files and folders, and the tree reloads when the window regains
  focus. Right-click a file for Open in Current Window and Open in New Tab;
  every entry also has Rename (`F2`), Duplicate, Copy, Cut, Copy Path, Copy
  Relative Path, Reveal in Finder, and Move to Trash (`⌘⌫`); folders and the
  tree's empty space take Paste too. Git's junk folders — `.git`,
  `node_modules`, `dist`, `build` — stay out of the tree.
- **Tabs and split panes.** A tab is a workspace, and each tab holds a binary
  split tree of panes. A split opens an empty pane with an inline file search
  instead of a dialog, and a tab can hold as many panes as you keep splitting.
  Drag a pane by its grip onto another pane's edge to insert it there, or move
  it with `⌥⇧` and the arrow keys. `⌥` and the arrow keys move the focus,
  `⇧⌘M` zooms a pane, and double-clicking its header does the same.
  Middle-click a tab to close it.
- **Markdown, source or rendered.** A Markdown pane has a Preview button at
  the top right; the Editor settings open new `.md` files rendered by default.
  The preview reads the live buffer and keeps its own scroll.
- **Pictures and PDFs.** Images — SVG included — open in a contained viewer
  with their size and an Open action. PDFs are rendered page by page (up to
  60) when `pdftoppm` is installed, and fall back to a Quick Look first-page
  preview otherwise.
- **A terminal in the window.** `⇧⌘T` opens a real terminal pane (MyGo's
  libghostty-vt plugin) in the project root; it tiles, splits, zooms and
  closes like any other pane. The shell process lives in a small session
  server, so quitting Mojian detaches from it and reopening restores it.
  `⇧⌘L` opens an SSH terminal using a host from `~/.ssh/config`.
- **Projects on a server.** The server chip switches between Local and the
  hosts added in Settings › Hosts. When connecting, Mojian uploads a per-user
  session server to `~/.local/share/mojian` over `ssh`; it needs no root,
  opens no port, and is not installed as a service. The remote project is
  mirrored into a local cache, saves upload back, and the Explorer, Git and
  terminals keep working. SSH hosts follow `~/.ssh/config`; the host model
  follows [Kero](https://kero.sh).
- **A Git diff page.** `⇧⌘G` shows the working tree against `HEAD`: changed
  files on the left with their status, and a line-numbered diff with tinted
  additions and deletions on the right. It includes staged and unstaged work;
  untracked files read as additions. Double-click a file to open it, or use
  the refresh button.
- **Settings in their own window.** `⌘,` opens a separate window, after Kero:
  searchable categories on the left, settings on the right. General,
  Appearance, Editor, Fonts, Terminal, Shell, Notifications, Keyboard, Hosts,
  Git and Advanced cover type, wrapping, Markdown, fonts, terminal input and
  scrollback, notifications, hosts and data paths. Every change applies and is
  saved at once.
- **A command palette.** `⇧⌘P` lists commands, files and tabs with fuzzy
  matching, so the keyboard can reach most of the app.
- **Light, dark, or the system.** Both palettes are hand-tuned; the window
  follows the system by default, and View › Appearance overrides it.

<p align="center">
  <img src="docs/ssh.png" alt="Connect to Server" width="440">
  <img src="docs/settings.png" alt="The Settings window" width="360">
</p>

## Shortcuts

| macOS | Action |
|---|---|
| `⌘O` | Open project folder |
| `⇧⌘O` | Find a file |
| `⌘N` | New file |
| `⇧⌘N` | New folder |
| `F2` | Rename the selected tree entry |
| `⌘⌫` | Move the selected tree entry to Trash |
| `⌘T` | New tab |
| `⌘W` / `⇧⌘W` | Close pane / tab |
| `⇧⌘R` | Rename tab |
| `⌃⇥` / `⌃⇧⇥` | Next / previous tab |
| `⌘1…⌘9` | Show tab |
| `⇧⌘B` | Show or hide the Explorer |
| `⇧⌘T` | Open or close the terminal pane |
| `⇧⌘L` | Open an SSH terminal |
| `⇧⌘G` | Show the Git diff page |
| `⇧⌘D` / `⇧⌘E` | Split right / down |
| `⌥⌘←` / `⌥⌘↑` | Split left / up |
| `⌥←` / `⌥→` / `⌥↑` / `⌥↓` | Move pane selection |
| `⌥⇧←` / `⌥⇧→` / `⌥⇧↑` / `⌥⇧↓` | Move the selected pane |
| `⇧⌘M` | Zoom the pane |
| `⇧⌘P` | Command palette |
| `⌘,` | Settings |
| `⌘S` | Save now |

Double-click a tab to rename it; double-click a pane's header to zoom it.
Right-click a tab for Reset Name, Close Other Tabs and the rest of its menu.

## Develop

Mojian needs Go 1.27.3 or later, and macOS 13 or later for the app. MyGo is
pinned as a Go tool in `go.mod`, so there is no global install, no Node and no
cgo:

```sh
go tool mygo dev     # "Mojian Dev", rebuilt as the code changes
go test ./...        # the model, the store, the view and integration tests
go tool mygo vet .   # go vet plus MyGo's UI lifetime checks
go tool mygo build   # build/<os>-<arch>/Mojian.app and a .dmg
```

The native UI also runs in a browser:

```sh
go tool mygo dev -web
```

There the files live in memory for the session; projects on disk, terminals,
SSH, Git and the PDF preview need the desktop app.

`mygo dev` runs the app as "Mojian Dev", with the same default project but its
own window state and preferences, apart from the installed app; its bundle is
built in `.mygo/dev`. `MOJIAN_VAULT` opens another project at launch, and
`MOJIAN_DIR` moves the app's settings, layouts, remote caches and session
socket. By default they live in MyGo's user data directory:

| Path | Contents |
|---|---|
| `settings.json` | appearance, editor and terminal preferences, recent projects, hosts |
| `layouts/<hash of the project path>.json` | each project's tabs, panes and open files |
| `remote/<hash>/` | the local mirror of a remote project |
| `sessions.sock` | the local terminal session server |

The uploaded server and its log live on the host at `~/.local/share/mojian/`.
The remote binaries are embedded in the app from `internal/remotebin`; after
changing `cmd/mojian-server` or the wire protocol, rebuild them:

```sh
sh tools/build-remote.sh
```

There is no CI: verification is local, with the commands above and the running
app.

## Layout

Go compiles one package per directory, so the root is only the entry point and
the window is split by concern under `internal/`.

| Path | Purpose |
|---|---|
| `main.go` | entry point, or `-server` for the session server |
| `cmd/mojian-server/` | the per-user server uploaded to SSH hosts |
| `internal/app/` | the window: shell, tabs, pane split tree, Explorer, pickers, palette, Settings, file operations, SSH and asset viewers |
| `internal/session/` | PTY server and client, the wire protocol, remote file operations |
| `internal/git/` | the working-tree page: status, diffs, parsing and view |
| `internal/theme/` | light and dark palettes, radius, background and editor theme |
| `internal/icons/` | Lucide action icons and Material Icon Theme file icons |
| `internal/assets/` | embedded icons, brand marks and JetBrains Mono |
| `internal/motion/` | the page, tab and tree transitions |
| `internal/remotebin/` | embedded Linux server and libghostty-vt builds (generated) |
| `test/` | black-box integration tests with real folders and Git |
| `tools/build-remote.sh` | cross-compiles the remote server into `internal/remotebin` |
| `docs/`, `logos/`, `resources/` | screenshots; logo masters and exports; the app icon |

Inside `internal/app/`:

| | |
|---|---|
| `app.go` | the app, its window, menus, commands, settings and lifecycle |
| `view.go` | the title bar, the split tree, pane cards and the drop indicator |
| `model.go` | files, tabs, panes, the split tree and the saved layout |
| `store.go` | a project folder: text files, assets, new notes and atomic saves |
| `tree.go` | the Explorer, folder creation and file search |
| `files.go` | rename, copy, move, trash and the context menu |
| `term.go` | terminal panes, attached to the session server |
| `remote.go` | SSH connections, remote project mirrors and the server picker |
| `ssh.go` | the SSH host picker and `~/.ssh/config` |
| `palette.go` | the command palette and the command table |
| `settings.go` | the Kero-inspired Settings window |
| `assetview.go` | image, PDF and Markdown preview rendering |

## Credits

Action icons: [Lucide](https://lucide.dev), ISC
(`internal/assets/icons/LICENSE`). File and folder icons: [Material Icon
Theme](https://github.com/PKief/vscode-material-icon-theme), MIT
(`internal/assets/icons/material/LICENSE`). Font: [JetBrains
Mono](https://www.jetbrains.com/lp/mono/), OFL
(`internal/assets/fonts/OFL.txt`). Pseudo-terminals:
[creack/pty](https://github.com/creack/pty), MIT.

The logo is by the Mojian project: ink, paper and one vermilion cursor; the
wordmark outlines Literata, SIL OFL. See [`logos/README.md`](logos/README.md)
for the files and their rules. The window's shape and interaction follow
[GoRex](https://github.com/egoist/gorex); remote hosts and the Settings window
follow [Kero](https://kero.sh). Everything is drawn with
[MyGo](https://mygo.egoist.dev).
