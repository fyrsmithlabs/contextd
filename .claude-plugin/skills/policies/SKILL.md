---
name: policies
description: Use to manage agent policies - STRICT guardrails that MUST be followed at all times. Policies define constraints and requirements for consistent agent behavior across sessions.
---

# Agent Policies

## What Are Policies?

**Policies are STRICT guidelines that MUST be adhered to at all times.** They enforce guardrails for agents, ensuring consistent behavior across sessions.

Unlike skills (which provide capabilities), policies define **constraints and requirements**:
- "Always run tests before claiming a fix is complete"
- "Never use `--no-verify` with git commit"
- "Search contextd memory before filesystem search"
- "Consensus reviews must be APPROVE or REQUEST CHANGES only"

## Policy Schema

```
┌──────────────────────────────────────────────────────────────────┐
│  Policy                                                          │
├──────────────────────────────────────────────────────────────────┤
│  id: string           - Unique identifier (auto-generated)       │
│  name: string         - Short name (e.g., "test-before-fix")     │
│  rule: string         - The MUST statement                       │
│  description: string  - Why this policy exists                   │
│  category: string     - "verification", "process", "security"    │
│  severity: string     - "critical", "high", "medium"             │
│  scope: string        - "global", "skill:{name}", "project:{p}"  │
│  source: string       - Where it came from                       │
│  violations: int      - Times violated                           │
│  successes: int       - Times followed                           │
│  enabled: bool        - Active or disabled                       │
└──────────────────────────────────────────────────────────────────┘
```

## Storage Pattern

Policies are stored as memories using the `memory_record` MCP tool:

```
mcp__contextd__memory_record(
  project_id: "global",
  title: "POLICY: test-before-fix",
  content: "RULE: Always run tests before claiming a fix is complete.\nDESCRIPTION: Prevents false claims of completion. Tests verify the fix actually works.\nCATEGORY: verification\nSEVERITY: high\nSCOPE: global\nVIOLATIONS: 3\nSUCCESSES: 47",
  outcome: "success",
  tags: ["type:policy", "category:verification", "severity:high", "scope:global", "enabled:true"]
)
```

The `content` field uses a structured text format with key-value pairs separated by newlines. This allows parsing while remaining human-readable.

## Managing Policies

### List All Policies

Use `/policies` command or search directly:

```
memory_search(project_id: "global", query: "type:policy enabled:true")
```

### Add New Policy

Via `/policies add` command or directly:

```json
{
  "project_id": "global",
  "title": "POLICY: contextd-first-search",
  "content": "RULE: Always search contextd (memory_search, repository_search) before filesystem search (grep, glob).\nDESCRIPTION: Contextd has semantic understanding and past learnings. Raw grep bloats context.\nCATEGORY: process\nSEVERITY: high\nSCOPE: global\nVIOLATIONS: 0\nSUCCESSES: 0",
  "outcome": "success",
  "tags": ["type:policy", "category:process", "severity:high", "scope:global", "enabled:true"]
}
```

### Disable Policy

Update the memory with `enabled:false` tag (don't delete - preserves history).

### Record Violation

When a policy is violated during `/reflect`:

```
# Step 1: Provide negative feedback (lowers confidence)
mcp__contextd__memory_feedback(memory_id: "<policy_memory_id>", helpful: false)

# Step 2: Record violation event as separate memory for tracking
mcp__contextd__memory_record(
  project_id: "global",
  title: "VIOLATION: test-before-fix",
  content: "Policy violated at: 2024-12-19T10:00:00Z\nEvidence: Claimed fix without running tests\nSession: abc123",
  outcome: "failure",
  tags: ["type:policy-event", "event:violation", "policy:test-before-fix"]
)
```

Violation counts are derived by searching for violation events, not stored in the policy itself.

### Record Success

When a policy is followed correctly:

```
# Step 1: Provide positive feedback (raises confidence)
mcp__contextd__memory_feedback(memory_id: "<policy_memory_id>", helpful: true)

# Step 2: Record success event (optional, for high-value policies)
mcp__contextd__memory_record(
  project_id: "global",
  title: "SUCCESS: test-before-fix",
  content: "Policy followed at: 2024-12-19T10:00:00Z\nEvidence: Ran tests before marking fix complete",
  outcome: "success",
  tags: ["type:policy-event", "event:success", "policy:test-before-fix"]
)
```

Success events are optional - only record for critical policies or audit purposes.

## Categories

| Category | Description | Examples |
|----------|-------------|----------|
| **verification** | Testing, validation, confirmation | Run tests, verify builds, check outputs |
| **process** | Workflow steps, ordering | Search before grep, plan before code |
| **security** | Safety, permissions, credentials | Never read secrets, validate inputs |
| **quality** | Code standards, best practices | DRY, no magic numbers, document decisions |
| **communication** | How to interact with users | Confirm before destructive ops, clear summaries |

## Severity Levels

| Severity | When to Use | Example |
|----------|-------------|---------|
| **critical** | Security, data loss potential | "Never commit secrets" |
| **high** | Process integrity, correctness | "Run tests before claiming fix" |
| **medium** | Quality, maintainability | "Document design decisions" |

## Scopes

| Scope | Applies To | Example |
|-------|------------|---------|
| `global` | All sessions, all projects | "Always search contextd first" |
| `skill:{name}` | When specific skill is active | "When using TDD, write test first" |
| `project:{path}` | Specific project only | "This repo requires Go 1.21+" |

## Policy Enforcement

### At Skill Load

When a skill is invoked, search for applicable policies and inject them into the skill's context:

```
# Step 1: Search for applicable policies
policies = mcp__contextd__memory_search(
  project_id: "global",
  query: "type:policy scope:global enabled:true"
)

skill_policies = mcp__contextd__memory_search(
  project_id: "global",
  query: "type:policy scope:skill:{skill_name} enabled:true"
)

# Step 2: Parse policy content to extract rules
for each policy in policies + skill_policies:
  - Parse RULE: line from content
  - Parse SEVERITY: line from content

# Step 3: Inject as constraint block at skill start
"""
## Active Policies (MUST FOLLOW)
- [critical] Never read secrets into context
- [high] Always run tests before claiming fix
- [high] Search contextd before filesystem search
"""
```

This injection happens at the START of skill execution, before any other work.

### During /reflect

The reflect command evaluates policy compliance:

1. Search all enabled policies
2. Check recent actions against policy rules
3. Flag violations with evidence
4. Update violation/success counts
5. Generate compliance report

## Built-in Policies (Recommended)

These policies are derived from common agent failure patterns:

### Critical

| Policy | Rule |
|--------|------|
| no-secrets-in-context | Never read secrets (.env, credentials) into context |
| no-force-push-main | Never force push to main/master branch |
| no-skip-verification | Never use --no-verify flag |

### High

| Policy | Rule |
|--------|------|
| test-before-fix | Always run tests before claiming a fix is complete |
| contextd-first | Search contextd before filesystem search |
| verify-before-complete | Run verification commands before marking task complete |

### Medium

| Policy | Rule |
|--------|------|
| consensus-binary | Consensus reviews must be APPROVE or REQUEST CHANGES only |
| document-decisions | Record significant design decisions with rationale |
| search-before-assume | Search before assuming something doesn't exist |

## Creating Policies from Conversations

Policies can be extracted from past conversations during `/onboard`:

1. Scan conversation JSONL for correction patterns:
   - "You should have..."
   - "Always do X before Y"
   - "Never do X"
   - User corrections of agent behavior

2. Extract rule and context
3. Deduplicate against existing policies
4. Store with `source:conversation:{uuid}:{turn}`

## Quick Reference

| Action | Command/Tool |
|--------|--------------|
| List policies | `/policies` |
| Add policy | `/policies add` |
| Remove policy | `/policies remove {id}` |
| View stats | `/policies stats` |
| Search policies | `memory_search(query: "type:policy ...")` |
| Check compliance | `/reflect` |

## Integration with /reflect

The reflect command uses policies as the source of truth for compliance checking:

```
/reflect →
  1. Search policies (type:policy enabled:true)
  2. Search recent memories for actions
  3. Match actions against policy rules
  4. Flag violations
  5. Update policy stats
  6. Generate report
```

See `/contextd:reflect` for full workflow.
