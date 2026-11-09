package main

import "os"

// detachedEnv marks the copy a terminal launch re-ran in its own session,
// so that copy does not try to detach again.
const detachedEnv = "MOJIAN_DETACHED"

// shouldDetach reports whether this launch should leave the terminal.
// Only a terminal launch detaches: launched from Finder or LaunchServices
// there is no terminal to leave, and the process must stay the app
// instance macOS started. mygo dev also stays attached because it watches
// the child's lifetime and console output.
func shouldDetach(stdinTTY, dev bool) bool {
	return stdinTTY && !dev && os.Getenv(detachedEnv) != "1"
}
