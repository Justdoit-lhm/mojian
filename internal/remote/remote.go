// Package remote provisions Mojian's session server on an SSH host and
// keeps the forwarded control connection to it. The UI and the pool stay
// in internal/app.
package remote

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

	"mojian/internal/fs"
	"mojian/internal/remotebin"
	"mojian/internal/session"
)

// Install is where a host keeps the server this build needs. It is
// probed over SSH before the binaries are compared and uploaded.
type Install struct {
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
func (i *Install) upload(host string, force bool) (bool, error) {
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
		if !force && remoteHashEqual(host, f.path, f.data) {
			continue
		}
		if err := uploadRemote(host, f.path, f.data, f.mode); err != nil {
			return uploaded, fmt.Errorf("uploading %s: %w", f.label, err)
		}
		uploaded = true
	}
	return uploaded, nil
}

// start launches the server if it is not already running. The server
// outlives the app: its terminals and files stay usable after the SSH
// connection drops. Stop it with `mojian-server -stop`.
func (i *Install) start(host string) error {
	cmd := "MOJIAN_DIR=" + shellQuote(i.dir) + " MOJIAN_RUNTIME_DIR=" + shellQuote(i.runtimeDir) +
		" nohup " +
		shellQuote(i.binPath) + " >>" + shellQuote(i.dir+"/server.log") + " 2>&1 &"
	if _, err := SSHOutput(host, cmd); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	return nil
}

// forward opens the local end of the SSH forward to the server socket and
// connects a control client through it.
func (i *Install) forward(host string) (*Conn, error) {
	local := filepath.Join(os.TempDir(), "mojian-"+fs.ShortHash(host)+".sock")
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
	return &Conn{Host: host, Socket: local, forward: fwd, Client: client}, nil
}

// Setup is what provisioning a host found: whether this run
// uploaded the binaries and where the server lives.
type Setup struct {
	Uploaded bool
	Path     string
	Protocol int
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

// probe reads the host's home, architecture and runtime directory,
// and makes sure the install and socket folders exist.
func probe(host string) (*Install, error) {
	home, err := SSHOutput(host, `printf %s "$HOME"`)
	if err != nil {
		return nil, err
	}
	arch, err := SSHOutput(host, "uname -m")
	if err != nil {
		return nil, err
	}
	dir := home + "/.local/share/mojian"
	// A socket under the home directory is not always reachable through
	// sshd's stream-local forwarding (SELinux labels it data_home_t on
	// Fedora). Put the socket in the user's runtime directory instead,
	// and tell the server where with MOJIAN_RUNTIME_DIR.
	uid := -1
	if out, err := SSHOutput(host, "id -u"); err == nil {
		uid, _ = strconv.Atoi(strings.TrimSpace(out))
	}
	runtimeDir, err := SSHOutput(host, `printf %s "${XDG_RUNTIME_DIR:-/tmp}/mojian-$(id -u)"`)
	if err != nil || strings.TrimSpace(runtimeDir) == "" {
		runtimeDir = "/tmp/mojian-" + strconv.Itoa(uid)
	}
	runtimeDir = strings.TrimSpace(runtimeDir)
	if _, err := SSHOutput(host, "mkdir -p "+shellQuote(dir)+" "+shellQuote(runtimeDir)+
		" && chmod 700 "+shellQuote(dir)+" "+shellQuote(runtimeDir)); err != nil {
		return nil, err
	}
	return &Install{
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
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	info, err := probe(host)
	if err != nil {
		return nil, err
	}
	if _, err := info.upload(host, false); err != nil {
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
	if _, err := info.upload(host, true); err != nil {
		return nil, err
	}
	if err := info.start(host); err != nil {
		return nil, err
	}
	return info.forward(host)
}

// Provision installs the server and checks that it answers. The
// connection is closed afterwards; the server keeps running for project
// and terminal use.
func Provision(host string, existing *Conn) (*Setup, error) {
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	connected := false
	if existing.Healthy() {
		if hello, err := existing.Client.Hello(); err == nil && hello.Version == session.ProtocolVersion {
			connected = true
		}
	}
	info, err := probe(host)
	if err != nil {
		return nil, err
	}
	uploaded, err := info.upload(host, false)
	if err != nil {
		return nil, err
	}
	if connected {
		// The binaries are current; the running server keeps its image
		// until it next starts.
		return &Setup{Uploaded: uploaded, Path: info.binPath, Protocol: session.ProtocolVersion}, nil
	}
	// Dial also upgrades a running server that speaks another
	// protocol: it shuts it down, uploads this build and starts again.
	rc, err := Dial(host)
	if err != nil {
		return nil, err
	}
	hello, helloErr := rc.Client.Hello()
	Close(rc)
	if helloErr != nil {
		return nil, helloErr
	}
	if hello.Version != session.ProtocolVersion {
		return nil, &session.ProtocolError{Server: hello.Version, Client: session.ProtocolVersion}
	}
	return &Setup{Uploaded: uploaded, Path: info.binPath, Protocol: hello.Version}, nil
}

// stop asks the running server to exit, through its own -stop command.
func (i *Install) stop(host string) error {
	_, err := SSHOutput(host, "MOJIAN_DIR="+shellQuote(i.dir)+
		" MOJIAN_RUNTIME_DIR="+shellQuote(i.runtimeDir)+
		" "+shellQuote(i.binPath)+" -stop")
	return err
}

// waitStopped waits for a remote server socket to disappear after its
// shutdown request, so the next process can take the lock.
func (i *Install) waitStopped(host string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := SSHOutput(host, "test -S "+shellQuote(i.sock)); err != nil {
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
func remoteHashEqual(host, path string, data []byte) bool {
	want := sha256.Sum256(data)
	sum, err := SSHOutput(host, "if command -v sha256sum >/dev/null 2>&1; then sha256sum -- "+shellQuote(path)+
		"; else shasum -a 256 -- "+shellQuote(path)+"; fi")
	if err != nil {
		return false
	}
	fields := strings.Fields(sum)
	return len(fields) > 0 && strings.EqualFold(fields[0], hex.EncodeToString(want[:]))
}

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
