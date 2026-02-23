package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/remediation"
)

// RemediationHandler wraps remediation service for MCP tool interface.
type RemediationHandler struct {
	service remediation.Service
}

// NewRemediationHandler creates a new remediation handler.
func NewRemediationHandler(service remediation.Service) *RemediationHandler {
	return &RemediationHandler{
		service: service,
	}
}

// RemediationSearchInput represents input for remediation_search tool.
type RemediationSearchInput struct {
	Query            string                    `json:"query"`
	Limit            int                       `json:"limit,omitempty"`
	MinConfidence    float64                   `json:"min_confidence,omitempty"`
	Category         remediation.ErrorCategory `json:"category,omitempty"`
	Scope            remediation.Scope         `json:"scope,omitempty"`
	TenantID         string                    `json:"tenant_id"`
	TeamID           string                    `json:"team_id,omitempty"`
	ProjectPath      string                    `json:"project_path,omitempty"`
	Tags             []string                  `json:"tags,omitempty"`
	IncludeHierarchy bool                      `json:"include_hierarchy,omitempty"`
}

// RemediationRecordInput represents input for remediation_record tool.
type RemediationRecordInput struct {
	Title         string                    `json:"title"`
	Problem       string                    `json:"problem"`
	Symptoms      []string                  `json:"symptoms,omitempty"`
	RootCause     string                    `json:"root_cause"`
	Solution      string                    `json:"solution"`
	CodeDiff      string                    `json:"code_diff,omitempty"`
	AffectedFiles []string                  `json:"affected_files,omitempty"`
	Category      remediation.ErrorCategory `json:"category"`
	Tags          []string                  `json:"tags,omitempty"`
	Scope         remediation.Scope         `json:"scope"`
	TenantID      string                    `json:"tenant_id"`
	TeamID        string                    `json:"team_id,omitempty"`
	ProjectPath   string                    `json:"project_path,omitempty"`
	SessionID     string                    `json:"session_id,omitempty"`
	Confidence    float64                   `json:"confidence,omitempty"`
}

// Search handles remediation_search MCP tool call.
func (h *RemediationHandler) Search(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req RemediationSearchInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Convert to service request
	searchReq := &remediation.SearchRequest{
		Query:            req.Query,
		Limit:            req.Limit,
		MinConfidence:    req.MinConfidence,
		Category:         req.Category,
		Scope:            req.Scope,
		TenantID:         req.TenantID,
		TeamID:           req.TeamID,
		ProjectPath:      req.ProjectPath,
		Tags:             req.Tags,
		IncludeHierarchy: req.IncludeHierarchy,
	}

	remediations, err := h.service.Search(ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search remediations: %w", err)
	}

	// Convert to MCP response format
	results := make([]map[string]interface{}, 0, len(remediations))
	for _, rem := range remediations {
		results = append(results, map[string]interface{}{
			"id":             rem.ID,
			"title":          rem.Title,
			"problem":        rem.Problem,
			"symptoms":       rem.Symptoms,
			"root_cause":     rem.RootCause,
			"solution":       rem.Solution,
			"code_diff":      rem.CodeDiff,
			"affected_files": rem.AffectedFiles,
			"category":       rem.Category,
			"confidence":     rem.Confidence,
			"score":          rem.Score,
			"usage_count":    rem.UsageCount,
			"tags":           rem.Tags,
			"scope":          rem.Scope,
			"created_at":     rem.CreatedAt,
			"updated_at":     rem.UpdatedAt,
		})
	}

	return map[string]interface{}{
		"remediations": results,
		"count":        len(results),
	}, nil
}

// Record handles remediation_record MCP tool call.
func (h *RemediationHandler) Record(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req RemediationRecordInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Convert to service request
	recordReq := &remediation.RecordRequest{
		Title:         req.Title,
		Problem:       req.Problem,
		Symptoms:      req.Symptoms,
		RootCause:     req.RootCause,
		Solution:      req.Solution,
		CodeDiff:      req.CodeDiff,
		AffectedFiles: req.AffectedFiles,
		Category:      req.Category,
		Tags:          req.Tags,
		Scope:         req.Scope,
		TenantID:      req.TenantID,
		TeamID:        req.TeamID,
		ProjectPath:   req.ProjectPath,
		SessionID:     req.SessionID,
		Confidence:    req.Confidence,
	}

	rem, err := h.service.Record(ctx, recordReq)
	if err != nil {
		return nil, fmt.Errorf("failed to record remediation: %w", err)
	}

	return map[string]interface{}{
		"id":         rem.ID,
		"title":      rem.Title,
		"confidence": rem.Confidence,
		"scope":      rem.Scope,
		"category":   rem.Category,
		"created_at": rem.CreatedAt,
	}, nil
}
