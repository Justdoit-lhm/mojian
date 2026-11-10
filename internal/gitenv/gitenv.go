// Package gitenv builds an environment a Git command can trust: the
// inherited one without the variables a shell could set to point Git at
// another repository, which would make a project's status and diff lie.
package gitenv

import (
	"os"
	"runtime"
	"strings"
)

// drop redirects Git away from the repository a command runs in.
var drop = map[string]bool{
	"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_INDEX_FILE": true,
	"GIT_OBJECT_DIRECTORY": true, "GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
	"GIT_COMMON_DIR": true,
}

// Env returns the inherited environment without Git's redirect variables,
// then appends extra.
func Env(extra ...string) []string {
	out := make([]string, 0, len(os.Environ())+len(extra))
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 && dropEnv(kv[:i]) {
			continue
		}
		out = append(out, kv)
	}
	return append(out, extra...)
}

func dropEnv(key string) bool {
	if runtime.GOOS == "windows" {
		key = strings.ToUpper(key)
	}
	return drop[key]
}
