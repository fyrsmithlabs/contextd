# 4. Meta-Learning Layer

[← Back to Process Rewards](03-process-rewards.md) | [Next: Branch-Aware Features →](05-branch-aware-features.md)

---

## Overview

Meta-learning analyzes patterns across sessions to optimize strategy selection. Composite scoring (40% token efficiency + 30% velocity + 30% accuracy) ranks strategies, NATS events trigger updates.

---

## Composite Score Formula

```
strategy_score = (0.4 × token_efficiency) + (0.3 × velocity) + (0.3 × accuracy)

Where:
- token_efficiency = 1 - (tokens_used / baseline_tokens)
- velocity = baseline_time / time_to_solution  
- accuracy = success_rate (remediation worked, task completed)
```

---

## Prometheus Metrics

**Real-time** (collected during operation):
```
contextd_branch_tokens_total{branch_id, session_id, project_hash}
contextd_folded_tokens_saved{branch_id}
```

**Session-end** (calculated after completion):
```
contextd_session_duration_seconds{session_id, outcome}
contextd_remediation_success_total{remediation_id}
```

---

## Strategy Storage

**Collection**: `strategies` (in `shared` database for cross-project learning)

```json
{
  "id": "strategy_abc123",
  "embedding": [...],
  "payload": {
    "strategy_type": "search_failure_recovery",
    "pattern": "semantic_search -> hybrid_matching -> keyword_fallback",
    "composite_score": 0.82,
    "token_efficiency": 0.85,
    "velocity": 0.80,
    "accuracy": 0.81,
    "usage_count": 156,
    "success_count": 126
  }
}
```

---

## NATS Event Triggers

**Real-time**:
- `remediation.saved` → Update strategy accuracy
- `branch.folded` → Update token efficiency

**Scheduled** (every 1000 ops OR 6 hours):
- `meta_learning.analyze` → Deep analysis, recalculate scores

---

## Strategy Recommendation

**MCP Tool**: `context_recommend_strategy`

**Input**:
```json
{
  "problem_type": "search_failure",
  "context": {
    "current_tokens": 25000,
    "codebase_size": 100000
  }
}
```

**Response**:
```json
{
  "recommended_strategy": {
    "pattern": "semantic_search -> hybrid_matching",
    "composite_score": 0.82,
    "confidence": 0.91,
    "expected_outcomes": {
      "token_efficiency": "15-20% savings",
      "success_probability": 0.81
    }
  }
}
```

---

[← Back to Process Rewards](03-process-rewards.md) | [Next: Branch-Aware Features →](05-branch-aware-features.md)
