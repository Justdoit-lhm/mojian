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
  eager listing; used by Windows local projects, tests and the browser build's neighbours.
- [`internal/fs/memory.go`](../../internal/fs/memory.go) — `Memory`, the in-memory editor of the
  browser build; every write returns `errReadOnly`.
- [`internal/fs/server.go`](../../internal/fs/server.go) — `Server`, a project reached through a
  `session.Client`, with the lazily cached asset files.
- [`internal/fs/write.go`](../../internal/fs/write.go) — `WriteFileAtomic`.
- [`internal/app/store.go`](../../internal/app/store.go) — `openStoreAt`, `openStore`,
  `openDefaultStore`, `defaultStoreDir`, `resolveProjectRoot`.
- [`internal/app/server.go`](../../internal/app/server.go) — `openServerStore` and
  `openLocalStore`; the latter opens a Windows project through `Direct` and macOS/Linux projects
  through the local session server.
- [`internal/app/project.go`](../../internal/app/project.go) and
  [`internal/app/pane_actions.go`](../../internal/app/pane_actions.go) — `openRel`, `openNote`.

## Runtime path

1. A tree row click reaches `(*app).openRel`, which asks `store.ByRelPath` for the note and calls
   `(*app).openNote`. The pane keeps a `*project.Note`; the pane renders `Note.Buffer()`, one
   `*editor.Buffer` shared by every pane showing that file.
2. `⌘S` runs `(*app).saveNow` → `store.SaveAll` → `store.Save` → the project's `fs.Backend`. A
   `Direct` backend writes with `fs.WriteFileAtomic`; a `Server` backend calls
   `session.Client.WriteFile` for a local server and the same call over the SSH forward for a remote
   one. Before writing, `Store.Save` reads the current file and compares its exact bytes with
   `Note.Saved`, the baseline from the last load or successful save. A mismatch returns
   `project.ErrFileChanged`; a read failure also aborts the save. The buffer is serialized with
   `Buffer.WriteTo`, preserving its LF or CRLF line endings. Only a successful write updates the
   baseline and calls `buf.MarkSaved()`.
3. `Store.Refresh` reads one listing and reconciles it into notes, assets and the tree: notes keep
   their shared buffers; a direct listing reloads changed clean notes in place with `Buffer.Reset`,
   while dirty notes retain both their edits and saved baseline. Vanished clean files drop out;
   vanished dirty files stay available for recovery and are not silently recreated on save.
   Assets keep their cached copy. The read
   is split into `Store.List` (backend only, safe off the UI thread) and `Store.Apply` (mutates the
   store on the UI thread). The Explorer re-reads a server project in the background and shows a
   spinner while it runs; a window focus and opening the file sidebar reuse a listing younger than
   15 seconds, while the Refresh button forces one.

## Contracts and failure behavior

- Edits stay in memory until an explicit save. Nothing in the app writes a buffer on a timer.
- A save conflict leaves both disk and the dirty buffer intact and identifies the file in the
  existing error banner. Users must preserve their local edits separately before reloading.
  Content comparison detects changes even if size and modification time are unchanged. It is a
  pre-write check, not an inter-process lock: another writer can still race between the read and
  atomic replacement. Server-backed projects use the same check through the existing read/write
  protocol; their metadata-only listings do not automatically reload already loaded clean notes.
- `Note.Saved` holds exact on-disk bytes, including line endings. `Note.Text` remains normalized
  once an editor buffer exists; it must not be used to serialize a file for saving.
- A server project has no local mirror. `Server.List` names files with their size, time, kind and an
  asset's content hash; a note's text is read when it opens. `Direct` keeps the eager listing,
  `Server` the lazy one. Listing classifies by extension first: data and model files (`.mat`, `.pt`,
  `.npy`, `.h5`, …) are skipped without opening them, and an unknown extension over 4 MB is treated
  as data rather than reading its head. The session server keeps each project's listing for five
  seconds; `WriteFile`, `Mkdir`, `Rename`, `Remove`, `Copy` and `Exec` clear it, and `ListFresh`
  bypasses it for the app's explicit Refresh. A repeated listing then costs a round trip instead of
  walking hundreds of thousands of files again.
- Everything user-visible on disk that is not a project file (settings, layouts, cached assets) goes
  through `fs.WriteFileAtomic`; never write in place.
- Direct `os.WriteFile` outside these paths is a guard failure: `internal/archtest` allows it only in
  `internal/fs/server.go` (asset cache), `internal/welcome/welcome.go` (starter files) and
  `internal/testutil/testutil.go` (fixtures).
- A path that escapes the project root is rejected by `cleanRel`/`joinRoot` rather than clamped; the
  file operation surfaces the error to `a.fail`.
- A listing that cannot read the project root (for example a denied macOS folder permission) returns
  that error instead of an empty listing, so the store and Explorer keep their previous tree. An
  unreadable folder below a readable root is skipped so the rest of the project stays usable.
- `Memory` lists nothing and refuses file operations with `errReadOnly` (its `Write` is a no-op); the browser build keeps its editor in memory.

## Verification

```sh
go test ./internal/project

go test ./internal/app -run 'TestProjectSwitch|TestLocalProjectStoreUsesSessionServer|TestRemoteProjectReadsAndSavesOnTheServer'
go test ./internal/archtest -run TestDirectWritesStayInTheAllowlist
```
