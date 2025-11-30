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

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/config"
	httpserver "github.com/fyrsmithlabs/contextd/internal/http"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/mcp"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/telemetry"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
)

// Version information (set at build time via ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

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
	httpPort := flag.Int("http-port", 0, "HTTP server port (overrides config, default: 9090)")
	httpHost := flag.String("http-host", "", "HTTP server host (overrides config, default: localhost)")
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
	// Always try to load from file first (default: ~/.config/contextd/config.yaml)
	// Falls back to environment-only config if file doesn't exist
	var cfg *config.Config
	cfg, err = config.LoadWithFile(*configPath)
	if err != nil {
		// Check if it's just a missing file (acceptable) vs actual error
		if *configPath == "" {
			// No explicit config path, try env-only fallback
			logger.Warn(ctx, "config file not found or invalid, using environment variables only",
				zap.Error(err),
			)
			cfg = config.Load()
		} else {
			// Explicit config path specified but failed - this is an error
			return fmt.Errorf("loading config from file: %w", err)
		}
	} else {
		if *configPath != "" {
			logger.Info(ctx, "config loaded from file", zap.String("path", *configPath))
		} else {
			logger.Info(ctx, "config loaded from default location (~/.config/contextd/config.yaml)")
		}
	}

	// ============================================================================
	// PHASE 3: MCP Server
	// ============================================================================
	// Initialize MCP server with stdio transport and core services.
	// Note: Some dependencies (Qdrant, embedder) are stubs pending implementation.

	// Initialize stub Qdrant client (TODO: replace with real implementation)
	// For now, we'll skip services that require Qdrant since it's not yet implemented
	logger.Warn(ctx, "Qdrant client not yet implemented, some services will be unavailable")

	// Initialize secret scrubber
	scrubCfg := secrets.DefaultConfig()
	scrubCfg.Enabled = true
	scrubber, err := secrets.New(scrubCfg)
	if err != nil {
		return fmt.Errorf("initializing secret scrubber: %w", err)
	}
	logger.Info(ctx, "secret scrubber initialized")

	// ============================================================================
	// Initialize HTTP Server
	// ============================================================================
	// Determine HTTP server configuration (flags override config)
	httpServerHost := "localhost"
	if *httpHost != "" {
		httpServerHost = *httpHost
	}

	httpServerPort := cfg.Server.Port
	if *httpPort != 0 {
		httpServerPort = *httpPort
	}

	httpCfg := &httpserver.Config{
		Host: httpServerHost,
		Port: httpServerPort,
	}

	httpSrv, err := httpserver.NewServer(scrubber, logger.Underlying(), httpCfg)
	if err != nil {
		return fmt.Errorf("initializing HTTP server: %w", err)
	}
	logger.Info(ctx, "HTTP server initialized",
		zap.String("host", httpServerHost),
		zap.Int("port", httpServerPort),
	)

	// Start HTTP server in background goroutine
	httpErrChan := make(chan error, 1)
	go func() {
		if err := httpSrv.Start(); err != nil {
			httpErrChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Initialize checkpoint service (stub - requires Qdrant)
	// Using nil for now since Qdrant is not implemented
	var checkpointSvc checkpoint.Service
	logger.Warn(ctx, "checkpoint service unavailable (requires Qdrant implementation)")

	// Initialize remediation service (stub - requires Qdrant + embedder)
	var remediationSvc remediation.Service
	logger.Warn(ctx, "remediation service unavailable (requires Qdrant + embedder)")

	// Initialize repository service (requires checkpoint service)
	var repositorySvc *repository.Service
	logger.Warn(ctx, "repository service unavailable (requires checkpoint service)")

	// Initialize troubleshoot service (stub - requires vectorstore)
	var troubleshootSvc *troubleshoot.Service
	logger.Warn(ctx, "troubleshoot service unavailable (requires vectorstore)")

	// Initialize reasoningbank service (stub - requires vectorstore)
	var reasoningbankSvc *reasoningbank.Service
	logger.Warn(ctx, "reasoningbank service unavailable (requires vectorstore)")

	// Create MCP server with available services
	// Note: Most tools will be unavailable until infrastructure is implemented
	mcpCfg := &mcp.Config{
		Name:    "contextd-v2",
		Version: version,
		Logger:  logger.Underlying(), // Get underlying *zap.Logger
	}

	// For now, skip MCP server creation since required services are unavailable
	// This will be enabled once Qdrant and vectorstore are implemented
	logger.Warn(ctx, "MCP server initialization skipped - required services unavailable")
	logger.Info(ctx, "To enable MCP server, implement: Qdrant client, embedder, vectorstore")

	// Keep the config for future use
	_ = mcpCfg
	_ = scrubber
	_ = checkpointSvc
	_ = remediationSvc
	_ = repositorySvc
	_ = troubleshootSvc
	_ = reasoningbankSvc

	logger.Info(ctx, "contextd initialized (infrastructure pending)",
		zap.String("http_host", httpServerHost),
		zap.Int("http_port", httpServerPort),
		zap.String("status", "Phase 4 complete, Phase 5-6 pending"),
	)

	// Log config for debugging (will be redacted if contains secrets)
	_ = cfg // Config loaded but not yet used by services

	// Wait for shutdown signal or HTTP server error
	select {
	case <-ctx.Done():
		logger.Info(ctx, "shutdown signal received")
	case err := <-httpErrChan:
		logger.Error(ctx, "HTTP server error", zap.Error(err))
		return err
	}

	logger.Info(ctx, "shutting down contextd")

	// Gracefully shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Info(ctx, "HTTP server stopped")
	}

	// Shutdown is handled by deferred functions above

	logger.Info(ctx, "contextd stopped")
	return nil
}
