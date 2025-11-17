package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONRPCRequest_Marshal tests JSON marshaling of request objects.
func TestJSONRPCRequest_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		req     JSONRPCRequest
		wantErr bool
		check   func(t *testing.T, data []byte)
	}{
		{
			name: "valid request with params",
			req: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "test-123",
				Method:  "checkpoint_save",
				Params:  json.RawMessage(`{"content":"test"}`),
			},
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), `"jsonrpc":"2.0"`)
				assert.Contains(t, string(data), `"id":"test-123"`)
				assert.Contains(t, string(data), `"method":"checkpoint_save"`)
			},
		},
		{
			name: "request without params",
			req: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "test-456",
				Method:  "status",
			},
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), `"jsonrpc":"2.0"`)
				assert.Contains(t, string(data), `"method":"status"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, data)
			}
		})
	}
}

// TestJSONRPCRequest_Unmarshal tests JSON unmarshaling of request objects.
func TestJSONRPCRequest_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    JSONRPCRequest
		wantErr bool
	}{
		{
			name: "valid request",
			data: `{"jsonrpc":"2.0","id":"test-123","method":"checkpoint_save","params":{"content":"test"}}`,
			want: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "test-123",
				Method:  "checkpoint_save",
				Params:  json.RawMessage(`{"content":"test"}`),
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			data:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got JSONRPCRequest
			err := json.Unmarshal([]byte(tt.data), &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.JSONRPC, got.JSONRPC)
			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.Method, got.Method)
		})
	}
}

// TestJSONRPCResponse_Marshal tests JSON marshaling of successful responses.
func TestJSONRPCResponse_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		resp    JSONRPCResponse
		wantErr bool
		check   func(t *testing.T, data []byte)
	}{
		{
			name: "response with string result",
			resp: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      "test-123",
				Result: map[string]string{
					"operation_id": "op-456",
					"status":       "pending",
				},
			},
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), `"jsonrpc":"2.0"`)
				assert.Contains(t, string(data), `"id":"test-123"`)
				assert.Contains(t, string(data), `"result"`)
			},
		},
		{
			name: "response with complex result",
			resp: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      "test-789",
				Result: []map[string]interface{}{
					{"id": "ckpt-1", "timestamp": "2025-01-15T10:00:00Z"},
					{"id": "ckpt-2", "timestamp": "2025-01-15T11:00:00Z"},
				},
			},
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), `"ckpt-1"`)
				assert.Contains(t, string(data), `"ckpt-2"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, data)
			}
		})
	}
}

// TestJSONRPCError_Marshal tests JSON marshaling of error responses.
func TestJSONRPCError_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		resp    JSONRPCError
		wantErr bool
		check   func(t *testing.T, data []byte)
	}{
		{
			name: "error with trace context",
			resp: JSONRPCError{
				JSONRPC: "2.0",
				ID:      "test-123",
				Error: &ErrorDetail{
					Code:    VectorStoreError,
					Message: "Failed to save checkpoint",
					Data: map[string]interface{}{
						"trace_id":   "4bf92f3577b34da6a3ce929d0e0e4736",
						"error_type": "VectorStoreError",
						"timestamp":  "2025-01-15T10:30:45Z",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), `"error"`)
				assert.Contains(t, string(data), `"code":-32000`)
				assert.Contains(t, string(data), `"Failed to save checkpoint"`)
				assert.Contains(t, string(data), `"trace_id"`)
			},
		},
		{
			name: "standard JSON-RPC error",
			resp: JSONRPCError{
				JSONRPC: "2.0",
				ID:      "test-456",
				Error: &ErrorDetail{
					Code:    InvalidParams,
					Message: "Invalid request parameters",
					Data:    nil,
				},
			},
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), `"code":-32602`)
				assert.Contains(t, string(data), `"Invalid request parameters"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, data)
			}
		})
	}
}

// TestErrorCodes verifies all error codes are distinct.
func TestErrorCodes(t *testing.T) {
	codes := map[int]string{
		ParseError:        "ParseError",
		InvalidRequest:    "InvalidRequest",
		MethodNotFound:    "MethodNotFound",
		InvalidParams:     "InvalidParams",
		InternalError:     "InternalError",
		VectorStoreError:  "VectorStoreError",
		SecretScrubError:  "SecretScrubError",
		GitError:          "GitError",
		NATSError:         "NATSError",
		EmbeddingError:    "EmbeddingError",
		AuthError:         "AuthError",
		ConfigError:       "ConfigError",
		OperationNotFound: "OperationNotFound",
	}

	// Check uniqueness
	seen := make(map[int]bool)
	for code, name := range codes {
		assert.False(t, seen[code], "duplicate error code %d for %s", code, name)
		seen[code] = true
	}

	// Check codes are in valid ranges
	for code := range codes {
		if code < -32768 || code > -32000 {
			// Must be standard JSON-RPC error or in application-specific range
			assert.True(t,
				code == ParseError || code == InvalidRequest || code == MethodNotFound || code == InvalidParams || code == InternalError,
				"code %d outside valid ranges", code,
			)
		}
	}
}

// TestOperation_Lifecycle tests operation state transitions.
func TestOperation_Lifecycle(t *testing.T) {
	now := time.Now()

	op := &Operation{
		ID:        "op-123",
		OwnerID:   "owner-456",
		Tool:      "checkpoint_save",
		Status:    "pending",
		Params:    map[string]string{"content": "test"},
		TraceID:   "trace-789",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Verify initial state
	assert.Equal(t, "pending", op.Status)
	assert.Nil(t, op.Result)
	assert.Nil(t, op.Error)

	// Test marshaling
	data, err := json.Marshal(op)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"status":"pending"`)
	assert.Contains(t, string(data), `"trace_id":"trace-789"`)

	// Test unmarshaling
	var decoded Operation
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, op.ID, decoded.ID)
	assert.Equal(t, op.Status, decoded.Status)
	assert.Equal(t, op.Tool, decoded.Tool)
}
