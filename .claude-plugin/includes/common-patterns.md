## Common Contextd Patterns

### SRE Debug Process (5 Steps)

| Step | Action | Contextd Tools |
|------|--------|----------------|
| **1. Problem Report** | Collect error, stack trace, reproduction steps | - |
| **2. Triage** | Assess severity, search known fixes | `troubleshoot_diagnose`, `remediation_search` |
| **3. Examine** | Find relevant code, gather context | `semantic_search`, `memory_search` |
| **4. Diagnose** | Form hypotheses, test in isolation | `branch_create` for each hypothesis |
| **5. Test/Treat** | Apply fix, verify, record learning | `remediation_record`, `memory_record` |

**Quick debug flow:**
```
1. troubleshoot_diagnose(error_message, stack_trace)
2. remediation_search(query, include_hierarchy: true)
3. semantic_search(query, project_path)
4. branch_create(...) for each hypothesis
5. branch_return(branch_id, "CONFIRMED/REJECTED: evidence")
6. Apply fix -> remediation_record(...)
```

---

### Context Folding Patterns

**When to use:**
- Exploring multiple files to find function/class
- Testing multiple debugging hypotheses
- Researching API docs or web sources
- Any task where process is verbose but result is concise

**When NOT to use:**
- Single file changes (no benefit)
- User needs full reasoning visible
- Simple, focused tasks

**Budgets:**
| Task Complexity | Budget | Use Case |
|-----------------|--------|----------|
| Simple lookup | 4,096 | Find function, check file |
| Moderate analysis | 8,192 | Analyze component, test hypothesis |
| Complex exploration | 12,288 | Multi-file analysis |
| Large investigation | 16,384 | Architecture analysis, debugging |

**File exploration:**
```
branch_create(session_id, "Find auth handler", "Search src/ for authenticate()", 4096)
# Branch reads files, returns: "Found in src/auth.go:42"
# Main context grows ~50 tokens, not 10K+
```

**Hypothesis testing:**
```
branch_create(session_id, "Test: race condition",
  "Check if symptom caused by race. Return: CONFIRMED/REJECTED with evidence", 6144)
```

---

### Pre-flight / Post-flight Workflow

**Pre-flight (BEFORE any work):**
```
semantic_search(query, project_path)  # Find relevant code
memory_search(project_id, query)      # Check past learnings
remediation_search(query, ...)        # If error-related
```

**Post-flight (AFTER work):**
```
memory_record(project_id, title, content, outcome, tags)
remediation_record(...)               # If error was fixed
memory_outcome(memory_id, succeeded)  # If used a memory
```

**Rules:**
- NEVER skip pre-flight or post-flight
- ALWAYS record learnings
- Report contextd actions in response

---

### Checkpoint Strategy

**When to checkpoint:**
- Context usage at 70%+
- Before risky changes
- Between major task phases
- Before compaction (auto via PreCompact hook)

**Naming convention:**
```
pre-{task}-{description}   # Before starting
step-{N}-{description}     # During incremental work
post-{task}-complete       # After completion
```

---

### Learning Capture

**Success pattern:**
```
memory_record(
  title: "[Category]: Brief description",
  content: "Approach: [what worked]. Key insight: [learning].",
  outcome: "success",
  tags: ["category", "technology"]
)
```

**Failure pattern:**
```
memory_record(
  title: "[Category] failure: what went wrong",
  content: "Attempted: [approach]. Failed: [cause]. Better: [alternative].",
  outcome: "failure",
  tags: ["failure", "lesson"]
)
```

---

### Response Format (Required)

All contextd-enabled responses include:
```
## [Task Type] Result
[What was accomplished]

## Contextd Actions
- Pre-flight: [semantic_search + memory_search findings]
- Work: [checkpoints, branches used]
- Post-flight: [memories/remediations recorded]
```
