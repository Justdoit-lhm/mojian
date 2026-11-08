# Repository Guidelines

## Project Overview

Mojian is a quiet text editor for plain files, written in Go against [MyGo](https://mygo.egoist.dev)'s native UI. It is not a web app: there is no webview, no cgo, no JavaScript build. One Go binary draws its own window (Metal on macOS). Tabs are workspaces; each tab holds a binary split tree of panes, and each pane edits a file, previews a picture or PDF, or runs a terminal. Version `0.2.4`, bundle id `dev.mojian.app`, macOS 13+.

A project is just a folder; it keeps its own tabs and layout. Opened projects stay loaded in a workspace cache (`internal/app/workspace.go`): switching stashes the active tabs, terminals, layout and Git page and restores the next project without reading it again. Edits stay in memory until an explicit `⌘S`. On macOS and Linux every local project is opened through the local session server, so its files, Git and terminals share one process; quitting the app detaches the terminals and reopening reattaches them. Remote projects are edited directly on the host: only a listing is cached, and a file's text is read when it opens. A remote server and its terminals stay alive after the app or SSH connection disconnects. Windows is not supported.

Read `.agents/skills/mygo-maintenance/SKILL.md` before touching UI code; it is the authoritative source for MyGo lifetime and threading rules.

## Architecture & Data Flow

**Immediate-mode UI, one state struct.** `internal/app/app.go:26` holds every piece of app state as plain fields on `*app`. There are no reducers, observables or stores. `(*app).view` (`internal/app/view.go:16`) rebuilds the whole element tree each frame from that state; MyGo calls it again after input, `Window.Update` or `Invalidate`. A second root, `(*app).settingsWindowView` (`internal/app/settings.go:68`), draws the Settings window.

Frame conventions that keep this correct:

- `a.later(fn)` (`app.go:398`) defers a mutation to the next frame. Use it whenever an action would change the collection being built (add/close a tab or pane).
- `a.changed()` (`app.go:421`) marks the layout dirty and arms a 700 ms `time.AfterFunc`; `a.persistLayout` (`app.go:445`) writes once 550 ms have passed without a change.
- `a.invalidate()` (`app.go:183`) requests a redraw of both windows after state changes made outside the view.
- Slow work never runs in a view. Background results land through `Window.Update` and are dropped when a sequence/generation counter moved on — see `git.Page.Seq`/`DiffSeq` (`internal/git/git.go:77-78`) and `assetView.pdfGen` (`internal/app/assetview.go:84`).

**One traced path (open → edit → save).** Tree row click → `(*app).openRel` → `store.byRelPath` → `(*app).openNote` → `pane.Note`; the pane renders `note.buffer`, a single `*editor.Buffer` shared by every pane showing that file. `⌘S` → menu/palette command → `(*app).saveNow` → `store.saveAll` → `store.save` → the project's `fsBackend` (`writeFileAtomic` for a direct file, `session.Client.WriteFile` for a server-backed one) → `buf.MarkSaved()`. Edits stay in memory until an explicit save.

**Terminals live in a second process.** `internal/session` is a Unix-socket server (`session.Serve`, `server.go:59`) speaking JSON lines, `ProtocolVersion = 3` (`proto.go:40`). The app spawns it by re-executing itself with `-server` (`spawn_unix.go:19`), so quitting the app detaches PTYs instead of killing them; reopening reattaches by session id. A remote server is started without a transient flag and stays alive while any terminal session exists, so a dropped SSH connection does not end its shells; `mojian-server -stop` ends it explicitly. Terminal panes are `terminal.Terminal` (MyGo's libghostty plugin) bound to a `session.Stream`. Each layout terminal records `Server` (`local` or an SSH host) so it reattaches to the server that owns it.

**Server projects.** `internal/app/server.go` opens local projects through the local session server on macOS and Linux; the browser keeps its in-memory editor. A `store` holds an `fsBackend` (`internal/app/fs.go`): `directFS` (tests/browser neighbours), `memoryFS`, or `serverFS` (`local` distinguishes a local server from a remote one). `session.Client.ListTree` lists names, sizes, times, kinds and an asset's content hash, a note is read when it opens, and saves and file operations write through the same client. There is no local mirror. `store.refresh` reconciles the one listing into notes, assets and the tree; `fs_direct.go` keeps the eager local listing, `fs_server.go` the lazy one. For remote projects, `internal/app/remote_conn.go` uploads the embedded Linux server + `libghostty-vt` from `internal/remotebin`, opens an `ssh -N -L` forward to the remote socket, and detaches without shutting the server down. Settings › Hosts is the only host source: the app no longer scans `~/.ssh/config`, and each host has a Set Up Server action that uploads the binaries and checks the server answers over the forward. Connections are pooled by host, shared by every loaded project and kept while one uses them; the first connection and project opening run off the UI thread behind a loading card (`openRemoteProjectAsync`). Git runs through the project's server (`internal/app/git_bridge.go`); pictures and PDFs are fetched into a local viewer cache.

**Cross-package contracts.** `git.Host` (`internal/git/git.go:52`) is the only interface the Git page uses to reach the window; `*app` implements it in `internal/app/git_bridge.go`. `internal/app/theme_bridge.go` exists so `app` touches `internal/theme` through one file.

**Persistence.** `settings.json` and `layouts/<fnv64a(projectDir)>.json` live in `mygo.App.Path(mygo.PathUserData)` (`app.go:195`, `598`). Environment overrides: `MOJIAN_VAULT` (project folder), `MOJIAN_DIR` (data/socket/log dir).

## Key Directories

| Path | Purpose |
|---|---|
| `main.go` | Entry point: `-server` runs the session server, otherwise `app.Run()`. |
|`cmd/mojian-server/`|The per-user server uploaded to SSH hosts. Never installed as a service.|
|`cmd/harness/`|Dev-only MCP server (mygo-mcp) that renders the real views headlessly over disposable projects.|
| `internal/app/` | The window in one package: shell, tabs, panes, Explorer, pickers, palette, Settings, file operations, SSH, asset viewers. |
| `internal/session/` | PTY server and client, wire protocol, remote file ops, spawn/lock, platform stubs. |
| `internal/git/` | Working-tree diff page: status/diff reading, parsing, view. |
| `internal/gitenv/` | The environment Git commands run in, without redirect variables. |
| `internal/theme/` | Light/dark `Colors`, `Radius`, background painting, editor theme, font registration. |
| `internal/icons/` | Lucide action icons and Material Icon Theme file icons, cached from embedded SVG. |
| `internal/assets/` | `embed.FS` of icons, brand marks and JetBrains Mono. |
| `internal/motion/` | `ui.ElementTransition` values (`Page`, `Tab`, `Tree`, `Row`, `Pane`, `Colors`), springs for interruptible changes. |
| `internal/remotebin/` | Embedded Linux `mojian-server` binaries and libghostty `.so` files (generated). |
| `test/` | Black-box integration tests (`package integration`, import path `mojian/test`). |
| `tools/build-remote.sh` | Cross-compiles the remote server into `internal/remotebin`. |
| `logos/`, `resources/` | Logo masters/exports; `resources/icon.png` is the app icon embedded at build. |

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
sh tools/build-remote.sh  # rebuild embedded Linux servers (needed after server/protocol changes)
```

`mygo` is pinned as a Go tool in `go.mod`; there is no global install, no Makefile, no justfile, no Bun/Node. `go tool mygo migrate-ui .` previews UI API migrations for MyGo version bumps (`-write` applies).

## Headless UI Harness

`cmd/harness` serves the real `(*app).view` and `settingsWindowView` to an MCP client through [mygo-mcp](https://github.com/auron-labs/mygo-mcp), so an agent can inspect, click, type into and screenshot the interface without a desktop window. `internal/app/harness.go` builds each fixture from a seeded app over a disposable project (welcome note, prose and code files) with `a.harness` set. It is a development tool: never imported by the app.

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

To add a fixture, register it in `cmd/harness/main.go` and add its factory beside the other two in `internal/app/harness.go`.

## Code Conventions & Common Patterns

- **Naming.** Unexported domain types in `app`: `app`, `store`, `note`, `assetFile`, `tab`, `pane`, `termTab`, `remoteConn`. Exported PascalCase only in reusable packages (`git.Page`, `session.Client`, `theme.Colors`). Constructors are `newX`; ids are `nextID("p"|"t")`; enum-like values are unexported ints with `iota` and methods; booleans use `no…`/`hide…`/`is…`.
- **Vocabulary.** Code still says *vault* where the UI says *project*: `store` is the project folder, `note` is a text file, `openStore`/`MOJIAN_VAULT` are the project entry points. Keep this translation in mind when reading identifiers.
- **Adapters.** Cross-package coupling goes through a `*_bridge.go` file with thin wrappers; never import `app` from a lower package.
- **Errors.** Return wrapped context (`fmt.Errorf("session: %w", err)`). User-visible failures go through `a.fail(err)` and render in the error banner; background best-effort failures use `log.Print`. Do not panic on I/O; message text is lowercase prose.
- **Concurrency.** Mutate `app` state only on the UI thread. Views must not start goroutines or keep elements/parts — store `ui.Handle`, app data and widget state instead. Give stateful widgets stable identity with `c.Key(key)` before construction; bind persistent fields with `.Bind(&handle)` / `.FocusBind(&field, value)`; read committed focus via `handle.Focused(c)`.
- **Deferred input.** Bound values update after construction; `OnClick`/`OnShortcut` run after construction, `OnChange`/`OnSubmit` observe the rebuilt view. Prefer `handle.OnShortcut(c, mods, key, fn)` for scoped commands.
- **Writes.** Anything user-visible on disk goes through `writeFileAtomic`; never write in place.
- **Motion.** Use `internal/motion` transitions: critically damped springs for pages, tabs, the tree and panes so a change mid-flight reverses instead of restarting, matched geometry for a pane the layout moves, and a short color curve for hover. A new pane fades and slides in with `AnimateTo` because its split is new too; a closing pane leaves the layout at once so the survivor expands, while a copy of the card fades and slides back the same way over its old place (the engine cannot keep an Exit copy when the split goes with it). Terminals skip the size match so their grid does not reflow. MyGo keeps the fade and drops movement when the system asks for less motion.
- **Formatting.** `gofmt` clean, stdlib import block first. Long fluent UI chains go one call per line. Comments are prose sentences explaining why, and cite the reference product (Kero, GoRex, Ghostty) when a design follows one.
- **Platform split.** Real implementations carry `//go:build darwin || linux`; stubs in `*_other.go` return `errUnsupported`. Browser builds are guarded with `runtime.GOOS == "js"`.

## Important Files

| File | Role |
|---|---|
| `internal/app/app.go` | `app` struct, `Run`, open, project list, layout debounce, `fail`/`later`. |
| `internal/app/window.go` | Window/tray lifecycle, Settings window opener, theme application. |
| `internal/app/settings_file.go` | `settingsFile`, load/save/reset persistence, atomic writes, font getters. |
| `internal/app/panes.go` | Opening notes/assets, tabs, pane close/zoom/selection. |
| `internal/app/menu.go` | Menu construction, command dispatch, window title, reveal project. |
| `internal/app/model.go` | `pane`/`tab`, the `node` split tree, pane directions and moves. |
| `internal/app/layout.go` | Layout snapshot/restore and its JSON format. |
| `internal/app/store.go` | `store` state: notes, assets, listing cache, save and refresh. |
| `internal/app/fs.go` | `fsBackend` interface, path guards and backend capability helpers. |
| `internal/app/fs_direct.go` / `fs_memory.go` / `fs_server.go` | Direct folder, in-memory and session-server backends. |
| `internal/app/store_fs.go` | Generic note loading, asset fetch, path guards and connection handoff. |
| `internal/app/prefs.go` | The one settings struct, defaults and range validation. |
| `internal/app/listui.go` | Shared list navigation, command dialog shell and file picker row. |
| `internal/app/settings.go` | Settings window, search model, rail and row primitives. |
| `internal/app/settings_rows.go` | Generic table-driven preference pages and prefs access. |
| `internal/app/settings_pages.go` | Advanced, Updates and About pages plus command availability. |
| `internal/app/settings_keyboard.go` / `settings_hosts.go` / `settings_fonts.go` | Keyboard, Hosts and font controls. |
| `internal/app/view.go` | Root view: title bar, workspace switch, closing panes, error banner. |
| `internal/app/pane.go` | Pane card, header, drag/drop and empty-pane picker. |
| `internal/app/welcome.go` | Welcome/empty state and recent files. |
| `internal/app/tree.go` / `tree_view.go` | Project listing tree/state and its sidebar rows/search. |
| `internal/app/files.go` / `fileclip.go` / `files_tree.go` | File operations, clipboard paste, tree drafts. |
| `internal/app/term.go` | Terminal panes, session attach, async restore, detach. |
| `internal/app/remote_conn.go` | SSH argv, install/provision, forward, connection pool. |
| `internal/app/remote_project.go` | Remote project open, reconnect, loading task state. |
| `internal/app/remote_picker.go` | Host config/UI, picker, busy card, server chip. |
| `internal/app/server.go` | Local session-server project opener and `localServerID`. |
| `internal/settings` | Pure Settings page table: categories, rows, kinds, search words. |
| `internal/git/git.go` / `parse.go` / `view.go` | Git page state, status/diff parsing, and view. |
| `internal/filekind` | Shared text/asset/skip-dir classification for the app and the session server. |
| `internal/app/palette.go` | Command table used by menu, palette and browser shortcuts. |
| `mygo.json` | App manifest: name, identifier, version, `out`, macOS minimum. |
| `go.mod` | Go version, MyGo pin, `tool` directive. |

## Runtime/Tooling Preferences

- **Go 1.27.3**, module `mojian`. No cgo anywhere (`mygo build` and `tools/build-remote.sh` set `CGO_ENABLED=0`); keep it that way. MyGo `v0.7.4` — check the pinned source at `$(go list -m -f '{{.Dir}}' github.com/egoist/mygo)` before using an API, especially `ui/doc.go` and the widget's implementation comments.
- No linter or formatter config exists; `gofmt` and `go vet` are the standard. `go tool mygo vet .` additionally rejects stored UI build values and goroutine captures.
- Build output is `build/<os>-<arch>/`; `/build/`, `/.mygo/` and the root `mojian` binary are gitignored. Do not commit them.
- `internal/remotebin/*.so` and `internal/remotebin/mojian-server-linux-*` are build artifacts embedded by `go:embed`; regenerate with `tools/build-remote.sh`, never hand-edit.
- `resources/icon.png` is a copy of `logos/png/mojian-app-icon-reversed-1024.png` for Finder/Dock; logo rules (palette, clear space, no recoloring the cursor) are in `logos/README.md`.
- No CI: verification is local. Run before yielding: `go test ./...`, `go tool mygo vet .`, and `go tool mygo dev` when the change is visible (a screenshot or the running app, not just a green test).

## Testing & QA

- **Framework:** stdlib `testing` only — no testify, gomock, golden files or coverage thresholds. Test functions live in `_test.go` files; no subtests, no `t.Parallel()`, no benchmarks.
- **White-box tests** sit beside the code (`package app`, `session`, `git`, `icons`) and read unexported state. **Black-box tests** live in `test/` (`package integration`) with real temp folders and real Git; `test/helpers.go` exports `WriteVault` and `InitRepo` (the latter isolates Git with `GIT_CONFIG_GLOBAL=/dev/null`).
- **UI tests** render headlessly with `ui.NewTester(a.view, 1100, 700)` (Settings: `settingsWindowView`, 900×640) and assert on `HasText`/`Texts`, clicks, typing and keys. Call `registerFonts()` before measuring text. Code that uses `a.later` needs an extra `tt.Frame()` — often two.
- **Setup patterns:** `t.TempDir()` plus `t.Setenv` for `MOJIAN_VAULT`, `MOJIAN_DIR` and `HOME`. Terminal tests start `session.Serve` in a goroutine and poll `session.SocketPath()` with a 3 s deadline. PTY and socket tests are Unix-only via build tags. Never add `t.Parallel()` to tests that set env or bind sockets.
- **Injection seams** exist for tests, keep them working: `app.shellStarter`, `trashPath`, `provisionRemoteFn`, `readClipboardFiles`/`writeClipboardFiles`, `remoteConn` values in the `remotes` pool, and the `dataDir`/`layoutPath`/`settingsPath` fields.
- **Expectations:** behavior, boundaries and regressions — not wiring. Add a regression test only where the failure mode is plausible; verify features with a throwaway script, the headless harness, or the running app instead of new permanent tests.
