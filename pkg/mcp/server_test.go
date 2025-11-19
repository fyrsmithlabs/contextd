package mcp

import (
	"context"
	"fmt"
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
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)

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
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)

	// Register routes (includes auth middleware)
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
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
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

// TestMCPServer_Status_ValidOperationID tests POST /mcp/status with valid operation.
func TestMCPServer_Status_ValidOperationID(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
	mcpServer.RegisterRoutes()

	// Create a test operation
	ctx := context.Background()
	opID := registry.Create(ctx, "checkpoint_save", map[string]interface{}{
		"content":      "test",
		"project_path": "/tmp/test",
	})
	require.NotEmpty(t, opID)

	// Mark as completed
	err = registry.Complete(opID, map[string]interface{}{
		"checkpoint_id": "ckpt-123",
	})
	require.NoError(t, err)

	// Query status
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "status-req-1",
		"method": "status",
		"params": {
			"operation_id": "` + opID + `"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/status", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"operation_id":"`+opID+`"`)
	assert.Contains(t, rec.Body.String(), `"status":"completed"`)
	assert.Contains(t, rec.Body.String(), `"checkpoint_id":"ckpt-123"`)
}

// TestMCPServer_Status_UnknownOperationID tests error for unknown operation.
func TestMCPServer_Status_UnknownOperationID(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
	mcpServer.RegisterRoutes()

	// Query non-existent operation
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "status-req-2",
		"method": "status",
		"params": {
			"operation_id": "non-existent-op-id"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/status", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"error"`)
	assert.Contains(t, rec.Body.String(), `"code":-32602`)
	assert.Contains(t, rec.Body.String(), "operation not found")
}

// TestMCPServer_Status_FailedOperation tests status of failed operation.
func TestMCPServer_Status_FailedOperation(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
	mcpServer.RegisterRoutes()

	// Create operation and fail it
	ctx := context.Background()
	opID := registry.Create(ctx, "checkpoint_save", map[string]interface{}{
		"content": "test",
	})
	require.NotEmpty(t, opID)

	err = registry.Error(opID, InternalError, fmt.Errorf("test error"))
	require.NoError(t, err)

	// Query status
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "status-req-3",
		"method": "status",
		"params": {
			"operation_id": "` + opID + `"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/status", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"failed"`)
	assert.Contains(t, rec.Body.String(), `"error":"test error"`)
}

// TestMCPServer_Status_PendingOperation tests status of pending operation.
func TestMCPServer_Status_PendingOperation(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
	mcpServer.RegisterRoutes()

	// Create pending operation
	ctx := context.Background()
	opID := registry.Create(ctx, "index_repository", map[string]interface{}{
		"project_path": "/tmp/test",
	})
	require.NotEmpty(t, opID)

	// Query status immediately (still pending)
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "status-req-4",
		"method": "status",
		"params": {
			"operation_id": "` + opID + `"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/status", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"pending"`)
	assert.Contains(t, rec.Body.String(), `"operation_id":"`+opID+`"`)
}

// TestMCPServer_Status_MissingOperationID tests validation error.
func TestMCPServer_Status_MissingOperationID(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
	mcpServer.RegisterRoutes()

	// Request without operation_id
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "status-req-5",
		"method": "status",
		"params": {}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/status", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"error"`)
	assert.Contains(t, rec.Body.String(), `"code":-32602`)
	assert.Contains(t, rec.Body.String(), "operation_id is required")
}

// TestMCPServer_AllEndpoints tests all endpoints are registered.
func TestMCPServer_AllEndpoints(t *testing.T) {
	e := echo.New()
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
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
