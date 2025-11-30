// Package main provides the entry point for the contextd server.
//
// contextd is a shared knowledge layer for AI agents, providing:
//   - ReasoningBank: Cross-session memory
//   - Context-Folding: Active context management
//   - Institutional Knowledge: Project → Team → Org hierarchy
//   - Secret Scrubbing: gitleaks-based security
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/telemetry"
)

// Version information (set at build time via ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// isFlagPassed returns true if the flag was explicitly set on the command line.
func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse flags
	configPath := flag.String("config", "", "path to config file (optional)")
	showVersion := flag.Bool("version", false, "show version information")
	port := flag.Int("port", 50051, "server port")
	host := flag.String("host", "localhost", "server host")
	flag.Parse()

	if *showVersion {
		fmt.Printf("contextd %s (commit: %s, built: %s)\n", version, commit, buildDate)
		return nil
	}

	// Create root context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ============================================================================
	// Initialize Logging
	// ============================================================================
	logCfg := logging.NewDefaultConfig()
	logger, err := logging.NewLogger(logCfg, nil)
	if err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}
	defer logger.Sync()

	logger.Info(ctx, "starting contextd",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_date", buildDate),
	)

	// ============================================================================
	// Initialize Telemetry (VITAL)
	// ============================================================================
	telCfg := telemetry.NewDefaultConfig()
	telCfg.ServiceName = "contextd"
	// Disabled by default until OTEL collector is available
	// Set OTEL_EXPORTER_OTLP_ENDPOINT to enable
	tel, err := telemetry.New(ctx, telCfg)
	if err != nil {
		logger.Warn(ctx, "telemetry initialization failed, continuing without telemetry",
			zap.Error(err),
		)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if shutdownErr := tel.Shutdown(shutdownCtx); shutdownErr != nil {
				logger.Error(ctx, "telemetry shutdown error", zap.Error(shutdownErr))
			}
		}()
		logger.Info(ctx, "telemetry initialized")
	}

	// ============================================================================
	// Load Configuration
	// ============================================================================
	var cfg *config.Config
	if *configPath != "" {
		cfg, err = config.LoadWithFile(*configPath)
		if err != nil {
			return fmt.Errorf("loading config from file: %w", err)
		}
		logger.Info(ctx, "config loaded from file", zap.String("path", *configPath))
	} else {
		cfg = config.Load()
		logger.Info(ctx, "using default config (no config file specified)")
	}

	// ============================================================================
	// PHASE 3: MCP Server (Not Yet Implemented)
	// ============================================================================
	// TODO: Initialize MCP server with:
	//   - stdio transport for local development
	//   - Tool discovery and registration
	//   - Session management
	//   - Memory service integration
	//
	// Example:
	//   mcpServer := mcp.NewServer(mcp.Config{
	//       Transport: mcp.TransportStdio,
	//       Logger:    logger,
	//   })
	//   defer mcpServer.Close()

	// ============================================================================
	// PHASE 5: HTTP Server (Not Yet Implemented)
	// ============================================================================
	// TODO: Initialize dual-protocol server (gRPC + HTTP) with:
	//   - cmux for port multiplexing
	//   - gRPC services for typed clients
	//   - Echo HTTP REST API for simplicity
	//   - Secret scrubbing on all responses
	//   - Process isolation for tool execution
	//
	// Example:
	//   serverCfg := server.Config{
	//       Port:       *port,
	//       Host:       *host,
	//       EnableGRPC: true,
	//   }
	//   if isFlagPassed("port") {
	//       serverCfg.Port = *port
	//   }
	//   if isFlagPassed("host") {
	//       serverCfg.Host = *host
	//   }
	//
	//   srv := server.NewDualServer(serverCfg, &server.Deps{
	//       Logger: logger,
	//       // ... other dependencies
	//   })
	//   defer srv.Shutdown(context.Background())
	//
	//   go srv.Start()

	logger.Info(ctx, "contextd initialized (foundation only)",
		zap.String("host", *host),
		zap.Int("port", *port),
		zap.String("phase", "Phase 1 complete"),
	)

	// Log config for debugging (will be redacted if contains secrets)
	_ = cfg // Config loaded but not yet used by services

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info(ctx, "shutdown signal received")

	logger.Info(ctx, "shutting down contextd")

	// Shutdown is handled by deferred functions above

	logger.Info(ctx, "contextd stopped")
	return nil
}
