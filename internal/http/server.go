// Package http provides HTTP API for contextd.
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Server provides HTTP endpoints for contextd.
type Server struct {
	echo     *echo.Echo
	scrubber secrets.Scrubber
	logger   *zap.Logger
	config   *Config
}

// Config holds HTTP server configuration.
type Config struct {
	Host string
	Port int
}

// NewServer creates a new HTTP server.
func NewServer(scrubber secrets.Scrubber, logger *zap.Logger, cfg *Config) (*Server, error) {
	if scrubber == nil {
		return nil, fmt.Errorf("scrubber cannot be nil")
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
		scrubber: scrubber,
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

// HealthResponse is the response body for GET /health.
type HealthResponse struct {
	Status string `json:"status"`
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
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
	result := s.scrubber.Scrub(req.Content)

	s.logger.Debug("scrubbed content",
		zap.Int("findings", result.TotalFindings),
		zap.Duration("duration", result.Duration),
	)

	return c.JSON(http.StatusOK, ScrubResponse{
		Content:       result.Scrubbed,
		FindingsCount: result.TotalFindings,
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
