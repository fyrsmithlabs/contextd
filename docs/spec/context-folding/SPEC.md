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
Branches SHOULD receive relevant ReasoningBank memories based on branch description. [NEEDS CLARIFICATION: injection budget allocation]

### FR-006: Nested Branches
The system MUST support nested branches up to configurable depth (default: 3).

### FR-007: Branch Timeout
Branches MUST have configurable timeout. System MUST force return on timeout.

### FR-008: Failed Branch Handling
Failed branches MUST return error context to parent. Failures SHOULD be candidates for anti-pattern extraction.

## Success Criteria

### SC-001: Context Compression
Branches MUST achieve >80% context compression compared to inline execution for file exploration tasks.

### SC-002: Latency Overhead
Branch creation and return MUST add <100ms latency overhead.

### SC-003: Memory Extraction Rate
>50% of successful branch patterns SHOULD be extractable to ReasoningBank.

### SC-004: Budget Compliance
100% of branches MUST complete within allocated budget (via return or forced termination).
