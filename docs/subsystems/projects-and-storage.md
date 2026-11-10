# Project store and filesystem backends

## Responsibilities

The open project: `Store`, `Note`, `Asset` and `TreeNode`; the `fs.Backend` implementations that
back them (`Direct`, `Memory`, `Server`); the difference between a local project opened through the
local session server and a remote project edited on its host; listing and `Refresh`, read-on-open,
the save path, path guards and the asset cache.

## Source of truth

- [`internal/project/`](../../internal/project) — `Store` (`New`, `Refresh`, `ByRelPath`, `Save`,
  `SaveAll`, `LoadNote`, `MaterializeAsset`, `BuildTree`, the rename/copy/move operations in
  `files.go`), `Note` with its shared `editor.Buffer`, `Asset`, `TreeNode`.
- [`internal/fs/fs.go`](../../internal/fs/fs.go) — the `Backend` interface, `Kind`, `cleanRel`,
  `joinRoot` and `relOfRoot`. Every path the store accepts goes through the guards.
- [`internal/fs/direct.go`](../../internal/fs/direct.go) — `Direct`, a plain folder on disk with an
  eager listing; used by tests and by the browser build's neighbours.
- [`internal/fs/memory.go`](../../internal/fs/memory.go) — `Memory`, the in-memory editor of the
  browser build; every write returns `errReadOnly`.
- [`internal/fs/server.go`](../../internal/fs/server.go) — `Server`, a project reached through a
  `session.Client`, with the lazily cached asset files.
- [`internal/fs/write.go`](../../internal/fs/write.go) — `WriteFileAtomic`.
- [`internal/app/store.go`](../../internal/app/store.go) — `openStoreAt`, `openStore`,
  `openDefaultStore`, `defaultStoreDir`, `resolveProjectRoot`.
- [`internal/app/server.go`](../../internal/app/server.go) — `openServerStore` and
  `openLocalStore`, the local-session-server opener.
- [`internal/app/project.go`](../../internal/app/project.go) and
  [`internal/app/pane_actions.go`](../../internal/app/pane_actions.go) — `openRel`, `openNote`.

## Runtime path

1. A tree row click reaches `(*app).openRel`, which asks `store.ByRelPath` for the note and calls
   `(*app).openNote`. The pane keeps a `*project.Note`; the pane renders `Note.Buffer()`, one
   `*editor.Buffer` shared by every pane showing that file.
2. `⌘S` runs `(*app).saveNow` → `store.SaveAll` → `store.Save` → the project's `fs.Backend`. A
   `Direct` backend writes with `fs.WriteFileAtomic`; a `Server` backend calls
   `session.Client.WriteFile` for a local server and the same call over the SSH forward for a remote
   one. The save ends with `buf.MarkSaved()`.
3. `Store.Refresh` reads one listing and reconciles it into notes, assets and the tree: notes keep
   their buffers and dirty state, vanished files drop out, assets keep their cached copy. The read
   is split into `Store.List` (backend only, safe off the UI thread) and `Store.Apply` (mutates the
   store on the UI thread). The Explorer re-reads a server project in the background and shows a
   spinner while it runs; a window focus reuses a listing younger than 15 seconds.

## Contracts and failure behavior

- Edits stay in memory until an explicit save. Nothing in the app writes a buffer on a timer.
- A server project has no local mirror. `Server.List` names files with their size, time, kind and an
  asset's content hash; a note's text is read when it opens. `Direct` keeps the eager listing,
  `Server` the lazy one. Listing classifies by extension first: data and model files (`.mat`, `.pt`,
  `.npy`, `.h5`, …) are skipped without opening them, and an unknown extension over 4 MB is treated
  as data rather than reading its head. This keeps a project with hundreds of thousands of data
  files listable in milliseconds.
- Everything user-visible on disk that is not a project file (settings, layouts, cached assets) goes
  through `fs.WriteFileAtomic`; never write in place.
- Direct `os.WriteFile` outside these paths is a guard failure: `internal/archtest` allows it only in
  `internal/fs/server.go` (asset cache), `internal/welcome/welcome.go` (starter files) and
  `internal/testutil/testutil.go` (fixtures).
- A path that escapes the project root is rejected by `cleanRel`/`joinRoot` rather than clamped; the
  file operation surfaces the error to `a.fail`.
- `Memory` lists nothing and refuses file operations with `errReadOnly` (its `Write` is a no-op); the browser build keeps its editor in memory.

## Verification

```sh
go test ./internal/app -run 'TestProjectSwitch|TestLocalProjectStoreUsesSessionServer|TestRemoteProjectReadsAndSavesOnTheServer'
go test ./internal/archtest -run TestDirectWritesStayInTheAllowlist
```
