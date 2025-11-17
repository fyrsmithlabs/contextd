# Session-Based Context Window Monitoring

## Overview

Enable `ctxd monitor` to track context window usage across **multiple Claude Code sessions simultaneously**, providing real-time visibility into when sessions approach the 70%/90% checkpoint thresholds.

## Problem Statement

**Current state:**
- Context metrics are session-agnostic (global aggregates)
- Cannot distinguish between multiple concurrent Claude Code sessions
- No way to identify which session is approaching context limits
- Dashboard shows only aggregate context usage

**Desired state:**
- Track context usage **per Claude Code session**
- Identify sessions by unique session ID
- Monitor multiple sessions simultaneously in dashboard
- Alert when any session hits 70%/90% thresholds
- Show per-session checkpoint effectiveness

## Requirements

### Functional Requirements

1. **Session Identification**
   - Extract session ID from MCP requests
   - Support multiple concurrent sessions
   - Persist session state across requests
   - Auto-expire stale sessions (>1 hour idle)

2. **Per-Session Metrics**
   - Context tokens used (current)
   - Context usage percentage (0-100%)
   - Threshold violations (70%, 90%)
   - Checkpoint count per session
   - Average tokens saved per checkpoint
   - Session duration and last activity

3. **Dashboard Display**
   - Show top 5 active sessions by context usage
   - Highlight sessions approaching thresholds
   - Display per-session sparklines
   - Aggregate view (all sessions combined)
   - Session details on demand

### Non-Functional Requirements

1. **Performance**
   - Metric collection overhead < 1ms per request
   - Support 100+ concurrent sessions
   - Efficient session cleanup (no memory leaks)

2. **Accuracy**
   - Context tracking accurate to ±1% of actual usage
   - Real-time updates (< 5s latency)
   - Handle Claude Code API changes gracefully

## Architecture

### Component Design

```
┌─────────────────────────────────────────────────────────────┐
│                    Claude Code Sessions                     │
│  Session A (80K)  Session B (150K)  Session C (40K)        │
└────────────┬────────────┬─────────────┬─────────────────────┘
             │            │             │
             │ MCP        │ MCP         │ MCP
             │ Request    │ Request     │ Request
             ▼            ▼             ▼
┌─────────────────────────────────────────────────────────────┐
│              contextd MCP Server (port :unix)               │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │        Session Context Tracking Middleware          │   │
│  │                                                      │   │
│  │  1. Extract session_id from request headers        │   │
│  │  2. Update session context state                    │   │
│  │  3. Export per-session metrics to OTEL             │   │
│  │  4. Check threshold violations                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           Session State Manager                      │   │
│  │                                                      │   │
│  │  sessions = map[sessionID]*SessionContext {         │   │
│  │    sessionID: {                                      │   │
│  │      tokensUsed: 80000,                             │   │
│  │      usagePercent: 40.0,                            │   │
│  │      checkpoints: 2,                                │   │
│  │      lastActivity: time.Now(),                      │   │
│  │      threshold70Hit: false,                         │   │
│  │      threshold90Hit: false,                         │   │
│  │    }                                                 │   │
│  │  }                                                   │   │
│  └─────────────────────────────────────────────────────┘   │
└────────────┬─────────────────────────────────────────────┬──┘
             │ OTLP/HTTP                          │ OTLP/HTTP
             ▼                                    ▼
┌─────────────────────────┐      ┌──────────────────────────┐
│   OTEL Collector        │      │   Prometheus Metrics     │
│   (localhost:4318)      │      │   (embedded)             │
└────────────┬────────────┘      └──────────┬───────────────┘
             │                               │
             ▼                               ▼
┌─────────────────────────┐      ┌──────────────────────────┐
│   VictoriaMetrics       │◄─────│   ctxd monitor           │
│   (localhost:8428)      │      │   (queries VM)           │
└─────────────────────────┘      └──────────────────────────┘
```

### Session ID Extraction

**Option 1: Claude Code Request Headers** (Preferred)
```go
// Check for Claude-specific headers
sessionID := c.Request().Header.Get("X-Claude-Session-ID")
if sessionID == "" {
    sessionID := c.Request().Header.Get("X-Request-ID")
}
if sessionID == "" {
    // Fallback: Generate from connection metadata
    sessionID = generateSessionID(c)
}
```

**Option 2: MCP Protocol Metadata**
```go
// Extract from MCP request body
type MCPRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params"`
    Meta    struct {
        SessionID string `json:"session_id"`
    } `json:"meta"`
}
```

**Option 3: Connection-Based Tracking**
```go
// Track by Unix socket connection (each Claude Code instance = unique socket)
// This works because Claude Code maintains persistent connections
sessionID := fmt.Sprintf("session_%d", c.Request().RemoteAddr)
```

### Session Context State

```go
// pkg/monitoring/session.go
package monitoring

import (
    "sync"
    "time"
)

// SessionContext tracks context window usage for a single Claude Code session
type SessionContext struct {
    SessionID        string
    TokensUsed       int64
    UsagePercent     float64
    CheckpointCount  int
    TokensSaved      int64  // Total tokens saved via checkpoints
    CreatedAt        time.Time
    LastActivity     time.Time
    Threshold70Hit   bool
    Threshold90Hit   bool
    Threshold70Count int  // How many times hit 70%
    Threshold90Count int  // How many times hit 90%
}

// SessionManager manages all active sessions
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*SessionContext
    maxAge   time.Duration  // Auto-expire after 1 hour idle
}

func NewSessionManager() *SessionManager {
    sm := &SessionManager{
        sessions: make(map[string]*SessionContext),
        maxAge:   1 * time.Hour,
    }

    // Start cleanup goroutine
    go sm.cleanupStale()

    return sm
}

func (sm *SessionManager) UpdateContext(sessionID string, tokensUsed int64) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    ctx, exists := sm.sessions[sessionID]
    if !exists {
        ctx = &SessionContext{
            SessionID:   sessionID,
            CreatedAt:   time.Now(),
        }
        sm.sessions[sessionID] = ctx
    }

    ctx.TokensUsed = tokensUsed
    ctx.UsagePercent = float64(tokensUsed) / 200000.0 * 100.0
    ctx.LastActivity = time.Now()

    // Check thresholds
    if ctx.UsagePercent >= 70 && !ctx.Threshold70Hit {
        ctx.Threshold70Hit = true
        ctx.Threshold70Count++
    }
    if ctx.UsagePercent >= 90 && !ctx.Threshold90Hit {
        ctx.Threshold90Hit = true
        ctx.Threshold90Count++
    }
}

func (sm *SessionManager) RecordCheckpoint(sessionID string, tokensSaved int64) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    if ctx, exists := sm.sessions[sessionID]; exists {
        ctx.CheckpointCount++
        ctx.TokensSaved += tokensSaved
        ctx.LastActivity = time.Now()

        // Reset threshold flags after checkpoint
        ctx.Threshold70Hit = false
        ctx.Threshold90Hit = false
    }
}

func (sm *SessionManager) GetActiveSessions() []*SessionContext {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    sessions := make([]*SessionContext, 0, len(sm.sessions))
    for _, ctx := range sm.sessions {
        sessions = append(sessions, ctx)
    }
    return sessions
}

func (sm *SessionManager) cleanupStale() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        sm.mu.Lock()
        now := time.Now()
        for id, ctx := range sm.sessions {
            if now.Sub(ctx.LastActivity) > sm.maxAge {
                delete(sm.sessions, id)
            }
        }
        sm.mu.Unlock()
    }
}
```

### Metrics Definition

```go
// pkg/metrics/session_metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Per-session context usage
    SessionContextTokens = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "contextd_session_context_tokens",
            Help: "Current context tokens used by Claude Code session",
        },
        []string{"session_id"},
    )

    SessionContextPercent = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "contextd_session_context_usage_percent",
            Help: "Current context usage percentage by session (0-100)",
        },
        []string{"session_id"},
    )

    // Threshold violations
    SessionThreshold70Hits = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "contextd_session_threshold_70_total",
            Help: "Number of times session hit 70% context threshold",
        },
        []string{"session_id"},
    )

    SessionThreshold90Hits = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "contextd_session_threshold_90_total",
            Help: "Number of times session hit 90% context threshold",
        },
        []string{"session_id"},
    )

    // Checkpoint effectiveness
    SessionCheckpointCount = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "contextd_session_checkpoint_total",
            Help: "Total checkpoints created per session",
        },
        []string{"session_id"},
    )

    SessionTokensSaved = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "contextd_session_tokens_saved_total",
            Help: "Total tokens saved via checkpoints per session",
        },
        []string{"session_id"},
    )

    SessionDuration = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "contextd_session_duration_seconds",
            Help: "Session duration in seconds",
        },
        []string{"session_id"},
    )

    // Aggregate metrics
    ActiveSessionsCount = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "contextd_active_sessions_total",
            Help: "Number of active Claude Code sessions",
        },
    )
)
```

### Middleware Implementation

```go
// pkg/monitoring/middleware.go
package monitoring

import (
    "github.com/labstack/echo/v4"
    "github.com/axyzlabs/contextd/pkg/metrics"
)

// ContextTrackingMiddleware tracks context usage per session
func ContextTrackingMiddleware(sm *SessionManager) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Extract session ID
            sessionID := extractSessionID(c)

            // Process request
            err := next(c)

            // Update metrics (after request completes)
            if sessionID != "" {
                updateSessionMetrics(sm, sessionID)
            }

            return err
        }
    }
}

func extractSessionID(c echo.Context) string {
    // Try Claude-specific headers first
    if id := c.Request().Header.Get("X-Claude-Session-ID"); id != "" {
        return id
    }

    // Try request ID
    if id := c.Request().Header.Get("X-Request-ID"); id != "" {
        return id
    }

    // Fallback: connection-based (Unix socket peer)
    // Each Claude Code instance maintains a persistent connection
    if addr := c.Request().RemoteAddr; addr != "" {
        return fmt.Sprintf("conn_%s", addr)
    }

    return "unknown"
}

func updateSessionMetrics(sm *SessionManager, sessionID string) {
    sessions := sm.GetActiveSession(sessionID)
    if ctx := sessions[sessionID]; ctx != nil {
        // Update Prometheus metrics
        metrics.SessionContextTokens.WithLabelValues(sessionID).Set(float64(ctx.TokensUsed))
        metrics.SessionContextPercent.WithLabelValues(sessionID).Set(ctx.UsagePercent)
        metrics.SessionDuration.WithLabelValues(sessionID).Set(time.Since(ctx.CreatedAt).Seconds())

        // Threshold counters (only increment when first hit)
        if ctx.Threshold70Count > 0 {
            metrics.SessionThreshold70Hits.WithLabelValues(sessionID).Add(float64(ctx.Threshold70Count))
            ctx.Threshold70Count = 0  // Reset counter
        }
        if ctx.Threshold90Count > 0 {
            metrics.SessionThreshold90Hits.WithLabelValues(sessionID).Add(float64(ctx.Threshold90Count))
            ctx.Threshold90Count = 0
        }
    }

    // Update aggregate count
    metrics.ActiveSessionsCount.Set(float64(len(sm.GetActiveSession(""))))
}
```

### Dashboard Updates

#### New VictoriaMetrics Queries

```promql
# Top 5 sessions by context usage
topk(5, contextd_session_context_usage_percent)

# Sessions above 70% threshold
contextd_session_context_usage_percent > 70

# Sessions above 90% threshold
contextd_session_context_usage_percent > 90

# Aggregate context usage across all sessions
sum(contextd_session_context_tokens)

# Average context usage across sessions
avg(contextd_session_context_usage_percent)

# Threshold hit rate (last 5 minutes)
rate(contextd_session_threshold_70_total[5m])
rate(contextd_session_threshold_90_total[5m])

# Checkpoint effectiveness per session
rate(contextd_session_tokens_saved_total[5m]) / rate(contextd_session_checkpoint_total[5m])

# Active sessions count
contextd_active_sessions_total
```

#### Updated Dashboard Layout

```
╭─ contextd Monitor ─────────────────────────────────────────╮
│ ✓ HEALTHY   Uptime: 2h 15m   Active Sessions: 3           │
├────────────────────────────────────────────────────────────┤
│ Context Window (All Sessions)                              │
│ ├─ Total Usage: 245K / 600K tokens (40.8%)                │
│ ├─ Avg Per Session: 81.7K / 200K (40.8%)                  │
│ └─ 70% Hits: 2.5/5m   90% Hits: 0.3/5m                    │
├────────────────────────────────────────────────────────────┤
│ Top Sessions by Context Usage                              │
│ ┌─ Session A (conn_127.0.0.1:52341)                       │
│ │  Usage: 180K / 200K (90%) [✗]  ▂▃▅▇█ ⚠ CRITICAL        │
│ │  Checkpoints: 5   Saved: 320K avg   Duration: 1h 23m   │
│ ├─ Session B (conn_127.0.0.1:52342)                       │
│ │  Usage: 140K / 200K (70%) [⚠]  ▂▃▅▇█ ⚠ WARNING         │
│ │  Checkpoints: 3   Saved: 180K avg   Duration: 45m      │
│ └─ Session C (conn_127.0.0.1:52343)                       │
│    Usage: 65K / 200K (32.5%) [✓]  ▂▃▅▄▃                   │
│    Checkpoints: 1   Saved: 95K avg    Duration: 12m      │
├────────────────────────────────────────────────────────────┤
│ HTTP Requests                                              │
│ └─ Rate: 45.7 req/min   Latency (p95): 12.3ms            │
├────────────────────────────────────────────────────────────┤
│ Embeddings                                                 │
│ └─ Ops: 120/min   Tokens: 15.2k/min   Cost: $0.0034/min  │
└────────────────────────────────────────────────────────────┘
[q] quit  [r] refresh  [s] sessions  [a] aggregate  Auto: 5s
```

## Implementation Plan

### Phase 1: Core Session Tracking (TDD)

**Test 1**: Session ID Extraction
```go
func TestExtractSessionID(t *testing.T) {
    // Test header extraction
    // Test fallback to connection
    // Test unknown session
}
```

**Test 2**: Session State Management
```go
func TestSessionManager_UpdateContext(t *testing.T) {
    // Test new session creation
    // Test context updates
    // Test threshold detection
}

func TestSessionManager_RecordCheckpoint(t *testing.T) {
    // Test checkpoint recording
    // Test tokens saved tracking
    // Test threshold reset
}
```

**Test 3**: Session Cleanup
```go
func TestSessionManager_cleanupStale(t *testing.T) {
    // Test stale session removal
    // Test max age enforcement
}
```

### Phase 2: Metrics Integration

**Test 4**: Metrics Export
```go
func TestUpdateSessionMetrics(t *testing.T) {
    // Test Prometheus gauge updates
    // Test counter increments
    // Test label values
}
```

### Phase 3: Middleware Integration

**Test 5**: Middleware Functionality
```go
func TestContextTrackingMiddleware(t *testing.T) {
    // Test session ID extraction
    // Test metrics update on request
    // Test no performance degradation
}
```

### Phase 4: Dashboard Updates

1. Add session queries to `internal/monitor/metrics.go`
2. Update `dashboard.go` to display top sessions
3. Add session detail view (press 's')
4. Add keyboard navigation between views

## Success Criteria

- [ ] Track context usage per Claude Code session
- [ ] Support 100+ concurrent sessions
- [ ] Display top 5 sessions in dashboard
- [ ] Highlight sessions approaching thresholds (70%, 90%)
- [ ] Show per-session checkpoint effectiveness
- [ ] Auto-expire stale sessions (>1 hour idle)
- [ ] Metrics overhead < 1ms per request
- [ ] All tests pass (≥80% coverage)
- [ ] TDD workflow followed (RED-GREEN-REFACTOR)

## Future Enhancements (Out of Scope)

- Session naming (user-defined labels)
- Session history (persist past sessions)
- Alert notifications (Slack, email when 90% hit)
- Session comparison view
- Export session metrics to CSV
- Session replay (view historical context usage)

## Questions to Resolve

1. **Session ID Source**: Which method for extracting session ID?
   - Headers (requires Claude Code support)
   - Connection tracking (works now, less precise)
   - MCP protocol metadata (requires spec change)

2. **Context Tracking Method**: How to measure actual token usage?
   - Estimate from request/response sizes
   - Parse Claude Code logs (if available)
   - Track via checkpoint deltas
   - Instrument Claude Code directly (ideal, requires changes)

3. **Dashboard View**: Single vs Multi-view?
   - Aggregate view (all sessions) + detail view (per session)
   - Split view (top sessions + aggregate)
   - Tabbed view (switch between sessions)

**Recommendation**: Start with connection-based tracking (works immediately), add header support later when available.
