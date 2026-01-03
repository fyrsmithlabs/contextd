package workflows

import (
	"fmt"
)

// Error severity levels for workflow errors
type ErrorSeverity string

const (
	// ErrorSeverityCritical indicates the workflow must fail
	ErrorSeverityCritical ErrorSeverity = "critical"
	// ErrorSeverityHigh indicates a major issue but workflow can continue
	ErrorSeverityHigh ErrorSeverity = "high"
	// ErrorSeverityLow indicates a minor issue that doesn't affect main functionality
	ErrorSeverityLow ErrorSeverity = "low"
)

// WorkflowError represents a structured error in a workflow
type WorkflowError struct {
	Operation string        // The operation that failed (e.g., "fetch_version_file", "validate_schema")
	Severity  ErrorSeverity // How severe the error is
	Err       error         // The underlying error
	Context   string        // Additional context about the error
}

// Error implements the error interface
func (e *WorkflowError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s failed: %s (%s)", e.Operation, e.Err.Error(), e.Context)
	}
	return fmt.Sprintf("%s failed: %s", e.Operation, e.Err.Error())
}

// Unwrap allows errors.Is and errors.As to work with WorkflowError
func (e *WorkflowError) Unwrap() error {
	return e.Err
}

// NewWorkflowError creates a new workflow error with context
func NewWorkflowError(operation string, severity ErrorSeverity, err error, context string) *WorkflowError {
	return &WorkflowError{
		Operation: operation,
		Severity:  severity,
		Err:       err,
		Context:   context,
	}
}

// WrapActivityError wraps an activity error with operation context.
// Use this when an activity fails to provide consistent error messages.
func WrapActivityError(operation string, err error) error {
	return fmt.Errorf("%s: %w", operation, err)
}

// FormatErrorForResult formats an error for inclusion in workflow result.Errors slice.
// This creates a human-readable error message for end users.
func FormatErrorForResult(operation string, err error) string {
	return fmt.Sprintf("%s: %v", operation, err)
}

// ErrorHandlingGuidelines documents the standard error handling pattern for workflows.
//
// CRITICAL (Propagate & Record):
//   - Activity failures that prevent workflow completion
//   - Invalid input data (empty files, malformed JSON)
//   - Missing required resources (files not found)
//   - Pattern: Add to result.Errors AND return error to fail workflow
//   - Example: Failed to fetch VERSION file, invalid JSON in plugin.json
//
// HIGH (Record but Continue):
//   - Failures in non-essential operations
//   - Operations with acceptable fallbacks
//   - Pattern: Add to result.Errors but DON'T return error, let workflow continue
//   - Example: Failed to post comment (workflow validated successfully)
//
// LOW (Log as Warning):
//   - Failures in cleanup operations
//   - Missing optional resources
//   - Pattern: Log as warning, DON'T add to result.Errors, DON'T return error
//   - Example: Failed to remove old comment (comment might not exist)
//
// Error Message Format:
//   - Use descriptive operation names: "failed to fetch VERSION file" not "fetch error"
//   - Include relevant identifiers: file paths, PR numbers, etc.
//   - Use %w for wrapping to preserve error chain: fmt.Errorf("operation failed: %w", err)
//   - Be consistent: Always use past tense ("failed to X" not "failed X" or "failure to X")
//
// Examples:
//
//   // CRITICAL - Propagate and record
//   if err != nil {
//       result.Errors = append(result.Errors, FormatErrorForResult("failed to fetch VERSION file", err))
//       return result, WrapActivityError("failed to fetch VERSION file", err)
//   }
//
//   // HIGH - Record but continue
//   if err != nil {
//       logger.Error("Failed to post comment", "error", err)
//       result.Errors = append(result.Errors, FormatErrorForResult("failed to post comment", err))
//       // Don't return - let workflow continue
//   }
//
//   // LOW - Log only
//   if err != nil {
//       logger.Warn("Failed to remove comment (non-fatal)", "error", err)
//       // Don't add to result.Errors, don't return
//   }
