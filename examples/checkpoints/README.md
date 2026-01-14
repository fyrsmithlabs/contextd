# Checkpoints Example

This example demonstrates the checkpoint lifecycle in contextd: **save → list → resume**. Checkpoints enable you to save session state at key points and resume later, managing token budgets and recovering from interruptions.

## Overview

Checkpoints provide context management by:

1. **Save** context snapshots at critical points (manual or auto-triggered)
2. **List** available checkpoints for a session or project
3. **Resume** from a checkpoint at different granularity levels
4. **Manage** token budgets by resuming at appropriate detail levels

This is how contextd enables long-running sessions without hitting token limits.

## The Pattern

```
┌─────────────────────────────────────────────────────────┐
│               Long-Running Session                       │
└────────────────────┬───────────────────────────────────┘
                     │
                     ▼
         ┌──────────────────────┐
         │  1. Save Checkpoint  │  ← checkpoint_save
         │  (Capture state)     │      (manual or auto)
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  Context fills up... │
         │  Session interrupted │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  2. List Checkpoints │  ← checkpoint_list
         │  (Find saved states) │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  3. Choose Level     │  ← summary, context, or full
         │  (Based on budget)   │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  4. Resume Session   │  ← checkpoint_resume
         │  (Restore context)   │
         └──────────────────────┘
```

## Resume Levels

Checkpoints support three resume levels for token budget control:

| Level | Content | Use Case | Token Cost |
|-------|---------|----------|------------|
| **summary** | Brief summary only | Quick reminder, tight budget | ~50-200 tokens |
| **context** | Summary + key context | Most common, balanced | ~500-2000 tokens |
| **full** | Complete session state | Deep continuation needed | Full checkpoint size |

## Quick Start

### Prerequisites

- contextd running (either as MCP server or standalone)
- Go 1.25+ installed

### Run the Example

```bash
# From the examples/checkpoints directory
go run main.go

# Or build first
go build -o checkpoints
./checkpoints
```

### Expected Output

```
Checkpoints Example - Demonstrating save->list->resume workflow
===============================================================

Step 1: Simulating work session (Task A)...
✓ Completed Task A (context: 1200 tokens)

Step 2: Saving checkpoint "After Task A"...
✓ Saved checkpoint: "After Task A" (ID: abc12345)

Step 3: Continuing work (Task B)...
✓ Completed Task B (context: 2800 tokens)

Step 4: Saving checkpoint "After Task B"...
✓ Saved checkpoint: "After Task B" (ID: def67890)

Step 5: Listing available checkpoints...
Found 2 checkpoints:
  - [abc12345] After Task A (1200 tokens, auto: false)
  - [def67890] After Task B (2800 tokens, auto: false)

Step 6: Simulating session interruption...
Session interrupted! Context lost.

Step 7: Resuming from checkpoint at 'context' level...
✓ Resumed checkpoint: "After Task B"
✓ Restored context (ID: def67890, tokens: ~900)

Content Preview:
---
Completed tasks A and B. Ready for Task C.
Key context: error handling patterns established, API client configured.
---

Session successfully resumed! Continuing from checkpoint.
```

## How It Works

### 1. Save Checkpoints

Save session state at strategic points:

```go
// Create checkpoint request
saveReq := &checkpoint.SaveRequest{
    SessionID:   sessionID,
    TenantID:    tenant,
    TeamID:      "demo-team",
    ProjectID:   projectID,
    ProjectPath: "/path/to/project",
    Name:        "After Task A",
    Description: "Completed authentication setup",
    Summary:     "Implemented OAuth flow, configured client",
    Context:     "Relevant code snippets and decisions...",
    FullState:   "Complete conversation history...",
    TokenCount:  1200,
    Threshold:   0.5, // Context is 50% full
    AutoCreated: false, // Manual checkpoint
}

// Save the checkpoint
cp, err := service.Save(ctx, saveReq)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Saved checkpoint: %s (ID: %s)\n", cp.Name, cp.ID)
```

**When to save?**
- ✅ After completing major tasks
- ✅ Before risky operations
- ✅ When context reaches thresholds (25%, 50%, 75%, 90%)
- ✅ End of work session
- ❌ Too frequently (checkpoint bloat)

### 2. List Checkpoints

Find available checkpoints for resuming:

```go
// List all checkpoints for current session
listReq := &checkpoint.ListRequest{
    SessionID: sessionID,
    TenantID:  tenant,
    TeamID:    "demo-team",
    ProjectID: projectID,
    Limit:     10,
    AutoOnly:  false, // Include both manual and auto checkpoints
}

checkpoints, err := service.List(ctx, listReq)
if err != nil {
    log.Fatal(err)
}

// Display available checkpoints
for _, cp := range checkpoints {
    fmt.Printf("- [%s] %s (%d tokens, auto: %v)\n",
        cp.ID[:8], cp.Name, cp.TokenCount, cp.AutoCreated)
}
```

**Filtering options:**
- By session ID (current session)
- By project path (all sessions in this project)
- Auto-created only (exclude manual checkpoints)
- Limit results (most recent N checkpoints)

### 3. Resume from Checkpoint

Restore session state at chosen granularity:

```go
// Resume at 'context' level (most common)
resumeReq := &checkpoint.ResumeRequest{
    CheckpointID: checkpointID,
    TenantID:     tenant,
    TeamID:       "demo-team",
    ProjectID:    projectID,
    Level:        checkpoint.ResumeContext,
}

response, err := service.Resume(ctx, resumeReq)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Resumed checkpoint: %s\n", response.Checkpoint.Name)
fmt.Printf("Restored %d tokens of context\n", response.TokenCount)
fmt.Printf("Content:\n%s\n", response.Content)
```

**Resume level selection:**

```go
// Summary: Just a brief reminder (50-200 tokens)
Level: checkpoint.ResumeSummary

// Context: Summary + key context (500-2000 tokens) - RECOMMENDED
Level: checkpoint.ResumeContext

// Full: Complete session state (original token count)
Level: checkpoint.ResumeFull
```

## Real-World Usage

### Example 1: Token Budget Management

```go
// Session approaching token limit
currentTokens := 48000  // 48k / 50k used
contextThreshold := 0.96

// Save full checkpoint
saveReq := &checkpoint.SaveRequest{
    SessionID:   sessionID,
    TenantID:    tenant,
    TeamID:      "demo-team",
    ProjectID:   projectID,
    Name:        "Before refactoring",
    Summary:     "Analyzed codebase, identified 3 services to refactor",
    Context:     "Key findings: service A couples to B, recommend extract interface...",
    FullState:   entireConversationHistory,
    TokenCount:  48000,
    Threshold:   contextThreshold,
    AutoCreated: true, // Auto-triggered by threshold
}

cp, _ := service.Save(ctx, saveReq)

// Clear context and resume at summary level (save 47.8k tokens!)
// ... clear context ...

resumeReq := &checkpoint.ResumeRequest{
    CheckpointID: cp.ID,
    TenantID:     tenant,
    TeamID:       "demo-team",
    ProjectID:    projectID,
    Level:        checkpoint.ResumeSummary, // Only 200 tokens instead of 48k
}

response, _ := service.Resume(ctx, resumeReq)
// Continue with 200 tokens of context instead of 48k
```

### Example 2: Multi-Day Work Sessions

```go
// Day 1: End of work session
saveReq := &checkpoint.SaveRequest{
    SessionID:   "day1-session",
    TenantID:    tenant,
    TeamID:      "demo-team",
    ProjectID:   projectID,
    Name:        "End of Day 1",
    Summary:     "Implemented user service, added tests, documented API",
    Context:     "User service handles CRUD, JWT auth, rate limiting...",
    FullState:   fullConversation,
    TokenCount:  12000,
    AutoCreated: false,
}

checkpoint1, _ := service.Save(ctx, saveReq)
// Log out, go home

// Day 2: Resume work
listReq := &checkpoint.ListRequest{
    ProjectID:   projectID,
    TenantID:    tenant,
    TeamID:      "demo-team",
    ProjectPath: "/home/user/project",
    Limit:       5,
}

checkpoints, _ := service.List(ctx, listReq)
// Find yesterday's checkpoint: "End of Day 1"

resumeReq := &checkpoint.ResumeRequest{
    CheckpointID: checkpoints[0].ID,
    TenantID:     tenant,
    TeamID:       "demo-team",
    ProjectID:    projectID,
    Level:        checkpoint.ResumeContext, // Get summary + key context
}

response, _ := service.Resume(ctx, resumeReq)
// Pick up where you left off with relevant context
```

### Example 3: Before Risky Operations

```go
// Before attempting complex refactor
saveReq := &checkpoint.SaveRequest{
    SessionID:   sessionID,
    TenantID:    tenant,
    TeamID:      "demo-team",
    ProjectID:   projectID,
    Name:        "Before database migration",
    Description: "Checkpoint before attempting schema changes",
    Summary:     "Current state: DB schema v2, 3 services connected",
    Context:     "Migration plan: add user_roles table, foreign keys...",
    FullState:   fullState,
    TokenCount:  8000,
    AutoCreated: false,
}

safepointCP, _ := service.Save(ctx, saveReq)

// Attempt risky operation
err := attemptDatabaseMigration()
if err != nil {
    // Something went wrong - resume from safe point
    resumeReq := &checkpoint.ResumeRequest{
        CheckpointID: safepointCP.ID,
        TenantID:     tenant,
        TeamID:       "demo-team",
        ProjectID:    projectID,
        Level:        checkpoint.ResumeFull, // Need full context to retry
    }

    response, _ := service.Resume(ctx, resumeReq)
    // Back to safe state, try different approach
}
```

## Integration with MCP

When running as an MCP server, these operations are exposed as tools:

| Go Method | MCP Tool | Purpose |
|-----------|----------|---------|
| `service.Save()` | `checkpoint_save` | Save session state |
| `service.List()` | `checkpoint_list` | List available checkpoints |
| `service.Resume()` | `checkpoint_resume` | Restore from checkpoint |

Example MCP tool call:

```json
{
  "tool": "checkpoint_save",
  "arguments": {
    "session_id": "session-123",
    "tenant_id": "myuser",
    "team_id": "myteam",
    "project_id": "myproject",
    "name": "After Task A",
    "summary": "Completed authentication implementation",
    "context": "OAuth flow configured, JWT tokens, refresh logic...",
    "full_state": "Complete conversation...",
    "token_count": 1200,
    "auto_created": false
  }
}
```

## Auto-Checkpointing

contextd can automatically save checkpoints at configured thresholds:

```yaml
# config.yaml
checkpoint:
  auto_checkpoint_thresholds: [0.25, 0.5, 0.75, 0.9]
  max_checkpoints_per_session: 10
```

When context reaches 25%, 50%, 75%, or 90% full, contextd automatically saves a checkpoint with `auto_created: true`.

**Benefits:**
- Never lose progress due to token limits
- Always have recovery points
- Seamless context management

**Filtering:**
```go
// List only manual checkpoints
listReq := &checkpoint.ListRequest{
    SessionID: sessionID,
    TenantID:  tenant,
    TeamID:    "demo-team",
    ProjectID: projectID,
    AutoOnly:  false, // Set to true for auto checkpoints only
}
```

## Best Practices

### Do's ✅

1. **Save at task boundaries** - After completing significant work
2. **Use descriptive names** - "After API implementation" not "checkpoint1"
3. **Save before risky operations** - Migrations, refactors, experiments
4. **Resume at 'context' level** - Best balance of context vs tokens
5. **Clean up old checkpoints** - Don't accumulate hundreds of checkpoints
6. **Leverage auto-checkpoints** - Let threshold-based saving protect you

### Don'ts ❌

1. **Don't save too frequently** - Every checkpoint costs storage
2. **Don't omit summaries** - Summary is crucial for 'summary' resume level
3. **Don't always resume 'full'** - Wastes tokens; 'context' is usually enough
4. **Don't skip manual checkpoints** - Auto checkpoints don't capture task semantics
5. **Don't forget tenant/team/project IDs** - Required for multi-tenancy

## Checkpoint Limits

Default limits (configurable):

| Limit | Default | Purpose |
|-------|---------|---------|
| `MaxCheckpointsPerSession` | 10 | Prevent checkpoint bloat |
| `AutoCheckpointThresholds` | [0.25, 0.5, 0.75, 0.9] | When to auto-save |
| `VectorSize` | 384 | Embedding dimension |

When max is reached, oldest checkpoints are automatically deleted.

## Troubleshooting

### "Checkpoint not found" on resume

**Cause**: Checkpoint ID is invalid or checkpoint was deleted.

**Solution**:
```go
// List checkpoints first to verify ID
checkpoints, _ := service.List(ctx, listReq)
for _, cp := range checkpoints {
    fmt.Printf("Available: %s\n", cp.ID)
}
```

### "Tenant ID not configured"

**Cause**: Checkpoint service requires tenant ID for multi-tenant isolation.

**Solution**:
```go
// Always provide tenant, team, and project IDs
saveReq := &checkpoint.SaveRequest{
    TenantID:  "myuser",      // Required
    TeamID:    "myteam",      // Required
    ProjectID: "myproject",   // Required
    // ... other fields
}
```

### Resume returns unexpected content

**Cause**: Wrong resume level selected, or checkpoint doesn't have expected content.

**Solution**:
```go
// Check checkpoint before resuming
cp, _ := service.Get(ctx, tenant, team, project, checkpointID)
fmt.Printf("Summary length: %d\n", len(cp.Summary))
fmt.Printf("Context length: %d\n", len(cp.Context))
fmt.Printf("Full state length: %d\n", len(cp.FullState))

// Choose appropriate level based on available content
if len(cp.Context) > 0 {
    level = checkpoint.ResumeContext
} else {
    level = checkpoint.ResumeSummary
}
```

### Checkpoints consuming too much storage

**Cause**: Too many checkpoints or very large full states.

**Solution**:
1. Reduce `MaxCheckpointsPerSession` in config
2. Delete old checkpoints manually
3. Compress full state before saving
4. Use fewer auto-checkpoint thresholds

```go
// Delete old checkpoint
err := service.Delete(ctx, tenant, team, project, oldCheckpointID)
if err != nil {
    log.Printf("Failed to delete: %v", err)
}
```

## Advanced: Checkpoint Metadata

Add custom metadata to checkpoints for filtering and organization:

```go
saveReq := &checkpoint.SaveRequest{
    // ... standard fields ...
    Metadata: map[string]string{
        "phase": "implementation",
        "sprint": "sprint-5",
        "feature": "user-auth",
        "risk_level": "low",
    },
}
```

This metadata is preserved and can be used for:
- Organizing checkpoints by project phase
- Filtering checkpoints by feature
- Tracking checkpoint creation context

## Next Steps

- **Session Lifecycle**: See `examples/session-lifecycle/` for memory search/record patterns
- **Remediation**: See `examples/remediation/` for error pattern tracking
- **Repository Indexing**: See `examples/repository-indexing/` for semantic code search
- **Context-Folding**: See `examples/context-folding/` for isolated subtask execution

## See Also

- Checkpoint spec: `docs/spec/checkpoint/`
- Checkpoint service: `internal/checkpoint/service.go`
- MCP handlers: `internal/mcp/handlers/checkpoint.go`
- Integration tests: `test/integration/framework/workflow.go`
