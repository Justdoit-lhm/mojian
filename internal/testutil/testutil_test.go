package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestPickNative chooses the library built for this machine out of a cache
// holding both architectures, which is what a CI runner that seeded arm64
// and amd64 has: taking the other one fails every terminal test.
func TestPickNative(t *testing.T) {
	dir := t.TempDir()
	other := "amd64"
	if runtime.GOARCH == "amd64" {
		other = "arm64"
	}
	macho := func(arch string) []byte {
		cputype := uint32(0x01000007)
		if arch == "arm64" {
			cputype = 0x0100000c
		}
		b := []byte{0xcf, 0xfa, 0xed, 0xfe}
		return append(b, byte(cputype), byte(cputype>>8), byte(cputype>>16), byte(cputype>>24))
	}
	write := func(name, arch string) string {
		path := filepath.Join(dir, name)
		data := make([]byte, 64)
		copy(data, macho(arch))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	wrong := write("a-"+other, other) // sorts first, as CI's amd64 folder does
	right := write("b-"+runtime.GOARCH, runtime.GOARCH)
	if got := pickNative([]string{wrong, right}); got != right {
		t.Fatalf("pickNative chose %q, want the %s library %q", got, runtime.GOARCH, right)
	}
	if got := pickNative([]string{wrong}); got != wrong {
		t.Fatalf("pickNative dropped a lone library: %q", got)
	}
	if got := pickNative(nil); got != "" {
		t.Fatalf("pickNative with nothing cached: %q", got)
	}
}

// TestNativeArch reads headers of each kind, because the flag that picks the
// library is read out of them.
func TestNativeArch(t *testing.T) {
	write := func(name string, head []byte) string {
		path := filepath.Join(t.TempDir(), name)
		data := make([]byte, 64)
		copy(data, head)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	macho := func(cputype uint32) []byte {
		b := []byte{0xcf, 0xfa, 0xed, 0xfe}
		return append(b, byte(cputype), byte(cputype>>8), byte(cputype>>16), byte(cputype>>24))
	}
	elf := func(machine uint16) []byte {
		b := []byte{0x7f, 'E', 'L', 'F', 2, 1, 1}
		for len(b) < 18 {
			b = append(b, 0)
		}
		return append(b, byte(machine), byte(machine>>8))
	}
	cases := []struct {
		name string
		head []byte
		want string
	}{
		{"macho-arm64.dylib", macho(0x0100000c), "arm64"},
		{"macho-amd64.dylib", macho(0x01000007), "amd64"},
		{"fat.dylib", []byte{0xca, 0xfe, 0xba, 0xbe}, runtime.GOARCH},
		{"elf-arm64.so", elf(0xb7), "arm64"},
		{"elf-amd64.so", elf(0x3e), "amd64"},
		{"text.so", []byte("not a library"), ""},
	}
	for _, c := range cases {
		if got := nativeArch(write(c.name, c.head)); got != c.want {
			t.Fatalf("%s: arch %q, want %q", c.name, got, c.want)
		}
	}
}
