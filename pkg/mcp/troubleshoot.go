package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"
)

// handleTroubleshoot handles POST /mcp/troubleshoot tool call.
//
// This is an async operation that:
//  1. Validates request parameters (error_message, optional context)
//  2. Creates NATS operation
//  3. Starts async worker to diagnose error
//  4. Returns operation_id immediately
//
// Request format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "params": {
//	    "error_message": "connection refused: dial tcp 127.0.0.1:6333",
//	    "context": "Qdrant startup during contextd initialization"
//	  }
//	}
//
// Response format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "result": {
//	    "operation_id": "uuid",
//	    "status": "pending"
//	  }
//	}
func (s *Server) handleTroubleshoot(c echo.Context) error {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Parse tool-specific params
	var params TroubleshootRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Validate params
	if params.ErrorMessage == "" {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("error_message is required"))
	}

	// Extract authenticated owner ID
	ownerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, AuthError, err)
	}

	// Create operation
	ctx := context.WithValue(c.Request().Context(), ownerIDKey, ownerID)
	ctx = context.WithValue(ctx, traceIDKey, c.Response().Header().Get("X-Request-ID"))
	opID := s.operations.Create(ctx, "troubleshoot", params)

	// Start async worker (AI operation can take 1-3 seconds)
	go s.doTroubleshoot(ctx, opID, params)

	// Return operation_id immediately
	return JSONRPCSuccess(c, req.ID, map[string]string{
		"operation_id": opID,
		"status":       "pending",
	})
}

// doTroubleshoot performs the actual error diagnosis operation.
func (s *Server) doTroubleshoot(ctx context.Context, opID string, params TroubleshootRequest) {
	if err := s.operations.Started(opID); err != nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("failed to start: %w", err))
		return
	}

	// Check service availability
	if s.troubleshootService == nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("troubleshoot service not available"))
		return
	}

	// Diagnose error via service
	diagnosis, err := s.troubleshootService.Diagnose(ctx, params.ErrorMessage, params.Context)
	if err != nil {
		_ = s.operations.Error(opID, InternalError, err)
		return
	}

	// Complete with diagnosis
	_ = s.operations.Complete(opID, map[string]interface{}{
		"diagnosis":        diagnosis.RootCause,
		"confidence":       diagnosis.Confidence,
		"hypotheses":       diagnosis.Hypotheses,
		"recommendations":  diagnosis.Recommendations,
		"related_patterns": diagnosis.RelatedPatterns,
	})
}

// handleListPatterns handles POST /mcp/list_patterns tool call.
//
// This is a synchronous operation that returns all known error patterns.
//
// Request format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "params": {}
//	}
//
// Response format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "result": {
//	    "patterns": [
//	      {
//	        "id": "pattern_uuid",
//	        "error_type": "ConnectionError",
//	        "description": "Qdrant connection refused",
//	        "solution": "Start Qdrant: docker-compose up -d qdrant",
//	        "frequency": 5,
//	        "confidence": 0.95
//	      },
//	      ...
//	    ]
//	  }
//	}
func (s *Server) handleListPatterns(c echo.Context) error {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Check service availability
	if s.troubleshootService == nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, fmt.Errorf("troubleshoot service not available"))
	}

	// Get patterns via service
	ctx := c.Request().Context()
	patterns, err := s.troubleshootService.GetPatterns(ctx)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, err)
	}

	return JSONRPCSuccess(c, req.ID, map[string]interface{}{
		"patterns": patterns,
	})
}
