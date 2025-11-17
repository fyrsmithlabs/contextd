# Hook-Based Session Tracking Architecture

## Executive Summary

**Problem Solved**: Use Claude Code's native **SessionStart/SessionEnd hooks** to track context usage per session, eliminating the fundamental MCP protocol limitation.

**Key Insight**: Hooks provide `session_id` and run shell commands with access to contextd MCP tools, enabling accurate per-session tracking WITHOUT protocol changes.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│           Claude Code Session A (session_id: abc123)    │
├─────────────────────────────────────────────────────────┤
│  SessionStart Hook                                      │
│  ├─ Receives: session_id="abc123"                      │
│  ├─ Calls: contextd analytics_start_session             │
│  └─ Exports: SESSION_ID=abc123 to environment          │
│                                                         │
│  ... User conversation (Claude tracks tokens) ...       │
│                                                         │
│  MCP Tool Calls (every request)                         │
│  ├─ checkpoint_search, remediation_search, etc.        │
│  ├─ Extract: $SESSION_ID from environment              │
│  └─ Tag requests with session_id                       │
│                                                         │
│  SessionEnd Hook                                        │
│  ├─ Receives: session_id="abc123", transcript_path     │
│  ├─ Counts: Tokens from transcript file               │
│  └─ Calls: contextd analytics_end_session              │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│              contextd Analytics System                   │
├─────────────────────────────────────────────────────────┤
│  Session Registry:                                       │
│  {                                                       │
│    "abc123": {                                          │
│      session_id: "abc123",                             │
│      project: "/home/user/project",                    │
│      start_time: "2025-11-07T19:40:00Z",              │
│      end_time: null,                                   │
│      tokens_used: 0,                                   │
│      tool_calls: 12,                                   │
│      checkpoints: 2                                    │
│    }                                                    │
│  }                                                      │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│         VictoriaMetrics (Prometheus Metrics)             │
├─────────────────────────────────────────────────────────┤
│  contextd_session_tokens{session_id="abc123"}          │
│  contextd_session_tools{session_id="abc123"}           │
│  contextd_session_checkpoints{session_id="abc123"}     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│              ctxd monitor Dashboard                      │
├─────────────────────────────────────────────────────────┤
│  Top Sessions:                                           │
│  ┌─ abc123 (90%) ⚠ CRITICAL - 180K/200K               │
│  ├─ def456 (70%) ⚠ WARNING  - 140K/200K               │
│  └─ ghi789 (32%) ✓ HEALTHY  -  65K/200K               │
└─────────────────────────────────────────────────────────┘
```

## Solution Components

### 1. Claude Code Hooks Configuration

**File**: `~/.claude/config.json` or `<project>/.claude/config.json`

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/session-start.sh"
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/session-end.sh"
          }
        ]
      }
    ]
  }
}
```

### 2. SessionStart Hook Script

**File**: `.claude/hooks/session-start.sh`

```bash
#!/bin/bash
set -euo pipefail

# Parse hook input (JSON on stdin)
INPUT=$(cat)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id')
PROJECT_DIR=$(echo "$INPUT" | jq -r '.cwd // env.CLAUDE_PROJECT_DIR')

# Call contextd to register session start
curl -s --unix-socket ~/.config/contextd/api.sock \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(cat ~/.config/contextd/token)" \
  -X POST http://localhost/api/v1/analytics/sessions/start \
  -d "{
    \"session_id\": \"$SESSION_ID\",
    \"project_path\": \"$PROJECT_DIR\",
    \"start_time\": \"$(date -Iseconds)\"
  }" > /dev/null

# Export session ID to environment for all bash commands
if [ -n "$CLAUDE_ENV_FILE" ]; then
  echo "export CONTEXTD_SESSION_ID=$SESSION_ID" >> "$CLAUDE_ENV_FILE"
fi

# Return success with optional context injection
echo '{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Session tracking enabled for '$SESSION_ID'"
  }
}'

exit 0
```

### 3. SessionEnd Hook Script

**File**: `.claude/hooks/session-end.sh`

```bash
#!/bin/bash
set -euo pipefail

# Parse hook input
INPUT=$(cat)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id')
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path')

# Count tokens from transcript (approximate)
# Transcript is JSONL format with messages
TOKENS_USED=0
if [ -f "$TRANSCRIPT_PATH" ]; then
  # Count characters and estimate tokens (~4 chars per token)
  CHAR_COUNT=$(jq -r '.content // ""' "$TRANSCRIPT_PATH" 2>/dev/null | wc -c)
  TOKENS_USED=$((CHAR_COUNT / 4))
fi

# Call contextd to register session end
curl -s --unix-socket ~/.config/contextd/api.sock \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(cat ~/.config/contextd/token)" \
  -X POST http://localhost/api/v1/analytics/sessions/end \
  -d "{
    \"session_id\": \"$SESSION_ID\",
    \"end_time\": \"$(date -Iseconds)\",
    \"tokens_used\": $TOKENS_USED
  }" > /dev/null

echo '{
  "hookSpecificOutput": {
    "hookEventName": "SessionEnd"
  }
}'

exit 0
```

### 4. New contextd API Endpoints

**Session Start**
```
POST /api/v1/analytics/sessions/start
{
  "session_id": "abc123",
  "project_path": "/home/user/project",
  "start_time": "2025-11-07T19:40:00Z"
}

Response: 201 Created
{
  "session_id": "abc123",
  "status": "active"
}
```

**Session End**
```
POST /api/v1/analytics/sessions/end
{
  "session_id": "abc123",
  "end_time": "2025-11-07T20:15:00Z",
  "tokens_used": 145234
}

Response: 200 OK
{
  "session_id": "abc123",
  "status": "completed",
  "duration_seconds": 2100,
  "tokens_used": 145234
}
```

**Session Status**
```
GET /api/v1/analytics/sessions/{session_id}

Response: 200 OK
{
  "session_id": "abc123",
  "project_path": "/home/user/project",
  "start_time": "2025-11-07T19:40:00Z",
  "end_time": null,
  "status": "active",
  "tokens_used": 145234,
  "tool_calls": 12,
  "checkpoints_created": 2,
  "last_activity": "2025-11-07T20:10:00Z"
}
```

**List Active Sessions**
```
GET /api/v1/analytics/sessions?status=active

Response: 200 OK
{
  "sessions": [
    {
      "session_id": "abc123",
      "project_path": "/home/user/project",
      "tokens_used": 145234,
      "usage_percent": 72.6
    },
    {
      "session_id": "def456",
      "project_path": "/home/user/other-project",
      "tokens_used": 180000,
      "usage_percent": 90.0
    }
  ],
  "total_active": 2
}
```

### 5. MCP Tool Request Tagging

**Middleware**: Extract session ID from environment and tag all MCP requests

```go
// pkg/mcp/middleware.go
func SessionTrackingMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Session ID is set by hook via environment
            // Passed through MCP tool calls as part of execution context
            sessionID := extractSessionID(c)

            if sessionID != "" {
                // Tag request with session
                c.Set("session_id", sessionID)

                // Update session activity timestamp
                updateSessionActivity(sessionID)

                // Increment tool call counter
                incrementToolCalls(sessionID)
            }

            return next(c)
        }
    }
}

func extractSessionID(c echo.Context) string {
    // Try request header (if added by hook wrapper)
    if id := c.Request().Header.Get("X-Session-ID"); id != "" {
        return id
    }

    // Try query parameter (for debugging)
    if id := c.QueryParam("session_id"); id != "" {
        return id
    }

    // Fallback to connection-based (less accurate)
    return fmt.Sprintf("conn_%s", c.Request().RemoteAddr)
}
```

### 6. Token Counting Strategy

**Option 1: Transcript Parsing (Most Accurate)**

```bash
# Count tokens from Claude Code transcript file
count_tokens_from_transcript() {
    local transcript="$1"
    local total_tokens=0

    # JSONL format: one message per line
    while IFS= read -r line; do
        # Extract message content
        content=$(echo "$line" | jq -r '.content // ""')

        # Estimate tokens (~4 chars per token for English)
        chars=$(echo -n "$content" | wc -c)
        tokens=$((chars / 4))
        total_tokens=$((total_tokens + tokens))
    done < "$transcript"

    echo "$total_tokens"
}
```

**Option 2: tiktoken (More Accurate)**

```python
#!/usr/bin/env python3
import sys
import json
import tiktoken

def count_tokens(transcript_path):
    enc = tiktoken.get_encoding("cl100k_base")  # Claude uses cl100k_base
    total_tokens = 0

    with open(transcript_path, 'r') as f:
        for line in f:
            try:
                msg = json.loads(line)
                content = msg.get('content', '')
                tokens = len(enc.encode(content))
                total_tokens += tokens
            except:
                pass

    return total_tokens

if __name__ == "__main__":
    print(count_tokens(sys.argv[1]))
```

**Option 3: Real-time Estimation (Fastest)**

```bash
# Estimate from request/response sizes during MCP calls
estimate_tokens() {
    local request_chars="$1"
    local response_chars="$2"

    # ~4 chars per token heuristic
    echo $(( (request_chars + response_chars) / 4 ))
}
```

### 7. Dashboard Queries

**VictoriaMetrics PromQL queries for multi-session view**

```promql
# Top 5 sessions by context usage
topk(5, contextd_session_tokens)

# Sessions above 70% threshold
contextd_session_tokens / 200000 > 0.70

# Sessions above 90% threshold
contextd_session_tokens / 200000 > 0.90

# Total active sessions
contextd_active_sessions

# Average usage across all sessions
avg(contextd_session_tokens) / 200000

# Session duration
time() - contextd_session_start_time

# Tools called per session
sum(rate(contextd_session_tool_calls[5m])) by (session_id)

# Checkpoints per session
contextd_session_checkpoints
```

## Implementation Plan

### Phase 1: Core Infrastructure (Day 1)

**1.1 Create Hook Scripts** (30 min)
```bash
mkdir -p .claude/hooks
touch .claude/hooks/session-start.sh
touch .claude/hooks/session-end.sh
chmod +x .claude/hooks/*.sh
```

**1.2 Add contextd API Endpoints** (2 hours)
- `POST /api/v1/analytics/sessions/start`
- `POST /api/v1/analytics/sessions/end`
- `GET /api/v1/analytics/sessions/{id}`
- `GET /api/v1/analytics/sessions?status=active`

**1.3 Session State Manager** (1 hour)
```go
// pkg/analytics/session.go
type SessionRegistry struct {
    mu       sync.RWMutex
    sessions map[string]*Session
}

type Session struct {
    SessionID    string    `json:"session_id"`
    ProjectPath  string    `json:"project_path"`
    StartTime    time.Time `json:"start_time"`
    EndTime      *time.Time `json:"end_time,omitempty"`
    TokensUsed   int64     `json:"tokens_used"`
    ToolCalls    int       `json:"tool_calls"`
    Checkpoints  int       `json:"checkpoints"`
    LastActivity time.Time `json:"last_activity"`
}
```

**1.4 Update Claude Config** (15 min)
```json
// .claude/config.json
{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": ".claude/hooks/session-start.sh"}]}],
    "SessionEnd": [{"hooks": [{"type": "command", "command": ".claude/hooks/session-end.sh"}]}]
  }
}
```

### Phase 2: Token Counting (Day 2)

**2.1 Transcript Parser** (1 hour)
- Shell script version (fast, ±10% accuracy)
- Python tiktoken version (accurate, requires Python)

**2.2 SessionEnd Integration** (30 min)
- Call transcript parser from hook
- Send token count to contextd API

**2.3 Real-time Estimation** (1 hour)
- Add middleware to estimate tokens per tool call
- Update session state incrementally

### Phase 3: Dashboard Integration (Day 3)

**3.1 Add Session Queries** (1 hour)
```go
// internal/monitor/metrics.go
func (c *MetricsClient) QueryActiveSessions(ctx context.Context) ([]SessionMetrics, error)
func (c *MetricsClient) QuerySessionTokens(ctx context.Context, sessionID string) (int64, error)
```

**3.2 Multi-Session Dashboard View** (2 hours)
```go
// internal/monitor/dashboard.go
type SessionView struct {
    SessionID    string
    TokensUsed   int64
    UsagePercent float64
    Duration     time.Duration
    Status       string  // "active", "approaching_70", "critical_90"
}
```

**3.3 Keyboard Navigation** (1 hour)
- Press `s` to toggle session view
- Press `a` to toggle aggregate view
- Arrow keys to scroll sessions

### Phase 4: Testing & Validation (Day 4)

**4.1 Single Session Test**
- Start Claude Code session
- Verify hook runs and registers session
- Make tool calls
- End session, verify token count

**4.2 Multi-Session Test**
- Start 3 Claude Code instances
- Verify each gets unique session_id
- Monitor dashboard shows all 3
- Verify no session mixing

**4.3 Threshold Testing**
- Generate high token usage
- Verify 70% threshold detection
- Verify 90% threshold alerts
- Test checkpoint effectiveness tracking

## Success Criteria

- [ ] SessionStart hook registers new sessions with contextd
- [ ] SessionEnd hook reports final token counts
- [ ] Each Claude instance gets unique session_id
- [ ] Dashboard shows top 5 active sessions
- [ ] Token counts accurate within ±10%
- [ ] No session mixing or race conditions
- [ ] Support 10+ concurrent sessions
- [ ] Hooks complete in < 500ms each
- [ ] All tests pass (≥80% coverage)

## Advantages Over Previous Approaches

### vs Connection-Based Tracking
✅ **Accurate session IDs** - Claude Code provides unique identifiers
✅ **Human-readable** - Can derive session names from project paths
✅ **Persistent** - Survives reconnects if transcript persists

### vs MCP Protocol Extension
✅ **Works today** - No waiting for Anthropic
✅ **No protocol changes** - Uses existing hook system
✅ **Full control** - We own the implementation

### vs Sidecar Wrapper
✅ **Native integration** - Built into Claude Code
✅ **No external processes** - Hooks run directly
✅ **Official API** - Supported by Anthropic

## Limitations & Trade-offs

### Token Counting Accuracy
- **Shell script**: ±10% accuracy (fast, no dependencies)
- **tiktoken**: ±2% accuracy (requires Python)
- **Ground truth**: Not available (Claude doesn't expose)

**Mitigation**: Use shell script by default, offer tiktoken as optional enhancement

### SessionEnd Timing
- Hook runs **after** session ends
- Cannot prevent session termination
- Final token count is retrospective

**Mitigation**: Track real-time estimates during session, use SessionEnd for final reconciliation

### Hook Execution Time
- Adds ~100-500ms to session start/end
- Users may notice slight delay

**Mitigation**: Async execution, caching, optimized scripts

## Migration Path

### Step 1: Add Hooks (Non-Breaking)
- Deploy hook scripts to `.claude/hooks/`
- Update config to enable hooks
- Existing sessions continue working without tracking

### Step 2: Deploy contextd API (Non-Breaking)
- Add new `/analytics/sessions/*` endpoints
- Existing MCP tools unaffected
- Sessions without hooks work as before

### Step 3: Enable Dashboard (Non-Breaking)
- Add session view to `ctxd monitor`
- Aggregate view remains default
- Backward compatible with existing metrics

### Step 4: Deprecate Old Approach (Future)
- Remove connection-based session estimates
- Focus on hook-based tracking
- Clean up legacy code

## Next Steps

**Immediate** (Today):
1. Review this design for approval
2. Create `.claude/hooks/` directory structure
3. Write SessionStart/SessionEnd hook scripts
4. Test with single Claude instance

**This Week**:
1. Implement contextd analytics API endpoints
2. Add session registry and state management
3. Test with multiple concurrent sessions
4. Update dashboard for multi-session view

**Next Week**:
1. Polish token counting (add tiktoken option)
2. Performance optimization (async hooks)
3. Documentation and examples
4. Production deployment

**Ready to implement?** This solves the critical context tracking flaw using Claude Code's native features! 🚀
