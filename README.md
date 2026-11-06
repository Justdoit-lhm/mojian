# Mojian

A quiet workspace for Markdown notes, built with [MyGo](https://mygo.egoist.dev)'s
native UI: no webview, no cgo, one small Go binary. Tabs hold workspaces,
workspaces hold panes, and every pane shows a note as its Markdown source or
as the rendered page.

![Mojian](docs/screenshot.png)

## What it does

- **Tabs and split panes.** A tab is a workspace; each pane opens a note. Split
  right or down, drag the divider, double-click it for half and half, zoom a
  pane, or close it. Panes are cards over the window's warm gradient.
- **Markdown, both ways.** The editor has syntax highlighting, soft wrap, undo,
  find, multiple carets and the usual editor keys. Toggle a pane to the preview
  to read the note with its headings, lists, tables, code and math.
- **One editor per note.** A note opens as an editor where it is being written;
  the same note elsewhere opens as a preview, so two views never fight over one
  buffer.
- **A vault of plain files.** Notes are `.md` files in `~/Documents/Mojian` (or
  `$MOJIAN_VAULT`), so another editor can open them too. Saving is atomic and
  automatic; the layout comes back after a restart.
- **A notes popover and a command palette.** The chip in the title bar lists and
  searches the vault; `⇧⌘P` runs every command and jumps to any note or tab.
- **Light, dark, or the system.** Both palettes are hand-tuned; the window
  follows the system by default, and View ▸ Appearance overrides it.

## Shortcuts

| macOS | Action |
|---|---|
| `⌘N` | New note |
| `⌘T` | New tab |
| `⇧⌘D` / `⇧⌘E` | Split right / down |
| `⇧⌘V` | Editor ↔ preview |
| `⌘W` / `⇧⌘W` | Close pane / tab |
| `⇧⌘M` | Zoom the pane |
| `⌃⌘=` | Equalize panes |
| `⌘1…⌘9` | Show tab |
| `⌃⇥` / `⌃⇧⇥` | Next / previous tab |
| `⇧⌘R` | Rename tab |
| `⇧⌘P` | Command palette |
| `⇧⌘O` | Open a note |
| `⌘S` | Save now |

Double-click a tab to rename it; double-click a pane's header to zoom it.

## Develop

Mojian needs Go 1.27 or later. MyGo is pinned in `go.mod` as a tool, so no
global install and no Bun are needed:

```sh
go tool mygo dev     # Mojian Dev, rebuilt as the code changes
go test ./...        # the model, the store and the view, without a window
go tool mygo build   # build/darwin-arm64/Mojian.app and a .dmg
```

`mygo dev` keeps its notes in the same `~/Documents/Mojian` vault, but its
window state and preferences in `Mojian Dev`, apart from the installed app.
`MOJIAN_VAULT` names another vault.

The native UI can also be developed in a browser:

```sh
go tool mygo dev -web
```

There, the notes live in memory for the session and the file system is not
touched.

## Layout

| | |
|---|---|
| `main.go` | the app, its window, menus, commands and lifecycle |
| `store.go` | the vault: Markdown files, new notes, atomic saves, autosave |
| `model.go` | notes, tabs, panes, the split tree and the saved layout |
| `view.go` | the title bar, pane cards, splitters, the empty state |
| `tabs.go` | the tab strip, note tiles, rename, tab menu |
| `palette.go` | the command palette and the vault popover |
| `style.go` | palettes, the window gradient, fonts and icons |
| `assets/` | Lucide icons (ISC) and JetBrains Mono (OFL) |

## Credits

Icons: [Lucide](https://lucide.dev), ISC (`assets/icons/LICENSE`).
Font: [JetBrains Mono](https://www.jetbrains.com/lp/mono/), OFL
(`assets/fonts/OFL.txt`).
The window's shape and interaction follow [GoRex](https://github.com/egoist/gorex),
also built with MyGo.
