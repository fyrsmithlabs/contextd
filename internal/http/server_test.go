package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Ensure mock implements the interface
var _ = (*mockRegistry)(nil)

func TestNewServer(t *testing.T) {
	t.Run("creates server with valid config", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)

		cfg := &Config{
			Host: "localhost",
			Port: 9090,
		}

		server, err := NewServer(registry, zap.NewNop(), cfg)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.echo)
		assert.Equal(t, cfg, server.config)
	})

	t.Run("uses defaults when config is nil", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)

		server, err := NewServer(registry, zap.NewNop(), nil)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "localhost", server.config.Host)
		assert.Equal(t, 9090, server.config.Port)
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)

		_, err = NewServer(registry, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logger is required")
	})

	t.Run("returns error when registry is nil", func(t *testing.T) {
		_, err := NewServer(nil, zap.NewNop(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registry cannot be nil")
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

func TestServerLifecycle(t *testing.T) {
	t.Run("starts and shuts down gracefully", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)

		cfg := &Config{
			Host: "localhost",
			Port: 0, // Use random available port
		}

		server, err := NewServer(registry, zap.NewNop(), cfg)
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

// mockRegistry is a mock implementation of services.Registry
type mockRegistry struct {
	mock.Mock
}

func (m *mockRegistry) Checkpoint() checkpoint.Service {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(checkpoint.Service)
}

func (m *mockRegistry) Remediation() remediation.Service {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(remediation.Service)
}

func (m *mockRegistry) Memory() *reasoningbank.Service {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*reasoningbank.Service)
}

func (m *mockRegistry) Repository() *repository.Service {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*repository.Service)
}

func (m *mockRegistry) Troubleshoot() *troubleshoot.Service {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*troubleshoot.Service)
}

func (m *mockRegistry) Hooks() *hooks.HookManager {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*hooks.HookManager)
}

func (m *mockRegistry) Distiller() *reasoningbank.Distiller {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*reasoningbank.Distiller)
}

func (m *mockRegistry) Scrubber() secrets.Scrubber {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(secrets.Scrubber)
}

func (m *mockRegistry) VectorStore() vectorstore.Store {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(vectorstore.Store)
}

// setupTestServer creates a test server with default configuration.
func setupTestServer(t *testing.T) *Server {
	t.Helper()

	scrubber, err := secrets.New(nil)
	require.NoError(t, err)

	registry := &mockRegistry{}
	registry.On("Scrubber").Return(scrubber)

	cfg := &Config{
		Host: "localhost",
		Port: 9090,
	}

	server, err := NewServer(registry, zap.NewNop(), cfg)
	require.NoError(t, err)

	return server
}
