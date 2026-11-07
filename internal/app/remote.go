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

	"mojian/internal/icons"
	"mojian/internal/remotebin"
	"mojian/internal/session"

	"github.com/egoist/mygo/ui"
)

// remoteProject is a project on an SSH host, remembered by the switcher.
type remoteProject struct {
	Host string `json:"host"`
	Path string `json:"path"`
}

// remoteConn is this app's bridge to one remote session server: an SSH
// forward to its Unix socket, and the control connection through it.
type remoteConn struct {
	host    string
	socket  string
	forward *exec.Cmd
	client  *session.Client
}

func (a *app) remoteName() string {
	if a.serverAlias == "" {
		return "Local"
	}
	return a.hostLabel(a.serverAlias)
}

// sshOutput runs a command over SSH with the user's config and agent.
func sshCommon() []string {
	return []string{
		"-o", "BatchMode=yes", "-o", "ConnectTimeout=20",
		"-o", "ServerAliveInterval=15",
		"-o", "ControlMaster=auto", "-o", "ControlPath=/tmp/mojian-ssh-%C",
		"-o", "ControlPersist=60",
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

// ensureRemote starts the server on host when needed, forwards its Unix
// socket over SSH, connects the control client, and makes it the open
// server.
func (a *app) ensureRemote(host string) (*remoteConn, error) {
	rc, err := a.connectRemote(host)
	if err != nil {
		return nil, err
	}
	a.adoptRemote(rc)
	return rc, nil
}

// connectRemote prepares a connection to host without replacing the
// current server, so a failed or cancelled switch leaves the open project
// talking to the machine it came from.
func (a *app) connectRemote(host string) (*remoteConn, error) {
	if a.remote != nil && a.remote.host == host {
		if c := a.remote.client; c != nil {
			select {
			case <-c.Closed():
				// The connection dropped; replace it below rather than
				// handing back a dead client.
			default:
				return a.remote, nil
			}
		}
		// A stale forward for this host would own the local socket path
		// the new one needs.
		stale := a.remote
		a.remote = nil
		a.closeRemoteConn(stale)
	}
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	home, err := sshOutput(host, `printf %s "$HOME"`)
	if err != nil {
		return nil, err
	}
	arch, err := sshOutput(host, "uname -m")
	if err != nil {
		return nil, err
	}
	bin, err := remotebin.Binary(arch)
	if err != nil {
		return nil, err
	}
	lib, err := remotebin.Library(arch)
	if err != nil {
		return nil, err
	}
	dir := home + "/.local/share/mojian"
	binPath := dir + "/mojian-server"
	libPath := dir + "/libghostty-vt.so"
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
	sock := session.SocketPathIn(runtimeDir, uid)
	if _, err := sshOutput(host, "mkdir -p "+shellQuote(dir)+" "+shellQuote(runtimeDir)+
		" && chmod 700 "+shellQuote(dir)+" "+shellQuote(runtimeDir)); err != nil {
		return nil, err
	}
	upload := func(path string, data []byte, mode string) error {
		return uploadRemote(host, path, data, mode)
	}
	// A file whose hash matches is already the build this app needs.
	if !remoteHashEqual(host, binPath, bin) {
		if err := upload(binPath, bin, "755"); err != nil {
			return nil, fmt.Errorf("uploading the session server: %w", err)
		}
	}
	if !remoteHashEqual(host, libPath, lib) {
		if err := upload(libPath, lib, "644"); err != nil {
			return nil, fmt.Errorf("uploading libghostty-vt: %w", err)
		}
	}
	start := func() error {
		// A transient server ends with the app connection, taking its
		// terminals with it; remote shells are not kept after a drop.
		cmd := "MOJIAN_DIR=" + shellQuote(dir) + " MOJIAN_RUNTIME_DIR=" + shellQuote(runtimeDir) +
			" MOJIAN_TRANSIENT=1 nohup " +
			shellQuote(binPath) + " >>" + shellQuote(dir+"/server.log") + " 2>&1 &"
		if _, err := sshOutput(host, cmd); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		return nil
	}
	open := func() (*remoteConn, string, error) {
		local := filepath.Join(os.TempDir(), "mojian-"+shortHash(host)+".sock")
		_ = os.Remove(local)
		fwd := exec.Command("ssh", "-N", "-o", "BatchMode=yes", "-o", "ConnectTimeout=20",
			"-o", "ExitOnForwardFailure=yes", "-o", "ServerAliveInterval=15",
			"-L", local+":"+sock, "--", host)
		var ferr bytes.Buffer
		fwd.Stderr = &ferr
		if err := fwd.Start(); err != nil {
			return nil, local, err
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
				return nil, local, fmt.Errorf("the SSH forward failed: %v %s", err, strings.TrimSpace(ferr.String()))
			}
			time.Sleep(50 * time.Millisecond)
		}
		client, err := session.ConnectAt(local)
		if err != nil {
			// A server from another build cannot answer this protocol.
			// End it through its socket, then upload and start the current
			// one before one more try.
			var pe *session.ProtocolError
			if errors.As(err, &pe) {
				shutdownAt(local)
				_ = fwd.Process.Kill()
				_, _ = fwd.Process.Wait()
				_ = os.Remove(local)
				return nil, local, pe
			}
			_ = fwd.Process.Kill()
			_, _ = fwd.Process.Wait()
			_ = os.Remove(local)
			return nil, local, err
		}
		return &remoteConn{host: host, socket: local, forward: fwd, client: client}, local, nil
	}
	if err := start(); err != nil {
		return nil, err
	}
	rc, _, err := open()
	if err == nil {
		return rc, nil
	}
	var pe *session.ProtocolError
	if !errors.As(err, &pe) {
		return nil, err
	}
	// The running server was another protocol. Its socket is gone; put
	// this build in place and start it before trying the forward again.
	if err := upload(binPath, bin, "755"); err != nil {
		return nil, fmt.Errorf("updating the session server: %w", err)
	}
	if err := upload(libPath, lib, "644"); err != nil {
		return nil, fmt.Errorf("updating libghostty-vt: %w", err)
	}
	if err := start(); err != nil {
		return nil, err
	}
	rc, _, err = open()
	return rc, err
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
	sum, err := sshOutput(host, "if command -v sha256sum >/dev/null 2>&1; then sha256sum -- "+shellQuote(path)+
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

// remoteAssetCache is where pictures and PDFs of a remote project are
// fetched for their viewers.
func (a *app) remoteAssetCache(host, remotePath string) string {
	base := a.dataDir
	if base == "" {
		base, _ = os.UserCacheDir()
	}
	return filepath.Join(base, "remote-assets", shortHash(host+"\x00"+remotePath))
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

// adoptRemote makes rc the open server and closes the one it replaces.
func (a *app) adoptRemote(rc *remoteConn) {
	if rc == nil || a.remote == rc {
		return
	}
	old := a.remote
	a.remote = rc
	if old != nil {
		a.closeRemoteConn(old)
	}
}

func (a *app) closeRemote() {
	a.dropPickerRemote()
	rc := a.remote
	a.remote = nil
	a.closeRemoteConn(rc)
}

func (a *app) closeRemoteConn(rc *remoteConn) {
	if rc == nil {
		return
	}
	if rc.client != nil {
		// The remote server is transient: ending it stops its shells now
		// instead of after the idle grace period.
		_ = rc.client.Shutdown()
		_ = rc.client.Close()
	}
	if rc.forward != nil && rc.forward.Process != nil {
		_ = rc.forward.Process.Kill()
		_, _ = rc.forward.Process.Wait()
	}
	_ = os.Remove(rc.socket)
}

// openRemoteProject connects to host and opens path there as the project.
// Files live on the host: the tree lists them and a file is read when it
// opens, with no local copy of the project.
func (a *app) openRemoteProject(host, remotePath string) error {
	if !terminalsSupported() {
		return fmt.Errorf("remote projects need macOS or Linux")
	}
	if host == "" || remotePath == "" {
		return fmt.Errorf("choose a server and a project")
	}
	remotePath = strings.TrimRight(remotePath, "/")
	if err := a.saveNow(); err != nil {
		return fmt.Errorf("could not save the open project: %w", err)
	}
	if a.layoutPath != "" {
		a.saveLayout()
	}
	a.dropPickerRemote()
	rc, err := a.connectRemote(host)
	if err != nil {
		return err
	}
	newConn := rc != a.remote
	fail := func(err error) error {
		if newConn {
			a.closeRemoteConn(rc)
		}
		return err
	}
	s, err := openRemoteStore(rc.client, remotePath, a.remoteAssetCache(host, remotePath))
	if err != nil {
		return fail(err)
	}
	// Detach the old project's terminals while their server is still up,
	// then make the new connection the open one.
	a.closeTerminals()
	a.adoptRemote(rc)
	a.git.Open = false
	a.git.Reset()
	a.fileClip = nil
	a.fileClipRemote = false
	a.fileCut = false
	a.dropPickerRemote()
	a.store = s
	a.treeOpen = true
	a.serverAlias = host
	a.remotePath = remotePath
	a.layoutPath = a.projectLayoutPath("remote:" + host + ":" + remotePath)
	a.rememberRemoteProject(remoteProject{Host: host, Path: remotePath})
	a.tabs = nil
	a.active = -1
	a.focusReq = nil
	a.treeSel = ""
	a.treeQuery = ""
	a.treeSearch.CancelFocus()
	a.treeReveal = ""
	a.openDirs = map[string]bool{}
	a.tabScroll = ui.ScrollState{}
	a.treeDraft = nil
	a.rename = nil
	a.paletteOpen = false
	a.projectOpen = false
	a.remotePicker = false
	a.err = ""
	if !a.restoreLayout() {
		a.focusFirstNote()
	}
	if t := a.tab(); t != nil {
		a.focusReq = t.editorPane()
	}
	a.revealTab = true
	a.saveLayout()
	a.saveSettings()
	a.updateTitle()
	a.invalidate()
	return nil
}

func (a *app) rememberRemoteProject(p remoteProject) {
	if p.Host == "" || p.Path == "" {
		return
	}
	out := []remoteProject{p}
	for _, q := range a.remoteProjects {
		if q.Host == p.Host && q.Path == p.Path {
			continue
		}
		out = append(out, q)
	}
	if len(out) > 12 {
		out = out[:12]
	}
	a.remoteProjects = out
}

// openRemotePicker asks the selected server for its home, then shows the
// directories to choose a project from.
func (a *app) openRemotePicker(host string) {
	if err := a.saveNow(); err != nil {
		a.fail(fmt.Errorf("could not save the open project: %w", err))
		return
	}
	a.dropPickerRemote()
	rc, err := a.connectRemote(host)
	if err != nil {
		a.fail(err)
		return
	}
	if rc != a.remote {
		a.remotePick = rc
	}
	homeDir, err := sshOutput(host, `printf %s "$HOME"`)
	if err != nil {
		a.dropPickerRemote()
		a.fail(err)
		return
	}
	a.remoteHost = host
	a.remoteQuery = homeDir
	a.remoteDirs, err = rc.client.ListDirs(homeDir)
	if err != nil {
		a.dropPickerRemote()
		a.fail(err)
		return
	}
	a.remotePicker = true
	a.serverOpen = false
	a.invalidate()
}

// pickerRemote is the server whose folders the project picker shows.
func (a *app) pickerRemote() *remoteConn {
	if a.remotePick != nil {
		return a.remotePick
	}
	return a.remote
}

// dropPickerRemote closes a connection made only for the picker. The
// open project's server is left alone.
func (a *app) dropPickerRemote() {
	a.remotePicker = false
	rc := a.remotePick
	a.remotePick = nil
	if rc != nil && rc != a.remote {
		a.closeRemoteConn(rc)
	}
}

func (a *app) remotePickerModal(c *ui.Context, k *colors) {
	if !a.remotePicker {
		return
	}
	dirs := a.remoteDirs
	q := strings.ToLower(strings.TrimSpace(a.remoteQuery))
	a.remoteErr = ""
	ui.DialogBase(c, &a.remotePicker, func(backdrop, panel ui.Element) {
		backdrop.Background(k.Backdrop)
		panel.Width(620).Height(520).Radius(radius).Background(k.Panel).
			Border(1, k.PanelBorder).
			Shadow(0, 24, 60, -8, ui.RGBA(0, 0, 0, 0.28)).Clip()
		ui.Column(c).Children(func() {
			ui.Row(c).Height(50).Padding(0, 14, 0, 12).Gap(9).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(k.PanelBorder).Children(func() {
				ui.Icon(c, icons.Icon("server")).Size(16, 16).TextColor(k.TextMuted)
				ui.Text(c, "Open Project on "+a.remoteHost).FontSize(13.5).
					FontWeight(600).TextColor(k.Text)
				ui.Spacer(c)
				if iconButton(c, k, "x", "Close", 24, 13).Clicked() {
					a.remotePicker = false
				}
			})
			ui.Row(c).Padding(10, 14).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Icon("search")).Size(14, 14).TextColor(k.TextFaint)
				in := ui.TextInputBase(c, &a.remoteQuery).Grow(1).FontSize(13).
					Placeholder("Server project path…").AutoFocus()
				open := func() {
					path := strings.TrimSpace(a.remoteQuery)
					if path == "" {
						return
					}
					a.remotePicker = false
					a.dropPickerRemote()
					a.later(func() {
						if err := a.openRemoteProject(a.remoteHost, path); err != nil {
							a.fail(err)
						}
					})
				}
				if in.Submitted() {
					open()
				}
				if ui.PrimaryButton(c, "Open").Clicked() {
					open()
				}
			})
			if strings.TrimSpace(q) != "" && dirs == nil {
				a.remoteErr = "Reading folders…"
			}
			ui.Scroll(c).Grow(1).MinHeight(0).Padding(6).Children(func() {
				for _, d := range dirs {
					if !d.Dir {
						continue
					}
					if q != "" && !strings.Contains(strings.ToLower(d.Rel), q) {
						continue
					}
					d := d
					row := ui.Row(c).Key(d.Rel).Height(34).Padding(0, 10).Gap(8).
						Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).
						Label("Open " + d.Rel)
					if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() {
						a.remoteQuery = d.Rel
						rc := a.pickerRemote()
						a.remoteDirs = nil
						if rc != nil {
							if entries, err := rc.client.ListDirs(d.Rel); err == nil {
								a.remoteDirs = entries
							}
						}
					}
					row.Children(func() {
						ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
						ui.Text(c, filepath.Base(d.Rel)).FontSize(12.5).TextColor(k.Text).
							SingleLine().Ellipsis("…")
					})
				}
				if len(dirs) == 0 {
					ui.Text(c, a.remoteErr).FontSize(12).TextColor(k.TextFaint).Padding(14)
				}
			})
		})
	})
	if !a.remotePicker {
		a.dropPickerRemote()
	}
}

func (a *app) serverChip(c *ui.Context, k *colors) {
	label := a.remoteName()
	chip := ui.Row(c).Gap(6).Padding(3, 8, 3, 6).Radius(radius).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Switch server: " + label).
		Tooltip("Server  " + nativeKeys("⇧⌘L")).MaxWidth(170).Shrink(0).ClipX()
	if chip.Hovered() || a.serverOpen {
		chip.Background(k.Hover)
	}
	if chip.Clicked() {
		a.serverOpen = !a.serverOpen
	}
	chip.Children(func() {
		ui.Icon(c, icons.Icon("server")).Size(16, 16).TextColor(k.TextMuted)
		ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(k.TextMuted).
			SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		ui.Icon(c, icons.Icon("chevron-down")).Size(12, 12).TextColor(k.TextFaint)
	})
	ui.PopoverBase(c, chip, &a.serverOpen, func(panel ui.Element) {
		panel.Width(280).Padding(9).Radius(radius).Background(k.Panel).
			Border(1, k.PanelBorder).Shadow(0, 12, 32, 0, k.ShadowFocused).Gap(1)
		ui.Text(c, "SERVERS").FontSize(10).FontWeight(650).TextColor(k.TextFaint).Padding(4, 8)
		a.serverRow(c, k, "", "Local")
		for _, h := range a.hosts {
			label := h.Name
			if label == "" {
				label = h.Address
			}
			a.serverRow(c, k, h.Address, label)
		}
		if len(a.hosts) == 0 {
			ui.Text(c, "Add a server in Settings › Hosts.").FontSize(11).
				TextColor(k.TextFaint).Padding(4, 8)
		}
	})
}

func (a *app) serverRow(c *ui.Context, k *colors, host, label string) {
	selected := a.serverAlias == host
	row := ui.Row(c).Key("server:"+host).Height(36).Padding(0, 8).Gap(8).
		Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Server " + label)
	if selected {
		row.Background(k.PanelSel)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		a.serverOpen = false
		if host == "" {
			a.later(func() {
				if err := a.saveNow(); err != nil {
					a.fail(err)
					return
				}
				a.closeRemote()
				a.serverAlias = ""
				a.remotePath = ""
				a.saveSettings()
				if a.project != "" {
					a.openProject(a.project)
				}
			})
		} else {
			a.later(func() { a.openRemotePicker(host) })
		}
	}
	row.Children(func() {
		ui.Icon(c, icons.Icon("server")).Size(14, 14).TextColor(k.TextMuted)
		ui.Text(c, label).FontSize(12.5).TextColor(k.Text)
		if selected {
			ui.Spacer(c)
			ui.Icon(c, icons.Icon("check")).Size(13, 13).TextColor(k.TextMuted)
		}
	})
}

func (a *app) remoteProjectRows(c *ui.Context, k *colors) {
	recent := a.remoteProjects
	ui.Text(c, "PROJECTS ON "+strings.ToUpper(a.remoteName())).
		FontSize(10).FontWeight(650).TextColor(k.TextFaint).Padding(4, 8)
	if len(recent) == 0 {
		ui.Text(c, "No recent remote projects.").FontSize(11.5).TextColor(k.TextFaint).Padding(4, 8)
	}
	for _, p := range recent {
		if p.Host != a.serverAlias {
			continue
		}
		p := p
		row := ui.Row(c).Key("remote:"+p.Host+":"+p.Path).Height(39).Padding(0, 8).
			Gap(8).Radius(radius).Cursor(ui.CursorPointer).Role(ui.RoleButton).
			Label(p.Path)
		if p.Path == a.remotePath {
			row.Background(k.PanelSel)
		} else if row.Hovered() {
			row.Background(k.Hover)
		}
		if row.Clicked() {
			a.projectOpen = false
			a.later(func() {
				if err := a.openRemoteProject(p.Host, p.Path); err != nil {
					a.fail(err)
				}
			})
		}
		row.Children(func() {
			ui.Icon(c, icons.FolderIcon()).Size(15, 15).TextColor(k.TextMuted)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, filepath.Base(p.Path)).FontSize(12.5).TextColor(k.Text).
					SingleLine().Ellipsis("…")
				ui.Text(c, p.Path).FontSize(10.5).TextColor(k.TextFaint).
					SingleLine().Ellipsis("…")
			})
		})
	}
}

// hostConfig is a server the user added in Settings › Hosts. Address is
// anything the system ssh accepts, such as buildbox, me@devbox or
// ssh://me@devbox:2222 (the last is passed through as ssh accepts it).
type hostConfig struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (a *app) hostLabel(address string) string {
	for _, h := range a.hosts {
		if h.Address == address {
			if h.Name != "" {
				return h.Name
			}
			return h.Address
		}
	}
	return address
}

func (a *app) addHost(name, address string) {
	address = strings.TrimSpace(address)
	name = strings.TrimSpace(name)
	if address == "" {
		return
	}
	if name == "" {
		name = address
	}
	for i := range a.hosts {
		if a.hosts[i].Address == address {
			a.hosts[i].Name = name
			a.saveSettings()
			a.invalidate()
			return
		}
	}
	a.hosts = append(a.hosts, hostConfig{Name: name, Address: address})
	a.saveSettings()
	a.invalidate()
}
