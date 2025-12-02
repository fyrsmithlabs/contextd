// internal/mcp/handlers/session.go
package handlers

import (
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
