package main

import (
	"log"

	"github.com/chrisw-dev/imap-mcp-go/internal/config"
	"github.com/chrisw-dev/imap-mcp-go/internal/mcpserver"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv := mcpserver.New(cfg)
	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("serve stdio: %v", err)
	}
}
