package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// handleCollectionCreate handles POST /mcp/collection/create.
//
// This is a long-running operation that:
//  1. Validates request parameters (collection_name, vector_size)
//  2. Validates collection name format
//  3. Creates NATS operation
//  4. Starts async worker to create collection
//  5. Returns operation_id immediately
//
// Request format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "params": {
//	    "collection_name": "owner_abc/project_def/main",
//	    "vector_size": 384
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
func (s *Server) handleCollectionCreate(c echo.Context) error {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Parse tool-specific params
	var params struct {
		CollectionName string `json:"collection_name"`
		VectorSize     int    `json:"vector_size"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Validate params
	if params.CollectionName == "" {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("collection_name is required"))
	}
	if params.VectorSize <= 0 {
		if params.VectorSize == 0 {
			return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("vector_size is required"))
		}
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("vector_size must be positive"))
	}

	// Extract authenticated owner ID
	ownerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, AuthError, err)
	}

	// Create operation
	ctx := context.WithValue(c.Request().Context(), ownerIDKey, ownerID)
	ctx = context.WithValue(ctx, traceIDKey, c.Response().Header().Get("X-Request-ID"))
	opID := s.operations.Create(ctx, "collection_create", params)

	// Start async worker with background context (not request context)
	// Request context gets cancelled when HTTP request completes
	bgCtx := context.WithValue(context.Background(), ownerIDKey, ownerID)
	bgCtx = context.WithValue(bgCtx, traceIDKey, c.Response().Header().Get("X-Request-ID"))
	go s.doCollectionCreate(bgCtx, opID, params)

	// Return operation_id immediately
	return JSONRPCSuccess(c, req.ID, map[string]string{
		"operation_id": opID,
		"status":       "pending",
	})
}

// doCollectionCreate performs the actual collection creation operation.
func (s *Server) doCollectionCreate(ctx context.Context, opID string, params struct {
	CollectionName string `json:"collection_name"`
	VectorSize     int    `json:"vector_size"`
}) {
	if err := s.operations.Started(opID); err != nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("failed to start: %w", err))
		return
	}

	// Check service availability
	if s.vectorStore == nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("vector store service not available"))
		return
	}

	// Create collection via service
	if err := s.vectorStore.CreateCollection(ctx, params.CollectionName, params.VectorSize); err != nil {
		_ = s.operations.Error(opID, VectorStoreError, err)
		return
	}

	// Complete
	_ = s.operations.Complete(opID, map[string]interface{}{
		"collection_name": params.CollectionName,
		"vector_size":     params.VectorSize,
	})
}

// handleCollectionDelete handles POST /mcp/collection/delete.
//
// This is a long-running operation that:
//  1. Validates request parameters (collection_name)
//  2. Validates collection exists and belongs to authenticated owner
//  3. Creates NATS operation
//  4. Starts async worker to delete collection
//  5. Returns operation_id immediately
//
// Request format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "params": {
//	    "collection_name": "owner_abc/project_def/main"
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
func (s *Server) handleCollectionDelete(c echo.Context) error {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Parse tool-specific params
	var params struct {
		CollectionName string `json:"collection_name"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Validate params
	if params.CollectionName == "" {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("collection_name is required"))
	}

	// Extract authenticated owner ID
	ownerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, AuthError, err)
	}

	// Create operation
	ctx := context.WithValue(c.Request().Context(), ownerIDKey, ownerID)
	ctx = context.WithValue(ctx, traceIDKey, c.Response().Header().Get("X-Request-ID"))
	opID := s.operations.Create(ctx, "collection_delete", params)

	// Start async worker with background context (not request context)
	// Request context gets cancelled when HTTP request completes
	bgCtx := context.WithValue(context.Background(), ownerIDKey, ownerID)
	bgCtx = context.WithValue(bgCtx, traceIDKey, c.Response().Header().Get("X-Request-ID"))
	go s.doCollectionDelete(bgCtx, opID, params)

	// Return operation_id immediately
	return JSONRPCSuccess(c, req.ID, map[string]string{
		"operation_id": opID,
		"status":       "pending",
	})
}

// doCollectionDelete performs the actual collection deletion operation.
func (s *Server) doCollectionDelete(ctx context.Context, opID string, params struct {
	CollectionName string `json:"collection_name"`
}) {
	if err := s.operations.Started(opID); err != nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("failed to start: %w", err))
		return
	}

	// Check service availability
	if s.vectorStore == nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("vector store service not available"))
		return
	}

	// Delete collection via service
	if err := s.vectorStore.DeleteCollection(ctx, params.CollectionName); err != nil {
		_ = s.operations.Error(opID, VectorStoreError, err)
		return
	}

	// Complete
	_ = s.operations.Complete(opID, map[string]interface{}{
		"collection_name": params.CollectionName,
		"deleted":         true,
	})
}

// handleCollectionList handles POST /mcp/collection/list.
//
// This endpoint lists all collections owned by the authenticated user.
// Collections are filtered by owner prefix (owner_<hash>/).
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
//	    "collections": [
//	      {"name": "owner_abc/project_1/main", "vector_size": 384, "point_count": 100},
//	      {"name": "owner_abc/project_2/main", "vector_size": 384, "point_count": 50}
//	    ]
//	  }
//	}
func (s *Server) handleCollectionList(c echo.Context) error {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Extract authenticated owner ID
	ownerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, AuthError, err)
	}

	// Check service availability
	if s.vectorStore == nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, fmt.Errorf("vector store service not available"))
	}

	// List all collections
	ctx := c.Request().Context()
	allCollections, err := s.vectorStore.ListCollections(ctx)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, VectorStoreError, err)
	}

	// Filter by owner prefix
	ownerPrefix := fmt.Sprintf("owner_%s", ownerID[:6]) // Use first 6 chars of owner hash as prefix
	var filteredCollections []map[string]interface{}

	for _, collectionName := range allCollections {
		// Check if collection belongs to this owner
		if strings.HasPrefix(collectionName, ownerPrefix) {
			// Get collection info (optional - could be expensive for many collections)
			info, err := s.vectorStore.GetCollectionInfo(ctx, collectionName)
			if err != nil {
				// Log error but continue with other collections
				s.logger.Warn("failed to get collection info",
					zap.String("collection", collectionName),
					zap.Error(err))
				continue
			}

			filteredCollections = append(filteredCollections, map[string]interface{}{
				"name":        info.Name,
				"vector_size": info.VectorSize,
				"point_count": info.PointCount,
			})
		}
	}

	return JSONRPCSuccess(c, req.ID, map[string]interface{}{
		"collections": filteredCollections,
	})
}
