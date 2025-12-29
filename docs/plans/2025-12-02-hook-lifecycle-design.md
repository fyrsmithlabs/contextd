# Hook Lifecycle Design

**Status**: Approved
**Created**: 2025-12-02
**Author**: Claude + dahendel

---

## Problem Statement

contextd has hook infrastructure (`internal/hooks/`) but nothing triggers it. The Distiller exists but is never called. This means:

- No automatic learning from sessions
- No workflow awareness
- "Self-improving" claim is aspirational, not functional

**Meta-irony**: This design session itself demonstrated the problem. Previous sessions covered the skills system and hook design, but that context wasn't available - we re-discovered it.

---

## Design Decisions

| Component | Decision |
|-----------|----------|
| Summary generation | Claude-generated (skills enforce behavior) |
| `session_end` input | Structured required (task/approach/outcome/tags) + optional notes |
| `session_start` behavior | Prompt for checkpoint resume → memory prime top 3 |
| `context_threshold` trigger | HTTP primary, MCP fallback, configurable |
| Enforcement | Skill teaches behavior + PreCompact hook backstop |
| Service access | Registry interface pattern |

---

## MCP Tools

Three new tools in `internal/mcp/tools.go`:

### `session_start`

```go
type sessionStartInput struct {
    ProjectID string `json:"project_id" jsonschema:"required"`
    SessionID string `json:"session_id" jsonschema:"required"`
}

type sessionStartOutput struct {
    Checkpoint *CheckpointSummary `json:"checkpoint,omitempty"` // if recent exists
    Memories   []MemorySummary    `json:"memories"`             // top 3 relevant
    Resumed    bool               `json:"resumed"`              // user chose to resume
}
```

**Behavior**:
1. Check for recent checkpoints in project
2. If found: return checkpoint summary (agent prompts user to resume)
3. Memory prime: search memories for project, surface top 3

### `session_end`

```go
type sessionEndInput struct {
    ProjectID string   `json:"project_id" jsonschema:"required"`
    SessionID string   `json:"session_id" jsonschema:"required"`
    Task      string   `json:"task" jsonschema:"required"`
    Approach  string   `json:"approach" jsonschema:"required"`
    Outcome   string   `json:"outcome" jsonschema:"required,enum=success|failure|partial"`
    Tags      []string `json:"tags" jsonschema:"required"`
    Notes     string   `json:"notes,omitempty"`
}
```

**Behavior**:
1. Build `SessionSummary` for Distiller
2. Call `Distiller.DistillSession()` - extracts learnings → memories
3. Execute `HookSessionEnd`

### `context_threshold`

```go
type contextThresholdInput struct {
    ProjectID string `json:"project_id" jsonschema:"required"`
    SessionID string `json:"session_id" jsonschema:"required"`
    Percent   int    `json:"percent" jsonschema:"required"` // 0-100
}
```

**Behavior**:
1. Auto-checkpoint with `AutoCreated: true`
2. Execute `HookContextThreshold`

---

## Service Registry

Registry as interface - idiomatic Go pattern:

```go
// internal/services/registry.go
type Registry interface {
    Checkpoint() checkpoint.Service
    Remediation() remediation.Service
    Memory() *reasoningbank.Service
    Hooks() *hooks.HookManager
    Distiller() *reasoningbank.Distiller
    Scrubber() secrets.Scrubber
}

// concrete implementation
type registry struct {
    checkpoint  checkpoint.Service
    remediation remediation.Service
    memory      *reasoningbank.Service
    hooks       *hooks.HookManager
    distiller   *reasoningbank.Distiller
    scrubber    secrets.Scrubber
}

func NewRegistry(...) Registry {
    return &registry{...}
}

func (r *registry) Checkpoint() checkpoint.Service { return r.checkpoint }
func (r *registry) Memory() *reasoningbank.Service { return r.memory }
// etc.
```

**Usage**:
```go
// Call pattern
registry.Checkpoint().Save(ctx, input)
registry.Distiller().DistillSession(ctx, summary)
registry.Hooks().Execute(ctx, hooks.HookSessionEnd, data)
```

**Benefits**:
- Single mock for all tests
- Clear dependency boundary
- main.go builds registry once, passes to both MCP and HTTP servers

---

## HTTP Endpoint

New endpoint for threshold notification:

```go
// POST /api/v1/threshold
type thresholdRequest struct {
    ProjectID string `json:"project_id"`
    SessionID string `json:"session_id"`
    Percent   int    `json:"percent"`
}

func (s *Server) handleThreshold(c echo.Context) error {
    var req thresholdRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }

    s.registry.Checkpoint().Save(ctx, checkpoint.SaveInput{
        ProjectID:   req.ProjectID,
        SessionID:   req.SessionID,
        Summary:     fmt.Sprintf("Auto-checkpoint at %d%% context", req.Percent),
        AutoCreated: true,
        Threshold:   req.Percent,
    })

    return c.JSON(200, map[string]string{"status": "ok"})
}
```

---

## Skill

`contextd-marketplace/skills/session-lifecycle/SKILL.md`:

```markdown
---
name: session-lifecycle
description: Use at session start and before session end - manages contextd
  memory priming, checkpoint resume, and learning extraction
---

# Session Lifecycle

## On Session Start

Call `session_start` tool with project_id and session_id.

Review the response:
- If checkpoint offered: Ask user "Resume from: {summary}?"
- Surface primed memories: "Relevant context from previous sessions..."

## Before Session End

Before `/clear`, context limit, or ending work:

1. Summarize the session:
   - **task**: What were you trying to accomplish?
   - **approach**: What strategy did you use?
   - **outcome**: success | failure | partial
   - **tags**: Keywords for future discovery

2. Call `session_end` with your summary

## On Context Threshold

If context exceeds 70%:
1. Call `context_threshold` tool OR
2. HTTP: `curl -X POST localhost:9090/api/v1/threshold -d '...'`

This triggers auto-checkpoint before you lose context.
```

---

## Claude Code Hooks

Shell script backstop for automatic triggering:

**`.claude/hooks/precompact.sh`**:
```bash
#!/bin/bash
# Triggers auto-checkpoint before context compaction

PROJECT_ID=$(git remote get-url origin 2>/dev/null | sed 's/.*github.com[:/]\(.*\)\.git/\1/' | tr '/' '_')
SESSION_ID=${CLAUDE_SESSION_ID:-$(date +%s)}
PERCENT=${1:-70}

# Primary: HTTP
curl -sf -X POST "http://localhost:9090/api/v1/threshold" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PROJECT_ID\",\"session_id\":\"$SESSION_ID\",\"percent\":$PERCENT}" \
  && exit 0

# Fallback: prompt Claude to call MCP tool
echo "Call context_threshold tool with project_id=$PROJECT_ID, percent=$PERCENT"
```

---

## Configuration

```yaml
# contextd config
hooks:
  context_threshold:
    method: "http"      # http | mcp | both
    fallback: true      # try mcp if http fails
  auto_checkpoint_on_clear: true
  auto_resume_on_start: false  # prompt instead
  checkpoint_threshold_percent: 70
```

---

## Implementation Order

1. **Service Registry** - `internal/services/registry.go`
2. **Wire Distiller** - Create in main.go, add to registry
3. **Wire HookManager** - Create in main.go, add to registry
4. **MCP Tools** - `session_start`, `session_end`, `context_threshold`
5. **HTTP Endpoint** - `/api/v1/threshold`
6. **Skill** - `contextd-marketplace/skills/session-lifecycle/`
7. **Claude Code Hook** - `.claude/hooks/precompact.sh`

---

## Success Criteria

After implementation:

1. `session_start` returns checkpoint offer + primed memories
2. `session_end` calls Distiller, memories appear in future searches
3. `context_threshold` triggers auto-checkpoint
4. Skill teaches agents the workflow
5. PreCompact hook fires as backstop

**The meta-test**: Next session about contextd should surface THIS design doc via memory search.
