# Session server and remote hosts

## Responsibilities

The process that owns terminals and project files: the Unix-socket PTY server, its JSON-lines
protocol, spawn/reattach/detach, the SSH provisioning of a remote server, the forwarded socket, and
the host list and connection pool the app keeps.

## Source of truth

- [`internal/session/proto.go`](../../internal/session/proto.go) — the wire types,
  `ProtocolVersion`, `ProtocolError`, and the directory helpers `Dir`, `RuntimeDir`, `SocketPath`,
  `SocketPathIn`.
- [`internal/session/server.go`](../../internal/session/server.go) — `Serve`, the request loop,
  `session` and `attached` streams, `listTree`, `execCommand`.
- [`internal/session/client.go`](../../internal/session/client.go) — `Connect`, `ConnectAt`,
  `Client`, `Stream`, `Stop`, and the file operations (`ListTree`, `ReadFile`, `WriteFile`,
  `Mkdir`, `Rename`, `Remove`, `Copy`, `Exec`, `Stat`).
- [`internal/session/spawn_unix.go`](../../internal/session/spawn_unix.go) — `Spawn`, which starts
  the server as a detached process of this binary.
- [`internal/remote/remote.go`](../../internal/remote/remote.go) — `Dial`, `Provision`, `Install`
  (upload, start, forward, stop), `Conn`, `Close`.
- [`internal/remotehost/`](../../internal/remotehost) — `Service`: the host list, remembered remote
  projects and the pool of `*remote.Conn` keyed by host.
- [`internal/app/remote_project.go`](../../internal/app/remote_project.go) — opening, reconnecting
  and the loading task state.
- [`internal/app/remote_pool.go`](../../internal/app/remote_pool.go) and
  [`internal/app/remote_bridge.go`](../../internal/app/remote_bridge.go) — the app's view of the
  pool and of `internal/remote`.
- [`cmd/mojian-server/main.go`](../../cmd/mojian-server/main.go) — the uploaded binary; `-stop`
  ends a remote server.
- [`internal/remotebin/`](../../internal/remotebin) — the embedded Linux server binaries and
  `libghostty` shared objects that provisioning uploads.

## Runtime path

1. The app re-executes itself with `-server` (`session.Spawn`) so the server is a child that
   detaches from the window. It keeps its socket, lock and log under `session.Dir()`: `MOJIAN_DIR`
   when set, otherwise `Mojian` inside `os.UserConfigDir()`. A remote server sets
   `MOJIAN_RUNTIME_DIR` to a path sshd can reach with stream-local forwarding, and keeps a long
   path's socket under a hashed name through `SocketPathIn`.
2. A terminal pane connects with `session.Connect`, asks for a `session.Stream` on its session id and
   attaches. Quitting the app closes the client, not the PTY: reopening reattaches to the same
   session by id, and each layout terminal records `Server` (`local` or an SSH host) so it
   reattaches to the server that owns it.
3. A remote host is provisioned once: `remote.Provision` uploads the binaries from
   `internal/remotebin` (comparing hashes first), starts `mojian-server` with `nohup`, and opens
   `ssh -N -L` so a local temporary socket reaches the remote one. `remote.Dial` talks to the server
   through that socket; `Provision` returns the live `Conn` it ended with, and the app keeps it in
   the pool.
4. The remote server stays alive while any of its sessions exists, so a dropped SSH connection does
   not end the shells; `mojian-server -stop` ends it explicitly.

## Contracts and failure behavior

- A server of another build answers with a different version; the client returns
  `session.ProtocolError{Server, Client}` and the app does not attach.
- One socket and one server per data directory: a client dials the socket first and spawns a server
  only when nothing answers, and `server.lock` (an exclusive `flock`) makes a server that loses the
  race exit instead of binding a second socket.
- Remote connections are pooled by host and kept while a loaded project uses them, and after
  `Set Up Server` until the last project on that host is removed (`closeRemoteProject`); `CloseAll`
  ends them at quit. A kept connection opens the folder picker at once, without the window-wide
  loading card, using the home cached in `remoteHomes`. A dead control channel is dropped by
  `remoteForHost` and dialed again on the next use.
- Settings › Hosts is the only host source; the app does not scan `~/.ssh/config`.
- Opening a remote project runs off the UI thread behind the loading card, so an unreachable host
  cannot block the window.
- Missing or unreachable binaries surface as an error from `Provision`; the app reports it and keeps
  the project closed rather than editing a stale listing.

## Verification

```sh
go test ./internal/session
go test -race ./internal/session
MOJIAN_REMOTE_TEST_HOST=user@host go test ./internal/app -run TestRemoteServerSetupFedora -v -timeout 10m
```
