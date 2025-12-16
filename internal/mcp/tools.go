package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/tenant"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
)

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

	return nil
}

// ===== CHECKPOINT TOOLS =====

type checkpointSaveInput struct {
	SessionID   string            `json:"session_id" jsonschema:"required,Session identifier"`
	TenantID    string            `json:"tenant_id" jsonschema:"required,Tenant identifier"`
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
	TenantID    string `json:"tenant_id" jsonschema:"required,Tenant identifier"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Filter by project path"`
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
		saveReq := &checkpoint.SaveRequest{
			SessionID:   args.SessionID,
			TenantID:    args.TenantID,
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
		listReq := &checkpoint.ListRequest{
			SessionID:   args.SessionID,
			TenantID:    args.TenantID,
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
	Query          string                  `json:"query" jsonschema:"required,Error message or pattern to search for"`
	TenantID       string                  `json:"tenant_id" jsonschema:"required,Tenant identifier"`
	Scope          remediation.Scope       `json:"scope,omitempty" jsonschema:"Search scope (project team or org)"`
	Category       remediation.ErrorCategory `json:"category,omitempty" jsonschema:"Error category filter"`
	MinConfidence  float64                 `json:"min_confidence,omitempty" jsonschema:"Minimum confidence threshold (0-1)"`
	Limit          int                     `json:"limit,omitempty" jsonschema:"Maximum results (default: 10)"`
	TeamID         string                  `json:"team_id,omitempty" jsonschema:"Team ID for team/project scope"`
	ProjectPath    string                  `json:"project_path,omitempty" jsonschema:"Project path for project scope"`
	IncludeHierarchy bool                  `json:"include_hierarchy,omitempty" jsonschema:"Search parent scopes (project→team→org)"`
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
	TenantID      string                    `json:"tenant_id" jsonschema:"required,Tenant identifier"`
	Scope         remediation.Scope         `json:"scope" jsonschema:"required,Scope level (project team or org)"`
	TeamID        string                    `json:"team_id,omitempty" jsonschema:"Team ID (for team/project scope)"`
	ProjectPath   string                    `json:"project_path,omitempty" jsonschema:"Project path (for project scope)"`
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
		searchReq := &remediation.SearchRequest{
			Query:            args.Query,
			TenantID:         args.TenantID,
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
			TenantID:      args.TenantID,
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
	// repository_search
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "repository_search",
		Description: "Semantic search over indexed repository code in _codebase collection. Prefer using collection_name from repository_index output.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args repositorySearchInput) (*mcp.CallToolResult, repositorySearchOutput, error) {
		// Prefer collection_name if provided (avoids tenant_id derivation issues)
		// Otherwise derive from tenant_id + project_path
		opts := repository.SearchOptions{
			CollectionName: args.CollectionName,
			ProjectPath:    args.ProjectPath,
			Branch:         args.Branch,
			Limit:          args.Limit,
		}

		// Only derive tenant_id if collection_name not provided
		if args.CollectionName == "" {
			if args.ProjectPath == "" {
				return nil, repositorySearchOutput{}, fmt.Errorf("either collection_name or project_path is required")
			}
			tenantID := args.TenantID
			if tenantID == "" {
				tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
			}
			opts.TenantID = tenantID
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
