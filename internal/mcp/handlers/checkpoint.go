package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
)

// CheckpointHandler wraps checkpoint service for MCP tool interface.
type CheckpointHandler struct {
	service checkpoint.Service
}

// NewCheckpointHandler creates a new checkpoint handler.
func NewCheckpointHandler(service checkpoint.Service) *CheckpointHandler {
	return &CheckpointHandler{
		service: service,
	}
}

// CheckpointSaveInput represents input for checkpoint_save tool.
type CheckpointSaveInput struct {
	SessionID   string            `json:"session_id"`
	TenantID    string            `json:"tenant_id"`
	TeamID      string            `json:"team_id"`
	ProjectID   string            `json:"project_id"`
	ProjectPath string            `json:"project_path"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Summary     string            `json:"summary"`
	Context     string            `json:"context"`
	FullState   string            `json:"full_state"`
	TokenCount  int32             `json:"token_count"`
	Threshold   float64           `json:"threshold"`
	AutoCreated bool              `json:"auto_created"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CheckpointListInput represents input for checkpoint_list tool.
type CheckpointListInput struct {
	SessionID   string `json:"session_id,omitempty"`
	TenantID    string `json:"tenant_id"`
	TeamID      string `json:"team_id"`
	ProjectID   string `json:"project_id"`
	ProjectPath string `json:"project_path,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	AutoOnly    bool   `json:"auto_only,omitempty"`
}

// CheckpointResumeInput represents input for checkpoint_resume tool.
type CheckpointResumeInput struct {
	CheckpointID string                 `json:"checkpoint_id"`
	TenantID     string                 `json:"tenant_id"`
	TeamID       string                 `json:"team_id"`
	ProjectID    string                 `json:"project_id"`
	Level        checkpoint.ResumeLevel `json:"level"`
}

// Save handles checkpoint_save MCP tool call.
func (h *CheckpointHandler) Save(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req CheckpointSaveInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Convert to service request
	saveReq := &checkpoint.SaveRequest{
		SessionID:   req.SessionID,
		TenantID:    req.TenantID,
		TeamID:      req.TeamID,
		ProjectID:   req.ProjectID,
		ProjectPath: req.ProjectPath,
		Name:        req.Name,
		Description: req.Description,
		Summary:     req.Summary,
		Context:     req.Context,
		FullState:   req.FullState,
		TokenCount:  req.TokenCount,
		Threshold:   req.Threshold,
		AutoCreated: req.AutoCreated,
		Metadata:    req.Metadata,
	}

	cp, err := h.service.Save(ctx, saveReq)
	if err != nil {
		return nil, fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return map[string]interface{}{
		"id":           cp.ID,
		"session_id":   cp.SessionID,
		"summary":      cp.Summary,
		"token_count":  cp.TokenCount,
		"auto_created": cp.AutoCreated,
		"created_at":   cp.CreatedAt,
	}, nil
}

// List handles checkpoint_list MCP tool call.
func (h *CheckpointHandler) List(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req CheckpointListInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Convert to service request
	listReq := &checkpoint.ListRequest{
		SessionID:   req.SessionID,
		TenantID:    req.TenantID,
		TeamID:      req.TeamID,
		ProjectID:   req.ProjectID,
		ProjectPath: req.ProjectPath,
		Limit:       req.Limit,
		AutoOnly:    req.AutoOnly,
	}

	checkpoints, err := h.service.List(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	// Convert to MCP response format
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

	return map[string]interface{}{
		"checkpoints": results,
		"count":       len(results),
	}, nil
}

// Resume handles checkpoint_resume MCP tool call.
func (h *CheckpointHandler) Resume(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req CheckpointResumeInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Convert to service request
	resumeReq := &checkpoint.ResumeRequest{
		CheckpointID: req.CheckpointID,
		TenantID:     req.TenantID,
		TeamID:       req.TeamID,
		ProjectID:    req.ProjectID,
		Level:        req.Level,
	}

	response, err := h.service.Resume(ctx, resumeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to resume checkpoint: %w", err)
	}

	return map[string]interface{}{
		"checkpoint_id": response.Checkpoint.ID,
		"session_id":    response.Checkpoint.SessionID,
		"content":       response.Content,
		"token_count":   response.TokenCount,
		"level":         req.Level,
	}, nil
}
