// Command harness serves Mojian's real views to an MCP client, so an
// agent can inspect, click and screenshot the interface without a window.
//
//	go run ./cmd/harness
//
// It listens on 127.0.0.1:8080 by default; the endpoint is local and
// unauthenticated, so keep it on loopback. Fixtures edit and save only
// their own throwaway project.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	mygomcp "github.com/auron-labs/mygo-mcp"

	"mojian/internal/app"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "loopback address to serve MCP on")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fixtures := []mygomcp.Fixture{
		{Name: "Mojian", Width: 1120, Height: 720, Factory: app.HarnessWindowView},
		{Name: "Mojian Settings", Width: 900, Height: 640, Factory: app.HarnessSettingsView},
	}
	if err := mygomcp.Run(ctx, *addr, fixtures); err != nil {
		log.Fatal(err)
	}
}
