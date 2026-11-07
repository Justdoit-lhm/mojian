// Mojian is a quiet text editor in MyGo's native UI. The app lives in
// internal/app; this file starts it, and its terminal session server.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go test ./...       the model, the store and the view, without a window
package main

import (
	"log"
	"os"

	"mojian/internal/app"
	"mojian/internal/session"

	"github.com/egoist/mygo"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-server" {
		log.SetPrefix("[session] ")
		if err := session.Serve(); err != nil {
			log.Fatal(err)
		}
		return
	}
	// The session server keeps its socket and log in the app's data
	// directory, apart between the installed app and mygo dev.
	if os.Getenv("MOJIAN_DIR") == "" {
		if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
			_ = os.Setenv("MOJIAN_DIR", dir)
		}
	}
	app.Run()
}
