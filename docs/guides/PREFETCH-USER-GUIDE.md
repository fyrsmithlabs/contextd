# Pre-Fetch User Guide

**Version**: 0.9.0-rc-1
**Last Updated**: 2025-01-15

---

## Overview

The Pre-Fetch Engine is contextd 0.9.0-rc-1's intelligent context loading system that automatically fetches relevant information when you switch git branches or make commits. Instead of waiting for Claude Code to request context, the pre-fetch engine proactively loads it in the background, reducing round trips and speeding up your development workflow.

### What is Pre-Fetch?

Pre-fetch detects git events (like switching branches) and **deterministically** executes pre-fetch rules to load context **before** Claude Code asks for it. When Claude searches for context, the pre-fetched results are already cached and injected into the response instantly.

**Example Workflow (Without Pre-Fetch)**:
1. You switch to `feature/auth` branch
2. You ask Claude: "What changed in this branch?"
3. Claude calls `checkpoint_search` MCP tool → searches vector DB → returns results (500ms)
4. You continue working

**Example Workflow (With Pre-Fetch)**:
1. You switch to `feature/auth` branch
2. **Pre-fetch engine automatically runs** (background, <2s)
   - Executes `git diff main..feature/auth`
   - Fetches recent commit messages
   - Identifies common changed files
3. You ask Claude: "What changed in this branch?"
4. Claude retrieves **cached pre-fetch results** instantly (cache hit, <10ms)
5. You continue working 20-30% faster

### How It Works

The pre-fetch engine uses a **git-centric, deterministic** approach:

1. **Git Event Detection**: Watches `.git/HEAD` for branch switches and commits
2. **Rule Execution**: Runs 3 deterministic rules in parallel (max 2s total)
   - `branch_diff`: Git diff summary between branches
   - `recent_commit`: Latest commit message and context
   - `common_files`: Pre-fetches frequently changed files
3. **Caching**: Stores results for 5 minutes (configurable TTL)
4. **Injection**: Adds cached results to next MCP tool response

**Worktree Support**: Each git worktree is treated as an independent project with isolated pre-fetch cache.

### Benefits

- **20-30% Token Savings**: Eliminates redundant MCP tool calls
- **Faster Response Times**: Cached results returned in <10ms vs 500ms search
- **Zero Configuration**: Works automatically after indexing a repository
- **Deterministic Behavior**: No AI guessing, rules are predictable and reliable
- **Background Execution**: Non-blocking, runs while you work

---

## Quick Start

### Enable Pre-Fetch

Pre-fetch is **enabled by default** in 0.9.0-rc-1. To verify:

```bash
# Check configuration
cat ~/.config/contextd/config.yaml | grep -A 20 "prefetch:"

# Should show:
# prefetch:
#   enabled: true
```

### Verify It's Working

**Method 1: Check Logs**

```bash
# View contextd logs (Linux)
journalctl --user -u contextd -f | grep prefetch

# View contextd logs (macOS)
tail -f ~/.config/contextd/logs/app.log | grep prefetch

# Expected output after branch switch:
# INFO  prefetch.detector  Git event detected  {"type": "branch_switch", "project": "/path/to/project"}
# DEBUG prefetch.executor   Rules executed     {"duration_ms": 1243, "rules": ["branch_diff", "recent_commit", "common_files"]}
# DEBUG prefetch.cache      Cache stored       {"project": "/path/to/project", "ttl": "5m"}
```

**Method 2: Check Prometheus Metrics**

```bash
# Query cache hit rate
curl -s http://localhost:9090/api/v1/query?query='rate(prefetch_cache_hits_total[5m])' | jq .

# Expected: Non-zero hit rate (≥70% is good)
```

**Method 3: Test with Claude Code**

1. Switch git branches in your terminal: `git checkout feature/new-feature`
2. Wait 2-3 seconds for pre-fetch to complete
3. Ask Claude Code: "What changed in this branch?"
4. Claude should respond **instantly** with pre-fetched diff summary

---

## Configuration

Pre-fetch is configured via `~/.config/contextd/config.yaml`. All settings can be overridden with environment variables.

### Configuration File Location

```bash
~/.config/contextd/config.yaml
```

### Full Configuration Example

```yaml
prefetch:
  # Master control
  enabled: true                 # Enable/disable entire pre-fetch engine

  # Cache settings
  cache_ttl: 5m                 # How long to cache results (5 minutes)
  cache_max_entries: 100        # Maximum projects cached (LRU eviction)

  # Pre-fetch rules
  rules:
    # Branch Diff Rule
    branch_diff:
      enabled: true             # Enable this rule
      max_files: 10             # Maximum files to include in diff
      max_size_kb: 50           # Maximum diff size (truncate if larger)
      timeout_ms: 1000          # Timeout in milliseconds

    # Recent Commit Rule
    recent_commit:
      enabled: true
      max_size_kb: 20           # Maximum commit info size
      timeout_ms: 500

    # Common Files Rule
    common_files:
      enabled: true
      max_files: 3              # Number of common files to pre-fetch
      timeout_ms: 500
```

### Environment Variables

Override any configuration setting with environment variables:

```bash
# Master control
export PREFETCH_ENABLED=false          # Disable pre-fetch entirely

# Cache settings
export PREFETCH_CACHE_TTL=10m          # Change TTL to 10 minutes
export PREFETCH_CACHE_MAX_ENTRIES=200  # Increase cache size

# Rule-level control
export PREFETCH_BRANCH_DIFF_ENABLED=false     # Disable branch_diff rule
export PREFETCH_RECENT_COMMIT_TIMEOUT_MS=1000 # Increase timeout to 1s
export PREFETCH_COMMON_FILES_MAX_FILES=5      # Fetch 5 common files
```

**Priority**: Environment variables > YAML file > defaults

---

## Rules

The pre-fetch engine executes 3 deterministic rules in parallel when git events occur.

### Branch Diff Rule

**Triggers On**: Branch switch
**Executes**: `git diff <old_branch>..<new_branch> --stat`
**Purpose**: Provide summary of changes between branches

**Configuration**:
```yaml
branch_diff:
  enabled: true
  max_files: 10        # Show top 10 changed files
  max_size_kb: 50      # Truncate if diff > 50KB
  timeout_ms: 1000     # Fail if takes > 1s
```

**Output Example**:
```json
{
  "branch_diff": {
    "old_branch": "main",
    "new_branch": "feature/auth",
    "files_changed": 8,
    "insertions": 234,
    "deletions": 67,
    "summary": "pkg/auth/middleware.go | 45 +++++\npkg/auth/token.go       | 89 +++++++\n..."
  }
}
```

**Use Cases**:
- "What changed in this branch?"
- "Show me the diff from main"
- Quick context when switching to feature branches

---

### Recent Commit Rule

**Triggers On**: New commit, branch switch
**Executes**: `git log -1 --format=fuller`
**Purpose**: Provide latest commit context

**Configuration**:
```yaml
recent_commit:
  enabled: true
  max_size_kb: 20      # Truncate large commit messages
  timeout_ms: 500      # Fast execution
```

**Output Example**:
```json
{
  "recent_commit": {
    "hash": "a1b2c3d",
    "author": "John Doe <john@example.com>",
    "date": "2025-01-15T10:30:00Z",
    "message": "feat: add JWT authentication middleware",
    "files_changed": ["pkg/auth/middleware.go", "pkg/auth/token.go"]
  }
}
```

**Use Cases**:
- "What was the last commit about?"
- "Continue implementing the latest feature"
- Context for ongoing work

---

### Common Files Rule

**Triggers On**: Branch switch, new commit
**Executes**: Identifies frequently changed files and pre-fetches content
**Purpose**: Proactively load files likely to be referenced

**Configuration**:
```yaml
common_files:
  enabled: true
  max_files: 3         # Pre-fetch top 3 common files
  timeout_ms: 500
```

**Output Example**:
```json
{
  "common_files": [
    {"path": "README.md", "size": 4523},
    {"path": "pkg/auth/middleware.go", "size": 2341},
    {"path": "config.yaml", "size": 1234}
  ]
}
```

**Use Cases**:
- Automatically load README, CLAUDE.md, main package files
- Reduce "Can you read X file?" requests
- Context for file-heavy workflows

---

## Metrics & Monitoring

Pre-fetch provides 8 Prometheus metrics for performance tracking and tuning.

### Available Metrics

```bash
# Git event detection
prefetch_git_events_total{type="branch_switch|new_commit"}

# Rule execution
prefetch_rules_executed_total{rule="branch_diff|recent_commit|common_files"}
prefetch_rule_timeouts_total{rule="..."}
prefetch_rule_duration_seconds{rule="..."}  # Histogram

# Cache performance
prefetch_cache_hits_total
prefetch_cache_misses_total
prefetch_cache_size

# Token savings
prefetch_tokens_saved_total
```

### Hit Rate Calculation

**Target**: ≥70% cache hit rate

```bash
# Query Prometheus for hit rate
curl -s 'http://localhost:9090/api/v1/query?query=rate(prefetch_cache_hits_total[5m])/(rate(prefetch_cache_hits_total[5m])+rate(prefetch_cache_misses_total[5m]))' | jq '.data.result[0].value[1]'

# Expected: "0.75" (75% hit rate)
```

**Interpreting Results**:
- **≥80%**: Excellent - Pre-fetch is highly effective
- **70-80%**: Good - Normal performance
- **50-70%**: Fair - Consider tuning cache TTL or rules
- **<50%**: Poor - Investigate logs for rule timeouts or configuration issues

### Token Savings Estimation

```bash
# Total tokens saved by pre-fetch
curl -s http://localhost:9090/api/v1/query?query='prefetch_tokens_saved_total' | jq .

# Expected: Increasing counter (e.g., 125000 tokens saved)
```

**Estimated Savings**: 20-30% reduction in total token usage for git-centric workflows.

### Grafana Dashboard (Optional)

contextd includes a Grafana dashboard for pre-fetch monitoring:

```bash
# Start Grafana (if not running)
docker-compose up -d grafana

# Access dashboard
# URL: http://localhost:3001
# Login: admin / admin
# Dashboard: "contextd - Pre-Fetch Performance"
```

**Dashboard Panels**:
- Cache hit rate (time series)
- Rule execution duration (histogram)
- Git events per hour
- Token savings cumulative

---

## Troubleshooting

### Pre-Fetch Not Working

**Symptom**: No pre-fetch logs, cache always misses

**Diagnosis**:
```bash
# 1. Check if pre-fetch is enabled
cat ~/.config/contextd/config.yaml | grep "enabled:"
# Should show: enabled: true

# 2. Check environment variable override
env | grep PREFETCH_ENABLED
# Should be empty or "true"

# 3. Check detector is running
journalctl --user -u contextd -f | grep "detector started"
# Should show: INFO prefetch.detector Detector started {"project": "..."}
```

**Solutions**:
- **If disabled**: Set `prefetch.enabled: true` in config.yaml
- **If env var overrides**: Unset `PREFETCH_ENABLED` or set to `true`
- **If no detector logs**: Repository not indexed - run `/index repository path=/path/to/project`

---

### Low Hit Rate (<50%)

**Symptom**: Cache hit rate below 50%, few tokens saved

**Diagnosis**:
```bash
# Check rule timeout rate
curl -s http://localhost:9090/api/v1/query?query='prefetch_rule_timeouts_total' | jq .

# If timeouts > 0, rules are timing out (not caching results)
```

**Solutions**:
1. **Increase timeouts** (if rules consistently timeout):
   ```yaml
   rules:
     branch_diff:
       timeout_ms: 2000  # Increase from 1000ms
   ```

2. **Disable slow rules** (if one rule always times out):
   ```yaml
   rules:
     common_files:
       enabled: false  # Disable if causing issues
   ```

3. **Increase cache TTL** (if hit rate improves after increasing TTL):
   ```yaml
   cache_ttl: 10m  # Increase from 5m
   ```

4. **Check git repository size**: Large repositories may need higher timeouts

---

### Detector Not Starting

**Symptom**: `ERROR prefetch.detector Failed to start detector`

**Diagnosis**:
```bash
# Check logs for error details
journalctl --user -u contextd -f | grep "ERROR.*detector"

# Common errors:
# - "not a git repository"
# - "permission denied"
# - ".git/HEAD not found"
```

**Solutions**:
- **Not a git repository**: Verify `git status` works in project directory
- **Permission denied**: Check file permissions on `.git/` directory
- **Worktree issue**: Ensure `.git` file contains valid `gitdir:` path

---

### High Memory Usage

**Symptom**: contextd using >500MB memory with pre-fetch enabled

**Diagnosis**:
```bash
# Check cache size
curl -s http://localhost:9090/api/v1/query?query='prefetch_cache_size' | jq .

# If cache_size > 100, cache is at max capacity
```

**Solutions**:
1. **Reduce cache size**:
   ```yaml
   cache_max_entries: 50  # Reduce from 100
   ```

2. **Reduce cache TTL** (shorter TTL = faster eviction):
   ```yaml
   cache_ttl: 2m  # Reduce from 5m
   ```

3. **Disable pre-fetch for unused projects**: Stop detectors for projects no longer active

---

### Common Issues and Solutions

| Issue | Symptom | Solution |
|-------|---------|----------|
| Pre-fetch disabled | No cache hits, no logs | Set `prefetch.enabled: true` |
| Rule timeouts | Low hit rate, timeout metrics | Increase `timeout_ms` or disable slow rules |
| Worktree not detected | No events on worktree branch switch | Verify `.git` file contains `gitdir:` path |
| Cache never evicts | Memory grows unbounded | Reduce `cache_ttl` or `cache_max_entries` |
| Duplicate events | Multiple events for single git action | Normal - detector may trigger on multiple file changes |

---

## Advanced Topics

### Worktree Support

contextd treats each git worktree as an **independent project** with isolated cache.

**Example**:
```bash
# Main repository
/project/.git/HEAD         → Detector watches this

# Worktree (feature branch)
/project-feature/.git      → File containing "gitdir: /project/.git/worktrees/feature"
/project/.git/worktrees/feature/HEAD  → Detector watches this

# Each has separate cache, events don't cross-contaminate
```

**Why This Matters**:
- Switching branches in one worktree doesn't invalidate cache in another
- Parallel development workflows benefit from independent pre-fetch per worktree
- Cache isolation prevents false hits across worktrees

---

### Cache Tuning

**Default Settings**:
- TTL: 5 minutes
- Max entries: 100 projects
- Eviction: LRU (least recently used)

**Tuning Recommendations**:

**For Long-Running Sessions** (8+ hours):
```yaml
cache_ttl: 30m          # Increase TTL for fewer re-executions
cache_max_entries: 200  # More projects cached
```

**For Short Sessions** (<2 hours):
```yaml
cache_ttl: 2m           # Shorter TTL, fresher results
cache_max_entries: 50   # Fewer projects, lower memory
```

**For Memory-Constrained Environments**:
```yaml
cache_ttl: 1m           # Aggressive eviction
cache_max_entries: 20   # Minimal cache size
```

---

### Disabling Specific Rules

Disable rules that don't provide value for your workflow:

**Example: Code-only workflow (no commit messages needed)**:
```yaml
rules:
  recent_commit:
    enabled: false  # Disable commit rule
```

**Example: Large repositories (diff timeouts)**:
```yaml
rules:
  branch_diff:
    enabled: false  # Disable diff rule if it times out
```

**Example: Minimal pre-fetch (cache only common files)**:
```yaml
rules:
  branch_diff:
    enabled: false
  recent_commit:
    enabled: false
  common_files:
    enabled: true   # Only pre-fetch common files
```

---

## Best Practices

1. **Monitor Hit Rate**: Aim for ≥70% cache hit rate
2. **Tune Timeouts**: If rules timeout, increase `timeout_ms` before disabling
3. **Use Worktrees**: Pre-fetch is optimized for parallel worktree workflows
4. **Check Logs**: Structured logs provide detailed pre-fetch activity
5. **Start Conservative**: Use defaults, tune only if performance degrades
6. **Disable on Low-End Systems**: If <4GB RAM, consider disabling pre-fetch
7. **Restart After Config Changes**: Changes to config.yaml require contextd restart

---

## Getting Help

- **Documentation**: [docs/specs/mcp-prefetch/SPEC.md](../../specs/mcp-prefetch/SPEC.md)
- **Design Document**: [docs/plans/2025-01-15-prefetch-engine-design.md](../../plans/2025-01-15-prefetch-engine-design.md)
- **GitHub Issues**: [github.com/axyzlabs/contextd/issues](https://github.com/axyzlabs/contextd/issues)
- **Discussions**: [github.com/axyzlabs/contextd/discussions](https://github.com/axyzlabs/contextd/discussions)

---

**Last Updated**: 2025-01-15
**Version**: 0.9.0-rc-1
