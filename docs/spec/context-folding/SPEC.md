# Context-Folding Specification

**Feature**: Context-Folding (Layer 1)
**Status**: Draft
**Created**: 2025-11-22

## Overview

Context-Folding provides active context management within a single agent session using `branch()` and `return()` MCP tools. This enables 90%+ context compression by isolating subtask reasoning from the main context.

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
The system MUST create isolated context branches via `branch(description, prompt)` tool call.

### FR-002: Branch Isolation
Branches MUST NOT have access to parent context beyond the description and prompt provided.

### FR-003: Return Mechanism
Branches MUST return results to parent via `return(message)` tool call.

### FR-004: Budget Enforcement
Each branch MUST have a token budget. The system MUST force return when budget is exhausted.

### FR-005: Memory Injection
Branches SHOULD receive relevant ReasoningBank memories based on branch description.
- Memory injection budget: 20% of branch budget (configurable)
- Maximum items: 10 memories (configurable)
- Minimum confidence: 0.7 (configurable)

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
ALL content passed to `return(message)` MUST be scrubbed for secrets before reaching parent context.
- Uses existing gitleaks SDK integration
- Secrets replaced with `[REDACTED:secret_type]`
- Scrubbing failures MUST NOT leak unscrubbed content (fail closed)

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

## Success Criteria

### SC-001: Context Compression
Branches MUST achieve >80% context compression compared to inline execution for file exploration tasks.

### SC-002: Latency Overhead
Branch creation and return MUST add <100ms latency overhead.

### SC-003: Memory Extraction Rate
>50% of successful branch patterns SHOULD be extractable to ReasoningBank.

### SC-004: Budget Compliance
100% of branches MUST complete within allocated budget (via return or forced termination).
