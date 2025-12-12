// Package http provides HTTP API for contextd.
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	"github.com/fyrsmithlabs/contextd/internal/services"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Server provides HTTP endpoints for contextd.
type Server struct {
	echo     *echo.Echo
	registry services.Registry
	logger   *zap.Logger
	config   *Config
}

// Config holds HTTP server configuration.
type Config struct {
	Host string
	Port int
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

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
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
		echo:     e,
		registry: registry,
		logger:   logger,
		config:   cfg,
	}

	// Register routes
	s.registerRoutes()

	return s, nil
}

// registerRoutes sets up the HTTP endpoints.
func (s *Server) registerRoutes() {
	// Health check
	s.echo.GET("/health", s.handleHealth)

	// API v1 routes
	v1 := s.echo.Group("/api/v1")
	v1.POST("/scrub", s.handleScrub)
	v1.POST("/threshold", s.handleThreshold)
	v1.GET("/status", s.handleStatus)
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
	Status string `json:"status"`
}

// StatusResponse is the response body for GET /api/v1/status.
type StatusResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
	Counts   StatusCounts      `json:"counts"`
}

// StatusCounts contains count information for various resources.
type StatusCounts struct {
	Checkpoints int `json:"checkpoints"`
	Memories    int `json:"memories"`
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
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

	// Get counts (best effort - don't fail if services unavailable)
	counts := StatusCounts{}

	// Get checkpoint count
	if s.registry.Checkpoint() != nil {
		// List checkpoints for count (use empty tenant for global count)
		checkpoints, err := s.registry.Checkpoint().List(ctx, &checkpoint.ListRequest{
			Limit: 1000, // reasonable max for counting
		})
		if err == nil {
			counts.Checkpoints = len(checkpoints)
		}
	}

	// Note: Memory count would require a Count method on ReasoningBank
	// For now, we report 0 if not available
	counts.Memories = 0

	return c.JSON(http.StatusOK, StatusResponse{
		Status:   "ok",
		Services: services,
		Counts:   counts,
	})
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

	// Scrub the content
	result := s.registry.Scrubber().Scrub(req.Content)

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

	if req.ProjectID == "" || req.SessionID == "" || req.Percent == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "project_id, session_id, and percent fields are required")
	}

	// Use provided values or fall back to defaults
	projectPath := req.ProjectPath
	if projectPath == "" {
		projectPath = req.ProjectID
	}

	summary := req.Summary
	if summary == "" {
		summary = fmt.Sprintf("Context at %d%% threshold", req.Percent)
	}

	name := fmt.Sprintf("Auto-checkpoint at %d%%", req.Percent)
	if req.Summary != "" {
		// Use first 50 chars of summary as name if provided
		name = req.Summary
		if len(name) > 50 {
			name = name[:47] + "..."
		}
	}

	// Create auto-checkpoint via checkpoint service
	ctx := c.Request().Context()
	checkpoint, err := s.registry.Checkpoint().Save(ctx, &checkpoint.SaveRequest{
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
		zap.String("checkpoint_id", checkpoint.ID),
		zap.String("session_id", req.SessionID),
		zap.Int("percent", req.Percent),
	)

	// Execute threshold hook
	if err := s.registry.Hooks().Execute(ctx, hooks.HookContextThreshold, map[string]interface{}{
		"session_id":    req.SessionID,
		"project_id":    req.ProjectID,
		"percent":       req.Percent,
		"checkpoint_id": checkpoint.ID,
	}); err != nil {
		s.logger.Warn("threshold hook failed",
			zap.Error(err),
			zap.String("checkpoint_id", checkpoint.ID),
		)
		// Don't fail the request if hook fails
	}

	return c.JSON(http.StatusOK, ThresholdResponse{
		CheckpointID: checkpoint.ID,
		Message:      fmt.Sprintf("Auto-checkpoint created at %d%% context threshold", req.Percent),
	})
}

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
