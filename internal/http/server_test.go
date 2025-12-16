package http

import (
	"bytes"
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

func TestScrubWithDisabledScrubber(t *testing.T) {
	// Create scrubber with disabled config
	cfg := &secrets.Config{
		Enabled: false,
	}
	scrubber, err := secrets.New(cfg)
	require.NoError(t, err)

	registry := &mockRegistry{}
	registry.On("Scrubber").Return(scrubber)

	serverCfg := &Config{
		Host: "localhost",
		Port: 9090,
	}

	server, err := NewServer(registry, zap.NewNop(), serverCfg)
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

// mockCheckpointService is a mock implementation of checkpoint.Service
type mockCheckpointService struct {
	mock.Mock
}

func (m *mockCheckpointService) Save(ctx context.Context, req *checkpoint.SaveRequest) (*checkpoint.Checkpoint, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*checkpoint.Checkpoint), args.Error(1)
}

func (m *mockCheckpointService) List(ctx context.Context, req *checkpoint.ListRequest) ([]*checkpoint.Checkpoint, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*checkpoint.Checkpoint), args.Error(1)
}

func (m *mockCheckpointService) Resume(ctx context.Context, req *checkpoint.ResumeRequest) (*checkpoint.ResumeResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*checkpoint.ResumeResponse), args.Error(1)
}

func (m *mockCheckpointService) Get(ctx context.Context, tenantID, teamID, projectID, checkpointID string) (*checkpoint.Checkpoint, error) {
	args := m.Called(ctx, tenantID, teamID, projectID, checkpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*checkpoint.Checkpoint), args.Error(1)
}

func (m *mockCheckpointService) Delete(ctx context.Context, tenantID, teamID, projectID, checkpointID string) error {
	args := m.Called(ctx, tenantID, teamID, projectID, checkpointID)
	return args.Error(0)
}

func (m *mockCheckpointService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestHandleThreshold(t *testing.T) {
	t.Run("creates auto-checkpoint and executes hook", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		mockCp := &mockCheckpointService{}
		mockHooks := hooks.NewHookManager(&hooks.Config{
			CheckpointThreshold: 70,
		})

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)
		registry.On("Checkpoint").Return(mockCp)
		registry.On("Hooks").Return(mockHooks)

		cfg := &Config{
			Host: "localhost",
			Port: 9090,
		}

		server, err := NewServer(registry, zap.NewNop(), cfg)
		require.NoError(t, err)

		// Mock checkpoint save
		mockCp.On("Save", mock.Anything, mock.MatchedBy(func(req *checkpoint.SaveRequest) bool {
			return req.SessionID == "sess_123" &&
				req.TenantID == "tenant_456" &&
				req.AutoCreated == true
		})).Return(&checkpoint.Checkpoint{
			ID:        "cp_auto_123",
			SessionID: "sess_123",
			TenantID:  "tenant_456",
		}, nil)

		reqBody := ThresholdRequest{
			ProjectID: "tenant_456",
			SessionID: "sess_123",
			Percent:   70,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/threshold", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp ThresholdResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "cp_auto_123", resp.CheckpointID)
		assert.Contains(t, resp.Message, "Auto-checkpoint created")

		mockCp.AssertExpectations(t)
	})

	t.Run("handles missing fields", func(t *testing.T) {
		server := setupTestServer(t)

		reqBody := ThresholdRequest{
			SessionID: "sess_123",
			// Missing ProjectID and Percent
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/threshold", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("handles checkpoint save error", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		mockCp := &mockCheckpointService{}
		mockHooks := hooks.NewHookManager(&hooks.Config{
			CheckpointThreshold: 70,
		})

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)
		registry.On("Checkpoint").Return(mockCp)
		registry.On("Hooks").Return(mockHooks)

		cfg := &Config{
			Host: "localhost",
			Port: 9090,
		}

		server, err := NewServer(registry, zap.NewNop(), cfg)
		require.NoError(t, err)

		// Mock checkpoint save error
		mockCp.On("Save", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		reqBody := ThresholdRequest{
			ProjectID: "tenant_456",
			SessionID: "sess_123",
			Percent:   70,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/threshold", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("handles invalid json", func(t *testing.T) {
		server := setupTestServer(t)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/threshold", bytes.NewReader([]byte("invalid json")))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("uses provided summary and context", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		mockCp := &mockCheckpointService{}
		mockHooks := hooks.NewHookManager(&hooks.Config{
			CheckpointThreshold: 70,
		})

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)
		registry.On("Checkpoint").Return(mockCp)
		registry.On("Hooks").Return(mockHooks)

		cfg := &Config{
			Host: "localhost",
			Port: 9090,
		}

		server, err := NewServer(registry, zap.NewNop(), cfg)
		require.NoError(t, err)

		// Mock checkpoint save - verify summary and context are passed
		mockCp.On("Save", mock.Anything, mock.MatchedBy(func(req *checkpoint.SaveRequest) bool {
			return req.SessionID == "sess_123" &&
				req.TenantID == "tenant_456" &&
				req.ProjectPath == "/home/user/project" &&
				req.Summary == "Implementing auth middleware" &&
				req.Context == "Using JWT with RS256" &&
				req.AutoCreated == true
		})).Return(&checkpoint.Checkpoint{
			ID:        "cp_auto_456",
			SessionID: "sess_123",
			TenantID:  "tenant_456",
		}, nil)

		reqBody := ThresholdRequest{
			ProjectID:   "tenant_456",
			SessionID:   "sess_123",
			Percent:     70,
			Summary:     "Implementing auth middleware",
			Context:     "Using JWT with RS256",
			ProjectPath: "/home/user/project",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/threshold", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp ThresholdResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "cp_auto_456", resp.CheckpointID)
		mockCp.AssertExpectations(t)
	})

	t.Run("truncates long summary for checkpoint name", func(t *testing.T) {
		scrubber, err := secrets.New(nil)
		require.NoError(t, err)

		mockCp := &mockCheckpointService{}
		mockHooks := hooks.NewHookManager(&hooks.Config{
			CheckpointThreshold: 70,
		})

		registry := &mockRegistry{}
		registry.On("Scrubber").Return(scrubber)
		registry.On("Checkpoint").Return(mockCp)
		registry.On("Hooks").Return(mockHooks)

		cfg := &Config{
			Host: "localhost",
			Port: 9090,
		}

		server, err := NewServer(registry, zap.NewNop(), cfg)
		require.NoError(t, err)

		longSummary := "This is a very long summary that exceeds fifty characters and should be truncated"

		// Mock checkpoint save - verify name is truncated
		mockCp.On("Save", mock.Anything, mock.MatchedBy(func(req *checkpoint.SaveRequest) bool {
			// Name should be truncated to 50 chars with "..."
			return len(req.Name) == 50 &&
				req.Name == longSummary[:47]+"..." &&
				req.Summary == longSummary // Full summary preserved
		})).Return(&checkpoint.Checkpoint{
			ID:        "cp_auto_789",
			SessionID: "sess_123",
			TenantID:  "tenant_456",
		}, nil)

		reqBody := ThresholdRequest{
			ProjectID: "tenant_456",
			SessionID: "sess_123",
			Percent:   70,
			Summary:   longSummary,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/threshold", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		server.echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		mockCp.AssertExpectations(t)
	})
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
