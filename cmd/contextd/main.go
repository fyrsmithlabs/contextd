// Contextd is a context daemon for Claude Code with HTTP/SSE transport.
//
// This binary starts the contextd HTTP server with full service initialization,
// including NATS, Qdrant, embeddings, and MCP endpoints.
//
// Configuration is loaded from environment variables. See pkg/config for details.
//
// Usage:
//
//	# Start server with defaults
//	contextd
//
//	# Configure via environment
//	SERVER_PORT=9090 QDRANT_URL=http://localhost:6333 contextd
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/checkpoint"
	"github.com/fyrsmithlabs/contextd/pkg/config"
	"github.com/fyrsmithlabs/contextd/pkg/embeddings"
	"github.com/fyrsmithlabs/contextd/pkg/mcp"
	"github.com/fyrsmithlabs/contextd/pkg/remediation"
	"github.com/fyrsmithlabs/contextd/pkg/server"
	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

func main() {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Run server
	if err := run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server shutdown complete")
}

// run starts the contextd server and blocks until context is cancelled.
//
// This function initializes all dependencies and services:
//  1. Loads and validates configuration
//  2. Initializes logger and telemetry
//  3. Connects to infrastructure (NATS, Qdrant)
//  4. Creates embedding service
//  5. Initializes business services (Checkpoint, Remediation)
//  6. Wires MCP server with all services
//  7. Starts HTTP server
//  8. Performs graceful shutdown on context cancellation
//
// Returns http.ErrServerClosed on graceful shutdown.
func run(ctx context.Context) error {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		_ = logger.Sync() // Best-effort sync on shutdown
	}()

	logger.Info("Starting contextd",
		zap.Int("port", cfg.Server.Port),
		zap.String("service", cfg.Observability.ServiceName),
		zap.Duration("shutdown_timeout", cfg.Server.ShutdownTimeout))

	// Initialize infrastructure dependencies
	deps, err := initDependencies(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	defer deps.Close()

	logger.Info("Dependencies initialized",
		zap.Bool("nats_connected", deps.natsConn != nil),
		zap.Bool("vectorstore_ready", deps.vectorStore != nil))

	// Initialize business services
	services, err := initServices(deps, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	logger.Info("Services initialized",
		zap.Bool("checkpoint_service", services.checkpointSvc != nil),
		zap.Bool("remediation_service", services.remediationSvc != nil))

	// Create HTTP server
	srv := server.NewServer(cfg)

	// Create MCP server and register routes
	mcpServer := mcp.NewServer(
		srv.Echo(),
		deps.operations,
		deps.natsConn,
		services.checkpointSvc,
		services.remediationSvc,
		logger,
	)

	// Register MCP routes
	mcpServer.RegisterRoutes()

	// Initialize prefetch engine (if enabled)
	if err := mcpServer.InitializePrefetch(&cfg.PreFetch, logger); err != nil {
		logger.Warn("Failed to initialize prefetch engine",
			zap.Error(err))
	} else if cfg.PreFetch.Enabled {
		logger.Info("Prefetch engine initialized",
			zap.Duration("cache_ttl", cfg.PreFetch.CacheTTL),
			zap.Int("max_entries", cfg.PreFetch.CacheMaxEntries))
	}

	// Register metrics endpoint
	srv.Echo().GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	logger.Info("Server configured",
		zap.String("health_endpoint", fmt.Sprintf("http://localhost:%d/health", cfg.Server.Port)),
		zap.String("mcp_prefix", "/mcp"),
		zap.String("metrics_endpoint", "/metrics"))

	// Start server (blocks until context cancellation)
	return srv.Start(ctx)
}

// dependencies holds all infrastructure dependencies.
type dependencies struct {
	natsConn    *nats.Conn
	vectorStore *vectorstore.Service
	operations  *mcp.OperationRegistry
	logger      *zap.Logger
}

// Close releases all infrastructure resources.
func (d *dependencies) Close() {
	if d.natsConn != nil {
		d.natsConn.Close()
	}
	if d.logger != nil {
		_ = d.logger.Sync() // Best-effort sync
	}
}

// services holds all business services.
type services struct {
	checkpointSvc  *checkpoint.Service
	remediationSvc *remediation.Service
}

// initLogger initializes the structured logger.
func initLogger(cfg *config.Config) (*zap.Logger, error) {
	// Use production logger for non-development environments
	if cfg.Observability.EnableTelemetry {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

// initDependencies initializes all infrastructure dependencies.
//
// This function:
//  1. Connects to NATS for operation tracking
//  2. Creates vector store with Qdrant + embedder
//  3. Initializes operation registry
func initDependencies(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*dependencies, error) {
	// Connect to NATS
	natsURL := getEnvOrDefault("NATS_URL", "nats://localhost:4222")
	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(1*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", natsURL, err)
	}

	logger.Info("Connected to NATS", zap.String("url", natsURL))

	// Create JetStream context for operation tracking (verify it works)
	_, err = nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create operation registry
	operations := mcp.NewOperationRegistry(nc)

	// Initialize embedding service (TEI or OpenAI)
	embeddingConfig := embeddings.ConfigFromEnv()
	embeddingSvc, err := embeddings.NewService(embeddingConfig)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create embedding service: %w", err)
	}

	logger.Info("Embedding service initialized",
		zap.String("base_url", embeddingConfig.BaseURL),
		zap.String("model", embeddingConfig.Model))

	// Initialize vector store with Qdrant
	qdrantURL := getEnvOrDefault("QDRANT_URL", "http://localhost:6333")
	vsConfig := vectorstore.Config{
		URL:            qdrantURL,
		CollectionName: "contextd", // Base collection name
		Embedder:       embeddingSvc.Embedder(),
	}

	vectorStore, err := vectorstore.NewService(vsConfig)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	logger.Info("Vector store initialized",
		zap.String("url", qdrantURL),
		zap.String("collection", vsConfig.CollectionName))

	return &dependencies{
		natsConn:    nc,
		vectorStore: vectorStore,
		operations:  operations,
		logger:      logger,
	}, nil
}

// initServices initializes all business services.
//
// This function creates checkpoint and remediation services
// with the initialized vector store.
func initServices(deps *dependencies, logger *zap.Logger) (*services, error) {
	// Create checkpoint service
	checkpointSvc := checkpoint.NewService(deps.vectorStore, logger)

	// Create remediation service
	remediationSvc := remediation.NewService(deps.vectorStore, logger)

	return &services{
		checkpointSvc:  checkpointSvc,
		remediationSvc: remediationSvc,
	}, nil
}

// getEnvOrDefault gets environment variable or returns default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
