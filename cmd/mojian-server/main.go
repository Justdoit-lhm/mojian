// mojian-server runs Mojian's session and project server on a remote
// machine. Mojian uploads it over SSH; it is not installed as a service.
package main

import (
	"log"

	"mojian/internal/session"
)

func main() {
	log.SetPrefix("[mojian-server] ")
	if err := session.Serve(); err != nil {
		log.Fatal(err)
	}
}
