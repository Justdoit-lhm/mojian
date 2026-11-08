<h1 align="center">Mojian · 墨间</h1>

<p align="center"><em>Words on paper. The tool steps back.</em></p>

<p align="center">
  macOS 13+ &nbsp;·&nbsp; Linux amd64/arm64 &nbsp;·&nbsp; MyGo native UI &nbsp;·&nbsp; No webview &nbsp;·&nbsp; No cgo
</p>

<p align="center">
  <a href="README.zh-CN.md">简体中文</a> &nbsp;·&nbsp; <a href="https://github.com/ox0pine/mojian/releases">Releases</a>
</p>

<p align="center">
  <img src="docs/screenshot.png" alt="Mojian with an editor, a pane choosing a file, and two terminals in one tab" width="1200">
</p>

## What Mojian is

Mojian is a plain-text editor that opens ordinary folders. There is no
library, index, database or account: a project is the folder you chose,
and a note is the file on disk. Edits stay in memory until you save.

A tab is a workspace of tiled panes. A pane can edit a file, preview an
image or PDF, or run a terminal. The window is native UI drawn by
[MyGo](https://mygo.egoist.dev): Metal on macOS, GTK 3 on Linux. There is
no webview, no Electron, and no JavaScript build.

## Install

Download the latest build from
[Releases](https://github.com/ox0pine/mojian/releases).

### macOS

- `mojian-<version>-arm64.dmg` for Apple Silicon
- `mojian-<version>-amd64.dmg` for Intel

Open the disk image and drag Mojian into Applications. Release builds are
ad-hoc signed rather than notarized; if macOS blocks the first launch,
right-click Mojian in Finder and choose **Open**.

### Linux

Install the Debian package:

```sh
sudo apt install ./mojian_<version>_<arch>.deb
```

Or install per user without root: put `install.sh` and
`mojian-<version>-linux-<arch>.tar.gz` in one folder, then run:

```sh
sh install.sh
```

The Linux app needs GTK 3 and a desktop session.

## The window

The top bar is the whole application: server and project at the left, the
open tab in the middle, and Explorer, split, terminal, new-tab and Settings
buttons at the right. Settings opens a separate window.

Each pane has its own header. It shows the file, picture or terminal, and
keeps split, zoom and close buttons visible whether the pane has focus or
not. Empty panes are useful: splitting opens one with an inline file search,
so the next file appears where you are looking.

### Projects and the Explorer

`Cmd+O` on macOS or `Ctrl+O` on Linux opens a folder. The project chip
switches between recent projects, and the server chip switches between
Local and configured hosts. Each project keeps its own tabs, panes and
recent files. With no project chosen, the first launch opens
`~/Documents/Mojian` and leaves a welcome note when the folder is empty.

`Shift+Cmd+B` / `Ctrl+Shift+B` shows or hides the Explorer. Its search field
filters the tree. **Open File** (`Shift+Cmd+O` / `Ctrl+Shift+O`) searches
every file in the project: type to filter, then press Enter or click a file.
The tree creates files and folders; right-click offers open, rename,
duplicate, copy, cut, copy path, reveal and move to trash. Generated folders
such as `.git`, `node_modules`, `dist` and `build` stay out of sight.

### Tabs and panes

`Cmd+T` / `Ctrl+T` opens a tab. A tab holds a binary split tree: split
right or down with `Shift+Cmd+D` / `Shift+Cmd+E`, or left and up with
`Option+Cmd` / `Ctrl+Alt` and the arrows. Drag a pane by its header onto
another pane's edge to put it there. `Option` / `Alt` and the arrows move
focus between panes; `Shift+Cmd+M` / `Ctrl+Shift+M` zooms the focused pane.

Double-click a tab to rename it, double-click a pane header to zoom it, and
right-click a tab for its menu. Closing a pane lets its neighbor take the
space at once; the closed card fades back over its old place.

### Terminal

`Shift+Cmd+T` / `Ctrl+Shift+T` opens a terminal at the project root. A
terminal is a pane like any other: split it, move it, zoom it or close it.
On macOS and Linux, terminals run in a detached session server, so quitting
Mojian leaves the shells alive and reopening attaches them again.

`Shift+Cmd+L` / `Ctrl+Shift+L` opens an SSH terminal to a host from
Settings › Hosts.

## Writing and previewing

Code keeps line numbers; prose wraps. Syntax highlighting covers Python,
JavaScript, TypeScript, Go, Rust, C and C++, Swift, Kotlin, Ruby, JSON,
YAML, TOML, SQL, HTML, CSS, Markdown and more.

Markdown panes switch between source and preview from the pane header, and
Settings › Editor can render new `.md` files by default. Images, including
SVG, open in a contained viewer. PDFs render page by page when `pdftoppm`
is installed, with a Quick Look first page on macOS.

`Cmd+S` / `Ctrl+S` writes every unsaved note atomically. Unsaved work is
also written when the window closes.

## Git, palette and Settings

`Shift+Cmd+G` / `Ctrl+Shift+G` shows the working tree against `HEAD`:
statuses on the left, a two-gutter diff on the right. Staged, unstaged and
untracked work are all included.

`Shift+Cmd+P` / `Ctrl+Shift+P` searches commands, files and tabs with fuzzy
matching. `Cmd+,` / `Ctrl+,` opens Settings: General, Appearance, Editor,
Terminal, Shell, Notifications, Keyboard, Hosts, Git, Advanced and Updates.
Its search finds any preference by name or keyword, and `Cmd+F` focuses the
search field. The window follows the system appearance by default; View ›
Appearance overrides it.

## Remote servers

Add a host in Settings › Hosts, then choose **Set Up Server** on its row.
Mojian uploads `mojian-server` and `libghostty-vt` to
`~/.local/share/mojian` on the host and checks that the server answers. No
root, no listening port and no system service are installed.

Choose the host from the server chip to work on a folder in place: the
Explorer lists the host's files, a file is read when it opens, and saves
and file operations write straight back. Git runs on the host too. A
project once opened stays loaded, so switching back restores its tabs and
terminals without reading it again.

The remote server and its terminals stay alive when the SSH connection or
Mojian disconnects. Each terminal remembers the server that owns it, and
`mojian-server -stop` ends a server explicitly.

## Updates

Settings › Updates checks the latest signed release on GitHub. **Check for
Updates** reports when the app is current or offers a newer version;
**Install Update** downloads it and verifies its signature; **Restart to
Update** applies a staged update and relaunches Mojian.

Self-updates need a writable installation: the macOS app bundle or the
per-user Linux archive. A `.deb` installed in `/opt` is updated by its
package manager instead.

## Keyboard shortcuts

The in-app labels follow the platform. This table shows both spellings.

| Action | macOS | Linux |
|---|---|---|
| Open project | `⌘O` | `Ctrl+O` |
| Open file | `⇧⌘O` | `Ctrl+Shift+O` |
| New file | `⌘N` | `Ctrl+N` |
| New folder | `⇧⌘N` | `Ctrl+Shift+N` |
| Rename selected entry | `F2` | `F2` |
| Move selected entry to Trash | `⌘⌫` | `Ctrl+Backspace` |
| Save | `⌘S` | `Ctrl+S` |
| New tab | `⌘T` | `Ctrl+T` |
| Close pane / tab | `⌘W` / `⇧⌘W` | `Ctrl+W` / `Ctrl+Shift+W` |
| Next / previous tab | `⌃⇥` / `⌃⇧⇥` | `Ctrl+Tab` / `Ctrl+Shift+Tab` |
| Show tab 1-9 | `⌘1…⌘9` | `Ctrl+1…Ctrl+9` |
| Split right / down | `⇧⌘D` / `⇧⌘E` | `Ctrl+Shift+D` / `Ctrl+Shift+E` |
| Split left / up | `⌥⌘←` / `⌥⌘↑` | `Ctrl+Alt+Left` / `Ctrl+Alt+Up` |
| Focus neighboring pane | `⌥` + arrows | `Alt` + arrows |
| Move focused pane | `⌥⇧` + arrows | `Alt+Shift` + arrows |
| Zoom pane | `⇧⌘M` | `Ctrl+Shift+M` |
| Toggle Explorer | `⇧⌘B` | `Ctrl+Shift+B` |
| Toggle terminal | `⇧⌘T` | `Ctrl+Shift+T` |
| SSH terminal | `⇧⌘L` | `Ctrl+Shift+L` |
| Git diff page | `⇧⌘G` | `Ctrl+Shift+G` |
| Command palette | `⇧⌘P` | `Ctrl+Shift+P` |
| Settings | `⌘,` | `Ctrl+,` |

These are the defaults; Settings › Keyboard rebinds them, resets a command
or unbinds it. Terminal and SSH shortcuts are available on macOS and Linux
only.

## Build from source

Mojian needs Go 1.27.3 or later.

```sh
git clone https://github.com/ox0pine/mojian.git
cd mojian

go tool mygo dev        # live-reloading development app
go test ./...           # unit and integration tests
go tool mygo vet .      # go vet plus MyGo's UI lifetime checks
```

Build the release targets you need:

```sh
go tool mygo build -platform darwin/arm64,darwin/amd64
go tool mygo build -platform linux/amd64,linux/arm64
```

The output lives under `build/<os>-<arch>/`. macOS produces `.app` and
`.dmg`; Linux produces `.tar.gz` and `.deb`. MyGo is pinned as a Go tool in
`go.mod`, so the first `go tool mygo` downloads the exact version.

`go tool mygo dev -web` opens the interface in a browser, with files kept
in memory. After changing the session protocol or `cmd/mojian-server`,
rebuild the embedded remote binaries with `sh tools/build-remote.sh`.

## Release

A release is a tag. Pushing `vX.Y.Z` runs
[`.github/workflows/release.yml`](.github/workflows/release.yml), which
tests the tree, builds every platform, creates a draft release, uploads
the installers and `SHA256SUMS`, and publishes only when every upload
succeeds. See [`RELEASING.md`](RELEASING.md).

Each release contains:

- macOS: `mojian-<version>-arm64.dmg`, `mojian-<version>-amd64.dmg`
- Linux: `mojian-<version>-linux-{amd64,arm64}.tar.gz`, matching `.deb` packages, `install.sh`
- Signed update archives (`*-darwin-*.tar.gz`, `*-linux-*.tar.gz`) and `update-<target>.json` manifests
- `SHA256SUMS`

## Design

Ink `#1F1B16`, paper `#FBF7EF`, vermilion `#D24A2B`. In Mojian's identity
almost everything is ink on paper; only the cursor keeps the vermilion,
the one voice the page raises. Marks and exports live in
[`logos/README.md`](logos/README.md).

## Credits

Action icons by [Lucide](https://lucide.dev) (ISC); file icons by
[Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme)
(MIT); type set in [JetBrains Mono](https://www.jetbrains.com/lp/mono/)
(OFL); terminals driven by libghostty-vt and PTYs by
[creack/pty](https://github.com/creack/pty) (MIT). The logo is ink, paper
and one vermilion cursor, its wordmark outlining Literata (SIL OFL).
Window and interaction follow [GoRex](https://github.com/egoist/gorex);
remote hosts and Settings follow [Kero](https://kero.sh). Drawn with
[MyGo](https://mygo.egoist.dev).
