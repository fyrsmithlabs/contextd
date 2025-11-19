# 1. Architecture Overview

[← Back to Index](INDEX.md) | [Next: MCP Protocol →](02-mcp-protocol.md)

---

## Overview

Contextd implements **context-folding** to manage long-horizon agent trajectories. Agents branch into sub-trajectories for focused work, then fold intermediate steps into summaries, compressing working context while preserving full history.

**Key Metaphor**: Like Git branches - create isolated workspace for subtask, merge summary back to main when done.

---

## High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                      Claude Code (LLM)                       │
│                                                              │
│  Decision: When to branch?  ←── Process reward heuristics   │
│  Decision: When to fold?    ←── Out-of-scope detection     │
└────────────────────────┬────────────────────────────────────┘
                         │ MCP over HTTP
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   Contextd Server                            │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ MCP Tools    │  │ Context Mgr  │  │ Meta-Learning│     │
│  │ (branch/fold)│  │ F(·)         │  │ (strategies) │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │             │
│  ┌──────▼──────────────────▼──────────────────▼───────┐   │
│  │           NATS Event Bus                           │   │
│  │  - Branch state: {owner}/{project}/{session}/br_*  │   │
│  │  - Events: branch.created, branch.folded           │   │
│  │  - Meta-learning triggers: remediation.saved       │   │
│  └──────┬──────────────────────────────────────┬──────┘   │
│         │                                       │           │
│  ┌──────▼───────┐                      ┌───────▼──────┐   │
│  │ Gitleaks SDK │◄─────────────────────│ Secret       │   │
│  │ (800+ rules) │  Double-scan         │ Scrubber     │   │
│  └──────┬───────┘  (exec + fold)       └───────┬──────┘   │
└─────────┼──────────────────────────────────────┼──────────┘
          │                                       │
          └───────────────┬───────────────────────┘
                          │
          ┌───────────────▼───────────────┐
          │   Qdrant Vector Database      │
          │                               │
          │  Main Collections (folded):   │
          │  - checkpoints                │
          │  - remediations               │
          │  - skills                     │
          │                               │
          │  Archive Collections (full):  │
          │  - branch_trajectories        │
          │  - unredacted_content         │
          │                               │
          │  Shared Collections:          │
          │  - strategies (meta-learning) │
          └───────────────────────────────┘
```

---

## Core Mechanism

**Branch Operation**:
1. LLM calls `context_branch(description, prompt)`
2. Contextd creates branch ID, stores in NATS: `{owner}/{project}/{session}/branches/{branch_id}`
3. LLM performs operations within branch
4. All operations tracked, secrets scrubbed, tokens counted

**Fold Operation**:
1. LLM calls `context_return(message)`
2. Contextd runs Gitleaks scan on summary (defense-in-depth)
3. Stores folded summary in main collection
4. Archives full trajectory (encrypted if contains secrets)
5. Removes branch from active NATS state
6. Returns to parent (or main thread)

**Context Manager F(·)**:
- Removes folded content from working context
- Preserves only branch summary in main thread
- Maintains KV-cache efficiency (rollback to branch point)
- Provides metadata for LLM decisions (tokens saved, branch depth)

---

## Two-Tier Storage

**Main Collections** (compressed, fast retrieval):
```json
{
  "id": "checkpoint_abc123",
  "summary": "Fixed auth bug in middleware",
  "branch_context": {
    "branch_id": "br_xyz",
    "tokens_folded": 8500,
    "operations_count": 12
  },
  "secrets_scrubbed": 2,
  "clean": true
}
```

**Archive Collections** (full detail, meta-learning):
```json
{
  "branch_id": "br_xyz",
  "operations": [
    {
      "id": "op_001",
      "type": "file_read",
      "file_path": "auth/middleware.go",
      "content_redacted": "...",
      "content_unredacted": "...",  // Encrypted
      "secrets_found": 2,
      "tokens": 1200
    }
  ],
  "total_tokens": 8500,
  "folded_to": "checkpoint_abc123"
}
```

---

## Multi-Tenant Integration

**Database-Per-Project** (existing architecture):
- Each project path → `project_<hash>` database
- Git worktrees = different paths = separate databases
- Branch state scoped: `{owner_id}/{project_hash}/{session_id}`

**Isolation Guarantees**:
- Different worktrees cannot see each other's branches
- Pre-fetch events route to correct project database
- Meta-learning strategies shared across projects (in `shared` database)

**Example** (3 worktrees):
```
/projects/myapp (main)        → project_abc123 → session_001 → br_001
/projects/myapp-feature (wt1) → project_def456 → session_002 → br_002
/projects/myapp-hotfix (wt2)  → project_ghi789 → session_003 → br_003
```

All independent, no cross-talk.

---

## Integration Points

**MCP Protocol** (HTTP):
- Base URL: `http://localhost:9090/mcp`
- Transport: `{"type": "http"}`
- Tools: `context_branch`, `context_return`, `context_branch_status`, etc.

**NATS Event Bus**:
- Branch state: Ephemeral, session-scoped
- Events: `branch.created`, `branch.folded`, `remediation.saved`
- Meta-learning triggers: `meta_learning.analyze.scheduled`

**Qdrant Storage**:
- Main collections: Folded summaries (fast search)
- Archive collections: Full trajectories (analysis)
- Shared collections: Strategies (cross-project learning)

**Gitleaks SDK**:
- Execution-time: Scan all file operations
- Fold-time: Scan return summaries (double-scan)
- Redaction format: `[REDACTED:rule-id:preview]`

**OTLP/Prometheus**:
- Real-time metrics: Token counts, branch operations
- Batch analysis: Velocity (time-to-solution), accuracy (success rate)
- Composite scoring: 40% token efficiency + 30% velocity + 30% accuracy

---

## Workflow Example

**User**: "Debug authentication error across 10 API endpoints"

**LLM Decision** (guided by heuristics):
```
Main thread (tokens: 5K)
  ├─ context_branch("Search API logs")       → br_001 (8K tokens)
  │  └─ context_return("Found 3 error patterns")
  │
  ├─ context_branch("Test endpoint 1")       → br_002 (4K tokens)
  │  └─ context_return("Auth token expired")
  │
  └─ context_branch("Fix token refresh")     → br_003 (6K tokens)
     └─ context_return("Applied fix, tests pass")

Final main thread: 5K + 150 + 100 + 120 = 5.4K tokens (not 23K!)
```

**Without folding**: 5K + 8K + 4K + 6K = 23K tokens in working context
**With folding**: 5.4K tokens (summaries only), 18K tokens saved (78% compression)

---

## Design Decisions

**No RL Training** (simplified from research):
- Paper uses FoldGRPO (reinforcement learning) to train branch/fold behavior
- We use heuristics instead (unfolded token penalty, out-of-scope detection)
- Simpler implementation, still effective

**HTTP Transport**:
- HTTP recommended for cloud services

**Hybrid State Management**:
- Active state in NATS (real-time, ephemeral)
- Snapshots in Qdrant (persistence, recovery)
- Best of both: Fast coordination + crash recovery

**Configurable Heuristics** (not hard-coded):
- Users control warning thresholds
- Optional hard limits (auto-fold on violation)
- Adapts to different workflows

---

## Success Criteria

**Context Compression**: 10× reduction (research paper benchmark)
**Search Performance**: <100ms latency (no degradation)
**Secret Detection**: 100% scan coverage (zero leaks)
**Test Coverage**: >80% (all new code)

---

[← Back to Index](INDEX.md) | [Next: MCP Protocol →](02-mcp-protocol.md)
