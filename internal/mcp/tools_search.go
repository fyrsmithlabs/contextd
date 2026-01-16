package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ===== TOOL REFERENCE CONTENT =====

// ToolReferenceContent represents a tool_reference content block as specified by the
// Anthropic tool search protocol. It embeds *mcp.TextContent to satisfy the mcp.Content
// interface (which has an unexported method), while providing custom JSON marshaling
// to produce the correct wire format: { "type": "tool_reference", "tool_name": "..." }
//
// See: https://platform.claude.com/docs/en/agents-and-tools/tool-use/tool-search-tool#custom-tool-search-implementation
type ToolReferenceContent struct {
	// Embed TextContent to satisfy the mcp.Content interface
	// The embedded type provides the required fromWire method
	*mcp.TextContent
	// ToolName is the name of the discovered tool being referenced
	ToolName string
}

// NewToolReferenceContent creates a new tool_reference content block for the given tool name.
func NewToolReferenceContent(toolName string) *ToolReferenceContent {
	return &ToolReferenceContent{
		TextContent: &mcp.TextContent{Text: ""}, // Placeholder to satisfy interface
		ToolName:    toolName,
	}
}

// MarshalJSON produces the tool_reference wire format expected by the Anthropic API.
// This overrides the embedded TextContent's MarshalJSON to output:
// { "type": "tool_reference", "tool_name": "tool_name_here" }
func (c *ToolReferenceContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type     string `json:"type"`
		ToolName string `json:"tool_name"`
	}{
		Type:     "tool_reference",
		ToolName: c.ToolName,
	})
}

// ===== TOOL SEARCH TOOLS =====

type toolSearchInput struct {
	Query    string `json:"query" jsonschema:"required,Regex pattern or search query to find tools. Searches tool names, descriptions, and keywords. Uses Python re.search() syntax for regex patterns (e.g., 'memory', 'get_.*_data', '(?i)search')."`
	Category string `json:"category,omitempty" jsonschema:"Filter results to a specific category (memory, checkpoint, remediation, repository, troubleshoot, folding, conversation, reflection, search)"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Maximum results to return (default: 5)"`
}

type toolSearchOutput struct {
	Query      string                   `json:"query" jsonschema:"Search query used"`
	Results    []map[string]interface{} `json:"results" jsonschema:"Matching tools with name, description, category, and match score"`
	Count      int                      `json:"count" jsonschema:"Number of tools found"`
	TotalTools int                      `json:"total_tools" jsonschema:"Total number of tools in registry"`
}

// toolListInput is empty because it lists all tools.
type toolListInput struct {
	Category     string `json:"category,omitempty" jsonschema:"Filter to a specific category"`
	DeferredOnly bool   `json:"deferred_only,omitempty" jsonschema:"Only list deferred tools (default: false)"`
}

type toolListOutput struct {
	Tools []map[string]interface{} `json:"tools" jsonschema:"List of all registered tools with metadata"`
	Count int                      `json:"count" jsonschema:"Number of tools returned"`
}

func (s *Server) registerSearchTools() {
	// Check if registry is available
	if s.toolRegistry == nil {
		s.logger.Warn("tool registry not configured, skipping search tools")
		return
	}

	// tool_search - Search for tools by query
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "tool_search",
		Description: "Search for available tools by name, description, or keyword. Returns tool_reference blocks for discovered tools. Use this to find relevant tools without loading all tool definitions into context. Uses regex pattern matching (Python re.search() syntax).",
		Meta:        s.toolMeta("tool_search"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args toolSearchInput) (*mcp.CallToolResult, toolSearchOutput, error) {
		if args.Query == "" {
			return nil, toolSearchOutput{}, fmt.Errorf("query is required")
		}

		limit := args.Limit
		if limit <= 0 {
			limit = 5
		}

		// Search the registry
		var searchResults []*SearchResult
		if args.Category != "" {
			// Search within category
			category := ToolCategory(args.Category)
			searchResults = s.toolRegistry.SearchByCategory(args.Query, category)
		} else {
			// Search all categories
			searchResults = s.toolRegistry.Search(args.Query)
		}

		// Apply limit
		if len(searchResults) > limit {
			searchResults = searchResults[:limit]
		}

		// Convert results to output format
		results := make([]map[string]interface{}, 0, len(searchResults))
		var toolNames []string

		for _, sr := range searchResults {
			tool := sr.Tool
			result := map[string]interface{}{
				"name":         tool.Name,
				"description":  tool.Description,
				"category":     string(tool.Category),
				"defer_loading": tool.DeferLoading,
				"score":        sr.Score,
				"match_reason": sr.MatchReason,
			}
			if len(tool.Keywords) > 0 {
				result["keywords"] = tool.Keywords
			}
			results = append(results, result)
			toolNames = append(toolNames, tool.Name)
		}

		output := toolSearchOutput{
			Query:      args.Query,
			Results:    results,
			Count:      len(results),
			TotalTools: s.toolRegistry.Count(),
		}

		// Build result message with tool references
		var resultText string
		if len(toolNames) == 0 {
			resultText = fmt.Sprintf("No tools found matching: %s", args.Query)
		} else {
			resultText = fmt.Sprintf("Found %d tool(s) for query '%s': %s",
				len(toolNames), args.Query, strings.Join(toolNames, ", "))
		}

		// Build content with tool_reference blocks
		// The MCP protocol (Claude) expands these to full tool definitions automatically
		content := make([]mcp.Content, 0, len(toolNames)+1)
		content = append(content, &mcp.TextContent{Text: resultText})

		// Add tool_reference blocks for each discovered tool
		// These blocks follow the Anthropic tool search protocol format:
		// { "type": "tool_reference", "tool_name": "discovered_tool_name" }
		// When Claude receives these, it automatically expands them to full tool
		// definitions from the tools provided with defer_loading: true
		for _, toolName := range toolNames {
			content = append(content, NewToolReferenceContent(toolName))
		}

		return &mcp.CallToolResult{
			Content: content,
		}, output, nil
	})

	// tool_list - List all available tools
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "tool_list",
		Description: "List all available tools in the registry with their metadata. Use this to see what tools are available without searching.",
		Meta:        s.toolMeta("tool_list"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args toolListInput) (*mcp.CallToolResult, toolListOutput, error) {
		var tools []*ToolMetadata

		if args.Category != "" {
			// List by category
			category := ToolCategory(args.Category)
			tools = s.toolRegistry.ListByCategory(category)
		} else if args.DeferredOnly {
			// List only deferred tools
			tools = s.toolRegistry.ListDeferred()
		} else {
			// List all tools
			tools = s.toolRegistry.List()
		}

		// Convert to output format
		results := make([]map[string]interface{}, 0, len(tools))
		for _, tool := range tools {
			result := map[string]interface{}{
				"name":         tool.Name,
				"description":  tool.Description,
				"category":     string(tool.Category),
				"defer_loading": tool.DeferLoading,
			}
			if len(tool.Keywords) > 0 {
				result["keywords"] = tool.Keywords
			}
			results = append(results, result)
		}

		output := toolListOutput{
			Tools: results,
			Count: len(results),
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d tools", output.Count)},
			},
		}, output, nil
	})
}
