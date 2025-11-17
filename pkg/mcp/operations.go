package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Context keys for operation metadata and authentication.
type contextKey string

const (
	// authenticatedOwnerIDKey is the context key for authenticated owner ID.
	// This MUST be set by authentication middleware after validating JWT/token.
	// SECURITY CRITICAL: Never accept owner IDs from user-controlled sources.
	authenticatedOwnerIDKey contextKey = "authenticated_owner_id"
	ownerIDKey              contextKey = "owner_id"
	traceIDKey              contextKey = "trace_id"
)

// OperationRegistry manages operation lifecycle with NATS JetStream persistence.
//
// The registry tracks operations in memory for fast lookups and publishes
// all operation events to NATS for persistence and SSE streaming.
//
// Operation events are published to subjects:
//   - operations.{owner_id}.{operation_id}.started
//   - operations.{owner_id}.{operation_id}.progress
//   - operations.{owner_id}.{operation_id}.log
//   - operations.{owner_id}.{operation_id}.error
//   - operations.{owner_id}.{operation_id}.completed
//
// Example usage:
//
//	registry := NewOperationRegistry(natsConn)
//	opID := registry.Create(ctx, "checkpoint_save", params)
//	registry.Started(opID)
//	registry.Progress(opID, 50, "Processing...")
//	registry.Complete(opID, result)
type OperationRegistry struct {
	nats       *nats.Conn
	operations sync.Map // operation_id -> *Operation
}

// NewOperationRegistry creates a new operation registry with NATS connection.
//
// The registry uses sync.Map for concurrent-safe operation storage and
// publishes events to NATS for persistence and streaming.
func NewOperationRegistry(nc *nats.Conn) *OperationRegistry {
	return &OperationRegistry{
		nats: nc,
	}
}

// Create creates a new operation and returns its ID.
//
// The operation is created in "pending" state and stored in the registry.
// Owner ID and trace ID are extracted from the context if present.
//
// Example:
//
//	ctx := context.WithValue(ctx, ownerIDKey, "owner-123")
//	ctx = context.WithValue(ctx, traceIDKey, "trace-456")
//	opID := registry.Create(ctx, "checkpoint_save", params)
func (r *OperationRegistry) Create(ctx context.Context, tool string, params interface{}) string {
	opID := uuid.New().String()
	ownerID := getOwnerID(ctx)
	traceID := getTraceID(ctx)

	op := &Operation{
		ID:        opID,
		OwnerID:   ownerID,
		Tool:      tool,
		Status:    "pending",
		Params:    params,
		TraceID:   traceID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	r.operations.Store(opID, op)

	return opID
}

// Started publishes "started" event and updates operation status to "running".
//
// This should be called when the operation begins execution.
// The event is published to NATS subject:
//
//	operations.{owner_id}.{operation_id}.started
//
// Returns error if operation not found, JSON marshaling fails, or NATS publish fails.
func (r *OperationRegistry) Started(opID string) error {
	value, ok := r.operations.Load(opID)
	if !ok {
		return fmt.Errorf("operation not found: %s", opID)
	}

	operation := value.(*Operation)
	operation.Status = "running"
	operation.UpdatedAt = time.Now()

	subject := fmt.Sprintf("operations.%s.%s.started", operation.OwnerID, opID)
	data, err := json.Marshal(operation)
	if err != nil {
		return fmt.Errorf("marshal operation: %w", err)
	}

	if err := r.nats.Publish(subject, data); err != nil {
		return fmt.Errorf("publish started event: %w", err)
	}

	return nil
}

// Progress publishes "progress" event with percent and message.
//
// Use this to report progress updates for long-running operations.
// The event is published to NATS subject:
//
//	operations.{owner_id}.{operation_id}.progress
//
// Example:
//
//	registry.Progress(opID, 50, "Processing file 5 of 10")
//
// Returns error if operation not found, JSON marshaling fails, or NATS publish fails.
func (r *OperationRegistry) Progress(opID string, percent int, message string) error {
	value, ok := r.operations.Load(opID)
	if !ok {
		return fmt.Errorf("operation not found: %s", opID)
	}

	operation := value.(*Operation)
	operation.UpdatedAt = time.Now()

	subject := fmt.Sprintf("operations.%s.%s.progress", operation.OwnerID, opID)
	data, err := json.Marshal(map[string]interface{}{
		"id":        opID,
		"percent":   percent,
		"message":   message,
		"timestamp": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}

	if err := r.nats.Publish(subject, data); err != nil {
		return fmt.Errorf("publish progress event: %w", err)
	}

	return nil
}

// Log publishes "log" event with level and message.
//
// Use this to publish informational log messages during operation execution.
// The event is published to NATS subject:
//
//	operations.{owner_id}.{operation_id}.log
//
// Example:
//
//	if err := registry.Log(opID, "info", "Indexed 123 files"); err != nil {
//	    return fmt.Errorf("failed to log: %w", err)
//	}
//
// Returns error if operation not found, JSON marshaling fails, or NATS publish fails.
func (r *OperationRegistry) Log(opID string, level string, message string) error {
	value, ok := r.operations.Load(opID)
	if !ok {
		return fmt.Errorf("operation not found: %s", opID)
	}

	operation := value.(*Operation)

	subject := fmt.Sprintf("operations.%s.%s.log", operation.OwnerID, opID)
	data, err := json.Marshal(map[string]interface{}{
		"id":        opID,
		"level":     level,
		"message":   message,
		"timestamp": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("marshal log event: %w", err)
	}

	if err := r.nats.Publish(subject, data); err != nil {
		return fmt.Errorf("publish log event: %w", err)
	}

	return nil
}

// Error publishes "error" event and marks operation as failed.
//
// This should be called when the operation encounters an error.
// The event is published to NATS subject:
//
//	operations.{owner_id}.{operation_id}.error
//
// Example:
//
//	if err := registry.Error(opID, VectorStoreError, fmt.Errorf("connection timeout")); err != nil {
//	    return fmt.Errorf("failed to publish error: %w", err)
//	}
//
// Returns error if operation not found, JSON marshaling fails, or NATS publish fails.
func (r *OperationRegistry) Error(opID string, code int, err error) error {
	value, ok := r.operations.Load(opID)
	if !ok {
		return fmt.Errorf("operation not found: %s", opID)
	}

	operation := value.(*Operation)
	operation.Status = "failed"
	operation.Error = &ErrorDetail{
		Code:    code,
		Message: err.Error(),
		Data: map[string]interface{}{
			"trace_id":  operation.TraceID,
			"timestamp": time.Now(),
		},
	}
	operation.UpdatedAt = time.Now()

	subject := fmt.Sprintf("operations.%s.%s.error", operation.OwnerID, opID)
	data, jsonErr := json.Marshal(operation.Error)
	if jsonErr != nil {
		return fmt.Errorf("marshal error event: %w", jsonErr)
	}

	if publishErr := r.nats.Publish(subject, data); publishErr != nil {
		return fmt.Errorf("publish error event: %w", publishErr)
	}

	return nil
}

// Complete publishes "completed" event and marks operation as completed.
//
// This should be called when the operation finishes successfully.
// The event is published to NATS subject:
//
//	operations.{owner_id}.{operation_id}.completed
//
// The operation is automatically cleaned up after 1 hour TTL.
//
// Example:
//
//	result := map[string]interface{}{"checkpoint_id": "ckpt-123"}
//	if err := registry.Complete(opID, result); err != nil {
//	    return fmt.Errorf("failed to complete operation: %w", err)
//	}
//
// Returns error if operation not found, JSON marshaling fails, or NATS publish fails.
func (r *OperationRegistry) Complete(opID string, result interface{}) error {
	value, ok := r.operations.Load(opID)
	if !ok {
		return fmt.Errorf("operation not found: %s", opID)
	}

	operation := value.(*Operation)
	operation.Status = "completed"
	operation.Result = result
	operation.UpdatedAt = time.Now()

	subject := fmt.Sprintf("operations.%s.%s.completed", operation.OwnerID, opID)
	data, err := json.Marshal(map[string]interface{}{
		"id":          opID,
		"result":      result,
		"duration_ms": time.Since(operation.CreatedAt).Milliseconds(),
		"timestamp":   time.Now(),
	})
	if err != nil {
		return fmt.Errorf("marshal completed event: %w", err)
	}

	if err := r.nats.Publish(subject, data); err != nil {
		return fmt.Errorf("publish completed event: %w", err)
	}

	// Schedule cleanup (remove from memory after TTL)
	go r.scheduleCleanup(opID, 1*time.Hour)

	return nil
}

// Get retrieves an operation by ID.
//
// Returns error if the operation does not exist.
func (r *OperationRegistry) Get(opID string) (*Operation, error) {
	value, ok := r.operations.Load(opID)
	if !ok {
		return nil, fmt.Errorf("operation not found: %s", opID)
	}
	return value.(*Operation), nil
}

// scheduleCleanup removes completed/failed operations after TTL.
//
// This prevents the in-memory registry from growing indefinitely.
// The NATS stream retains events for 24 hours regardless of cleanup.
func (r *OperationRegistry) scheduleCleanup(opID string, ttl time.Duration) {
	time.Sleep(ttl)
	r.operations.Delete(opID)
}

// getOwnerID extracts owner ID from context.
//
// Returns "unknown" if owner ID is not present in context.
func getOwnerID(ctx context.Context) string {
	if ownerID, ok := ctx.Value(ownerIDKey).(string); ok {
		return ownerID
	}
	return "unknown"
}

// getTraceID extracts trace ID from context.
//
// Returns empty string if trace ID is not present in context.
func getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}
