package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/config"
	"github.com/fyrsmithlabs/contextd/pkg/mcp/stdio"
)

// runStdioServer starts the MCP server in stdio mode for Claude Code integration.
//
// This function implements the HTTP delegation architecture:
//  1. Creates stdio MCP server with SDK
//  2. Delegates all tool calls to HTTP daemon (localhost:9090)
//  3. HTTP daemon handles all service logic (Qdrant, embeddings, etc.)
//
// This architecture:
//   - Reuses all existing HTTP service logic
//   - Supports multiple concurrent stdio sessions
//   - Maintains multi-tenant isolation
//   - No service duplication
func runStdioServer(ctx context.Context, cfg *config.Config) error {
	// Initialize logger for stdio mode
	var err error
	logger, err = initLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("Starting contextd in MCP stdio mode (HTTP delegation)")

	// Build daemon URL from config
	daemonURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)

	logger.Info("Delegating to HTTP daemon",
		zap.String("daemon_url", daemonURL),
		zap.Int("port", cfg.Server.Port))

	// Create stdio MCP server with HTTP delegation
	mcpServer, err := stdio.NewServer(daemonURL)
	if err != nil {
		return fmt.Errorf("failed to create stdio server: %w", err)
	}

	logger.Info("stdio MCP server created successfully")

	// Log startup message to stderr (stdio uses stdout for MCP protocol)
	fmt.Fprintf(os.Stderr, "contextd stdio mode started (delegating to daemon at %s)\n", daemonURL)

	// Run stdio server (blocks until context canceled)
	if err := mcpServer.Run(ctx); err != nil {
		return fmt.Errorf("stdio server error: %w", err)
	}

	logger.Info("stdio MCP server shutdown complete")
	return nil
}
