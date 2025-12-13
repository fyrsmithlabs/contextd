// Package folding implements context-folding for LLM agent context management.
// It provides branch() and return() MCP tools for isolated subtask execution.
package folding

import (
	"time"
)

// BranchStatus represents the lifecycle state of a branch.
type BranchStatus string

const (
	BranchStatusCreated   BranchStatus = "created"
	BranchStatusActive    BranchStatus = "active"
	BranchStatusCompleted BranchStatus = "completed"
	BranchStatusTimeout   BranchStatus = "timeout"
	BranchStatusFailed    BranchStatus = "failed"
)

// ValidTransitions defines allowed state transitions.
var ValidTransitions = map[BranchStatus][]BranchStatus{
	BranchStatusCreated:   {BranchStatusActive},
	BranchStatusActive:    {BranchStatusCompleted, BranchStatusTimeout, BranchStatusFailed},
	BranchStatusCompleted: {}, // terminal
	BranchStatusTimeout:   {}, // terminal
	BranchStatusFailed:    {}, // terminal
}

// CanTransitionTo checks if a transition from current status to target is valid.
func (s BranchStatus) CanTransitionTo(target BranchStatus) bool {
	allowed, ok := ValidTransitions[s]
	if !ok {
		return false
	}
	for _, t := range allowed {
		if t == target {
			return true
		}
	}
	return false
}

// IsTerminal returns true if this is a terminal state.
func (s BranchStatus) IsTerminal() bool {
	return s == BranchStatusCompleted || s == BranchStatusTimeout || s == BranchStatusFailed
}

// Branch represents an isolated context branch for subtask execution.
type Branch struct {
	ID                string       `json:"id"`
	SessionID         string       `json:"session_id"`
	ParentID          *string      `json:"parent_id,omitempty"`
	Depth             int          `json:"depth"`
	Description       string       `json:"description"`
	Prompt            string       `json:"prompt"`
	BudgetTotal       int          `json:"budget_total"`
	BudgetUsed        int          `json:"budget_used"`
	TimeoutSeconds    int          `json:"timeout_seconds"`
	Status            BranchStatus `json:"status"`
	Result            *string      `json:"result,omitempty"`
	Error             *string      `json:"error,omitempty"`
	InjectedMemoryIDs []string     `json:"injected_memory_ids,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	CompletedAt       *time.Time   `json:"completed_at,omitempty"`
}

// BudgetRemaining returns the remaining token budget.
func (b *Branch) BudgetRemaining() int {
	return b.BudgetTotal - b.BudgetUsed
}

// BranchRequest represents a request to create a new branch.
type BranchRequest struct {
	SessionID      string `json:"session_id"`
	Description    string `json:"description"`
	Prompt         string `json:"prompt"`
	Budget         int    `json:"budget,omitempty"`
	InjectMemories bool   `json:"inject_memories,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// Input validation constants (from SEC-001).
const (
	MaxDescriptionLength = 500
	MaxPromptLength      = 10000
	MaxReturnMsgLength   = 50000
	DefaultBudget        = 8192
	DefaultTimeout       = 300
)

// Validate checks the request against SEC-001 requirements.
func (r *BranchRequest) Validate() error {
	if r.SessionID == "" {
		return ErrEmptySessionID
	}
	if len(r.Description) == 0 {
		return ErrEmptyDescription
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescriptionTooLong
	}
	if len(r.Prompt) == 0 {
		return ErrEmptyPrompt
	}
	if len(r.Prompt) > MaxPromptLength {
		return ErrPromptTooLong
	}
	return nil
}

// ApplyDefaults sets default values for optional fields.
func (r *BranchRequest) ApplyDefaults() {
	if r.Budget <= 0 {
		r.Budget = DefaultBudget
	}
	if r.TimeoutSeconds <= 0 {
		r.TimeoutSeconds = DefaultTimeout
	}
}

// BranchResponse is returned when a branch is created.
type BranchResponse struct {
	BranchID        string         `json:"branch_id"`
	InjectedContext []InjectedItem `json:"injected_context,omitempty"`
	BudgetAllocated int            `json:"budget_allocated"`
	Depth           int            `json:"depth"`
}

// InjectedItem represents a memory or other context injected into a branch.
type InjectedItem struct {
	Type    string `json:"type"` // "memory", "policy", "standard"
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Tokens  int    `json:"tokens"`
}

// ReturnRequest represents a request to complete a branch.
type ReturnRequest struct {
	BranchID      string `json:"branch_id"`
	Message       string `json:"message"`
	ExtractMemory bool   `json:"extract_memory,omitempty"`
}

// Validate checks the return request.
func (r *ReturnRequest) Validate() error {
	if r.BranchID == "" {
		return ErrEmptyBranchID
	}
	if len(r.Message) > MaxReturnMsgLength {
		return ErrMessageTooLong
	}
	return nil
}

// ReturnResponse is returned when a branch completes.
type ReturnResponse struct {
	Success      bool   `json:"success"`
	TokensUsed   int    `json:"tokens_used"`
	MemoryQueued bool   `json:"memory_queued"`
	ScrubbedMsg  string `json:"scrubbed_message"` // Message after secret scrubbing
}
