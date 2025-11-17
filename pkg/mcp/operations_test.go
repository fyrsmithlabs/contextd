package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startTestNATSServer starts an embedded NATS server for testing.
func startTestNATSServer(t *testing.T) *natsserver.Server {
	opts := &natsserver.Options{
		Host:           "127.0.0.1",
		Port:           -1, // Random port
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 2048,
		JetStream:      true,
	}

	server, err := natsserver.NewServer(opts)
	require.NoError(t, err)

	go server.Start()

	if !server.ReadyForConnections(5 * time.Second) {
		t.Fatal("NATS server not ready")
	}

	t.Cleanup(func() {
		server.Shutdown()
		server.WaitForShutdown()
	})

	return server
}

// TestNewOperationRegistry tests operation registry initialization.
func TestNewOperationRegistry(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.nats)
}

// TestOperationRegistry_Create tests operation creation.
func TestOperationRegistry_Create(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	ctx = context.WithValue(ctx, traceIDKey, "trace-456")

	params := map[string]string{"content": "test"}
	opID := registry.Create(ctx, "checkpoint_save", params)

	assert.NotEmpty(t, opID)

	// Verify operation exists
	op, err := registry.Get(opID)
	require.NoError(t, err)
	assert.Equal(t, "owner-123", op.OwnerID)
	assert.Equal(t, "checkpoint_save", op.Tool)
	assert.Equal(t, "pending", op.Status)
	assert.Equal(t, "trace-456", op.TraceID)
}

// TestOperationRegistry_Started tests operation start event.
func TestOperationRegistry_Started(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Subscribe to NATS events
	subject := fmt.Sprintf("operations.owner-123.%s.started", opID)
	ch := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(subject, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish started event
	registry.Started(opID)

	// Verify event received
	select {
	case msg := <-ch:
		var op Operation
		err = json.Unmarshal(msg.Data, &op)
		require.NoError(t, err)
		assert.Equal(t, opID, op.ID)
		assert.Equal(t, "running", op.Status)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for started event")
	}

	// Verify status updated in registry
	op, err := registry.Get(opID)
	require.NoError(t, err)
	assert.Equal(t, "running", op.Status)
}

// TestOperationRegistry_Progress tests progress event publishing.
func TestOperationRegistry_Progress(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Subscribe to progress events
	subject := fmt.Sprintf("operations.owner-123.%s.progress", opID)
	ch := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(subject, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish progress event
	registry.Progress(opID, 50, "Processing...")

	// Verify event received
	select {
	case msg := <-ch:
		var progress map[string]interface{}
		err := json.Unmarshal(msg.Data, &progress)
		require.NoError(t, err)
		assert.Equal(t, opID, progress["id"])
		assert.Equal(t, float64(50), progress["percent"])
		assert.Equal(t, "Processing...", progress["message"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for progress event")
	}
}

// TestOperationRegistry_Log tests log event publishing.
func TestOperationRegistry_Log(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Subscribe to log events
	subject := fmt.Sprintf("operations.owner-123.%s.log", opID)
	ch := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(subject, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish log event
	registry.Log(opID, "info", "Test log message")

	// Verify event received
	select {
	case msg := <-ch:
		var logEvent map[string]interface{}
		err := json.Unmarshal(msg.Data, &logEvent)
		require.NoError(t, err)
		assert.Equal(t, opID, logEvent["id"])
		assert.Equal(t, "info", logEvent["level"])
		assert.Equal(t, "Test log message", logEvent["message"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for log event")
	}
}

// TestOperationRegistry_Error tests error event publishing.
func TestOperationRegistry_Error(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	ctx = context.WithValue(ctx, traceIDKey, "trace-456")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Subscribe to error events
	subject := fmt.Sprintf("operations.owner-123.%s.error", opID)
	ch := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(subject, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish error event
	registry.Error(opID, VectorStoreError, fmt.Errorf("test error"))

	// Verify event received
	select {
	case msg := <-ch:
		var errorDetail ErrorDetail
		err = json.Unmarshal(msg.Data, &errorDetail)
		require.NoError(t, err)
		assert.Equal(t, VectorStoreError, errorDetail.Code)
		assert.Equal(t, "test error", errorDetail.Message)
		assert.Equal(t, "trace-456", errorDetail.Data["trace_id"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for error event")
	}

	// Verify status updated
	op, err := registry.Get(opID)
	require.NoError(t, err)
	assert.Equal(t, "failed", op.Status)
	assert.NotNil(t, op.Error)
}

// TestOperationRegistry_Complete tests completion event publishing.
func TestOperationRegistry_Complete(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Subscribe to completed events
	subject := fmt.Sprintf("operations.owner-123.%s.completed", opID)
	ch := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(subject, ch)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish completed event
	result := map[string]interface{}{
		"checkpoint_id": "ckpt-123",
		"tokens":        2500,
	}
	registry.Complete(opID, result)

	// Verify event received
	select {
	case msg := <-ch:
		var completed map[string]interface{}
		err = json.Unmarshal(msg.Data, &completed)
		require.NoError(t, err)
		assert.Equal(t, opID, completed["id"])
		assert.NotNil(t, completed["result"])
		assert.NotNil(t, completed["duration_ms"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for completed event")
	}

	// Verify status updated
	op, err := registry.Get(opID)
	require.NoError(t, err)
	assert.Equal(t, "completed", op.Status)
	assert.NotNil(t, op.Result)
}

// TestOperationRegistry_Get tests operation retrieval.
func TestOperationRegistry_Get(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{"key": "value"})

	// Test successful get
	op, err := registry.Get(opID)
	require.NoError(t, err)
	assert.Equal(t, opID, op.ID)
	assert.Equal(t, "owner-123", op.OwnerID)

	// Test non-existent operation
	_, err = registry.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation not found")
}

// TestGetOwnerID tests extracting owner ID from context.
func TestGetOwnerID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "owner ID present",
			ctx:  context.WithValue(context.Background(), ownerIDKey, "owner-123"),
			want: "owner-123",
		},
		{
			name: "owner ID missing",
			ctx:  context.Background(),
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOwnerID(tt.ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestGetTraceID tests extracting trace ID from context.
func TestGetTraceID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "trace ID present",
			ctx:  context.WithValue(context.Background(), traceIDKey, "trace-456"),
			want: "trace-456",
		},
		{
			name: "trace ID missing",
			ctx:  context.Background(),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTraceID(tt.ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestOperationRegistry_LogError tests that Log returns error on NATS failure.
//
// This test verifies the error handling for the Log method when NATS
// publish fails (e.g., connection closed, network error).
func TestOperationRegistry_LogError(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Close NATS connection to force error
	nc.Close()

	// Log should return error when NATS publish fails
	err = registry.Log(opID, "info", "Test message")
	assert.Error(t, err, "Log should return error when NATS connection is closed")
	assert.Contains(t, err.Error(), "publish log event")
}

// TestOperationRegistry_ErrorError tests that Error returns error on NATS failure.
//
// This test verifies the error handling for the Error method when NATS
// publish fails (e.g., connection closed, network error).
func TestOperationRegistry_ErrorError(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Close NATS connection to force error
	nc.Close()

	// Error should return error when NATS publish fails
	err = registry.Error(opID, VectorStoreError, fmt.Errorf("test error"))
	assert.Error(t, err, "Error should return error when NATS connection is closed")
	assert.Contains(t, err.Error(), "publish error event")
}

// TestOperationRegistry_CompleteError tests that Complete returns error on NATS failure.
//
// This test verifies the error handling for the Complete method when NATS
// publish fails (e.g., connection closed, network error).
func TestOperationRegistry_CompleteError(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)

	registry := NewOperationRegistry(nc)

	ctx := context.WithValue(context.Background(), ownerIDKey, "owner-123")
	opID := registry.Create(ctx, "test_tool", map[string]string{})

	// Close NATS connection to force error
	nc.Close()

	// Complete should return error when NATS publish fails
	err = registry.Complete(opID, map[string]string{"result": "test"})
	assert.Error(t, err, "Complete should return error when NATS connection is closed")
	assert.Contains(t, err.Error(), "publish completed event")
}

// TestOperationRegistry_LogNonExistentOperation tests Log with non-existent operation.
func TestOperationRegistry_LogNonExistentOperation(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	// Log with non-existent operation should return error
	err = registry.Log("nonexistent", "info", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation not found")
}

// TestOperationRegistry_ErrorNonExistentOperation tests Error with non-existent operation.
func TestOperationRegistry_ErrorNonExistentOperation(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	// Error with non-existent operation should return error
	err = registry.Error("nonexistent", VectorStoreError, fmt.Errorf("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation not found")
}

// TestOperationRegistry_CompleteNonExistentOperation tests Complete with non-existent operation.
func TestOperationRegistry_CompleteNonExistentOperation(t *testing.T) {
	server := startTestNATSServer(t)
	nc, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)

	// Complete with non-existent operation should return error
	err = registry.Complete("nonexistent", map[string]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation not found")
}
