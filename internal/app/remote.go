package app

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

func (a *app) remoteProjectName() string {
	if a.remotePath == "" {
		return ""
	}
	return filepath.Base(a.remotePath)
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
	args := append(sshCommon(), host, command)
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
// socket over SSH, and connects the control client.
func (a *app) ensureRemote(host string) (*remoteConn, error) {
	if a.remote != nil && a.remote.host == host && a.remote.client != nil {
		return a.remote, nil
	}
	if host == "" {
		return nil, fmt.Errorf("no server selected")
	}
	a.closeRemote()
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
	dir := home + "/.local/share/mojian"
	binPath := dir + "/mojian-server"
	sock := dir + "/sessions.sock"
	lib, libErr := remotebin.Library(arch)
	if libErr != nil {
		return nil, libErr
	}
	libPath := dir + "/libghostty-vt.so"
	if _, err := sshOutput(host, "mkdir -p "+shellQuote(dir)+" && chmod 700 "+shellQuote(dir)); err != nil {
		return nil, err
	}
	// Upload the server when the remote copy differs.
	size, _ := sshOutput(host, "stat -c %s "+shellQuote(binPath)+" 2>/dev/null || true")
	if size != fmt.Sprint(len(bin)) {
		cmd := exec.Command("ssh", append(sshCommon(), host,
			"mkdir -p "+shellQuote(dir)+" && cat > "+shellQuote(binPath+".tmp")+
				" && chmod 755 "+shellQuote(binPath+".tmp")+
				" && mv "+shellQuote(binPath+".tmp")+" "+shellQuote(binPath))...)
		cmd.Stdin = bytes.NewReader(bin)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("uploading the session server: %v %s", err, strings.TrimSpace(stderr.String()))
		}
	}
	// The server's headless terminal needs libghostty-vt beside it.
	libSize, _ := sshOutput(host, "stat -c %s "+shellQuote(libPath)+" 2>/dev/null || true")
	if libSize != fmt.Sprint(len(lib)) {
		cmd := exec.Command("ssh", append(sshCommon(), host,
			"cat > "+shellQuote(libPath+".tmp")+" && mv "+shellQuote(libPath+".tmp")+" "+shellQuote(libPath))...)
		cmd.Stdin = bytes.NewReader(lib)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("uploading libghostty-vt: %v %s", err, strings.TrimSpace(stderr.String()))
		}
	}
	// Start the per-user server. A live server refuses the second one
	// through its lock; a stale socket from a server that exited is
	// replaced by the new process. Starting it every time also recovers
	// from the server's idle timeout.
	startCmd := "MOJIAN_DIR=" + shellQuote(dir) + " nohup " + shellQuote(binPath) +
		" >>" + shellQuote(dir+"/server.log") + " 2>&1 &"
	if _, err := sshOutput(host, startCmd); err != nil {
		return nil, err
	}
	time.Sleep(250 * time.Millisecond)
	local := filepath.Join(os.TempDir(), "mojian-"+shortHash(host)+".sock")
	_ = os.Remove(local)
	fwd := exec.Command("ssh", "-N", "-o", "BatchMode=yes", "-o", "ConnectTimeout=20",
		"-o", "ExitOnForwardFailure=yes", "-o", "ServerAliveInterval=15",
		"-L", local+":"+sock, host)
	var ferr bytes.Buffer
	fwd.Stderr = &ferr
	if err := fwd.Start(); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(20 * time.Second)
	var conn net.Conn
	for {
		conn, err = net.DialTimeout("unix", local, 200*time.Millisecond)
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
		_ = fwd.Process.Kill()
		return nil, err
	}
	rc := &remoteConn{host: host, socket: local, forward: fwd, client: client}
	a.remote = rc
	return rc, nil
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

func (a *app) closeRemote() {
	if a.remote == nil {
		return
	}
	if a.remote.client != nil {
		_ = a.remote.client.Close()
	}
	if a.remote.forward != nil && a.remote.forward.Process != nil {
		_ = a.remote.forward.Process.Kill()
		_, _ = a.remote.forward.Process.Wait()
	}
	_ = os.Remove(a.remote.socket)
	a.remote = nil
}

// remoteCacheDir is a local mirror of a remote project. The server is
// still the source of truth; the mirror lets the existing editor, assets,
// and Git page work while edits sync back.
func (a *app) remoteCacheDir(host, remotePath string) string {
	base := a.dataDir
	if base == "" {
		base, _ = os.UserCacheDir()
	}
	return filepath.Join(base, "remote", shortHash(host+"\x00"+remotePath))
}

// archiveRemote downloads the project over the session server and opens
// it as a local mirror.
func (a *app) downloadRemote(c *session.Client, remotePath, cache string) error {
	data, err := c.Archive(remotePath)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(cache); err != nil {
		return err
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return err
	}
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(h.Name)
		if name == "." || strings.HasPrefix(name, ".."+string(filepath.Separator)) || filepath.IsAbs(name) {
			continue
		}
		dst := filepath.Join(cache, name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	return nil
}

func remoteManifestPath(cache string) string { return cache + ".files" }

func localFileList(cache string) []string {
	var out []string
	_ = filepath.WalkDir(cache, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(cache, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out
}

func writeRemoteManifest(cache string, files []string) {
	_ = os.WriteFile(remoteManifestPath(cache), []byte(strings.Join(files, "\n")), 0o644)
}

func readRemoteManifest(cache string) map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(remoteManifestPath(cache))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			out[line] = true
		}
	}
	return out
}

func (a *app) uploadRemoteNote(n *note) error {
	if a.remote == nil || a.remote.client == nil || a.remotePath == "" {
		return nil
	}
	rel := strings.ReplaceAll(n.Rel, string(filepath.Separator), "/")
	return a.remote.client.WriteFile(path.Join(a.remotePath, rel), []byte(n.text()))
}

// openRemoteProject mirrors path from host and makes it the open project.
func (a *app) openRemoteProject(host, remotePath string) error {
	if host == "" || remotePath == "" {
		return fmt.Errorf("choose a server and a project")
	}
	remotePath = strings.TrimRight(remotePath, "/")
	a.saveNow()
	if a.layoutPath != "" {
		a.saveLayout()
	}
	rc, err := a.ensureRemote(host)
	if err != nil {
		return err
	}
	a.closeTerminals()
	a.git.Open = false
	cache := a.remoteCacheDir(host, remotePath)
	if err := a.downloadRemote(rc.client, remotePath, cache); err != nil {
		return err
	}
	s, err := openStoreAt(cache)
	if err != nil {
		return err
	}
	s.onSave = a.uploadRemoteNote
	a.store = s
	a.treeOpen = true
	a.serverAlias = host
	a.remotePath = remotePath
	a.remoteCache = cache
	a.layoutPath = a.projectLayoutPath("remote:" + host + ":" + remotePath)
	a.rememberRemoteProject(remoteProject{Host: host, Path: remotePath})
	a.tabs = nil
	a.active = -1
	a.focusReq = nil
	a.overview = false
	a.overviewFocus = false
	a.treeSel = ""
	a.treeQuery = ""
	a.treeFocusSearch = false
	a.treeReveal = ""
	a.openDirs = map[string]bool{}
	a.tabScroll = ui.ScrollState{}
	a.treeDraft = nil
	a.rename = nil
	a.paletteOpen = false
	a.splitPick = false
	a.splitFrom = nil
	a.splitTab = nil
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
	a.saveNow()
	rc, err := a.ensureRemote(host)
	if err != nil {
		a.fail(err)
		return
	}
	homeDir, err := sshOutput(host, `printf %s "$HOME"`)
	if err != nil {
		a.fail(err)
		return
	}
	a.remoteHost = host
	a.remoteQuery = homeDir
	a.remoteDirs, err = rc.client.ListDirs(homeDir)
	if err != nil {
		a.fail(err)
		return
	}
	a.remotePicker = true
	a.serverOpen = false
	a.invalidate()
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
		panel.Width(620).Height(520).Radius(16).Background(k.Panel).
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
						Radius(7).Cursor(ui.CursorPointer).Role(ui.RoleButton).
						Label("Open " + d.Rel)
					if row.Hovered() {
						row.Background(k.Hover)
					}
					if row.Clicked() {
						a.remoteQuery = d.Rel
						rc := a.remote
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
}

func (a *app) serverChip(c *ui.Context, k *colors) {
	label := a.remoteName()
	chip := ui.Row(c).Gap(6).Padding(3, 8, 3, 6).Radius(9).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Switch server: " + label).
		Tooltip("Server  ⇧⌘L").MaxWidth(170).Shrink(0).ClipX()
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
		panel.Width(280).Padding(9).Radius(14).Background(k.Panel).
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
		Radius(7).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Server " + label)
	if selected {
		row.Background(k.PanelSel)
	} else if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		a.serverOpen = false
		if host == "" {
			a.later(func() {
				a.saveNow()
				a.closeRemote()
				a.serverAlias = ""
				a.remotePath = ""
				a.remoteCache = ""
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
			Gap(8).Radius(7).Cursor(ui.CursorPointer).Role(ui.RoleButton).
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

// syncRemoteProject uploads the local mirror, except .git, and removes
// files that were deleted locally. It is used after file operations; text
// saves use the per-file hook instead.
func (a *app) syncRemoteProject() {
	if a.remote == nil || a.remote.client == nil || a.remotePath == "" || a.remoteCache == "" {
		return
	}
	local := map[string]bool{}
	err := filepath.WalkDir(a.remoteCache, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(a.remoteCache, p)
		if err != nil || rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			return nil
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if err := a.remote.client.WriteFile(path.Join(a.remotePath, rel), data); err == nil {
			local[rel] = true
		}
		return nil
	})
	if err != nil {
		return
	}
	old := readRemoteManifest(a.remoteCache)
	entries, err := a.remote.client.ListTree(a.remotePath)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.Dir || local[e.Rel] || !old[e.Rel] || e.Rel == ".git" || strings.HasPrefix(e.Rel, ".git/") {
			continue
		}
		_ = a.remote.client.Remove(path.Join(a.remotePath, e.Rel))
	}
	files := make([]string, 0, len(local))
	for rel := range local {
		files = append(files, rel)
	}
	writeRemoteManifest(a.remoteCache, files)
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

func (a *app) removeHost(address string) {
	out := a.hosts[:0]
	for _, h := range a.hosts {
		if h.Address != address {
			out = append(out, h)
		}
	}
	a.hosts = out
	a.saveSettings()
	a.invalidate()
}
