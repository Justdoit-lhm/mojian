// Package main starts Mojian, the MyGo app, and its terminal session
// server; the window itself lives in internal/app.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go test ./...       the model, the store and the view, without a window
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"mojian/internal/app"
	"mojian/internal/session"

	"github.com/egoist/mygo"
)

const usage = `usage: mojian [path]

Open a project folder, or a file inside one:

  mojian .
  mojian ~/git/abc
  mojian ~/git/abc/README.md

  -h, --help  show this help
`

// errHelp ends argument parsing for -h and --help.
var errHelp = errors.New("help")

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-server" {
		log.SetPrefix("[session] ")
		if err := session.Serve(); err != nil {
			log.Fatal(err)
		}
		return
	}
	startup, err := parseStartup(os.Args[1:])
	if err != nil {
		if errors.Is(err, errHelp) {
			fmt.Print(usage)
			return
		}
		fmt.Fprintf(os.Stderr, "mojian: %v\n\n%s", err, usage)
		os.Exit(2)
	}
	// The session server keeps its socket and log in the app's data
	// directory, apart between the installed app and mygo dev.
	if os.Getenv("MOJIAN_DIR") == "" {
		if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
			_ = os.Setenv("MOJIAN_DIR", dir)
		}
	}
	app.Run(startup)
}

// parseStartup reads the one path the command line may name. A folder
// becomes the project; a file makes its folder the project and names the
// file to show.
func parseStartup(args []string) (app.Startup, error) {
	var paths []string
	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help" || arg == "help":
			return app.Startup{}, errHelp
		case strings.HasPrefix(arg, "-psn_"):
			// macOS LaunchServices passes this when it opens the app.
		case strings.HasPrefix(arg, "-") && arg != "-":
			return app.Startup{}, fmt.Errorf("unknown option %q", arg)
		default:
			paths = append(paths, arg)
		}
	}
	if len(paths) > 1 {
		return app.Startup{}, fmt.Errorf("expected one path, got %d", len(paths))
	}
	if len(paths) == 0 {
		return app.Startup{}, nil
	}
	abs, err := resolvePath(paths[0])
	if err != nil {
		return app.Startup{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return app.Startup{}, fmt.Errorf("%s: %w", paths[0], err)
	}
	if info.IsDir() {
		return app.Startup{Dir: abs}, nil
	}
	if !info.Mode().IsRegular() {
		return app.Startup{}, fmt.Errorf("%s: not a file or folder", paths[0])
	}
	return app.Startup{Dir: filepath.Dir(abs), File: abs}, nil
}

// resolvePath expands a leading ~ and makes p absolute, so the window
// opens the path the shell meant even when the tilde was quoted.
func resolvePath(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			p = home
		} else {
			p = filepath.Join(home, p[2:])
		}
	}
	return filepath.Abs(p)
}
