// Package main provides a Temporal worker for plugin validation workflows.
//
// This worker listens for plugin validation workflows triggered by GitHub webhooks
// and executes them using Temporal's durable execution engine.
//
// Usage:
//
//	TEMPORAL_HOST=localhost:7233 \
//	GITHUB_TOKEN=ghp_xxx \
//	./plugin-validator
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/workflows"
)

// Config holds worker configuration.
type Config struct {
	TemporalHost string
	GitHubToken  config.Secret
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Create root context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize logging
	logCfg := logging.NewDefaultConfig()
	logger, err := logging.NewLogger(logCfg, nil)
	if err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	// Load configuration
	cfg := loadConfig()

	logger.Info(ctx, "plugin validation worker starting",
		zap.String("temporal_host", cfg.TemporalHost),
	)

	// Validate configuration
	if !cfg.GitHubToken.IsSet() {
		return fmt.Errorf("GITHUB_TOKEN not set")
	}

	// Note: GitHub token is passed to workflows via config, not set globally

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: cfg.TemporalHost,
	})
	if err != nil {
		return fmt.Errorf("unable to create Temporal client: %w", err)
	}
	defer c.Close()

	logger.Info(ctx, "temporal client connected", zap.String("host", cfg.TemporalHost))

	// Create worker
	w := worker.New(c, "plugin-validation-queue", worker.Options{})

	// Register workflow
	w.RegisterWorkflow(workflows.PluginUpdateValidationWorkflow)

	// Register activities
	w.RegisterActivity(workflows.FetchPRFilesActivity)
	w.RegisterActivity(workflows.CategorizeFilesActivity)
	w.RegisterActivity(workflows.ValidatePluginSchemasActivity)
	w.RegisterActivity(workflows.PostReminderCommentActivity)
	w.RegisterActivity(workflows.PostSuccessCommentActivity)

	logger.Info(ctx, "worker configured",
		zap.String("task_queue", "plugin-validation-queue"),
	)

	// Start worker in background
	workerErrors := make(chan error, 1)
	go func() {
		logger.Info(ctx, "worker starting")
		workerErrors <- w.Run(worker.InterruptCh())
	}()

	// Wait for shutdown signal or worker error
	select {
	case err := <-workerErrors:
		if err != nil {
			return fmt.Errorf("worker error: %w", err)
		}
	case <-ctx.Done():
		logger.Info(ctx, "shutdown signal received")
	}

	// Worker stops automatically on interrupt signal
	logger.Info(ctx, "worker stopped gracefully")
	return nil
}

func loadConfig() *Config {
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	return &Config{
		TemporalHost: temporalHost,
		GitHubToken:  config.Secret(os.Getenv("GITHUB_TOKEN")),
	}
}
