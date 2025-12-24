---
name: error-remediation
description: Use when encountering any error, exception, or unexpected behavior - diagnoses with troubleshoot_diagnose, searches past fixes with remediation_search, and records new solutions with remediation_record
---

# Error Remediation

## Overview

Don't re-solve solved problems. Search for past fixes first, record new ones for the future.

## When to Use

**Any error encountered**:
- Build failures
- Test failures
- Runtime exceptions
- Unexpected behavior
- Configuration issues

## The Error Resolution Flow

```
┌─────────────────────────────────────────┐
│  1. DIAGNOSE the error                  │
│     troubleshoot_diagnose(error_message,│
│                          error_context) │
├─────────────────────────────────────────┤
│  2. SEARCH for past fixes               │
│     remediation_search(query, tenant_id)│
├─────────────────────────────────────────┤
│  3. APPLY fix or investigate further    │
│     (use diagnosis + past remediations) │
├─────────────────────────────────────────┤
│  4. RECORD the solution                 │
│     remediation_record(title, problem,  │
│       root_cause, solution, category)   │
└─────────────────────────────────────────┘
```

## Tool Reference

### troubleshoot_diagnose

```json
{
  "error_message": "cannot use vectorStore as vectorstore.Store value: missing method DeleteDocumentsFromCollection",
  "error_context": "Running go test ./... after adding new interface method"
}
```

Returns:
- `root_cause`: Likely cause
- `hypotheses`: Ranked diagnostic theories
- `recommendations`: Suggested actions
- `related_patterns`: Similar known issues
- `confidence`: Overall confidence (0-1)

### remediation_search

```json
{
  "query": "interface implementation missing method",
  "tenant_id": "fyrsmithlabs",
  "category": "syntax",
  "limit": 5
}
```

**Categories**: `syntax`, `runtime`, `logic`, `config`, `dependency`, `network`, `auth`, `data`, `performance`, `other`

### remediation_record

```json
{
  "title": "Mock missing interface method after interface change",
  "problem": "Test fails with 'missing method X' after adding method to interface",
  "symptoms": ["go test fails", "cannot use X as Y value", "missing method"],
  "root_cause": "Mock implementations don't automatically get new interface methods",
  "solution": "Add the new method to all mock implementations of the interface",
  "code_diff": "func (m *mockStore) NewMethod(...) { ... }",
  "affected_files": ["service_test.go"],
  "category": "syntax",
  "confidence": 0.9,
  "tags": ["go", "interfaces", "mocks", "testing"],
  "tenant_id": "fyrsmithlabs",
  "scope": "org"
}
```

**Scopes** control BOTH storage visibility AND search behavior:

| Scope | Stored For | Searchable From |
|-------|------------|-----------------|
| `project` | This project only | This project only |
| `team` | Team members | Team projects (with `include_hierarchy=true`) |
| `org` | Entire organization | Any project (with `include_hierarchy=true`) |

Default search is project-only. Set `include_hierarchy=true` to search parent scopes.

## What to Record

**Good remediations**:
- Non-obvious root causes
- Errors with misleading messages
- Multi-step solutions
- Environment-specific fixes

**Include**:
- Exact error message (for search matching)
- Root cause (not just symptoms)
- Complete solution steps
- Affected files

## Quick Reference

| Step | Tool | Purpose |
|------|------|---------|
| 1 | `troubleshoot_diagnose` | Get AI analysis |
| 2 | `remediation_search` | Find past fixes |
| 3 | Apply fix | Use diagnosis + history |
| 4 | `remediation_record` | Save for future |

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Skipping diagnosis | `troubleshoot_diagnose` first, always |
| Not searching history | Someone may have fixed this before |
| Recording symptoms only | Document root cause and full solution |
| Too narrow scope | Use `org` scope unless truly project-specific |
| Vague problem description | Include exact error message |
