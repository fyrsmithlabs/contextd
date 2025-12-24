// Package main provides a GitHub webhook server that triggers Temporal workflows.
//
// This server receives GitHub webhook events and triggers corresponding Temporal
// workflows for plugin validation and other automation tasks.
//
// Usage:
//
//	TEMPORAL_HOST=localhost:7233 \
//	GITHUB_WEBHOOK_SECRET=your_secret \
//	GITHUB_TOKEN=ghp_xxx \
//	PORT=3000 \
//	./github-webhook
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/go-github/v57/github"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/workflows"
)

// Config holds webhook server configuration.
type Config struct {
	TemporalHost  string
	WebhookSecret config.Secret
	GitHubToken   config.Secret
	Port          string
}

type WebhookServer struct {
	temporalClient client.Client
	webhookSecret  config.Secret
	logger         *logging.Logger
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
	defer logger.Sync()

	// Load configuration from environment
	cfg := loadConfig()

	logger.Info(ctx, "github webhook server starting",
		zap.String("port", cfg.Port),
		zap.String("temporal_host", cfg.TemporalHost),
	)

	// Validate configuration
	if !cfg.WebhookSecret.IsSet() {
		return fmt.Errorf("GITHUB_WEBHOOK_SECRET not set")
	}
	if !cfg.GitHubToken.IsSet() {
		return fmt.Errorf("GITHUB_TOKEN not set")
	}

	// Set GitHub token for workflow activities
	workflows.SetGitHubToken(cfg.GitHubToken)

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: cfg.TemporalHost,
	})
	if err != nil {
		return fmt.Errorf("unable to create Temporal client: %w", err)
	}
	defer c.Close()

	logger.Info(ctx, "temporal client connected", zap.String("host", cfg.TemporalHost))

	// Create webhook server
	server := &WebhookServer{
		temporalClient: c,
		webhookSecret:  cfg.WebhookSecret,
		logger:         logger,
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", server.handleWebhook)
	mux.HandleFunc("/health", handleHealth)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Start server in background
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(ctx, "HTTP server listening", zap.String("addr", httpServer.Addr))
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		logger.Info(ctx, "shutdown signal received")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "server shutdown error", zap.Error(err))
		return err
	}

	logger.Info(ctx, "server stopped gracefully")
	return nil
}

func loadConfig() *Config {
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	return &Config{
		TemporalHost:  temporalHost,
		WebhookSecret: config.Secret(os.Getenv("GITHUB_WEBHOOK_SECRET")),
		GitHubToken:   config.Secret(os.Getenv("GITHUB_TOKEN")),
		Port:          port,
	}
}

func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate webhook signature
	payload, err := github.ValidatePayload(r, []byte(s.webhookSecret.Value()))
	if err != nil {
		s.logger.Warn(ctx, "invalid webhook signature", zap.Error(err))
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse webhook event
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		s.logger.Warn(ctx, "failed to parse webhook", zap.Error(err))
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Handle different event types
	switch e := event.(type) {
	case *github.PullRequestEvent:
		if err := s.handlePullRequestEvent(ctx, e); err != nil {
			s.logger.Error(ctx, "error handling PR event", zap.Error(err))
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

	default:
		s.logger.Debug(ctx, "ignoring event type", zap.String("type", fmt.Sprintf("%T", event)))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *WebhookServer) handlePullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	// Only trigger on opened, synchronize (new commits), and reopened
	action := event.GetAction()
	if action != "opened" && action != "synchronize" && action != "reopened" {
		s.logger.Debug(ctx, "ignoring PR action", zap.String("action", action))
		return nil
	}

	pr := event.GetPullRequest()
	repo := event.GetRepo()

	s.logger.Info(ctx, "processing PR event",
		zap.Int("pr_number", pr.GetNumber()),
		zap.String("owner", repo.GetOwner().GetLogin()),
		zap.String("repo", repo.GetName()),
		zap.String("action", action),
	)

	// Create workflow config
	config := workflows.PluginUpdateValidationConfig{
		Owner:      repo.GetOwner().GetLogin(),
		Repo:       repo.GetName(),
		PRNumber:   pr.GetNumber(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadBranch: pr.GetHead().GetRef(),
		HeadSHA:    pr.GetHead().GetSHA(),
	}

	// Start Temporal workflow (use commit SHA for idempotency)
	workflowID := fmt.Sprintf("plugin-validation-%s-%s-pr-%d-%s",
		config.Owner,
		config.Repo,
		config.PRNumber,
		config.HeadSHA)

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "plugin-validation-queue",
	}

	workflowCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	we, err := s.temporalClient.ExecuteWorkflow(workflowCtx, options, workflows.PluginUpdateValidationWorkflow, config)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	s.logger.Info(ctx, "workflow started",
		zap.String("workflow_id", we.GetID()),
		zap.String("run_id", we.GetRunID()),
	)
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
