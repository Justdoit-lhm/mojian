// Package main is Mojian's session and project server, the binary Mojian
// uploads to an SSH host; `-stop` ends a running one. It is never
// installed as a service.
package main

import (
	"log"
	"os"

	"mojian/internal/session"
)

func main() {
	log.SetPrefix("[mojian-server] ")
	if len(os.Args) > 1 && os.Args[1] == "-stop" {
		if err := session.Stop(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := session.Serve(); err != nil {
		log.Fatal(err)
	}
}
