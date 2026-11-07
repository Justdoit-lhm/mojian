# Changelog

## 0.1.4

- Remote servers keep their socket in `XDG_RUNTIME_DIR/mojian-<uid>` (or
  `/tmp/mojian-<uid>`), not under the home directory. Fedora's SELinux
  labels the latter `data_home_t`, and sshd refuses stream-local
  forwarding to it with `open failed`, so 0.1.3 could not reach a remote
  server on such hosts. Verified over real SSH against Fedora 44: server
  upload, tree listing, lazy reads, saves, file operations, remote Git, a
  terminal's process-group kill, and transient shutdown on disconnect.

## 0.1.3

- Remote projects now edit the host's files directly, like VS Code Remote:
  the Explorer lists names and kinds, a file is read when it opens, saves
  and file operations write straight back, and Git runs on the host. The
  local mirror, its manifest and the full re-upload on every file operation
  are gone; pictures and PDFs are fetched on demand.
- A remote terminal ends with its SSH connection: the remote server is
  transient and shuts down (taking its shells) shortly after the app
  disconnects. Closing a terminal now signals the whole foreground process
  group, so `npm run dev`, `ssh` and `vim` no longer survive the pane.
- The session protocol is version 2, checked with `hello` on connect; a
  stale local server is replaced, and a remote one is restarted with the
  current binary, compared by SHA-256 rather than file size. Session
  frames are bounded, remote writes keep file modes and fsync, and the
  remote archive op is gone.
- Git: rename diffs now pair the old and new paths instead of showing the
  file as new; an untracked file is sized before it is read; refreshing the
  page on focus no longer resets the diff scroll; `GIT_DIR` and friends are
  stripped from the environment.
- Motion runs on short, critically damped springs: page and tab switches
  and the tree reverse from where they are when the state changes again
  mid-flight, instead of restarting a curve. Tree rows glide when a folder
  opens or closes, and the folder's children fold instead of letting the
  rows below jump. Hover colors share one 120 ms curve.
- Opening a pane or a terminal fades and slides the new card in from the
  edge of its split, while the pane it splits glides to its new width; a
  pane the layout moves into another split, or back to full width when one
  closes, follows from where it showed. Terminals skip the size animation,
  so their grid is not reflowed at every width on the way.
- **Open File** now opens a file picker over the window instead of
  focusing the Explorer's search field. Every file of the project is
  listed, typing filters them, and Enter or a click opens one in the
  active tab. This fixes the welcome page's row, which did nothing visible
  when the Explorer was already open.

## 0.1.2

- Settings › Updates now has **Check for Updates**, **Install Update** and
  **Restart to Update**. The check reads the latest signed GitHub release;
  the app verifies the update before installing it, and the restart applies
  a staged or replaced version.
- Signed update archives and manifests are built and published with every
  release. The update signing public key is in `mygo.json`; the private key
  lives only in the repository's GitHub Actions secret.
- Windows releases now also include the NSIS installer beside the portable
  zip and the signed update archive.

## 0.1.1

- Fix **Open File** on the welcome page: it now focuses the Explorer's
  search field, including when the Explorer was already open, so typing
  filters files at once.
- Show shortcuts, reveal labels and the welcome note in the platform's
  own keys and words: `⌘`/`⌥` on macOS, `Ctrl`/`Alt` on Linux and Windows.
- Add a portable Windows 10/11 amd64 build with the editor, Explorer,
  panes, Git page and settings. Terminal, SSH and remote-project features
  are hidden because they need the Unix session server.
- Build and test macOS, Linux and Windows in the tag-driven release
  workflow. Linux now runs the test suite in CI too.
- Rewrite both READMEs with per-platform install steps, a support matrix
  and a cross-platform shortcut table.

## 0.1.0

First release.

- A quiet plain-text editor for macOS and Linux, drawn by MyGo's native UI:
  no webview, no cgo, one Go binary.
- Projects are ordinary folders. The in-window folder picker, recent
  projects and per-project layouts keep each workspace separate.
- Tabs hold a tile of panes. Splits open an empty pane with an inline file
  search; drag a pane onto another pane's edge to place it there, or move it
  with Option+Shift and the arrows.
- Syntax highlighting for the languages MyGo knows, line numbers for code,
  wrapped prose, and Markdown source or preview.
- Explorer with file operations, in-project search, and Material file icons.
- A real terminal pane backed by a persistent session server; quitting the
  app detaches from shells instead of ending them.
- Git working-tree page with status, per-file diff and change stats.
- An SSH project workflow that mirrors a folder from a host and runs its
  terminals and Git diff there.
- Images and PDFs in a contained viewer, Settings in their own window, light
  and dark palettes, and a command palette over every command.
- Linux builds as per-user tar.gz archives and Debian packages for amd64
  and arm64, beside the macOS disk images.
