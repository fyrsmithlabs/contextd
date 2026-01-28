// Package http provides HTTP API for contextd.
package http

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	"github.com/fyrsmithlabs/contextd/internal/services"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	// CheckpointNameMaxLength is the UI display limit for checkpoint names.
	CheckpointNameMaxLength = 50
	// CheckpointNameTruncationSuffix is added when names are truncated.
	CheckpointNameTruncationSuffix = "..."
	// MaxSummaryLength is the maximum length for summary fields.
	MaxSummaryLength = 10000
	// MaxContextLength is the maximum length for context fields.
	MaxContextLength = 50000
	// MinThresholdPercent is the minimum valid threshold percentage.
	MinThresholdPercent = 1
	// MaxThresholdPercent is the maximum valid threshold percentage.
	MaxThresholdPercent = 100
)

// Server provides HTTP endpoints for contextd.
type Server struct {
	echo          *echo.Echo
	registry      services.Registry
	logger        *zap.Logger
	config        *Config
	healthChecker *vectorstore.MetadataHealthChecker
	metrics       *HTTPMetrics
}

// Config holds HTTP server configuration.
type Config struct {
	Host          string
	Port          int
	Version       string
	HealthChecker *vectorstore.MetadataHealthChecker // Optional metadata health checker
}

// NewServer creates a new HTTP server.
func NewServer(registry services.Registry, logger *zap.Logger, cfg *Config) (*Server, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required for request tracking and debugging")
	}
	if cfg == nil {
		cfg = &Config{
			Host: "localhost",
			Port: 9090,
		}
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Create metrics
	httpMetrics := NewHTTPMetrics(logger)

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(httpMetrics.MetricsMiddleware()) // OTEL metrics
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			logger.Info("http request",
				zap.String("method", c.Request().Method),
				zap.String("uri", c.Request().RequestURI),
				zap.Int("status", c.Response().Status),
				zap.Duration("duration", duration),
				zap.String("request_id", c.Response().Header().Get(echo.HeaderXRequestID)),
			)

			return err
		}
	})

	s := &Server{
		echo:          e,
		registry:      registry,
		logger:        logger,
		config:        cfg,
		healthChecker: cfg.HealthChecker,
		metrics:       httpMetrics,
	}

	// Register routes
	s.registerRoutes()

	return s, nil
}

// registerRoutes sets up the HTTP endpoints.
func (s *Server) registerRoutes() {
	// Health check
	s.echo.GET("/health", s.handleHealth)

	// Prometheus metrics endpoint
	s.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// API v1 routes
	v1 := s.echo.Group("/api/v1")
	v1.POST("/scrub", s.handleScrub)
	v1.POST("/threshold", s.handleThreshold)
	v1.GET("/status", s.handleStatus)
	v1.GET("/health/metadata", s.handleMetadataHealth)

	// Note: Checkpoint management is available via MCP tools (checkpoint_save, checkpoint_list, checkpoint_resume)
	// HTTP endpoints were removed due to security concerns (CVE-2025-CONTEXTD-001)
}

// ScrubRequest is the request body for POST /api/v1/scrub.
type ScrubRequest struct {
	Content string `json:"content"`
}

// ScrubResponse is the response body for POST /api/v1/scrub.
type ScrubResponse struct {
	Content       string `json:"content"`
	FindingsCount int    `json:"findings_count"`
}

// ThresholdRequest is the request body for POST /api/v1/threshold.
type ThresholdRequest struct {
	ProjectID   string `json:"project_id"`
	SessionID   string `json:"session_id"`
	Percent     int    `json:"percent"`
	Summary     string `json:"summary,omitempty"`      // Brief summary of session work (recommended)
	Context     string `json:"context,omitempty"`      // Additional context for resumption
	ProjectPath string `json:"project_path,omitempty"` // Full project path (defaults to project_id)
}

// ThresholdResponse is the response body for POST /api/v1/threshold.
type ThresholdResponse struct {
	CheckpointID string `json:"checkpoint_id"`
	Message      string `json:"message"`
}

// HealthResponse is the response body for GET /health.
type HealthResponse struct {
	Status   string                `json:"status"`
	Metadata *MetadataHealthStatus `json:"metadata,omitempty"` // Optional metadata health
}

// StatusResponse, StatusCounts, ContextStatus, CompressionStatus, and MemoryStatus
// are defined in types.go to enable reuse across packages.

// Note: Checkpoint request/response types removed (CVE-2025-CONTEXTD-001 fix).
// Use MCP tools for checkpoint operations: checkpoint_save, checkpoint_list, checkpoint_resume.

// handleHealth returns a health check response including metadata integrity status.
func (s *Server) handleHealth(c echo.Context) error {
	ctx := c.Request().Context()
	resp := HealthResponse{Status: "ok"}

	// Check metadata health if checker is available
	if s.healthChecker != nil {
		health, err := s.healthChecker.Check(ctx)
		if err != nil {
			s.logger.Warn("metadata health check failed", zap.Error(err))
			// Don't fail the health endpoint, just log the error
		} else {
			resp.Metadata = &MetadataHealthStatus{
				Status:        health.Status(),
				HealthyCount:  health.HealthyCount,
				CorruptCount:  health.CorruptCount,
				EmptyCount:    len(health.Empty),
				Total:         health.Total,
				CorruptHashes: health.Corrupt,
			}

			// Set overall status to degraded if metadata is corrupt
			if !health.IsHealthy() {
				resp.Status = "degraded"
			}
		}
	}

	// Determine HTTP status code based on health
	statusCode := http.StatusOK
	if resp.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, resp)
}

// handleMetadataHealth returns detailed metadata integrity information.
// NOTE: This endpoint exposes collection hashes which are internal identifiers.
// In production, protect this endpoint with authentication/authorization or use
// a reverse proxy to restrict access. The /health endpoint provides summary info only.
func (s *Server) handleMetadataHealth(c echo.Context) error {
	if s.healthChecker == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "metadata health checker not configured")
	}

	ctx := c.Request().Context()
	health, err := s.healthChecker.Check(ctx)
	if err != nil {
		s.logger.Error("metadata health check failed", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "metadata health check failed")
	}

	// Return full health details
	return c.JSON(http.StatusOK, health)
}

// handleStatus returns service status and resource counts.
func (s *Server) handleStatus(c echo.Context) error {
	ctx := c.Request().Context()

	// Build service status map
	services := make(map[string]string)

	// Check checkpoint service
	if s.registry.Checkpoint() != nil {
		services["checkpoint"] = "ok"
	} else {
		services["checkpoint"] = "unavailable"
	}

	// Check memory service (ReasoningBank)
	if s.registry.Memory() != nil {
		services["memory"] = "ok"
	} else {
		services["memory"] = "unavailable"
	}

	// Check remediation service
	if s.registry.Remediation() != nil {
		services["remediation"] = "ok"
	} else {
		services["remediation"] = "unavailable"
	}

	// Check repository service
	if s.registry.Repository() != nil {
		services["repository"] = "ok"
	} else {
		services["repository"] = "unavailable"
	}

	// Check troubleshoot service
	if s.registry.Troubleshoot() != nil {
		services["troubleshoot"] = "ok"
	} else {
		services["troubleshoot"] = "unavailable"
	}

	// Check scrubber
	if s.registry.Scrubber() != nil {
		services["scrubber"] = "ok"
	} else {
		services["scrubber"] = "unavailable"
	}

	// Check compression service
	if s.registry.Compression() != nil {
		services["compression"] = "ok"
	} else {
		services["compression"] = "unavailable"
	}

	// Get counts via VectorStore collections using shared helper
	checkpoints, memories := CountFromCollections(ctx, s.registry.VectorStore())
	counts := StatusCounts{
		Checkpoints: checkpoints,
		Memories:    memories,
	}

	// Build response with optional status fields
	resp := StatusResponse{
		Status:   "ok",
		Version:  s.config.Version,
		Services: services,
		Counts:   counts,
	}

	// Add compression stats if available
	if s.registry.Compression() != nil {
		compStats := s.registry.Compression().Stats()
		resp.Compression = &CompressionStatus{
			LastRatio:       compStats.LastRatio,
			LastQuality:     compStats.LastQuality,
			OperationsTotal: compStats.OperationsTotal,
		}
	}

	// Add memory stats if available
	if s.registry.Memory() != nil {
		memStats := s.registry.Memory().Stats()
		resp.Memory = &MemoryStatus{
			LastConfidence: memStats.LastConfidence,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// handleScrub scrubs secrets from the provided content.
func (s *Server) handleScrub(c echo.Context) error {
	var req ScrubRequest
	if err := c.Bind(&req); err != nil {
		s.logger.Warn("invalid scrub request", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "content field is required")
	}

	// Check if scrubber service is available
	scrubber := s.registry.Scrubber()
	if scrubber == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "scrubber service unavailable")
	}

	// Scrub the content
	result := scrubber.Scrub(req.Content)

	s.logger.Debug("scrubbed content",
		zap.Int("findings", result.TotalFindings),
		zap.Duration("duration", result.Duration),
	)

	return c.JSON(http.StatusOK, ScrubResponse{
		Content:       result.Scrubbed,
		FindingsCount: result.TotalFindings,
	})
}

// handleThreshold handles context threshold reached event.
func (s *Server) handleThreshold(c echo.Context) error {
	var req ThresholdRequest
	if err := c.Bind(&req); err != nil {
		s.logger.Warn("invalid threshold request", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.ProjectID == "" || req.SessionID == "" || req.Percent == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "project_id, session_id, and percent fields are required")
	}

	// Validate percent range
	if req.Percent < MinThresholdPercent || req.Percent > MaxThresholdPercent {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("percent must be between %d and %d", MinThresholdPercent, MaxThresholdPercent))
	}

	// Validate and sanitize summary length
	if len(req.Summary) > MaxSummaryLength {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("summary exceeds maximum length of %d characters", MaxSummaryLength))
	}

	// Validate and sanitize context length
	if len(req.Context) > MaxContextLength {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("context exceeds maximum length of %d characters", MaxContextLength))
	}

	// Use provided values or fall back to defaults
	projectPath := req.ProjectPath
	if projectPath == "" {
		projectPath = req.ProjectID
	}

	// Check for path traversal BEFORE cleaning (Clean removes .. sequences)
	if strings.Contains(projectPath, "..") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project_path: path traversal not allowed")
	}

	// Sanitize project path
	projectPath = filepath.Clean(projectPath)

	summary := req.Summary
	if summary == "" {
		summary = fmt.Sprintf("Context at %d%% threshold", req.Percent)
	}

	name := fmt.Sprintf("Auto-checkpoint at %d%%", req.Percent)
	if req.Summary != "" {
		// Use first N chars of summary as name if provided
		name = req.Summary
		if len(name) > CheckpointNameMaxLength {
			name = name[:CheckpointNameMaxLength-len(CheckpointNameTruncationSuffix)] + CheckpointNameTruncationSuffix
		}
	}

	// Check if checkpoint service is available
	checkpointSvc := s.registry.Checkpoint()
	if checkpointSvc == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "checkpoint service unavailable")
	}

	// Create auto-checkpoint via checkpoint service
	ctx := c.Request().Context()
	chkpt, err := checkpointSvc.Save(ctx, &checkpoint.SaveRequest{
		SessionID:   req.SessionID,
		TenantID:    req.ProjectID,
		ProjectPath: projectPath,
		Name:        name,
		Description: fmt.Sprintf("Automatic checkpoint created when context reached %d%% threshold", req.Percent),
		Summary:     summary,
		Context:     req.Context,
		FullState:   "",
		TokenCount:  0,
		Threshold:   float64(req.Percent) / 100.0,
		AutoCreated: true,
		Metadata:    map[string]string{"trigger": "threshold"},
	})

	if err != nil {
		s.logger.Error("failed to create auto-checkpoint",
			zap.Error(err),
			zap.String("session_id", req.SessionID),
			zap.Int("percent", req.Percent),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create checkpoint")
	}

	s.logger.Info("created auto-checkpoint",
		zap.String("checkpoint_id", chkpt.ID),
		zap.String("session_id", req.SessionID),
		zap.Int("percent", req.Percent),
	)

	// Execute threshold hook if available
	if hooksSvc := s.registry.Hooks(); hooksSvc != nil {
		if err := hooksSvc.Execute(ctx, hooks.HookContextThreshold, map[string]interface{}{
			"session_id":    req.SessionID,
			"project_id":    req.ProjectID,
			"percent":       req.Percent,
			"checkpoint_id": chkpt.ID,
		}); err != nil {
			s.logger.Warn("threshold hook failed",
				zap.Error(err),
				zap.String("checkpoint_id", chkpt.ID),
			)
			// Don't fail the request if hook fails
		}
	}

	return c.JSON(http.StatusOK, ThresholdResponse{
		CheckpointID: chkpt.ID,
		Message:      fmt.Sprintf("Auto-checkpoint created at %d%% context threshold", req.Percent),
	})
}

// Note: handleCheckpointSave, handleCheckpointList, and handleCheckpointResume methods
// were removed to address CVE-2025-CONTEXTD-001 (missing tenant context injection).
// Checkpoint operations are available via MCP tools with proper security:
//   - checkpoint_save
//   - checkpoint_list
//   - checkpoint_resume

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("starting http server", zap.String("addr", addr))
	return s.echo.Start(addr)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	return s.echo.Shutdown(ctx)
}
