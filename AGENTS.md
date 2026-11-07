# Repository Guidelines

## Project Overview

Mojian is a quiet text editor for plain files, written in Go against [MyGo](https://mygo.egoist.dev)'s native UI. It is not a web app: there is no webview, no cgo, no JavaScript build. One Go binary draws its own window (Metal on macOS). Tabs are workspaces; each tab holds a binary split tree of panes, and each pane edits a file, previews a picture or PDF, or runs a terminal. Version `0.1.0`, bundle id `dev.mojian.app`, macOS 13+.

A project is just a folder; it keeps its own tabs and layout. Edits stay in memory until an explicit `⌘S`, terminals survive quitting because they live in a detached session server, and remote projects are edited over SSH.

Read `.agents/skills/mygo-maintenance/SKILL.md` before touching UI code; it is the authoritative source for MyGo lifetime and threading rules.

## Architecture & Data Flow

**Immediate-mode UI, one state struct.** `internal/app/app.go:28` holds every piece of app state as plain fields on `*app`. There are no reducers, observables or stores. `(*app).view` (`internal/app/view.go:21`) rebuilds the whole element tree each frame from that state; MyGo calls it again after input, `Window.Update` or `Invalidate`. A second root, `(*app).settingsWindowView` (`internal/app/settings.go:50`), draws the Settings window.

Frame conventions that keep this correct:

- `a.later(fn)` (`app.go:675`) defers a mutation to the next frame. Use it whenever an action would change the collection being built (add/close a tab or pane).
- `a.changed()` (`app.go:698`) marks the layout dirty and arms a 700 ms `time.AfterFunc`; `a.persistLayout` (`app.go:722`) writes once 550 ms have passed without a change.
- `a.invalidate()` (`app.go:551`) requests a redraw of both windows after state changes made outside the view.
- Slow work never runs in a view. Background results land through `Window.Update` and are dropped when a sequence/generation counter moved on — see `git.Page.Seq`/`DiffSeq` (`internal/git/git.go:103-118`) and `assetView.pdfGen` (`internal/app/assetview.go:78`).

**One traced path (open → edit → save).** Tree row click → `(*app).openRel` (`app.go:876`) → `store.byRelPath` (`store.go:572`) → `(*app).openNote` → `pane.Note`; the pane renders `note.buffer`, a single `*editor.Buffer` shared by every pane showing that file. `⌘S` → menu/palette command → `(*app).saveNow` (`app.go:647`) → `store.saveAll` → `store.save` → `writeFileAtomic` (`app.go:451`, tmp file + fsync + rename) → optional `store.onSave` remote upload → `buf.MarkSaved()`. Edits stay in memory until an explicit save.

**Terminals live in a second process.** `internal/session` is a Unix-socket server (`session.Serve`, `server.go:48`) speaking JSON lines, `ProtocolVersion = 1` (`proto.go:16`). The app spawns it by re-executing itself with `-server` (`spawn_unix.go:19`), so quitting the app detaches PTYs instead of killing them; reopening reattaches by session id. Terminal panes are `terminal.Terminal` (MyGo's libghostty plugin) bound to a `session.Stream`.

**Remote projects.** `internal/app/remote.go` uploads the embedded Linux server + `libghostty-vt` from `internal/remotebin`, opens an `ssh -N -L` forward to the remote socket, and mirrors the project into a local cache (`remoteCacheDir`) synced by a `.files` manifest. Remote writes go through `session.Client.WriteFile`.

**Cross-package contracts.** `git.Host` (`internal/git/git.go:34`) is the only interface the Git page uses to reach the window; `*app` implements it in `internal/app/git_bridge.go`. `internal/app/theme_bridge.go` exists so `app` touches `internal/theme` through one file.

**Persistence.** `settings.json` and `layouts/<fnv64a(projectDir)>.json` live in `mygo.App.Path(mygo.PathUserData)` (`app.go:195`, `598`). Environment overrides: `MOJIAN_VAULT` (project folder), `MOJIAN_DIR` (data/socket/log dir).

## Key Directories

| Path | Purpose |
|---|---|
| `main.go` | Entry point: `-server` runs the session server, otherwise `app.Run()`. |
| `cmd/mojian-server/` | The per-user server uploaded to SSH hosts. Never installed as a service. |
| `internal/app/` | The window in one package: shell, tabs, panes, Explorer, pickers, palette, Settings, file operations, SSH, asset viewers. |
| `internal/session/` | PTY server and client, wire protocol, remote file ops, spawn/lock, platform stubs. |
| `internal/git/` | Working-tree diff page: status/diff reading, parsing, view. |
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
CGO_ENABLED=0 go test ./...
go test ./internal/app -run '^TestGitPage$' -v
go test -race ./internal/session ./internal/app
sh tools/build-remote.sh  # rebuild embedded Linux servers (needed after server/protocol changes)
```

`mygo` is pinned as a Go tool in `go.mod`; there is no global install, no Makefile, no justfile, no Bun/Node. `go tool mygo migrate-ui .` previews UI API migrations for MyGo version bumps (`-write` applies).

## Code Conventions & Common Patterns

- **Naming.** Unexported domain types in `app`: `app`, `store`, `note`, `assetFile`, `tab`, `pane`, `termTab`, `remoteConn`. Exported PascalCase only in reusable packages (`git.Page`, `session.Client`, `theme.Colors`). Constructors are `newX`; ids are `nextID("p"|"t")`; enum-like values are unexported ints with `iota` and methods; booleans use `no…`/`hide…`/`is…`.
- **Vocabulary.** Code still says *vault* where the UI says *project*: `store` is the project folder, `note` is a text file, `openStore`/`MOJIAN_VAULT` are the project entry points. Keep this translation in mind when reading identifiers.
- **Adapters.** Cross-package coupling goes through a `*_bridge.go` file with thin wrappers; never import `app` from a lower package.
- **Errors.** Return wrapped context (`fmt.Errorf("session: %w", err)`). User-visible failures go through `a.fail(err)` and render in the error banner; background best-effort failures use `log.Print`. Do not panic on I/O; message text is lowercase prose.
- **Concurrency.** Mutate `app` state only on the UI thread. Views must not start goroutines or keep elements/parts — store `ui.Handle`, app data and widget state instead. Give stateful widgets stable identity with `c.Key(key)` before construction; bind persistent fields with `.Bind(&handle)` / `.FocusBind(&field, value)`; read committed focus via `handle.Focused(c)`.
- **Deferred input.** Bound values update after construction; `OnClick`/`OnShortcut` run after construction, `OnChange`/`OnSubmit` observe the rebuilt view. Prefer `handle.OnShortcut(c, mods, key, fn)` for scoped commands.
- **Writes.** Anything user-visible on disk goes through `writeFileAtomic`; never write in place.
- **Motion.** Use `internal/motion` transitions: critically damped springs for pages, tabs, the tree and panes so a change mid-flight reverses instead of restarting, matched geometry for a pane the layout moves, and a short color curve for hover. A new pane fades and slides in with `AnimateTo` because its split is new too; terminals skip the size match so their grid does not reflow. MyGo keeps the fade and drops movement when the system asks for less motion.
- **Formatting.** `gofmt` clean, stdlib import block first. Long fluent UI chains go one call per line. Comments are prose sentences explaining why, and cite the reference product (Kero, GoRex, Ghostty) when a design follows one.
- **Platform split.** Real implementations carry `//go:build darwin || linux`; stubs in `*_other.go` return `errUnsupported`. Browser builds are guarded with `runtime.GOOS == "js"`.

## Important Files

| File | Role |
|---|---|
| `internal/app/app.go` | `app` struct, `Run`, window lifecycle, menus/commands, settings, layout debounce, `fail`/`later`. |
| `internal/app/model.go` | `pane`/`tab`, the `node` split tree, pane directions and moves, layout snapshot/restore (persisted as `columns`). |
| `internal/app/store.go` | Project folder, `note` buffers, text/asset classification, atomic save, `onSave` hook. |
| `internal/app/view.go` | Root view: title bar, workspace switch, pane cards, error banner. |
| `internal/app/term.go` | Terminal panes, session client bootstrap, attach, restore, detach on close. |
| `internal/app/remote.go` | SSH connections, server upload, mirror download/sync. |
| `internal/app/files.go` | Rename/copy/move/trash/paste, `absRel` path guard, tree drafts. |
| `internal/app/palette.go` | Command table used by menu, palette and browser shortcuts. |
| `mygo.json` | App manifest: name, identifier, version, `out`, macOS minimum. |
| `go.mod` | Go version, MyGo pin, `tool` directive. |

## Runtime/Tooling Preferences

- **Go 1.27.3**, module `mojian`. No cgo anywhere (`mygo build` and `tools/build-remote.sh` set `CGO_ENABLED=0`); keep it that way. MyGo `v0.7.4` — check the pinned source at `$(go list -m -f '{{.Dir}}' github.com/egoist/mygo)` before using an API, especially `ui/doc.go` and the widget's implementation comments.
- No linter or formatter config exists; `gofmt` and `go vet` are the standard. `go tool mygo vet .` additionally rejects stored UI build values and goroutine captures.
- Build output is `build/<os>-<arch>/`; `/build/`, `/.mygo/` and the root `mojian` binary are gitignored. Do not commit them.
- `internal/remotebin/*.so` and `internal/remotebin/mojian-server-linux-*` are build artifacts embedded by `go:embed`; regenerate with `tools/build-remote.sh`, never hand-edit.
- `resources/icon.png` is a copy of `logos/png/mojian-app-icon-1024.png`; logo rules (palette, clear space, no recoloring the cursor) are in `logos/README.md`.
- No CI: verification is local. Run before yielding: `go test ./...`, `go tool mygo vet .`, and `go tool mygo dev` when the change is visible (a screenshot or the running app, not just a green test).

## Testing & QA

- **Framework:** stdlib `testing` only — no testify, gomock, golden files or coverage thresholds. 74 test functions across 15 `_test.go` files; no subtests, no `t.Parallel()`, no benchmarks.
- **White-box tests** sit beside the code (`package app`, `session`, `git`, `icons`) and read unexported state. **Black-box tests** live in `test/` (`package integration`) with real temp folders and real Git; `test/helpers.go` exports `WriteVault` and `InitRepo` (the latter isolates Git with `GIT_CONFIG_GLOBAL=/dev/null`).
- **UI tests** render headlessly with `ui.NewTester(a.view, 1100, 700)` (Settings: `settingsWindowView`, 900×640) and assert on `HasText`/`Texts`, clicks, typing and keys. Call `registerFonts()` before measuring text. Code that uses `a.later` needs an extra `tt.Frame()` — often two.
- **Setup patterns:** `t.TempDir()` plus `t.Setenv` for `MOJIAN_VAULT`, `MOJIAN_DIR` and `HOME` (SSH tests write `.ssh/config`). Terminal tests start `session.Serve` in a goroutine and poll `session.SocketPath()` with a 3 s deadline. PTY and socket tests are Unix-only via build tags. Never add `t.Parallel()` to tests that set env or bind sockets.
- **Injection seams** exist for tests, keep them working: `app.shellStarter` (`app.go:92`), `trashPath` (`files.go:684`), `store.onSave`, `remoteConn` literals, and the `dataDir`/`layoutPath`/`settingsPath` fields.
- **Expectations:** behavior, boundaries and regressions — not wiring. Add a regression test only where the failure mode is plausible; verify features with a throwaway script or the running app instead of new permanent tests.
- **Known drift:** `README.md` claims white-box tests in `internal/theme`; none exist. `internal/app/main_test.go` contains an unused `newTestApp` helper. Do not "fix" either by inventing coverage.
