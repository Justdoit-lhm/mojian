// Package remotebin holds the server binaries Mojian uploads to SSH
// hosts. They are built by tools/build-remote.sh.
package remotebin

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed mojian-server-linux-amd64 mojian-server-linux-arm64 libghostty-vt-linux-amd64.so libghostty-vt-linux-arm64.so
var files embed.FS

// machineArch maps `uname -m` to the architecture name of the embedded
// server and library files.
func machineArch(machine string) string {
	switch strings.ToLower(strings.TrimSpace(machine)) {
	case "aarch64", "arm64":
		return "arm64"
	case "x86_64", "amd64", "x64":
		return "amd64"
	}
	return ""
}

// Library returns libghostty-vt for the remote architecture.
func Library(machine string) ([]byte, error) {
	arch := machineArch(machine)
	if arch == "" {
		return nil, fmt.Errorf("unsupported remote architecture %q", machine)
	}
	return files.ReadFile("libghostty-vt-linux-" + arch + ".so")
}

// Binary returns the server for a remote Linux machine, from `uname -m`.
func Binary(machine string) ([]byte, error) {
	arch := machineArch(machine)
	if arch == "" {
		return nil, fmt.Errorf("unsupported remote architecture %q (Mojian server supports Linux x86_64 and arm64)", machine)
	}
	return files.ReadFile("mojian-server-linux-" + arch)
}
