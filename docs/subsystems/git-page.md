# Git page

## Responsibilities

The working-tree page: reading status and diffs for the open project, parsing Git's output, the page
state and its view, and the two runners that reach Git (a local command or the project's session
server).

## Source of truth

- [`internal/git/git.go`](../../internal/git/git.go) — `Runner`, `LocalRunner`, `Host`, `Page`
  (`Seq`, `DiffSeq`, `Loading`, `DiffLoad`, `Changes`, `Sel`, `Diff`), `StatusData`, `Toggle`,
  `Refresh`, `applyStatus`, `Select`, `loadDiff`, `applyDiff`.
- [`internal/git/parse.go`](../../internal/git/parse.go) — `ReadStatus`, `parseStatus`,
  `statusLetter`, `ReadDiff`, `untrackedDiff`, `runGit`, `ParseDiff`, `DiffStat`.
- [`internal/git/view.go`](../../internal/git/view.go) — `(*Page).View`: header, file list, diff
  pane and its empty/loading/error states.
- [`internal/gitenv/gitenv.go`](../../internal/gitenv/gitenv.go) — `Env`, the environment Git runs
  in.
- [`internal/app/git_bridge.go`](../../internal/app/git_bridge.go) — `*app`'s implementation of
  `git.Host`.

## Runtime path

1. `Page.Toggle` opens the page and starts a refresh. `Refresh` calls `ReadStatus` through the
   `Runner` from `Host.GitRunner()` — `LocalRunner` for a direct project, the session server's
   `Exec` for a server-backed one — and stores the result with the sequence number it started at.
2. `applyStatus` drops a result whose sequence is stale, then parses the NUL-separated
   `git status --porcelain` output into `Change` values. `Select` starts `loadDiff`, which calls
   `ReadDiff` in the background and lands through `applyDiff`, guarded by `DiffSeq`.
3. The view reads `Page` and draws; it never starts work. `Host.Invalidate` asks the window for the
   next frame.

## Contracts and failure behavior

- `Change.Status` is one letter: `M` modified, `A` added, `D` deleted, `R` renamed, `U` untracked,
  `C` conflict. `statusLetter` maps `??` to `U` and any code containing `U`, plus `AA`/`DD`, to `C`.
- `Change.Path` stays relative to the repository root so Git commands keep working; `OldPath` keeps
  a rename's source so Git can pair it; `Display` is relative to the project for people.
- A project nested inside a repository reports changes with `Display` relative to the project and
  skips entries outside it.
- `gitenv.Env` removes the variables that redirect Git at another repository, so a hook or a parent
  process cannot change what the page reads.
- Failures render inside the page (`Page.Err`, `Page.DiffErr`) instead of the app's error banner; a
  stale result is dropped, never shown over a newer one.

## Verification

```sh
go test ./internal/git
go test ./internal/app -run '^TestGitPage$' -v
```
