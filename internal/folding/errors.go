package folding

import "errors"

// Validation errors (SEC-001).
var (
	ErrEmptySessionID     = errors.New("session_id is required")
	ErrEmptyDescription   = errors.New("description is required")
	ErrDescriptionTooLong = errors.New("description exceeds maximum length")
	ErrEmptyPrompt        = errors.New("prompt is required")
	ErrPromptTooLong      = errors.New("prompt exceeds maximum length")
	ErrEmptyBranchID      = errors.New("branch_id is required")
	ErrMessageTooLong     = errors.New("message exceeds maximum length")
)

// Branch lifecycle errors.
var (
	ErrBranchNotFound       = errors.New("branch not found")
	ErrBranchAlreadyExists  = errors.New("branch already exists")
	ErrBranchNotActive      = errors.New("branch is not active")
	ErrMaxDepthExceeded     = errors.New("maximum branch depth exceeded")
	ErrInvalidTransition    = errors.New("invalid state transition")
	ErrCannotReturnFromRoot = errors.New("cannot return from root session context")
	ErrActiveChildBranches  = errors.New("branch has active children")
)

// Budget errors.
var (
	ErrBudgetExhausted  = errors.New("budget exhausted")
	ErrBudgetNotFound   = errors.New("budget not found for branch")
	ErrInvalidBudget    = errors.New("invalid budget amount")
	ErrBudgetOverflow   = errors.New("token consumption would overflow budget")
)

// Rate limiting errors (SEC-003).
var (
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrMaxConcurrentBranches = errors.New("maximum concurrent branches reached")
)

// Secret scrubbing errors (SEC-002).
var (
	ErrScrubbingFailed = errors.New("secret scrubbing failed")
)
