// DocumentMCP stdio entry point.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/local/go-mcp-anthropic-introduction/internal/docserver"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := docserver.New().Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Error("server exited", "err", err)
		os.Exit(1)
	}
}
