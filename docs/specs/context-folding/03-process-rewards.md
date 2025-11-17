# 3. Process Reward Heuristics

[← Back to MCP Protocol](02-mcp-protocol.md) | [Next: Meta-Learning →](04-meta-learning.md)

---

## Overview

Process rewards guide effective branch/fold behavior without requiring RL training. Contextd calculates heuristics in real-time and returns warnings/scores to help LLM make better decisions.

---

## Unfolded Token Penalty

**Goal**: Encourage token-heavy operations in branches, not main thread.

**Calculation**:
```
penalty_score = main_thread_tokens / context_limit
if penalty_score > 0.5:
    warning_level = "high" if > 0.7 else "medium"
```

**Response Example**:
```json
{
  "result": {...},
  "context_health": {
    "main_thread_tokens": 18432,
    "context_limit": 32768,
    "penalty_score": 0.56,
    "warning": "medium",
    "suggestion": "Consider branching for token-heavy operations"
  }
}
```

**Configuration**:
```yaml
heuristics:
  unfolded_token_penalty:
    enabled: true
    warning_threshold: 0.5
    hard_limit: 0.8  # Optional: reject tool calls if exceeded
```

---

## Out-of-Scope Penalty

**Goal**: Keep branches focused on assigned subtask.

**Detection**: Compare operation semantics to branch prompt using embedding similarity.

**Implementation**:
1. Embed branch `prompt` (subtask description)
2. Embed each operation in branch (tool calls, file reads)
3. Calculate cosine similarity
4. Flag operations with similarity < 0.6

**On `context_return`, analyze branch**:
```json
{
  "branch_id": "br_abc123",
  "scope_analysis": {
    "operations_count": 15,
    "in_scope": 12,
    "out_of_scope": 3,
    "scope_score": 0.80,
    "warning": "Branch performed some out-of-scope operations",
    "details": [
      "Operation 'search remediation' (similarity: 0.45) diverged from prompt"
    ]
  }
}
```

**Configuration**:
```yaml
heuristics:
  out_of_scope_penalty:
    enabled: true
    similarity_threshold: 0.6
    auto_return: false  # Optional: auto-fold if out-of-scope detected
```

---

## Failure Penalty

**Goal**: Penalize failed tool calls to encourage error handling.

**Tracking**:
- Count failed operations per branch
- Calculate failure rate
- Warn if rate > threshold

**Response**:
```json
{
  "context_health": {
    "failure_rate": 0.15,
    "failed_operations": 3,
    "total_operations": 20,
    "warning": "High failure rate detected",
    "suggestion": "Review error messages and adjust approach"
  }
}
```

**Configuration**:
```yaml
heuristics:
  failure_penalty:
    enabled: true
    warning_threshold: 0.2
```

---

[← Back to MCP Protocol](02-mcp-protocol.md) | [Next: Meta-Learning →](04-meta-learning.md)
