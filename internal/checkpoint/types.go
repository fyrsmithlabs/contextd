package checkpoint

import (
	"time"
)

// ResumeLevel indicates how much context to restore.
type ResumeLevel string

const (
	// ResumeSummary restores only the summary (minimal context).
	ResumeSummary ResumeLevel = "summary"
	// ResumeContext restores summary + relevant context.
	ResumeContext ResumeLevel = "context"
	// ResumeFull restores the complete checkpoint.
	ResumeFull ResumeLevel = "full"
)

// Checkpoint represents a saved session state.
type Checkpoint struct {
	// ID is the unique identifier for this checkpoint.
	ID string `json:"id"`

	// SessionID is the session this checkpoint belongs to.
	SessionID string `json:"session_id"`

	// TenantID is the organization this checkpoint belongs to.
	TenantID string `json:"tenant_id"`

	// ProjectPath is the project context for this checkpoint.
	ProjectPath string `json:"project_path"`

	// Name is a human-readable name for the checkpoint.
	Name string `json:"name"`

	// Description describes what was happening at checkpoint time.
	Description string `json:"description"`

	// Summary is a condensed summary of the session state.
	Summary string `json:"summary"`

	// Context contains relevant context fragments.
	Context string `json:"context"`

	// FullState contains the complete session state.
	FullState string `json:"full_state"`

	// TokenCount is the approximate token count of full state.
	TokenCount int32 `json:"token_count"`

	// Threshold is the context percentage at which this was created.
	Threshold float64 `json:"threshold"`

	// AutoCreated indicates if this was auto-generated.
	AutoCreated bool `json:"auto_created"`

	// Metadata contains additional checkpoint metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// CreatedAt is when this checkpoint was created.
	CreatedAt time.Time `json:"created_at"`
}

// SaveRequest represents parameters for saving a checkpoint.
type SaveRequest struct {
	SessionID   string
	TenantID    string
	ProjectPath string
	Name        string
	Description string
	Summary     string
	Context     string
	FullState   string
	TokenCount  int32
	Threshold   float64
	AutoCreated bool
	Metadata    map[string]string
}

// ListRequest represents parameters for listing checkpoints.
type ListRequest struct {
	SessionID   string
	TenantID    string
	ProjectPath string
	Limit       int
	AutoOnly    bool // Only return auto-created checkpoints
}

// ResumeRequest represents parameters for resuming from a checkpoint.
type ResumeRequest struct {
	CheckpointID string
	TenantID     string
	Level        ResumeLevel
}

// ResumeResponse contains the restored checkpoint data.
type ResumeResponse struct {
	Checkpoint *Checkpoint
	Content    string // Content based on resume level
	TokenCount int32
}
