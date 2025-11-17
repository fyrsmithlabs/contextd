# 5. Branch-Aware Features

[← Back to Meta-Learning](04-meta-learning.md) | [Next: Secret Scrubbing →](06-secret-scrubbing.md)

---

## Overview

Existing features (checkpoints, remediations, skills) become branch-aware, enabling powerful workflows like "restore to checkpoint in branch 2" or "this remediation works best in isolated branches."

---

## Branch-Aware Checkpoints

**Enhanced Storage**:
```json
{
  "id": "checkpoint_abc123",
  "summary": "Completed feature X",
  "branch_context": {
    "session_id": "sess_xyz",
    "active_branch_id": "br_123",
    "branch_depth": 2,
    "branch_path": ["main", "br_001", "br_123"],
    "tokens_at_checkpoint": {
      "main": 5000,
      "br_001": 12000,
      "br_123": 3000
    }
  }
}
```

**New MCP Tool**: `checkpoint_restore_with_branches`

Restores checkpoint AND reconstructs branch state in NATS.

---

## Branch-Aware Remediations

**Track Branch Patterns**:
```json
{
  "id": "remediation_abc123",
  "error_msg": "TypeError: cannot read property 'length' of undefined",
  "solution": "Add null check",
  "branch_patterns": {
    "works_best_in_branch": true,
    "isolation_recommended": true,
    "avg_token_cost": 2500
  }
}
```

**Enhanced Search Response**:
```json
{
  "remediations": [...],
  "branching_recommendation": {
    "recommended": true,
    "reason": "87% of similar errors resolved faster in isolated branches"
  }
}
```

---

## Branch-Aware Skills

**Branch Templates**:
```json
{
  "id": "skill_tdd_workflow",
  "name": "TDD Workflow",
  "branch_template": {
    "uses_branches": true,
    "typical_branch_count": 3,
    "branch_patterns": [
      {"prompt": "Write failing test", "avg_tokens": 1500},
      {"prompt": "Implement code", "avg_tokens": 2000},
      {"prompt": "Refactor", "avg_tokens": 1000}
    ]
  }
}
```

**New MCP Tool**: `skill_apply_with_branching`

Auto-creates recommended branches based on skill template.

---

[← Back to Meta-Learning](04-meta-learning.md) | [Next: Secret Scrubbing →](06-secret-scrubbing.md)
