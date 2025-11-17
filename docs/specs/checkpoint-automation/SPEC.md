# Checkpoint Automation Specification

**Epic**: 2.3 - Intelligent Checkpoint Orchestration
**Task**: Auto-checkpoint on `/clear` and auto-resume on session start
**Date**: 2025-01-10
**Status**: Specification
**Priority**: HIGH (blocking autonomous workflows)

---

## Problem Statement

**Current State** (Manual):
- User must remember to save checkpoint before `/clear`
- User must manually search and load checkpoint after session restart
- No automatic verification that `/clear` matches last checkpoint context
- Risk of losing context if user forgets to checkpoint

**Desired State** (Automatic):
- `/clear` automatically saves checkpoint before clearing
- Session start automatically resumes from last checkpoint for current project
- `/clear` verifies current work matches last checkpoint
- Configurable behavior (auto/prompt/manual)

---

## User Stories

### Story 1: Auto-Checkpoint on /clear
**As a** Claude Code user
**I want** `/clear` to automatically save a checkpoint before clearing
**So that** I never lose context when starting fresh

**Acceptance Criteria**:
- ✅ `/clear` triggers `checkpoint_save` before clearing
- ✅ Checkpoint summary auto-generated from recent work
- ✅ User sees confirmation: "Checkpoint saved: [summary]"
- ✅ Checkpoint ID provided for manual resume
- ✅ Can be configured: `CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=true|false|prompt`

### Story 2: Auto-Resume on Session Start
**As a** Claude Code user
**I want** new sessions to automatically load my last checkpoint
**So that** I can continue where I left off without manual steps

**Acceptance Criteria**:
- ✅ SessionStart hook searches for last checkpoint (current project)
- ✅ Automatically loads most recent checkpoint context
- ✅ Reports: "Resumed from checkpoint: [summary] (saved [time ago])"
- ✅ Can be configured: `CONTEXTD_AUTO_RESUME=true|false|prompt`
- ✅ Falls back gracefully if no checkpoint found

### Story 3: Verify Checkpoint Before /clear
**As a** Claude Code user
**I want** `/clear` to verify current work matches last checkpoint
**So that** I know I'm not losing unsaved progress

**Acceptance Criteria**:
- ✅ `/clear` compares current session to last checkpoint
- ✅ If different: Prompt "Work changed since last checkpoint. Save new checkpoint? (y/n)"
- ✅ If same: Proceed silently
- ✅ Shows diff summary: "Files changed: X, Lines added: Y"

### Story 4: Configurable Auto-Checkpoint
**As a** Claude Code user
**I want** to configure when auto-checkpoint happens
**So that** I control the automation level

**Acceptance Criteria**:
- ✅ Config file: `~/.config/contextd/config.json`
- ✅ Settings:
  - `auto_checkpoint_on_clear`: true | false | prompt (default: prompt)
  - `auto_resume_on_start`: true | false | prompt (default: true)
  - `checkpoint_threshold_percent`: 70-95 (default: 70)
  - `verify_before_clear`: true | false (default: true)

---

## Architecture

### Component 1: Hook System Enhancement

**File**: `pkg/hooks/hooks.go` (new package)

**Core Types**:
```go
type HookType string

const (
    HookSessionStart HookType = "session_start"
    HookSessionEnd   HookType = "session_end"
    HookBeforeClear  HookType = "before_clear"
    HookAfterClear   HookType = "after_clear"
    HookContextThreshold HookType = "context_threshold"
)

type HookConfig struct {
    AutoCheckpointOnClear bool   `json:"auto_checkpoint_on_clear"`
    AutoResumeOnStart     bool   `json:"auto_resume_on_start"`
    CheckpointThreshold   int    `json:"checkpoint_threshold_percent"`
    VerifyBeforeClear     bool   `json:"verify_before_clear"`
}

type HookManager struct {
    config     *HookConfig
    checkpoint *checkpoint.Service
    logger     *log.Logger
}

// Execute runs hooks for the given type
func (h *HookManager) Execute(ctx context.Context, hookType HookType, data map[string]interface{}) error
```

### Component 2: /clear Hook Handler

**File**: `pkg/mcp/hooks.go` (new)

**Implementation**:
```go
// HandleBeforeClear executes before /clear command
func (s *Server) HandleBeforeClear(ctx context.Context, projectPath string) error {
    config := s.hookManager.Config()

    // 1. Check if verification enabled
    if config.VerifyBeforeClear {
        lastCheckpoint, err := s.checkpoint.GetLatest(ctx, projectPath)
        if err != nil {
            return fmt.Errorf("failed to get last checkpoint: %w", err)
        }

        // Compare current session to checkpoint
        changed := s.compareSessionToCheckpoint(ctx, lastCheckpoint)

        if changed {
            // Prompt user or auto-save based on config
            if config.AutoCheckpointOnClear {
                return s.autoSaveCheckpoint(ctx, projectPath)
            } else {
                return fmt.Errorf("work changed since last checkpoint - save first")
            }
        }
    }

    // 2. Auto-checkpoint if enabled
    if config.AutoCheckpointOnClear {
        return s.autoSaveCheckpoint(ctx, projectPath)
    }

    return nil
}
```

### Component 3: SessionStart Hook Handler

**File**: `pkg/mcp/hooks.go`

**Implementation**:
```go
// HandleSessionStart executes on new session
func (s *Server) HandleSessionStart(ctx context.Context, projectPath string) error {
    config := s.hookManager.Config()

    if !config.AutoResumeOnStart {
        return nil
    }

    // 1. Search for last checkpoint
    checkpoints, err := s.checkpoint.Search(ctx, &checkpoint.SearchOptions{
        Query: "most recent",
        ProjectPath: projectPath,
        Limit: 1,
        SortBy: "created_at",
        SortOrder: "desc",
    })

    if err != nil || len(checkpoints) == 0 {
        s.logger.Info("no previous checkpoint found - starting fresh")
        return nil
    }

    lastCheckpoint := checkpoints[0]

    // 2. Report resume
    age := time.Since(lastCheckpoint.CreatedAt)
    s.logger.Info(fmt.Sprintf(
        "Resumed from checkpoint: %s (saved %s ago)",
        lastCheckpoint.Summary,
        formatDuration(age),
    ))

    // 3. Inject checkpoint context (summary + description)
    // This is returned as part of the MCP response
    return nil
}
```

### Component 4: Configuration Management

**File**: `~/.config/contextd/config.json`

**Format**:
```json
{
  "hooks": {
    "auto_checkpoint_on_clear": "prompt",
    "auto_resume_on_start": true,
    "checkpoint_threshold_percent": 70,
    "verify_before_clear": true
  },
  "checkpoint": {
    "auto_summary_enabled": true,
    "max_auto_summaries": 10,
    "cleanup_after_days": 30
  }
}
```

**Environment Variable Override**:
```bash
# Override config with env vars
CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=true
CONTEXTD_AUTO_RESUME_ON_START=false
CONTEXTD_CHECKPOINT_THRESHOLD=80
CONTEXTD_VERIFY_BEFORE_CLEAR=true
```

---

## Implementation Plan

### Phase 1: Hook System Foundation (Day 1)

**Files to Create**:
- `pkg/hooks/hooks.go` (~200 lines)
  - HookManager struct
  - HookConfig type
  - Execute() method
  - Config loading from file/env

**Files to Modify**:
- `cmd/contextd/main.go`
  - Initialize HookManager
  - Load config on startup

**Tests**:
- `pkg/hooks/hooks_test.go`
  - Test config loading
  - Test hook execution
  - Test env var override

### Phase 2: /clear Hook Integration (Day 2)

**Files to Create**:
- `pkg/mcp/hooks.go` (~300 lines)
  - HandleBeforeClear()
  - HandleAfterClear()
  - compareSessionToCheckpoint()
  - autoSaveCheckpoint()

**Files to Modify**:
- `pkg/mcp/server.go`
  - Add HookManager field
  - Call HandleBeforeClear() on /clear

**Tests**:
- `pkg/mcp/hooks_test.go`
  - Test auto-checkpoint on /clear
  - Test verification logic
  - Test prompt behavior

### Phase 3: SessionStart Hook Integration (Day 3)

**Files to Create**:
- None (add to existing `pkg/mcp/hooks.go`)

**Files to Modify**:
- `pkg/mcp/server.go`
  - Call HandleSessionStart() on connection
  - Return checkpoint context in handshake

**Tests**:
- `pkg/mcp/hooks_test.go`
  - Test auto-resume on start
  - Test fallback when no checkpoint
  - Test resume message format

### Phase 4: Configuration & Polish (Day 4)

**Files to Create**:
- `~/.config/contextd/config.json.example`
- `docs/guides/CHECKPOINT-AUTOMATION.md`

**Files to Modify**:
- `pkg/hooks/config.go`
  - Add validation
  - Add defaults
  - Add config reload

**Tests**:
- `pkg/hooks/config_test.go`
  - Test config validation
  - Test defaults
  - Test reload

---

## MCP Tool Updates

### New Tool: `checkpoint_configure`

**Purpose**: Configure auto-checkpoint behavior

**Input**:
```json
{
  "auto_checkpoint_on_clear": true,
  "auto_resume_on_start": true,
  "checkpoint_threshold": 70,
  "verify_before_clear": true
}
```

**Output**:
```json
{
  "success": true,
  "config": {
    "auto_checkpoint_on_clear": true,
    "auto_resume_on_start": true,
    "checkpoint_threshold_percent": 70,
    "verify_before_clear": true
  }
}
```

---

## User Experience

### Example 1: Auto-Checkpoint on /clear (First Time)

```
User: /clear

Claude: Checking for unsaved work...
Claude: Work changed since last checkpoint:
  - 3 files modified
  - 127 lines added
  - Last checkpoint: 45 minutes ago

Claude: Save checkpoint before clearing? (y/n)

User: y

Claude: Saving checkpoint...
Claude: ✅ Checkpoint saved: "Implemented auth middleware"
Claude: Checkpoint ID: abc123-def456
Claude: Clearing context...

User: [fresh session starts]
```

### Example 2: Auto-Resume on Session Start

```
[New session starts]

Claude: Resumed from checkpoint: "Implemented auth middleware" (saved 2 hours ago)
Claude:
Last session progress:
- Added JWT authentication
- Updated user service
- All tests passing

Claude: Ready to continue. What would you like to work on?
```

### Example 3: Prompt Mode

```
User: /clear

Claude: Auto-checkpoint enabled. Save current session?
  (y) Yes - save and clear
  (n) No - just clear
  (c) Cancel

User: y

Claude: ✅ Checkpoint saved
Claude: Clearing context...
```

---

## Success Criteria

- ✅ `/clear` automatically saves checkpoint (configurable)
- ✅ Session start automatically resumes from last checkpoint
- ✅ Verification before `/clear` prevents data loss
- ✅ Configuration via config file and env vars
- ✅ Test coverage ≥80%
- ✅ All existing tests pass
- ✅ Documentation complete
- ✅ CHANGELOG updated

---

## Configuration Examples

### Fully Automatic (Recommended)
```json
{
  "hooks": {
    "auto_checkpoint_on_clear": true,
    "auto_resume_on_start": true,
    "checkpoint_threshold_percent": 70,
    "verify_before_clear": false
  }
}
```

### Prompt Mode (Safe)
```json
{
  "hooks": {
    "auto_checkpoint_on_clear": "prompt",
    "auto_resume_on_start": true,
    "checkpoint_threshold_percent": 70,
    "verify_before_clear": true
  }
}
```

### Manual Mode (Current Behavior)
```json
{
  "hooks": {
    "auto_checkpoint_on_clear": false,
    "auto_resume_on_start": false,
    "checkpoint_threshold_percent": 70,
    "verify_before_clear": false
  }
}
```

---

## Related Documents

- **AUTO-CHECKPOINT-SYSTEM.md** - User guide for auto-checkpoint features
- **Epic 2.3** - Parent task in roadmap
- **Task #8** - TaskMaster task for this feature
- **CLAUDE.md** - Context management instructions

---

## Next Steps

1. **Review this spec** - Validate requirements and architecture
2. **Expand Task #8** - Break down into subtasks in TaskMaster
3. **Implement Phase 1** - Hook system foundation
4. **Integrate with Claude Code** - Test `/clear` and session start hooks
5. **Document** - Update AUTO-CHECKPOINT-SYSTEM.md with new features
