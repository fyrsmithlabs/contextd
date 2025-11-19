package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/fyrsmithlabs/contextd/pkg/auth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SessionStore manages MCP protocol sessions in memory.
//
// Sessions are created during initialize handshake and tracked via
// Mcp-Session-Id header. This implementation uses an in-memory map
// with mutex protection for concurrent access.
//
// Future: Could be replaced with Redis or database-backed store
// for distributed deployments and session persistence.
type SessionStore struct {
	sessions sync.Map // map[string]*Session
}

// NewSessionStore creates a new in-memory session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{}
}

// Create creates a new session with the given owner ID and client info.
//
// Returns the created session with a generated UUID.
func (s *SessionStore) Create(ownerID string, params InitializeParams) *Session {
	session := &Session{
		ID:              uuid.New().String(),
		OwnerID:         ownerID,
		ProtocolVersion: negotiateProtocolVersion(params.ProtocolVersion),
		ClientInfo:      params.ClientInfo,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
	}
	s.sessions.Store(session.ID, session)
	return session
}

// Get retrieves a session by ID.
//
// Returns nil if session doesn't exist.
func (s *SessionStore) Get(sessionID string) *Session {
	if val, ok := s.sessions.Load(sessionID); ok {
		if session, ok := val.(*Session); ok {
			// Update last accessed time
			session.LastAccessedAt = time.Now()
			s.sessions.Store(sessionID, session)
			return session
		}
	}
	return nil
}

// Delete removes a session from the store.
func (s *SessionStore) Delete(sessionID string) {
	s.sessions.Delete(sessionID)
}

// negotiateProtocolVersion negotiates the protocol version between client and server.
//
// Currently supports:
//   - 2024-11-05 (MCP Streamable HTTP spec)
//
// Defaults to 2024-11-05 if client requests unsupported version.
func negotiateProtocolVersion(requested string) string {
	supportedVersions := []string{
		"2024-11-05",
	}

	for _, supported := range supportedVersions {
		if requested == supported {
			return supported
		}
	}

	// Default to latest supported version
	return "2024-11-05"
}

// handleMCPRequest handles POST /mcp with JSON-RPC 2.0 method routing.
//
// This is the main MCP protocol endpoint that routes requests based on
// the JSON-RPC method field:
//   - initialize: Create new session and return capabilities
//   - tools/list: List available tools
//   - tools/call: Call a specific tool
//   - resources/list: List available resources
//   - resources/read: Read a resource
//
// Per MCP spec 2025-03-26, this endpoint MUST:
//   - Validate Accept header includes both application/json AND text/event-stream
//   - Return Mcp-Session-Id header after successful initialize
//   - Require Mcp-Session-Id header for all non-initialize requests
//   - Return JSON-RPC 2.0 formatted responses
func (s *Server) handleMCPRequest(c echo.Context) error {
	// SECURITY: Validate Accept header per MCP spec
	// Client must accept BOTH application/json AND text/event-stream
	accept := c.Request().Header.Get("Accept")
	if !validateAcceptHeader(accept) {
		return c.JSON(http.StatusNotAcceptable, JSONRPCError{
			JSONRPC: "2.0",
			ID:      "",
			Error: &ErrorDetail{
				Code:    -32000,
				Message: "Not Acceptable: Client must accept both application/json and text/event-stream",
				Data: map[string]interface{}{
					"accept_header": accept,
					"required":      "application/json, text/event-stream",
				},
			},
		})
	}

	// MVP: Set default owner ID if not authenticated
	// TODO: Remove when OAuth 2.0 is implemented
	if _, ok := c.Get(string(authenticatedOwnerIDKey)).(string); !ok {
		// Use OS user to derive owner ID (same as auth middleware)
		if currentUser, err := user.Current(); err == nil {
			ownerID, _ := auth.DeriveOwnerID(currentUser.Username)
			c.Set(string(authenticatedOwnerIDKey), ownerID)
		}
	}

	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Route based on method
	switch req.Method {
	case "initialize":
		return s.handleInitialize(c, req)

	case "tools/list":
		return s.handleToolsListMethod(c, req)

	case "tools/call":
		return s.handleToolsCallMethod(c, req)

	case "resources/list":
		return s.handleResourcesListMethod(c, req)

	case "resources/read":
		return s.handleResourcesReadMethod(c, req)

	default:
		return JSONRPCErrorWithContext(c, req.ID, MethodNotFound,
			fmt.Errorf("unknown method: %s", req.Method))
	}
}

// validateAcceptHeader checks if Accept header includes required media types.
//
// Per MCP spec, client MUST accept both:
//   - application/json (for JSON-RPC responses)
//   - text/event-stream (for SSE streaming)
func validateAcceptHeader(accept string) bool {
	if accept == "" {
		return false
	}

	hasJSON := strings.Contains(accept, "application/json")
	hasSSE := strings.Contains(accept, "text/event-stream")

	return hasJSON && hasSSE
}

// handleInitialize handles the initialize method.
//
// This method:
//  1. Validates authentication (owner ID must be set by middleware)
//  2. Creates a new session
//  3. Returns Mcp-Session-Id header
//  4. Returns server capabilities and info
//
// The initialize method does NOT require an existing session ID
// (it's the method that creates the session).
func (s *Server) handleInitialize(c echo.Context, req JSONRPCRequest) error {
	// Extract authenticated owner ID
	ownerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, AuthError, err)
	}

	// Parse initialize params
	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Create session
	if s.sessionStore == nil {
		s.sessionStore = NewSessionStore()
	}
	session := s.sessionStore.Create(ownerID, params)

	// Set session ID header per MCP spec
	c.Response().Header().Set("Mcp-Session-Id", session.ID)
	c.Response().Header().Set("Mcp-Protocol-Version", session.ProtocolVersion)

	// Return initialize result
	result := InitializeResult{
		ProtocolVersion: session.ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools:     map[string]interface{}{},
			Resources: map[string]interface{}{},
		},
		ServerInfo: ServerInfo{
			Name:    "contextd",
			Version: "0.9.0-rc-1",
		},
	}

	return JSONRPCSuccess(c, req.ID, result)
}

// handleToolsListMethod handles the tools/list method via /mcp endpoint.
//
// Requires valid session ID in Mcp-Session-Id header.
func (s *Server) handleToolsListMethod(c echo.Context, req JSONRPCRequest) error {
	// Validate session
	if err := s.validateSession(c); err != nil {
		return c.JSON(http.StatusBadRequest, JSONRPCError{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorDetail{
				Code:    AuthError,
				Message: "Bad Request: Valid session ID required",
				Data:    map[string]interface{}{"details": err.Error()},
			},
		})
	}

	// Reuse existing tools/list handler logic
	return s.handleToolsList(c)
}

// handleToolsCallMethod handles the tools/call method via /mcp endpoint.
//
// Requires valid session ID in Mcp-Session-Id header.
// Routes to appropriate tool handler based on tool name.
func (s *Server) handleToolsCallMethod(c echo.Context, req JSONRPCRequest) error {
	// Validate session
	if err := s.validateSession(c); err != nil {
		return c.JSON(http.StatusBadRequest, JSONRPCError{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorDetail{
				Code:    AuthError,
				Message: "Bad Request: Valid session ID required",
				Data:    map[string]interface{}{"details": err.Error()},
			},
		})
	}

	// Parse tools/call params
	var params ToolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Route to tool handler based on name
	switch params.Name {
	case "status":
		// Wrap arguments back into JSON-RPC request format for existing handler
		wrappedReq := JSONRPCRequest{
			JSONRPC: req.JSONRPC,
			ID:      req.ID,
			Method:  "status",
			Params:  mustMarshalJSON(params.Arguments),
		}
		return s.handleStatusTool(c, wrappedReq)

	case "checkpoint_save", "checkpoint_search", "checkpoint_list":
		// Wrap arguments back into JSON-RPC request format and update request body
		wrappedReq := JSONRPCRequest{
			JSONRPC: req.JSONRPC,
			ID:      req.ID,
			Method:  params.Name,
			Params:  mustMarshalJSON(params.Arguments),
		}
		// Update request body for handler to bind
		c.Request().Body = io.NopCloser(bytes.NewReader(mustMarshalJSON(wrappedReq)))

		switch params.Name {
		case "checkpoint_save":
			return s.handleCheckpointSave(c)
		case "checkpoint_search":
			return s.handleCheckpointSearch(c)
		case "checkpoint_list":
			return s.handleCheckpointList(c)
		default:
			return JSONRPCErrorWithContext(c, req.ID, InternalError,
				fmt.Errorf("tool %s routing not implemented", params.Name))
		}

	case "remediation_save", "remediation_search":
		wrappedReq := JSONRPCRequest{
			JSONRPC: req.JSONRPC,
			ID:      req.ID,
			Method:  params.Name,
			Params:  mustMarshalJSON(params.Arguments),
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(mustMarshalJSON(wrappedReq)))

		switch params.Name {
		case "remediation_save":
			return s.handleRemediationSave(c)
		case "remediation_search":
			return s.handleRemediationSearch(c)
		default:
			return JSONRPCErrorWithContext(c, req.ID, InternalError,
				fmt.Errorf("tool %s routing not implemented", params.Name))
		}

	case "skill_save", "skill_search":
		wrappedReq := JSONRPCRequest{
			JSONRPC: req.JSONRPC,
			ID:      req.ID,
			Method:  params.Name,
			Params:  mustMarshalJSON(params.Arguments),
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(mustMarshalJSON(wrappedReq)))

		switch params.Name {
		case "skill_save":
			return s.handleSkillSave(c)
		case "skill_search":
			return s.handleSkillSearch(c)
		default:
			return JSONRPCErrorWithContext(c, req.ID, InternalError,
				fmt.Errorf("tool %s routing not implemented", params.Name))
		}

	case "index_repository":
		wrappedReq := JSONRPCRequest{
			JSONRPC: req.JSONRPC,
			ID:      req.ID,
			Method:  params.Name,
			Params:  mustMarshalJSON(params.Arguments),
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(mustMarshalJSON(wrappedReq)))
		return s.handleIndexRepository(c)

	case "collection_create", "collection_delete", "collection_list":
		wrappedReq := JSONRPCRequest{
			JSONRPC: req.JSONRPC,
			ID:      req.ID,
			Method:  params.Name,
			Params:  mustMarshalJSON(params.Arguments),
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(mustMarshalJSON(wrappedReq)))

		switch params.Name {
		case "collection_create":
			return s.handleCollectionCreate(c)
		case "collection_delete":
			return s.handleCollectionDelete(c)
		case "collection_list":
			return s.handleCollectionList(c)
		default:
			return JSONRPCErrorWithContext(c, req.ID, InternalError,
				fmt.Errorf("tool %s routing not implemented", params.Name))
		}

	default:
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams,
			fmt.Errorf("unknown tool: %s", params.Name))
	}
}

// handleResourcesListMethod handles the resources/list method via /mcp endpoint.
//
// Requires valid session ID in Mcp-Session-Id header.
func (s *Server) handleResourcesListMethod(c echo.Context, req JSONRPCRequest) error {
	// Validate session
	if err := s.validateSession(c); err != nil {
		return c.JSON(http.StatusBadRequest, JSONRPCError{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorDetail{
				Code:    AuthError,
				Message: "Bad Request: Valid session ID required",
				Data:    map[string]interface{}{"details": err.Error()},
			},
		})
	}

	// Reuse existing resources/list handler
	return s.handleResourcesList(c)
}

// handleResourcesReadMethod handles the resources/read method via /mcp endpoint.
//
// Requires valid session ID in Mcp-Session-Id header.
func (s *Server) handleResourcesReadMethod(c echo.Context, req JSONRPCRequest) error {
	// Validate session
	if err := s.validateSession(c); err != nil {
		return c.JSON(http.StatusBadRequest, JSONRPCError{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorDetail{
				Code:    AuthError,
				Message: "Bad Request: Valid session ID required",
				Data:    map[string]interface{}{"details": err.Error()},
			},
		})
	}

	// Reuse existing resources/read handler
	// The handler will extract params from context
	return s.handleResourceRead(c)
}

// validateSession checks if a valid session ID is provided in the request header.
//
// Returns error if:
//   - Mcp-Session-Id header is missing
//   - Session ID is invalid (not found in store)
func (s *Server) validateSession(c echo.Context) error {
	sessionID := c.Request().Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		return fmt.Errorf("missing Mcp-Session-Id header")
	}

	if s.sessionStore == nil {
		return fmt.Errorf("session store not initialized")
	}

	session := s.sessionStore.Get(sessionID)
	if session == nil {
		return fmt.Errorf("invalid session ID: %s", sessionID)
	}

	return nil
}

// handleStatusTool handles the status tool call.
//
// This is a minimal implementation that returns server status.
// It's called by handleToolsCallMethod when tool name is "status".
func (s *Server) handleStatusTool(c echo.Context, req JSONRPCRequest) error {
	// Return status response
	return JSONRPCSuccess(c, req.ID, map[string]interface{}{
		"service": "contextd",
		"status":  "healthy",
		"version": "0.9.0-rc-1",
	})
}

// mustMarshalJSON marshals a value to JSON or panics.
//
// Used for converting tool arguments back to JSON-RPC params.
func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return data
}
