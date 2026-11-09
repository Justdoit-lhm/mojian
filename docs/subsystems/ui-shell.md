# UI shell and frame conventions

## Responsibilities

The window: one `app` state struct, the immediate-mode `view` that redraws the whole element tree
each frame, tabs and the binary split tree of panes, pane overlays and zoom, the separate Settings
window, the error banner and command dispatch from menu, palette and keyboard.

## Source of truth

- [`main.go`](../../main.go) — the command line, `detachFromTerminal`, and the `app.Run` entry.
- [`internal/app/app.go`](../../internal/app/app.go) — `app`, `Run`, `open`, `initialStore`,
  `later`, `runPending`, `changed`, `scheduleLayoutSave`, `persistLayout`, `fail`.
- [`internal/app/view.go`](../../internal/app/view.go) — `(*app).view`, the window root:
  title bar, workspace switch, pane overlays and the error banner.
- [`internal/app/window.go`](../../internal/app/window.go) — `(*app).invalidate`, window and tray
  lifecycle.
- [`internal/app/settings_host.go`](../../internal/app/settings_host.go) —
  `(*app).settingsWindowView`, the second root, which draws Settings through
  [`internal/app/settingsui`](../../internal/app/settingsui).
- [`internal/app/model.go`](../../internal/app/model.go) — `tab`, `pane`, the `node` split tree,
  directions, moves and split shares.
- [`internal/app/splitter.go`](../../internal/app/splitter.go) — draggable dividers and share math.
- [`internal/app/headless.go`](../../internal/app/headless.go) — the `Headless` view API that
  `cmd/harness` drives.
- [`internal/app/commands/`](../../internal/app/commands) and
  [`internal/app/palette.go`](../../internal/app/palette.go) — static command metadata, the
  per-window run registry and the command table used by menu, palette and browser shortcuts.
- [`internal/motion/`](../../internal/motion) — the spring transitions the view animates with.

## Runtime path

1. `main` validates the command line and, when standard input is a terminal, `detachFromTerminal`
   re-runs the binary in its own session and exits the parent: the shell returns at once, and
   closing the terminal or pressing Ctrl+C does not end the window. A launch without a terminal
   (Finder, LaunchServices) stays in place, `mygo dev` keeps its child attached, and the detached
   copy appends stdout and stderr to `app.log` under `session.Dir()`.
2. MyGo delivers input, `Window.Update` or `Invalidate`, and calls `(*app).view` again. The view
   builds a new element tree every frame from the fields on `*app`; nothing is cached between
   frames.
3. An action mutates `app` directly. When the mutation would change a collection the current frame
   is still iterating (add or close a tab or pane), the action wraps it in `a.later(fn)`;
   `runPending` calls those closures once the frame is done.
4. `a.changed()` marks the layout dirty, records `lastChange` and arms a 700 ms `time.AfterFunc`.
   `persistLayout` writes through `saveLayout` only after 550 ms without another change; file
   contents are written only by an explicit save.
5. `a.invalidate()` requests a redraw of both windows after a state change made outside the view.
6. `a.fail(err)` stores the message in `a.err`, which the root view renders as the error banner.

## Contracts and failure behavior

- Mutate `app` only on the UI thread.
- Views start no goroutines and keep no elements or parts; store `ui.Handle`, app data and widget
  state on `app` instead.
- Stateful widgets need stable identity: `c.Key(key)` before construction, `.Bind(&handle)` /
  `.FocusBind(&field, value)` for persistent fields, `handle.Focused(c)` for committed focus.
- Slow work never runs in a view. Background results land through `Window.Update` and are dropped
  when a generation or sequence counter moved on — `git.Page.Seq`/`DiffSeq` and `assetView.pdfGen`
  are the examples.
- Bound values update after construction: `OnClick`/`OnShortcut` run after construction,
  `OnChange`/`OnSubmit` observe the rebuilt view.
- The harness replaces native behavior through `a.harness`; any action that needs a real window or
  the session server must be guarded by it, or the headless worker blocks.
- Motion uses `internal/motion`: critically damped springs for pages, tabs, the tree and panes, a
  short color curve for hover; MyGo drops movement when the system asks for less motion.
- Settings › Advanced › Shell command writes the `mojian` alias through `internal/shellcmd` into
  the startup file of the chosen shell, in one marked block. The row's Install, Update or Remove
  button reads `shellcmd.Inspect`, and the headless harness has no executable to install.
- The palette and the appearance are one decision made per frame: `(*app).dark` reads
  `prefs.Appearance` (System follows the frame's theme), `(*app).colorsOf` is
  `theme.Window(a.Theme, dark)`, and `(*app).applyTheme` hands the frame `theme.UI` of that
  palette. `(*app).editorTheme` is cached per theme and appearance pair and re-read by every pane
  each frame, so a theme switch re-colors notes that are already open.
- A forced appearance must reach the native side, which frame colors cannot do: `themeSource`
  maps the preference to `mygo.ThemeSource` and `applyAppearanceSource` calls
  `mygo.Theme.SetSource`, both at window creation (before `mygo.NewWindow`) and on every change.
  A window's own `BackgroundColor` comes from `(*app).windowBackground`, a `light-dark()` pair of
  the open theme, so the strip behind the traffic lights matches without a flash.
- `internal/app/theme_bridge.go` is the only file in `internal/app` that names `internal/theme`
  (besides `settings_host.go`); `archtest` enforces it.

## Verification

```sh
go test ./internal/app
go test ./internal/app -run '^TestGitPage$' -v
go run ./cmd/harness   # then drive the fixture over MCP; see AGENTS.md
```
