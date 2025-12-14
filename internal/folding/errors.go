package folding

import (
	"errors"
	"fmt"
)

// Error codes for structured error handling.
// These codes are stable and can be used for programmatic error handling.
const (
	// Branch lifecycle errors (FOLD001-FOLD007)
	ErrCodeBranchNotFound       = "FOLD001"
	ErrCodeBranchAlreadyExists  = "FOLD002"
	ErrCodeBranchNotActive      = "FOLD003"
	ErrCodeMaxDepthExceeded     = "FOLD004"
	ErrCodeInvalidTransition    = "FOLD005"
	ErrCodeCannotReturnFromRoot = "FOLD006"
	ErrCodeActiveChildBranches  = "FOLD007"

	// Budget errors (FOLD008-FOLD011)
	ErrCodeBudgetExhausted = "FOLD008"
	ErrCodeBudgetNotFound  = "FOLD009"
	ErrCodeInvalidBudget   = "FOLD010"
	ErrCodeBudgetOverflow  = "FOLD011"

	// Rate limiting errors (FOLD012-FOLD013)
	ErrCodeRateLimitExceeded     = "FOLD012"
	ErrCodeMaxConcurrentBranches = "FOLD013"

	// Secret scrubbing errors (FOLD014)
	ErrCodeScrubbingFailed = "FOLD014"

	// Validation errors (FOLD015-FOLD021)
	ErrCodeEmptySessionID     = "FOLD015"
	ErrCodeEmptyDescription   = "FOLD016"
	ErrCodeDescriptionTooLong = "FOLD017"
	ErrCodeEmptyPrompt        = "FOLD018"
	ErrCodePromptTooLong      = "FOLD019"
	ErrCodeEmptyBranchID      = "FOLD020"
	ErrCodeMessageTooLong     = "FOLD021"

	// Authorization errors (FOLD022) - SEC-004
	ErrCodeSessionUnauthorized = "FOLD022"
)

// FoldingError represents a structured error with context and categorization.
// It implements the error interface and supports error wrapping via Unwrap.
type FoldingError struct {
	Code      string // Error code for categorization (e.g., "FOLD001")
	Message   string // Human-readable error message
	Cause     error  // Underlying cause (if any)
	BranchID  string // Branch ID context (if applicable)
	SessionID string // Session ID context (if applicable)
}

// Error implements the error interface.
func (e *FoldingError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)

	// Add context if available
	if e.BranchID != "" || e.SessionID != "" {
		msg += " ("
		if e.BranchID != "" {
			msg += fmt.Sprintf("branch_id=%s", e.BranchID)
			if e.SessionID != "" {
				msg += ", "
			}
		}
		if e.SessionID != "" {
			msg += fmt.Sprintf("session_id=%s", e.SessionID)
		}
		msg += ")"
	}

	// Add cause if present
	if e.Cause != nil {
		msg += fmt.Sprintf(": %s", e.Cause.Error())
	}

	return msg
}

// Unwrap implements the errors.Unwrap interface for error chaining.
func (e *FoldingError) Unwrap() error {
	return e.Cause
}

// NewFoldingError creates a new FoldingError with the given parameters.
func NewFoldingError(code, message string, cause error, branchID, sessionID string) *FoldingError {
	return &FoldingError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		BranchID:  branchID,
		SessionID: sessionID,
	}
}

// WrapError wraps an existing error with folding context.
// This is a convenience function that creates a FoldingError with a cause.
func WrapError(code, message string, cause error, branchID, sessionID string) error {
	return NewFoldingError(code, message, cause, branchID, sessionID)
}

// IsRetryable returns true if the error represents a transient condition
// that may succeed on retry (e.g., rate limiting, resource exhaustion).
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var foldingErr *FoldingError
	if !errors.As(err, &foldingErr) {
		return false
	}

	switch foldingErr.Code {
	case ErrCodeRateLimitExceeded, ErrCodeMaxConcurrentBranches, ErrCodeBudgetExhausted:
		return true
	default:
		return false
	}
}

// IsNotFoundError returns true if the error indicates a resource was not found.
// CORR-008: Added missing categorization for not-found errors.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var foldingErr *FoldingError
	if !errors.As(err, &foldingErr) {
		return false
	}

	switch foldingErr.Code {
	case ErrCodeBranchNotFound, ErrCodeBudgetNotFound:
		return true
	default:
		return false
	}
}

// IsUserError returns true if the error was caused by invalid user input
// or policy violations that the user can correct.
func IsUserError(err error) bool {
	if err == nil {
		return false
	}

	var foldingErr *FoldingError
	if !errors.As(err, &foldingErr) {
		return false
	}

	switch foldingErr.Code {
	case ErrCodeEmptySessionID,
		ErrCodeEmptyDescription,
		ErrCodeDescriptionTooLong,
		ErrCodeEmptyPrompt,
		ErrCodePromptTooLong,
		ErrCodeEmptyBranchID,
		ErrCodeMessageTooLong,
		ErrCodeMaxDepthExceeded,
		ErrCodeInvalidBudget,
		ErrCodeBudgetOverflow,
		ErrCodeInvalidTransition,
		ErrCodeBranchNotActive,
		ErrCodeCannotReturnFromRoot,
		ErrCodeActiveChildBranches,
		ErrCodeBranchAlreadyExists:
		return true
	default:
		return false
	}
}

// IsSystemError returns true if the error represents an internal system failure
// that is not caused by user input.
func IsSystemError(err error) bool {
	if err == nil {
		return false
	}

	var foldingErr *FoldingError
	if !errors.As(err, &foldingErr) {
		return false
	}

	switch foldingErr.Code {
	case ErrCodeScrubbingFailed, ErrCodeBudgetNotFound:
		return true
	default:
		return false
	}
}

// IsAuthorizationError returns true if the error indicates an authorization failure.
// SEC-004: These errors should result in HTTP 403 Forbidden responses.
func IsAuthorizationError(err error) bool {
	if err == nil {
		return false
	}

	// Check for sentinel error
	if errors.Is(err, ErrSessionUnauthorized) {
		return true
	}

	var foldingErr *FoldingError
	if !errors.As(err, &foldingErr) {
		return false
	}

	switch foldingErr.Code {
	case ErrCodeSessionUnauthorized:
		return true
	default:
		return false
	}
}

// Validation errors (SEC-001) - kept for backward compatibility.
var (
	ErrEmptySessionID     = errors.New("session_id is required")
	ErrEmptyDescription   = errors.New("description is required")
	ErrDescriptionTooLong = errors.New("description exceeds maximum length")
	ErrEmptyPrompt        = errors.New("prompt is required")
	ErrPromptTooLong      = errors.New("prompt exceeds maximum length")
	ErrEmptyBranchID      = errors.New("branch_id is required")
	ErrMessageTooLong     = errors.New("message exceeds maximum length")
)

// Branch lifecycle errors - kept for backward compatibility.
var (
	ErrBranchNotFound       = errors.New("branch not found")
	ErrBranchAlreadyExists  = errors.New("branch already exists")
	ErrBranchNotActive      = errors.New("branch is not active")
	ErrMaxDepthExceeded     = errors.New("maximum branch depth exceeded")
	ErrInvalidTransition    = errors.New("invalid state transition")
	ErrCannotReturnFromRoot = errors.New("cannot return from root session context")
	ErrActiveChildBranches  = errors.New("branch has active children")
)

// Budget errors - kept for backward compatibility.
var (
	ErrBudgetExhausted = errors.New("budget exhausted")
	ErrBudgetNotFound  = errors.New("budget not found for branch")
	ErrInvalidBudget   = errors.New("invalid budget amount")
	ErrBudgetOverflow  = errors.New("token consumption would overflow budget")
)

// Rate limiting errors (SEC-003) - kept for backward compatibility.
var (
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrMaxConcurrentBranches = errors.New("maximum concurrent branches reached")
)

// Secret scrubbing errors (SEC-002) - kept for backward compatibility.
var (
	ErrScrubbingFailed = errors.New("secret scrubbing failed")
)

// Authorization errors (SEC-004) - kept for backward compatibility.
var (
	ErrSessionUnauthorized = errors.New("session access unauthorized")
)
