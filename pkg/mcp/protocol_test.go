package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleMCPRequest_Initialize tests the initialize method via POST /mcp endpoint.
//
// This test verifies:
//   - Initialize request is handled correctly
//   - Session ID is generated and returned in header
//   - Protocol version is negotiated
//   - Server capabilities are returned
//   - Response follows JSON-RPC 2.0 format
func TestHandleMCPRequest_Initialize(t *testing.T) {
	tests := []struct {
		name           string
		request        JSONRPCRequest
		acceptHeader   string
		wantStatusCode int
		wantSessionID  bool // Should response include Mcp-Session-Id header
		wantError      bool
	}{
		{
			name: "valid initialize request",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "1",
				Method:  "initialize",
				Params: mustMarshal(InitializeParams{
					ProtocolVersion: "2024-11-05",
					Capabilities:    map[string]interface{}{},
					ClientInfo: ClientInfo{
						Name:    "test-client",
						Version: "1.0.0",
					},
				}),
			},
			acceptHeader:   "application/json, text/event-stream",
			wantStatusCode: http.StatusOK,
			wantSessionID:  true,
			wantError:      false,
		},
		{
			name: "invalid protocol version",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "2",
				Method:  "initialize",
				Params: mustMarshal(InitializeParams{
					ProtocolVersion: "invalid-version",
					Capabilities:    map[string]interface{}{},
					ClientInfo: ClientInfo{
						Name:    "test-client",
						Version: "1.0.0",
					},
				}),
			},
			acceptHeader:   "application/json, text/event-stream",
			wantStatusCode: http.StatusOK,
			wantSessionID:  true, // Still creates session, but may downgrade protocol
			wantError:      false,
		},
		{
			name: "missing accept header",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "3",
				Method:  "initialize",
				Params: mustMarshal(InitializeParams{
					ProtocolVersion: "2024-11-05",
					Capabilities:    map[string]interface{}{},
					ClientInfo: ClientInfo{
						Name:    "test-client",
						Version: "1.0.0",
					},
				}),
			},
			acceptHeader:   "",
			wantStatusCode: http.StatusNotAcceptable,
			wantSessionID:  false,
			wantError:      true,
		},
		{
			name: "wrong accept header",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "4",
				Method:  "initialize",
				Params: mustMarshal(InitializeParams{
					ProtocolVersion: "2024-11-05",
					Capabilities:    map[string]interface{}{},
					ClientInfo: ClientInfo{
						Name:    "test-client",
						Version: "1.0.0",
					},
				}),
			},
			acceptHeader:   "text/html", // Wrong, should include application/json AND text/event-stream
			wantStatusCode: http.StatusNotAcceptable,
			wantSessionID:  false,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set authenticated owner ID (simulating auth middleware)
			c.Set(string(authenticatedOwnerIDKey), "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678")

			// Create server with minimal setup
			server := &Server{
				echo:   e,
				logger: nil, // Use nil logger for tests
			}

			// Execute
			err = server.handleMCPRequest(c)

			// Assert
			require.NoError(t, err) // Echo handlers return nil on c.JSON()
			assert.Equal(t, tt.wantStatusCode, rec.Code)

			// For error cases, verify response contains error field
			if tt.wantError {
				var errResp JSONRPCError
				err = json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err, "Response should be valid JSON-RPC error")
				assert.NotNil(t, errResp.Error, "Response should contain error field")
				return
			}

			// Check session ID header
			sessionID := rec.Header().Get("Mcp-Session-Id")
			if tt.wantSessionID {
				assert.NotEmpty(t, sessionID, "Mcp-Session-Id header should be present")
				assert.Len(t, sessionID, 36, "Session ID should be a valid UUID (36 chars with hyphens)")
			} else {
				assert.Empty(t, sessionID, "Mcp-Session-Id header should not be present")
			}

			// Check response body
			if tt.wantStatusCode == http.StatusOK && !tt.wantError {
				var resp JSONRPCResponse
				err = json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.Equal(t, "2.0", resp.JSONRPC)
				assert.Equal(t, tt.request.ID, resp.ID)

				// Validate initialize result structure
				result, ok := resp.Result.(map[string]interface{})
				require.True(t, ok, "Result should be a map")

				assert.Contains(t, result, "protocolVersion")
				assert.Contains(t, result, "capabilities")
				assert.Contains(t, result, "serverInfo")

				// Validate server info
				serverInfo, ok := result["serverInfo"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "contextd", serverInfo["name"])
			}
		})
	}
}

// TestHandleMCPRequest_ToolsList tests the tools/list method via POST /mcp endpoint.
func TestHandleMCPRequest_ToolsList(t *testing.T) {
	tests := []struct {
		name           string
		request        JSONRPCRequest
		sessionID      string // Provide session ID in header
		wantStatusCode int
		wantError      bool
	}{
		{
			name: "valid tools/list request with session",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "5",
				Method:  "tools/list",
				Params:  json.RawMessage(`{}`),
			},
			sessionID:      "550e8400-e29b-41d4-a716-446655440000",
			wantStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name: "tools/list without session ID",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "6",
				Method:  "tools/list",
				Params:  json.RawMessage(`{}`),
			},
			sessionID:      "",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tt.sessionID != "" {
				req.Header.Set("Mcp-Session-Id", tt.sessionID)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set authenticated owner ID
			ownerID := "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678"
			c.Set(string(authenticatedOwnerIDKey), ownerID)

			// Create server with session store
			server := &Server{
				echo:         e,
				logger:       nil,
				sessionStore: NewSessionStore(),
			}

			// Create session if test provides sessionID
			if tt.sessionID != "" {
				session := &Session{
					ID:              tt.sessionID,
					OwnerID:         ownerID,
					ProtocolVersion: "2024-11-05",
					ClientInfo: ClientInfo{
						Name:    "test-client",
						Version: "1.0.0",
					},
					CreatedAt:      time.Now(),
					LastAccessedAt: time.Now(),
				}
				server.sessionStore.sessions.Store(tt.sessionID, session)
			}

			// Execute
			err = server.handleMCPRequest(c)

			// Assert
			if tt.wantError {
				require.NoError(t, err) // Echo handlers return nil
				assert.Equal(t, http.StatusBadRequest, rec.Code)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatusCode, rec.Code)

			// Validate response contains tools
			var resp JSONRPCResponse
			err = json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)

			result, ok := resp.Result.(map[string]interface{})
			require.True(t, ok)
			tools, ok := result["tools"].([]interface{})
			require.True(t, ok)
			assert.Greater(t, len(tools), 0, "Should return at least one tool")
		})
	}
}

// TestHandleMCPRequest_ToolsCall tests the tools/call method via POST /mcp endpoint.
func TestHandleMCPRequest_ToolsCall(t *testing.T) {
	tests := []struct {
		name           string
		request        JSONRPCRequest
		sessionID      string
		wantStatusCode int
		wantError      bool
	}{
		{
			name: "valid tools/call for status",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "7",
				Method:  "tools/call",
				Params: mustMarshal(ToolsCallParams{
					Name:      "status",
					Arguments: map[string]interface{}{},
				}),
			},
			sessionID:      "550e8400-e29b-41d4-a716-446655440000",
			wantStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name: "tools/call with unknown tool",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "8",
				Method:  "tools/call",
				Params: mustMarshal(ToolsCallParams{
					Name:      "unknown_tool",
					Arguments: map[string]interface{}{},
				}),
			},
			sessionID:      "550e8400-e29b-41d4-a716-446655440000",
			wantStatusCode: http.StatusOK,
			wantError:      true, // Should return JSON-RPC error
		},
		{
			name: "tools/call without session ID",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "9",
				Method:  "tools/call",
				Params: mustMarshal(ToolsCallParams{
					Name:      "status",
					Arguments: map[string]interface{}{},
				}),
			},
			sessionID:      "",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tt.sessionID != "" {
				req.Header.Set("Mcp-Session-Id", tt.sessionID)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set authenticated owner ID
			ownerID := "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678"
			c.Set(string(authenticatedOwnerIDKey), ownerID)

			// Create server with session store
			server := &Server{
				echo:         e,
				logger:       nil,
				sessionStore: NewSessionStore(),
			}

			// Create session if test provides sessionID
			if tt.sessionID != "" {
				session := &Session{
					ID:              tt.sessionID,
					OwnerID:         ownerID,
					ProtocolVersion: "2024-11-05",
					ClientInfo: ClientInfo{
						Name:    "test-client",
						Version: "1.0.0",
					},
					CreatedAt:      time.Now(),
					LastAccessedAt: time.Now(),
				}
				server.sessionStore.sessions.Store(tt.sessionID, session)
			}

			// Execute
			err = server.handleMCPRequest(c)

			// Assert
			require.NoError(t, err) // Echo handlers return nil
			assert.Equal(t, tt.wantStatusCode, rec.Code)

			// For error cases, verify response contains error field
			if tt.wantError && tt.wantStatusCode != http.StatusOK {
				var errResp JSONRPCError
				err = json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err, "Response should be valid JSON-RPC error")
				assert.NotNil(t, errResp.Error, "Response should contain error field")
			}
		})
	}
}

// TestHandleMCPRequest_MethodNotFound tests unknown methods.
func TestHandleMCPRequest_MethodNotFound(t *testing.T) {
	// Setup
	e := echo.New()
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "10",
		Method:  "unknown/method",
		Params:  json.RawMessage(`{}`),
	}
	reqBody, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set authenticated owner ID
	c.Set(string(authenticatedOwnerIDKey), "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678")

	// Create server
	server := &Server{
		echo:   e,
		logger: nil,
	}

	// Execute
	err = server.handleMCPRequest(c)

	// Assert - should return JSON-RPC error for unknown method
	require.NoError(t, err) // HTTP layer succeeds
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp JSONRPCError
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, MethodNotFound, resp.Error.Code)
}

// TestSessionStore_Create tests session creation.
func TestSessionStore_Create(t *testing.T) {
	store := NewSessionStore()
	ownerID := "test-owner-123"
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	session := store.Create(ownerID, params)

	// Validate session
	assert.NotEmpty(t, session.ID, "Session ID should be generated")
	assert.Len(t, session.ID, 36, "Session ID should be a UUID")
	assert.Equal(t, ownerID, session.OwnerID)
	assert.Equal(t, "2024-11-05", session.ProtocolVersion)
	assert.Equal(t, params.ClientInfo, session.ClientInfo)
	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.LastAccessedAt.IsZero())

	// Verify session can be retrieved
	retrieved := store.Get(session.ID)
	require.NotNil(t, retrieved)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.OwnerID, retrieved.OwnerID)
}

// TestSessionStore_Get tests session retrieval.
func TestSessionStore_Get(t *testing.T) {
	store := NewSessionStore()
	ownerID := "test-owner-456"
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	// Create session
	session := store.Create(ownerID, params)
	originalLastAccessed := session.LastAccessedAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Get session
	retrieved := store.Get(session.ID)
	require.NotNil(t, retrieved)

	// Validate LastAccessedAt was updated
	assert.True(t, retrieved.LastAccessedAt.After(originalLastAccessed),
		"LastAccessedAt should be updated on Get")

	// Test non-existent session
	notFound := store.Get("non-existent-session-id")
	assert.Nil(t, notFound, "Non-existent session should return nil")
}

// TestSessionStore_Delete tests session deletion.
func TestSessionStore_Delete(t *testing.T) {
	store := NewSessionStore()
	ownerID := "test-owner-789"
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	// Create session
	session := store.Create(ownerID, params)

	// Verify it exists
	retrieved := store.Get(session.ID)
	require.NotNil(t, retrieved)

	// Delete session
	store.Delete(session.ID)

	// Verify it's gone
	deleted := store.Get(session.ID)
	assert.Nil(t, deleted, "Deleted session should not be retrievable")
}

// TestNegotiateProtocolVersion tests protocol version negotiation.
func TestNegotiateProtocolVersion(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		want      string
	}{
		{
			name:      "supported version",
			requested: "2024-11-05",
			want:      "2024-11-05",
		},
		{
			name:      "unsupported version - defaults to latest",
			requested: "2025-01-01",
			want:      "2024-11-05",
		},
		{
			name:      "empty version - defaults to latest",
			requested: "",
			want:      "2024-11-05",
		},
		{
			name:      "invalid format - defaults to latest",
			requested: "invalid-version",
			want:      "2024-11-05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := negotiateProtocolVersion(tt.requested)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValidateAcceptHeader tests Accept header validation.
func TestValidateAcceptHeader(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{
			name:   "valid - both types present",
			accept: "application/json, text/event-stream",
			want:   true,
		},
		{
			name:   "valid - reversed order",
			accept: "text/event-stream, application/json",
			want:   true,
		},
		{
			name:   "valid - with additional types",
			accept: "text/html, application/json, text/event-stream, */*",
			want:   true,
		},
		{
			name:   "invalid - missing application/json",
			accept: "text/event-stream",
			want:   false,
		},
		{
			name:   "invalid - missing text/event-stream",
			accept: "application/json",
			want:   false,
		},
		{
			name:   "invalid - empty",
			accept: "",
			want:   false,
		},
		{
			name:   "invalid - wrong types",
			accept: "text/html, application/xml",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateAcceptHeader(tt.accept)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestHandleMCPRequest_Resources tests resources/list and resources/read methods.
func TestHandleMCPRequest_Resources(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		sessionID      string
		wantStatusCode int
		wantError      bool
	}{
		{
			name:           "resources/list with valid session",
			method:         "resources/list",
			sessionID:      "550e8400-e29b-41d4-a716-446655440000",
			wantStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name:           "resources/list without session",
			method:         "resources/list",
			sessionID:      "",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
		{
			name:           "resources/read with valid session",
			method:         "resources/read",
			sessionID:      "550e8400-e29b-41d4-a716-446655440000",
			wantStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name:           "resources/read without session",
			method:         "resources/read",
			sessionID:      "",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			request := JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "resource-test",
				Method:  tt.method,
				Params:  json.RawMessage(`{}`),
			}
			reqBody, err := json.Marshal(request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tt.sessionID != "" {
				req.Header.Set("Mcp-Session-Id", tt.sessionID)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set authenticated owner ID
			ownerID := "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678"
			c.Set(string(authenticatedOwnerIDKey), ownerID)

			// Create server with session store
			server := &Server{
				echo:         e,
				logger:       nil,
				sessionStore: NewSessionStore(),
			}

			// Create session if test provides sessionID
			if tt.sessionID != "" {
				session := &Session{
					ID:              tt.sessionID,
					OwnerID:         ownerID,
					ProtocolVersion: "2024-11-05",
					ClientInfo: ClientInfo{
						Name:    "test-client",
						Version: "1.0.0",
					},
					CreatedAt:      time.Now(),
					LastAccessedAt: time.Now(),
				}
				server.sessionStore.sessions.Store(tt.sessionID, session)
			}

			// Execute
			err = server.handleMCPRequest(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantError {
				var errResp JSONRPCError
				err = json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err, "Response should be valid JSON-RPC error")
				assert.NotNil(t, errResp.Error, "Response should contain error field")
			}
		})
	}
}

// TestHandleMCPRequest_Concurrency tests concurrent session creation and access.
func TestHandleMCPRequest_Concurrency(t *testing.T) {
	store := NewSessionStore()
	ownerID := "concurrent-test-owner"

	// Create multiple sessions concurrently
	const numSessions = 100
	sessions := make([]*Session, numSessions)
	var wg sync.WaitGroup

	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			params := InitializeParams{
				ProtocolVersion: "2024-11-05",
				Capabilities:    map[string]interface{}{},
				ClientInfo: ClientInfo{
					Name:    fmt.Sprintf("client-%d", index),
					Version: "1.0.0",
				},
			}
			sessions[index] = store.Create(ownerID, params)
		}(i)
	}

	wg.Wait()

	// Verify all sessions were created with unique IDs
	seenIDs := make(map[string]bool)
	for _, session := range sessions {
		require.NotNil(t, session)
		assert.NotEmpty(t, session.ID)
		assert.False(t, seenIDs[session.ID], "Session IDs should be unique")
		seenIDs[session.ID] = true
	}

	// Verify all sessions can be retrieved concurrently
	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			retrieved := store.Get(sessions[index].ID)
			assert.NotNil(t, retrieved)
			assert.Equal(t, sessions[index].ID, retrieved.ID)
		}(i)
	}

	wg.Wait()
}

// Helper function to marshal data for test params
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
