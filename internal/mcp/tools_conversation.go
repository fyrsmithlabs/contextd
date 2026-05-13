package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fyrsmithlabs/contextd/internal/conversation"
	"github.com/fyrsmithlabs/contextd/internal/sanitize"
)

// ===== CONVERSATION TOOLS =====

type conversationIndexInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"required,Path to project to index conversations for"`
	TenantID    string   `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	SessionIDs  []string `json:"session_ids,omitempty" jsonschema:"Specific session IDs to index (empty = all)"`
	EnableLLM   bool     `json:"enable_llm,omitempty" jsonschema:"Enable LLM-based decision extraction (reserved; not yet implemented)"`
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
	Limit       int      `json:"limit,omitempty" jsonschema:"Maximum results to return (default: 10, max: 100)"`
}

// conversationSearchRow is a single typed search result row. Replaces the
// previous untyped map[string]interface{} so the SDK can emit a complete
// output schema (see HANDLER-GUIDE.md §4.4).
type conversationSearchRow struct {
	ID        string   `json:"id" jsonschema:"Document identifier"`
	SessionID string   `json:"session_id" jsonschema:"Originating conversation session ID"`
	Type      string   `json:"type" jsonschema:"Document type: message, decision, or summary"`
	Content   string   `json:"content" jsonschema:"Document content (secret-scrubbed)"`
	Score     float64  `json:"score" jsonschema:"Relevance score (0-1)"`
	Timestamp int64    `json:"timestamp" jsonschema:"Document timestamp (Unix seconds)"`
	Tags      []string `json:"tags,omitempty" jsonschema:"Document tags, if any"`
	Domain    string   `json:"domain,omitempty" jsonschema:"Document domain, if classified"`
}

type conversationSearchOutput struct {
	Query   string                  `json:"query" jsonschema:"Search query used"`
	Results []conversationSearchRow `json:"results" jsonschema:"Search results with score and content"`
	Total   int                     `json:"total" jsonschema:"Total number of results"`
	TookMs  int64                   `json:"took_ms" jsonschema:"Search duration in milliseconds"`
}

func (s *Server) registerConversationTools() {
	if s.conversationSvc == nil {
		s.logger.Warn("conversation service not configured, skipping conversation tools")
		return
	}

	// conversation_index — append-only write touching the vectorstore.
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "conversation_index",
		Description: "Index Claude Code conversation JSONL files for a project, storing messages and heuristic decisions for semantic search.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptrFalse(),
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args conversationIndexInput) (*mcp.CallToolResult, conversationIndexOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "conversation_index", &toolErr)()

		// Reject LLM requests explicitly until implemented.
		if args.EnableLLM {
			toolErr = fmt.Errorf("enable_llm=true is not yet supported: LLM-based decision extraction is not implemented; set enable_llm=false or omit the parameter to use heuristic extraction")
			return nil, conversationIndexOutput{}, toolErr
		}

		// project_path is required for conversation_index — surface that
		// explicitly before tenantCtx so the error names the missing field.
		if args.ProjectPath == "" {
			toolErr = fmt.Errorf("project_path is required")
			return nil, conversationIndexOutput{}, toolErr
		}

		ctx, rt, err := s.tenantCtx(ctx, args.ProjectPath, args.TenantID, "", "")
		if err != nil {
			toolErr = err
			return nil, conversationIndexOutput{}, toolErr
		}

		opts := conversation.IndexOptions{
			ProjectPath: rt.ValidPath,
			TenantID:    rt.TenantID,
			SessionIDs:  args.SessionIDs,
			EnableLLM:   args.EnableLLM,
			Force:       args.Force,
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

	// conversation_search — pure read touching the vectorstore.
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "conversation_search",
		Description: "Search indexed Claude Code conversations for relevant past context, decisions, and patterns.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args conversationSearchInput) (*mcp.CallToolResult, conversationSearchOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "conversation_search", &toolErr)()

		if args.ProjectPath == "" {
			toolErr = fmt.Errorf("project_path is required")
			return nil, conversationSearchOutput{}, toolErr
		}

		ctx, rt, err := s.tenantCtx(ctx, args.ProjectPath, args.TenantID, "", "")
		if err != nil {
			toolErr = err
			return nil, conversationSearchOutput{}, toolErr
		}

		// Validate file_path filter if provided (CWE-22 path traversal protection).
		validFilePath := args.FilePath
		if validFilePath != "" {
			validFilePath, err = sanitize.ValidatePath(args.FilePath, "")
			if err != nil {
				toolErr = fmt.Errorf("invalid file_path filter: %w", err)
				return nil, conversationSearchOutput{}, toolErr
			}
		}

		// Convert string types to DocumentType.
		var docTypes []conversation.DocumentType
		for _, t := range args.Types {
			docTypes = append(docTypes, conversation.DocumentType(t))
		}

		limit := args.Limit
		if limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}

		opts := conversation.SearchOptions{
			Query:       args.Query,
			ProjectPath: rt.ValidPath,
			TenantID:    rt.TenantID,
			Types:       docTypes,
			Tags:        args.Tags,
			FilePath:    validFilePath,
			Domain:      args.Domain,
			Limit:       limit,
		}

		result, err := s.conversationSvc.Search(ctx, opts)
		if err != nil {
			toolErr = fmt.Errorf("search failed: %w", err)
			return nil, conversationSearchOutput{}, toolErr
		}

		// Convert results to typed rows, scrubbing content along the way.
		rows := make([]conversationSearchRow, 0, len(result.Results))
		for _, hit := range result.Results {
			content := hit.Document.Content
			if s.scrubber != nil {
				content = s.scrubber.Scrub(hit.Document.Content).Scrubbed
			}
			rows = append(rows, conversationSearchRow{
				ID:        hit.Document.ID,
				SessionID: hit.Document.SessionID,
				Type:      string(hit.Document.Type),
				Content:   content,
				Score:     hit.Score,
				Timestamp: hit.Document.Timestamp.Unix(),
				Tags:      hit.Document.Tags,
				Domain:    hit.Document.Domain,
			})
		}

		output := conversationSearchOutput{
			Query:   result.Query,
			Results: rows,
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
