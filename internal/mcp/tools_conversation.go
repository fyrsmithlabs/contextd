package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fyrsmithlabs/contextd/internal/conversation"
	"github.com/fyrsmithlabs/contextd/internal/sanitize"
	"github.com/fyrsmithlabs/contextd/internal/tenant"
)

// ===== CONVERSATION TOOLS =====

type conversationIndexInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"required,Path to project to index conversations for"`
	TenantID    string   `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	SessionIDs  []string `json:"session_ids,omitempty" jsonschema:"Specific session IDs to index (empty = all)"`
	EnableLLM   bool     `json:"enable_llm,omitempty" jsonschema:"Enable LLM-based decision extraction (default: false). NOTE: LLM summarization is not yet implemented - this flag is reserved for future use. Currently uses heuristic extraction only."`
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
	Types       []string `json:"types,omitempty" jsonschema:"Filter by document types: 'message', 'decision', or 'summary'"`
	Tags        []string `json:"tags,omitempty" jsonschema:"Filter by tags"`
	FilePath    string   `json:"file_path,omitempty" jsonschema:"Filter by file path discussed"`
	Domain      string   `json:"domain,omitempty" jsonschema:"Filter by domain (e.g., 'kubernetes', 'frontend', 'database')"`
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
		Description: "Index Claude Code conversation files for a project. Parses JSONL files, extracts messages and decisions, and stores them for semantic search. Note: LLM-based decision extraction (enable_llm) is not yet implemented - currently uses heuristic pattern matching only.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args conversationIndexInput) (*mcp.CallToolResult, conversationIndexOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "conversation_index", &toolErr)()

		// Validate project_path (CWE-22 path traversal protection)
		if args.ProjectPath == "" {
			toolErr = fmt.Errorf("project_path is required")
			return nil, conversationIndexOutput{}, toolErr
		}
		validPath, err := sanitize.ValidateProjectPath(args.ProjectPath)
		if err != nil {
			toolErr = fmt.Errorf("invalid project_path: %w", err)
			return nil, conversationIndexOutput{}, toolErr
		}

		tenantID := args.TenantID
		if tenantID == "" {
			tenantID = tenant.GetTenantIDForPath(validPath)
		}
		// Validate tenant_id format (CWE-287 authentication bypass protection)
		if tenantID == "" {
			toolErr = fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
			return nil, conversationIndexOutput{}, toolErr
		}
		if err := sanitize.ValidateTenantID(tenantID); err != nil {
			toolErr = fmt.Errorf("invalid tenant_id: %w", err)
			return nil, conversationIndexOutput{}, toolErr
		}

		opts := conversation.IndexOptions{
			ProjectPath: validPath,
			TenantID:    tenantID,
			SessionIDs:  args.SessionIDs,
			EnableLLM:   args.EnableLLM,
			Force:       args.Force,
		}

		// Reject LLM requests explicitly until implemented
		if args.EnableLLM {
			toolErr = fmt.Errorf("enable_llm=true is not yet supported: LLM-based decision extraction is not implemented; set enable_llm=false or omit the parameter to use heuristic extraction")
			return nil, conversationIndexOutput{}, toolErr
		}

		result, err := s.conversationSvc.Index(ctx, opts)
		if err != nil {
			toolErr = fmt.Errorf("indexing failed: %w", err)
			return nil, conversationIndexOutput{}, toolErr
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
		var toolErr error
		defer s.startMetrics(ctx, "conversation_search", &toolErr)()

		// Validate project_path (CWE-22 path traversal protection)
		if args.ProjectPath == "" {
			toolErr = fmt.Errorf("project_path is required")
			return nil, conversationSearchOutput{}, toolErr
		}
		validPath, err := sanitize.ValidateProjectPath(args.ProjectPath)
		if err != nil {
			toolErr = fmt.Errorf("invalid project_path: %w", err)
			return nil, conversationSearchOutput{}, toolErr
		}

		// Validate file_path filter if provided (CWE-22 path traversal protection)
		validFilePath := args.FilePath
		if validFilePath != "" {
			validFilePath, err = sanitize.ValidatePath(args.FilePath, "")
			if err != nil {
				toolErr = fmt.Errorf("invalid file_path filter: %w", err)
				return nil, conversationSearchOutput{}, toolErr
			}
		}

		tenantID := args.TenantID
		if tenantID == "" {
			tenantID = tenant.GetTenantIDForPath(validPath)
		}
		// Validate tenant_id format (CWE-287 authentication bypass protection)
		if tenantID == "" {
			toolErr = fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
			return nil, conversationSearchOutput{}, toolErr
		}
		if err := sanitize.ValidateTenantID(tenantID); err != nil {
			toolErr = fmt.Errorf("invalid tenant_id: %w", err)
			return nil, conversationSearchOutput{}, toolErr
		}

		// Convert string types to DocumentType
		var docTypes []conversation.DocumentType
		for _, t := range args.Types {
			docTypes = append(docTypes, conversation.DocumentType(t))
		}

		opts := conversation.SearchOptions{
			Query:       args.Query,
			ProjectPath: validPath,
			TenantID:    tenantID,
			Types:       docTypes,
			Tags:        args.Tags,
			FilePath:    validFilePath,
			Domain:      args.Domain,
			Limit:       args.Limit,
		}

		result, err := s.conversationSvc.Search(ctx, opts)
		if err != nil {
			toolErr = fmt.Errorf("search failed: %w", err)
			return nil, conversationSearchOutput{}, toolErr
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
