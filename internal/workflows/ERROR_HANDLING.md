# Error Handling Standard for Temporal Workflows

## Overview

This document defines the standardized error handling pattern for all Temporal workflows in the contextd project. It addresses the inconsistencies identified in PR #65 consensus review and provides clear guidelines for consistent error propagation, logging, and formatting.

## Error Severity Levels

All errors in workflows and activities are categorized into three severity levels:

### 1. CRITICAL - Propagate & Record
**When to use:** Activity failures that prevent workflow completion.

**Pattern:**
```go
if err != nil {
    // CRITICAL: Can't proceed without <resource>
    result.Errors = append(result.Errors, FormatErrorForResult("operation failed", err))
    return result, WrapActivityError("operation failed", err)
}
```

**Examples:**
- Failed to fetch VERSION file (404, network errors)
- Invalid input data (empty files, malformed JSON)
- Missing required resources
- Authentication failures

### 2. HIGH - Record but Continue
**When to use:** Failures in non-essential operations with acceptable fallbacks.

**Pattern:**
```go
if err != nil {
    // HIGH: Validation succeeded, but notification failed - record but continue
    logger.Error("Operation failed", "error", err)
    result.Errors = append(result.Errors, FormatErrorForResult("operation failed", err))
    // Don't return error - let workflow continue
}
```

**Examples:**
- Failed to post comment (workflow validated successfully)
- Agent validation failed (optional enhancement)
- Non-critical schema validation errors

### 3. LOW - Log Only
**When to use:** Failures in cleanup operations or missing optional resources.

**Pattern:**
```go
if err != nil {
    // LOW: Comment removal is cleanup - log only, don't record or propagate
    logger.Warn("Operation failed (non-fatal)", "error", err)
    // Don't add to result.Errors, don't return error
}
```

**Examples:**
- Failed to remove old comment (comment might not exist)
- Cleanup operations
- Optional resource fetching

## Error Wrapping Rules

### Always Use %w for Error Wrapping
This preserves the error chain for `errors.Is` and `errors.As`:

```go
// CORRECT
return fmt.Errorf("failed to fetch file: %w", err)

// INCORRECT
return fmt.Errorf("failed to fetch file: %v", err)
```

### Use Helper Functions
The `errors.go` file provides helper functions for consistent formatting:

```go
// Wrap activity errors with context
WrapActivityError("failed to fetch VERSION file", err)

// Format errors for result.Errors slice
FormatErrorForResult("failed to post comment", err)
```

## Error Message Format

### Standard Format
```
<operation> failed: <details>
```

### Guidelines
1. Use past tense: "failed to X" not "failure to X" or "failed X"
2. Include relevant identifiers: file paths, PR numbers, etc.
3. Be specific about what failed: "failed to fetch VERSION file" not "fetch error"
4. Be consistent: always follow the same pattern

### Examples
```go
// CORRECT
"failed to fetch VERSION file"
"failed to parse plugin.json"
"failed to post version mismatch comment"

// INCORRECT
"fetch error"
"parsing failed"
"comment posting failure"
```

## Activity-Level Error Handling

### Add Logging to All Activities
```go
func MyActivity(ctx context.Context, input MyInput) (MyOutput, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("Starting activity", "key", input.Key)

    // ... activity logic ...

    if err != nil {
        // CRITICAL: Describe why this is critical
        logger.Error("Operation failed", "key", input.Key, "error", err)
        return nil, fmt.Errorf("operation failed: %w", err)
    }

    logger.Info("Activity completed successfully", "result_count", len(results))
    return results, nil
}
```

### Document Error Behavior
Add error handling documentation to all activity functions:

```go
// MyActivity does something important.
//
// Error Handling:
//   - Returns error if GitHub client creation fails
//   - Returns error if file fetch fails (404, network errors, etc.)
//   - Returns error if file content decoding fails
//
// Note: This is a critical operation. Failures will fail the workflow.
func MyActivity(ctx context.Context, input MyInput) (MyOutput, error) {
    // ... implementation ...
}
```

## Workflow-Level Error Handling

### Add Severity Comments
Every error handling block should have a comment explaining the severity:

```go
if err != nil {
    // CRITICAL: Can't proceed without VERSION file
    result.Errors = append(result.Errors, FormatErrorForResult("failed to fetch VERSION file", err))
    return result, WrapActivityError("failed to fetch VERSION file", err)
}
```

### Consistent Result Population
Always populate both `result.Errors` and return the error for critical failures:

```go
// DO THIS (both)
if err != nil {
    result.Errors = append(result.Errors, FormatErrorForResult("operation failed", err))
    return result, WrapActivityError("operation failed", err)
}

// NOT THIS (only one)
if err != nil {
    return result, err  // Missing result.Errors
}
```

## Testing Error Handling

### Test All Error Paths
Every error path should have a test:

```go
func TestWorkflow_VersionFileFetchError(t *testing.T) {
    // ... setup ...

    // Mock activity to return error
    env.OnActivity(FetchFileContentActivity, mock.Anything, mock.Anything).
        Return("", assert.AnError)

    // Execute workflow
    env.ExecuteWorkflow(VersionValidationWorkflow, config)

    // Verify workflow errored
    require.True(t, env.IsWorkflowCompleted())
    require.Error(t, env.GetWorkflowError())

    // Verify result.Errors populated
    var result VersionValidationResult
    require.NoError(t, env.GetWorkflowResult(&result))
    assert.Len(t, result.Errors, 1)
    assert.Contains(t, result.Errors[0], "failed to fetch VERSION file")
}
```

### Test Error Message Format
Verify error messages follow the standard format:

```go
func TestErrorMessages(t *testing.T) {
    tests := []struct{
        name     string
        err      error
        expected string
    }{
        {
            name:     "fetch error",
            err:      errors.New("network timeout"),
            expected: "failed to fetch VERSION file: network timeout",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            msg := FormatErrorForResult("failed to fetch VERSION file", tt.err)
            assert.Equal(t, tt.expected, msg)
        })
    }
}
```

## Migration Checklist

When refactoring existing workflows to use this standard:

- [ ] Replace `fmt.Errorf("msg: %v", err)` with `fmt.Errorf("msg: %w", err)`
- [ ] Add severity comments to all error handling blocks
- [ ] Use `FormatErrorForResult` for result.Errors
- [ ] Use `WrapActivityError` for returned errors
- [ ] Add logging to all activities (Info at start/end, Error on failures)
- [ ] Document error handling behavior in function comments
- [ ] Add/update tests for all error paths
- [ ] Verify error messages follow standard format

## Examples

### Complete Workflow Example

```go
func MyValidationWorkflow(ctx workflow.Context, config MyConfig) (*MyResult, error) {
    logger := workflow.GetLogger(ctx)
    logger.Info("Starting validation")

    result := &MyResult{}

    // Step 1: Fetch required resource
    var data string
    err := workflow.ExecuteActivity(ctx, FetchDataActivity, input).Get(ctx, &data)
    if err != nil {
        // CRITICAL: Can't proceed without data
        result.Errors = append(result.Errors, FormatErrorForResult("failed to fetch data", err))
        return result, WrapActivityError("failed to fetch data", err)
    }

    // Step 2: Process data
    var processed ProcessedData
    err = workflow.ExecuteActivity(ctx, ProcessDataActivity, data).Get(ctx, &processed)
    if err != nil {
        // CRITICAL: Data processing is required
        result.Errors = append(result.Errors, FormatErrorForResult("failed to process data", err))
        return result, WrapActivityError("failed to process data", err)
    }

    // Step 3: Post notification (non-critical)
    err = workflow.ExecuteActivity(ctx, PostNotificationActivity, processed).Get(ctx, nil)
    if err != nil {
        // HIGH: Notification failed, but processing succeeded - record but continue
        logger.Error("Failed to post notification", "error", err)
        result.Errors = append(result.Errors, FormatErrorForResult("failed to post notification", err))
        // Don't return error - let workflow continue
    }

    // Step 4: Cleanup (best effort)
    err = workflow.ExecuteActivity(ctx, CleanupActivity, input).Get(ctx, nil)
    if err != nil {
        // LOW: Cleanup is optional - log only
        logger.Warn("Failed to cleanup (non-fatal)", "error", err)
        // Don't add to result.Errors, don't return error
    }

    logger.Info("Validation complete")
    return result, nil
}
```

### Complete Activity Example

```go
// FetchDataActivity fetches data from an external source.
//
// Error Handling:
//   - Returns error if client creation fails
//   - Returns error if data fetch fails (404, network errors, etc.)
//   - Returns error if data validation fails
func FetchDataActivity(ctx context.Context, input FetchInput) (string, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("Fetching data", "id", input.ID)

    // Create client
    client, err := NewClient(ctx, input.Token)
    if err != nil {
        // CRITICAL: Can't proceed without client
        return "", fmt.Errorf("failed to create client: %w", err)
    }

    // Fetch data
    data, err := client.Fetch(input.ID)
    if err != nil {
        // CRITICAL: Data fetch failed
        logger.Error("Failed to fetch data", "id", input.ID, "error", err)
        return "", fmt.Errorf("failed to fetch data: %w", err)
    }

    // Validate data
    if err := validateData(data); err != nil {
        // CRITICAL: Invalid data
        logger.Error("Data validation failed", "id", input.ID, "error", err)
        return "", fmt.Errorf("data validation failed: %w", err)
    }

    logger.Info("Successfully fetched data", "id", input.ID, "size", len(data))
    return data, nil
}
```

## References

- `internal/workflows/errors.go` - Error helper functions and detailed guidelines
- `internal/workflows/version_validation.go` - Example of standardized error handling
- `internal/workflows/version_validation_activities.go` - Example of activity-level error handling
- `internal/workflows/plugin_validation.go` - Example of handling optional vs required errors
