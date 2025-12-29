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
	"regexp"
	"context"
	"net"
	"strings"
	"sync"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/time/rate"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/workflows"
)

// Validation regexes compiled once at package initialization
var (
	validNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	validSHARegex  = regexp.MustCompile(`^[0-9a-f]{40}$`)
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
	gitHubToken    config.Secret
	logger         *logging.Logger
	rateLimiters   map[string]*rate.Limiter
	mu             sync.RWMutex
	lastCleanup    time.Time
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
		gitHubToken:    cfg.GitHubToken,
		logger:         logger,
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", server.handleWebhook)
	mux.HandleFunc("/health", handleHealth)

	// Create HTTP server with timeouts to prevent slowloris attacks
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
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


// getRateLimiter returns a rate limiter for the given IP address.
// Rate limit: 60 requests per minute per IP address.
func (s *WebhookServer) getRateLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Initialize map if needed
	if s.rateLimiters == nil {
		s.rateLimiters = make(map[string]*rate.Limiter)
		s.lastCleanup = time.Now()
	}

	// Clean up old limiters every hour to prevent memory leaks
	if time.Since(s.lastCleanup) > time.Hour {
		s.rateLimiters = make(map[string]*rate.Limiter)
		s.lastCleanup = time.Now()
	}

	// Get or create limiter for this IP
	limiter, exists := s.rateLimiters[ip]
	if !exists {
		// 60 requests per minute = 1 per second with burst of 10
		limiter = rate.NewLimiter(rate.Limit(1), 10)
		s.rateLimiters[ip] = limiter
	}

	return limiter
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in the comma-separated list
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Rate limiting: Check if this IP has exceeded the rate limit
	clientIP := getClientIP(r)
	limiter := s.getRateLimiter(clientIP)
	if !limiter.Allow() {
		s.logger.Warn(ctx, "rate limit exceeded", zap.String("ip", clientIP))
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Limit request body size to prevent DoS attacks (1MB max)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// validatePREvent validates PR event data to prevent injection attacks
func validatePREvent(e *github.PullRequestEvent) error {
	// Validate PR number
	if e.PullRequest == nil || e.PullRequest.Number == nil || *e.PullRequest.Number <= 0 {
		return fmt.Errorf("invalid PR number")
	}
	
	// Validate owner and repo names (alphanumeric, hyphens, underscores, dots)
	
	if e.Repo == nil || e.Repo.Owner == nil || e.Repo.Owner.Login == nil {
		return fmt.Errorf("invalid repository owner")
	}
	if !validNameRegex.MatchString(*e.Repo.Owner.Login) {
		return fmt.Errorf("invalid repository owner format")
	}
	
	if e.Repo.Name == nil {
		return fmt.Errorf("invalid repository name")
	}
	if !validNameRegex.MatchString(*e.Repo.Name) {
		return fmt.Errorf("invalid repository name format")
	}
	
	// Validate SHA format (40-character hex string)
	if e.PullRequest.Head == nil || e.PullRequest.Head.SHA == nil {
		return fmt.Errorf("invalid PR head SHA")
	}
	if !validSHARegex.MatchString(*e.PullRequest.Head.SHA) {
		return fmt.Errorf("invalid SHA format")
	}
	
	return nil
}

func (s *WebhookServer) handlePullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	// Validate PR event data to prevent injection attacks
	if err := validatePREvent(event); err != nil {
		s.logger.Warn(ctx, "invalid PR event data", zap.Error(err))
		return fmt.Errorf("invalid PR event: %w", err)
	}

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
		Owner:       repo.GetOwner().GetLogin(),
		Repo:        repo.GetName(),
		PRNumber:    pr.GetNumber(),
		BaseBranch:  pr.GetBase().GetRef(),
		HeadBranch:  pr.GetHead().GetRef(),
		HeadSHA:     pr.GetHead().GetSHA(),
		GitHubToken: s.gitHubToken,
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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
