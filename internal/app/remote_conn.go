package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mojian/internal/remotebin"
	"mojian/internal/session"
)

type remoteInstall struct {
	arch       string
	dir        string
	binPath    string
	libPath    string
	runtimeDir string
	sock       string
}

// probeRemote reads the host's home, architecture and runtime directory,
// and makes sure the install and socket folders exist.
func (i *remoteInstall) uploadBinaries(host string) (bool, error) {
	bin, err := remotebin.Binary(i.arch)
	if err != nil {
		return false, err
	}
	lib, err := remotebin.Library(i.arch)
	if err != nil {
		return false, err
	}
	uploaded := false
	if !remoteHashEqual(host, i.binPath, bin) {
		if err := uploadRemote(host, i.binPath, bin, "755"); err != nil {
			return uploaded, fmt.Errorf("uploading the session server: %w", err)
		}
		uploaded = true
	}
	if !remoteHashEqual(host, i.libPath, lib) {
		if err := uploadRemote(host, i.libPath, lib, "644"); err != nil {
			return uploaded, fmt.Errorf("uploading libghostty-vt: %w", err)
		}
		uploaded = true
	}
	return uploaded, nil
}

// uploadAll replaces both files even when their hash matches, for a
// server that speaks another protocol.
func (i *remoteInstall) uploadAll(host string) error {
	bin, err := remotebin.Binary(i.arch)
	if err != nil {
		return err
	}
	lib, err := remotebin.Library(i.arch)
	if err != nil {
		return err
	}
	if err := uploadRemote(host, i.binPath, bin, "755"); err != nil {
		return fmt.Errorf("updating the session server: %w", err)
	}
	if err := uploadRemote(host, i.libPath, lib, "644"); err != nil {
		return fmt.Errorf("updating libghostty-vt: %w", err)
	}
	return nil
}

// start launches the server if it is not already running. The server
// outlives the app: its terminals and files stay usable after the SSH
// connection drops. Stop it with `mojian-server -stop`.
func (i *remoteInstall) start(host string) error {
	cmd := "MOJIAN_DIR=" + shellQuote(i.dir) + " MOJIAN_RUNTIME_DIR=" + shellQuote(i.runtimeDir) +
		" nohup " +
		shellQuote(i.binPath) + " >>" + shellQuote(i.dir+"/server.log") + " 2>&1 &"
	if _, err := sshOutput(host, cmd); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	return nil
}

// forward opens the local end of the SSH forward to the server socket and
// connects a control client through it.
func (i *remoteInstall) forward(host string) (*remoteConn, error) {
	local := filepath.Join(os.TempDir(), "mojian-"+shortHash(host)+".sock")
	_ = os.Remove(local)
	args := append([]string{"-N"}, sshCommon()...)
	args = append(args, "-o", "ExitOnForwardFailure=yes",
		"-L", local+":"+i.sock, "--", host)
	fwd := exec.Command("ssh", args...)
	var ferr bytes.Buffer
	fwd.Stderr = &ferr
	if err := fwd.Start(); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		conn, err := net.DialTimeout("unix", local, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			_ = fwd.Process.Kill()
			return nil, fmt.Errorf("the SSH forward failed: %v %s", err, strings.TrimSpace(ferr.String()))
		}
		time.Sleep(50 * time.Millisecond)
	}
	client, err := session.ConnectAt(local)
	if err != nil {
		// A server from another build cannot answer this protocol.
		var pe *session.ProtocolError
		if errors.As(err, &pe) {
			shutdownAt(local)
		}
		_ = fwd.Process.Kill()
		_, _ = fwd.Process.Wait()
		_ = os.Remove(local)
		return nil, err
	}
	return &remoteConn{host: host, socket: local, forward: fwd, client: client}, nil
}

// dialRemote prepares a connection without touching the app's active
// project, so a switch can run it away from the UI thread.

type remoteSetup struct {
	Uploaded bool
	Path     string
	Protocol int
}

// provisionRemote installs the server and checks that it answers. The
// connection is closed afterwards; the server keeps running for project
// and terminal use.

// sshOutput runs a command over SSH with the user's config and agent.
// sshCommon is the non-interactive SSH base. It deliberately leaves out
// ControlMaster: a stale master socket makes every later ssh wait on a
// process that is gone, and the commands here are few enough that a fresh
// connection is the reliable choice.
func sshCommon() []string {
	return []string{
		"-o", "BatchMode=yes", "-o", "ConnectTimeout=20",
		"-o", "ServerAliveInterval=15",
	}
}

func sshOutput(host, command string) (string, error) {
	args := append(append(sshCommon(), "--"), host, command)
	cmd := exec.Command("ssh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("ssh %s: %s", host, msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// connectRemote prepares a connection to host without replacing the
// current server, so a failed or cancelled switch leaves the open project
// talking to the machine it came from.
// connectRemote prepares a connection to host without replacing the
// current server, so a failed or cancelled switch leaves the open project
// talking to the machine it came from.
func (a *app) connectRemote(host string) (*remoteConn, error) {
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	if rc := a.remoteForHost(host); rc != nil {
		return rc, nil
	}
	rc, err := dialRemote(host)
	if err != nil {
		return nil, err
	}
	a.remotes[host] = rc
	return rc, nil
}

// remoteForHost returns the live connection the pool owns for host,
// dropping it first when it has gone stale.
// remoteForHost returns the live connection the pool owns for host,
// dropping it first when it has gone stale.
func (a *app) remoteForHost(host string) *remoteConn {
	a.initWorkspaces()
	rc := a.remotes[host]
	if rc == nil {
		return nil
	}
	if remoteHealthy(rc) {
		return rc
	}
	delete(a.remotes, host)
	closeRemoteConn(rc)
	return nil
}

// activeRemote is the connection of the project on screen, if it is a
// remote one. The pool owns it; nothing else keeps a copy.
// activeRemote is the connection of the project on screen, if it is a
// remote one. The pool owns it; nothing else keeps a copy.
func (a *app) activeRemote() *remoteConn {
	if a.serverAlias == "" {
		return nil
	}
	return a.remotes[a.serverAlias]
}

// remoteHealthy reports whether rc still has a live control client.
// remoteHealthy reports whether rc still has a live control client.
func remoteHealthy(rc *remoteConn) bool {
	if rc == nil || rc.client == nil {
		return false
	}
	select {
	case <-rc.client.Closed():
		return false
	default:
		return true
	}
}

// remoteInstall is where a host keeps the server this build needs. It is
// probed over SSH before the binaries are compared and uploaded.
// probeRemote reads the host's home, architecture and runtime directory,
// and makes sure the install and socket folders exist.
func probeRemote(host string) (*remoteInstall, error) {
	home, err := sshOutput(host, `printf %s "$HOME"`)
	if err != nil {
		return nil, err
	}
	arch, err := sshOutput(host, "uname -m")
	if err != nil {
		return nil, err
	}
	dir := home + "/.local/share/mojian"
	// A socket under the home directory is not always reachable through
	// sshd's stream-local forwarding (SELinux labels it data_home_t on
	// Fedora). Put the socket in the user's runtime directory instead,
	// and tell the server where with MOJIAN_RUNTIME_DIR.
	uid := -1
	if out, err := sshOutput(host, "id -u"); err == nil {
		uid, _ = strconv.Atoi(strings.TrimSpace(out))
	}
	runtimeDir, err := sshOutput(host, `printf %s "${XDG_RUNTIME_DIR:-/tmp}/mojian-$(id -u)"`)
	if err != nil || strings.TrimSpace(runtimeDir) == "" {
		runtimeDir = "/tmp/mojian-" + strconv.Itoa(uid)
	}
	runtimeDir = strings.TrimSpace(runtimeDir)
	if _, err := sshOutput(host, "mkdir -p "+shellQuote(dir)+" "+shellQuote(runtimeDir)+
		" && chmod 700 "+shellQuote(dir)+" "+shellQuote(runtimeDir)); err != nil {
		return nil, err
	}
	return &remoteInstall{
		arch:       arch,
		dir:        dir,
		binPath:    dir + "/mojian-server",
		libPath:    dir + "/libghostty-vt.so",
		runtimeDir: runtimeDir,
		sock:       session.SocketPathIn(runtimeDir, uid),
	}, nil
}

// uploadBinaries puts the server and the viewer library in place when the
// files on the host are not already this build. It reports whether it
// uploaded anything.
// dialRemote prepares a connection without touching the app's active
// project, so a switch can run it away from the UI thread.
func dialRemote(host string) (*remoteConn, error) {
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	info, err := probeRemote(host)
	if err != nil {
		return nil, err
	}
	if _, err := info.uploadBinaries(host); err != nil {
		return nil, err
	}
	if err := info.start(host); err != nil {
		return nil, err
	}
	rc, err := info.forward(host)
	if err == nil {
		return rc, nil
	}
	var pe *session.ProtocolError
	if !errors.As(err, &pe) {
		return nil, err
	}
	// The running server was another protocol. Stop it through its own
	// command, wait for its socket and lock to go, then put this build in
	// place and start it before trying the forward again.
	if stopErr := info.stop(host); stopErr != nil {
		return nil, stopErr
	}
	info.waitStopped(host)
	if err := info.uploadAll(host); err != nil {
		return nil, err
	}
	if err := info.start(host); err != nil {
		return nil, err
	}
	return info.forward(host)
}

// remoteSetup is what provisioning a host found: whether this run
// uploaded the binaries and where the server lives.
// provisionRemote installs the server and checks that it answers. The
// connection is closed afterwards; the server keeps running for project
// and terminal use.
func provisionRemote(host string, existing *remoteConn) (*remoteSetup, error) {
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	connected := false
	if remoteHealthy(existing) {
		if hello, err := existing.client.Hello(); err == nil && hello.Version == session.ProtocolVersion {
			connected = true
		}
	}
	info, err := probeRemote(host)
	if err != nil {
		return nil, err
	}
	uploaded, err := info.uploadBinaries(host)
	if err != nil {
		return nil, err
	}
	if connected {
		// The binaries are current; the running server keeps its image
		// until it next starts.
		return &remoteSetup{Uploaded: uploaded, Path: info.binPath, Protocol: session.ProtocolVersion}, nil
	}
	// dialRemote also upgrades a running server that speaks another
	// protocol: it shuts it down, uploads this build and starts again.
	rc, err := dialRemote(host)
	if err != nil {
		return nil, err
	}
	hello, helloErr := rc.client.Hello()
	closeRemoteConn(rc)
	if helloErr != nil {
		return nil, helloErr
	}
	if hello.Version != session.ProtocolVersion {
		return nil, &session.ProtocolError{Server: hello.Version, Client: session.ProtocolVersion}
	}
	return &remoteSetup{Uploaded: uploaded, Path: info.binPath, Protocol: hello.Version}, nil
}

// stop asks the running server to exit, through its own -stop command.
func (i *remoteInstall) stop(host string) error {
	_, err := sshOutput(host, "MOJIAN_DIR="+shellQuote(i.dir)+
		" MOJIAN_RUNTIME_DIR="+shellQuote(i.runtimeDir)+
		" "+shellQuote(i.binPath)+" -stop")
	return err
}

// waitStopped waits for a remote server socket to disappear after its
// shutdown request, so the next process can take the lock.
func (i *remoteInstall) waitStopped(host string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := sshOutput(host, "test -S "+shellQuote(i.sock)); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// shutdownAt asks a server at a local socket to stop without a client.
func shutdownAt(socket string) {
	conn, err := net.DialTimeout("unix", socket, time.Second)
	if err != nil {
		return
	}
	_, _ = conn.Write([]byte(`{"id":1,"op":"shutdown"}` + "\n"))
	_ = conn.Close()
}

// remoteHashEqual reports whether the file on host already has data.
// remoteHashEqual reports whether the file on host already has data.
func remoteHashEqual(host, path string, data []byte) bool {
	want := sha256.Sum256(data)
	sum, err := sshOutput(host, "if command -v sha256sum >/dev/null 2>&1; then sha256sum -- "+shellQuote(path)+
		"; else shasum -a 256 -- "+shellQuote(path)+"; fi")
	if err != nil {
		return false
	}
	fields := strings.Fields(sum)
	return len(fields) > 0 && strings.EqualFold(fields[0], hex.EncodeToString(want[:]))
}

// uploadRemote writes data to path on host through SSH, atomically.
// uploadRemote writes data to path on host through SSH, atomically.
func uploadRemote(host, path string, data []byte, mode string) error {
	tmp := path + ".tmp"
	cmd := exec.Command("ssh", append(sshCommon(), "--", host,
		"mkdir -p "+shellQuote(filepath.Dir(path))+" && cat > "+shellQuote(tmp)+
			" && chmod "+mode+" "+shellQuote(tmp)+" && mv "+shellQuote(tmp)+" "+shellQuote(path))...)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// serverAssetCache is where pictures and PDFs of a server project are
// fetched for their viewers.
// serverAssetCache is where pictures and PDFs of a server project are
// fetched for their viewers.
func (a *app) serverAssetCache(serverID, root string) string {
	base := a.dataDir
	if base == "" {
		base, _ = os.UserCacheDir()
	}
	return filepath.Join(base, "server-assets", shortHash(serverID+"\x00"+root))
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

func (a *app) detachRemoteConn(rc *remoteConn) {
	if rc == nil {
		return
	}
	if a.remotes != nil && a.remotes[rc.host] == rc {
		delete(a.remotes, rc.host)
	}
	closeRemoteConn(rc)
}

// closeRemoteConn closes the control client and the SSH forward without
// stopping the server or its terminals on the host.
// closeRemoteConn closes the control client and the SSH forward without
// stopping the server or its terminals on the host.
func closeRemoteConn(rc *remoteConn) {
	if rc == nil {
		return
	}
	if rc.client != nil {
		// Detach only: the server and its terminals keep running for the
		// next connection.
		_ = rc.client.Close()
	}
	if rc.forward != nil && rc.forward.Process != nil {
		_ = rc.forward.Process.Kill()
		_, _ = rc.forward.Process.Wait()
	}
	_ = os.Remove(rc.socket)
}

// workspaceReachable reports whether a cached remote workspace still has
// a live connection. A local workspace always does.
// remoteConn is this app's bridge to one remote session server: an SSH
// forward to its Unix socket, and the control connection through it.
type remoteConn struct {
	host    string
	socket  string
	forward *exec.Cmd
	client  *session.Client
}
