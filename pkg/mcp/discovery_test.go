package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleToolsList tests the GET /mcp/tools/list endpoint.
//
// This endpoint returns a list of all available MCP tools with their
// input schemas for client discovery.
func TestHandleToolsList(t *testing.T) {
	tests := []struct {
		name           string
		wantToolCount  int
		wantToolNames  []string
		checkFirstTool bool
	}{
		{
			name:          "returns all available tools",
			wantToolCount: 12, // checkpoint(3) + remediation(2) + skill(2) + index(1) + status(1) + collection(3)
			wantToolNames: []string{
				"checkpoint_save",
				"checkpoint_search",
				"checkpoint_list",
				"remediation_save",
				"remediation_search",
				"skill_save",
				"skill_search",
				"index_repository",
				"status",
				"collection_create",
				"collection_delete",
				"collection_list",
			},
			checkFirstTool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/mcp/tools/list", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create server (vectorstore not needed for tools list)
			server := &Server{
				echo:        e,
				logger:      nil, // Use no-op logger for tests
				vectorStore: nil,
			}

			// Execute
			err := server.handleToolsList(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			// Parse response
			var resp JSONRPCResponse
			err = json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)

			// Check JSON-RPC fields
			assert.Equal(t, "2.0", resp.JSONRPC)

			// Check result structure
			result, ok := resp.Result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			tools, ok := result["tools"].([]interface{})
			require.True(t, ok, "tools should be an array")
			assert.Len(t, tools, tt.wantToolCount)

			// Check tool names
			actualNames := make([]string, len(tools))
			for i, tool := range tools {
				toolMap, ok := tool.(map[string]interface{})
				require.True(t, ok)
				actualNames[i] = toolMap["name"].(string)
			}
			assert.ElementsMatch(t, tt.wantToolNames, actualNames)

			// Check first tool structure if requested
			if tt.checkFirstTool {
				firstTool := tools[0].(map[string]interface{})
				assert.Contains(t, firstTool, "name")
				assert.Contains(t, firstTool, "description")
				assert.Contains(t, firstTool, "input_schema")

				// Verify input_schema is an object
				schema, ok := firstTool["input_schema"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "object", schema["type"])
				assert.Contains(t, schema, "properties")
			}
		})
	}
}

// TestHandleResourcesList tests the GET /mcp/resources/list endpoint.
//
// This endpoint returns a list of available MCP resources (collections).
func TestHandleResourcesList(t *testing.T) {
	tests := []struct {
		name             string
		ownerID          string
		wantResourceType string
		wantCount        int
		wantErr          bool
		wantErrCode      int
	}{
		{
			name:             "returns resources for authenticated owner",
			ownerID:          "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678", // Valid 64-char hex
			wantResourceType: "collection",
			wantCount:        0, // Empty for new tests
			wantErr:          false,
		},
		{
			name:        "returns auth error for missing owner",
			ownerID:     "",
			wantErr:     true,
			wantErrCode: AuthError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/mcp/resources/list", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set owner ID in Echo context if provided (simulating middleware)
			if tt.ownerID != "" {
				c.Set(string(authenticatedOwnerIDKey), tt.ownerID)
			}

			// Create server (vectorstore required for resources list)
			server := &Server{
				echo:        e,
				logger:      nil,
				vectorStore: nil, // Will be replaced with mock in future test iterations
			}

			// Execute
			err := server.handleResourcesList(c)

			// Assert
			if tt.wantErr {
				// JSON-RPC errors are returned as successful HTTP responses
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)

				// Parse response to check for JSON-RPC error
				var errResp JSONRPCError
				parseErr := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, parseErr)
				assert.Equal(t, "2.0", errResp.JSONRPC)
				assert.NotNil(t, errResp.Error)
				assert.Equal(t, tt.wantErrCode, errResp.Error.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)

				// Parse response
				var resp JSONRPCResponse
				err = json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err, "Response body: %s", rec.Body.String())
				assert.Equal(t, "2.0", resp.JSONRPC)

				// Check result structure
				result, ok := resp.Result.(map[string]interface{})
				require.True(t, ok, "Result should be a map, got: %T, Response: %s", resp.Result, rec.Body.String())

				resources, ok := result["resources"].([]interface{})
				require.True(t, ok, "resources should be an array, got: %T", result["resources"])
				assert.GreaterOrEqual(t, len(resources), tt.wantCount)
			}
		})
	}
}

// TestHandleResourceRead tests the POST /mcp/resources/read endpoint.
//
// This endpoint reads a specific resource (collection metadata).
func TestHandleResourceRead(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  string
		ownerID      string
		wantErr      bool
		wantErrCode  int
		checkContent bool
	}{
		{
			name: "returns error when vectorstore is nil",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-1",
				"method": "resources/read",
				"params": {
					"uri": "collection://owner_a1b2c3d4e5f67890123456789012345678901234567890123456789012345678/project_def/main"
				}
			}`,
			ownerID:      "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678", // Valid 64-char hex
			wantErr:      true,
			wantErrCode:  InternalError,
			checkContent: false,
		},
		{
			name: "returns error for missing URI",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-2",
				"method": "resources/read",
				"params": {}
			}`,
			ownerID:     "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678", // Valid 64-char hex
			wantErr:     true,
			wantErrCode: InvalidParams,
		},
		{
			name: "returns auth error for missing owner",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-3",
				"method": "resources/read",
				"params": {
					"uri": "collection://owner_abc/project_def/main"
				}
			}`,
			ownerID:     "",
			wantErr:     true,
			wantErrCode: AuthError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/mcp/resources/read", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set owner ID in Echo context if provided (simulating middleware)
			if tt.ownerID != "" {
				c.Set(string(authenticatedOwnerIDKey), tt.ownerID)
			}

			// Create server (vectorstore required for resource read)
			server := &Server{
				echo:        e,
				logger:      nil,
				vectorStore: nil, // Will be replaced with mock in future test iterations
			}

			// Execute
			err := server.handleResourceRead(c)

			// Assert
			if tt.wantErr {
				// JSON-RPC errors are returned as successful HTTP responses
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)

				// Parse response to check for JSON-RPC error
				var errResp JSONRPCError
				parseErr := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, parseErr)
				assert.Equal(t, "2.0", errResp.JSONRPC)
				assert.NotNil(t, errResp.Error)
				assert.Equal(t, tt.wantErrCode, errResp.Error.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)

				// Parse response
				var resp JSONRPCResponse
				err = json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, "2.0", resp.JSONRPC)

				if tt.checkContent {
					result, ok := resp.Result.(map[string]interface{})
					require.True(t, ok)
					assert.Contains(t, result, "uri")
					assert.Contains(t, result, "mime_type")
					assert.Contains(t, result, "content")
				}
			}
		})
	}
}
