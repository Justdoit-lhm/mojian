// mojian-server runs Mojian's session and project server on a remote
// machine. Mojian uploads it over SSH; it is not installed as a service.
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
