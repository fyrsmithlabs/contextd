// Package http provides HTTP API for contextd.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/services"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
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
	v1.GET("/status", s.handleStatus)
	v1.GET("/health/metadata", s.handleMetadataHealth)

	// Note: Checkpoint management is available via MCP tools (checkpoint_save, checkpoint_list, checkpoint_resume)
	// HTTP endpoints were removed due to security concerns (CVE-2025-CONTEXTD-001).
	//
	// Note: The secret-scrubbing and context-threshold POST endpoints were also
	// removed - agents should use the MCP secrets_scrub and checkpoint_save flows
	// directly instead of the HTTP surface.
}

// HealthResponse is the response body for GET /health.
type HealthResponse struct {
	Status   string                `json:"status"`
	Metadata *MetadataHealthStatus `json:"metadata,omitempty"` // Optional metadata health
}

// StatusResponse, StatusCounts, ContextStatus, and MemoryStatus
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
// Restricted to localhost connections only to prevent internal metadata exposure.
func (s *Server) handleMetadataHealth(c echo.Context) error {
	// Restrict to localhost only (CWE-200: prevent internal metadata exposure)
	// Use c.Request().RemoteAddr directly instead of c.RealIP() which trusts
	// X-Forwarded-For/X-Real-IP headers that can be spoofed by clients.
	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		// RemoteAddr without port (shouldn't happen with net/http, but be safe)
		host = c.Request().RemoteAddr
	}
	remoteIP := net.ParseIP(host)
	if remoteIP == nil || !remoteIP.IsLoopback() {
		return echo.NewHTTPError(http.StatusForbidden, "metadata health endpoint is restricted to localhost")
	}

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

	// Add memory stats if available
	if s.registry.Memory() != nil {
		memStats := s.registry.Memory().Stats()
		resp.Memory = &MemoryStatus{
			LastConfidence: memStats.LastConfidence,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// Note: The POST /scrub and POST /threshold handlers were removed - agents
// should use the MCP secrets_scrub and checkpoint_save tools directly. The
// scrub endpoint was an unauthed loopback-only surface (CWE-200 risk if
// binding misconfigured) and the threshold endpoint duplicated checkpoint_save.

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
