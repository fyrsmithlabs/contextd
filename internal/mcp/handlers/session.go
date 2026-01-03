// internal/mcp/handlers/session.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/services"
)

// SessionStartInput is the input for session_start tool.
type SessionStartInput struct {
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
}

// SessionStartOutput is the output for session_start tool.
type SessionStartOutput struct {
	Checkpoint *CheckpointSummary `json:"checkpoint,omitempty"`
	Memories   []MemorySummary    `json:"memories"`
	Resumed    bool               `json:"resumed"`
}

// CheckpointSummary is a brief checkpoint description.
type CheckpointSummary struct {
	ID        string `json:"id"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

// MemorySummary is a brief memory description.
type MemorySummary struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Confidence float64 `json:"confidence"`
}

// SessionEndInput is the input for session_end tool.
type SessionEndInput struct {
	ProjectID string   `json:"project_id"`
	SessionID string   `json:"session_id"`
	Task      string   `json:"task"`
	Approach  string   `json:"approach"`
	Outcome   string   `json:"outcome"` // success, failure, partial
	Tags      []string `json:"tags"`
	Notes     string   `json:"notes,omitempty"`
}

// SessionEndOutput is the output for session_end tool.
type SessionEndOutput struct {
	MemoriesCreated int    `json:"memories_created"`
	Message         string `json:"message"`
}

// ContextThresholdInput is the input for context_threshold tool.
type ContextThresholdInput struct {
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
	Percent   int    `json:"percent"`
}

// ContextThresholdOutput is the output for context_threshold tool.
type ContextThresholdOutput struct {
	CheckpointID string `json:"checkpoint_id"`
	Message      string `json:"message"`
}

// SessionHandler handles session lifecycle tools.
type SessionHandler struct {
	registry services.Registry
}

// NewSessionHandler creates a new session handler.
func NewSessionHandler(registry services.Registry) *SessionHandler {
	return &SessionHandler{registry: registry}
}

// Start handles the session_start tool.
// It checks for recent checkpoints and primes with relevant memories.
func (h *SessionHandler) Start(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req SessionStartInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	output := &SessionStartOutput{
		Memories: make([]MemorySummary, 0),
	}

	// Execute session start hook
	if h.registry.Hooks() != nil {
		_ = h.registry.Hooks().Execute(ctx, hooks.HookSessionStart, map[string]interface{}{
			"project_id": req.ProjectID,
			"session_id": req.SessionID,
		})
	}

	// Check for recent checkpoint
	if h.registry.Checkpoint() != nil {
		checkpoints, err := h.registry.Checkpoint().List(ctx, &checkpoint.ListRequest{
			TenantID: req.ProjectID,
			Limit:    1,
		})
		if err == nil && len(checkpoints) > 0 {
			cp := checkpoints[0]
			output.Checkpoint = &CheckpointSummary{
				ID:        cp.ID,
				Summary:   cp.Summary,
				CreatedAt: cp.CreatedAt.Format("2006-01-02 15:04"),
			}
		}
	}

	// Prime with relevant memories
	if h.registry.Memory() != nil {
		memories, err := h.registry.Memory().Search(ctx, req.ProjectID, "recent work context", 3)
		if err == nil {
			for _, m := range memories {
				output.Memories = append(output.Memories, MemorySummary{
					ID:         m.ID,
					Title:      m.Title,
					Confidence: m.Confidence,
				})
			}
		}
	}

	return output, nil
}

// End handles the session_end tool.
// It calls the Distiller to extract learnings and create memories.
func (h *SessionHandler) End(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req SessionEndInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate required fields
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if req.Task == "" {
		return nil, fmt.Errorf("task is required")
	}
	if req.Approach == "" {
		return nil, fmt.Errorf("approach is required")
	}
	if req.Outcome == "" {
		return nil, fmt.Errorf("outcome is required")
	}
	if req.Outcome != "success" && req.Outcome != "failure" && req.Outcome != "partial" {
		return nil, fmt.Errorf("outcome must be success, failure, or partial")
	}
	if len(req.Tags) == 0 {
		return nil, fmt.Errorf("tags is required (at least one tag)")
	}

	memoriesCreated := 0

	// Call Distiller if available
	if h.registry.Distiller() != nil {
		summary := reasoningbank.SessionSummary{
			SessionID: req.SessionID,
			ProjectID: req.ProjectID,
			Task:      req.Task,
			Approach:  req.Approach,
			Outcome:   reasoningbank.SessionOutcome(req.Outcome),
			Tags:      req.Tags,
		}

		if err := h.registry.Distiller().DistillSession(ctx, summary); err != nil {
			// Log but don't fail - distillation is best-effort
			// In production, we'd log this error
		} else {
			memoriesCreated = 1 // Distiller creates at least one memory
		}
	}

	// Execute session end hook
	if h.registry.Hooks() != nil {
		_ = h.registry.Hooks().Execute(ctx, hooks.HookSessionEnd, map[string]interface{}{
			"project_id": req.ProjectID,
			"session_id": req.SessionID,
			"outcome":    req.Outcome,
		})
	}

	return &SessionEndOutput{
		MemoriesCreated: memoriesCreated,
		Message:         fmt.Sprintf("Session ended. Outcome: %s. Learnings extracted.", req.Outcome),
	}, nil
}

// ContextThreshold handles the context_threshold tool.
// It creates an auto-checkpoint when context usage is high.
func (h *SessionHandler) ContextThreshold(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req ContextThresholdInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate required fields
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if req.Percent < 0 || req.Percent > 100 {
		return nil, fmt.Errorf("percent must be between 0 and 100")
	}

	var checkpointID string

	// Create auto-checkpoint
	if h.registry.Checkpoint() != nil {
		cp, err := h.registry.Checkpoint().Save(ctx, &checkpoint.SaveRequest{
			TenantID:    req.ProjectID,
			SessionID:   req.SessionID,
			Summary:     fmt.Sprintf("Auto-checkpoint at %d%% context usage", req.Percent),
			AutoCreated: true,
		})
		if err == nil && cp != nil {
			checkpointID = cp.ID
		}
	}

	// Execute threshold hook
	if h.registry.Hooks() != nil {
		_ = h.registry.Hooks().Execute(ctx, hooks.HookContextThreshold, map[string]interface{}{
			"project_id": req.ProjectID,
			"session_id": req.SessionID,
			"percent":    req.Percent,
		})
	}

	return &ContextThresholdOutput{
		CheckpointID: checkpointID,
		Message:      fmt.Sprintf("Auto-checkpoint created at %d%% context usage", req.Percent),
	}, nil
}
