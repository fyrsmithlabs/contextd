# Package prefetch - CLAUDE.md

**Purpose**: Git-centric pre-fetching engine that deterministically fetches context on git events to eliminate wasteful round trips.

---

## Specification

**Design Document**: [docs/plans/2025-01-15-prefetch-engine-design.md](../../docs/plans/2025-01-15-prefetch-engine-design.md)

**Core Principle**: "If you already know what tools Claude will want to call, just call them DETERMINISTICALLY." — 12-Factor Agents

**Scope**: Detect git events (branch switch, commit) and execute deterministic pre-fetch rules in background.

---

## Architecture

### Components

1. **`detector.go`** - Git event detection via filesystem watcher
2. **`rules.go`** - Deterministic pre-fetch rules engine
3. **`executor.go`** - Parallel rule execution with timeout protection
4. **`cache.go`** - In-memory result caching with TTL

### Flow

```
Git Event (branch switch)
    ↓
Detector (fsnotify watcher)
    ↓
Rules Engine (3 deterministic rules)
    ↓
Executor (parallel execution, 2s timeout)
    ↓
Cache (5min TTL, inject into next MCP response)
```

---

## Key Components

### GitEventDetector

**Purpose**: Detect git events using filesystem watchers on `.git/HEAD`.

**Critical Features**:
- **Worktree Support** - Each worktree treated as independent project
- **Event Types** - BranchSwitch, NewCommit
- **Background Operation** - Non-blocking goroutines per project

**Worktree Handling**:
```go
// Main repo: .git is a directory
Watch: /project/.git/HEAD

// Worktree: .git is a file pointing to worktree location
Parse: "gitdir: /main/.git/worktrees/feature"
Watch: /main/.git/worktrees/feature/HEAD
```

### RulesEngine

**Purpose**: Execute deterministic pre-fetch rules based on git events.

**3 Rules**:
1. **branch_diff** - Git diff summary on branch switch (1s timeout)
2. **related_files** - Search changed files in vector store (2s timeout)
3. **recent_commit** - Commit message and context (500ms timeout)

**Execution**: All rules run in parallel, failures don't block, timeouts logged.

### PreFetchCache

**Purpose**: Store pre-fetched results with TTL and inject into MCP responses.

**Features**:
- TTL: 5 minutes (configurable)
- LRU eviction: Max 100 projects
- Invalidation: On new git event
- Thread-safe: `sync.RWMutex`

**Injection**: Results added to MCP response as `prefetch` field.

---

## Usage Example

### Starting Pre-Fetch for Project

```go
// In MCP server initialization
prefetcher := prefetch.NewService(vectorCore, gitClient, config)

// Start detector for indexed project
err := prefetcher.StartDetector(ctx, projectPath)
if err != nil {
    log.Warn("Pre-fetch disabled for project", zap.Error(err))
}
```

### MCP Response with Pre-Fetch Data

```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [...],
    "prefetch": {
      "branch_diff": {
        "old_branch": "main",
        "new_branch": "feature/auth",
        "summary": "3 files changed, 45 insertions(+), 12 deletions(-)"
      },
      "related_files": [
        {"path": "pkg/auth/middleware.go", "score": 0.92}
      ]
    }
  }
}
```

---

## Testing

### Unit Tests (≥80% coverage)

**CRITICAL: Worktree Test Coverage**
- ✅ Main repo + 2 worktrees with independent branch switches
- ✅ Worktree deletion cleanup
- ✅ Cross-worktree cache isolation

**Standard Coverage**:
- Branch switch detection
- Commit detection
- Rule execution (all 3 rules)
- Timeout handling
- Cache operations (write/read/expire/evict)
- Concurrent access (race detector)

### Integration Tests

- End-to-end: Git event → Rules → Cache → MCP injection
- Worktree scenarios (CRITICAL)
- Large diff handling (1000+ files)
- 100 concurrent events (stress test)

---

## Configuration

```yaml
prefetch:
  enabled: true
  cache_ttl: 5m
  cache_max_entries: 100
  rule_timeout: 2s
  max_parallel_rules: 3

  git_watcher_enabled: true
  debounce_interval: 500ms

  rules:
    branch_diff:
      enabled: true
      max_size_kb: 50
      timeout: 1s
    related_files:
      enabled: true
      max_results: 10
      timeout: 2s
    recent_commit:
      enabled: true
      max_size_kb: 20
      timeout: 500ms
```

---

## Security Considerations

**No Additional Attack Surface**:
- Pre-fetch only executes existing git commands (diff, show, log)
- Results subject to same multi-tenant isolation as regular tools
- Timeout protection prevents resource exhaustion
- No user-controlled input in pre-fetch logic

**Resource Limits**:
- Max 100 cached projects (LRU eviction)
- 2s timeout per rule (configurable)
- 5min TTL prevents unbounded growth
- Background cleanup every 1 minute

---

## Performance Notes

**Memory Usage**:
- ~1KB per cached pre-fetch result
- ~100KB for 100 projects at max capacity
- One fsnotify watcher per indexed project (~50KB each)

**CPU Usage**:
- Event-driven (no polling)
- Parallel rule execution (max 3 concurrent per event)
- Background cleanup goroutine (minimal overhead)

**Target Metrics**:
- Hit rate: ≥70% (cache used by Claude)
- Token savings: 20-30% for git workflows
- Latency: <2s for all rules combined

---

## Error Handling

**Principle**: Pre-fetch is best-effort, never critical path.

1. **Detector Failures** → Log warning, disable pre-fetch for project
2. **Rule Timeouts** → **Log timeout**, skip rule, continue with other rules
3. **Cache Failures** → Degrade gracefully, execute normal MCP tools
4. **Git Command Errors** → Log and skip, don't crash

**All errors use structured logging (zap) with context.**

---

## Metrics

**Prometheus Metrics**:
```
prefetch_git_events_total{type="branch_switch|new_commit"}
prefetch_rules_executed_total{rule="branch_diff|related_files|recent_commit"}
prefetch_rule_timeouts_total{rule="..."}
prefetch_cache_hits_total
prefetch_cache_misses_total
prefetch_cache_size
prefetch_tokens_saved_total
```

**Hit Rate Formula**:
```
Hit Rate = prefetch_cache_hits / (prefetch_cache_hits + prefetch_cache_misses)
```

---

## Related Documentation

- **Design**: [docs/plans/2025-01-15-prefetch-engine-design.md](../../docs/plans/2025-01-15-prefetch-engine-design.md)
- **Spec**: [docs/specs/mcp-prefetch/SPEC.md](../../docs/specs/mcp-prefetch/SPEC.md)
- **V3 Rebuild**: [docs/specs/v3-rebuild/SPEC.md](../../docs/specs/v3-rebuild/SPEC.md)
- **12-Factor Agents**: https://github.com/humanlayer/12-factor-agents

---

## Implementation Checklist

- [ ] `detector.go` - Git event detection with worktree support
- [ ] `rules.go` - 3 deterministic rules (branch_diff, related_files, recent_commit)
- [ ] `executor.go` - Parallel execution with timeout protection
- [ ] `cache.go` - TTL cache with LRU eviction
- [ ] Unit tests (≥80% coverage)
- [ ] Integration tests (including worktree scenarios)
- [ ] Metrics implementation (Prometheus + OpenTelemetry)
- [ ] MCP server integration (response injection)
- [ ] Configuration loading
- [ ] Error handling and logging

---

**Summary**: Git-centric pre-fetching eliminates round trips by deterministically fetching context on git events. Worktree support is critical. Pre-fetch is best-effort, never blocks normal operations.
