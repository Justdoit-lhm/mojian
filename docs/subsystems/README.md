# Subsystem references

These pages describe the current responsibilities, source of truth, runtime paths, state and
contracts of Mojian's subsystems. Start with the page for the behavior being changed, then follow
its source links; `AGENTS.md` is the short index that points here.

| Subsystem | Page |
|---|---|
| UI shell and frame conventions | [ui-shell.md](ui-shell.md) |
| Project store and filesystem backends | [projects-and-storage.md](projects-and-storage.md) |
| Session server and remote hosts | [session-and-remote.md](session-and-remote.md) |
| Git page | [git-page.md](git-page.md) |
| Persistence, settings and layout | [persistence-and-layout.md](persistence-and-layout.md) |

## Maintaining a reference

- One fact lives on one page. When another page needs it, link to the owning page instead of
  copying the sentence.
- Links name files and symbols, never line numbers: a line number is stale the next commit.
- Describe what the code does today. A proposal belongs to an issue, not to a reference.
- Update the owning page in the same change that alters a responsibility, an interface, a state
  transition, a runtime path, a compatibility rule or a failure behavior. A refactor that preserves
  all of those needs no prose change.
- Verification commands must stay runnable from the repository root.
