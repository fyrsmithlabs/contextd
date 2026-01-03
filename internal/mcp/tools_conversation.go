package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fyrsmithlabs/contextd/internal/conversation"
	"github.com/fyrsmithlabs/contextd/internal/tenant"
)

// ===== CONVERSATION TOOLS =====

type conversationIndexInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"required,Path to project to index conversations for"`
	TenantID    string   `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	SessionIDs  []string `json:"session_ids,omitempty" jsonschema:"Specific session IDs to index (empty = all)"`
	EnableLLM   bool     `json:"enable_llm,omitempty" jsonschema:"Enable LLM-based decision extraction (default: false)"`
	Force       bool     `json:"force,omitempty" jsonschema:"Force reindexing of existing sessions (default: false)"`
}

type conversationIndexOutput struct {
	SessionsIndexed    int      `json:"sessions_indexed" jsonschema:"Number of sessions indexed"`
	MessagesIndexed    int      `json:"messages_indexed" jsonschema:"Number of messages indexed"`
	DecisionsExtracted int      `json:"decisions_extracted" jsonschema:"Number of decisions extracted"`
	FilesReferenced    []string `json:"files_referenced" jsonschema:"Files referenced in conversations"`
	ErrorCount         int      `json:"error_count" jsonschema:"Number of errors during indexing"`
}

type conversationSearchInput struct {
	Query       string   `json:"query" jsonschema:"required,Semantic search query"`
	ProjectPath string   `json:"project_path" jsonschema:"required,Project path to search within"`
	TenantID    string   `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	Types       []string `json:"types,omitempty" jsonschema:"Filter by document types (message decision summary)"`
	Tags        []string `json:"tags,omitempty" jsonschema:"Filter by tags"`
	FilePath    string   `json:"file_path,omitempty" jsonschema:"Filter by file path discussed"`
	Domain      string   `json:"domain,omitempty" jsonschema:"Filter by domain (e.g. kubernetes frontend)"`
	Limit       int      `json:"limit,omitempty" jsonschema:"Maximum results to return (default: 10)"`
}

type conversationSearchOutput struct {
	Query   string                   `json:"query" jsonschema:"Search query used"`
	Results []map[string]interface{} `json:"results" jsonschema:"Search results with score and content"`
	Total   int                      `json:"total" jsonschema:"Total number of results"`
	TookMs  int64                    `json:"took_ms" jsonschema:"Search duration in milliseconds"`
}

func (s *Server) registerConversationTools() {
	if s.conversationSvc == nil {
		s.logger.Warn("conversation service not configured, skipping conversation tools")
		return
	}

	// conversation_index
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "conversation_index",
		Description: "Index Claude Code conversation files for a project. Parses JSONL files, extracts messages and decisions, and stores them for semantic search.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args conversationIndexInput) (*mcp.CallToolResult, conversationIndexOutput, error) {
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}
		if tenantID == "" {
			return nil, conversationIndexOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		opts := conversation.IndexOptions{
			ProjectPath: args.ProjectPath,
			TenantID:    tenantID,
			SessionIDs:  args.SessionIDs,
			EnableLLM:   args.EnableLLM,
			Force:       args.Force,
		}

		result, err := s.conversationSvc.Index(ctx, opts)
		if err != nil {
			return nil, conversationIndexOutput{}, fmt.Errorf("indexing failed: %w", err)
		}

		output := conversationIndexOutput{
			SessionsIndexed:    result.SessionsIndexed,
			MessagesIndexed:    result.MessagesIndexed,
			DecisionsExtracted: result.DecisionsExtracted,
			FilesReferenced:    result.FilesReferenced,
			ErrorCount:         len(result.Errors),
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf(
					"Indexed %d sessions, %d messages, %d decisions. %d files referenced.",
					output.SessionsIndexed, output.MessagesIndexed, output.DecisionsExtracted, len(output.FilesReferenced),
				)},
			},
		}, output, nil
	})

	// conversation_search
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "conversation_search",
		Description: "Search indexed Claude Code conversations for relevant past context, decisions, and patterns.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args conversationSearchInput) (*mcp.CallToolResult, conversationSearchOutput, error) {
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}
		if tenantID == "" {
			return nil, conversationSearchOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		// Convert string types to DocumentType
		var docTypes []conversation.DocumentType
		for _, t := range args.Types {
			docTypes = append(docTypes, conversation.DocumentType(t))
		}

		opts := conversation.SearchOptions{
			Query:       args.Query,
			ProjectPath: args.ProjectPath,
			TenantID:    tenantID,
			Types:       docTypes,
			Tags:        args.Tags,
			FilePath:    args.FilePath,
			Domain:      args.Domain,
			Limit:       args.Limit,
		}

		result, err := s.conversationSvc.Search(ctx, opts)
		if err != nil {
			return nil, conversationSearchOutput{}, fmt.Errorf("search failed: %w", err)
		}

		// Convert results to maps
		results := make([]map[string]interface{}, 0, len(result.Results))
		for _, hit := range result.Results {
			// Scrub content before returning
			scrubbedContent := hit.Document.Content
			if s.scrubber != nil {
				scrubbedContent = s.scrubber.Scrub(hit.Document.Content).Scrubbed
			}

			r := map[string]interface{}{
				"id":         hit.Document.ID,
				"session_id": hit.Document.SessionID,
				"type":       string(hit.Document.Type),
				"content":    scrubbedContent,
				"score":      hit.Score,
				"timestamp":  hit.Document.Timestamp.Unix(),
			}
			if len(hit.Document.Tags) > 0 {
				r["tags"] = hit.Document.Tags
			}
			if hit.Document.Domain != "" {
				r["domain"] = hit.Document.Domain
			}
			results = append(results, r)
		}

		output := conversationSearchOutput{
			Query:   result.Query,
			Results: results,
			Total:   result.Total,
			TookMs:  result.Took.Milliseconds(),
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf(
					"Found %d results for query: %s",
					output.Total, output.Query,
				)},
			},
		}, output, nil
	})
}
