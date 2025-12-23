package mcp

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/folding"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/tenant"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// injectTenantContext adds tenant information to context for payload-based isolation.
// This enables vectorstore operations to automatically filter by tenant when using
// PayloadIsolation mode. The tenant info is derived from:
//   - tenantID: organization/user identifier (required)
//   - teamID: team scope (optional, empty string if not applicable)
//   - projectID: project scope (optional, empty string if not applicable)
//
// Returns the original context if tenantID is empty.
func injectTenantContext(ctx context.Context, tenantID, teamID, projectID string) context.Context {
	if tenantID == "" {
		return ctx
	}
	return vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  tenantID,
		TeamID:    teamID,
		ProjectID: projectID,
	})
}

// registerTools registers all MCP tools with the server.
func (s *Server) registerTools() error {
	// Checkpoint tools
	s.registerCheckpointTools()

	// Remediation tools
	s.registerRemediationTools()

	// Repository tools
	s.registerRepositoryTools()

	// Troubleshoot tools
	s.registerTroubleshootTools()

	// Memory tools (ReasoningBank)
	s.registerMemoryTools()

	// Folding tools (context-folding branch/return)
	s.registerFoldingTools()

	return nil
}

// ===== CHECKPOINT TOOLS =====

type checkpointSaveInput struct {
	SessionID   string            `json:"session_id" jsonschema:"required,Session identifier"`
	TenantID    string            `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	ProjectPath string            `json:"project_path" jsonschema:"required,Project path"`
	Name        string            `json:"name" jsonschema:"Checkpoint name"`
	Description string            `json:"description" jsonschema:"Human-readable description"`
	Summary     string            `json:"summary" jsonschema:"Brief summary for quick reference"`
	Context     string            `json:"context" jsonschema:"Contextual information"`
	FullState   string            `json:"full_state" jsonschema:"Complete session state"`
	TokenCount  int32             `json:"token_count" jsonschema:"Token count estimate"`
	Threshold   float64           `json:"threshold" jsonschema:"Context threshold that triggered checkpoint"`
	AutoCreated bool              `json:"auto_created" jsonschema:"True if auto-created by system"`
	Metadata    map[string]string `json:"metadata,omitempty" jsonschema:"Additional metadata"`
}

type checkpointSaveOutput struct {
	ID          string `json:"id" jsonschema:"Checkpoint ID"`
	SessionID   string `json:"session_id" jsonschema:"Session ID"`
	Summary     string `json:"summary" jsonschema:"Checkpoint summary"`
	TokenCount  int32  `json:"token_count" jsonschema:"Token count"`
	AutoCreated bool   `json:"auto_created" jsonschema:"Auto-created flag"`
}

type checkpointListInput struct {
	SessionID   string `json:"session_id,omitempty" jsonschema:"Filter by session ID"`
	TenantID    string `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Filter by project path (used to derive tenant_id via git remote)"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum results to return (default: 20)"`
	AutoOnly    bool   `json:"auto_only,omitempty" jsonschema:"Only return auto-created checkpoints"`
}

type checkpointListOutput struct {
	Checkpoints []map[string]interface{} `json:"checkpoints" jsonschema:"List of checkpoints"`
	Count       int                      `json:"count" jsonschema:"Number of checkpoints returned"`
}

type checkpointResumeInput struct {
	CheckpointID string                 `json:"checkpoint_id" jsonschema:"required,Checkpoint ID to resume"`
	TenantID     string                 `json:"tenant_id" jsonschema:"required,Tenant identifier"`
	Level        checkpoint.ResumeLevel `json:"level" jsonschema:"required,Resume level (summary context or full)"`
}

type checkpointResumeOutput struct {
	CheckpointID string `json:"checkpoint_id" jsonschema:"Checkpoint ID"`
	SessionID    string `json:"session_id" jsonschema:"Original session ID"`
	Content      string `json:"content" jsonschema:"Restored content at requested level"`
	TokenCount   int32  `json:"token_count" jsonschema:"Token count of restored content"`
	Level        string `json:"level" jsonschema:"Resume level used"`
}

func (s *Server) registerCheckpointTools() {
	// checkpoint_save
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "checkpoint_save",
		Description: "Save a session checkpoint for later resumption",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args checkpointSaveInput) (*mcp.CallToolResult, checkpointSaveOutput, error) {
		// Auto-derive tenant_id from project_path if not provided
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}

		// Validate tenant_id was derived successfully
		if tenantID == "" {
			return nil, checkpointSaveOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		// Derive project_id from project_path (directory name)
		projectID := ""
		if args.ProjectPath != "" {
			projectID = filepath.Base(args.ProjectPath)
		}

		saveReq := &checkpoint.SaveRequest{
			SessionID:   args.SessionID,
			TenantID:    tenantID,
			TeamID:      "", // Empty team is allowed
			ProjectID:   projectID,
			ProjectPath: args.ProjectPath,
			Name:        args.Name,
			Description: args.Description,
			Summary:     args.Summary,
			Context:     args.Context,
			FullState:   args.FullState,
			TokenCount:  args.TokenCount,
			Threshold:   args.Threshold,
			AutoCreated: args.AutoCreated,
			Metadata:    args.Metadata,
		}

		cp, err := s.checkpointSvc.Save(ctx, saveReq)
		if err != nil {
			return nil, checkpointSaveOutput{}, fmt.Errorf("checkpoint save failed: %w", err)
		}

		result := checkpointSaveOutput{
			ID:          cp.ID,
			SessionID:   cp.SessionID,
			Summary:     cp.Summary,
			TokenCount:  cp.TokenCount,
			AutoCreated: cp.AutoCreated,
		}

		// Scrub response
		scrubbed := s.scrubber.Scrub(result.Summary)
		result.Summary = scrubbed.Scrubbed

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Checkpoint saved: %s", result.ID)},
			},
		}, result, nil
	})

	// checkpoint_list
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "checkpoint_list",
		Description: "List checkpoints for a session or project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args checkpointListInput) (*mcp.CallToolResult, checkpointListOutput, error) {
		// Auto-derive tenant_id from project_path if not provided
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}

		// Validate tenant_id was derived successfully
		if tenantID == "" {
			return nil, checkpointListOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		// Derive project_id from project_path (directory name)
		projectID := ""
		if args.ProjectPath != "" {
			projectID = filepath.Base(args.ProjectPath)
		}

		listReq := &checkpoint.ListRequest{
			SessionID:   args.SessionID,
			TenantID:    tenantID,
			TeamID:      "", // Empty team is allowed
			ProjectID:   projectID,
			ProjectPath: args.ProjectPath,
			Limit:       args.Limit,
			AutoOnly:    args.AutoOnly,
		}

		checkpoints, err := s.checkpointSvc.List(ctx, listReq)
		if err != nil {
			return nil, checkpointListOutput{}, fmt.Errorf("checkpoint list failed: %w", err)
		}

		results := make([]map[string]interface{}, 0, len(checkpoints))
		for _, cp := range checkpoints {
			results = append(results, map[string]interface{}{
				"id":           cp.ID,
				"session_id":   cp.SessionID,
				"name":         cp.Name,
				"description":  cp.Description,
				"summary":      cp.Summary,
				"token_count":  cp.TokenCount,
				"threshold":    cp.Threshold,
				"auto_created": cp.AutoCreated,
				"created_at":   cp.CreatedAt,
			})
		}

		output := checkpointListOutput{
			Checkpoints: results,
			Count:       len(results),
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d checkpoints", output.Count)},
			},
		}, output, nil
	})

	// checkpoint_resume
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "checkpoint_resume",
		Description: "Resume from a checkpoint at specified level (summary, context, or full)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args checkpointResumeInput) (*mcp.CallToolResult, checkpointResumeOutput, error) {
		resumeReq := &checkpoint.ResumeRequest{
			CheckpointID: args.CheckpointID,
			TenantID:     args.TenantID,
			Level:        args.Level,
		}

		response, err := s.checkpointSvc.Resume(ctx, resumeReq)
		if err != nil {
			return nil, checkpointResumeOutput{}, fmt.Errorf("checkpoint resume failed: %w", err)
		}

		result := checkpointResumeOutput{
			CheckpointID: response.Checkpoint.ID,
			SessionID:    response.Checkpoint.SessionID,
			Content:      response.Content,
			TokenCount:   response.TokenCount,
			Level:        string(args.Level),
		}

		// Scrub content
		scrubbed := s.scrubber.Scrub(result.Content)
		result.Content = scrubbed.Scrubbed

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Resumed checkpoint %s at level %s", result.CheckpointID, result.Level)},
			},
		}, result, nil
	})
}

// ===== REMEDIATION TOOLS =====

type remediationSearchInput struct {
	Query            string                    `json:"query" jsonschema:"required,Error message or pattern to search for"`
	TenantID         string                    `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	Scope            remediation.Scope         `json:"scope,omitempty" jsonschema:"Search scope (project team or org)"`
	Category         remediation.ErrorCategory `json:"category,omitempty" jsonschema:"Error category filter"`
	MinConfidence    float64                   `json:"min_confidence,omitempty" jsonschema:"Minimum confidence threshold (0-1)"`
	Limit            int                       `json:"limit,omitempty" jsonschema:"Maximum results (default: 10)"`
	TeamID           string                    `json:"team_id,omitempty" jsonschema:"Team ID for team/project scope"`
	ProjectPath      string                    `json:"project_path,omitempty" jsonschema:"Project path for project scope (used to auto-derive tenant_id if empty)"`
	IncludeHierarchy bool                      `json:"include_hierarchy,omitempty" jsonschema:"Search parent scopes (project→team→org)"`
}

type remediationSearchOutput struct {
	Remediations []map[string]interface{} `json:"remediations" jsonschema:"Matching remediations with scores"`
	Count        int                      `json:"count" jsonschema:"Number of results"`
}

type remediationRecordInput struct {
	Title         string                    `json:"title" jsonschema:"required,Brief title"`
	Problem       string                    `json:"problem" jsonschema:"required,Problem description"`
	Symptoms      []string                  `json:"symptoms,omitempty" jsonschema:"Observable symptoms"`
	RootCause     string                    `json:"root_cause" jsonschema:"required,Root cause analysis"`
	Solution      string                    `json:"solution" jsonschema:"required,How to fix it"`
	CodeDiff      string                    `json:"code_diff,omitempty" jsonschema:"Code changes (diff format)"`
	AffectedFiles []string                  `json:"affected_files,omitempty" jsonschema:"Files that were changed"`
	Category      remediation.ErrorCategory `json:"category" jsonschema:"required,Error category"`
	Confidence    float64                   `json:"confidence,omitempty" jsonschema:"Confidence score (0-1 default 0.5)"`
	Tags          []string                  `json:"tags,omitempty" jsonschema:"Tags for categorization"`
	TenantID      string                    `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
	Scope         remediation.Scope         `json:"scope" jsonschema:"required,Scope level (project team or org)"`
	TeamID        string                    `json:"team_id,omitempty" jsonschema:"Team ID (for team/project scope)"`
	ProjectPath   string                    `json:"project_path,omitempty" jsonschema:"Project path (used to derive tenant_id via git remote)"`
	SessionID     string                    `json:"session_id,omitempty" jsonschema:"Session that created this remediation"`
}

type remediationRecordOutput struct {
	ID         string  `json:"id" jsonschema:"Remediation ID"`
	Title      string  `json:"title" jsonschema:"Remediation title"`
	Category   string  `json:"category" jsonschema:"Error category"`
	Confidence float64 `json:"confidence" jsonschema:"Confidence score"`
}

func (s *Server) registerRemediationTools() {
	// remediation_search
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "remediation_search",
		Description: "Search for remediations by error message or pattern",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args remediationSearchInput) (*mcp.CallToolResult, remediationSearchOutput, error) {
		// Auto-derive tenant_id from project_path if not provided
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}

		// Validate tenant_id was derived successfully
		if tenantID == "" {
			return nil, remediationSearchOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		searchReq := &remediation.SearchRequest{
			Query:            args.Query,
			TenantID:         tenantID,
			Scope:            args.Scope,
			Category:         args.Category,
			MinConfidence:    args.MinConfidence,
			Limit:            args.Limit,
			TeamID:           args.TeamID,
			ProjectPath:      args.ProjectPath,
			IncludeHierarchy: args.IncludeHierarchy,
		}

		results, err := s.remediationSvc.Search(ctx, searchReq)
		if err != nil {
			return nil, remediationSearchOutput{}, fmt.Errorf("remediation search failed: %w", err)
		}

		remediations := make([]map[string]interface{}, 0, len(results))
		for _, r := range results {
			remediations = append(remediations, map[string]interface{}{
				"id":          r.Remediation.ID,
				"title":       r.Remediation.Title,
				"problem":     r.Remediation.Problem,
				"root_cause":  r.Remediation.RootCause,
				"solution":    r.Remediation.Solution,
				"category":    string(r.Remediation.Category),
				"confidence":  r.Remediation.Confidence,
				"score":       r.Score,
				"usage_count": r.Remediation.UsageCount,
			})
		}

		output := remediationSearchOutput{
			Remediations: remediations,
			Count:        len(remediations),
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d remediations", output.Count)},
			},
		}, output, nil
	})

	// remediation_record
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "remediation_record",
		Description: "Record a new remediation for an error that was successfully fixed",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args remediationRecordInput) (*mcp.CallToolResult, remediationRecordOutput, error) {
		// Auto-derive tenant_id from project_path if not provided
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}

		// Validate tenant_id was derived successfully
		if tenantID == "" {
			return nil, remediationRecordOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		recordReq := &remediation.RecordRequest{
			Title:         args.Title,
			Problem:       args.Problem,
			Symptoms:      args.Symptoms,
			RootCause:     args.RootCause,
			Solution:      args.Solution,
			CodeDiff:      args.CodeDiff,
			AffectedFiles: args.AffectedFiles,
			Category:      args.Category,
			Confidence:    args.Confidence,
			Tags:          args.Tags,
			TenantID:      tenantID,
			Scope:         args.Scope,
			TeamID:        args.TeamID,
			ProjectPath:   args.ProjectPath,
			SessionID:     args.SessionID,
		}

		rem, err := s.remediationSvc.Record(ctx, recordReq)
		if err != nil {
			return nil, remediationRecordOutput{}, fmt.Errorf("remediation record failed: %w", err)
		}

		result := remediationRecordOutput{
			ID:         rem.ID,
			Title:      rem.Title,
			Category:   string(rem.Category),
			Confidence: rem.Confidence,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Remediation recorded: %s", result.ID)},
			},
		}, result, nil
	})
}

// ===== REPOSITORY TOOLS =====

type semanticSearchInput struct {
	Query       string `json:"query" jsonschema:"required,Search query (natural language or pattern)"`
	ProjectPath string `json:"project_path" jsonschema:"required,Project path to search within"`
	TenantID    string `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (defaults to git username)"`
	Branch      string `json:"branch,omitempty" jsonschema:"Filter by branch (empty = all branches)"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum results (default: 10)"`
}

type semanticSearchOutput struct {
	Results []map[string]interface{} `json:"results" jsonschema:"Search results with file paths and content"`
	Count   int                      `json:"count" jsonschema:"Number of results returned"`
	Query   string                   `json:"query" jsonschema:"Original search query"`
	Source  string                   `json:"source" jsonschema:"Source of results (semantic or grep)"`
}

type repositoryIndexInput struct {
	Path            string   `json:"path" jsonschema:"required,Repository path to index"`
	TenantID        string   `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (defaults to git username)"`
	Branch          string   `json:"branch,omitempty" jsonschema:"Git branch to index (auto-detects if empty)"`
	IncludePatterns []string `json:"include_patterns,omitempty" jsonschema:"Glob patterns to include (e.g. *.go)"`
	ExcludePatterns []string `json:"exclude_patterns,omitempty" jsonschema:"Glob patterns to exclude (e.g. vendor/**)"`
	MaxFileSize     int64    `json:"max_file_size,omitempty" jsonschema:"Maximum file size in bytes (default 1MB)"`
}

type repositoryIndexOutput struct {
	Path            string   `json:"path" jsonschema:"Indexed path"`
	Branch          string   `json:"branch" jsonschema:"Git branch indexed"`
	CollectionName  string   `json:"collection_name" jsonschema:"Qdrant collection name"`
	FilesIndexed    int      `json:"files_indexed" jsonschema:"Number of files indexed"`
	IncludePatterns []string `json:"include_patterns" jsonschema:"Include patterns used"`
	ExcludePatterns []string `json:"exclude_patterns" jsonschema:"Exclude patterns used"`
	MaxFileSize     int64    `json:"max_file_size" jsonschema:"Max file size used"`
}

type repositorySearchInput struct {
	Query          string `json:"query" jsonschema:"required,Semantic search query"`
	ProjectPath    string `json:"project_path,omitempty" jsonschema:"Project path to search within (optional if collection_name provided)"`
	CollectionName string `json:"collection_name,omitempty" jsonschema:"Collection name from repository_index (preferred - avoids tenant_id derivation issues)"`
	TenantID       string `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (defaults to git username)"`
	Branch         string `json:"branch,omitempty" jsonschema:"Filter by branch (empty = all branches)"`
	Limit          int    `json:"limit,omitempty" jsonschema:"Maximum results (default: 10)"`
}

type repositorySearchOutput struct {
	Results []map[string]interface{} `json:"results" jsonschema:"Search results with file paths and content"`
	Count   int                      `json:"count" jsonschema:"Number of results returned"`
	Query   string                   `json:"query" jsonschema:"Original search query"`
	Branch  string                   `json:"branch,omitempty" jsonschema:"Branch filter applied (if any)"`
}

func (s *Server) registerRepositoryTools() {
	// semantic_search
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "semantic_search",
		Description: "Smart search that uses semantic understanding, falling back to grep if needed. Use this when the agent would normally use the Search tool.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args semanticSearchInput) (*mcp.CallToolResult, semanticSearchOutput, error) {
		// Default tenant ID from project path if not specified
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}

		// Validate tenant_id was derived successfully
		if tenantID == "" {
			return nil, semanticSearchOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		opts := repository.SearchOptions{
			ProjectPath: args.ProjectPath,
			TenantID:    tenantID,
			Branch:      args.Branch,
			Limit:       args.Limit,
		}

		// 1. Try Semantic Search
		results, err := s.repositorySvc.Search(ctx, args.Query, opts)

		// Decide if fallback is needed
		useFallback := false
		if err != nil {
			s.logger.Warn("semantic search failed, falling back to grep", zap.Error(err))
			useFallback = true
		} else if len(results) == 0 {
			useFallback = true
		}

		outputResults := make([]map[string]interface{}, 0)
		source := "semantic"

		if !useFallback {
			for _, r := range results {
				scrubbed := s.scrubber.Scrub(r.Content).Scrubbed
				outputResults = append(outputResults, map[string]interface{}{
					"file_path": r.FilePath,
					"content":   scrubbed,
					"score":     r.Score,
					"branch":    r.Branch,
					"metadata":  r.Metadata,
				})
			}
		} else {
			// 2. Fallback to Grep
			source = "grep"

			// Parse project's ignore files for grep
			excludePatterns := []string{}
			parsed, parseErr := s.ignoreParser.ParseProject(args.ProjectPath)
			if parseErr != nil {
				s.logger.Warn("failed to parse ignore files for grep, using fallback",
					zap.String("path", args.ProjectPath),
					zap.Error(parseErr))
				excludePatterns = s.ignoreParser.FallbackPatterns
			} else {
				excludePatterns = parsed
			}

			grepOpts := repository.GrepOptions{
				ProjectPath:     args.ProjectPath,
				ExcludePatterns: excludePatterns,
				CaseSensitive:   false, // Default to case-insensitive for better fallback experience
			}

			grepResults, err := s.repositorySvc.Grep(ctx, args.Query, grepOpts)
			if err != nil {
				// If semantic failed AND grep failed, return error
				return nil, semanticSearchOutput{}, fmt.Errorf("search failed: %w", err)
			}

			// Apply limit manually for grep results
			limit := args.Limit
			if limit <= 0 {
				limit = 10
			}
			if len(grepResults) > limit {
				grepResults = grepResults[:limit]
			}

			for _, r := range grepResults {
				scrubbed := s.scrubber.Scrub(r.Content).Scrubbed
				outputResults = append(outputResults, map[string]interface{}{
					"file_path":   r.FilePath,
					"content":     scrubbed,
					"line_number": r.LineNumber,
					"score":       1.0,
				})
			}
		}

		output := semanticSearchOutput{
			Results: outputResults,
			Count:   len(outputResults),
			Query:   args.Query,
			Source:  source,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d results for query: %s (source: %s)", output.Count, args.Query, output.Source)},
			},
		}, output, nil
	})

	// repository_search
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "repository_search",
		Description: "Semantic search over indexed repository code in _codebase collection. Prefer using collection_name from repository_index output.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args repositorySearchInput) (*mcp.CallToolResult, repositorySearchOutput, error) {
		// project_path is always required for tenant context (fail-closed security)
		if args.ProjectPath == "" {
			return nil, repositorySearchOutput{}, fmt.Errorf("project_path is required for tenant context")
		}

		// Derive tenant_id if not provided
		tenantID := args.TenantID
		if tenantID == "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}
		if tenantID == "" {
			return nil, repositorySearchOutput{}, fmt.Errorf("tenant_id is required: provide tenant_id explicitly or ensure project_path is set")
		}

		opts := repository.SearchOptions{
			CollectionName: args.CollectionName,
			ProjectPath:    args.ProjectPath,
			TenantID:       tenantID,
			Branch:         args.Branch,
			Limit:          args.Limit,
		}

		results, err := s.repositorySvc.Search(ctx, args.Query, opts)
		if err != nil {
			return nil, repositorySearchOutput{}, fmt.Errorf("repository search failed: %w", err)
		}

		// Convert to output format
		outputResults := make([]map[string]interface{}, 0, len(results))
		for _, r := range results {
			// Scrub content before returning
			scrubbedContent := s.scrubber.Scrub(r.Content).Scrubbed

			outputResults = append(outputResults, map[string]interface{}{
				"file_path": r.FilePath,
				"content":   scrubbedContent,
				"score":     r.Score,
				"branch":    r.Branch,
				"metadata":  r.Metadata,
			})
		}

		output := repositorySearchOutput{
			Results: outputResults,
			Count:   len(outputResults),
			Query:   args.Query,
			Branch:  args.Branch,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d results for query: %s", output.Count, args.Query)},
			},
		}, output, nil
	})

	// repository_index
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "repository_index",
		Description: "Index a repository for semantic code search",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args repositoryIndexInput) (*mcp.CallToolResult, repositoryIndexOutput, error) {
		// Default include patterns to ["*"] for full indexing (explicit patterns for differential)
		includePatterns := args.IncludePatterns
		if len(includePatterns) == 0 {
			includePatterns = []string{"*"}
		}

		// Get exclude patterns: use explicit args, or parse project's ignore files
		excludePatterns := args.ExcludePatterns
		if len(excludePatterns) == 0 {
			// Parse project's ignore files (.gitignore, .dockerignore, etc.)
			// Falls back to config defaults if no ignore files found
			parsed, err := s.ignoreParser.ParseProject(args.Path)
			if err != nil {
				s.logger.Warn("failed to parse ignore files, using fallback",
					zap.String("path", args.Path),
					zap.Error(err))
				excludePatterns = s.ignoreParser.FallbackPatterns
			} else {
				excludePatterns = parsed
			}
		}

		// Ensure excludePatterns is never nil for output
		if excludePatterns == nil {
			excludePatterns = []string{}
		}

		opts := repository.IndexOptions{
			TenantID:        args.TenantID,
			Branch:          args.Branch,
			IncludePatterns: includePatterns,
			ExcludePatterns: excludePatterns,
			MaxFileSize:     args.MaxFileSize,
		}

		result, err := s.repositorySvc.IndexRepository(ctx, args.Path, opts)
		if err != nil {
			return nil, repositoryIndexOutput{}, fmt.Errorf("repository index failed: %w", err)
		}

		// Ensure output arrays are never nil (MCP schema validation requires arrays, not null)
		outputInclude := result.IncludePatterns
		if outputInclude == nil {
			outputInclude = includePatterns
		}
		outputExclude := result.ExcludePatterns
		if outputExclude == nil {
			outputExclude = excludePatterns
		}

		output := repositoryIndexOutput{
			Path:            result.Path,
			Branch:          result.Branch,
			CollectionName:  result.CollectionName,
			FilesIndexed:    result.FilesIndexed,
			IncludePatterns: outputInclude,
			ExcludePatterns: outputExclude,
			MaxFileSize:     result.MaxFileSize,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Indexed %d files from %s (branch: %s, collection: %s)", output.FilesIndexed, output.Path, output.Branch, output.CollectionName)},
			},
		}, output, nil
	})
}

// ===== TROUBLESHOOT TOOLS =====

type troubleshootDiagnoseInput struct {
	ErrorMessage string `json:"error_message" jsonschema:"required,Error message to diagnose"`
	ErrorContext string `json:"error_context,omitempty" jsonschema:"Additional context (stack trace logs etc)"`
}

type troubleshootDiagnoseOutput struct {
	ErrorMessage    string                     `json:"error_message" jsonschema:"Original error message"`
	RootCause       string                     `json:"root_cause" jsonschema:"Likely root cause"`
	Hypotheses      []troubleshoot.Hypothesis  `json:"hypotheses" jsonschema:"Diagnostic hypotheses"`
	Recommendations []string                   `json:"recommendations" jsonschema:"Recommended actions"`
	RelatedPatterns []troubleshoot.Pattern     `json:"related_patterns" jsonschema:"Similar known patterns"`
	Confidence      float64                    `json:"confidence" jsonschema:"Overall confidence (0-1)"`
}

func (s *Server) registerTroubleshootTools() {
	// troubleshoot_diagnose
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "troubleshoot_diagnose",
		Description: "Diagnose an error using AI and known patterns",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args troubleshootDiagnoseInput) (*mcp.CallToolResult, troubleshootDiagnoseOutput, error) {
		diagnosis, err := s.troubleshootSvc.Diagnose(ctx, args.ErrorMessage, args.ErrorContext)
		if err != nil {
			return nil, troubleshootDiagnoseOutput{}, fmt.Errorf("troubleshoot diagnose failed: %w", err)
		}

		output := troubleshootDiagnoseOutput{
			ErrorMessage:    diagnosis.ErrorMessage,
			RootCause:       diagnosis.RootCause,
			Hypotheses:      diagnosis.Hypotheses,
			Recommendations: diagnosis.Recommendations,
			RelatedPatterns: diagnosis.RelatedPatterns,
			Confidence:      diagnosis.Confidence,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Diagnosis complete (confidence: %.2f): %s", output.Confidence, output.RootCause)},
			},
		}, output, nil
	})
}

// ===== MEMORY TOOLS (ReasoningBank) =====

type memorySearchInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,Project identifier"`
	Query     string `json:"query" jsonschema:"required,Search query for relevant memories"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum results (default: 5)"`
}

type memorySearchOutput struct {
	Memories []map[string]interface{} `json:"memories" jsonschema:"Matching memories"`
	Count    int                      `json:"count" jsonschema:"Number of results"`
}

type memoryRecordInput struct {
	ProjectID string   `json:"project_id" jsonschema:"required,Project identifier"`
	Title     string   `json:"title" jsonschema:"required,Brief title for the memory"`
	Content   string   `json:"content" jsonschema:"required,The strategy or learning to remember"`
	Outcome   string   `json:"outcome" jsonschema:"required,Outcome type (success or failure)"`
	Tags      []string `json:"tags,omitempty" jsonschema:"Tags for categorization"`
}

type memoryRecordOutput struct {
	ID         string  `json:"id" jsonschema:"Memory ID"`
	Title      string  `json:"title" jsonschema:"Memory title"`
	Outcome    string  `json:"outcome" jsonschema:"Outcome type"`
	Confidence float64 `json:"confidence" jsonschema:"Initial confidence"`
}

type memoryFeedbackInput struct {
	MemoryID string `json:"memory_id" jsonschema:"required,Memory ID to provide feedback on"`
	Helpful  bool   `json:"helpful" jsonschema:"required,Whether the memory was helpful"`
}

type memoryFeedbackOutput struct {
	MemoryID      string  `json:"memory_id" jsonschema:"Memory ID"`
	NewConfidence float64 `json:"new_confidence" jsonschema:"Updated confidence after feedback"`
	Helpful       bool    `json:"helpful" jsonschema:"Feedback provided"`
}

type memoryOutcomeInput struct {
	MemoryID  string `json:"memory_id" jsonschema:"required,ID of the memory that was used"`
	Succeeded bool   `json:"succeeded" jsonschema:"required,Whether the task succeeded after using this memory"`
	SessionID string `json:"session_id,omitempty" jsonschema:"Optional session ID for correlation"`
}

type memoryOutcomeOutput struct {
	Recorded      bool    `json:"recorded" jsonschema:"Whether outcome was recorded"`
	NewConfidence float64 `json:"new_confidence" jsonschema:"Updated confidence after outcome"`
	Message       string  `json:"message" jsonschema:"Result message"`
}

func (s *Server) registerMemoryTools() {
	// memory_search
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_search",
		Description: "Search for relevant memories/strategies from past sessions",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memorySearchInput) (*mcp.CallToolResult, memorySearchOutput, error) {
		limit := args.Limit
		if limit <= 0 {
			limit = 5
		}

		memories, err := s.reasoningbankSvc.Search(ctx, args.ProjectID, args.Query, limit)
		if err != nil {
			return nil, memorySearchOutput{}, fmt.Errorf("memory search failed: %w", err)
		}

		results := make([]map[string]interface{}, 0, len(memories))
		for _, m := range memories {
			results = append(results, map[string]interface{}{
				"id":         m.ID,
				"title":      m.Title,
				"content":    s.scrubber.Scrub(m.Content).Scrubbed,
				"outcome":    m.Outcome,
				"confidence": m.Confidence,
				"tags":       m.Tags,
			})
		}

		output := memorySearchOutput{
			Memories: results,
			Count:    len(results),
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d relevant memories", output.Count)},
			},
		}, output, nil
	})

	// memory_record
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_record",
		Description: "Record a new memory/learning from the current session",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryRecordInput) (*mcp.CallToolResult, memoryRecordOutput, error) {
		outcome := reasoningbank.OutcomeSuccess
		if args.Outcome == "failure" {
			outcome = reasoningbank.OutcomeFailure
		}

		memory, err := reasoningbank.NewMemory(args.ProjectID, args.Title, args.Content, outcome, args.Tags)
		if err != nil {
			return nil, memoryRecordOutput{}, fmt.Errorf("invalid memory: %w", err)
		}

		if err := s.reasoningbankSvc.Record(ctx, memory); err != nil {
			return nil, memoryRecordOutput{}, fmt.Errorf("memory record failed: %w", err)
		}

		output := memoryRecordOutput{
			ID:         memory.ID,
			Title:      memory.Title,
			Outcome:    string(memory.Outcome),
			Confidence: memory.Confidence,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory recorded: %s (confidence: %.2f)", output.Title, output.Confidence)},
			},
		}, output, nil
	})

	// memory_feedback
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_feedback",
		Description: "Provide feedback on a memory to adjust its confidence",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryFeedbackInput) (*mcp.CallToolResult, memoryFeedbackOutput, error) {
		if err := s.reasoningbankSvc.Feedback(ctx, args.MemoryID, args.Helpful); err != nil {
			return nil, memoryFeedbackOutput{}, fmt.Errorf("memory feedback failed: %w", err)
		}

		// Get updated memory to return new confidence
		memory, err := s.reasoningbankSvc.Get(ctx, args.MemoryID)
		if err != nil {
			return nil, memoryFeedbackOutput{}, fmt.Errorf("failed to get updated memory: %w", err)
		}

		output := memoryFeedbackOutput{
			MemoryID:      args.MemoryID,
			NewConfidence: memory.Confidence,
			Helpful:       args.Helpful,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Feedback recorded, new confidence: %.2f", output.NewConfidence)},
			},
		}, output, nil
	})

	// memory_outcome
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_outcome",
		Description: "Report whether a task succeeded after using a memory. Call this after completing a task that used a retrieved memory to help the system learn which memories are actually useful.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryOutcomeInput) (*mcp.CallToolResult, memoryOutcomeOutput, error) {
		// Record the outcome signal
		newConfidence, err := s.reasoningbankSvc.RecordOutcome(ctx, args.MemoryID, args.Succeeded, args.SessionID)
		if err != nil {
			return nil, memoryOutcomeOutput{}, fmt.Errorf("memory outcome failed: %w", err)
		}

		output := memoryOutcomeOutput{
			Recorded:      true,
			NewConfidence: newConfidence,
			Message:       "Outcome recorded",
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Outcome recorded, confidence: %.2f", output.NewConfidence)},
			},
		}, output, nil
	})
}

// ===== FOLDING TOOLS (Context-Folding) =====

type branchCreateInput struct {
	SessionID      string `json:"session_id" jsonschema:"required,Session identifier"`
	Description    string `json:"description" jsonschema:"required,Brief description of what the branch will do"`
	Prompt         string `json:"prompt,omitempty" jsonschema:"Detailed prompt/instructions for the branch"`
	Budget         int    `json:"budget,omitempty" jsonschema:"Token budget for this branch (default: 8192)"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"Timeout in seconds (default: 300)"`
}

type branchCreateOutput struct {
	BranchID        string `json:"branch_id" jsonschema:"Unique branch identifier"`
	BudgetAllocated int    `json:"budget_allocated" jsonschema:"Actual budget allocated"`
	Depth           int    `json:"depth" jsonschema:"Nesting depth of this branch"`
}

type branchReturnInput struct {
	BranchID string `json:"branch_id" jsonschema:"required,Branch ID to return from"`
	Message  string `json:"message" jsonschema:"Result message/summary from the branch"`
}

type branchReturnOutput struct {
	Success    bool   `json:"success" jsonschema:"Whether return succeeded"`
	TokensUsed int    `json:"tokens_used" jsonschema:"Tokens consumed by the branch"`
	Message    string `json:"message" jsonschema:"Scrubbed result message"`
}

type branchStatusInput struct {
	BranchID string `json:"branch_id,omitempty" jsonschema:"Specific branch ID to check"`
	SessionID string `json:"session_id,omitempty" jsonschema:"Session ID to get active branch for"`
}

type branchStatusOutput struct {
	BranchID       string `json:"branch_id,omitempty" jsonschema:"Branch ID"`
	SessionID      string `json:"session_id,omitempty" jsonschema:"Session ID"`
	Status         string `json:"status" jsonschema:"Branch status (active, completed, failed, timeout)"`
	Depth          int    `json:"depth" jsonschema:"Branch depth"`
	BudgetUsed     int    `json:"budget_used" jsonschema:"Tokens consumed"`
	BudgetTotal    int    `json:"budget_total" jsonschema:"Total budget allocated"`
	BudgetRemaining int   `json:"budget_remaining" jsonschema:"Remaining budget"`
}

func (s *Server) registerFoldingTools() {
	// Only register if folding service is configured
	if s.foldingSvc == nil {
		s.logger.Info("folding service not configured, skipping folding tools registration")
		return
	}

	// branch_create - Create a new context branch
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "branch_create",
		Description: "Create a new context-folding branch. Branches allow isolated sub-tasks with their own token budget, automatically cleaned up on return. Use for complex multi-step operations that need context isolation.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args branchCreateInput) (*mcp.CallToolResult, branchCreateOutput, error) {
		branchReq := folding.BranchRequest{
			SessionID:      args.SessionID,
			Description:    args.Description,
			Prompt:         args.Prompt,
			Budget:         args.Budget,
			TimeoutSeconds: args.TimeoutSeconds,
		}

		resp, err := s.foldingSvc.Create(ctx, branchReq)
		if err != nil {
			return nil, branchCreateOutput{}, fmt.Errorf("branch create failed: %w", err)
		}

		output := branchCreateOutput{
			BranchID:        resp.BranchID,
			BudgetAllocated: resp.BudgetAllocated,
			Depth:           resp.Depth,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Branch created: %s (depth: %d, budget: %d tokens)", output.BranchID, output.Depth, output.BudgetAllocated)},
			},
		}, output, nil
	})

	// branch_return - Return from a branch with results
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "branch_return",
		Description: "Return from a context-folding branch with results. The message will be scrubbed for secrets before being returned to the parent context. Any child branches will be force-returned first.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args branchReturnInput) (*mcp.CallToolResult, branchReturnOutput, error) {
		returnReq := folding.ReturnRequest{
			BranchID: args.BranchID,
			Message:  args.Message,
		}

		resp, err := s.foldingSvc.Return(ctx, returnReq)
		if err != nil {
			return nil, branchReturnOutput{}, fmt.Errorf("branch return failed: %w", err)
		}

		output := branchReturnOutput{
			Success:    resp.Success,
			TokensUsed: resp.TokensUsed,
			Message:    resp.ScrubbedMsg,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Branch returned successfully (tokens used: %d)", output.TokensUsed)},
			},
		}, output, nil
	})

	// branch_status - Get branch status
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "branch_status",
		Description: "Get the status of a specific branch or the active branch for a session. Returns branch state, budget usage, and depth information.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args branchStatusInput) (*mcp.CallToolResult, branchStatusOutput, error) {
		var branch *folding.Branch
		var err error

		if args.BranchID != "" {
			branch, err = s.foldingSvc.Get(ctx, args.BranchID)
		} else if args.SessionID != "" {
			branch, err = s.foldingSvc.GetActive(ctx, args.SessionID)
		} else {
			return nil, branchStatusOutput{}, fmt.Errorf("either branch_id or session_id is required")
		}

		if err != nil {
			return nil, branchStatusOutput{}, fmt.Errorf("branch status failed: %w", err)
		}

		if branch == nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "No active branch found"},
				},
			}, branchStatusOutput{Status: "none"}, nil
		}

		output := branchStatusOutput{
			BranchID:        branch.ID,
			SessionID:       branch.SessionID,
			Status:          string(branch.Status),
			Depth:           branch.Depth,
			BudgetUsed:      branch.BudgetUsed,
			BudgetTotal:     branch.BudgetTotal,
			BudgetRemaining: branch.BudgetTotal - branch.BudgetUsed,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Branch %s: status=%s, depth=%d, budget=%d/%d", output.BranchID, output.Status, output.Depth, output.BudgetUsed, output.BudgetTotal)},
			},
		}, output, nil
	})
}
