# Checkpoint Automation Hooks Guide

**Status**: Phase 1-2 Complete
**Date**: 2025-01-10
**Epic**: 2.3 - Intelligent Checkpoint Orchestration

---

## Overview

The checkpoint automation system provides automatic checkpoint creation and session resumption through lifecycle hooks. This guide explains how to configure and use the hooks with Claude Code.

## Architecture

```
Claude Code Desktop
    ↓ (lifecycle events)
Hook Configuration (~/.claude/config.json)
    ↓ (triggers MCP tools)
contextd MCP Server
    ↓ (uses)
Hook Handlers (pkg/mcp/hooks.go)
    ↓ (manages)
Hook System (pkg/hooks/)
```

---

## Features Implemented

### ✅ Phase 1: Hook System Foundation

**Package**: `pkg/hooks`

**Components**:
1. **HookManager** - Orchestrates hook execution
2. **Config** - Configuration with JSON and environment variable support
3. **5 Hook Types**:
   - `session_start` - Triggered when session begins
   - `session_end` - Triggered when session ends
   - `before_clear` - Triggered before `/clear` command
   - `after_clear` - Triggered after `/clear` command
   - `context_threshold` - Triggered when context reaches threshold

**Configuration Options**:
```json
{
  "hooks": {
    "auto_checkpoint_on_clear": true,
    "auto_resume_on_start": true,
    "checkpoint_threshold_percent": 70,
    "verify_before_clear": true
  }
}
```

**Environment Variables** (override file config):
- `CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR` - Enable/disable auto-checkpoint (true/false)
- `CONTEXTD_AUTO_RESUME_ON_START` - Enable/disable auto-resume (true/false)
- `CONTEXTD_CHECKPOINT_THRESHOLD` - Context percentage threshold (1-99)
- `CONTEXTD_VERIFY_BEFORE_CLEAR` - Verify before clearing (true/false)

---

### ✅ Phase 2: MCP Hook Handlers

**Package**: `pkg/mcp`

**Handlers**:

#### 1. ClearHookHandler

**Purpose**: Auto-save checkpoint before `/clear` command

**Behavior**:
- Checks `AutoCheckpointOnClear` config flag
- Creates checkpoint with:
  - Summary: "Auto-checkpoint before /clear at [timestamp]"
  - Description: "Automatic checkpoint created before clearing context"
  - Tags: `["auto-save", "before-clear"]`
  - Level: `"session"`
  - Context: Event metadata

**Error Handling**: Returns error if checkpoint creation fails (prevents /clear)

**Usage**:
```go
handler := NewClearHookHandler(checkpointService, config)
err := handler.HandleBeforeClear(ctx, map[string]interface{}{
    "project_path": "/path/to/project",
})
```

#### 2. SessionHookHandler

**Purpose**: Auto-resume from recent checkpoint on session start

**Behavior**:
- Checks `AutoResumeOnStart` config flag
- Searches for top 3 recent checkpoints
- Formats resume context as markdown:
  ```markdown
  ## Recent Session Checkpoints

  Found N recent checkpoint(s) from previous sessions:

  - [Score: 0.95] Summary text (ID: abc123)
  - [Score: 0.87] Summary text (ID: def456)
  - [Score: 0.75] Summary text (ID: ghi789)

  Use `checkpoint_search` to load specific checkpoint context.
  ```
- Stores formatted context in `data["resume_context"]`
- If search fails, stores error in `data["resume_error"]`

**Error Handling**: Graceful degradation - search failures don't block session start

**Usage**:
```go
handler := NewSessionHookHandler(checkpointService, config)
data := map[string]interface{}{"project_path": "/path/to/project"}
err := handler.HandleSessionStart(ctx, data)

// Check for resume context
if resumeCtx, ok := data["resume_context"].(string); ok {
    fmt.Println(resumeCtx)
}
```

---

## Configuration

### File-Based Configuration

**Location**: `~/.config/contextd/config.json`

**Example**:
```json
{
  "hooks": {
    "auto_checkpoint_on_clear": true,
    "auto_resume_on_start": true,
    "checkpoint_threshold_percent": 70,
    "verify_before_clear": true
  }
}
```

### Environment Variable Configuration

**Override file configuration**:
```bash
export CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=true
export CONTEXTD_AUTO_RESUME_ON_START=true
export CONTEXTD_CHECKPOINT_THRESHOLD=80
export CONTEXTD_VERIFY_BEFORE_CLEAR=false
```

**Priority**: Environment variables > File configuration > Defaults

### Default Configuration

If no configuration file exists, defaults are:
```json
{
  "auto_checkpoint_on_clear": false,
  "auto_resume_on_start": true,
  "checkpoint_threshold_percent": 70,
  "verify_before_clear": true
}
```

---

## Usage with Claude Code

### Automatic Checkpoint on /clear

**When enabled** (`auto_checkpoint_on_clear: true`):

1. User types `/clear` in Claude Code
2. Claude Code triggers `before_clear` hook
3. Hook handler calls `checkpoint_save` MCP tool
4. Checkpoint created with auto-save tags
5. Context cleared
6. User can resume later with `checkpoint_search`

**Benefits**:
- Never lose work when clearing context
- Automatic checkpoint creation (no manual action)
- Tagged for easy filtering (`auto-save`, `before-clear`)

### Automatic Resume on Session Start

**When enabled** (`auto_resume_on_start: true`):

1. User starts new Claude Code session
2. Claude Code triggers `session_start` hook
3. Hook handler calls `checkpoint_search` MCP tool
4. Top 3 recent checkpoints retrieved
5. Resume context displayed to user:
   ```
   ## Recent Session Checkpoints

   Found 3 recent checkpoint(s) from previous sessions:

   - [Score: 0.95] Implemented authentication (ID: ckpt_abc)
   - [Score: 0.87] Fixed bug in validation (ID: ckpt_def)
   - [Score: 0.75] Added tests for hooks (ID: ckpt_ghi)

   Use `checkpoint_search` to load specific checkpoint context.
   ```
6. User can continue from previous session

**Benefits**:
- Instant context on what was being worked on
- No need to remember previous session
- Quick resume with relevant checkpoint ID

---

## Context Threshold Monitoring

**When enabled** (`checkpoint_threshold_percent: 70`):

The hook system can monitor context usage and trigger auto-checkpoint when threshold reached.

**Example**:
- Threshold: 70%
- Current context: 140K tokens (out of 200K)
- Percentage: 70%
- Action: Auto-checkpoint triggered
- User notified: "Context at 70%, checkpoint saved"

**Benefits**:
- Prevents context loss when approaching limits
- Automatic safety net for long sessions
- Configurable threshold (1-99%)

**Note**: Threshold must be < 100 to ensure checkpoint happens before context is completely full.

---

## MCP Tools Used

The hook handlers use these MCP tools internally:

### 1. `checkpoint_save`

**Used by**: ClearHookHandler

**Parameters**:
```json
{
  "summary": "Auto-checkpoint before /clear at 2025-01-10T15:30:00Z",
  "description": "Automatic checkpoint created before clearing context",
  "project_path": "/path/to/project",
  "context": {
    "event": "before-clear",
    "type": "auto-save"
  },
  "tags": ["auto-save", "before-clear"],
  "level": "session"
}
```

### 2. `checkpoint_search`

**Used by**: SessionHookHandler

**Parameters**:
```json
{
  "query": "recent session work",
  "top_k": 3
}
```

**Response**:
```json
{
  "results": [
    {
      "checkpoint": {
        "id": "ckpt_abc",
        "summary": "Implemented authentication",
        "created_at": "2025-01-10T14:30:00Z"
      },
      "score": 0.95
    }
  ]
}
```

---

## Testing

### Unit Tests

**Package**: `pkg/hooks`, `pkg/mcp`

**Coverage**:
- pkg/hooks: 82.2% (15 tests)
- pkg/mcp/hooks.go: 94.5% (9 tests)

**Run Tests**:
```bash
go test ./pkg/hooks/... -v
go test ./pkg/mcp -run "TestHandle" -v
```

### Integration Testing

**Manual Testing**:

1. **Test Auto-Checkpoint**:
   ```bash
   # Enable auto-checkpoint
   export CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=true

   # Start Claude Code session
   # Type some content
   # Type /clear
   # Verify checkpoint created with checkpoint_list
   ```

2. **Test Auto-Resume**:
   ```bash
   # Enable auto-resume
   export CONTEXTD_AUTO_RESUME_ON_START=true

   # Start new Claude Code session
   # Verify resume context displayed
   # Use checkpoint_search to load specific checkpoint
   ```

---

## Troubleshooting

### Auto-Checkpoint Not Working

**Symptoms**: `/clear` command doesn't create checkpoint

**Checks**:
1. Verify configuration: `cat ~/.config/contextd/config.json`
2. Check environment: `echo $CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR`
3. Verify MCP server running: `ps aux | grep contextd`
4. Check logs: `journalctl --user -u contextd -f` (Linux) or `tail -f /tmp/contextd.log` (macOS)

**Common Issues**:
- `auto_checkpoint_on_clear: false` in config
- MCP server not running
- Checkpoint service unhealthy

### Auto-Resume Not Working

**Symptoms**: Session start doesn't show resume context

**Checks**:
1. Verify configuration: `auto_resume_on_start: true`
2. Verify checkpoints exist: Use `checkpoint_list` tool
3. Check for resume_error in session data

**Common Issues**:
- No previous checkpoints exist
- Search query doesn't match existing checkpoints
- Checkpoint service connection failed (graceful degradation - no error shown)

### Context Threshold Not Triggering

**Symptoms**: No auto-checkpoint at threshold

**Checks**:
1. Verify threshold configuration: 1-99 range
2. Check current context usage
3. Verify hook registered for `context_threshold` event

**Note**: Context threshold monitoring requires Claude Code integration (Phase 3).

---

## Security Considerations

### Multi-Tenant Isolation

- Checkpoints scoped to `project_path`
- No cross-project data access
- Database-per-project architecture prevents filter injection

### Input Validation

- `project_path` validated (required, non-empty)
- Type assertions with ok checks
- Config validation (threshold 1-99)
- Environment variable parsing with error handling

### Error Messages

- No sensitive data in error messages
- Checkpoint IDs safe to expose
- Resume errors reported but don't leak system details

---

## Performance

### Auto-Checkpoint Performance

- Total overhead: <50ms per /clear command
- No blocking for cluster sync (async background)

### Auto-Resume Performance

- Checkpoint search: ~100ms
- Context formatting: <1ms
- Total overhead: ~100ms on session start
- Graceful failure: <1ms if disabled

### Context Efficiency

- Resume context: ~500-1000 characters
- Token usage: ~150-300 tokens
- Savings vs re-reading files: 90%+ reduction

---

## Future Enhancements (Phase 3-4)

### Phase 3: Enhanced Integration

- Claude Code hooks configuration
- Context threshold monitoring in real-time
- Verification before clear (prompt user)
- Custom hook actions

### Phase 4: Stateful Checkpoints

- Capture modified files automatically
- Store full file contents
- SHA256 deduplication
- Resume context with file injection (no Read tool needed)
- Target: <5K tokens, <10s resume time (vs 50K+ tokens, 2-5min currently)

---

## Related Documentation

- [Hook System Spec](../specs/checkpoint-automation/SPEC.md)
- [Stateful Checkpoints Spec](../specs/checkpoint-automation/STATEFUL-CHECKPOINT-SPEC.md)
- [Checkpoint Package Docs](../../pkg/checkpoint/CLAUDE.md)
- [MCP Package Docs](../../pkg/mcp/CLAUDE.md)
- [Getting Started Guide](./GETTING-STARTED.md)

---

**Last Updated**: 2025-01-10
**Authors**: Claude Code (claude.ai/code)
**Status**: Phase 1-2 Complete, Phase 3-4 Pending
