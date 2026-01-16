# Context-Folding Specification

**Feature**: Context-Folding (Layer 1)
**Status**: Implemented
**Created**: 2025-11-22
**Updated**: 2026-01-06

## Overview

Context-Folding provides active context management within a single agent session using `branch_create`, `branch_return`, and `branch_status` MCP tools. This enables 90%+ context compression by isolating subtask reasoning from the main context.

## User Scenarios

### P1: Developer Debugging with File Exploration

**Story**: As a developer debugging an issue, I want the agent to explore multiple files without polluting my main context, so that my primary task remains focused.

**Acceptance Criteria**:
```gherkin
Given an agent session with 16K tokens used
When the agent needs to search 10 files for a function definition
Then the agent creates a branch for file exploration
And the branch completes with a summary of findings
And the main context increases by <500 tokens (summary only)
And the main context does NOT contain the 10 file contents
```

**Edge Cases**:
- Branch exceeds its budget before finding target
- Multiple nested branches needed
- Branch encounters error and must return early

### P2: Research Task Isolation

**Story**: As a developer asking for implementation help, I want web searches and documentation lookups isolated, so that only relevant findings enter my context.

**Acceptance Criteria**:
```gherkin
Given an agent researching API documentation
When the agent performs 5 web fetches
Then each fetch occurs in a branch
And only extracted relevant information returns to main context
And main context contains <200 tokens per research task
```

### P3: Trial-and-Error Debugging

**Story**: As a developer with a failing test, I want the agent to try multiple fixes in isolation, so that failed attempts don't clutter my session.

**Acceptance Criteria**:
```gherkin
Given an agent debugging a test failure
When the agent tries 3 different fix approaches
Then each attempt runs in a separate branch
And only the successful approach returns to main context
And failed attempts are recorded for potential memory extraction
```

## Functional Requirements

### FR-001: Branch Creation
The system MUST create isolated context branches via `branch_create` MCP tool with the following parameters:
- `session_id` (required): Session identifier for this branch
- `description` (required): Brief description of what the branch will do (max 500 chars)
- `prompt` (optional): Detailed prompt/instructions for the branch (max 10,000 chars)
- `budget` (optional): Token budget for this branch (default: 8192, max: 32768)
- `timeout_seconds` (optional): Timeout in seconds (default: 300, max: 600)

### FR-002: Branch Isolation
Branches MUST NOT have access to parent context beyond the description and prompt provided. Each branch executes in an isolated goroutine with strict data separation.

### FR-003: Return Mechanism
Branches MUST return results to parent via `branch_return` MCP tool with the following parameters:
- `branch_id` (required): Branch ID to return from
- `message` (required): Result message/summary from the branch (max 50,000 chars)

### FR-003a: Branch Status Query
The system MUST provide a `branch_status` MCP tool to query branch state with the following parameters:
- `branch_id` (optional): Specific branch ID to check
- `session_id` (optional): Session ID to get active branch for
(Either `branch_id` or `session_id` is required)

Returns: branch status (created, active, completed, timeout, failed), depth, budget usage

### FR-004: Budget Enforcement
Each branch MUST have a token budget. The system MUST force return when budget is exhausted.

### FR-005: Memory Injection
**Status**: Implemented

The system provides memory injection capabilities via the `MemorySearcher` interface, which injects relevant ReasoningBank memories based on branch description. Memory injection:
- Uses injection budget ratio: 20% of branch budget (configurable via `injection_budget_ratio`)
- Returns maximum items: 10 memories (configurable via `memory_max_items`)
- Filters by minimum confidence: 0.7 (configurable via `memory_min_confidence`)

Implementation: `BranchManager.Create()` invokes the `MemorySearcher` when `BranchRequest.InjectMemories` is enabled, prepending relevant memories to the branch prompt within the allocated injection budget.

### FR-006: Nested Branches
The system MUST support nested branches up to configurable depth (default: 3).
- When max depth exceeded: reject with `ErrMaxDepthExceeded`
- Depth is zero-indexed (root session = depth 0, first branch = depth 1)

### FR-007: Branch Timeout
Branches MUST have configurable timeout. System MUST force return on timeout.
- Timeout enforcement via goroutine with context.WithTimeout
- Timeout goroutine MUST be cancelled on normal return

### FR-008: Failed Branch Handling
Failed branches MUST return error context to parent. Failures SHOULD be candidates for anti-pattern extraction.

### FR-009: Child Branch Cleanup
Before a branch can return, the system MUST force-return all active child branches recursively.
- Children are force-returned with reason "parent returning"
- Cleanup proceeds deepest-first (leaf branches first)
- Parent return blocks until all children completed

### FR-010: Session End Cleanup
On session end, the system MUST force-return all active branches for that session.
- Cleanup has bounded total timeout (default: 30 seconds)
- Branches not completed within timeout transition to "failed" state

## Security Requirements

### SEC-001: Input Validation
The system MUST validate all inputs to branch() and return():
- `description`: Max 500 characters, strip control characters
- `prompt`: Max 10,000 characters, strip control characters
- `message`: Max 50,000 characters, secret scrubbing required
- Invalid inputs rejected with 400 Bad Request

### SEC-002: Secret Scrubbing on Return
ALL content passed to `branch_return` MUST be scrubbed for secrets before reaching parent context.
- Uses existing gitleaks SDK integration via `SecretScrubber` interface
- Scrubbing failures MUST NOT leak unscrubbed content (fail closed)
- Implementation: `BranchManager.Return()` calls `m.scrubber.Scrub()` and returns `ErrScrubbingFailed` if scrubber is nil or scrubbing fails
- Scrubbed message is returned in `branchReturnOutput.Message`

### SEC-003: Rate Limiting
The system MUST enforce rate limits on branch creation:
- Per session: max 10 concurrent branches (configurable)
- Per instance: max 100 concurrent branches (configurable)
- Creation rate: max 5 branches/minute/session (configurable)
- Exceeded limits rejected with 429 Too Many Requests

### SEC-004: Authentication
Branch and return operations MUST validate session authentication:
- Session ID validated against active sessions
- JWT claims propagated to branches (no privilege escalation)
- Cross-session branch access prohibited

### SEC-005: Memory Injection Safety
Injected memories MUST be validated before injection:
- Memories from untrusted sources flagged
- Prompt injection patterns in memories logged and optionally blocked
- Memory provenance tracked (source session, user)

## MCP Tools

### branch_create

Creates a new isolated context branch for sub-task execution.

**Parameters:**
```json
{
  "session_id": "string (required)",      // Session identifier
  "description": "string (required)",     // Brief description (max 500 chars)
  "prompt": "string (optional)",          // Detailed instructions (max 10,000 chars)
  "budget": "number (optional)",          // Token budget (default: 8192, max: 32768)
  "timeout_seconds": "number (optional)"  // Timeout (default: 300, max: 600)
}
```

**Returns:**
```json
{
  "branch_id": "string",        // Unique branch identifier (format: "br_<uuid>")
  "budget_allocated": "number", // Actual budget allocated (capped by max)
  "depth": "number"             // Branch depth (0 = root level)
}
```

### branch_return

Returns from a context branch with results. All child branches are force-returned first.

**Parameters:**
```json
{
  "branch_id": "string (required)", // Branch ID to return from
  "message": "string (required)"    // Result message (max 50,000 chars, scrubbed)
}
```

**Returns:**
```json
{
  "success": "boolean",   // Whether return succeeded
  "tokens_used": "number", // Tokens consumed by the branch
  "message": "string"      // Scrubbed result message (secrets removed)
}
```

### branch_status

Queries the status of a specific branch or the active branch for a session.

**Parameters:**
```json
{
  "branch_id": "string (optional)",  // Specific branch to check
  "session_id": "string (optional)"  // Session to get active branch for
}
// Note: Either branch_id or session_id is required
```

**Returns:**
```json
{
  "branch_id": "string",         // Branch ID (if found)
  "session_id": "string",        // Session ID
  "status": "string",            // created | active | completed | timeout | failed
  "depth": "number",             // Branch depth
  "budget_used": "number",       // Tokens consumed
  "budget_total": "number",      // Total budget allocated
  "description": "string",       // Branch description
  "created_at": "timestamp",     // Creation time
  "completed_at": "timestamp?"   // Completion time (if completed)
}
```

**Returns empty/null if no branch found:**
```json
{
  "branch_id": null,
  "status": "No active branch found"
}
```

## Implementation

### Package Structure

The context-folding implementation is located in `internal/folding/`:

- `manager.go`: `BranchManager` orchestrates branch lifecycle
- `types.go`: Core types (`Branch`, `BranchRequest`, `BranchResponse`, `BranchStatus`)
- `interfaces.go`: Interfaces (`BranchRepository`, `SecretScrubber`, `MemorySearcher`, `SessionValidator`)
- `budget.go`: `BudgetTracker` manages token budgets and emits events
- `repository.go`: In-memory implementation of `BranchRepository`
- `errors.go`: Domain-specific errors
- `telemetry.go`: OpenTelemetry instrumentation
- `logging.go`: Structured logging

### Configuration

Default configuration values (`FoldingConfig`):

```go
DefaultBudget:            8192   // Default token budget if not specified
MaxBudget:                32768  // Maximum allowed budget
MaxDepth:                 3      // Maximum nesting depth
DefaultTimeoutSeconds:    300    // Default timeout (5 minutes)
MaxTimeoutSeconds:        600    // Maximum timeout (10 minutes)
InjectionBudgetRatio:     0.2    // 20% of budget for memory injection
MemoryMinConfidence:      0.7    // Minimum confidence for injected memories
MemoryMaxItems:           10     // Maximum memories to inject
MaxConcurrentPerSession:  10     // Max concurrent branches per session
MaxConcurrentPerInstance: 100    // Max concurrent branches per instance
```

### Token Budget Management

- **Allocation**: Budget allocated when branch is created via `BudgetTracker.Allocate()`
- **Tracking**: Token usage tracked via server-side MCP interceptor (agent cannot self-report)
- **Events**: `BudgetTracker` emits `BudgetWarningEvent` (80% usage) and `BudgetExhaustedEvent` (100%)
- **Enforcement**: `BranchManager` subscribes to budget events and force-returns branches on exhaustion
- **Independence**: Each branch has independent budget; no per-token propagation to ancestors

### Timeout Management

- **Implementation**: Each branch has a goroutine that watches for timeout
- **Cleanup**: Timeout goroutine is cancelled when branch returns normally
- **Force Return**: On timeout, branch is force-returned with status `BranchStatusTimeout`

### Child Branch Cleanup (FR-009)

Before a branch can return, all active child branches are force-returned:
1. Query `BranchRepository.ListByParent()` to find children
2. Recursively force-return children (deepest-first)
3. Log errors but continue parent return

### Session Validation (SEC-004)

- **Interface**: `SessionValidator` validates caller access to sessions
- **Default**: `PermissiveSessionValidator` allows all access (single-user deployments)
- **Strict**: `StrictSessionValidator` requires session ownership match
- **Implementation**: Checked in both `Create()` and `Return()` operations

## Success Criteria

### SC-001: Context Compression
**Status**: Verified

Branches MUST achieve >80% context compression compared to inline execution for file exploration tasks.

**Verification**: Tested with file exploration scenarios showing 90%+ compression ratio. Example: 10-file search (3.5K tokens consumed in branch) returned as ~50 token summary to main context.

### SC-002: Latency Overhead
Branch creation and return MUST add <100ms latency overhead.

### SC-003: Memory Extraction Rate
>50% of successful branch patterns SHOULD be extractable to ReasoningBank.

### SC-004: Budget Compliance
100% of branches MUST complete within allocated budget (via return or forced termination).

## Testing

The context-folding implementation has comprehensive test coverage:

- **Unit tests**: `internal/folding/*_test.go`
  - `manager_test.go`: Branch lifecycle, rate limiting, depth limits, timeout enforcement
  - `budget_test.go`: Budget allocation, tracking, event emission
  - `types_test.go`: Request validation, state transitions
  - `repository_test.go`: In-memory repository operations
  - `errors_test.go`: Error handling and edge cases

- **Integration tests**: `internal/mcp/tools_folding_test.go`
  - End-to-end MCP tool invocation via actual service
  - Branch creation, return, and status query flows
  - Secret scrubbing on return
  - Error handling and validation

Run tests:
```bash
go test ./internal/folding/... -v
go test ./internal/mcp/tools_folding_test.go -v
```

## References

- **Architecture**: `docs/spec/context-folding/ARCH.md` - System architecture and design decisions
- **Design**: `docs/spec/context-folding/DESIGN.md` - Detailed design documentation
- **Consensus Review**: `docs/spec/context-folding/CONSENSUS-REVIEW.md` - Architectural consensus decisions
- **Research**: [arXiv:2510.11967](https://arxiv.org/abs/2510.11967) - ByteDance research paper on context-folding (Oct 2025)
