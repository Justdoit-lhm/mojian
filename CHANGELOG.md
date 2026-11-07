# Changelog

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
