// Package testutil provides the temporary-project and Git fixtures shared by
// Mojian's tests.
package testutil

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// envVar is one environment variable as it was before Main changed it.
type envVar struct {
	name  string
	value string
	set   bool
}

// envSet is an environment variable Main gives a test binary.
type envSet struct {
	name  string
	value string
}

// mygoNative finds the libghostty-vt MyGo downloaded for the real user, so
// a sandboxed test binary loads it instead of fetching it again into a
// temporary home where nothing is cached. MyGo's cache keeps one folder per
// platform, so a machine that once built both architectures holds both
// files; take the one built for this one, or a test loads the other and
// fails to link.
func mygoNative() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(cache, "mygo", "natives", "*")
	matches, err := filepath.Glob(filepath.Join(dir, "libghostty-vt.*"))
	if err != nil {
		return ""
	}
	windows, err := filepath.Glob(filepath.Join(dir, "ghostty-vt.dll"))
	if err != nil {
		return ""
	}
	return pickNative(append(matches, windows...))
}

// pickNative chooses among the cached libraries: the one built for this
// machine when the cache holds several, and otherwise the first, so a
// machine MyGo reads nothing for still hands over what it has.
func pickNative(matches []string) string {
	for _, m := range matches {
		if nativeArch(m) == runtime.GOARCH {
			return m
		}
	}
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// nativeArch reads the architecture a shared library was built for from its
// header: Mach-O's cputype or ELF's machine. A universal binary, which
// holds several, matches this process's architecture; a file of neither
// kind, or of a kind not read here, matches nothing.
func nativeArch(path string) string {
	data := make([]byte, 64)
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	n, _ := io.ReadFull(f, data)
	f.Close()
	data = data[:n]
	if len(data) < 20 {
		return ""
	}
	be := func(o int) uint32 {
		return uint32(data[o])<<24 | uint32(data[o+1])<<16 | uint32(data[o+2])<<8 | uint32(data[o+3])
	}
	le := func(o int) uint32 {
		return uint32(data[o]) | uint32(data[o+1])<<8 | uint32(data[o+2])<<16 | uint32(data[o+3])<<24
	}
	switch {
	case be(0) == 0xcafebabe, le(0) == 0xcafebabe, be(0) == 0xcafebabf:
		// A universal binary loads on every architecture it carries.
		return runtime.GOARCH
	case le(0) == 0xfeedfacf || le(0) == 0xfeedface: // Mach-O, 64- or 32-bit
		switch le(4) {
		case 0x0100000c, 12: // CPU_TYPE_ARM64, CPU_TYPE_ARM
			return "arm64"
		case 0x01000007, 7: // CPU_TYPE_X86_64, CPU_TYPE_X86
			return "amd64"
		}
	case be(0) == 0x7f454c46: // ELF, whose header is little-endian here
		if data[4] != 2 || data[5] != 1 { // ELFCLASS64, ELFDATA2LSB
			return ""
		}
		switch uint16(data[18]) | uint16(data[19])<<8 {
		case 0xb7: // EM_AARCH64
			return "arm64"
		case 0x3e: // EM_X86_64
			return "amd64"
		}
	}
	return ""
}

// Main gives a test binary a home of its own: HOME, USERPROFILE, the XDG
// folders and Windows' APPDATA/LOCALAPPDATA point into a fresh temporary
// folder for the whole run, MOJIAN_VAULT and MOJIAN_DIR are cleared so an
// inherited shell cannot point a test at a real project, and Git ignores
// the user's and system's config. The one thing read from the real home is
// MyGo's downloaded libghostty-vt, handed over as MYGO_GHOSTTY_VT so a
// terminal test loads it instead of downloading it into the empty cache.
// A package uses it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }
func Main(m *testing.M) int {
	home, err := os.MkdirTemp("", "mojian-test-home-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "testutil: temp home: %v\n", err)
		return 1
	}
	sets := []envSet{
		{"HOME", home},
		{"USERPROFILE", home},
		{"XDG_CONFIG_HOME", filepath.Join(home, ".config")},
		{"XDG_CACHE_HOME", filepath.Join(home, ".cache")},
		{"XDG_DATA_HOME", filepath.Join(home, ".local", "share")},
		{"XDG_STATE_HOME", filepath.Join(home, ".local", "state")},
		{"APPDATA", filepath.Join(home, "AppData", "Roaming")},
		{"LOCALAPPDATA", filepath.Join(home, "AppData", "Local")},
		{"GIT_CONFIG_NOSYSTEM", "1"},
	}
	if os.Getenv("MYGO_GHOSTTY_VT") == "" {
		// MyGo looks for its native terminal library under the home this
		// function replaces, and downloads it when it is missing. Keep
		// the one the user already has instead.
		if lib := mygoNative(); lib != "" {
			sets = append(sets, envSet{"MYGO_GHOSTTY_VT", lib})
		}
	}
	unsets := []string{"MOJIAN_VAULT", "MOJIAN_DIR", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM"}
	previous := make([]envVar, 0, len(sets)+len(unsets))
	remember := func(name string) {
		value, ok := os.LookupEnv(name)
		previous = append(previous, envVar{name: name, value: value, set: ok})
	}
	for _, s := range sets {
		remember(s.name)
		_ = os.Setenv(s.name, s.value)
	}
	for _, name := range unsets {
		remember(name)
		_ = os.Unsetenv(name)
	}

	code := m.Run()

	for _, v := range previous {
		if v.set {
			_ = os.Setenv(v.name, v.value)
		} else {
			_ = os.Unsetenv(v.name)
		}
	}
	if err := os.RemoveAll(home); err != nil {
		fmt.Fprintf(os.Stderr, "testutil: remove temp home: %v\n", err)
	}
	return code
}

// WriteVault writes files under dir, making their folders.
func WriteVault(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// InitRepo commits dir's files in a fresh Git repository.
func InitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "mojian@example.com")
	run("config", "user.name", "Mojian Test")
	run("add", ".")
	run("commit", "-qm", "initial")
}
