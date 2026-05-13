package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/folding"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/sanitize"
	"github.com/fyrsmithlabs/contextd/internal/tenant"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// withTenantContext adds tenant context to the Go context for vectorstore operations.
// This is required for payload-based tenant isolation to work correctly.
// Returns an error if any ID fails validation (fail-closed security).
func withTenantContext(ctx context.Context, tenantID, teamID, projectID string) (context.Context, error) {
	// Validate tenant ID (required)
	if err := sanitize.ValidateTenantID(tenantID); err != nil {
		return ctx, fmt.Errorf("invalid tenant_id: %w", err)
	}

	// Validate team ID (optional, but must be valid if provided)
	if err := sanitize.ValidateTeamID(teamID); err != nil {
		return ctx, fmt.Errorf("invalid team_id: %w", err)
	}

	// Validate project ID (optional, but must be valid if provided)
	if err := sanitize.ValidateProjectID(projectID); err != nil {
		return ctx, fmt.Errorf("invalid project_id: %w", err)
	}

	return vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  tenantID,
		TeamID:    teamID,
		ProjectID: projectID,
	}), nil
}

// deriveProjectID safely extracts a project ID from a path.
// This is a secure replacement for filepath.Base() on untrusted input.
// Returns empty string and error if path is invalid, or sanitized ID on success.
func deriveProjectID(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Use SafeBasename to get the base name securely
	baseName, err := sanitize.SafeBasename(path)
	if err != nil {
		return "", fmt.Errorf("invalid project path: %w", err)
	}

	// Sanitize the base name for use as a project ID
	projectID := sanitize.Identifier(baseName)

	// Validate the result
	if err := sanitize.ValidateProjectID(projectID); err != nil {
		return "", fmt.Errorf("derived project_id invalid: %w", err)
	}

	return projectID, nil
}

// resolvedTenant carries the validated, derived IDs that a handler typically
// also passes to the underlying service request.
type resolvedTenant struct {
	ValidPath string
	TenantID  string
	TeamID    string
	ProjectID string
}

// tenantCtx is the canonical helper for resolving tenant context inside MCP
// tool handlers. It folds the previous validateAndDeriveProjectPath +
// withTenantContext pair into a single call:
//
//  1. Validates projectPath (when provided) via sanitize.ValidateProjectPath.
//  2. Derives tenantID from the path (git remote → git user → $USER → "local")
//     when no explicit value is supplied.
//  3. Derives projectID from the path basename when no explicit value is
//     supplied. When both projectPath and an explicit projectID are empty
//     the resulting context omits projectID - this is appropriate for
//     team/org-scoped operations but means the caller is opting out of the
//     project isolation floor.
//  4. Validates and sanitizes all three identifiers (CWE-287 hardening).
//  5. Returns a context carrying the resulting TenantInfo for vectorstore ops.
//
// Callers pass empty strings for IDs they want auto-derived.
//
// The returned context is always safe to use; on error the original ctx is
// returned unchanged so deferred metrics can still pull request context. The
// resolvedTenant carries the validated IDs that the caller typically also
// threads into service request structs.
func (s *Server) tenantCtx(ctx context.Context, projectPath, tenantID, teamID, projectID string) (context.Context, resolvedTenant, error) {
	resolved := resolvedTenant{}

	// Validate path if provided.
	if projectPath != "" {
		var err error
		resolved.ValidPath, err = sanitize.ValidateProjectPath(projectPath)
		if err != nil {
			return ctx, resolved, fmt.Errorf("invalid project_path: %w", err)
		}
	}

	// Derive tenantID from path if not explicit. Falls back through git
	// remote → git user → $USER → "local" so solo devs never see ErrMissingTenant.
	if tenantID == "" {
		if resolved.ValidPath != "" {
			tenantID = tenant.GetTenantIDForPath(resolved.ValidPath)
		} else {
			tenantID = tenant.GetDefaultTenantID()
		}
	}
	if tenantID == "" {
		return ctx, resolved, fmt.Errorf("tenant_id could not be derived; pass tenant_id explicitly or run from within a git repository")
	}
	if err := sanitize.ValidateTenantID(tenantID); err != nil {
		return ctx, resolved, fmt.Errorf("invalid tenant_id: %w", err)
	}
	resolved.TenantID = tenantID

	// Derive projectID from path if not explicit. Empty projectID is a valid
	// choice for team/org-scoped operations; downstream isolation enforces the
	// project floor only when no TenantInfo is on the context at all.
	if projectID == "" && resolved.ValidPath != "" {
		derived, err := deriveProjectID(resolved.ValidPath)
		if err != nil {
			return ctx, resolved, err
		}
		projectID = derived
	}
	if projectID != "" {
		if err := sanitize.ValidateProjectID(projectID); err != nil {
			return ctx, resolved, fmt.Errorf("invalid project_id: %w", err)
		}
	}
	resolved.ProjectID = projectID

	// Team is optional but must validate when present.
	if err := sanitize.ValidateTeamID(teamID); err != nil {
		return ctx, resolved, fmt.Errorf("invalid team_id: %w", err)
	}
	resolved.TeamID = teamID

	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  resolved.TenantID,
		TeamID:    resolved.TeamID,
		ProjectID: resolved.ProjectID,
	})
	return ctx, resolved, nil
}

// validateAndDeriveProjectPath validates a project path and derives tenant context info.
// Returns the validated path, tenant ID, and project ID, or an error.
//
// Deprecated: Prefer tenantCtx for new handlers - it folds path validation,
// derivation, sanitisation, and context wrapping into one call. Existing
// callers below are tracked for migration in a follow-up PR.
func (s *Server) validateAndDeriveProjectPath(path, explicitTenantID string) (validPath, tenantID, projectID string, err error) {
	// Validate project path if provided
	if path != "" {
		validPath, err = sanitize.ValidateProjectPath(path)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid project_path: %w", err)
		}
	}

	// Derive tenant_id from validated path if not explicitly provided
	tenantID = explicitTenantID
	if tenantID == "" && validPath != "" {
		tenantID = tenant.GetTenantIDForPath(validPath)
	}

	// Validate tenant_id was derived successfully
	if tenantID == "" {
		return "", "", "", fmt.Errorf("tenant_id is required for data isolation. It is usually auto-detected from your git repository. Try running from within a git repository, or set tenant_id explicitly")
	}

	// Validate derived tenant_id format (CWE-287: prevent malformed tenant IDs)
	if err := sanitize.ValidateTenantID(tenantID); err != nil {
		return "", "", "", fmt.Errorf("derived tenant_id invalid: %w", err)
	}

	// Derive project_id from validated path
	if validPath != "" {
		projectID, err = deriveProjectID(validPath)
		if err != nil {
			return "", "", "", err
		}
	}

	return validPath, tenantID, projectID, nil
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

	// Conversation tools (conversation indexing and search)
	s.registerConversationTools()

	// Reflection tools (pattern analysis and reporting)
	s.registerReflectionTools()

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

// checkpointListRow is a typed row for checkpoint_list output. Field names
// mirror checkpoint.Checkpoint so the SDK can derive a stable OutputSchema.
type checkpointListRow struct {
	ID          string    `json:"id" jsonschema:"Checkpoint ID"`
	SessionID   string    `json:"session_id" jsonschema:"Session ID"`
	Name        string    `json:"name" jsonschema:"Checkpoint name"`
	Description string    `json:"description" jsonschema:"Scrubbed human-readable description"`
	Summary     string    `json:"summary" jsonschema:"Scrubbed brief summary"`
	TokenCount  int32     `json:"token_count" jsonschema:"Token count estimate"`
	Threshold   float64   `json:"threshold" jsonschema:"Context threshold that triggered the checkpoint"`
	AutoCreated bool      `json:"auto_created" jsonschema:"True if auto-created by system"`
	CreatedAt   time.Time `json:"created_at" jsonschema:"Creation timestamp"`
}

type checkpointListOutput struct {
	Checkpoints []checkpointListRow `json:"checkpoints" jsonschema:"List of checkpoints"`
	Count       int                 `json:"count" jsonschema:"Number of checkpoints returned"`
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
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptrFalse(),
			IdempotentHint:  false,
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args checkpointSaveInput) (*mcp.CallToolResult, checkpointSaveOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "checkpoint_save", &toolErr)()

		// Resolve tenant context (validates path, derives tenant/project, sets ctx).
		ctx, rt, err := s.tenantCtx(ctx, args.ProjectPath, args.TenantID, "", "")
		if err != nil {
			toolErr = err
			return nil, checkpointSaveOutput{}, err
		}

		saveReq := &checkpoint.SaveRequest{
			SessionID:   args.SessionID,
			TenantID:    rt.TenantID,
			TeamID:      "", // Empty team is allowed
			ProjectID:   rt.ProjectID,
			ProjectPath: rt.ValidPath,
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
			toolErr = fmt.Errorf("checkpoint save failed: %w", err)
			return nil, checkpointSaveOutput{}, toolErr
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
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args checkpointListInput) (*mcp.CallToolResult, checkpointListOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "checkpoint_list", &toolErr)()

		// Resolve tenant context (validates path, derives tenant/project, sets ctx).
		ctx, rt, err := s.tenantCtx(ctx, args.ProjectPath, args.TenantID, "", "")
		if err != nil {
			toolErr = err
			return nil, checkpointListOutput{}, err
		}

		listReq := &checkpoint.ListRequest{
			SessionID:   args.SessionID,
			TenantID:    rt.TenantID,
			TeamID:      "", // Empty team is allowed
			ProjectID:   rt.ProjectID,
			ProjectPath: rt.ValidPath,
			Limit:       args.Limit,
			AutoOnly:    args.AutoOnly,
		}

		checkpoints, err := s.checkpointSvc.List(ctx, listReq)
		if err != nil {
			toolErr = fmt.Errorf("checkpoint list failed: %w", err)
			return nil, checkpointListOutput{}, toolErr
		}

		results := make([]checkpointListRow, 0, len(checkpoints))
		for _, cp := range checkpoints {
			// Scrub text fields uniformly (consistent with checkpoint_save/resume)
			scrubbedSummary := s.scrubber.Scrub(cp.Summary).Scrubbed
			scrubbedDesc := s.scrubber.Scrub(cp.Description).Scrubbed

			results = append(results, checkpointListRow{
				ID:          cp.ID,
				SessionID:   cp.SessionID,
				Name:        cp.Name,
				Description: scrubbedDesc,
				Summary:     scrubbedSummary,
				TokenCount:  cp.TokenCount,
				Threshold:   cp.Threshold,
				AutoCreated: cp.AutoCreated,
				CreatedAt:   cp.CreatedAt,
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
		var toolErr error
		defer s.startMetrics(ctx, "checkpoint_resume", &toolErr)()

		// Validate tenant_id
		if err := sanitize.ValidateTenantID(args.TenantID); err != nil {
			toolErr = fmt.Errorf("invalid tenant_id: %w", err)
			return nil, checkpointResumeOutput{}, toolErr
		}

		resumeReq := &checkpoint.ResumeRequest{
			CheckpointID: args.CheckpointID,
			TenantID:     args.TenantID,
			Level:        args.Level,
		}

		// Add tenant context to Go context for vectorstore operations
		ctx, err := withTenantContext(ctx, args.TenantID, "", "")
		if err != nil {
			toolErr = err
			return nil, checkpointResumeOutput{}, err
		}

		response, err := s.checkpointSvc.Resume(ctx, resumeReq)
		if err != nil {
			toolErr = fmt.Errorf("checkpoint resume failed: %w", err)
			return nil, checkpointResumeOutput{}, toolErr
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

// remediationSearchRow is a typed row for remediation_search output. Fields
// mirror remediation.ScoredRemediation so the SDK can derive an OutputSchema.
type remediationSearchRow struct {
	ID         string  `json:"id" jsonschema:"Remediation ID"`
	Title      string  `json:"title" jsonschema:"Remediation title"`
	Problem    string  `json:"problem" jsonschema:"Scrubbed problem description"`
	RootCause  string  `json:"root_cause" jsonschema:"Scrubbed root cause analysis"`
	Solution   string  `json:"solution" jsonschema:"Scrubbed solution description"`
	Category   string  `json:"category" jsonschema:"Error category"`
	Confidence float64 `json:"confidence" jsonschema:"Current confidence score (0-1)"`
	Score      float64 `json:"score" jsonschema:"Relevance score (0-1)"`
	UsageCount int64   `json:"usage_count" jsonschema:"Times this remediation has been retrieved"`
}

type remediationSearchOutput struct {
	Remediations []remediationSearchRow `json:"remediations" jsonschema:"Matching remediations with scores"`
	Count        int                    `json:"count" jsonschema:"Number of results"`
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

type remediationFeedbackInput struct {
	RemediationID string `json:"remediation_id" jsonschema:"required,Remediation ID to provide feedback on"`
	Helpful       bool   `json:"helpful" jsonschema:"required,Whether the remediation was helpful (true) or not (false)"`
	TenantID      string `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived if not provided)"`
	ProjectPath   string `json:"project_path,omitempty" jsonschema:"Project path (used to auto-derive tenant_id if empty)"`
}

type remediationFeedbackOutput struct {
	RemediationID string  `json:"remediation_id" jsonschema:"Remediation ID that received feedback"`
	NewConfidence float64 `json:"new_confidence" jsonschema:"Updated confidence score after feedback"`
	Helpful       bool    `json:"helpful" jsonschema:"Feedback provided (helpful or not)"`
}

func (s *Server) registerRemediationTools() {
	// remediation_search
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "remediation_search",
		Description: "Search for remediations by error message or pattern",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args remediationSearchInput) (*mcp.CallToolResult, remediationSearchOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "remediation_search", &toolErr)()

		// Resolve tenant context. Remediation search is team/org-scoped, so we
		// deliberately pass projectID="" - the helper will keep projectID empty
		// on the resulting TenantInfo. Project floor isn't applicable for
		// cross-project queries.
		ctx, rt, err := s.tenantCtx(ctx, args.ProjectPath, args.TenantID, args.TeamID, "")
		if err != nil {
			toolErr = err
			return nil, remediationSearchOutput{}, err
		}

		searchReq := &remediation.SearchRequest{
			Query:            args.Query,
			TenantID:         rt.TenantID,
			Scope:            args.Scope,
			Category:         args.Category,
			MinConfidence:    args.MinConfidence,
			Limit:            args.Limit,
			TeamID:           rt.TeamID,
			ProjectPath:      rt.ValidPath,
			IncludeHierarchy: args.IncludeHierarchy,
		}

		results, err := s.remediationSvc.Search(ctx, searchReq)
		if err != nil {
			toolErr = fmt.Errorf("remediation search failed: %w", err)
			return nil, remediationSearchOutput{}, toolErr
		}

		remediations := make([]remediationSearchRow, 0, len(results))
		for _, r := range results {
			// Scrub free-form text fields (HANDLER-GUIDE §7.1).
			problem := r.Remediation.Problem
			rootCause := r.Remediation.RootCause
			solution := r.Remediation.Solution
			if s.scrubber != nil {
				problem = s.scrubber.Scrub(problem).Scrubbed
				rootCause = s.scrubber.Scrub(rootCause).Scrubbed
				solution = s.scrubber.Scrub(solution).Scrubbed
			}
			remediations = append(remediations, remediationSearchRow{
				ID:         r.Remediation.ID,
				Title:      r.Remediation.Title,
				Problem:    problem,
				RootCause:  rootCause,
				Solution:   solution,
				Category:   string(r.Remediation.Category),
				Confidence: r.Remediation.Confidence,
				Score:      r.Score,
				UsageCount: r.Remediation.UsageCount,
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
		var toolErr error
		defer s.startMetrics(ctx, "remediation_record", &toolErr)()

		// Validate and derive tenant context from project path
		validPath, tenantID, _, err := s.validateAndDeriveProjectPath(args.ProjectPath, args.TenantID)
		if err != nil {
			toolErr = err
			return nil, remediationRecordOutput{}, err
		}

		// Validate team_id if provided
		if err := sanitize.ValidateTeamID(args.TeamID); err != nil {
			toolErr = fmt.Errorf("invalid team_id: %w", err)
			return nil, remediationRecordOutput{}, toolErr
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
			ProjectPath:   validPath,
			SessionID:     args.SessionID,
		}

		// Add tenant context to Go context for vectorstore operations
		ctx, err = withTenantContext(ctx, tenantID, args.TeamID, "")
		if err != nil {
			toolErr = err
			return nil, remediationRecordOutput{}, err
		}

		rem, err := s.remediationSvc.Record(ctx, recordReq)
		if err != nil {
			toolErr = fmt.Errorf("remediation record failed: %w", err)
			return nil, remediationRecordOutput{}, toolErr
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

	// remediation_feedback
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "remediation_feedback",
		Description: "Provide feedback on whether a remediation was helpful. Updates confidence score based on real-world success/failure.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args remediationFeedbackInput) (*mcp.CallToolResult, remediationFeedbackOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "remediation_feedback", &toolErr)()

		// Validate input
		if args.RemediationID == "" {
			toolErr = fmt.Errorf("remediation_id is required")
			return nil, remediationFeedbackOutput{}, toolErr
		}

		// Auto-derive tenant_id from project_path if not provided
		tenantID := args.TenantID
		if tenantID == "" && args.ProjectPath != "" {
			tenantID = tenant.GetTenantIDForPath(args.ProjectPath)
		}

		// Validate tenant_id was derived successfully
		if tenantID == "" {
			toolErr = fmt.Errorf("tenant_id is required for data isolation. It is usually auto-detected from your git repository. Try running from within a git repository, or set tenant_id explicitly")
			return nil, remediationFeedbackOutput{}, toolErr
		}

		// Validate derived tenant_id format (CWE-287: prevent malformed tenant IDs)
		if err := sanitize.ValidateTenantID(tenantID); err != nil {
			toolErr = fmt.Errorf("invalid tenant_id: %w", err)
			return nil, remediationFeedbackOutput{}, toolErr
		}

		// Convert boolean helpful to Rating enum
		rating := remediation.RatingNotHelpful
		if args.Helpful {
			rating = remediation.RatingHelpful
		}

		// Call service method
		feedbackReq := &remediation.FeedbackRequest{
			RemediationID: args.RemediationID,
			TenantID:      tenantID,
			Rating:        rating,
		}

		if err := s.remediationSvc.Feedback(ctx, feedbackReq); err != nil {
			toolErr = fmt.Errorf("remediation feedback failed: %w", err)
			return nil, remediationFeedbackOutput{}, toolErr
		}

		// Get updated remediation to return new confidence (mirror memory_feedback pattern)
		rem, err := s.remediationSvc.Get(ctx, tenantID, args.RemediationID)
		if err != nil {
			// Fallback if we can't fetch the updated remediation
			return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Feedback recorded for remediation %s", args.RemediationID)},
					},
				}, remediationFeedbackOutput{
					RemediationID: args.RemediationID,
					Helpful:       args.Helpful,
					NewConfidence: 0.0, // Unknown since get failed
				}, nil
		}

		output := remediationFeedbackOutput{
			RemediationID: args.RemediationID,
			NewConfidence: rem.Confidence,
			Helpful:       args.Helpful,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Feedback recorded. New confidence: %.2f", output.NewConfidence)},
			},
		}, output, nil
	})
}

// ===== REPOSITORY TOOLS =====

type semanticSearchInput struct {
	Query          string `json:"query" jsonschema:"required,Search query (natural language or pattern)"`
	ProjectPath    string `json:"project_path,omitempty" jsonschema:"Project path to search within (required unless collection_name is provided)"`
	CollectionName string `json:"collection_name,omitempty" jsonschema:"Collection name from repository_index output. When set, bypasses project-path to collection derivation and disables grep fallback."`
	TenantID       string `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (defaults to git username)"`
	Branch         string `json:"branch,omitempty" jsonschema:"Filter by branch (empty = all branches)"`
	Limit          int    `json:"limit,omitempty" jsonschema:"Maximum results (default: 10)"`
	ContentMode    string `json:"content_mode,omitempty" jsonschema:"Content payload size when collection_name is set: minimal (default), preview, or full. Ignored without collection_name."`
}

type semanticSearchOutput struct {
	Results     []map[string]interface{} `json:"results" jsonschema:"Search results with file paths and content"`
	Count       int                      `json:"count" jsonschema:"Number of results returned"`
	Query       string                   `json:"query" jsonschema:"Original search query"`
	Source      string                   `json:"source" jsonschema:"Source of results (semantic or grep)"`
	Branch      string                   `json:"branch,omitempty" jsonschema:"Branch filter applied (if any)"`
	ContentMode string                   `json:"content_mode,omitempty" jsonschema:"Content mode used (only when collection_name is set)"`
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

func (s *Server) registerRepositoryTools() {
	// semantic_search
	//
	// Two modes:
	//   1. project_path mode (default): derives the collection from project_path
	//      and falls back to grep if semantic search returns nothing. Use when
	//      you don't already know the collection name (e.g. ad-hoc searches).
	//   2. collection_name mode: when collection_name is provided (typically
	//      from repository_index output), skips path-based collection lookup
	//      and the grep fallback. Adds content_mode (minimal/preview/full) to
	//      control payload size. Use after explicitly indexing a repo.
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "semantic_search",
		Description: "Semantic code search with optional grep fallback. Pass collection_name (from repository_index) for direct collection lookup with content_mode control, or pass project_path for derivation with grep fallback.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args semanticSearchInput) (*mcp.CallToolResult, semanticSearchOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "semantic_search", &toolErr)()

		// project_path is always required for tenant context derivation
		// (fail-closed security). collection_name alone is not enough.
		if args.ProjectPath == "" {
			toolErr = fmt.Errorf("project_path is required for tenant context")
			return nil, semanticSearchOutput{}, toolErr
		}

		// Validate and derive tenant context from project path
		validPath, tenantID, projectID, err := s.validateAndDeriveProjectPath(args.ProjectPath, args.TenantID)
		if err != nil {
			toolErr = err
			return nil, semanticSearchOutput{}, err
		}

		// Add tenant context to Go context for vectorstore operations
		ctx, err = withTenantContext(ctx, tenantID, "", projectID)
		if err != nil {
			toolErr = err
			return nil, semanticSearchOutput{}, err
		}

		// collection_name mode: bypass path-derived collection lookup and grep
		// fallback. Honor content_mode to keep payloads small.
		if args.CollectionName != "" {
			return s.semanticSearchInCollection(ctx, args, validPath, tenantID, &toolErr)
		}

		// project_path mode: derived collection + grep fallback.
		opts := repository.SearchOptions{
			ProjectPath: validPath,
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

			// Parse project's ignore files for grep using validated path
			var excludePatterns []string
			if parsed, parseErr := s.ignoreParser.ParseProject(validPath); parseErr != nil {
				s.logger.Warn("failed to parse ignore files for grep, using fallback",
					zap.String("path", validPath),
					zap.Error(parseErr))
				excludePatterns = s.ignoreParser.FallbackPatterns
			} else {
				excludePatterns = parsed
			}

			grepOpts := repository.GrepOptions{
				ProjectPath:     validPath,
				ExcludePatterns: excludePatterns,
				CaseSensitive:   false, // Default to case-insensitive for better fallback experience
			}

			grepResults, err := s.repositorySvc.Grep(ctx, args.Query, grepOpts)
			if err != nil {
				// If semantic failed AND grep failed, return error
				toolErr = fmt.Errorf("search failed: %w", err)
				return nil, semanticSearchOutput{}, toolErr
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

	// repository_index
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "repository_index",
		Description: "Index a repository for semantic code search",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args repositoryIndexInput) (*mcp.CallToolResult, repositoryIndexOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "repository_index", &toolErr)()

		// Path is required
		if args.Path == "" {
			toolErr = fmt.Errorf("path is required")
			return nil, repositoryIndexOutput{}, toolErr
		}

		// Validate project path and derive tenant context (CWE-22 path traversal protection)
		validPath, tenantID, projectID, err := s.validateAndDeriveProjectPath(args.Path, args.TenantID)
		if err != nil {
			toolErr = err
			return nil, repositoryIndexOutput{}, err
		}

		// Validate include patterns (CWE-20 input validation)
		includePatterns := args.IncludePatterns
		if len(includePatterns) == 0 {
			includePatterns = []string{"*"}
		} else {
			if err := sanitize.ValidateGlobPatterns(includePatterns); err != nil {
				toolErr = fmt.Errorf("invalid include_patterns: %w. Examples: *.go, **/*.ts, src/**/*.{js,ts}", err)
				return nil, repositoryIndexOutput{}, toolErr
			}
		}

		// Validate exclude patterns (CWE-20 input validation)
		excludePatterns := args.ExcludePatterns
		if len(excludePatterns) == 0 {
			// Parse project's ignore files (.gitignore, .dockerignore, etc.)
			// Falls back to config defaults if no ignore files found
			parsed, parseErr := s.ignoreParser.ParseProject(validPath)
			if parseErr != nil {
				s.logger.Warn("failed to parse ignore files, using fallback",
					zap.String("path", validPath),
					zap.Error(parseErr))
				excludePatterns = s.ignoreParser.FallbackPatterns
			} else {
				excludePatterns = parsed
			}
		} else {
			if err := sanitize.ValidateGlobPatterns(excludePatterns); err != nil {
				toolErr = fmt.Errorf("invalid exclude_patterns: %w. Examples: *.go, **/*.ts, src/**/*.{js,ts}", err)
				return nil, repositoryIndexOutput{}, toolErr
			}
		}

		// Ensure excludePatterns is never nil for output
		if excludePatterns == nil {
			excludePatterns = []string{}
		}

		opts := repository.IndexOptions{
			TenantID:        tenantID,
			Branch:          args.Branch,
			IncludePatterns: includePatterns,
			ExcludePatterns: excludePatterns,
			MaxFileSize:     args.MaxFileSize,
		}

		// Add tenant context to Go context for vectorstore operations
		ctx, err = withTenantContext(ctx, tenantID, "", projectID)
		if err != nil {
			toolErr = fmt.Errorf("failed to set tenant context: %w", err)
			return nil, repositoryIndexOutput{}, toolErr
		}

		result, err := s.repositorySvc.IndexRepository(ctx, validPath, opts)
		if err != nil {
			toolErr = fmt.Errorf("repository index failed: %w", err)
			return nil, repositoryIndexOutput{}, toolErr
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

// semanticSearchInCollection executes semantic_search when an explicit
// collection_name is provided. It routes directly to repositorySvc.Search via
// the CollectionName option (which uses SearchInCollection under the hood),
// skips the grep fallback, and honors content_mode for payload sizing.
func (s *Server) semanticSearchInCollection(ctx context.Context, args semanticSearchInput, validPath, tenantID string, toolErr *error) (*mcp.CallToolResult, semanticSearchOutput, error) {
	opts := repository.SearchOptions{
		CollectionName: args.CollectionName,
		ProjectPath:    validPath,
		TenantID:       tenantID,
		Branch:         args.Branch,
		Limit:          args.Limit,
	}

	results, err := s.repositorySvc.Search(ctx, args.Query, opts)
	if err != nil {
		*toolErr = fmt.Errorf("repository search failed: %w", err)
		return nil, semanticSearchOutput{}, *toolErr
	}

	// Content mode constants
	const (
		previewMaxRunes = 200
		previewEllipsis = "..."
	)

	// Determine content mode (default: minimal)
	contentMode := args.ContentMode
	if contentMode == "" {
		contentMode = "minimal"
	}

	// Validate content_mode enum value
	switch contentMode {
	case "minimal", "preview", "full":
		// Valid content mode
	default:
		*toolErr = fmt.Errorf("invalid content_mode: %q (must be 'minimal', 'preview', or 'full')", contentMode)
		return nil, semanticSearchOutput{}, *toolErr
	}

	// Convert to output format based on content mode
	outputResults := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		result := map[string]interface{}{
			"file_path": r.FilePath,
			"score":     r.Score,
			"branch":    r.Branch,
		}

		// Scrub content once before use (only if needed)
		var scrubbedContent string
		if contentMode == "full" || contentMode == "preview" {
			scrubbedContent = s.scrubber.Scrub(r.Content).Scrubbed
		}

		switch contentMode {
		case "full":
			// Full mode: include complete content and metadata
			result["content"] = scrubbedContent
			result["metadata"] = r.Metadata
		case "preview":
			// Preview mode: include first 200 characters (UTF-8 safe)
			preview := scrubbedContent
			runes := []rune(preview)
			if len(runes) > previewMaxRunes {
				preview = string(runes[:previewMaxRunes]) + previewEllipsis
			}
			result["content_preview"] = preview
		case "minimal":
			// Minimal mode: no content added - just file_path, score, branch
		}

		outputResults = append(outputResults, result)
	}

	output := semanticSearchOutput{
		Results:     outputResults,
		Count:       len(outputResults),
		Query:       args.Query,
		Source:      "semantic",
		Branch:      args.Branch,
		ContentMode: contentMode,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Found %d results for query: %s", output.Count, args.Query)},
		},
	}, output, nil
}

// ===== TROUBLESHOOT TOOLS =====

type troubleshootDiagnoseInput struct {
	ErrorMessage string `json:"error_message" jsonschema:"required,Error message to diagnose"`
	ErrorContext string `json:"error_context,omitempty" jsonschema:"Additional context (stack trace logs etc)"`
}

type troubleshootDiagnoseOutput struct {
	ErrorMessage    string                    `json:"error_message" jsonschema:"Original error message"`
	RootCause       string                    `json:"root_cause" jsonschema:"Likely root cause"`
	Hypotheses      []troubleshoot.Hypothesis `json:"hypotheses" jsonschema:"Diagnostic hypotheses"`
	Recommendations []string                  `json:"recommendations" jsonschema:"Recommended actions"`
	RelatedPatterns []troubleshoot.Pattern    `json:"related_patterns" jsonschema:"Similar known patterns"`
	Confidence      float64                   `json:"confidence" jsonschema:"Overall confidence (0-1)"`
}

func (s *Server) registerTroubleshootTools() {
	// troubleshoot_diagnose
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "troubleshoot_diagnose",
		Description: "Diagnose an error using AI and known patterns",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args troubleshootDiagnoseInput) (*mcp.CallToolResult, troubleshootDiagnoseOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "troubleshoot_diagnose", &toolErr)()

		diagnosis, err := s.troubleshootSvc.Diagnose(ctx, args.ErrorMessage, args.ErrorContext)
		if err != nil {
			toolErr = fmt.Errorf("troubleshoot diagnose failed: %w", err)
			return nil, troubleshootDiagnoseOutput{}, toolErr
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

// memorySearchRow is a typed result row for memory_search. Replaces the
// previous map[string]interface{} payload so the SDK can derive a proper
// output schema (HANDLER-GUIDE.md §4.4).
type memorySearchRow struct {
	ID         string   `json:"id" jsonschema:"Memory ID"`
	Title      string   `json:"title" jsonschema:"Memory title"`
	Content    string   `json:"content" jsonschema:"Scrubbed memory content"`
	Outcome    string   `json:"outcome" jsonschema:"Outcome type (success or failure)"`
	Confidence float64  `json:"confidence" jsonschema:"Memory confidence score (0-1)"`
	Relevance  float64  `json:"relevance" jsonschema:"Search relevance score (0-1)"`
	Tags       []string `json:"tags,omitempty" jsonschema:"Memory tags"`
}

// memorySearchMetadata mirrors reasoningbank.SearchMetadata for the wire.
type memorySearchMetadata struct {
	SuggestedRefinements []string `json:"suggested_refinements" jsonschema:"Recommended search terms from results"`
	QueryCoverage        float64  `json:"query_coverage" jsonschema:"How well results matched the query (0-1)"`
	EntityMatches        int      `json:"entity_matches" jsonschema:"Count of distinct entities in results"`
}

type memorySearchOutput struct {
	Memories []memorySearchRow    `json:"memories" jsonschema:"Matching memories"`
	Count    int                  `json:"count" jsonschema:"Number of results"`
	Metadata memorySearchMetadata `json:"metadata" jsonschema:"Search metadata for iterative refinement"`
}

type memoryRecordInput struct {
	ProjectID   string   `json:"project_id" jsonschema:"required,Project identifier"`
	Title       string   `json:"title" jsonschema:"required,Brief title for the memory"`
	Content     string   `json:"content" jsonschema:"required,The strategy or learning to remember"`
	Outcome     string   `json:"outcome" jsonschema:"required,Outcome type (success or failure)"`
	Tags        []string `json:"tags,omitempty" jsonschema:"Tags for categorization"`
	SessionID   string   `json:"session_id,omitempty" jsonschema:"Session ID for session-level buffering (when granularity=session)"`
	SessionDate string   `json:"session_date,omitempty" jsonschema:"Session date in RFC3339 format (optional, defaults to now)"`
}

type memoryRecordOutput struct {
	ID         string  `json:"id" jsonschema:"Memory ID"`
	Title      string  `json:"title" jsonschema:"Memory title"`
	Outcome    string  `json:"outcome" jsonschema:"Outcome type"`
	Confidence float64 `json:"confidence" jsonschema:"Initial confidence"`
}

type memoryFeedbackInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,Project identifier (tenant + project scope)"`
	MemoryID  string `json:"memory_id" jsonschema:"required,Memory ID to provide feedback on"`
	Helpful   bool   `json:"helpful" jsonschema:"required,Whether the memory was helpful"`
}

type memoryFeedbackOutput struct {
	MemoryID      string  `json:"memory_id" jsonschema:"Memory ID"`
	NewConfidence float64 `json:"new_confidence" jsonschema:"Updated confidence after feedback"`
	Helpful       bool    `json:"helpful" jsonschema:"Feedback provided"`
}

type memoryOutcomeInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,Project identifier (tenant + project scope)"`
	MemoryID  string `json:"memory_id" jsonschema:"required,ID of the memory that was used"`
	Succeeded bool   `json:"succeeded" jsonschema:"required,Whether the task succeeded after using this memory"`
	SessionID string `json:"session_id,omitempty" jsonschema:"Optional session ID for correlation"`
}

type memoryOutcomeOutput struct {
	Recorded      bool    `json:"recorded" jsonschema:"Whether outcome was recorded"`
	NewConfidence float64 `json:"new_confidence" jsonschema:"Updated confidence after outcome"`
	Message       string  `json:"message" jsonschema:"Result message"`
}

type memoryConsolidateInput struct {
	ProjectID           string  `json:"project_id" jsonschema:"required,Project identifier"`
	SimilarityThreshold float64 `json:"similarity_threshold,omitempty" jsonschema:"Minimum similarity score for consolidation (0-1 default 0.8)"`
	DryRun              bool    `json:"dry_run,omitempty" jsonschema:"Preview consolidation without making changes (default false)"`
	MaxClusters         int     `json:"max_clusters,omitempty" jsonschema:"Maximum number of clusters to consolidate in one run (0 = no limit)"`
}

type memoryConsolidateOutput struct {
	CreatedMemories  []string `json:"created_memories" jsonschema:"IDs of newly created consolidated memories"`
	ArchivedMemories []string `json:"archived_memories" jsonschema:"IDs of source memories that were archived"`
	SkippedCount     int      `json:"skipped_count" jsonschema:"Number of memories skipped (below threshold)"`
	TotalProcessed   int      `json:"total_processed" jsonschema:"Total number of memories examined"`
	DurationSeconds  float64  `json:"duration_seconds" jsonschema:"Time taken for consolidation operation"`
}

// memoryConsolidateSessionInput / memoryConsolidateSessionOutput replace
// the previous anonymous-struct args/return on the memory_consolidate_session
// handler (HANDLER-GUIDE.md §3.1).
type memoryConsolidateSessionInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,Project identifier"`
	SessionID string `json:"session_id" jsonschema:"required,Session ID to flush"`
}

type memoryConsolidateSessionOutput struct {
	MemoryIDs []string `json:"memory_ids" jsonschema:"IDs of memories created from the session"`
	Count     int      `json:"count" jsonschema:"Number of session memories created"`
	Message   string   `json:"message" jsonschema:"Human-readable summary"`
}

func (s *Server) registerMemoryTools() {
	// memory_search — pure read; ReadOnlyHint + closed-world.
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_search",
		Description: "Search for relevant memories from past sessions scoped to project_id.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memorySearchInput) (*mcp.CallToolResult, memorySearchOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "memory_search", &toolErr)()

		// For memory tools project_id serves as both tenant and project scope.
		// Pass it as the explicit tenantID *and* projectID; tenantCtx validates
		// both and sets the resulting TenantInfo on ctx for the vectorstore.
		ctx, rt, err := s.tenantCtx(ctx, "", args.ProjectID, "", args.ProjectID)
		if err != nil {
			toolErr = err
			return nil, memorySearchOutput{}, err
		}

		limit := args.Limit
		if limit <= 0 {
			limit = 5
		}
		if limit > 100 {
			limit = 100
		}

		scoredMemories, metadata, err := s.reasoningbankSvc.SearchWithMetadata(ctx, rt.ProjectID, args.Query, limit)
		if err != nil {
			toolErr = fmt.Errorf("memory search failed: %w", err)
			return nil, memorySearchOutput{}, toolErr
		}

		rows := make([]memorySearchRow, 0, len(scoredMemories))
		for _, sm := range scoredMemories {
			content := sm.Memory.Content
			if s.scrubber != nil {
				content = s.scrubber.Scrub(content).Scrubbed
			}
			rows = append(rows, memorySearchRow{
				ID:         sm.Memory.ID,
				Title:      sm.Memory.Title,
				Content:    content,
				Outcome:    string(sm.Memory.Outcome),
				Confidence: sm.Memory.Confidence,
				Relevance:  sm.Relevance,
				Tags:       sm.Memory.Tags,
			})
		}

		md := memorySearchMetadata{}
		if metadata != nil {
			md.SuggestedRefinements = metadata.SuggestedRefinements
			md.QueryCoverage = metadata.QueryCoverage
			md.EntityMatches = metadata.EntityMatches
		}

		output := memorySearchOutput{
			Memories: rows,
			Count:    len(rows),
			Metadata: md,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d relevant memories", output.Count)},
			},
		}, output, nil
	})

	// memory_record — append-only write; not destructive.
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_record",
		Description: "Record a new memory/learning from the current session.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptrFalse(),
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryRecordInput) (*mcp.CallToolResult, memoryRecordOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "memory_record", &toolErr)()

		ctx, rt, err := s.tenantCtx(ctx, "", args.ProjectID, "", args.ProjectID)
		if err != nil {
			toolErr = err
			return nil, memoryRecordOutput{}, err
		}

		outcome := reasoningbank.OutcomeSuccess
		if args.Outcome == "failure" {
			outcome = reasoningbank.OutcomeFailure
		}

		memory, err := reasoningbank.NewMemory(rt.ProjectID, args.Title, args.Content, outcome, args.Tags)
		if err != nil {
			toolErr = fmt.Errorf("invalid memory: %w", err)
			return nil, memoryRecordOutput{}, toolErr
		}

		// Set optional session fields for session-level buffering.
		if args.SessionID != "" {
			memory.SessionID = args.SessionID
			if args.SessionDate != "" {
				if sd, parseErr := time.Parse(time.RFC3339, args.SessionDate); parseErr == nil {
					memory.SessionDate = &sd
				}
			}
		}

		if err := s.reasoningbankSvc.Record(ctx, memory); err != nil {
			toolErr = fmt.Errorf("memory record failed: %w", err)
			return nil, memoryRecordOutput{}, toolErr
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

	// memory_feedback — mutating write (overwrites confidence).
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_feedback",
		Description: "Provide feedback on a memory to adjust its confidence score.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptrTrue(),
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryFeedbackInput) (*mcp.CallToolResult, memoryFeedbackOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "memory_feedback", &toolErr)()

		// project_id is required so the mutating write is tenant-scoped — this
		// closes the data-leak risk where Feedback previously ran on whatever
		// (or no) tenant context happened to be on ctx.
		ctx, rt, err := s.tenantCtx(ctx, "", args.ProjectID, "", args.ProjectID)
		if err != nil {
			toolErr = err
			return nil, memoryFeedbackOutput{}, err
		}
		_ = rt // tenant scope is enforced via ctx; service uses memory_id directly.

		if err := s.reasoningbankSvc.Feedback(ctx, args.MemoryID, args.Helpful); err != nil {
			toolErr = fmt.Errorf("memory feedback failed: %w", err)
			return nil, memoryFeedbackOutput{}, toolErr
		}

		memory, err := s.reasoningbankSvc.Get(ctx, args.MemoryID)
		if err != nil {
			toolErr = fmt.Errorf("failed to get updated memory: %w", err)
			return nil, memoryFeedbackOutput{}, toolErr
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

	// memory_outcome — mutating write (overwrites confidence).
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_outcome",
		Description: "Report whether a task succeeded after using a memory, updating its confidence.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptrTrue(),
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryOutcomeInput) (*mcp.CallToolResult, memoryOutcomeOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "memory_outcome", &toolErr)()

		ctx, _, err := s.tenantCtx(ctx, "", args.ProjectID, "", args.ProjectID)
		if err != nil {
			toolErr = err
			return nil, memoryOutcomeOutput{}, err
		}

		newConfidence, err := s.reasoningbankSvc.RecordOutcome(ctx, args.MemoryID, args.Succeeded, args.SessionID)
		if err != nil {
			toolErr = fmt.Errorf("memory outcome failed: %w", err)
			return nil, memoryOutcomeOutput{}, toolErr
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

	// memory_consolidate — mutating write (merges/archives existing memories).
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_consolidate",
		Description: "Consolidate similar memories above the similarity threshold into synthesized refined summaries.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptrTrue(),
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryConsolidateInput) (*mcp.CallToolResult, memoryConsolidateOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "memory_consolidate", &toolErr)()

		ctx, rt, err := s.tenantCtx(ctx, "", args.ProjectID, "", args.ProjectID)
		if err != nil {
			toolErr = err
			return nil, memoryConsolidateOutput{}, err
		}

		if s.distiller == nil {
			toolErr = fmt.Errorf("memory consolidation not available: distiller not configured")
			return nil, memoryConsolidateOutput{}, toolErr
		}

		threshold := args.SimilarityThreshold
		if threshold == 0 {
			threshold = 0.8 // Default per spec.
		}

		opts := reasoningbank.ConsolidationOptions{
			SimilarityThreshold: threshold,
			DryRun:              args.DryRun,
			MaxClustersPerRun:   args.MaxClusters,
		}

		result, err := s.distiller.Consolidate(ctx, rt.ProjectID, opts)
		if err != nil {
			toolErr = fmt.Errorf("consolidation failed: %w", err)
			return nil, memoryConsolidateOutput{}, toolErr
		}

		output := memoryConsolidateOutput{
			CreatedMemories:  result.CreatedMemories,
			ArchivedMemories: result.ArchivedMemories,
			SkippedCount:     result.SkippedCount,
			TotalProcessed:   result.TotalProcessed,
			DurationSeconds:  result.Duration.Seconds(),
		}

		resultMsg := fmt.Sprintf("Consolidation complete: created %d, archived %d, skipped %d, processed %d memories (%.2fs)",
			len(output.CreatedMemories),
			len(output.ArchivedMemories),
			output.SkippedCount,
			output.TotalProcessed,
			output.DurationSeconds)

		if args.DryRun {
			resultMsg = "[DRY RUN] " + resultMsg
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultMsg},
			},
		}, output, nil
	})

	// memory_consolidate_session — mutating write; named struct args.
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "memory_consolidate_session",
		Description: "Flush a session's buffered turns into session-level memories (granularity=session).",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptrTrue(),
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args memoryConsolidateSessionInput) (*mcp.CallToolResult, memoryConsolidateSessionOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "memory_consolidate_session", &toolErr)()

		ctx, rt, err := s.tenantCtx(ctx, "", args.ProjectID, "", args.ProjectID)
		if err != nil {
			toolErr = err
			return nil, memoryConsolidateSessionOutput{}, err
		}

		if args.SessionID == "" {
			toolErr = fmt.Errorf("session_id is required")
			return nil, memoryConsolidateSessionOutput{}, toolErr
		}

		ids, err := s.reasoningbankSvc.FlushSession(ctx, rt.ProjectID, args.SessionID)
		if err != nil {
			toolErr = fmt.Errorf("session flush failed: %w", err)
			return nil, memoryConsolidateSessionOutput{}, toolErr
		}

		if ids == nil {
			ids = []string{}
		}

		msg := fmt.Sprintf("Session %s flushed: %d memories created", args.SessionID, len(ids))
		out := memoryConsolidateSessionOutput{
			MemoryIDs: ids,
			Count:     len(ids),
			Message:   msg,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: msg},
			},
		}, out, nil
	})
}

// ===== FOLDING TOOLS (Context-Folding) =====

type branchCreateInput struct {
	SessionID      string `json:"session_id" jsonschema:"required,Session identifier"`
	ProjectID      string `json:"project_id,omitempty" jsonschema:"Project identifier for metrics tracking"`
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
	BranchID  string `json:"branch_id,omitempty" jsonschema:"Specific branch ID to check"`
	SessionID string `json:"session_id,omitempty" jsonschema:"Session ID to get active branch for"`
}

type branchStatusOutput struct {
	BranchID        string `json:"branch_id,omitempty" jsonschema:"Branch ID"`
	SessionID       string `json:"session_id,omitempty" jsonschema:"Session ID"`
	Status          string `json:"status" jsonschema:"Branch status (active, completed, failed, timeout)"`
	Depth           int    `json:"depth" jsonschema:"Branch depth"`
	BudgetUsed      int    `json:"budget_used" jsonschema:"Tokens consumed"`
	BudgetTotal     int    `json:"budget_total" jsonschema:"Total budget allocated"`
	BudgetRemaining int    `json:"budget_remaining" jsonschema:"Remaining budget"`
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
		Description: "Open an isolated context-folding branch with its own token budget for a complex sub-task; auto-cleaned on return.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptrFalse(),
			IdempotentHint:  false,
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args branchCreateInput) (*mcp.CallToolResult, branchCreateOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "branch_create", &toolErr)()

		// Validate project_id if provided (CWE-287 authentication bypass protection)
		if args.ProjectID != "" {
			if err := sanitize.ValidateProjectID(args.ProjectID); err != nil {
				toolErr = fmt.Errorf("invalid project_id: %w", err)
				return nil, branchCreateOutput{}, toolErr
			}
		}

		branchReq := folding.BranchRequest{
			SessionID:      args.SessionID,
			ProjectID:      args.ProjectID,
			Description:    args.Description,
			Prompt:         args.Prompt,
			Budget:         args.Budget,
			TimeoutSeconds: args.TimeoutSeconds,
		}

		resp, err := s.foldingSvc.Create(ctx, branchReq)
		if err != nil {
			toolErr = fmt.Errorf("branch create failed: %w", err)
			return nil, branchCreateOutput{}, toolErr
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
		Description: "Close a context-folding branch with a scrubbed result message; force-returns child branches first.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptrTrue(),
			IdempotentHint:  false,
			OpenWorldHint:   ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args branchReturnInput) (*mcp.CallToolResult, branchReturnOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "branch_return", &toolErr)()

		returnReq := folding.ReturnRequest{
			BranchID: args.BranchID,
			Message:  args.Message,
		}

		resp, err := s.foldingSvc.Return(ctx, returnReq)
		if err != nil {
			toolErr = fmt.Errorf("branch return failed: %w", err)
			return nil, branchReturnOutput{}, toolErr
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
		Description: "Report the status, depth, and budget usage of a specific branch or the active branch for a session.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: ptrFalse(),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args branchStatusInput) (*mcp.CallToolResult, branchStatusOutput, error) {
		var toolErr error
		defer s.startMetrics(ctx, "branch_status", &toolErr)()

		var branch *folding.Branch
		var err error

		if args.BranchID != "" {
			branch, err = s.foldingSvc.Get(ctx, args.BranchID)
		} else if args.SessionID != "" {
			branch, err = s.foldingSvc.GetActive(ctx, args.SessionID)
		} else {
			toolErr = fmt.Errorf("either branch_id or session_id is required")
			return nil, branchStatusOutput{}, toolErr
		}

		if err != nil {
			toolErr = fmt.Errorf("branch status failed: %w", err)
			return nil, branchStatusOutput{}, toolErr
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
