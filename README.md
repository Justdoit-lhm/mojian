<p align="center">
  <img src="logos/png/mojian-logo-horizontal-1600.png" alt="Mojian" width="400">
</p>

<p align="center"><em>Words on paper. The tool steps back.</em></p>

<p align="center">
  macOS 13+ &nbsp;·&nbsp; Linux amd64/arm64 &nbsp;·&nbsp; MyGo native UI &nbsp;·&nbsp; No webview &nbsp;·&nbsp; Built without cgo
</p>

<p align="center">
  <img src="docs/screenshot.png" alt="A Markdown note open in Mojian" width="960">
</p>
<p align="center"><sub>A folder on the left, a note in front, and nothing else asking for attention.</sub></p>

## Mojian

Mojian is a quiet plain-text editor for macOS and Linux. It opens ordinary folders and edits ordinary files. There is no library, no index, no database, no account.

The window is drawn by [MyGo](https://mygo.egoist.dev) as native UI — Metal on macOS, GTK on Linux — so there is no webview, no Electron, and no JavaScript build. A tab holds tiled panes; a pane can edit a file, preview a picture or PDF, or run a terminal. Terminals live in a separate session server and survive quitting the app, and a project can live on a server reached over SSH.

The Chinese name 墨间 (mò jiān) is ink, and the space it needs. The tool steps behind the paper, the file stays in front of you, and the pace of writing stays yours.

## Install

Download the latest build from [Releases](https://github.com/ox0pine/mojian/releases).

On macOS, use `mojian-<version>-arm64.dmg` for Apple Silicon or `mojian-<version>-amd64.dmg` for Intel, open the disk image, and drag Mojian into Applications. Mojian requires macOS 13 or later. Release builds are ad-hoc signed rather than notarized; if macOS blocks the first launch, right-click Mojian in Finder and choose **Open**.

On Linux, install the Debian package with `sudo apt install ./mojian_<version>_<arch>.deb` (amd64 or arm64), or unpack `mojian-<version>-linux-<arch>.tar.gz`, run `sh install.sh` inside it, and launch `mojian` from the applications menu. The archive installs per user without root. The Linux build needs GTK 3.

## Features

### A project is a folder

`⌘O` opens any folder, and the chips at the top left switch between recent projects and servers. Each project remembers its own tabs and pane layout. With no project chosen, the first launch opens `~/Documents/Mojian` and leaves a welcome note when the folder is empty.

### Writing comes first

`⌘S` saves atomically. Edits stay in memory until then, and anything unsaved is written when the window closes.

Syntax highlighting covers the languages MyGo knows — Python, JavaScript, TypeScript, Go, Rust, C and C++, Swift, Kotlin, Ruby, JSON, YAML, TOML, SQL, HTML, CSS, Markdown and more. Code keeps its line numbers; prose wraps. The editor follows Zed's keymap for find and replace, multiple carets, line moves, comments and folding.

A Markdown pane switches between source and preview from its header, and Settings › Editor can open new `.md` files rendered. Images, SVG included, open in a contained viewer; PDFs render page by page — up to 60 — when `pdftoppm` is installed, and fall back to a Quick Look first page on macOS.

### Tabs, panes, and the Explorer

A tab is a workspace made of panes. Splits open an empty pane with an inline file search; drag a pane by its grip onto another pane's edge, or move it with `⌥⇧` and the arrows. `⌥` and the arrows move focus, and `⇧⌘M` or a double-click zooms the pane.

`⇧⌘B` opens the Explorer. Its search field filters files and `⇧⌘O` takes you there. The header creates files and folders; right-click offers Open in Current Window, Open in New Tab, Rename (`F2`), Duplicate, Copy, Cut, Copy Path, Copy Relative Path, Reveal in Finder and Move to Trash (`⌘⌫`); folders and empty space take Paste. The tree re-reads the disk when the window regains focus, and generated trees such as `.git`, `node_modules`, `dist` and `build` stay out of sight.

### Terminals that outlive the window

`⇧⌘T` opens a real terminal at the project root. It tiles, splits, zooms and closes like any pane. Shells run in a detached session server: quitting Mojian detaches them, and reopening reattaches. `⇧⌘L` opens an SSH terminal from `~/.ssh/config`.

### Projects on a server

Add a host in Settings › Hosts, then choose it from the server chip; the address is anything `ssh` accepts (`buildbox`, `me@devbox`, `ssh://me@devbox:2222`). On connection, Mojian prepares a small per-user session server over `ssh`, forwards its Unix socket and mirrors the project into a local cache; saves are written back. No root, no listening port, no service installed. The Explorer, Git page and terminals keep working on the remote copy. The workflow follows [Kero](https://kero.sh).

### The working tree at a glance

`⇧⌘G` shows the working tree against `HEAD`: changed files and their status on the left, a two-gutter diff with restrained additions and deletions on the right. Staged, unstaged and untracked work are all there. Double-click a file to open it, or refresh to re-read.

### Settings, palette, and light or dark

`⌘,` opens Settings in its own window: searchable categories on the left — General, Appearance, Editor, Fonts, Terminal, Shell, Notifications, Keyboard, Hosts, Git, Advanced — and the settings on the right. Every change applies and is saved at once.

`⇧⌘P` searches commands, files and tabs with fuzzy matching. Both palettes are tuned by hand; the window follows the system by default, and View › Appearance overrides it.

## Keyboard shortcuts

The tables use macOS keys. On Linux, Ctrl takes the place of `⌘`, and Alt the place of `⌥`.

### Files and projects

| macOS | Action |
|---|---|
| `⌘O` | Open project folder |
| `⇧⌘O` | Find a file |
| `⌘N` | New file |
| `⇧⌘N` | New folder |
| `F2` | Rename the selected tree entry |
| `⌘⌫` | Move the selected tree entry to Trash |
| `⌘S` | Save now |

### Tabs and panes

| macOS | Action |
|---|---|
| `⌘T` | New tab |
| `⌘W` / `⇧⌘W` | Close pane / tab |
| `⇧⌘R` | Rename tab |
| `⌃⇥` / `⌃⇧⇥` | Next / previous tab |
| `⌘1…⌘9` | Show tab |
| `⇧⌘D` / `⇧⌘E` | Split right / down |
| `⌥⌘←` / `⌥⌘↑` | Split left / up |
| `⌥←` / `⌥→` / `⌥↑` / `⌥↓` | Move pane selection |
| `⌥⇧←` / `⌥⇧→` / `⌥⇧↑` / `⌥⇧↓` | Move the selected pane |
| `⇧⌘M` | Zoom the pane |

### Panels and commands

| macOS | Action |
|---|---|
| `⇧⌘B` | Show or hide the Explorer |
| `⇧⌘T` | Open or close the terminal pane |
| `⇧⌘L` | Open an SSH terminal |
| `⇧⌘G` | Show the Git diff page |
| `⇧⌘P` | Command palette |
| `⌘,` | Settings |

Double-click a tab to rename it; double-click a pane's header to zoom it. Right-click a tab for Reset Name, Close Other Tabs and the rest of its menu.

## Build from source

Mojian needs Go 1.27.3 or later.

```sh
git clone https://github.com/ox0pine/mojian.git
cd mojian

go tool mygo dev        # live-reloading development app
go test ./...           # unit and integration tests
go tool mygo vet .      # go vet plus MyGo's UI lifetime checks
go tool mygo build      # build/<os>-<arch>/: .app and .dmg, or .tar.gz and .deb
```

MyGo is pinned as a Go tool in `go.mod`, so the first `go tool mygo` downloads the exact version. `go tool mygo dev -web` opens the interface in a browser, with files kept in memory. After changing the session protocol or `cmd/mojian-server`, rebuild the embedded remote binaries with `sh tools/build-remote.sh`.

## Design

Ink `#1F1B16`, paper `#FBF7EF`, vermilion `#D24A2B`. In Mojian's identity almost everything is ink on paper; only the cursor keeps the vermilion — the one voice the page raises. Marks and exports live in [`logos/README.md`](logos/README.md).

## Credits

Action icons by [Lucide](https://lucide.dev) (ISC); file icons by [Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme) (MIT); type set in [JetBrains Mono](https://www.jetbrains.com/lp/mono/) (OFL); terminals driven by libghostty-vt and PTYs by [creack/pty](https://github.com/creack/pty) (MIT). The logo is ink, paper and one vermilion cursor, its wordmark outlining Literata (SIL OFL). Window and interaction follow [GoRex](https://github.com/egoist/gorex); remote hosts and Settings follow [Kero](https://kero.sh). Drawn with [MyGo](https://mygo.egoist.dev).
