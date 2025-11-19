package mcp

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// hexPattern matches valid hexadecimal strings (for owner ID validation).
var hexPattern = regexp.MustCompile(`^[a-fA-F0-9]+$`)

// JSONRPCSuccess returns a successful JSON-RPC 2.0 response.
//
// Use this helper to return successful results from MCP tool handlers.
//
// Example:
//
//	return JSONRPCSuccess(c, req.ID, map[string]string{
//	    "operation_id": opID,
//	    "status": "pending",
//	})
func JSONRPCSuccess(c echo.Context, id interface{}, result interface{}) error {
	return c.JSON(http.StatusOK, JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

// JSONRPCErrorWithContext returns a JSON-RPC 2.0 error response with enhanced context.
//
// This helper automatically extracts trace ID from the request context and
// includes it in the error response data for correlation with observability systems.
//
// Example:
//
//	if err := validate(); err != nil {
//	    return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
//	}
func JSONRPCErrorWithContext(c echo.Context, id interface{}, code int, err error) error {
	// Extract trace ID from request headers or context
	traceID := c.Response().Header().Get("X-Trace-Id")
	if traceID == "" {
		traceID = c.Request().Header.Get("X-Request-ID")
	}

	return c.JSON(http.StatusOK, JSONRPCError{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorDetail{
			Code:    code,
			Message: err.Error(),
			Data: map[string]interface{}{
				"trace_id":   traceID,
				"error_type": fmt.Sprintf("%T", err),
				"timestamp":  time.Now().Format(time.RFC3339),
			},
		},
	})
}

// ExtractOwnerID validates and extracts owner ID from authenticated context.
//
// SECURITY CRITICAL: Owner ID MUST be derived from authenticated user credentials
// (e.g., JWT claims set by authentication middleware), NOT from user-controlled
// headers or path parameters.
//
// This function enforces multi-tenant isolation by ensuring owner IDs come from
// trusted authentication sources only. Never accept owner IDs from:
//   - HTTP headers (X-Owner-ID) - user-controlled
//   - Path parameters (/api/{owner_id}) - user-controlled
//   - Query parameters (?owner_id=...) - user-controlled
//
// Returns error if:
//   - Owner ID is not present in context (unauthenticated request)
//   - Owner ID format is invalid (not 64-character hex string for SHA256)
//
// Example usage:
//
//	// In authentication middleware (sets authenticated owner ID):
//	c.Set(authenticatedOwnerIDKey, deriveOwnerID(jwtClaims))
//
//	// In handler (extracts validated owner ID):
//	ownerID, err := ExtractOwnerID(c)
//	if err != nil {
//	    return JSONRPCErrorWithContext(c, "", AuthError, err)
//	}
func ExtractOwnerID(c echo.Context) (string, error) {
	// Extract from authenticated context ONLY
	// This value is set by authentication middleware after validating JWT/token
	ownerID, ok := c.Get(string(authenticatedOwnerIDKey)).(string)
	if !ok || ownerID == "" {
		return "", ErrUnauthenticated
	}

	// Validate format: SHA256 produces 64-character hex string
	// This prevents injection attacks via malformed owner IDs
	if len(ownerID) != 64 || !hexPattern.MatchString(ownerID) {
		return "", ErrInvalidOwnerIDFormat
	}

	return ownerID, nil
}

// ParseCollectionURI parses a collection URI into its components.
//
// Expected format: collection://owner_<hash>/project_<hash>/<branch>
//
// Returns:
//   - ownerID: The owner hash (without "owner_" prefix)
//   - collectionName: The full collection name (owner_<hash>/project_<hash>/<branch>)
//   - error: If URI format is invalid
//
// Example:
//
//	ownerID, collectionName, err := ParseCollectionURI("collection://owner_abc123/project_def456/main")
//	// Returns: "abc123", "owner_abc123/project_def456/main", nil
func ParseCollectionURI(uri string) (string, string, error) {
	// Validate URI scheme
	if !strings.HasPrefix(uri, "collection://") {
		return "", "", fmt.Errorf("invalid URI scheme: expected 'collection://', got '%s'", uri)
	}

	// Extract collection name (everything after scheme)
	collectionName := strings.TrimPrefix(uri, "collection://")
	if collectionName == "" {
		return "", "", fmt.Errorf("empty collection name")
	}

	// Parse collection name format: owner_<hash>/project_<hash>/<branch>
	parts := strings.SplitN(collectionName, "/", 2)
	if len(parts) < 1 {
		return "", "", fmt.Errorf("invalid collection name format: expected 'owner_<hash>/...', got '%s'", collectionName)
	}

	// Extract owner ID from owner_<hash> prefix
	ownerPart := parts[0]
	if !strings.HasPrefix(ownerPart, "owner_") {
		return "", "", fmt.Errorf("invalid owner prefix: expected 'owner_<hash>', got '%s'", ownerPart)
	}

	ownerID := strings.TrimPrefix(ownerPart, "owner_")
	if ownerID == "" {
		return "", "", fmt.Errorf("empty owner ID")
	}

	return ownerID, collectionName, nil
}
