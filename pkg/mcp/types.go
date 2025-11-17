// Package mcp provides Model Context Protocol server implementation over HTTP with SSE streaming.
//
// This package implements JSON-RPC 2.0 protocol for tool invocation, NATS-based
// operation tracking, and Server-Sent Events (SSE) for long-running operations.
//
// Example usage:
//
//	server := mcp.NewServer(cfg, services)
//	if err := server.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
package mcp

import (
	"encoding/json"
	"errors"
	"time"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request.
//
// The method field is typically implicit from the HTTP endpoint path,
// but included here for protocol compliance.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"` // Always "2.0"
	ID      string          `json:"id"`      // Request ID (UUID)
	Method  string          `json:"method"`  // Tool name (implicit from endpoint)
	Params  json.RawMessage `json:"params"`  // Tool-specific parameters
}

// JSONRPCResponse represents a successful JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"` // Always "2.0"
	ID      string      `json:"id"`      // Matches request ID
	Result  interface{} `json:"result"`  // Tool-specific result
}

// JSONRPCError represents an error JSON-RPC 2.0 response.
//
// Error responses include enhanced debugging context in the Data field,
// including trace IDs, error types, and timestamps for correlation with
// observability systems.
type JSONRPCError struct {
	JSONRPC string       `json:"jsonrpc"` // Always "2.0"
	ID      string       `json:"id"`      // Matches request ID
	Error   *ErrorDetail `json:"error"`   // Error details with context
}

// ErrorDetail provides enhanced error information beyond JSON-RPC 2.0 spec.
//
// The Data field contains debugging context such as:
//   - trace_id: OTLP trace ID for correlation
//   - error_type: Go error type name
//   - timestamp: ISO 8601 timestamp
//   - operation_id: NATS operation ID (if applicable)
//   - owner_id: Owner hash (for multi-tenant debugging)
//   - cause: Root cause error message
//   - stack_trace: Full stack trace (development mode only)
type ErrorDetail struct {
	Code    int                    `json:"code"`    // JSON-RPC error code
	Message string                 `json:"message"` // Human-readable message
	Data    map[string]interface{} `json:"data"`    // Enhanced debugging context
}

// JSON-RPC 2.0 standard error codes.
const (
	ParseError     = -32700 // Invalid JSON
	InvalidRequest = -32600 // Invalid Request object
	MethodNotFound = -32601 // Tool doesn't exist
	InvalidParams  = -32602 // Invalid tool params
	InternalError  = -32603 // Internal server error
)

// Application-specific error codes (reserved range: -32000 to -32099).
const (
	VectorStoreError  = -32000 // Vector database errors
	SecretScrubError  = -32001 // Secret redaction errors
	GitError          = -32002 // Git operation errors
	NATSError         = -32003 // NATS messaging errors
	EmbeddingError    = -32004 // Embedding generation errors
	AuthError         = -32005 // Authentication/authorization errors
	ConfigError       = -32006 // Configuration errors
	OperationNotFound = -32007 // Operation ID not found
)

// Sentinel errors for common validation failures.
var (
	ErrInvalidParams        = errors.New("invalid parameters")
	ErrUnauthenticated      = errors.New("unauthenticated request: owner ID required")
	ErrInvalidOwnerIDFormat = errors.New("invalid owner ID format")
)

// Operation represents a tracked operation with NATS persistence.
//
// Operations are created for long-running tasks (e.g., index_repository)
// and provide progress streaming via SSE. Each operation publishes events
// to NATS subjects: operations.{owner_id}.{operation_id}.{event_type}
//
// Operation lifecycle states:
//   - pending: Created but not started
//   - running: Currently executing
//   - completed: Finished successfully
//   - failed: Finished with error
type Operation struct {
	ID        string       `json:"id"`               // Operation UUID
	OwnerID   string       `json:"owner_id"`         // Owner hash for multi-tenant isolation
	Tool      string       `json:"tool"`             // Tool name (e.g., "checkpoint_save")
	Status    string       `json:"status"`           // pending|running|completed|failed
	Params    interface{}  `json:"params"`           // Tool-specific parameters
	Result    interface{}  `json:"result,omitempty"` // Tool result (when completed)
	Error     *ErrorDetail `json:"error,omitempty"`  // Error details (when failed)
	TraceID   string       `json:"trace_id"`         // OTLP trace ID
	CreatedAt time.Time    `json:"created_at"`       // Operation creation time
	UpdatedAt time.Time    `json:"updated_at"`       // Last update time
}
