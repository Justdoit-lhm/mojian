// Package remote provisions Mojian's session server on an SSH host and
// keeps the forwarded control connection to it. The UI and the pool stay
// in internal/app.
package remote

import (
	"bytes"
	"context"
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

	"mojian/internal/fs"
	"mojian/internal/remotebin"
	"mojian/internal/session"
)

// Install is where a host keeps the server this build needs. It is
// probed over SSH before the binaries are compared and uploaded.
type Install struct {
	home       string
	arch       string
	dir        string
	binPath    string
	libPath    string
	runtimeDir string
	sock       string
}

// upload puts the server and the viewer library in place, skipping a file
// whose hash matches unless force is set: force replaces both even when
// they match, for a server that speaks another protocol. It reports
// whether it uploaded anything.
func (i *Install) upload(ctx context.Context, host string, force bool) (bool, error) {
	bin, err := remotebin.Binary(i.arch)
	if err != nil {
		return false, err
	}
	lib, err := remotebin.Library(i.arch)
	if err != nil {
		return false, err
	}
	files := []struct {
		path, mode, label string
		data              []byte
	}{
		{i.binPath, "755", "the session server", bin},
		{i.libPath, "644", "libghostty-vt", lib},
	}
	uploaded := false
	for _, f := range files {
		if !force && remoteHashEqual(ctx, host, f.path, f.data) {
			continue
		}
		if err := uploadRemote(ctx, host, f.path, f.data, f.mode); err != nil {
			return uploaded, fmt.Errorf("uploading %s: %w", f.label, err)
		}
		uploaded = true
	}
	return uploaded, nil
}

// start launches the server if it is not already running. The server
// outlives the app: its terminals and files stay usable after the SSH
// connection drops. Stop it with `mojian-server -stop`.
func (i *Install) start(ctx context.Context, host string) error {
	cmd := "MOJIAN_DIR=" + shellQuote(i.dir) + " MOJIAN_RUNTIME_DIR=" + shellQuote(i.runtimeDir) +
		" nohup " +
		shellQuote(i.binPath) + " >>" + shellQuote(i.dir+"/server.log") + " 2>&1 &"
	if _, err := SSHOutputContext(ctx, host, cmd); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(250 * time.Millisecond):
	}
	return nil
}

// forward opens the local end of the SSH forward to the server socket and
// connects a control client through it.
func (i *Install) forward(ctx context.Context, host string) (*Conn, error) {
	local := filepath.Join(os.TempDir(), "mojian-"+fs.ShortHash(host)+".sock")
	_ = os.Remove(local)
	args := append([]string{"-N"}, sshCommon()...)
	args = append(args, "-o", "ExitOnForwardFailure=yes",
		"-L", local+":"+i.sock, "--", host)
	fwd := exec.CommandContext(ctx, "ssh", args...)
	var ferr bytes.Buffer
	fwd.Stderr = &ferr
	if err := fwd.Start(); err != nil {
		return nil, err
	}
	fail := func(err error) (*Conn, error) {
		_ = fwd.Process.Kill()
		_, _ = fwd.Process.Wait()
		_ = os.Remove(local)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		conn, err := net.DialTimeout("unix", local, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			return fail(fmt.Errorf("the SSH forward failed: %v %s", err, strings.TrimSpace(ferr.String())))
		}
		select {
		case <-ctx.Done():
			return fail(ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
	client, err := session.ConnectAt(local)
	if err != nil {
		// A server from another build cannot answer this protocol.
		var pe *session.ProtocolError
		if errors.As(err, &pe) {
			shutdownAt(local)
		}
		return fail(err)
	}
	return &Conn{Host: host, Socket: local, forward: fwd, Client: client}, nil
}

// Setup is what provisioning a host found: whether this run
// uploaded the binaries and where the server lives.
type Setup struct {
	Uploaded bool
	Path     string
	Protocol int
	// Home is the host's home directory, where its folder picker starts.
	Home string
	// Conn is the live connection provisioning ended with. The caller owns
	// it and must Close it if it is not used.
	Conn *Conn
}

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

// SSHOutput runs a command over SSH with the user's config and agent.
func SSHOutput(host, command string) (string, error) {
	return SSHOutputContext(context.Background(), host, command)
}

// SSHOutputContext is SSHOutput that stops the command when ctx is done.
func SSHOutputContext(ctx context.Context, host, command string) (string, error) {
	args := append(append(sshCommon(), "--"), host, command)
	cmd := exec.CommandContext(ctx, "ssh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
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

// probe reads the host's home, architecture and runtime directory,
// and makes sure the install and socket folders exist.
func probe(ctx context.Context, host string) (*Install, error) {
	home, err := SSHOutputContext(ctx, host, `printf %s "$HOME"`)
	if err != nil {
		return nil, err
	}
	arch, err := SSHOutputContext(ctx, host, "uname -m")
	if err != nil {
		return nil, err
	}
	dir := home + "/.local/share/mojian"
	// A socket under the home directory is not always reachable through
	// sshd's stream-local forwarding (SELinux labels it data_home_t on
	// Fedora). Put the socket in the user's runtime directory instead,
	// and tell the server where with MOJIAN_RUNTIME_DIR.
	uid := -1
	if out, err := SSHOutputContext(ctx, host, "id -u"); err == nil {
		uid, _ = strconv.Atoi(strings.TrimSpace(out))
	}
	runtimeDir, err := SSHOutputContext(ctx, host, `printf %s "${XDG_RUNTIME_DIR:-/tmp}/mojian-$(id -u)"`)
	if err != nil || strings.TrimSpace(runtimeDir) == "" {
		runtimeDir = "/tmp/mojian-" + strconv.Itoa(uid)
	}
	runtimeDir = strings.TrimSpace(runtimeDir)
	if _, err := SSHOutputContext(ctx, host, "mkdir -p "+shellQuote(dir)+" "+shellQuote(runtimeDir)+
		" && chmod 700 "+shellQuote(dir)+" "+shellQuote(runtimeDir)); err != nil {
		return nil, err
	}
	return &Install{
		home:       home,
		arch:       arch,
		dir:        dir,
		binPath:    dir + "/mojian-server",
		libPath:    dir + "/libghostty-vt.so",
		runtimeDir: runtimeDir,
		sock:       session.SocketPathIn(runtimeDir, uid),
	}, nil
}

// Dial prepares a connection without touching the app's active
// project, so a switch can run it away from the UI thread.
func Dial(host string) (*Conn, error) {
	return DialContext(context.Background(), host)
}

// DialContext is Dial that stops at the next step when ctx is done.
func DialContext(ctx context.Context, host string) (*Conn, error) {
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := probe(ctx, host)
	if err != nil {
		return nil, err
	}
	if _, err := info.upload(ctx, host, false); err != nil {
		return nil, err
	}
	if err := info.start(ctx, host); err != nil {
		return nil, err
	}
	rc, err := info.forward(ctx, host)
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
	if stopErr := info.stop(ctx, host); stopErr != nil {
		return nil, stopErr
	}
	info.waitStopped(ctx, host)
	if _, err := info.upload(ctx, host, true); err != nil {
		return nil, err
	}
	if err := info.start(ctx, host); err != nil {
		return nil, err
	}
	return info.forward(ctx, host)
}

// Provision installs the server and checks that it answers. The returned
// connection is live and belongs to the caller; the server keeps running
// for project and terminal use.
func Provision(host string, existing *Conn) (*Setup, error) {
	return ProvisionContext(context.Background(), host, existing)
}

// ProvisionContext is Provision that stops at the next step when ctx is
// done.
func ProvisionContext(ctx context.Context, host string, existing *Conn) (*Setup, error) {
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	connected := false
	if existing.Healthy() {
		if hello, err := existing.Client.Hello(); err == nil && hello.Version == session.ProtocolVersion {
			connected = true
		}
	}
	info, err := probe(ctx, host)
	if err != nil {
		return nil, err
	}
	uploaded, err := info.upload(ctx, host, false)
	if err != nil {
		return nil, err
	}
	if connected {
		// The binaries are current; the running server keeps its image
		// until it next starts.
		return &Setup{Uploaded: uploaded, Path: info.binPath, Protocol: session.ProtocolVersion, Home: info.home, Conn: existing}, nil
	}
	// Dial also upgrades a running server that speaks another
	// protocol: it shuts it down, uploads this build and starts again.
	rc, err := DialContext(ctx, host)
	if err != nil {
		return nil, err
	}
	hello, helloErr := rc.Client.Hello()
	if helloErr != nil {
		Close(rc)
		return nil, helloErr
	}
	if hello.Version != session.ProtocolVersion {
		Close(rc)
		return nil, &session.ProtocolError{Server: hello.Version, Client: session.ProtocolVersion}
	}
	return &Setup{Uploaded: uploaded, Path: info.binPath, Protocol: hello.Version, Home: info.home, Conn: rc}, nil
}

// StopServer asks the server on host to exit. Its terminals and the
// processes inside them end; the next Dial or Provision starts a fresh
// server.
func StopServer(host string) error {
	return StopServerContext(context.Background(), host)
}

// StopServerContext is StopServer that stops waiting when ctx is done.
func StopServerContext(ctx context.Context, host string) error {
	if host == "" {
		return fmt.Errorf("no server selected")
	}
	info, err := probe(ctx, host)
	if err != nil {
		return err
	}
	if err := info.stop(ctx, host); err != nil {
		return err
	}
	info.waitStopped(ctx, host)
	return nil
}

// ServerRunning reports whether the server on host still has its socket.
// A host that cannot be reached is reported as not running: the next
// Start Server or Dial surfaces the SSH error.
func ServerRunning(ctx context.Context, host string) (bool, error) {
	if host == "" {
		return false, fmt.Errorf("no server selected")
	}
	info, err := probe(ctx, host)
	if err != nil {
		return false, err
	}
	if _, err := SSHOutputContext(ctx, host, "test -S "+shellQuote(info.sock)); err != nil {
		return false, nil
	}
	return true, nil
}

// stop asks the running server to exit, through its own -stop command.
func (i *Install) stop(ctx context.Context, host string) error {
	_, err := SSHOutputContext(ctx, host, "MOJIAN_DIR="+shellQuote(i.dir)+
		" MOJIAN_RUNTIME_DIR="+shellQuote(i.runtimeDir)+
		" "+shellQuote(i.binPath)+" -stop")
	return err
}

// waitStopped waits for a remote server socket to disappear after its
// shutdown request, so the next process can take the lock.
func (i *Install) waitStopped(ctx context.Context, host string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return
		}
		if _, err := SSHOutputContext(ctx, host, "test -S "+shellQuote(i.sock)); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
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
func remoteHashEqual(ctx context.Context, host, path string, data []byte) bool {
	want := sha256.Sum256(data)
	sum, err := SSHOutputContext(ctx, host, "if command -v sha256sum >/dev/null 2>&1; then sha256sum -- "+shellQuote(path)+
		"; else shasum -a 256 -- "+shellQuote(path)+"; fi")
	if err != nil {
		return false
	}
	fields := strings.Fields(sum)
	return len(fields) > 0 && strings.EqualFold(fields[0], hex.EncodeToString(want[:]))
}

// uploadRemote writes data to path on host through SSH, atomically.
func uploadRemote(ctx context.Context, host, path string, data []byte, mode string) error {
	tmp := path + ".tmp"
	cmd := exec.CommandContext(ctx, "ssh", append(sshCommon(), "--", host,
		"mkdir -p "+shellQuote(filepath.Dir(path))+" && cat > "+shellQuote(tmp)+
			" && chmod "+mode+" "+shellQuote(tmp)+" && mv "+shellQuote(tmp)+" "+shellQuote(path))...)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("%v %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Close closes the control client and the SSH forward without
// stopping the server or its terminals on the host.
func Close(rc *Conn) {
	if rc == nil {
		return
	}
	if rc.Client != nil {
		// Detach only: the server and its terminals keep running for the
		// next connection.
		_ = rc.Client.Close()
	}
	if rc.forward != nil && rc.forward.Process != nil {
		_ = rc.forward.Process.Kill()
		_, _ = rc.forward.Process.Wait()
	}
	_ = os.Remove(rc.Socket)
}

// Conn is a bridge to one remote session server: an SSH forward to its
// Unix socket, and the control connection through it.
type Conn struct {
	Host    string
	Socket  string
	forward *exec.Cmd
	Client  *session.Client
}

// Healthy reports whether the connection still has a live control client.
func (c *Conn) Healthy() bool {
	if c == nil || c.Client == nil {
		return false
	}
	select {
	case <-c.Client.Closed():
		return false
	default:
		return true
	}
}
