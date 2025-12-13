package folding

import (
	"errors"
	"testing"
)

func TestErrorCodes(t *testing.T) {
	// Ensure all error codes are unique
	codes := []string{
		ErrCodeBranchNotFound,
		ErrCodeBranchAlreadyExists,
		ErrCodeBranchNotActive,
		ErrCodeMaxDepthExceeded,
		ErrCodeInvalidTransition,
		ErrCodeCannotReturnFromRoot,
		ErrCodeActiveChildBranches,
		ErrCodeBudgetExhausted,
		ErrCodeBudgetNotFound,
		ErrCodeInvalidBudget,
		ErrCodeBudgetOverflow,
		ErrCodeRateLimitExceeded,
		ErrCodeMaxConcurrentBranches,
		ErrCodeScrubbingFailed,
		ErrCodeEmptySessionID,
		ErrCodeEmptyDescription,
		ErrCodeDescriptionTooLong,
		ErrCodeEmptyPrompt,
		ErrCodePromptTooLong,
		ErrCodeEmptyBranchID,
		ErrCodeMessageTooLong,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate error code: %s", code)
		}
		seen[code] = true
	}
}

func TestFoldingError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *FoldingError
		contains []string
	}{
		{
			name: "basic error",
			err: &FoldingError{
				Code:    ErrCodeBranchNotFound,
				Message: "branch not found",
			},
			contains: []string{"FOLD001", "branch not found"},
		},
		{
			name: "with branch ID",
			err: &FoldingError{
				Code:     ErrCodeBranchNotActive,
				Message:  "branch is not active",
				BranchID: "br_123",
			},
			contains: []string{"FOLD003", "branch is not active", "branch_id=br_123"},
		},
		{
			name: "with session ID",
			err: &FoldingError{
				Code:      ErrCodeMaxDepthExceeded,
				Message:   "maximum depth exceeded",
				SessionID: "sess_001",
			},
			contains: []string{"FOLD004", "maximum depth exceeded", "session_id=sess_001"},
		},
		{
			name: "with both IDs",
			err: &FoldingError{
				Code:      ErrCodeBranchNotActive,
				Message:   "branch is not active",
				BranchID:  "br_123",
				SessionID: "sess_001",
			},
			contains: []string{"branch_id=br_123", "session_id=sess_001"},
		},
		{
			name: "with cause",
			err: &FoldingError{
				Code:    ErrCodeScrubbingFailed,
				Message: "scrubbing failed",
				Cause:   errors.New("gitleaks error"),
			},
			contains: []string{"FOLD014", "scrubbing failed", "gitleaks error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !containsString(errStr, substr) {
					t.Errorf("error string %q does not contain %q", errStr, substr)
				}
			}
		})
	}
}

func TestFoldingError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &FoldingError{
		Code:    ErrCodeBudgetExhausted,
		Message: "budget exhausted",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test errors.Is with cause
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
}

func TestNewFoldingError(t *testing.T) {
	cause := errors.New("test cause")
	err := NewFoldingError(
		ErrCodeBranchNotFound,
		"branch not found",
		cause,
		"br_123",
		"sess_001",
	)

	if err.Code != ErrCodeBranchNotFound {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeBranchNotFound)
	}
	if err.Message != "branch not found" {
		t.Errorf("Message = %s, want %s", err.Message, "branch not found")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
	if err.BranchID != "br_123" {
		t.Errorf("BranchID = %s, want %s", err.BranchID, "br_123")
	}
	if err.SessionID != "sess_001" {
		t.Errorf("SessionID = %s, want %s", err.SessionID, "sess_001")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-folding error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "rate limit exceeded",
			err: &FoldingError{
				Code:    ErrCodeRateLimitExceeded,
				Message: "rate limit exceeded",
			},
			expected: true,
		},
		{
			name: "max concurrent branches",
			err: &FoldingError{
				Code:    ErrCodeMaxConcurrentBranches,
				Message: "max concurrent branches",
			},
			expected: true,
		},
		{
			name: "branch not found - not retryable",
			err: &FoldingError{
				Code:    ErrCodeBranchNotFound,
				Message: "branch not found",
			},
			expected: false,
		},
		{
			name: "budget exhausted - retryable (CORR-008)",
			err: &FoldingError{
				Code:    ErrCodeBudgetExhausted,
				Message: "budget exhausted",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-folding error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "branch not found",
			err: &FoldingError{
				Code:    ErrCodeBranchNotFound,
				Message: "branch not found",
			},
			expected: true,
		},
		{
			name: "budget not found",
			err: &FoldingError{
				Code:    ErrCodeBudgetNotFound,
				Message: "budget not found",
			},
			expected: true,
		},
		{
			name: "user error - not a not-found error",
			err: &FoldingError{
				Code:    ErrCodeEmptySessionID,
				Message: "session ID required",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsUserError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-folding error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "empty session ID",
			err: &FoldingError{
				Code:    ErrCodeEmptySessionID,
				Message: "session ID is required",
			},
			expected: true,
		},
		{
			name: "description too long",
			err: &FoldingError{
				Code:    ErrCodeDescriptionTooLong,
				Message: "description too long",
			},
			expected: true,
		},
		{
			name: "max depth exceeded",
			err: &FoldingError{
				Code:    ErrCodeMaxDepthExceeded,
				Message: "max depth exceeded",
			},
			expected: true,
		},
		{
			name: "invalid budget",
			err: &FoldingError{
				Code:    ErrCodeInvalidBudget,
				Message: "invalid budget",
			},
			expected: true,
		},
		{
			name: "scrubbing failed - not user error",
			err: &FoldingError{
				Code:    ErrCodeScrubbingFailed,
				Message: "scrubbing failed",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUserError(tt.err)
			if result != tt.expected {
				t.Errorf("IsUserError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsSystemError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-folding error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "scrubbing failed",
			err: &FoldingError{
				Code:    ErrCodeScrubbingFailed,
				Message: "scrubbing failed",
			},
			expected: true,
		},
		{
			name: "budget not found",
			err: &FoldingError{
				Code:    ErrCodeBudgetNotFound,
				Message: "budget not found",
			},
			expected: true,
		},
		{
			name: "branch not found - not system error",
			err: &FoldingError{
				Code:    ErrCodeBranchNotFound,
				Message: "branch not found",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSystemError(tt.err)
			if result != tt.expected {
				t.Errorf("IsSystemError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrapped := WrapError(ErrCodeBranchNotFound, "branch not found", originalErr, "br_123", "sess_001")

	// Should be a FoldingError
	var foldingErr *FoldingError
	if !errors.As(wrapped, &foldingErr) {
		t.Fatal("wrapped error is not a FoldingError")
	}

	if foldingErr.Code != ErrCodeBranchNotFound {
		t.Errorf("Code = %s, want %s", foldingErr.Code, ErrCodeBranchNotFound)
	}

	// Should unwrap to original error
	if !errors.Is(wrapped, originalErr) {
		t.Error("wrapped error should unwrap to original error")
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
