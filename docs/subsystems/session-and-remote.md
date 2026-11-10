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
   `internal/remotebin` (comparing hashes first), starts `mojian-server` with `nohup` when its socket is
   not there, and opens `ssh -N -L` so a local temporary socket reaches the remote one. `remote.Dial`
   talks to the server through that socket; `Provision` returns the live `Conn` it ended with, and the
   app keeps it in the pool.
4. The remote server stays alive while any of its sessions exists, so a dropped SSH connection does
   not end its shells. It never stops on its own: Settings › Hosts offers **Start Server** and
   **Stop Server** for each configured host, and a stop asks before ending its terminals.
   `mojian-server -stop` ends it explicitly.

## Contracts and failure behavior

- A server of another build answers with a different version; the client returns
  `session.ProtocolError{Server, Client}` and the app does not attach.
- One socket and one server per data directory: a client dials the socket first and spawns a server
  only when nothing answers, and `server.lock` (an exclusive `flock`) makes a server that loses the
  race exit instead of binding a second socket.
- Remote connections are pooled by host and kept while a loaded project uses them, and after
  `Set Up Server` until the last project on that host is removed (`closeRemoteProject`); `CloseAll`
  ends them at quit. Choosing a server that is not the one on screen resumes its most recently
  opened project when that workspace is still cached, so switching to Local and back keeps the tabs,
  terminals and layout; only when nothing on the host is loaded does it open the folder picker, at
  once and without the window-wide loading card, using the home cached in `remoteHomes`. Choosing
  the server already on screen opens the picker, so another folder on it can be opened. A dead
  control channel is dropped by `remoteForHost` and dialed again on the next use.
- Settings › Hosts is the only host source; the app does not scan `~/.ssh/config`. At launch, and
  when the page opens, the app asks each host once (`remote.ServerRunning`) whether its server
  socket still exists, so a server left running by the previous launch shows Stop Server rather
  than Set Up Server. Each host's row sets up, starts and stops that host's server; stopping
  removes the pooled connection and the terminal views it owned, and starting again reconnects the
  loaded projects on that host.
- Opening a remote project runs off the UI thread behind the loading card, so an unreachable host
  cannot block the window. The card's Cancel button cancels the task's context, which kills the
  `ssh` commands it waits on and closes the card at once; a result the cancelled task still
  produces is dropped, and a folder-picker connection made only for that attempt is released.
  The `ssh -N` forward is deliberately not tied to that context: a connection handed to the pool
  outlives the task, and only a dial still in progress is killed by a cancel. Closing the window
  cancels a task that is still running, since its result would update a window that no longer
  exists; the next window starts a fresh connection instead of inheriting its card.
- A restored terminal on the project on screen waits for the project's connection instead of
  showing Try Again; `installRemoteConn` clears the waiting state and the pane attaches. A failed
  reconnect puts the connection error behind Try Again.
- Missing or unreachable binaries surface as an error from `Provision`; the app reports it and keeps
  the project closed rather than editing a stale listing.

## Verification

```sh
go test ./internal/session
go test -race ./internal/session
MOJIAN_REMOTE_TEST_HOST=user@host go test ./internal/app -run TestRemoteServerSetupFedora -v -timeout 10m
```
