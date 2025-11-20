package stdio

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server implements MCP stdio transport with HTTP delegation to contextd daemon.
//
// This server provides the MCP protocol over stdin/stdout for Claude Code integration.
// All tool calls are delegated to the HTTP daemon running on localhost:9090.
//
// Architecture:
//
//	Claude Code → stdio (this server) → HTTP client → contextd daemon → services
//
// Example usage:
//
//	server, err := NewServer("http://localhost:9090")
//	if err != nil {
//	    return err
//	}
//	if err := server.Run(ctx); err != nil {
//	    return err
//	}
type Server struct {
	mcpServer *mcpsdk.Server
	client    *DaemonClient
}

// NewServer creates a new stdio MCP server.
//
// The daemonURL should point to the contextd HTTP daemon (e.g., "http://localhost:9090").
func NewServer(daemonURL string) (*Server, error) {
	if daemonURL == "" {
		return nil, fmt.Errorf("daemonURL cannot be empty")
	}

	// Create HTTP client to daemon
	client := NewDaemonClient(daemonURL)

	// Create MCP SDK server
	mcpServer := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "contextd",
		Version: "1.0.0",
	}, nil)

	s := &Server{
		mcpServer: mcpServer,
		client:    client,
	}

	// Register MCP tools
	s.registerTools()

	return s, nil
}

// Run starts the MCP server using stdio transport.
//
// This method blocks until the context is cancelled or an error occurs.
func (s *Server) Run(ctx context.Context) error {
	// Run with stdio transport (stdin/stdout)
	if err := s.mcpServer.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// registerTools registers all MCP tools with the server.
//
// Each tool delegates to the corresponding HTTP endpoint on the daemon.
func (s *Server) registerTools() {
	// Checkpoint tools
	mcpsdk.AddTool(s.mcpServer, &mcpsdk.Tool{
		Name:        "checkpoint_save",
		Description: "Save a session checkpoint for resuming work later. Stores summary, content, project path, and tags with automatic vector embeddings for semantic search.",
	}, s.handleCheckpointSave)

	mcpsdk.AddTool(s.mcpServer, &mcpsdk.Tool{
		Name:        "checkpoint_search",
		Description: "Search checkpoints using semantic similarity. Finds relevant checkpoints based on query meaning, with optional filtering by project path.",
	}, s.handleCheckpointSearch)

	// Status tool
	mcpsdk.AddTool(s.mcpServer, &mcpsdk.Tool{
		Name:        "status",
		Description: "Get contextd service status and health information. Shows service health, version, and uptime.",
	}, s.handleStatus)
}

// CheckpointSaveParams defines parameters for checkpoint_save tool.
type CheckpointSaveParams struct {
	Summary     string   `json:"summary" jsonschema:"Brief summary of checkpoint"`
	ProjectPath string   `json:"project_path" jsonschema:"Absolute path to project directory"`
	Content     string   `json:"content,omitempty" jsonschema:"Full checkpoint content (optional)"`
	Tags        []string `json:"tags,omitempty" jsonschema:"Tags for categorization (optional)"`
}

// CheckpointSearchParams defines parameters for checkpoint_search tool.
type CheckpointSearchParams struct {
	Query       string `json:"query" jsonschema:"Search query"`
	ProjectPath string `json:"project_path" jsonschema:"Absolute path to project directory"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum number of results (default 10)"`
}

// StatusParams defines parameters for status tool (empty - no params needed).
type StatusParams struct{}

// handleCheckpointSave handles the checkpoint_save tool call.
//
// Delegates to POST /mcp/checkpoint/save on the daemon.
func (s *Server) handleCheckpointSave(ctx context.Context, req *mcpsdk.CallToolRequest, params *CheckpointSaveParams) (*mcpsdk.CallToolResult, any, error) {
	// Build request for daemon
	request := map[string]interface{}{
		"summary":      params.Summary,
		"project_path": params.ProjectPath,
	}
	if params.Content != "" {
		request["content"] = params.Content
	}
	if len(params.Tags) > 0 {
		request["tags"] = params.Tags
	}

	// Delegate to HTTP daemon
	var response map[string]interface{}
	if err := s.client.Post(ctx, "/mcp/checkpoint/save", request, &response); err != nil {
		return nil, nil, fmt.Errorf("checkpoint save failed: %w", err)
	}

	// Extract checkpoint ID from response
	checkpointID, _ := response["checkpoint_id"].(string)

	// Return MCP result
	result := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: fmt.Sprintf("✓ Checkpoint saved successfully\n\nID: %s\nSummary: %s\nProject: %s", checkpointID, params.Summary, params.ProjectPath),
			},
		},
	}
	return result, nil, nil
}

// handleCheckpointSearch handles the checkpoint_search tool call.
//
// Delegates to POST /mcp/checkpoint/search on the daemon.
func (s *Server) handleCheckpointSearch(ctx context.Context, req *mcpsdk.CallToolRequest, params *CheckpointSearchParams) (*mcpsdk.CallToolResult, any, error) {
	// Set default limit if not provided
	limit := params.Limit
	if limit == 0 {
		limit = 10
	}

	// Build request for daemon
	request := map[string]interface{}{
		"query":        params.Query,
		"project_path": params.ProjectPath,
		"limit":        limit,
	}

	// Delegate to HTTP daemon
	var response map[string]interface{}
	if err := s.client.Post(ctx, "/mcp/checkpoint/search", request, &response); err != nil {
		return nil, nil, fmt.Errorf("checkpoint search failed: %w", err)
	}

	// Extract results from response
	resultsRaw, _ := response["results"].([]interface{})
	resultCount := len(resultsRaw)

	// Format results
	var resultText string
	if resultCount == 0 {
		resultText = fmt.Sprintf("No checkpoints found for query: %s", params.Query)
	} else {
		resultText = fmt.Sprintf("Found %d checkpoint(s) for query: %s\n\n", resultCount, params.Query)
		for i, r := range resultsRaw {
			if result, ok := r.(map[string]interface{}); ok {
				summary, _ := result["summary"].(string)
				score, _ := result["score"].(float64)
				resultText += fmt.Sprintf("%d. %s (score: %.2f)\n", i+1, summary, score)
			}
		}
	}

	// Return MCP result
	result := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: resultText,
			},
		},
	}
	return result, nil, nil
}

// handleStatus handles the status tool call.
//
// Delegates to GET /health on the daemon.
func (s *Server) handleStatus(ctx context.Context, req *mcpsdk.CallToolRequest, params *StatusParams) (*mcpsdk.CallToolResult, any, error) {
	// Delegate to HTTP daemon
	var response map[string]interface{}
	if err := s.client.Get(ctx, "/health", &response); err != nil {
		return nil, nil, fmt.Errorf("status check failed: %w", err)
	}

	// Extract status from response
	status, _ := response["status"].(string)
	version, _ := response["version"].(string)

	// Return MCP result
	result := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: fmt.Sprintf("✓ contextd daemon is healthy\n\nStatus: %s\nVersion: %s", status, version),
			},
		},
	}
	return result, nil, nil
}
