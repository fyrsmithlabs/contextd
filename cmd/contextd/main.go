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
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/compression"
	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/folding"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	httpserver "github.com/fyrsmithlabs/contextd/internal/http"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/mcp"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/services"
	"github.com/fyrsmithlabs/contextd/internal/telemetry"
	"github.com/fyrsmithlabs/contextd/internal/tenant"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// Version information (set at build time via ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// foldingScrubberAdapter adapts secrets.Scrubber to folding.SecretScrubber interface.
type foldingScrubberAdapter struct {
	scrubber secrets.Scrubber
}

// Scrub implements folding.SecretScrubber.
func (a *foldingScrubberAdapter) Scrub(content string) (string, error) {
	result := a.scrubber.Scrub(content)
	return result.Scrubbed, nil
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
	httpPort := flag.Int("http-port", 0, "HTTP server port (overrides config, default: 9090)")
	httpHost := flag.String("http-host", "", "HTTP server host (overrides config, default: localhost)")
	noHTTP := flag.Bool("no-http", false, "disable HTTP server (allows multiple instances)")
	mcpMode := flag.Bool("mcp", false, "run in MCP mode (stdio transport)")
	downloadModels := flag.Bool("download-models", false, "download embedding models and exit (for airgap/container builds)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("contextd %s (commit: %s, built: %s)\n", version, commit, buildDate)
		return nil
	}

	// Handle model download mode (for container builds)
	if *downloadModels {
		return downloadEmbeddingModels()
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
	defer func() { _ = logger.Sync() }()

	logger.Info(ctx, "starting contextd",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_date", buildDate),
		zap.Bool("mcp_mode", *mcpMode),
	)

	// ============================================================================
	// Load Configuration (before telemetry so we can use config values)
	// ============================================================================
	// Ensure config directory exists for new users
	if err := config.EnsureConfigDir(); err != nil {
		logger.Warn(ctx, "failed to create config directory", zap.Error(err))
	}

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
	// Initialize Telemetry (using config values)
	// ============================================================================
	telCfg := telemetry.NewDefaultConfig()
	telCfg.ServiceName = cfg.Observability.ServiceName
	telCfg.ServiceVersion = version
	telCfg.Enabled = cfg.Observability.EnableTelemetry
	if cfg.Observability.OTLPEndpoint != "" {
		telCfg.Endpoint = cfg.Observability.OTLPEndpoint
	}
	if cfg.Observability.OTLPProtocol != "" {
		telCfg.Protocol = cfg.Observability.OTLPProtocol
	}
	// Only override insecure/TLS settings if protocol is set (indicates intentional config)
	if cfg.Observability.OTLPProtocol != "" {
		telCfg.Insecure = cfg.Observability.OTLPInsecure
		telCfg.TLSSkipVerify = cfg.Observability.OTLPTLSSkipVerify
	}
	// Environment variables can still override config file
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
		logger.Info(ctx, "telemetry initialized",
			zap.Bool("enabled", telCfg.Enabled),
			zap.String("endpoint", telCfg.Endpoint),
			zap.String("protocol", telCfg.Protocol),
			zap.Bool("insecure", telCfg.Insecure),
			zap.String("service_name", telCfg.ServiceName),
		)
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
	// Initialize Infrastructure (VectorStore + Embeddings)
	// ============================================================================
	var store vectorstore.Store
	var embeddingProvider embeddings.Provider

	// Initialize embeddings provider using config values
	embeddingCfg := embeddings.ProviderConfig{
		Provider: cfg.Embeddings.Provider,
		Model:    cfg.Embeddings.Model,
		BaseURL:  cfg.Embeddings.BaseURL,
		CacheDir: cfg.Embeddings.CacheDir,
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

		// Get provider dimension and update config
		providerDim := embeddingProvider.Dimension()
		cfg.VectorStore.Chromem.VectorSize = providerDim

		logger.Info(ctx, "embeddings provider initialized",
			zap.String("provider", cfg.Embeddings.Provider),
			zap.String("model", cfg.Embeddings.Model),
			zap.Int("dimension", providerDim),
		)

		// Initialize vectorstore using factory
		store, err = vectorstore.NewStore(cfg, embeddingProvider, logger.Underlying())
		if err != nil {
			logger.Warn(ctx, "vectorstore initialization failed",
				zap.String("provider", cfg.VectorStore.Provider),
				zap.Error(err),
			)
		} else {
			defer store.Close()
			logger.Info(ctx, "vectorstore initialized",
				zap.String("provider", cfg.VectorStore.Provider),
			)
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
	// TODO: Migrate to StoreProvider for database-per-project isolation
	if store != nil {
		checkpointCfg := checkpoint.DefaultServiceConfig()
		checkpointSvc, err = checkpoint.NewServiceWithStore(checkpointCfg, store, logger.Underlying())
		if err != nil {
			logger.Warn(ctx, "checkpoint service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "checkpoint service initialized")
		}
	}

	// Initialize remediation service
	if store != nil {
		remediationCfg := remediation.DefaultServiceConfig()
		remediationSvc, err = remediation.NewService(remediationCfg, store, logger.Underlying())
		if err != nil {
			logger.Warn(ctx, "remediation service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "remediation service initialized")
		}
	}

	// Initialize repository service (depends on vectorstore)
	if store != nil {
		repositorySvc = repository.NewService(store)
		logger.Info(ctx, "repository service initialized")
	}

	// Initialize troubleshoot service
	if store != nil {
		troubleshootAdapter := vectorstore.NewTroubleshootAdapter(store)
		troubleshootSvc, err = troubleshoot.NewService(troubleshootAdapter, logger.Underlying(), nil)
		if err != nil {
			logger.Warn(ctx, "troubleshoot service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "troubleshoot service initialized")
		}
	}

	// Initialize reasoningbank service
	var distillerSvc *reasoningbank.Distiller
	if store != nil {
		// Build service options
		rbOpts := []reasoningbank.ServiceOption{
			reasoningbank.WithDefaultTenant(tenant.GetDefaultTenantID()),
		}

		// Enable session granularity if configured
		if cfg.ReasoningBank.Granularity == "session" {
			extractor := reasoningbank.NewSimpleExtractor()
			rbOpts = append(rbOpts, reasoningbank.WithSessionGranularity(
				extractor, logger.Underlying(), cfg.ReasoningBank.MaxBufferedTurns))
			logger.Info(ctx, "reasoningbank session granularity enabled",
				zap.Int("max_buffered_turns", cfg.ReasoningBank.MaxBufferedTurns))
		}

		reasoningbankSvc, err = reasoningbank.NewService(store, logger.Underlying(), rbOpts...)
		if err != nil {
			logger.Warn(ctx, "reasoningbank service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "reasoningbank service initialized",
				zap.String("granularity", cfg.ReasoningBank.Granularity))

			// Initialize distiller for memory consolidation
			distillerSvc, err = reasoningbank.NewDistiller(reasoningbankSvc, logger.Underlying())
			if err != nil {
				logger.Warn(ctx, "distiller initialization failed", zap.Error(err))
			} else {
				logger.Info(ctx, "distiller initialized")
			}
		}
	}

	// Initialize folding service (context-folding for branch/return)
	var foldingSvc *folding.BranchManager
	{
		// Create folding dependencies
		foldingEmitter := folding.NewSimpleEventEmitter()
		foldingBudget := folding.NewBudgetTracker(foldingEmitter)
		foldingRepo := folding.NewMemoryBranchRepository()
		foldingScrubber := &foldingScrubberAdapter{scrubber: scrubber}
		foldingConfig := folding.DefaultFoldingConfig()

		// Create the branch manager with OTEL metrics
		foldingMetrics, _ := folding.NewMetrics(nil) // uses global meter provider
		foldingLogger := folding.NewLogger(logger.Underlying())
		foldingSvc = folding.NewBranchManager(
			foldingRepo,
			foldingBudget,
			foldingScrubber,
			foldingEmitter,
			foldingConfig,
			folding.WithMetrics(foldingMetrics),
			folding.WithLogger(foldingLogger),
		)
		logger.Info(ctx, "folding service initialized",
			zap.Int("max_depth", foldingConfig.MaxDepth),
			zap.Int("default_budget", foldingConfig.DefaultBudget),
		)
	}

	// Initialize compression service
	var compressionSvc *compression.Service
	{
		compressionCfg := compression.Config{
			DefaultAlgorithm:  compression.AlgorithmHybrid,
			TargetRatio:       2.0,
			QualityThreshold:  0.7,
			MaxProcessingTime: 30 * time.Second,
		}
		compressionSvc, err = compression.NewService(compressionCfg)
		if err != nil {
			logger.Warn(ctx, "compression service initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "compression service initialized")
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
		Distiller:    distillerSvc,
		Scrubber:     scrubber,
		Compression:  compressionSvc,
		VectorStore:  store,
	})
	logger.Info(ctx, "services registry initialized")

	// ============================================================================
	// Initialize Consolidation Scheduler (if enabled in config)
	// ============================================================================
	var consolidationScheduler *reasoningbank.ConsolidationScheduler
	if cfg.ConsolidationScheduler.Enabled && distillerSvc != nil {
		// Create consolidation options from config
		consolidationOpts := reasoningbank.ConsolidationOptions{
			SimilarityThreshold: cfg.ConsolidationScheduler.SimilarityThreshold,
			DryRun:              false,
			ForceAll:            false,
			MaxClustersPerRun:   0, // No limit
		}

		// Create scheduler with configured interval
		consolidationScheduler, err = reasoningbank.NewConsolidationScheduler(
			distillerSvc,
			logger.Underlying(),
			reasoningbank.WithInterval(cfg.ConsolidationScheduler.Interval),
			reasoningbank.WithConsolidationOptions(consolidationOpts),
			// Note: WithProjectIDs should be configured in config file or via MCP
		)
		if err != nil {
			logger.Warn(ctx, "consolidation scheduler initialization failed", zap.Error(err))
		} else {
			logger.Info(ctx, "consolidation scheduler initialized",
				zap.Duration("interval", cfg.ConsolidationScheduler.Interval),
				zap.Float64("threshold", cfg.ConsolidationScheduler.SimilarityThreshold),
			)

			// Start the scheduler
			if err := consolidationScheduler.Start(); err != nil {
				logger.Warn(ctx, "failed to start consolidation scheduler", zap.Error(err))
			} else {
				logger.Info(ctx, "consolidation scheduler started")
			}
		}
	} else if cfg.ConsolidationScheduler.Enabled {
		logger.Warn(ctx, "consolidation scheduler enabled but distiller not available")
	}

	// ============================================================================
	// Initialize HTTP Server (unless --no-http)
	// ============================================================================
	var httpSrv *httpserver.Server
	var httpErrChan chan error
	var httpServerHost string
	var httpServerPort int
	var bgScanner *vectorstore.BackgroundScanner

	if !*noHTTP {
		// Determine HTTP server configuration (flags override config)
		httpServerHost = "localhost"
		if *httpHost != "" {
			httpServerHost = *httpHost
		}

		httpServerPort = cfg.Server.Port
		if *httpPort != 0 {
			httpServerPort = *httpPort
		}

		// Create metadata health checker for vectorstore monitoring
		var healthChecker *vectorstore.MetadataHealthChecker
		if cfg.VectorStore.Provider == "chromem" && cfg.VectorStore.Chromem.Path != "" {
			// Expand the path (handles ~ for home directory)
			expandedPath := os.ExpandEnv(cfg.VectorStore.Chromem.Path)
			if strings.HasPrefix(expandedPath, "~/") {
				home, err := os.UserHomeDir()
				if err == nil {
					expandedPath = filepath.Join(home, expandedPath[2:])
				}
			}
			healthChecker = vectorstore.NewMetadataHealthChecker(expandedPath, logger.Underlying())
			logger.Info(ctx, "metadata health checker initialized",
				zap.String("path", expandedPath))

			// Run startup validation (pre-flight health checks)
			validationCfg := &vectorstore.StartupValidationConfig{
				FailOnCorruption: false, // Continue with graceful degradation by default
				FailOnDegraded:   false, // Don't block on empty collections
			}
			result, err := vectorstore.ValidateStartup(ctx, healthChecker, validationCfg, logger.Underlying())
			if err != nil {
				logger.Error(ctx, "startup validation failed",
					zap.Error(err))
				// Note: We don't return here because FailOnCorruption=false
				// The error is only returned when FailOnCorruption=true
			} else if result != nil && !result.Passed {
				logger.Warn(ctx, "startup validation completed with warnings",
					zap.Int("warnings", result.WarningCount),
					zap.Int("errors", result.ErrorCount))
			}
		}

		httpCfg := &httpserver.Config{
			Host:          httpServerHost,
			Port:          httpServerPort,
			Version:       version,
			HealthChecker: healthChecker,
		}

		var err error
		httpSrv, err = httpserver.NewServer(registry, logger.Underlying(), httpCfg)
		if err != nil {
			return fmt.Errorf("initializing HTTP server: %w", err)
		}
		logger.Info(ctx, "HTTP server initialized",
			zap.String("host", httpServerHost),
			zap.Int("port", httpServerPort),
		)

		// Start background health scanner if health checker is available
		if healthChecker != nil {
			scannerCfg := &vectorstore.BackgroundScannerConfig{
				Interval: 5 * time.Minute, // Default: scan every 5 minutes
				OnDegraded: func(health *vectorstore.MetadataHealth) {
					logger.Warn(ctx, "background scanner detected degraded state",
						zap.Int("corrupt_count", health.CorruptCount),
						zap.Strings("corrupt_hashes", health.Corrupt))
				},
				OnRecovered: func(health *vectorstore.MetadataHealth) {
					logger.Info(ctx, "background scanner detected recovery",
						zap.Int("healthy_count", health.HealthyCount))
				},
			}
			bgScanner = vectorstore.NewBackgroundScanner(healthChecker, scannerCfg, logger.Underlying())
			bgScanner.Start(ctx)
			logger.Info(ctx, "background health scanner started",
				zap.Duration("interval", scannerCfg.Interval))
		}

		// Start HTTP server in background goroutine
		httpErrChan = make(chan error, 1)
		go func() {
			if err := httpSrv.Start(); err != nil {
				httpErrChan <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()
	} else {
		logger.Info(ctx, "HTTP server disabled (--no-http)")
	}

	// ============================================================================
	// Initialize MCP Server (if all services available)
	// ============================================================================
	var mcpServer *mcp.Server
	var mcpErrChan chan error
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
			foldingSvc,
			registry.Distiller(),
			scrubber,
		)
		if err != nil {
			return fmt.Errorf("initializing MCP server: %w", err)
		}
		defer mcpServer.Close()

		logger.Info(ctx, "MCP server initialized, starting stdio transport")

		// Run MCP server in background goroutine (no longer blocks)
		mcpErrChan = make(chan error, 1)
		go func() {
			if err := mcpServer.Run(ctx); err != nil {
				mcpErrChan <- fmt.Errorf("MCP server error: %w", err)
			}
			close(mcpErrChan)
		}()
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
	if foldingSvc != nil {
		serviceStatus = append(serviceStatus, "folding:ok")
	} else {
		serviceStatus = append(serviceStatus, "folding:unavailable")
	}
	if compressionSvc != nil {
		serviceStatus = append(serviceStatus, "compression:ok")
	} else {
		serviceStatus = append(serviceStatus, "compression:unavailable")
	}

	if httpSrv != nil {
		logger.Info(ctx, "contextd initialized",
			zap.String("http_host", httpServerHost),
			zap.Int("http_port", httpServerPort),
			zap.Strings("services", serviceStatus),
		)
	} else {
		logger.Info(ctx, "contextd initialized (HTTP disabled)",
			zap.Strings("services", serviceStatus),
		)
	}

	// Wait for shutdown signal, HTTP server error, or MCP server error
	// Use a goroutine to forward httpErrChan to avoid nil channel select
	combinedErrChan := make(chan error, 1)
	if httpErrChan != nil {
		go func() {
			if err := <-httpErrChan; err != nil {
				combinedErrChan <- err
			}
		}()
	}

	select {
	case <-ctx.Done():
		logger.Info(ctx, "shutdown signal received")
	case err := <-combinedErrChan:
		logger.Error(ctx, "HTTP server error", zap.Error(err))
		return err
	case err, ok := <-mcpErrChan:
		if ok && err != nil {
			logger.Error(ctx, "MCP server error", zap.Error(err))
			return err
		}
		// MCP server exited cleanly (e.g., stdin closed)
		logger.Info(ctx, "MCP server exited")
	}

	logger.Info(ctx, "shutting down contextd")

	// Gracefully stop consolidation scheduler (if running)
	if consolidationScheduler != nil {
		if err := consolidationScheduler.Stop(); err != nil {
			logger.Error(ctx, "consolidation scheduler shutdown error", zap.Error(err))
		} else {
			logger.Info(ctx, "consolidation scheduler stopped")
		}
	}

	// Stop background health scanner (if running)
	if bgScanner != nil {
		bgScanner.Stop()
		logger.Info(ctx, "background health scanner stopped")
	}

	// Gracefully shutdown HTTP server (if running)
	if httpSrv != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer shutdownCancel()

		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error(ctx, "HTTP server shutdown error", zap.Error(err))
		} else {
			logger.Info(ctx, "HTTP server stopped")
		}
	}

	logger.Info(ctx, "contextd stopped")
	return nil
}

// downloadEmbeddingModels downloads the FastEmbed models for airgap/container builds.
// This is called with --download-models flag during Docker build or for local setup.
func downloadEmbeddingModels() error {
	fmt.Println("Downloading embedding models...")

	// Get model from environment or use default
	model := os.Getenv("EMBEDDINGS_MODEL")
	if model == "" {
		model = "BAAI/bge-small-en-v1.5"
	}

	// Get cache directory from environment or use default (~/.config/contextd/models)
	cacheDir := os.Getenv("EMBEDDINGS_CACHE_DIR")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".config", "contextd", "models")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Cache directory: %s\n", cacheDir)

	// Initialize the FastEmbed provider - this triggers model download
	cfg := embeddings.ProviderConfig{
		Provider: "fastembed",
		Model:    model,
		CacheDir: cacheDir,
	}

	provider, err := embeddings.NewProvider(cfg)
	if err != nil {
		return fmt.Errorf("initializing embedding provider: %w", err)
	}
	defer provider.Close()

	// Test embedding generation to verify model works
	testVec, err := provider.EmbedQuery(context.Background(), "test")
	if err != nil {
		return fmt.Errorf("testing embedding generation: %w", err)
	}

	fmt.Printf("Model downloaded successfully (dimension: %d)\n", len(testVec))
	return nil
}
