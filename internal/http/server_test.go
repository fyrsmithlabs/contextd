package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewServer(t *testing.T) {
	t.Run("creates server with valid config", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		cfg := &Config{
			Host: "localhost",
			Port: 9090,
		}

		server, err := NewServer(scrubber, zap.NewNop(), cfg)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.echo)
		assert.Equal(t, cfg, server.config)
	})

	t.Run("uses defaults when config is nil", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		server, err := NewServer(scrubber, zap.NewNop(), nil)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "localhost", server.config.Host)
		assert.Equal(t, 9090, server.config.Port)
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		_, err = NewServer(scrubber, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logger is required")
	})

	t.Run("returns error when scrubber is nil", func(t *testing.T) {
		_, err := NewServer(nil, zap.NewNop(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scrubber cannot be nil")
	})
}

func TestHandleHealth(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestHandleScrub(t *testing.T) {
	t.Run("scrubs secrets from content", func(t *testing.T) {
		server := setupTestServer(t)

		reqBody := ScrubRequest{
			Content: "my api key is AKIAIOSFODNN7EXAMPLE",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrub", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp ScrubResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Contains(t, resp.Content, "[REDACTED]")
		assert.NotContains(t, resp.Content, "AKIAIOSFODNN7EXAMPLE")
		assert.Equal(t, 1, resp.FindingsCount)
	})

	t.Run("handles content with no secrets", func(t *testing.T) {
		server := setupTestServer(t)

		reqBody := ScrubRequest{
			Content: "This is just regular text with no secrets.",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrub", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp ScrubResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, reqBody.Content, resp.Content)
		assert.Equal(t, 0, resp.FindingsCount)
	})

	t.Run("handles empty content field", func(t *testing.T) {
		server := setupTestServer(t)

		reqBody := ScrubRequest{
			Content: "",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrub", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["message"], "content field is required")
	})

	t.Run("handles invalid json", func(t *testing.T) {
		server := setupTestServer(t)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrub", bytes.NewReader([]byte("invalid json")))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("scrubs multiple secrets", func(t *testing.T) {
		server := setupTestServer(t)

		reqBody := ScrubRequest{
			Content: `
AWS_KEY=AKIAIOSFODNN7EXAMPLE
GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij
`,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrub", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp ScrubResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Contains(t, resp.Content, "[REDACTED]")
		assert.NotContains(t, resp.Content, "AKIAIOSFODNN7EXAMPLE")
		assert.NotContains(t, resp.Content, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij")
		assert.GreaterOrEqual(t, resp.FindingsCount, 2)
	})

	t.Run("handles large content", func(t *testing.T) {
		server := setupTestServer(t)

		// Generate 10KB of content with a secret
		largeContent := ""
		for i := 0; i < 100; i++ {
			largeContent += "This is some test content. "
		}
		largeContent += "secret: AKIAIOSFODNN7EXAMPLE"

		reqBody := ScrubRequest{
			Content: largeContent,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrub", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp ScrubResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotContains(t, resp.Content, "AKIAIOSFODNN7EXAMPLE")
		assert.Equal(t, 1, resp.FindingsCount)
	})
}

func TestServerLifecycle(t *testing.T) {
	t.Run("starts and shuts down gracefully", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		cfg := &Config{
			Host: "localhost",
			Port: 0, // Use random available port
		}

		server, err := NewServer(scrubber, zap.NewNop(), cfg)
		require.NoError(t, err)

		// Start server in background
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start()
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = server.Shutdown(ctx)
		assert.NoError(t, err)

		// Wait for server to finish
		select {
		case err := <-errChan:
			// Server should shut down cleanly (http.ErrServerClosed is expected)
			assert.True(t, err == nil || err == http.ErrServerClosed)
		case <-time.After(6 * time.Second):
			t.Fatal("server did not shut down in time")
		}
	})
}

func TestMiddleware(t *testing.T) {
	t.Run("adds request ID to response", func(t *testing.T) {
		server := setupTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.NotEmpty(t, rec.Header().Get(echo.HeaderXRequestID))
	})

	t.Run("recovers from panic", func(t *testing.T) {
		server := setupTestServer(t)

		// Add a route that panics
		server.echo.GET("/panic", func(c echo.Context) error {
			panic("test panic")
		})

		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		rec := httptest.NewRecorder()

		// Should not panic, middleware should recover
		assert.NotPanics(t, func() {
			server.echo.ServeHTTP(rec, req)
		})

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestScrubWithDisabledScrubber(t *testing.T) {
	// Create scrubber with disabled config
	cfg := &secrets.Config{
		Enabled: false,
	}
	scrubber, err := secrets.New(cfg)
	require.NoError(t, err)

	serverCfg := &Config{
		Host: "localhost",
		Port: 9090,
	}

	server, err := NewServer(scrubber, zap.NewNop(), serverCfg)
	require.NoError(t, err)

	reqBody := ScrubRequest{
		Content: "my api key is AKIAIOSFODNN7EXAMPLE",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scrub", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ScrubResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	// When scrubber is disabled, content should be unchanged
	assert.Equal(t, reqBody.Content, resp.Content)
	assert.Equal(t, 0, resp.FindingsCount)
}

// setupTestServer creates a test server with default configuration.
func setupTestServer(t *testing.T) *Server {
	t.Helper()

	scrubber, err := secrets.New(nil)
	require.NoError(t, err)

	cfg := &Config{
		Host: "localhost",
		Port: 9090,
	}

	server, err := NewServer(scrubber, zap.NewNop(), cfg)
	require.NoError(t, err)

	return server
}
