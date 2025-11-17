// Package server provides HTTP server functionality for contextd v3.
//
// This package implements a graceful HTTP server with Echo router,
// health check endpoints, and context-aware shutdown.
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fyrsmithlabs/contextd/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Server represents the HTTP server.
type Server struct {
	config *config.Config
	echo   *echo.Echo
}

// HealthResponse is the JSON response for /health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// NewServer creates a new HTTP server with the given configuration.
//
// The server includes:
//   - Echo router for HTTP routing
//   - Standard middleware (logger, recoverer, request ID)
//   - Health check endpoint at GET /health
//   - Graceful shutdown support
//
// Example:
//
//	cfg := config.Load()
//	srv := server.NewServer(cfg)
//	if err := srv.Start(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
func NewServer(cfg *config.Config) *Server {
	e := echo.New()

	// Disable Echo's default logger and recover middleware
	e.HideBanner = true
	e.HidePort = true

	// Setup middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	s := &Server{
		config: cfg,
		echo:   e,
	}

	// Register routes
	s.registerRoutes()

	return s
}

// registerRoutes registers all HTTP routes.
func (s *Server) registerRoutes() {
	s.echo.GET("/health", s.handleHealth)
}

// handleHealth handles GET /health requests.
func (s *Server) handleHealth(c echo.Context) error {
	response := HealthResponse{
		Status:  "ok",
		Service: s.config.Observability.ServiceName,
	}

	return c.JSON(http.StatusOK, response)
}

// Start starts the HTTP server and blocks until context is cancelled.
//
// The server listens on the port specified in the configuration.
// When the context is cancelled, the server performs graceful shutdown
// with the configured timeout.
//
// Returns http.ErrServerClosed on graceful shutdown, or any other
// error encountered during startup or shutdown.
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
//	    log.Fatalf("server error: %v", err)
//	}
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)

	// Channel to receive server errors
	errCh := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := s.echo.Start(addr); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server start: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Context cancelled, perform graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			s.config.Server.ShutdownTimeout,
		)
		defer cancel()

		if err := s.echo.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}

		return http.ErrServerClosed
	}
}

// Echo returns the underlying Echo instance for registering additional routes.
//
// This is useful for extending the server with MCP endpoints or other handlers.
//
// Example:
//
//	srv := server.NewServer(cfg)
//	mcpServer := mcp.NewServer(srv.Echo(), operations, nats)
//	mcpServer.RegisterRoutes()
func (s *Server) Echo() *echo.Echo {
	return s.echo
}
