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
	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	httpserver "github.com/fyrsmithlabs/contextd/internal/http"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/mcp"
	"github.com/fyrsmithlabs/contextd/internal/qdrant"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/services"
	"github.com/fyrsmithlabs/contextd/internal/telemetry"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
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
	mcpMode := flag.Bool("mcp", false, "run in MCP mode (stdio transport)")
	qdrantHost := flag.String("qdrant-host", "", "Qdrant server host (overrides config, default: localhost)")
	qdrantPort := flag.Int("qdrant-port", 0, "Qdrant server gRPC port (overrides config, default: 6334)")
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
		zap.Bool("mcp_mode", *mcpMode),
	)

	// ============================================================================
	// Initialize Telemetry (VITAL)
	// ============================================================================
	telCfg := telemetry.NewDefaultConfig()
	telCfg.ServiceName = "contextd"
	// Disable telemetry if OTEL_SDK_DISABLED=true or TELEMETRY_ENABLED=false
	if os.Getenv("OTEL_SDK_DISABLED") == "true" || os.Getenv("TELEMETRY_ENABLED") == "false" {
		telCfg.Enabled = false
	}
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
	// Initialize Secret Scrubber
	// ============================================================================
	scrubCfg := secrets.DefaultConfig()
	scrubCfg.Enabled = true
	scrubber, err := secrets.New(scrubCfg)
	if err != nil {
		return fmt.Errorf("initializing secret scrubber: %w", err)
	}
	logger.Info(ctx, "secret scrubber initialized")

	// ============================================================================
	// Initialize Infrastructure (Qdrant + Embeddings)
	// ============================================================================
	var qdrantClient *qdrant.GRPCClient
	var qdrantStore *vectorstore.QdrantStore
	var embeddingProvider embeddings.Provider

	// Determine Qdrant configuration (flags override config file)
	qdrantCfgHost := cfg.Qdrant.Host
	if *qdrantHost != "" {
		qdrantCfgHost = *qdrantHost
	}
	qdrantCfgPort := cfg.Qdrant.Port
	if *qdrantPort != 0 {
		qdrantCfgPort = *qdrantPort
	}

	// Initialize Qdrant gRPC client
	qdrantCfg := &qdrant.ClientConfig{
		Host: qdrantCfgHost,
		Port: qdrantCfgPort,
	}
	qdrantCfg.ApplyDefaults()

	qdrantClient, err = qdrant.NewGRPCClient(qdrantCfg, logger)
	if err != nil {
		logger.Warn(ctx, "Qdrant client initialization failed, MCP services will be unavailable",
			zap.String("host", qdrantCfgHost),
			zap.Int("port", qdrantCfgPort),
			zap.Error(err),
		)
		// Continue without Qdrant - HTTP server can still run
	} else {
		defer qdrantClient.Close()
		logger.Info(ctx, "Qdrant client initialized",
			zap.String("host", qdrantCfgHost),
			zap.Int("port", qdrantCfgPort),
		)

		// Initialize embeddings provider using config values
		embeddingCfg := embeddings.ProviderConfig{
			Provider: cfg.Embeddings.Provider,
			Model:    cfg.Embeddings.Model,
			BaseURL:  cfg.Embeddings.BaseURL,
			CacheDir: cfg.Qdrant.DataPath,
		}
		embeddingProvider, err = embeddings.NewProvider(embeddingCfg)
		if err != nil {
			logger.Warn(ctx, "embeddings provider initialization failed",
				zap.String("provider", embeddingCfg.Provider),
				zap.Error(err),
			)
			// Continue without embedder - some services may be degraded
		} else {
			defer embeddingProvider.Close()

			// Validate and override dimension if provider reports different value
			providerDim := embeddingProvider.Dimension()
			if providerDim != int(cfg.Qdrant.VectorSize) {
				logger.Warn(ctx, "dimension mismatch - using provider dimension",
					zap.Int("config_dimension", int(cfg.Qdrant.VectorSize)),
					zap.Int("provider_dimension", providerDim),
				)
				cfg.Qdrant.VectorSize = uint64(providerDim)
			}

			logger.Info(ctx, "embeddings provider initialized",
				zap.String("provider", cfg.Embeddings.Provider),
				zap.String("model", cfg.Embeddings.Model),
				zap.Int("dimension", providerDim),
			)

			// Initialize QdrantStore with embedder using config values
			vectorStoreCfg := vectorstore.QdrantConfig{
				Host:           qdrantCfgHost,
				Port:           qdrantCfgPort,
				CollectionName: cfg.Qdrant.CollectionName,
				VectorSize:     cfg.Qdrant.VectorSize,
			}

			qdrantStore, err = vectorstore.NewQdrantStore(vectorStoreCfg, embeddingProvider)
			if err != nil {
				logger.Warn(ctx, "QdrantStore initialization failed",
					zap.Error(err),
				)
			} else {
				defer qdrantStore.Close()
				logger.Info(ctx, "QdrantStore initialized",
					zap.String("collection", vectorStoreCfg.CollectionName),
					zap.Uint64("vector_size", vectorStoreCfg.VectorSize),
				)
			}
		}
	}

	// ============================================================================
	// Initialize Services
	// ============================================================================
	var checkpointSvc checkpoint.Service
	var remediationSvc remediation.Service
	var repositorySvc *repository.Service
	var troubleshootSvc *troubleshoot.Service
	var reasoningbankSvc *reasoningbank.Service

	// Initialize checkpoint service
	if qdrantClient != nil {
		checkpointCfg := checkpoint.DefaultServiceConfig()
		checkpointCfg.VectorSize = cfg.Qdrant.VectorSize
		checkpointSvc, err = checkpoint.NewService(checkpointCfg, qdrantClient, logger.Underlying())
		if err != nil {
			logger.Warn(ctx, "checkpoint service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "checkpoint service initialized")
		}
	}

	// Initialize remediation service
	if qdrantClient != nil && embeddingProvider != nil {
		remediationCfg := remediation.DefaultServiceConfig()
		remediationCfg.VectorSize = cfg.Qdrant.VectorSize

		// Create adapters for remediation service
		remediationQdrant := qdrant.NewRemediationAdapter(qdrantClient)
		remediationEmbedder := embeddings.NewRemediationEmbedder(embeddingProvider, int(cfg.Qdrant.VectorSize))

		remediationSvc, err = remediation.NewService(remediationCfg, remediationQdrant, remediationEmbedder, logger.Underlying())
		if err != nil {
			logger.Warn(ctx, "remediation service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "remediation service initialized")
		}
	}

	// Initialize repository service (depends on checkpoint service)
	if checkpointSvc != nil {
		repositorySvc = repository.NewService(checkpointSvc)
		logger.Info(ctx, "repository service initialized")
	}

	// Initialize troubleshoot service
	if qdrantStore != nil {
		troubleshootAdapter := vectorstore.NewTroubleshootAdapter(qdrantStore)
		troubleshootSvc, err = troubleshoot.NewService(troubleshootAdapter, logger.Underlying(), nil)
		if err != nil {
			logger.Warn(ctx, "troubleshoot service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "troubleshoot service initialized")
		}
	}

	// Initialize reasoningbank service
	if qdrantStore != nil {
		reasoningbankSvc, err = reasoningbank.NewService(qdrantStore, logger.Underlying())
		if err != nil {
			logger.Warn(ctx, "reasoningbank service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "reasoningbank service initialized")
		}
	}

	// Initialize hooks manager
	hooksCfg := &hooks.Config{
		AutoCheckpointOnClear: true,
		AutoResumeOnStart:     false,
		CheckpointThreshold:   70,
		VerifyBeforeClear:     true,
	}
	hooksMgr := hooks.NewHookManager(hooksCfg)
	logger.Info(ctx, "hooks manager initialized",
		zap.Int("checkpoint_threshold", hooksCfg.CheckpointThreshold))

	// Create services registry
	registry := services.NewRegistry(services.Options{
		Checkpoint:   checkpointSvc,
		Remediation:  remediationSvc,
		Memory:       reasoningbankSvc,
		Repository:   repositorySvc,
		Troubleshoot: troubleshootSvc,
		Hooks:        hooksMgr,
		Distiller:    nil, // Distiller not yet implemented
		Scrubber:     scrubber,
	})
	logger.Info(ctx, "services registry initialized")

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

	httpSrv, err := httpserver.NewServer(registry, logger.Underlying(), httpCfg)
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

	// ============================================================================
	// Initialize MCP Server (if all services available)
	// ============================================================================
	var mcpServer *mcp.Server
	if *mcpMode {
		// MCP mode requires all services
		if checkpointSvc == nil || remediationSvc == nil || repositorySvc == nil ||
			troubleshootSvc == nil || reasoningbankSvc == nil {
			logger.Error(ctx, "MCP mode requires all services, but some are unavailable",
				zap.Bool("checkpoint", checkpointSvc != nil),
				zap.Bool("remediation", remediationSvc != nil),
				zap.Bool("repository", repositorySvc != nil),
				zap.Bool("troubleshoot", troubleshootSvc != nil),
				zap.Bool("reasoningbank", reasoningbankSvc != nil),
			)
			return fmt.Errorf("MCP mode requires all services to be available")
		}

		mcpCfg := &mcp.Config{
			Name:    "contextd-v2",
			Version: version,
			Logger:  logger.Underlying(),
		}

		mcpServer, err = mcp.NewServer(
			mcpCfg,
			checkpointSvc,
			remediationSvc,
			repositorySvc,
			troubleshootSvc,
			reasoningbankSvc,
			scrubber,
		)
		if err != nil {
			return fmt.Errorf("initializing MCP server: %w", err)
		}
		defer mcpServer.Close()

		logger.Info(ctx, "MCP server initialized, starting stdio transport")

		// Run MCP server (blocks until context is cancelled)
		if err := mcpServer.Run(ctx); err != nil {
			return fmt.Errorf("MCP server error: %w", err)
		}
		return nil
	}

	// Log service availability summary
	serviceStatus := make([]string, 0)
	if checkpointSvc != nil {
		serviceStatus = append(serviceStatus, "checkpoint:ok")
	} else {
		serviceStatus = append(serviceStatus, "checkpoint:unavailable")
	}
	if remediationSvc != nil {
		serviceStatus = append(serviceStatus, "remediation:ok")
	} else {
		serviceStatus = append(serviceStatus, "remediation:unavailable")
	}
	if repositorySvc != nil {
		serviceStatus = append(serviceStatus, "repository:ok")
	} else {
		serviceStatus = append(serviceStatus, "repository:unavailable")
	}
	if troubleshootSvc != nil {
		serviceStatus = append(serviceStatus, "troubleshoot:ok")
	} else {
		serviceStatus = append(serviceStatus, "troubleshoot:unavailable")
	}
	if reasoningbankSvc != nil {
		serviceStatus = append(serviceStatus, "reasoningbank:ok")
	} else {
		serviceStatus = append(serviceStatus, "reasoningbank:unavailable")
	}

	logger.Info(ctx, "contextd initialized",
		zap.String("http_host", httpServerHost),
		zap.Int("http_port", httpServerPort),
		zap.Strings("services", serviceStatus),
	)

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

	logger.Info(ctx, "contextd stopped")
	return nil
}
