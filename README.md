# Mojian · 墨间

> Words on paper. The tool steps back.

<p>
  <a href="https://awesomemygo.dev/apps/mojian"><img src="https://awesomemygo.dev/badge.svg" alt="Listed on Awesome MyGo" height="22"></a>
  <a href="https://github.com/ox0pine/mojian/releases"><img src="https://img.shields.io/github/v/release/ox0pine/mojian?style=flat&amp;labelColor=1F1B16&amp;color=D24A2B" alt="Latest release" width="104" height="22"></a>
  <a href="https://github.com/ox0pine/mojian/releases"><img src="https://img.shields.io/github/downloads/ox0pine/mojian/total?style=flat&amp;labelColor=1F1B16&amp;color=D24A2B" alt="Total downloads" width="110" height="22"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/ox0pine/mojian?style=flat&amp;labelColor=1F1B16&amp;color=1F1B16" alt="MIT license" width="86" height="22"></a>
</p>

Mojian is a native plain-text editor for ordinary folders. Edit files, preview images and PDFs, run persistent terminals, and inspect Git diffs in tiled workspaces. No database, account, webview, or cgo.

[Download the latest release](https://github.com/ox0pine/mojian/releases) · [简体中文](README.zh-CN.md) · [Build from source](#build-from-source)

![Mojian with an editor, a pane choosing a file, and two terminals in one tab](docs/screenshot.png?v=6b38e5e)

## Features

- **Ordinary files and folders.** A project is a folder on disk. Nothing is imported, indexed, or synced, and edits stay in memory until you save.
- **Tabs as workspaces.** Each tab contains a split tree of editors, image and PDF viewers, and terminals.
- **Persistent terminals.** Shells run in a detached session server and reattach when Mojian opens again.
- **Built-in Git diffs.** Inspect staged, unstaged, and untracked work without leaving the window.
- **Remote projects over SSH.** Files, Git, and terminals run directly on the configured host.
- **Native UI in one Go binary.** [MyGo](https://mygo.egoist.dev) draws the interface with Metal on macOS and GTK 3 on Linux. No Electron or JavaScript build.

## Installation

Download the latest build from [Releases](https://github.com/ox0pine/mojian/releases).

### macOS

| Architecture | Download |
|---|---|
| Apple Silicon | `mojian-<version>-arm64.dmg` |
| Intel | `mojian-<version>-amd64.dmg` |

Open the disk image and drag Mojian into Applications. Release builds are ad-hoc signed rather than notarized. If macOS blocks the first launch, right-click Mojian in Finder and choose **Open**.

### Linux

Mojian requires GTK 3 and a desktop session.

Install the Debian package:

```sh
sudo apt install ./mojian_<version>_<arch>.deb
```

For a per-user installation without root, put `install.sh` and `mojian-<version>-linux-<arch>.tar.gz` in the same folder, then run:

```sh
sh install.sh
```

## Projects and files

Press `Cmd+O` on macOS or `Ctrl+O` on Linux to open a folder. The project chip switches between recent projects, and the server chip switches between Local and configured SSH hosts. Each project keeps its own tabs, panes, and recent files.

From a terminal, pass the project folder or a file:

```sh
mojian .                        # the current folder
mojian ~/git/abc                # a project folder
mojian ~/git/abc/README.md      # a file, with its folder as the project
```

The file opens in the focused pane of the project's saved layout. The app bundle's binary is `Mojian.app/Contents/MacOS/Mojian`: link it as `mojian` to use the name in these examples, and run `mojian --help` for the usage.

On first launch, Mojian opens `~/Documents/Mojian` when no project is selected and leaves a welcome note if the folder is empty.

The Explorer shows the folder as it exists on disk:

- Toggle it with `Shift+Cmd+B` / `Ctrl+Shift+B`.
- Filter the visible tree with its search field.
- Search every project file with **Open File**, `Shift+Cmd+O` / `Ctrl+Shift+O`.
- Create files and folders directly in the tree.
- Right-click to open, rename, duplicate, copy, cut, copy a path, reveal, or move an entry to trash.

Generated folders such as `.git`, `node_modules`, `dist`, and `build` stay hidden.

## Tabs, panes, and terminals

Press `Cmd+T` / `Ctrl+T` to open a tab. Split the current pane right or down with `Shift+Cmd+D` / `Shift+Cmd+E`, or left and up with `Option+Cmd` / `Ctrl+Alt` and the arrow keys. A new pane opens with an inline file search.

Drag a pane by its header onto another pane's edge to move it. Use `Option` / `Alt` and the arrows to move focus, `Option+Shift` / `Alt+Shift` and the arrows to move the focused pane, and `Shift+Cmd+M` / `Ctrl+Shift+M` to zoom it.

Double-click a tab to rename it, double-click a pane header to zoom it, or right-click a tab for more actions.

Press `Shift+Cmd+T` / `Ctrl+Shift+T` to open a terminal at the project root. Terminals are panes, so they can be split, moved, zoomed, and closed like any other pane. On macOS and Linux, quitting Mojian leaves their shells alive and reopening attaches them again.

Press `Shift+Cmd+L` / `Ctrl+Shift+L` to open an SSH terminal to a host from **Settings › Hosts**.

## Writing and previewing

Code keeps line numbers and prose wraps. Syntax highlighting covers Python, JavaScript, TypeScript, Go, Rust, C and C++, Swift, Kotlin, Ruby, JSON, YAML, TOML, SQL, HTML, CSS, Markdown, and more.

Markdown panes switch between source and preview from the pane header. **Settings › Editor** can render new `.md` files by default. Images, including SVG, open in a contained viewer. PDFs render page by page when `pdftoppm` is installed, with a Quick Look first page on macOS.

Press `Cmd+S` / `Ctrl+S` to write every unsaved note atomically. Unsaved work is also written when the window closes.

## Git, commands, and settings

### Git

Press `Shift+Cmd+G` / `Ctrl+Shift+G` to compare the working tree with `HEAD`. Statuses appear on the left and a two-gutter diff on the right, including staged, unstaged, and untracked work.

### Command palette

Press `Shift+Cmd+P` / `Ctrl+Shift+P` to search commands, files, and tabs with fuzzy matching.

### Settings

Press `Cmd+,` / `Ctrl+,` to open the separate Settings window. It includes General, Appearance, Editor, Terminal, Shell, Notifications, Keyboard, Hosts, Git, Advanced, and Updates.

Search finds preferences by name or keyword, and `Cmd+F` focuses the search field. Mojian follows the system appearance by default; **View › Appearance** overrides it.

## Remote projects

1. Add a host in **Settings › Hosts**.
2. Choose **Set Up Server** on its row. Mojian uploads `mojian-server` and `libghostty-vt` to `~/.local/share/mojian` and checks that the server answers.
3. Choose the host from the server chip and open a folder.

Setup requires no root access, listening port, or system service.

The Explorer lists files on the host. Files are read when opened, and saves and file operations write straight back. Git also runs on the host. Once opened, a project stays loaded, so switching back restores its tabs and terminals without reading it again.

The remote server and its terminals stay alive when Mojian or the SSH connection disconnects. Each terminal remembers the server that owns it. Run `mojian-server -stop` to end a server explicitly.

## Updates

Open **Settings › Updates** to check the latest signed GitHub release:

- **Check for Updates** reports whether a newer version is available.
- **Install Update** downloads it and verifies its signature.
- **Restart to Update** applies a staged update and relaunches Mojian.

Self-updates require a writable installation, such as the macOS app bundle or the per-user Linux archive. A `.deb` installed in `/opt` is updated by its package manager instead.

## Keyboard shortcuts

The labels in Mojian follow the platform. These are the default bindings; **Settings › Keyboard** can rebind, reset, or unbind them.

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

Terminal and SSH shortcuts are available on macOS and Linux only.

## Build from source

Mojian requires Go 1.27.3 or later. MyGo is pinned as a Go tool in `go.mod`, so the first `go tool mygo` command downloads the exact version.

```sh
git clone https://github.com/ox0pine/mojian.git
cd mojian

go tool mygo dev        # live-reloading development app
go test ./...           # unit and integration tests
go tool mygo vet .      # go vet plus MyGo's UI lifetime checks
```

Use `go tool mygo dev -web` to open the interface in a browser with files kept in memory.

### Headless UI harness

`cmd/harness` serves the real window and Settings views to an MCP client
through [mygo-mcp](https://github.com/auron-labs/mygo-mcp), so an agent can
inspect, click, type into and screenshot the interface without a desktop
window:

```sh
go run ./cmd/harness                 # http://127.0.0.1:8080/mcp
go run ./cmd/harness -addr 127.0.0.1:8090
```

Each run seeds a disposable project in the temp directory, and the fixtures
never open native windows, start terminals or touch your projects and
settings. The endpoint has no authentication, so keep it on loopback.

### Release builds

```sh
go tool mygo build -platform darwin/arm64,darwin/amd64
go tool mygo build -platform linux/amd64,linux/arm64
```

Build output lives under `build/<os>-<arch>/`. macOS produces `.app` and `.dmg`; Linux produces `.tar.gz` and `.deb`.

After changing the session protocol or `cmd/mojian-server`, rebuild the embedded remote binaries:

```sh
sh tools/build-remote.sh
```

### Publishing a release

Push a `vX.Y.Z` tag to run [`.github/workflows/release.yml`](.github/workflows/release.yml). The workflow tests the tree, builds every platform, creates a draft release, uploads the installers and `SHA256SUMS`, and publishes only after every upload succeeds. See [`RELEASING.md`](RELEASING.md) for the full process.

Each release contains:

- macOS: `mojian-<version>-arm64.dmg` and `mojian-<version>-amd64.dmg`
- Linux: `mojian-<version>-linux-{amd64,arm64}.tar.gz`, matching `.deb` packages, and `install.sh`
- Signed update archives (`*-darwin-*.tar.gz`, `*-linux-*.tar.gz`) and `update-<target>.json` manifests
- `SHA256SUMS`

## Design

Mojian uses ink `#1F1B16`, paper `#FBF7EF`, and vermilion `#D24A2B`. Almost everything is ink on paper; only the cursor keeps the vermilion, the one voice the page raises.

## Credits

- Action icons: [Lucide](https://lucide.dev), ISC
- File icons: [Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme), MIT
- Interface font: [JetBrains Mono](https://www.jetbrains.com/lp/mono/), OFL
- Wordmark: Literata, SIL OFL
- Terminals: libghostty-vt and [creack/pty](https://github.com/creack/pty), MIT
- Window and interaction references: [GoRex](https://github.com/egoist/gorex) and [Kero](https://kero.sh)
- Native UI: [MyGo](https://mygo.egoist.dev)

## License

MIT. See [`LICENSE`](LICENSE).
