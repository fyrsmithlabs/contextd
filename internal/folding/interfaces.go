package folding

import "context"

// BranchRepository provides persistence for branches.
type BranchRepository interface {
	// Create stores a new branch.
	Create(ctx context.Context, branch *Branch) error
	// Get retrieves a branch by ID.
	Get(ctx context.Context, id string) (*Branch, error)
	// Update modifies an existing branch.
	Update(ctx context.Context, branch *Branch) error
	// Delete removes a branch.
	Delete(ctx context.Context, id string) error
	// ListBySession returns all branches for a session.
	ListBySession(ctx context.Context, sessionID string) ([]*Branch, error)
	// ListByParent returns all child branches of a parent.
	ListByParent(ctx context.Context, parentID string) ([]*Branch, error)
	// GetActiveBySession returns the currently active branch for a session.
	GetActiveBySession(ctx context.Context, sessionID string) (*Branch, error)
	// CountActiveBySession returns the count of active branches in a session.
	CountActiveBySession(ctx context.Context, sessionID string) (int, error)
}

// TokenCounter counts tokens in text content.
type TokenCounter interface {
	// Count returns the token count for the given content.
	Count(content string) (int, error)
}

// BranchEvent represents an event in the branch lifecycle.
type BranchEvent interface {
	// Type returns the event type identifier.
	Type() string
	// BranchID returns the branch this event relates to.
	BranchID() string
}

// EventEmitter emits and routes branch events.
type EventEmitter interface {
	// Emit sends an event to all subscribers.
	Emit(event BranchEvent)
	// Subscribe registers a handler for events.
	Subscribe(handler func(BranchEvent))
}

// SecretScrubber removes secrets from content.
type SecretScrubber interface {
	// Scrub removes secrets from the content, returning scrubbed version.
	Scrub(content string) (string, error)
}

// MemorySearcher searches for relevant memories.
type MemorySearcher interface {
	// Search finds memories relevant to the query.
	Search(ctx context.Context, query string, limit int, minConfidence float64) ([]InjectedItem, error)
}

// SessionValidator validates that a caller has access to a session.
// This interface enables authentication/authorization enforcement (SEC-004).
type SessionValidator interface {
	// ValidateSession checks if the caller (identified by callerID) has access to sessionID.
	// Returns nil if access is allowed, ErrSessionUnauthorized if not.
	// The callerID can be extracted from MCP context, JWT claims, or other auth mechanisms.
	ValidateSession(ctx context.Context, sessionID string, callerID string) error
}

// PermissiveSessionValidator allows all session access (for single-user deployments).
type PermissiveSessionValidator struct{}

// ValidateSession always returns nil (allows all access).
func (v *PermissiveSessionValidator) ValidateSession(ctx context.Context, sessionID string, callerID string) error {
	return nil
}

// StrictSessionValidator requires session ownership match.
type StrictSessionValidator struct{}

// ValidateSession requires callerID to match sessionID prefix or be the session owner.
// Sessions are expected to follow format: "user_<userID>_<sessionID>" or callerID must match exactly.
func (v *StrictSessionValidator) ValidateSession(ctx context.Context, sessionID string, callerID string) error {
	if callerID == "" {
		return ErrSessionUnauthorized
	}
	// Check if session belongs to caller (session starts with caller's ID or matches exactly)
	if sessionID == callerID || len(sessionID) > len(callerID) && sessionID[:len(callerID)+1] == callerID+"_" {
		return nil
	}
	return ErrSessionUnauthorized
}

// --- Event Types ---

// BudgetExhaustedEvent is emitted when a branch exhausts its budget.
type BudgetExhaustedEvent struct {
	branchID    string
	BudgetUsed  int
	BudgetTotal int
}

func (e BudgetExhaustedEvent) Type() string     { return "budget_exhausted" }
func (e BudgetExhaustedEvent) BranchID() string { return e.branchID }

// BudgetWarningEvent is emitted when a branch reaches 80% budget usage.
type BudgetWarningEvent struct {
	branchID    string
	BudgetUsed  int
	BudgetTotal int
	Percentage  float64
}

func (e BudgetWarningEvent) Type() string     { return "budget_warning" }
func (e BudgetWarningEvent) BranchID() string { return e.branchID }

// TimeoutEvent is emitted when a branch times out.
type TimeoutEvent struct {
	branchID       string
	TimeoutSeconds int
}

func (e TimeoutEvent) Type() string     { return "timeout" }
func (e TimeoutEvent) BranchID() string { return e.branchID }

// BranchCompletedEvent is emitted when a branch completes normally.
type BranchCompletedEvent struct {
	branchID   string
	TokensUsed int
	Success    bool
}

func (e BranchCompletedEvent) Type() string     { return "branch_completed" }
func (e BranchCompletedEvent) BranchID() string { return e.branchID }
