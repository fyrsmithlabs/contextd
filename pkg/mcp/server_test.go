package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServer tests MCP server creation.
func TestNewServer(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil)

	assert.NotNil(t, mcpServer)
	assert.NotNil(t, mcpServer.echo)
	assert.NotNil(t, mcpServer.operations)
	assert.NotNil(t, mcpServer.nats)
}

// TestMCPServer_CheckpointSave tests POST /mcp/checkpoint/save.
func TestMCPServer_CheckpointSave(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil)

	// Add test middleware to set authenticated owner ID (simulates auth middleware)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set valid 64-character hex owner ID for testing
			c.Set(string(authenticatedOwnerIDKey), "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678")
			return next(c)
		}
	})

	mcpServer.RegisterRoutes()

	// Valid request
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "test-123",
		"method": "checkpoint_save",
		"params": {
			"content": "test content",
			"project_path": "/tmp/test"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/checkpoint/save", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "operation_id")
	assert.Contains(t, rec.Body.String(), "pending")
}

// TestMCPServer_CheckpointSave_InvalidParams tests validation.
func TestMCPServer_CheckpointSave_InvalidParams(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil)
	mcpServer.RegisterRoutes()

	// Missing required field
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "test-456",
		"method": "checkpoint_save",
		"params": {
			"content": "test"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/checkpoint/save", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "error")
	assert.Contains(t, rec.Body.String(), "project_path is required")
}

// TestMCPServer_Status tests POST /mcp/status.
func TestMCPServer_Status(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil)
	mcpServer.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/mcp/status", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")
	assert.Contains(t, rec.Body.String(), "contextd")
}

// TestMCPServer_AllEndpoints tests all endpoints are registered.
func TestMCPServer_AllEndpoints(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil)
	mcpServer.RegisterRoutes()

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/mcp/checkpoint/save"},
		{http.MethodPost, "/mcp/checkpoint/search"},
		{http.MethodPost, "/mcp/checkpoint/list"},
		{http.MethodPost, "/mcp/remediation/save"},
		{http.MethodPost, "/mcp/remediation/search"},
		{http.MethodPost, "/mcp/skill/save"},
		{http.MethodPost, "/mcp/skill/search"},
		{http.MethodPost, "/mcp/index/repository"},
		{http.MethodPost, "/mcp/status"},
	}

	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Just verify endpoint exists (not 404)
			assert.NotEqual(t, http.StatusNotFound, rec.Code, "endpoint %s should be registered", ep.path)
		})
	}
}
