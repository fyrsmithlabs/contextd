package mcp

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// bearerTokenMiddleware enforces "Authorization: Bearer <token>" using a
// constant-time comparison to avoid leaking the token via timing.
func bearerTokenMiddleware(token string) echo.MiddlewareFunc {
	want := []byte(token)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get(echo.HeaderAuthorization)
			const prefix = "Bearer "
			if !strings.HasPrefix(h, prefix) {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}
			got := []byte(strings.TrimSpace(h[len(prefix):]))
			if subtle.ConstantTimeEq(int32(len(got)), int32(len(want))) != 1 ||
				subtle.ConstantTimeCompare(got, want) != 1 {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid bearer token")
			}
			return next(c)
		}
	}
}

// StreamableHTTPConfig configures the standalone Streamable HTTP MCP server.
type StreamableHTTPConfig struct {
	// Host to bind (default: "localhost").
	Host string
	// Port to listen on.
	Port int
	// Path the MCP endpoint is mounted at (default: "/mcp").
	Path string
	// Stateless, when true, disables Mcp-Session-Id validation.
	//
	// Keep this false (the default) for agent-swarm use: stateful sessions are
	// what allow multiple connected clients (agents) to subscribe to
	// resource-update notifications and receive server-initiated messages. See
	// docs/spec/mcp-protocol/notifications-agent-swarm.md.
	Stateless bool
	// Token, when non-empty, requires clients to present
	// "Authorization: Bearer <Token>" on the MCP endpoint. When empty, the
	// endpoint is served unauthenticated (intended for localhost/testing) and
	// RunHTTP logs a prominent warning.
	Token string
	// ReadHeaderTimeout guards against slow-loris clients (default: 10s).
	ReadHeaderTimeout time.Duration
}

func (c *StreamableHTTPConfig) withDefaults() StreamableHTTPConfig {
	out := *c
	if out.Host == "" {
		out.Host = "localhost"
	}
	if out.Path == "" {
		out.Path = "/mcp"
	}
	if out.ReadHeaderTimeout == 0 {
		out.ReadHeaderTimeout = 10 * time.Second
	}
	return out
}

// StreamableHandler returns the SDK Streamable HTTP handler bound to this
// server. It is exposed so the handler can be mounted into any net/http or Echo
// server. The same underlying MCP server is reused across sessions.
func (s *Server) StreamableHandler(stateless bool) http.Handler {
	return mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return s.mcp },
		&mcpsdk.StreamableHTTPOptions{Stateless: stateless},
	)
}

// RunHTTP runs the MCP server over the Streamable HTTP transport using a
// dedicated Echo server.
//
// This is intentionally a SEPARATE server from internal/http (the REST API).
// Keeping them apart lets the MCP transport be developed and tested in
// isolation; the two are expected to be consolidated later.
//
// RunHTTP blocks until ctx is cancelled or the server fails, then shuts down
// gracefully.
func (s *Server) RunHTTP(ctx context.Context, cfg StreamableHTTPConfig) error {
	cfg = cfg.withDefaults()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	// MCP Streamable HTTP uses a single endpoint for both POST (client→server
	// messages) and GET (server→client SSE stream). Bearer auth (when a token
	// is configured) guards only this endpoint; /health stays open.
	mcpEndpoint := e.Group(cfg.Path)
	if cfg.Token != "" {
		mcpEndpoint.Use(bearerTokenMiddleware(cfg.Token))
	} else {
		s.logger.Warn("MCP streamable HTTP server running WITHOUT authentication; " +
			"set a token (--mcp-http-token / CONTEXTD_MCP_HTTP_TOKEN) before exposing beyond localhost")
	}
	mcpEndpoint.Any("", echo.WrapHandler(s.StreamableHandler(cfg.Stateless)))

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":    "ok",
			"transport": "streamable-http",
			"endpoint":  cfg.Path,
		})
	})

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	s.logger.Info("starting MCP server on streamable HTTP transport",
		zap.String("addr", addr),
		zap.String("path", cfg.Path),
		zap.Bool("stateless", cfg.Stateless),
	)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("streamable HTTP shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("streamable HTTP server: %w", err)
		}
		return nil
	}
}
