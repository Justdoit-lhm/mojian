# Persistence, settings and layout

## Responsibilities

What survives a launch: the preferences and everything else in `settings.json`, the per-project
`layouts/<fnv64a>.json`, the workspace cache that keeps opened projects loaded, the debounced writes
that produce both files, and the environment overrides that move them.

## Source of truth

- [`internal/prefs/prefs.go`](../../internal/prefs/prefs.go) — `Prefs`, `Default`, `Sanitize`, the
  one preference schema, its defaults and range validation.
- [`internal/app/settings_file.go`](../../internal/app/settings_file.go) — `settingsFile`,
  `loadSettings`, `saveSettingsSoon`, `saveSettings`, and the preference getters.
- [`internal/app/layout.go`](../../internal/app/layout.go) — `layoutFile`, `layoutTab`,
  `layoutNode`, `layoutPane`, `layoutTerminal`, `layoutOf`, `snapshotLayout`, `restoreLayout`,
  `restoreColumns`, `saveLayout`.
- [`internal/app/app.go`](../../internal/app/app.go) — `dataDir`, `settingsPath`,
  `projectLayoutPath`, `migrateLayout`, `changed`, `scheduleLayoutSave`, `persistLayout`.
- [`internal/app/workspace.go`](../../internal/app/workspace.go) — `workspace`,
  `stashActiveWorkspace`, `activateWorkspace`, `closeWorkspaces`.
- [`internal/fs/write.go`](../../internal/fs/write.go) — `WriteFileAtomic`, the only writer both
  files go through.

## Runtime path

1. The data directory comes from `mygo.App.Path(mygo.PathUserData)`; `settingsPath` is
   `settings.json` inside it and each project's layout is `layouts/<fnv64a(projectDir)>.json`.
2. `loadSettings` starts from `prefs.Default()` and unmarshals over it, so a file written by an
   older version keeps sane values for fields it did not know, then calls `Sanitize`. A missing or
   unreadable file leaves the defaults in place.
3. A preference change calls `saveSettingsSoon`, which debounces the write so dragging a slider
   writes once; the write goes through `fs.WriteFileAtomic`.
4. A layout change calls `changed`, which marks the layout dirty and arms the timer;
   `persistLayout` saves once 550 ms pass without another change. `saveLayout` snapshots the tabs
   through `snapshotLayout` and writes atomically.
5. Switching projects calls `stashActiveWorkspace` (the active workspace is the cache entry itself,
   so nothing is copied) and `activateWorkspace` restores the next project's tabs, terminals,
   layout and Git page without reading it again.

## Contracts and failure behavior

- The layout JSON field names are compatibility surface: `tabs`, `active`, `name`, `root`, `focus`,
  `zoom`, `vertical`, `share`, `a`, `b`, `pane`, `id`, `note`, `asset`, `preview`, `terminal`,
  `sid`, `host`, `server`. Legacy `columns`/`panes` are still read through `restoreColumns` so old
  layouts survive.
- A missing layout is not a failure: `restoreLayout` reports false and the window falls back to the
  first note.
- A failed read is not an empty layout. An unreadable or malformed file leaves the current state
  alone rather than replacing it with defaults.
- `MOJIAN_VAULT` selects the project folder; `MOJIAN_DIR` moves the data, socket and log directory
  (see [session-and-remote.md](session-and-remote.md)).
- Both files are written with `fs.WriteFileAtomic`; a partial file is never left behind.
- Switching projects bumps `git.Page.Seq`/`DiffSeq` so a refresh from the project being stashed
  cannot land in the one being activated.

## Verification

```sh
go test ./internal/app -run 'TestSettingsPageAndPersistence|TestLayoutRoundTrip|TestTerminalLayoutRoundTrip|TestProjectSwitch'
```
