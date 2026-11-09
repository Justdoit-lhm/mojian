# Repository Guidelines

## Project Overview

Mojian is a quiet text editor for plain files, written in Go against [MyGo](https://mygo.egoist.dev)'s native UI. It is not a web app: there is no webview, no cgo, no JavaScript build. One Go binary draws its own window (Metal on macOS). Tabs are workspaces; each tab holds a binary split tree of panes, and each pane edits a file, previews a picture or PDF, or runs a terminal. The gap between two panes is a divider you drag (or move with the arrows) to resize them; each split saves the share you chose. Version `0.2.6`, bundle id `dev.mojian.app`, macOS 13+.

A project is just a folder; it keeps its own tabs and layout. Opened projects stay loaded in a workspace cache (`internal/app/workspace.go`): switching stashes the active tabs, terminals, layout and Git page and restores the next project without reading it again. Edits stay in memory until an explicit `⌘S`. On macOS and Linux every local project is opened through the local session server, so its files, Git and terminals share one process; quitting the app detaches the terminals and reopening reattaches them. Remote projects are edited directly on the host: only a listing is cached, and a file's text is read when it opens. A remote server and its terminals stay alive after the app or SSH connection disconnects. Windows is not supported.

## Architecture & Data Flow

The paragraphs below are the short version; `docs/subsystems/` holds the contracts, source links and verification commands. Read the owning page before changing a subsystem, and update it when a responsibility, interface, state transition or failure behavior changes.

**Immediate-mode UI, one state struct.** `internal/app/app.go` holds every piece of app state as plain fields on `*app`; there are no reducers, observables or stores. `(*app).view` (`internal/app/view.go`) rebuilds the whole element tree each frame from that state, and `(*app).settingsWindowView` (`internal/app/settings_host.go`) draws the Settings window as a second root. The frame conventions — `a.later`, `a.changed`/`a.persistLayout`, `a.invalidate`, widget identity and the generation counters that drop stale background results — are in [docs/subsystems/ui-shell.md](docs/subsystems/ui-shell.md).

**One traced path (open → edit → save).** Tree row click → `(*app).openRel` → `store.ByRelPath` → `(*app).openNote` → `pane.Note`; the pane renders `Note.Buffer()`, a single `*editor.Buffer` shared by every pane showing that file. `⌘S` → menu/palette command → `(*app).saveNow` → `store.SaveAll` → `store.Save` → the project's `fs.Backend` (`fs.WriteFileAtomic` for a direct file, `session.Client.WriteFile` for a server-backed one) → `buf.MarkSaved()`. Edits stay in memory until an explicit save.

**Terminals live in a second process.** `internal/session` is a Unix-socket server speaking JSON lines (`ProtocolVersion = 3`) that the app spawns by re-executing itself with `-server`, so quitting the app detaches PTYs instead of killing them and reopening reattaches by session id. A remote server is started without a transient flag and stays alive while any terminal session exists, so a dropped SSH connection does not end its shells; `mojian-server -stop` ends it explicitly. Protocol rules, spawn, forwarding and the host pool: [docs/subsystems/session-and-remote.md](docs/subsystems/session-and-remote.md).

**Server projects.** `internal/project.Store` holds an `fs.Backend`: `fs.Direct`, `fs.Memory`, or `fs.Server` (`local` distinguishes a local server from a remote one), and `session.Client.ListTree` lists the project while a note's text is read only when it opens — there is no local mirror, and `Store.Refresh` reconciles that one listing into notes, assets and the tree. Storage rules: [docs/subsystems/projects-and-storage.md](docs/subsystems/projects-and-storage.md); provisioning, forwarding and the host pool: [docs/subsystems/session-and-remote.md](docs/subsystems/session-and-remote.md).

**Cross-package contracts.** `git.Host` (`internal/git/git.go:52`) is the only interface the Git page uses to reach the window; `*app` implements it in `internal/app/git_bridge.go`. `internal/app/theme_bridge.go` exists so `app` touches `internal/theme` through one file, and `platform_bridge.go` / `remote_bridge.go` do the same for `internal/platform` and `internal/remote`.

**Persistence.** `settings.json` and `layouts/<fnv64a(projectDir)>.json` live in `mygo.App.Path(mygo.PathUserData)`; `MOJIAN_VAULT` selects the project folder and `MOJIAN_DIR` the data/socket/log directory. The preference schema is `internal/prefs.Prefs`; the compatibility rules for the files and the layout JSON are in [docs/subsystems/persistence-and-layout.md](docs/subsystems/persistence-and-layout.md).

## Key Directories

`internal/app` is the window's Go package: the immediate-mode view and every action read and mutate the same `*app` state, so the view itself stays in one package. Its UI layers live in subpackages under it: `explorer`, `settingsui` and `widgets`. The domain and shared layers are siblings: `internal/project` (Store/Note/Asset/tree model), `internal/welcome` (starter notes and fixture files), `internal/fs`, `internal/remote`, `internal/update`, `internal/platform`, `internal/prefs`, `internal/session`, `internal/settings`, `internal/git`.

| Path | Purpose |
|---|---|
|`main.go`|Entry point: `-server` runs the session server; otherwise it reads the optional folder or file argument and calls `app.Run` with it.|
|`cmd/mojian-server/`|The per-user server uploaded to SSH hosts. Never installed as a service.|
|`cmd/harness/`|Dev-only MCP server (mygo-mcp) that renders the real views headlessly over disposable projects.|
| `docs/subsystems/` | Contract pages for the UI shell, storage, session/remote, Git and persistence; AGENTS.md is the index. |
| `internal/app/` | The window in one package: shell, tabs, panes, Explorer, pickers, palette, Settings, file operations, SSH, asset viewers. |
| `internal/session/` | PTY server and client, wire protocol, remote file ops, spawn/lock, platform stubs. |
| `internal/git/` | Working-tree diff page: status/diff reading, parsing, view. |
| `internal/gitenv/` | The environment Git commands run in, without redirect variables. |
| `internal/theme/` | Light/dark `Colors`, `Radius`, background painting, editor theme, font registration. |
| `internal/icons/` | Lucide action icons and Material Icon Theme file icons, cached from embedded SVG. |
| `internal/assets/` | `embed.FS` of icons, brand marks and JetBrains Mono. |
| `internal/motion/` | `ui.ElementTransition` values (`Page`, `Tab`, `Tree`, `Row`, `Pane`, `Colors`), springs for interruptible changes. |
| `internal/remotebin/` | Embedded Linux `mojian-server` binaries and libghostty `.so` files (generated); `build.sh` rebuilds the servers. |
| `internal/testutil/` | Fixtures shared by tests: `WriteVault` writes a throwaway project, `InitRepo` commits it in a fresh Git repository. |
| `resources/` | `resources/icon.png` is the app icon embedded at build. |

## Development Commands

```sh
go tool mygo dev          # "Mojian Dev", live reload; state in .mygo/dev
go tool mygo dev -web     # native UI in a browser (wasm); files live in memory
go test ./...             # whole suite, no window needed
go tool mygo vet .        # go vet + MyGo UI lifetime checker
go tool mygo build        # build/<os>-<arch>/Mojian.app and .dmg
go run ./cmd/harness      # long-running MCP server; real views, no window
CGO_ENABLED=0 go test ./...
go test ./internal/app -run '^TestGitPage$' -v
go test -race ./internal/session ./internal/app
go generate ./internal/remotebin  # rebuild embedded Linux servers (needed after server/protocol changes)
```

`mygo` is pinned as a Go tool in `go.mod`; there is no global install, no Makefile, no justfile, no Bun/Node. `go tool mygo migrate-ui .` previews UI API migrations for MyGo version bumps (`-write` applies).

## Headless UI Harness

`cmd/harness` serves the real `(*app).view` and `settingsWindowView` to an MCP client through [mygo-mcp](https://github.com/auron-labs/mygo-mcp), so an agent can inspect, click, type into and screenshot the interface without a desktop window. `cmd/harness/fixtures.go` builds each fixture over a disposable project (welcome note, prose and code files) and `internal/app/headless.go` exposes the real views; the harness is never imported by the app.

Start it in its own long-lived terminal (`rex`), never with `&` or `nohup`:

```sh
go run ./cmd/harness                 # http://127.0.0.1:8080/mcp
go run ./cmd/harness -addr 127.0.0.1:8090
```

The endpoint speaks MCP Streamable HTTP, has no authentication, and must stay on loopback. magpie already registers it for `omp` as `mygo-mcp http://127.0.0.1:8080/mcp` with `--local`; a new agent session is needed before its tools appear, named `mygo-mcp__mygo_*` because the router shares one server. There is no live reload: stop with Ctrl+C and start again after code changes, or the MCP client still drives the old build.

Fixtures are `Mojian` (1120×720) and `Mojian Settings` (900×640). The workflow:

1. `mygo_sessions` gives each fixture's `session_id`.
2. `mygo_screenshot` or `mygo_inspect` gives an observation and its targets.
3. Act with `mygo_click`, `mygo_click_label`, `mygo_type`, `mygo_key` or `mygo_scroll`.
4. Verify with a new screenshot or `mygo_wait_text`.

Every action invalidates the previous observation: pass the new `observation_id`, or request `inspect: true`/`screenshot: true` with the action. `mygo_click_label` accepts exact text or explicit MyGo labels and fails with `ambiguous` when two elements share the label; those rows need screenshot coordinates instead. Key modifiers are `shift`, `control`, `alt`, `meta` and `primary` (Cmd on macOS); printable text goes through `mygo_type`, not `mygo_key`.

A fixture deliberately stops at the view. Native windows, the session server, terminals and project switching are replaced or guarded by `a.harness`: the Settings gear shows a banner pointing at the `Mojian Settings` fixture, terminals fail with a message, and the disposable project stays put. Because there is no native menu, the harness also registers the command accelerators in the view, so `mygo_key` with `primary` drives Save (`S`), the palette (`Shift+P`), tabs and splits as a user would. Fixtures edit and save only their own temp project, which cleanup removes. Any new action that needs a real window or the session server must be guarded by `a.harness` or it can block the headless worker forever.

To add a fixture, register it in `cmd/harness/main.go` and add its factory in `cmd/harness/fixtures.go`.

## Code Conventions & Common Patterns

- **Naming.** Unexported domain types in `app`: `app`, `store`, `note`, `assetFile`, `tab`, `pane`, `termTab`, `remoteConn`. Exported PascalCase only in reusable packages (`git.Page`, `session.Client`, `theme.Colors`). Constructors are `newX`; ids are `nextID("p"|"t")`; enum-like values are unexported ints with `iota` and methods; booleans use `no…`/`hide…`/`is…`.
- **Vocabulary.** Code still says *vault* where the UI says *project*: `store` is the project folder, `note` is a text file, `openStore`/`MOJIAN_VAULT` are the project entry points. Keep this translation in mind when reading identifiers.
- **Adapters.** Cross-package coupling goes through a `*_bridge.go` file with thin wrappers; never import `app` from a lower package.
- **Errors.** Return wrapped context (`fmt.Errorf("session: %w", err)`). User-visible failures go through `a.fail(err)` and render in the error banner; background best-effort failures use `log.Print`. Do not panic on I/O; message text is lowercase prose.
- **Concurrency.** Mutate `app` state only on the UI thread. Views must not start goroutines or keep elements/parts — store `ui.Handle`, app data and widget state instead. Give stateful widgets stable identity with `c.Key(key)` before construction; bind persistent fields with `.Bind(&handle)` / `.FocusBind(&field, value)`; read committed focus via `handle.Focused(c)`.
- **Deferred input.** Bound values update after construction; `OnClick`/`OnShortcut` run after construction, `OnChange`/`OnSubmit` observe the rebuilt view. Prefer `handle.OnShortcut(c, mods, key, fn)` for scoped commands.
- **Writes.** Anything user-visible on disk goes through `writeFileAtomic`; never write in place.
- **Motion.** Use `internal/motion` transitions: critically damped springs for pages, tabs, the tree and panes so a change mid-flight reverses instead of restarting, matched geometry for a pane the layout moves, and a short color curve for hover. A new pane fades and slides in with `AnimateTo` because its split is new too; a closing pane leaves the layout at once so the survivor expands, while a copy of the card fades and slides back the same way over its old place (the engine cannot keep an Exit copy when the split goes with it). Zooming a pane hides the others behind copies that fade over their old places, and the zoomed card, file or terminal, flies to the tab through the same match. MyGo keeps the fade and drops movement when the system asks for less motion.
- **Formatting.** `gofmt` clean, stdlib import block first. Long fluent UI chains go one call per line. Comments are prose sentences explaining why, and cite the reference product (Kero, GoRex, Ghostty) when a design follows one.
- **Platform split.** Real implementations carry `//go:build darwin || linux`; stubs in `*_other.go` return `errUnsupported`. Browser builds are guarded with `runtime.GOOS == "js"`.

## Important Files

| File | Role |
|---|---|
| `internal/app/app.go` | `app` struct, `Run`, open, project list, layout debounce, `fail`/`later`. |
| `internal/app/window.go` | Window/tray lifecycle, Settings window opener, theme application. |
| `internal/app/settings_file.go` | `settingsFile`, load/save/reset persistence, font getters. |
| `internal/app/pane_actions.go` | Opening notes/assets, tabs, pane close/zoom/selection. |
| `internal/app/menu.go` | Menu construction, command dispatch, window title, reveal project. |
| `internal/app/model.go` | `pane`/`tab`, the `node` split tree, pane directions, moves and split shares. |
| `internal/app/layout.go` | Layout snapshot/restore and its JSON format. |
| `internal/app/store.go` | Project openers (`openStoreAt`, `openDefaultStore`, `openStore`) and the default folder. |
| `internal/project/` | `Store`, `Note`, `Asset` and `TreeNode`: listing, save, rename/copy/move, asset cache, editor buffers. |
| `internal/fs/` | Project storage: `Backend`, `Direct`, `Memory`, `Server`, path guards, atomic writes and cached-file names. |
| `internal/prefs/` | The one preference schema, defaults and range validation. |
| `internal/app/widgets_bridge.go` | Thin bridge to `internal/app/widgets`. |
| `internal/app/aliases.go` | The window's names for the project model and the Explorer's draft type. |
| `internal/app/widgets/` | Shared list navigation, command dialog shell and file picker row. |
| `internal/welcome/` | Welcome and sample note text, in the platform's key spelling. |
| `internal/app/settingsui/` | Settings window: search rail, table-driven pages, keyboard recorder, font picker. |
| `internal/app/settings_host.go` | `settingsui.Host` adapter, Settings window wrapper and editor font helper. |
| `internal/app/settings_hosts.go` | Hosts page; it drives host setup and stays app-side. |
| `internal/app/view.go` | Root view: title bar, workspace switch, pane overlays, error banner. |
| `internal/app/splitter.go` | Draggable split dividers, share math, drag and keyboard resizing. |
| `internal/app/pane.go` | Pane card, header, drag/drop and empty-pane picker. |
| `internal/app/welcome.go` | Welcome/empty state and recent files. |
| `internal/app/commands/` | Static command metadata and the per-window run registry. |
| `internal/app/explorer/` | Explorer tree view, file actions, clipboard paste, drafts and search; state stays on the active workspace. |
| `internal/app/term.go` | Terminal panes, session attach, async restore, detach. |
| `internal/remote/` | SSH provisioning, binary upload, forwarding, connection lifetime. |
| `internal/remotehost/` | Host list, remembered remote projects and the connection pool. |
| `internal/app/remote_bridge.go` | The pool and the open project's view of `internal/remote`. |
| `internal/app/remote_project.go` | Remote project open, reconnect, loading task state. |
| `internal/app/remote_picker.go` | Host config/UI, picker, busy card, server chip. |
| `internal/app/headless.go` | `Headless` view API used by `cmd/harness`. |
| `internal/app/server.go` | Local session-server project opener and `localServerID`. |
| `internal/settings` | Pure Settings page table: categories, rows, kinds, search words. |
| `internal/update/` | Release checks, signed install and the GitHub release source. |
| `internal/platform/` | OS key spelling, file-manager labels and terminal support. |
| `internal/git/git.go` / `parse.go` / `view.go` | Git page state, status/diff parsing, and view. |
| `internal/filekind` | Shared text/asset/skip-dir classification for the app and the session server. |
| `internal/app/palette.go` | Command table used by menu, palette and browser shortcuts. |
| `mygo.json` | App manifest: name, identifier, version, `out`, macOS minimum. |
| `go.mod` | Go version, MyGo pin, `tool` directive. |

## Runtime/Tooling Preferences

- **Go 1.27.3**, module `mojian`. No cgo anywhere (`mygo build` and `internal/remotebin/build.sh` set `CGO_ENABLED=0`); keep it that way. MyGo `v0.8.3` — check the pinned source at `$(go list -m -f '{{.Dir}}' github.com/egoist/mygo)` before using an API, especially `ui/doc.go` and the widget's implementation comments.
- No linter or formatter config exists; `gofmt` and `go vet` are the standard. `go tool mygo vet .` additionally rejects stored UI build values and goroutine captures.
- Build output is `build/<os>-<arch>/`; `/build/`, `/.mygo/` and the root `mojian` binary are gitignored. Do not commit them.
- `internal/remotebin/mojian-server-linux-*` and the `.so` files are build artifacts embedded by `go:embed`, never hand-edit. `go generate ./internal/remotebin` rebuilds the servers; the Linux `libghostty-vt-*.so` files are the assets the pinned MyGo's `plugins/terminal/mygo-plugin.json` names, so copy them with their SHA-256 checked after a MyGo upgrade.
- `resources/icon.png` is the app icon for Finder/Dock and is embedded at build; the brand marks the app draws come from `internal/assets/brand`.
- No pull-request CI; `.github/workflows/release.yml` runs `go test ./...` on tag pushes. Verification is local otherwise. Run before yielding: `go test ./...`, `go tool mygo vet .`, and `go tool mygo dev` when the change is visible (a screenshot or the running app, not just a green test).

## Testing & QA

- **Framework:** stdlib `testing` only — no testify, gomock, golden files or coverage thresholds. Test functions live in `_test.go` files; no subtests, no `t.Parallel()`, no benchmarks.
- **White-box tests** sit beside the code (`package app`, `session`, `git`, `icons`) and read unexported state. **Black-box tests** use an external test package beside the code (`package git_test`) with real temp folders and real Git; the fixtures they share live in `internal/testutil`, where `Main` gives a test binary a temporary home and clears `MOJIAN_VAULT`, `MOJIAN_DIR` and Git's user/system config, `WriteVault` writes a throwaway project, and `InitRepo` commits it in a fresh repository. `internal/app`, `internal/session` and `internal/git` call `Main` from `TestMain`, so a test binary cannot read the developer's home or an inherited `MOJIAN_*` path; the only thing taken from the real home is MyGo's downloaded `libghostty-vt`, handed over as `MYGO_GHOSTTY_VT`.
- **UI tests** render headlessly with `ui.NewTester(a.view, 1100, 700)` (Settings: `settingsWindowView`, 900×640) and assert on `HasText`/`Texts`, clicks, typing and keys. Call `registerFonts()` before measuring text. Code that uses `a.later` needs an extra `tt.Frame()` — often two.
- **Setup patterns:** `t.TempDir()` plus `t.Setenv` for `MOJIAN_VAULT`, `MOJIAN_DIR` and `HOME`. Terminal tests start `session.Serve` in a goroutine and poll `session.SocketPath()` with a 3 s deadline. PTY and socket tests are Unix-only via build tags. Never add `t.Parallel()` to tests that set env or bind sockets.
- **Injection seams** exist for tests, keep them working: `app.shellStarter`, `trashPath`, `provisionRemoteFn`, `readClipboardFiles`/`writeClipboardFiles`, `remoteConn` values in the `remotes` pool, and the `dataDir`/`layoutPath`/`settingsPath` fields.
- **Expectations:** behavior, boundaries and regressions — not wiring. Add a regression test only where the failure mode is plausible; verify features with a throwaway script, the headless harness, or the running app instead of new permanent tests.
