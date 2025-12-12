---
name: consensus-review
description: Use when reviewing code changes, PRs, plugins, or documentation using multiple specialized agents in parallel - dispatches Security, Correctness, Architecture, and UX agents then synthesizes findings
---

# Consensus Review

Multi-agent code review that dispatches 4 specialized reviewers in parallel, then synthesizes their findings into actionable recommendations.

## When to Use

- Reviewing pull requests before merge
- Evaluating plugin configurations and skills
- Auditing security-sensitive code changes
- Assessing architectural decisions
- Getting comprehensive feedback on documentation

## When NOT to Use

- Single-file typo fixes (overkill)
- Simple refactoring with no behavioral changes
- When you need quick feedback (use single reviewer instead)

## The 4 Reviewers

| Agent | Focus Areas |
|-------|-------------|
| **Security** | Injection, secrets, supply chain, permissions |
| **Correctness** | Logic errors, schema compliance, edge cases |
| **Architecture** | Structure, maintainability, patterns, extensibility |
| **UX/Documentation** | Clarity, completeness, examples, discoverability |

## Workflow

### Step 1: Identify Scope

Determine what to review:
- Specific files (PR diff)
- Directory (plugin, package)
- Commit range

### Step 2: Dispatch Agents in Parallel

Launch all 4 agents simultaneously using the Task tool:

```
Task(subagent_type="general-purpose", run_in_background=true):
  "You are a SECURITY REVIEWER analyzing [scope].
   Focus on: injection, secrets exposure, supply chain, permissions.
   For each issue: Severity (CRITICAL/HIGH/MEDIUM/LOW), Location, Issue, Recommendation."

Task(subagent_type="general-purpose", run_in_background=true):
  "You are a CORRECTNESS REVIEWER analyzing [scope].
   Focus on: logic errors, schema compliance, edge cases, validation.
   For each issue: Severity, Location, Issue, Recommendation."

Task(subagent_type="general-purpose", run_in_background=true):
  "You are an ARCHITECTURE REVIEWER analyzing [scope].
   Focus on: structure, patterns, maintainability, extensibility.
   For each issue: Severity, Location, Issue, Recommendation."

Task(subagent_type="general-purpose", run_in_background=true):
  "You are a UX/DOCUMENTATION REVIEWER analyzing [scope].
   Focus on: clarity, completeness, examples, error guidance.
   For each issue: Severity, Location, Issue, Recommendation."
```

### Step 3: Collect Results

Use AgentOutputTool to retrieve results from all 4 agents:
- Wait for all to complete
- Collect findings from each

### Step 4: Synthesize

Combine findings into consensus report:

1. **Tally by severity** - Count issues per severity across all agents
2. **Identify consensus** - Issues flagged by multiple agents = higher priority
3. **De-duplicate** - Merge similar findings
4. **Prioritize** - Critical > High > Medium > Low

### Step 5: Present Recommendations

Format output as:

```markdown
# Consensus Review: [Subject]

## Summary
| Agent | Issues | Critical | High | Medium | Low |
|-------|--------|----------|------|--------|-----|
| Security | N | n | n | n | n |
| Correctness | N | n | n | n | n |
| Architecture | N | n | n | n | n |
| UX/Docs | N | n | n | n | n |

## Critical Issues (Must Fix)
[Issues with CRITICAL severity from any agent]

## High Priority (Should Fix)
[Issues with HIGH severity OR flagged by 2+ agents]

## Recommendations
[Prioritized action items]
```

### Step 6: Record Memory

After completing review, record findings:

```
memory_record(
  project_id: "<project>",
  title: "Consensus review of [subject] - [date]",
  content: "Key findings: [summary]. Critical: [count]. Recommendations: [top 3]",
  outcome: "success",
  tags: ["code-review", "consensus", "<subject-type>"]
)
```

## Severity Definitions

| Severity | Definition | Action |
|----------|------------|--------|
| **CRITICAL** | Blocks release, security vulnerability, data loss risk | Fix immediately |
| **HIGH** | Significant issue, poor UX, maintainability risk | Fix before merge |
| **MEDIUM** | Should improve, technical debt | Plan to fix |
| **LOW** | Nice to have, minor polish | Backlog |

## Common Mistakes

| Mistake | Prevention |
|---------|------------|
| Running agents sequentially | Always use `run_in_background=true` for all 4 |
| Not waiting for all agents | Use `AgentOutputTool(block=true)` for each |
| Skipping synthesis | Raw output is overwhelming - always synthesize |
| Not recording memory | Capture findings for future reference |

## Example Prompts for Agents

### Security Agent
```
You are a SECURITY REVIEWER analyzing the contextd Claude Code plugin.

Review files in: /path/to/.claude-plugin/

Focus on:
1. Script Security: Command injection, unsafe downloads, privilege escalation
2. Data Exposure: Secrets in configs, PII handling
3. Supply Chain: Download verification, URL safety, version pinning
4. MCP Security: Tool permissions, environment variable handling

For each issue found, provide:
- Severity: CRITICAL / HIGH / MEDIUM / LOW
- Location: file:line or file (general)
- Issue: What's wrong
- Recommendation: How to fix

Return a structured report.
```

### Correctness Agent
```
You are a CORRECTNESS REVIEWER analyzing [scope].

Focus on:
1. JSON Validity: All JSON files parse correctly
2. Schema Compliance: Tool schemas match actual signatures
3. Logic Errors: Edge cases, validation gaps
4. Version Consistency: Versions match across files

For each issue: Severity, Location, Issue, Recommendation.
```

### Architecture Agent
```
You are an ARCHITECTURE REVIEWER analyzing [scope].

Focus on:
1. Structure: Follows conventions and patterns
2. Organization: Components are focused, non-overlapping
3. Design: Intuitive naming, proper documentation
4. Maintainability: Easy to update, version, extend

For each issue: Severity, Location, Issue, Recommendation.
```

### UX/Documentation Agent
```
You are a UX/DOCUMENTATION REVIEWER analyzing [scope].

Focus on:
1. Clarity: Instructions clear and unambiguous
2. Completeness: All necessary steps documented
3. Examples: Sufficient examples for each feature
4. Error Guidance: What to do when things go wrong
5. Discoverability: Users can find what they need

For each issue: Severity, Location, Issue, Recommendation.
```

## Quick Reference

| Step | Action |
|------|--------|
| 1 | Identify files/scope to review |
| 2 | Dispatch 4 agents in parallel (Task tool, background) |
| 3 | Wait for all results (AgentOutputTool) |
| 4 | Synthesize: tally, de-duplicate, prioritize |
| 5 | Present summary table + recommendations |
| 6 | Record memory with key findings |
