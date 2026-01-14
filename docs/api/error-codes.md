# Error Codes Reference

This document provides comprehensive reference documentation for all error codes in ContextD. Errors are organized by category with detailed descriptions, causes, and resolution steps.

---

## Overview

ContextD uses structured error codes to provide clear, actionable error messages. Error codes are stable across versions and can be used for programmatic error handling.

| Category | Code Range | Description |
|----------|------------|-------------|
| **Context-Folding** | FOLD001-FOLD022 | Branch lifecycle, budgets, rate limiting, validation |
| **Multi-Tenancy** | (Sentinel errors) | Tenant context and security errors |
| **API** | (Sentinel errors) | Common API operation errors |
| **ReasoningBank** | (Sentinel errors) | Memory storage and retrieval errors |
| **Project** | (Sentinel errors) | Project management errors |
| **Troubleshooting** | (Sentinel errors) | Error diagnosis and pattern matching |
| **Embeddings** | (Sentinel errors) | Model loading and embedding generation |
| **Compression** | (Sentinel errors) | A/B testing and experiment management |
| **Workflow** | (Severity-based) | Temporal workflow errors |

---

## Error Response Format

### Structured Errors (Context-Folding)

Context-Folding operations return structured `FoldingError` objects:

```json
{
  "error": {
    "code": "FOLD001",
    "message": "branch not found",
    "branch_id": "branch_abc123",
    "session_id": "sess_xyz789",
    "cause": "underlying error details"
  }
}
```

### Sentinel Errors (Other Categories)

Most other categories use Go sentinel errors for matching:

```json
{
  "error": "tenant info missing from context"
}
```

---

## Context-Folding Errors (FOLD001-FOLD022)

Context-Folding provides isolated sub-task execution with token budgets. These structured errors include error codes, context, and error chaining.

### Branch Lifecycle Errors (FOLD001-FOLD007)

#### FOLD001: Branch Not Found

**Error:** `ErrBranchNotFound`

**Description:** The requested branch does not exist in the session.

**Causes:**
- Branch ID is incorrect or misspelled
- Branch was never created
- Branch was deleted

**Resolution:**
```bash
# List all branches in session
branch_status --session-id=<session_id>

# Create the branch first
branch_create --session-id=<session_id> \
  --description="Task description" \
  --prompt="Task prompt" \
  --budget=4000
```

**Example:**
```json
{
  "tool": "branch_return",
  "arguments": {
    "branch_id": "nonexistent_branch",
    "message": "Task complete"
  }
}
// Error: [FOLD001] branch not found (branch_id=nonexistent_branch)
```

---

#### FOLD002: Branch Already Exists

**Error:** `ErrBranchAlreadyExists`

**Description:** A branch with the given ID already exists in the session.

**Causes:**
- Attempting to create a branch with a duplicate ID
- Branch creation was retried without checking existence

**Resolution:**
- Use `branch_status` to check existing branches
- Choose a different branch ID
- Resume the existing branch instead of creating a new one

**Example:**
```json
{
  "tool": "branch_create",
  "arguments": {
    "session_id": "sess_123",
    "description": "Debug issue",
    "prompt": "Find the bug",
    "budget": 4000
  }
}
// Error: [FOLD002] branch already exists (session_id=sess_123)
```

---

#### FOLD003: Branch Not Active

**Error:** `ErrBranchNotActive`

**Description:** Operation requires an active branch, but the branch is in a different state.

**Causes:**
- Branch has been completed
- Branch was terminated due to budget exhaustion
- Branch is in error state

**Resolution:**
- Check branch status with `branch_status`
- Create a new branch for the task
- If budget was exhausted, create a new branch with larger budget

**See Also:** [Budget Exhausted (FOLD008)](#fold008-budget-exhausted)

---

#### FOLD004: Max Depth Exceeded

**Error:** `ErrMaxDepthExceeded`

**Description:** The maximum nesting depth for branches has been exceeded.

**Causes:**
- Attempting to create nested branches beyond the configured limit (default: 3 levels)
- Recursive branch creation pattern

**Resolution:**
```bash
# Return from current branch before creating a new one
branch_return --branch-id=<current_branch> --message="Subtask complete"

# Then create the new branch
branch_create --session-id=<session_id> ...
```

**Configuration:**
```go
config := &folding.Config{
    MaxDepth: 5, // Increase if needed (default: 3)
}
```

**See Also:** [Context-Folding Architecture](../spec/context-folding/ARCH.md#nesting-limits)

---

#### FOLD005: Invalid Transition

**Error:** `ErrInvalidTransition`

**Description:** The requested state transition is not allowed by the branch state machine.

**Causes:**
- Attempting to return from a branch that has active children
- Invalid state transition sequence

**Valid State Transitions:**
```
Created → Active → Completed
Created → Active → Failed
Active → Suspended → Active
```

**Resolution:**
- Ensure all child branches are completed before returning
- Follow the correct state transition sequence

**See Also:** [Branch Lifecycle](../spec/context-folding/SPEC.md#branch-lifecycle)

---

#### FOLD006: Cannot Return From Root

**Error:** `ErrCannotReturnFromRoot`

**Description:** Attempted to call `branch_return` from the root session context.

**Causes:**
- Calling `branch_return` without being in a branch
- Branch ID matches the session ID (root context)

**Resolution:**
- Only call `branch_return` from within an active branch
- Use session management instead for root context

**Example:**
```json
{
  "tool": "branch_return",
  "arguments": {
    "branch_id": "sess_123",  // This is the root session, not a branch!
    "message": "Done"
  }
}
// Error: [FOLD006] cannot return from root session context
```

---

#### FOLD007: Active Child Branches

**Error:** `ErrActiveChildBranches`

**Description:** Cannot complete or terminate a branch that has active child branches.

**Causes:**
- Parent branch trying to complete while children are still running
- Child branches not properly cleaned up

**Resolution:**
```bash
# Return from all child branches first
branch_return --branch-id=<child1> --message="Child complete"
branch_return --branch-id=<child2> --message="Child complete"

# Then return from parent
branch_return --branch-id=<parent> --message="Parent complete"
```

---

### Budget Errors (FOLD008-FOLD011)

#### FOLD008: Budget Exhausted

**Error:** `ErrBudgetExhausted`

**Description:** The branch has consumed its entire token budget.

**Causes:**
- Task consumed more tokens than allocated
- Budget was set too low for the task complexity
- Inefficient token usage (e.g., reading large files into context)

**Resolution:**
```bash
# Create a new branch with larger budget
branch_create --session-id=<session_id> \
  --description="Continue previous task" \
  --prompt="Pick up where we left off" \
  --budget=8000  # Increased from 4000

# Or optimize token usage
# - Use semantic search instead of reading full files
# - Summarize findings before returning
# - Split task into smaller sub-branches
```

**Prevention:**
- Estimate token requirements before creating branch
- Use extractive compression for large content
- Monitor budget with `branch_status`

**Retryable:** Yes - Create a new branch with adjusted budget

---

#### FOLD009: Budget Not Found

**Error:** `ErrBudgetNotFound`

**Description:** No budget record exists for the specified branch.

**Causes:**
- Internal system error
- Branch was deleted but cleanup incomplete
- Database inconsistency

**Resolution:**
- Contact system administrator
- Check system logs for underlying cause
- If retrying, create a new branch

**Retryable:** No - System error requiring investigation

**See Also:** [Troubleshooting Guide](../troubleshooting.md#budget-tracking-issues)

---

#### FOLD010: Invalid Budget

**Error:** `ErrInvalidBudget`

**Description:** The specified budget amount is invalid.

**Causes:**
- Budget is zero or negative
- Budget exceeds maximum allowed value
- Budget is not an integer

**Resolution:**
```bash
# Valid budget range: 1 to 100,000 tokens
branch_create --session-id=<session_id> \
  --description="Task" \
  --prompt="Prompt" \
  --budget=4000  # Must be positive integer ≤ 100,000
```

**Valid Budget Ranges:**
- Minimum: 1 token
- Maximum: 100,000 tokens
- Recommended: 4,000-10,000 tokens for typical tasks

---

#### FOLD011: Budget Overflow

**Error:** `ErrBudgetOverflow`

**Description:** Token consumption would cause the budget counter to overflow.

**Causes:**
- Internal accounting error
- Integer overflow in token tracking
- Corrupted budget state

**Resolution:**
- Report as system bug
- Create a new branch
- Check for underlying integer overflow issues

**Retryable:** No - System error requiring investigation

---

### Rate Limiting Errors (FOLD012-FOLD013)

#### FOLD012: Rate Limit Exceeded

**Error:** `ErrRateLimitExceeded`

**Description:** Too many operations in a short time period.

**Causes:**
- Rapid branch creation (>10/second)
- Automated retry loops without backoff
- Burst traffic pattern

**Resolution:**
```bash
# Wait before retrying (exponential backoff recommended)
sleep 2
# Retry operation
```

**Rate Limits:**
- Branch creation: 10 per second
- Branch return: 20 per second
- Status queries: 100 per second

**Retryable:** Yes - Wait and retry with exponential backoff

**See Also:** [Rate Limiting](../spec/context-folding/SPEC.md#rate-limiting)

---

#### FOLD013: Max Concurrent Branches

**Error:** `ErrMaxConcurrentBranches`

**Description:** Maximum number of concurrent branches reached for this session.

**Causes:**
- Too many branches created without completing previous ones
- Concurrent branch creation pattern
- Leaked branches not properly cleaned up

**Resolution:**
```bash
# Complete existing branches first
branch_return --branch-id=<branch1> --message="Complete"
branch_return --branch-id=<branch2> --message="Complete"

# Check branch status
branch_status --session-id=<session_id>

# Then create new branch
branch_create ...
```

**Limits:**
- Default: 10 concurrent branches per session
- Configurable via `MaxConcurrentBranches`

**Retryable:** Yes - After completing existing branches

---

### Secret Scrubbing Errors (FOLD014)

#### FOLD014: Scrubbing Failed

**Error:** `ErrScrubbingFailed`

**Description:** Secret scrubbing failed on branch return.

**Causes:**
- Gitleaks detector initialization failed
- I/O error during scanning
- Corrupted detector rules
- Resource exhaustion

**Resolution:**
1. **Check system resources:**
   ```bash
   # Ensure sufficient memory
   free -m

   # Check disk space
   df -h
   ```

2. **Verify gitleaks configuration:**
   ```bash
   # Test gitleaks manually
   echo "test content" | gitleaks detect --no-git -
   ```

3. **Retry operation:**
   - Secret scrubbing failures are typically transient
   - Retry with exponential backoff

**Security Note:** When scrubbing fails, the operation is **blocked** to prevent secret leakage. This is fail-safe behavior.

**See Also:** [Secret Scrubbing](../spec/context-folding/SPEC.md#security), [Troubleshooting](../troubleshooting.md#secret-detection-issues)

---

### Validation Errors (FOLD015-FOLD021)

#### FOLD015: Empty Session ID

**Error:** `ErrEmptySessionID`

**Description:** The `session_id` parameter is required but was not provided.

**Resolution:**
```json
{
  "tool": "branch_create",
  "arguments": {
    "session_id": "sess_abc123",  // Required!
    "description": "Task",
    "prompt": "Prompt",
    "budget": 4000
  }
}
```

---

#### FOLD016: Empty Description

**Error:** `ErrEmptyDescription`

**Description:** The `description` parameter is required but was not provided.

**Resolution:**
```json
{
  "tool": "branch_create",
  "arguments": {
    "session_id": "sess_123",
    "description": "Search for authentication function",  // Required!
    "prompt": "Find authenticate() in src/",
    "budget": 4000
  }
}
```

---

#### FOLD017: Description Too Long

**Error:** `ErrDescriptionTooLong`

**Description:** The `description` exceeds the maximum allowed length.

**Limits:**
- Maximum: 200 characters
- Recommended: 50-100 characters

**Resolution:**
```json
{
  "tool": "branch_create",
  "arguments": {
    "session_id": "sess_123",
    "description": "Search auth function",  // Concise ✓
    "prompt": "Find authenticate() function in src/ directory...",
    "budget": 4000
  }
}
```

---

#### FOLD018: Empty Prompt

**Error:** `ErrEmptyPrompt`

**Description:** The `prompt` parameter is required but was not provided.

**Resolution:**
```json
{
  "tool": "branch_create",
  "arguments": {
    "session_id": "sess_123",
    "description": "Search auth function",
    "prompt": "Search src/ for authenticate() function definition",  // Required!
    "budget": 4000
  }
}
```

---

#### FOLD019: Prompt Too Long

**Error:** `ErrPromptTooLong`

**Description:** The `prompt` exceeds the maximum allowed length.

**Limits:**
- Maximum: 2000 characters
- Recommended: 200-500 characters for focused tasks

**Resolution:**
- Break large prompts into multiple branches
- Use concise, focused instructions
- Reference external context instead of including it

---

#### FOLD020: Empty Branch ID

**Error:** `ErrEmptyBranchID`

**Description:** The `branch_id` parameter is required but was not provided.

**Resolution:**
```json
{
  "tool": "branch_return",
  "arguments": {
    "branch_id": "branch_abc123",  // Required!
    "message": "Task completed successfully"
  }
}
```

---

#### FOLD021: Message Too Long

**Error:** `ErrMessageTooLong`

**Description:** The `message` returned from a branch exceeds the maximum allowed length.

**Limits:**
- Maximum: 10,000 characters
- Recommended: 500-2000 characters

**Resolution:**
```bash
# Bad: Including full file contents
branch_return --branch-id=<id> \
  --message="Found it: <entire 5000-line file>"

# Good: Summarize findings
branch_return --branch-id=<id> \
  --message="Found authenticate() in src/auth.go:142. Uses JWT tokens with RS256."
```

**Best Practices:**
- Summarize findings, don't copy full content
- Include specific locations (file:line)
- Focus on actionable insights
- Use extractive compression for large results

---

### Authorization Errors (FOLD022)

#### FOLD022: Session Unauthorized

**Error:** `ErrSessionUnauthorized`

**Description:** The caller is not authorized to access this session or branch.

**HTTP Status:** 403 Forbidden

**Causes:**
- Session belongs to a different tenant
- Missing or invalid authentication token
- Session ID does not exist
- Authorization context not provided

**Resolution:**
1. **Verify session ownership:**
   ```bash
   # Check current session
   branch_status --session-id=<session_id>
   ```

2. **Ensure authentication:**
   - Provide valid authentication token
   - Use session IDs from your own tenant

3. **Check tenant context:**
   ```go
   // Ensure tenant context is set
   ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
       TenantID: "your-tenant-id",
   })
   ```

**Security Note:** This is a security-critical error. Access to sessions is strictly isolated by tenant.

**See Also:** [Multi-Tenancy](#multi-tenancy-errors), [Security](../spec/context-folding/SPEC.md#security)

---

## Multi-Tenancy Errors

Multi-tenancy errors enforce strict isolation between tenants, teams, and projects. These use a **fail-closed** security model.

### Missing Tenant Context

**Error:** `ErrMissingTenant`

**Description:** Tenant information is missing from the request context.

**Causes:**
- Context not initialized with tenant info
- Middleware not applied
- Direct API calls without authentication

**Resolution:**
```go
// Always create tenant-scoped context
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",      // Required
    TeamID:    "platform",     // Optional
    ProjectID: "contextd",     // Optional
})

// All operations automatically filtered
results, err := store.Search(ctx, "query", 10)
```

**Security:** Operations fail with error instead of returning empty results. This prevents accidental data leakage.

**See Also:** [Multi-Tenancy Architecture](../CLAUDE.md#multi-tenancy-architecture)

---

### Invalid Tenant Identifier

**Error:** `ErrInvalidTenant`

**Description:** Tenant identifier is invalid or malformed.

**Causes:**
- Empty tenant ID
- Tenant ID is not a string
- Tenant ID contains invalid characters

**Resolution:**
```go
// Valid tenant info
tenant := &vectorstore.TenantInfo{
    TenantID: "org-123",  // Non-empty string
}

// Validate before use
if err := tenant.Validate(); err != nil {
    return err
}
```

**Validation Rules:**
- TenantID must be non-empty string
- TeamID and ProjectID are optional but must be strings if provided
- No special validation on format (application-defined)

---

### Tenant Filter Injection

**Error:** `ErrTenantFilterInUserFilters`

**Description:** User attempted to inject tenant fields into query filters.

**Causes:**
- User-provided filters contain `tenant_id`, `team_id`, or `project_id`
- Attempting to bypass tenant isolation

**Resolution:**
```go
// BAD: Don't include tenant fields in user filters
userFilters := map[string]interface{}{
    "status": "active",
    "tenant_id": "org-999",  // ❌ REJECTED!
}

// GOOD: Tenant context is separate
ctx := vectorstore.ContextWithTenant(ctx, tenantInfo)
userFilters := map[string]interface{}{
    "status": "active",  // ✓ OK
}
results, err := store.Search(ctx, "query", 10, userFilters)
```

**Security:** This prevents users from accessing other tenants' data by injecting tenant filters.

**See Also:** [Payload-Based Isolation](../spec/vector-storage/security.md#payload-isolation)

---

## API Errors

Common errors for API operations.

### Session Required

**Error:** `ErrSessionRequired`

**Description:** The `session_id` parameter is required but was not provided.

**Resolution:**
```json
{
  "tool": "memory_search",
  "arguments": {
    "project_id": "contextd",
    "query": "debugging strategies"
  }
}
```

---

### Invalid Request

**Error:** `ErrInvalidRequest`

**Description:** The request is malformed or contains invalid parameters.

**Causes:**
- Missing required fields
- Invalid JSON format
- Type mismatch in parameters
- Parameter validation failed

**Resolution:**
- Check API documentation for required parameters
- Validate JSON syntax
- Ensure parameter types match schema

**See Also:** [MCP Tools Reference](./mcp-tools.md)

---

### Resource Not Found

**Error:** `ErrNotFound`

**Description:** The requested resource does not exist.

**HTTP Status:** 404 Not Found

**Causes:**
- Invalid resource ID
- Resource was deleted
- Typo in resource identifier

**Resolution:**
- Verify resource ID is correct
- Check that resource exists with list operation
- Ensure you have access to the resource

---

### Operation Timeout

**Error:** `ErrTimeout`

**Description:** The operation exceeded the maximum allowed time.

**Causes:**
- Large dataset processing
- Slow network
- Database query timeout
- Vector search on large collection

**Resolution:**
1. **Retry with backoff:**
   ```bash
   # Retry with exponential backoff
   sleep 2 && retry_operation
   ```

2. **Optimize query:**
   - Reduce result limit
   - Add more specific filters
   - Use pagination

3. **Check system health:**
   ```bash
   curl http://localhost:9090/health
   ```

**Timeout Limits:**
- Default API timeout: 30 seconds
- Vector search: 10 seconds
- Embedding generation: 5 seconds per batch

---

### Permission Denied

**Error:** `ErrPermissionDenied`

**Description:** The caller does not have permission to perform this operation.

**HTTP Status:** 403 Forbidden

**Causes:**
- Insufficient privileges
- Resource belongs to different tenant
- Operation requires admin role

**Resolution:**
- Verify your authentication credentials
- Check that you own the resource
- Request elevated privileges if needed

---

## ReasoningBank Errors

Errors related to cross-session memory operations.

### Memory Not Found

**Error:** `ErrMemoryNotFound`

**Description:** The requested memory does not exist.

**Causes:**
- Invalid memory ID
- Memory was deleted
- Memory belongs to different project

**Resolution:**
```bash
# Search for memories
memory_search --project-id=<project> --query="your search"

# Verify memory ID
memory_feedback --memory-id=<id> --helpful=true
```

---

### Invalid Memory

**Error:** `ErrInvalidMemory`

**Description:** Memory validation failed.

**Causes:**
- Required fields missing
- Invalid field values
- Memory structure malformed

**Resolution:**
```json
{
  "tool": "memory_record",
  "arguments": {
    "project_id": "contextd",           // Required
    "title": "Memory title",             // Required, non-empty
    "content": "Detailed description",   // Required, non-empty
    "outcome": "success",                // Required: "success" or "failure"
    "tags": ["tag1", "tag2"]            // Optional
  }
}
```

---

### Empty Title

**Error:** `ErrEmptyTitle`

**Description:** Memory title cannot be empty.

**Resolution:**
```json
{
  "tool": "memory_record",
  "arguments": {
    "project_id": "contextd",
    "title": "Use table-driven tests for Go",  // Required!
    "content": "...",
    "outcome": "success"
  }
}
```

**Best Practices:**
- Keep titles concise (50-100 characters)
- Use action-oriented titles ("Use X for Y", "Avoid X when Y")
- Include key context (language, framework, component)

---

### Empty Content

**Error:** `ErrEmptyContent`

**Description:** Memory content cannot be empty.

**Resolution:**
```json
{
  "tool": "memory_record",
  "arguments": {
    "project_id": "contextd",
    "title": "Use table-driven tests",
    "content": "When writing Go tests, use table-driven tests with t.Run() for better coverage and clarity.",
    "outcome": "success"
  }
}
```

**Best Practices:**
- Explain the "why" not just the "what"
- Include specific examples or code snippets
- Note any prerequisites or constraints
- 200-500 characters recommended

---

### Invalid Confidence

**Error:** `ErrInvalidConfidence`

**Description:** Confidence score must be between 0.0 and 1.0.

**Causes:**
- Confidence value outside valid range
- Invalid number format
- Manual confidence override with bad value

**Resolution:**
```go
// Valid confidence values
confidence := 0.85  // 0.0 ≤ confidence ≤ 1.0
```

**Confidence Scoring:**
- Initial: 0.5 (neutral)
- Positive feedback: +0.1 (up to 1.0)
- Negative feedback: -0.15 (down to 0.0)
- Success outcome: +0.05
- Failure outcome: -0.10

**See Also:** [Memory Confidence Scoring](./mcp-tools.md#confidence-scoring)

---

### Invalid Outcome

**Error:** `ErrInvalidOutcome`

**Description:** Memory outcome must be "success" or "failure".

**Resolution:**
```json
{
  "tool": "memory_record",
  "arguments": {
    "project_id": "contextd",
    "title": "Memory title",
    "content": "Memory content",
    "outcome": "success"  // Must be "success" or "failure"
  }
}
```

**Valid Values:**
- `"success"` - Strategy worked, pattern is useful
- `"failure"` - Anti-pattern, avoid this approach

---

### Empty Project ID

**Error:** `ErrEmptyProjectID`

**Description:** Project ID is required for memory operations.

**Resolution:**
```json
{
  "tool": "memory_search",
  "arguments": {
    "project_id": "contextd",  // Required!
    "query": "debugging strategies"
  }
}
```

**Note:** Project ID typically corresponds to the repository path or name.

---

## Project Management Errors

Errors related to project CRUD operations.

### Project Not Found

**Error:** `ErrProjectNotFound`

**Description:** The requested project does not exist.

**Causes:**
- Invalid project ID
- Project was deleted
- Typo in project identifier

**Resolution:**
```bash
# List all projects
ctxd project list

# Get specific project
ctxd project get --id=<project_id>
```

---

### Project Already Exists

**Error:** `ErrProjectExists`

**Description:** A project with this path already exists.

**Causes:**
- Attempting to create duplicate project
- Project creation was retried

**Resolution:**
```bash
# Check existing projects
ctxd project list

# Get project by path
ctxd project get --path=/path/to/repo

# Use existing project instead of creating new one
```

---

### Invalid Project ID

**Error:** `ErrInvalidProjectID`

**Description:** Project ID is invalid or malformed.

**Causes:**
- Empty project ID
- Invalid UUID format
- Project ID contains invalid characters

**Resolution:**
- Use valid UUID format
- Get project ID from `project list` command
- Verify ID was copied correctly

---

### Invalid Project Name

**Error:** `ErrInvalidProjectName`

**Description:** Project name is invalid or empty.

**Resolution:**
```bash
# Valid project name
ctxd project create \
  --name="contextd" \
  --path=/Users/user/projects/contextd
```

**Validation:**
- Name must be non-empty
- No specific format requirements
- Recommended: Use repository name

---

### Invalid Project Path

**Error:** `ErrInvalidProjectPath`

**Description:** Project path is invalid or empty.

**Resolution:**
```bash
# Valid project path
ctxd project create \
  --name="contextd" \
  --path=/Users/user/projects/contextd  # Absolute path required
```

**Validation:**
- Path must be non-empty
- Should be absolute path
- Directory should exist (not strictly validated)

---

## Troubleshooting Errors

Errors related to error diagnosis and pattern matching.

### Empty Error Message

**Error:** `ErrEmptyErrorMessage`

**Description:** Error message is required for diagnosis.

**Resolution:**
```json
{
  "tool": "troubleshoot_diagnose",
  "arguments": {
    "error_message": "connection refused: dial tcp 127.0.0.1:6334",  // Required!
    "context": "Attempting to connect to Qdrant"
  }
}
```

---

### Invalid Confidence (Troubleshoot)

**Error:** `ErrInvalidConfidence`

**Description:** Pattern confidence score must be between 0.0 and 1.0.

**Causes:**
- Pattern validation failed
- Manual confidence override with bad value

**Resolution:**
```go
pattern := &troubleshoot.Pattern{
    ErrorType:   "connection_refused",
    Description: "Qdrant not responding",
    Solution:    "Ensure Qdrant is running",
    Confidence:  0.85,  // Must be 0.0-1.0
}
```

---

## Embeddings Errors

Errors related to embedding model loading and generation.

### Unsupported Platform

**Error:** `ErrUnsupportedPlatform`

**Description:** The current platform is not supported for local embeddings.

**Causes:**
- Running on unsupported OS/architecture
- ONNX runtime not available for platform
- CGO not enabled

**Resolution:**
1. **Use TEI (Text Embeddings Inference) provider instead:**
   ```bash
   docker run -e EMBEDDINGS_PROVIDER=tei \
     -e TEI_URL=http://localhost:8080 \
     contextd
   ```

2. **Check platform support:**
   - Supported: Linux (amd64, arm64), macOS (amd64, arm64)
   - Requires: CGO enabled during build

**See Also:** [Embeddings Configuration](../troubleshooting.md#onnx-runtime-errors)

---

### FastEmbed Not Available

**Error:** `ErrFastEmbedNotAvailable`

**Description:** FastEmbed is not available because binary was built without CGO.

**Causes:**
- Binary built with `CGO_ENABLED=0`
- No CGO compiler available at build time
- Cross-compilation without CGO support

**Resolution:**
1. **Use TEI provider:**
   ```bash
   docker run -e EMBEDDINGS_PROVIDER=tei \
     -e TEI_URL=http://localhost:8080 \
     contextd
   ```

2. **Rebuild with CGO:**
   ```bash
   CGO_ENABLED=1 go build -tags fastembed ./cmd/contextd
   ```

**See Also:** [Embeddings Providers](./mcp-tools.md#embeddings)

---

## Compression Errors

Errors related to context compression and A/B testing.

### Invalid Experiment ID

**Error:** `ErrInvalidExperimentID`

**Description:** Experiment ID cannot be empty.

**Resolution:**
```go
experiment := &compression.Experiment{
    ID:       "exp_20240115_compression",  // Required!
    Variants: []string{"extractive", "abstractive", "hybrid"},
}
```

---

### Insufficient Variants

**Error:** `ErrInsufficientVariants`

**Description:** Experiment must have at least 2 variants.

**Resolution:**
```go
experiment := &compression.Experiment{
    ID: "exp_compression",
    Variants: []string{
        "extractive",   // At least 2 variants required
        "abstractive",
    },
}
```

---

### Invalid Session ID (Compression)

**Error:** `ErrInvalidSessionID`

**Description:** Session ID cannot be empty for experiment assignment.

**Resolution:**
```go
assignment, err := manager.GetAssignment(ctx, "exp_123", "sess_abc")
// Session ID "sess_abc" is required
```

---

### Algorithm Not In Experiment

**Error:** `ErrAlgorithmNotInExp`

**Description:** The requested compression algorithm is not a variant in the experiment.

**Resolution:**
```go
// Ensure algorithm matches a variant
experiment := &compression.Experiment{
    ID:       "exp_compression",
    Variants: []string{"extractive", "abstractive"},
}

// This will fail:
manager.RecordResult(ctx, "exp_compression", "sess_123", "hybrid", result)
// "hybrid" is not in variants!
```

---

### Experiment Not Found

**Error:** `ErrExperimentNotFound`

**Description:** The requested experiment does not exist.

**Resolution:**
```bash
# List active experiments
ctxd compression experiments list

# Create experiment if needed
ctxd compression experiments create \
  --id=exp_compression \
  --variants=extractive,abstractive,hybrid
```

---

## Workflow Errors

Errors related to Temporal workflows (internal automation).

Workflow errors use severity levels instead of error codes:

| Severity | Description | Behavior |
|----------|-------------|----------|
| **Critical** | Workflow must fail | Recorded AND propagated |
| **High** | Major issue, can continue | Recorded but not propagated |
| **Low** | Minor issue | Logged as warning only |

### Workflow Error Structure

```go
type WorkflowError struct {
    Operation string        // e.g., "fetch_version_file"
    Severity  ErrorSeverity // critical, high, low
    Err       error         // underlying error
    Context   string        // additional context
}
```

### Critical Errors

**Examples:**
- Failed to fetch required files
- Invalid JSON in plugin.json
- Missing required resources
- Activity initialization failed

**Behavior:**
- Added to `result.Errors` slice
- Error returned to fail workflow
- Execution stops

---

### High Severity Errors

**Examples:**
- Failed to post GitHub comment
- Non-essential operation failed
- Failed with acceptable fallback

**Behavior:**
- Added to `result.Errors` slice
- Logged as error
- Workflow continues

---

### Low Severity Errors

**Examples:**
- Failed to remove old comment (might not exist)
- Cleanup operation failed
- Missing optional resource

**Behavior:**
- Logged as warning
- NOT added to `result.Errors`
- Workflow continues

**See Also:** [Workflow Documentation](../../internal/workflows/README.md)

---

## Error Helper Functions

### Is Functions

ContextD provides helper functions to categorize errors:

```go
// Context-Folding
folding.IsRetryable(err)       // true for rate limits, budget exhaustion
folding.IsNotFoundError(err)   // true for FOLD001, FOLD009
folding.IsUserError(err)       // true for validation errors
folding.IsSystemError(err)     // true for internal failures
folding.IsAuthorizationError(err) // true for FOLD022

// Multi-Tenancy
vectorstore.HasTenant(ctx)     // check if tenant context present
```

### Usage Examples

```go
// Retry logic
if err := branchOp(); err != nil {
    if folding.IsRetryable(err) {
        time.Sleep(2 * time.Second)
        return retry(branchOp)
    }
    return err
}

// User vs system errors
if err := validate(); err != nil {
    if folding.IsUserError(err) {
        return &APIError{
            Code:    400,
            Message: "Invalid input: " + err.Error(),
        }
    }
    return &APIError{
        Code:    500,
        Message: "Internal server error",
    }
}

// Authorization
if err := operation(); err != nil {
    if folding.IsAuthorizationError(err) {
        return &APIError{Code: 403, Message: "Forbidden"}
    }
    return err
}
```

---

## Common Resolution Patterns

### Pattern 1: Retry with Exponential Backoff

**Use for:** Rate limiting, transient failures, timeout errors

```go
func retryWithBackoff(operation func() error) error {
    backoff := 1 * time.Second
    maxRetries := 3

    for i := 0; i < maxRetries; i++ {
        err := operation()
        if err == nil {
            return nil
        }

        if !folding.IsRetryable(err) {
            return err // Don't retry non-retryable errors
        }

        time.Sleep(backoff)
        backoff *= 2
    }

    return errors.New("max retries exceeded")
}
```

---

### Pattern 2: Fail-Safe Error Handling

**Use for:** Security-critical operations, tenant isolation

```go
func getTenantData(ctx context.Context) ([]Data, error) {
    tenant, err := vectorstore.TenantFromContext(ctx)
    if err != nil {
        // Fail closed - no data returned on missing tenant
        return nil, err
    }

    return store.Search(ctx, tenant.TenantFilter())
}
```

---

### Pattern 3: Graceful Degradation

**Use for:** Non-critical operations with fallbacks

```go
func search(ctx context.Context, query string) ([]Result, error) {
    // Try semantic search first
    results, err := semanticSearch(ctx, query)
    if err != nil {
        log.Warn("Semantic search failed, falling back to keyword search", "error", err)
        // Fallback to keyword search
        return keywordSearch(ctx, query)
    }
    return results, nil
}
```

---

### Pattern 4: Resource Cleanup on Error

**Use for:** Branch operations, resource allocation

```go
func processWithBranch(sessionID, description, prompt string) error {
    branchID, err := branchCreate(sessionID, description, prompt, 4000)
    if err != nil {
        return err
    }

    // Ensure branch is cleaned up
    defer func() {
        if err := branchReturn(branchID, "Cleanup"); err != nil {
            log.Error("Failed to cleanup branch", "error", err)
        }
    }()

    // Process in branch
    return processBranchWork(branchID)
}
```

---

## Getting Help

If you encounter an error not documented here:

1. **Check system logs:**
   ```bash
   docker logs <container_id>
   journalctl -u contextd
   ```

2. **Run diagnostics:**
   ```bash
   curl http://localhost:9090/health
   curl http://localhost:9090/api/v1/status
   ```

3. **Search documentation:**
   - [Troubleshooting Guide](../troubleshooting.md)
   - [MCP Tools Reference](./mcp-tools.md)
   - [Context-Folding Spec](../spec/context-folding/SPEC.md)

4. **Report issues:**
   - GitHub: https://github.com/fyrsmithlabs/contextd/issues
   - Include error code, logs, and reproduction steps

---

## See Also

- [MCP Tools API Reference](./mcp-tools.md) - Complete tool documentation
- [Troubleshooting Guide](../troubleshooting.md) - Common issues and solutions
- [Context-Folding Specification](../spec/context-folding/SPEC.md) - Detailed feature spec
- [Multi-Tenancy Architecture](../CLAUDE.md#multi-tenancy-architecture) - Security model
- [Contributing](../../CONTRIBUTING.md) - How to contribute to ContextD
