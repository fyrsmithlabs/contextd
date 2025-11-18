package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// ToolDefinition represents an MCP tool with its metadata and input schema.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ResourceDefinition represents an MCP resource (collection).
type ResourceDefinition struct {
	URI      string                 `json:"uri"`
	Name     string                 `json:"name"`
	MimeType string                 `json:"mime_type"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// handleToolsList handles GET /mcp/tools/list.
//
// This endpoint returns all available MCP tools for client discovery.
// It provides tool names, descriptions, and input schemas following
// the MCP specification.
//
// Response format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "result": {
//	    "tools": [
//	      {
//	        "name": "checkpoint_save",
//	        "description": "Save a checkpoint",
//	        "input_schema": {
//	          "type": "object",
//	          "properties": {...},
//	          "required": [...]
//	        }
//	      },
//	      ...
//	    ]
//	  }
//	}
func (s *Server) handleToolsList(c echo.Context) error {
	// Define all available tools
	tools := []ToolDefinition{
		{
			Name:        "checkpoint_save",
			Description: "Save a checkpoint with context and metadata",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Checkpoint content",
					},
					"project_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to project directory",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Additional metadata",
					},
				},
				"required": []string{"content", "project_path"},
			},
		},
		{
			Name:        "checkpoint_search",
			Description: "Search checkpoints semantically",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"project_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to project directory",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     10,
					},
				},
				"required": []string{"query", "project_path"},
			},
		},
		{
			Name:        "checkpoint_list",
			Description: "List recent checkpoints",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to project directory",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     20,
					},
				},
				"required": []string{"project_path"},
			},
		},
		{
			Name:        "remediation_save",
			Description: "Save an error remediation solution",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"error_msg": map[string]interface{}{
						"type":        "string",
						"description": "Error message",
					},
					"solution": map[string]interface{}{
						"type":        "string",
						"description": "Solution description",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Additional context",
					},
				},
				"required": []string{"error_msg", "solution"},
			},
		},
		{
			Name:        "remediation_search",
			Description: "Search for error remediations",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"error_msg": map[string]interface{}{
						"type":        "string",
						"description": "Error message to match",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     5,
					},
				},
				"required": []string{"error_msg"},
			},
		},
		{
			Name:        "skill_save",
			Description: "Save a reusable skill or template",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Skill name",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Skill content",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"description": "Skill tags",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"name", "content"},
			},
		},
		{
			Name:        "skill_search",
			Description: "Search for skills by query or tags",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     10,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "index_repository",
			Description: "Index a repository for semantic search",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to project directory",
					},
					"force": map[string]interface{}{
						"type":        "boolean",
						"description": "Force re-indexing",
						"default":     false,
					},
				},
				"required": []string{"project_path"},
			},
		},
		{
			Name:        "status",
			Description: "Get operation status by ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation_id": map[string]interface{}{
						"type":        "string",
						"description": "Operation UUID",
					},
				},
				"required": []string{"operation_id"},
			},
		},
		{
			Name:        "collection_create",
			Description: "Create a new vector collection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection_name": map[string]interface{}{
						"type":        "string",
						"description": "Collection name in format: owner_<hash>/project_<hash>/<branch>",
					},
					"vector_size": map[string]interface{}{
						"type":        "integer",
						"description": "Vector dimension (e.g., 384 for BGE-small, 1536 for OpenAI)",
					},
				},
				"required": []string{"collection_name", "vector_size"},
			},
		},
		{
			Name:        "collection_delete",
			Description: "Delete an existing vector collection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection_name": map[string]interface{}{
						"type":        "string",
						"description": "Collection name to delete",
					},
				},
				"required": []string{"collection_name"},
			},
		},
		{
			Name:        "collection_list",
			Description: "List all collections owned by authenticated user",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	// Return JSON-RPC success response
	return JSONRPCSuccess(c, "", map[string]interface{}{
		"tools": tools,
	})
}

// handleResourcesList handles GET /mcp/resources/list.
//
// This endpoint returns all available resources (collections) for the
// authenticated owner.
//
// Response format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "result": {
//	    "resources": [
//	      {
//	        "uri": "collection://owner_abc/project_def/main",
//	        "name": "main",
//	        "mime_type": "application/x-qdrant-collection",
//	        "metadata": {"vector_size": 384, "point_count": 1234}
//	      },
//	      ...
//	    ]
//	  }
//	}
func (s *Server) handleResourcesList(c echo.Context) error {
	// Extract authenticated owner ID
	ownerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, "", AuthError, err)
	}

	// Initialize resources slice
	resources := []ResourceDefinition{}

	// List collections if vectorstore service is available
	if s.vectorStore != nil {
		collections, err := s.vectorStore.ListCollections(c.Request().Context())
		if err != nil {
			return JSONRPCErrorWithContext(c, "", InternalError, fmt.Errorf("failed to list collections: %w", err))
		}

		// Filter collections by owner prefix (format: owner_<hash>/project_<hash>/branch)
		ownerPrefix := fmt.Sprintf("owner_%s/", ownerID)
		for _, collection := range collections {
			if strings.HasPrefix(collection, ownerPrefix) {
				// Get collection info for metadata
				info, err := s.vectorStore.GetCollectionInfo(c.Request().Context(), collection)
				if err != nil {
					// Log error but continue with other collections
					s.logger.Warn("Failed to get collection info",
						zap.String("collection", collection),
						zap.Error(err))
					continue
				}

				resources = append(resources, ResourceDefinition{
					URI:      fmt.Sprintf("collection://%s", collection),
					Name:     collection,
					MimeType: "application/x-qdrant-collection",
					Metadata: map[string]interface{}{
						"vector_size": info.VectorSize,
						"point_count": info.PointCount,
					},
				})
			}
		}
	}

	return JSONRPCSuccess(c, "", map[string]interface{}{
		"resources": resources,
	})
}

// handleResourceRead handles POST /mcp/resources/read.
//
// This endpoint reads a specific resource (collection metadata) by URI.
//
// Request format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "method": "resources/read",
//	  "params": {
//	    "uri": "collection://owner_abc/project_def/main"
//	  }
//	}
//
// Response format:
//
//	{
//	  "jsonrpc": "2.0",
//	  "id": "request-id",
//	  "result": {
//	    "uri": "collection://owner_abc/project_def/main",
//	    "mime_type": "application/x-qdrant-collection",
//	    "content": "{\"name\": \"main\", \"vector_size\": 384, \"point_count\": 1234}"
//	  }
//	}
func (s *Server) handleResourceRead(c echo.Context) error {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Parse params
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Validate URI
	if params.URI == "" {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("uri is required"))
	}

	// Extract authenticated owner ID
	authenticatedOwnerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, AuthError, err)
	}

	// Parse URI to extract owner ID and collection name
	uriOwnerID, collectionName, err := ParseCollectionURI(params.URI)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("invalid URI: %w", err))
	}

	// Validate that the requested collection belongs to the authenticated owner
	if uriOwnerID != authenticatedOwnerID {
		return JSONRPCErrorWithContext(c, req.ID, AuthError,
			fmt.Errorf("access denied: collection does not belong to authenticated owner"))
	}

	// Get collection info from vectorstore (if available)
	if s.vectorStore == nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError,
			fmt.Errorf("vectorstore service not available"))
	}

	info, err := s.vectorStore.GetCollectionInfo(c.Request().Context(), collectionName)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError,
			fmt.Errorf("failed to get collection info: %w", err))
	}

	// Build content from collection info
	content := map[string]interface{}{
		"name":        info.Name,
		"vector_size": info.VectorSize,
		"point_count": info.PointCount,
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, err)
	}

	return JSONRPCSuccess(c, req.ID, map[string]interface{}{
		"uri":       params.URI,
		"mime_type": "application/x-qdrant-collection",
		"content":   string(contentJSON),
	})
}
