---
name: consensus-review
description: Use when reviewing code changes, PRs, or plugins using multiple specialized agents in parallel - dispatches Security, Correctness, Architecture, and UX agents then synthesizes findings
---

# Consensus Review

Multi-agent code review that dispatches 4 specialized reviewers in parallel, then synthesizes findings into actionable recommendations.

## When to Use

- Reviewing pull requests before merge
- Evaluating plugin configurations
- Auditing security-sensitive code
- Assessing architectural decisions

## When NOT to Use

- Single-file typo fixes (overkill)
- Simple refactoring with no behavioral changes
- When you need quick feedback (use single reviewer)

## The 4 Reviewers

| Agent | Focus Areas |
|-------|-------------|
| **Security** | Injection, secrets, supply chain, permissions |
| **Correctness** | Logic errors, schema compliance, edge cases |
| **Architecture** | Structure, maintainability, patterns, extensibility |
| **UX/Documentation** | Clarity, completeness, examples, discoverability |

## Workflow

### Step 1: Identify Scope

Determine what to review: specific files (PR diff), directory, or commit range.

### Step 2: Dispatch Agents in Parallel

Launch all 4 agents using Task tool with `run_in_background=true`:

```
Task(subagent_type="general-purpose", run_in_background=true):
  "You are a SECURITY REVIEWER analyzing [scope].
   Focus on: injection, secrets exposure, supply chain, permissions.
   For each issue: Severity (CRITICAL/HIGH/MEDIUM/LOW), Location, Issue, Recommendation."

Task(subagent_type="general-purpose", run_in_background=true):
  "You are a CORRECTNESS REVIEWER analyzing [scope]..."

Task(subagent_type="general-purpose", run_in_background=true):
  "You are an ARCHITECTURE REVIEWER analyzing [scope]..."

Task(subagent_type="general-purpose", run_in_background=true):
  "You are a UX/DOCUMENTATION REVIEWER analyzing [scope]..."
```

### Step 3: Collect & Synthesize

1. Wait for all agents with `AgentOutputTool(block=true)`
2. Tally by severity across all agents
3. Identify consensus (issues flagged by 2+ agents = higher priority)
4. De-duplicate similar findings
5. Prioritize: Critical > High > Medium > Low

### Step 4: Present Report

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
[Issues with CRITICAL severity]

## High Priority (Should Fix)
[Issues with HIGH severity OR flagged by 2+ agents]

## Recommendations
[Prioritized action items]
```

### Step 5: Record Memory

```
memory_record(
  project_id: "<project>",
  title: "Consensus review of [subject]",
  content: "Key findings: [summary]. Critical: [count]. Recommendations: [top 3]",
  outcome: "success",
  tags: ["code-review", "consensus"]
)
```

## Severity Definitions

| Severity | Definition | Action |
|----------|------------|--------|
| **CRITICAL** | Security vulnerability, data loss risk | Fix immediately |
| **HIGH** | Significant issue, maintainability risk | Fix before merge |
| **MEDIUM** | Technical debt, should improve | Plan to fix |
| **LOW** | Minor polish, nice to have | Backlog |

## Common Mistakes

| Mistake | Prevention |
|---------|------------|
| Running agents sequentially | Always use `run_in_background=true` |
| Not waiting for all agents | Use `AgentOutputTool(block=true)` for each |
| Skipping synthesis | Raw output is overwhelming - always synthesize |
| Not recording memory | Capture findings for future reference |

## Quick Reference

| Step | Action |
|------|--------|
| 1 | Identify files/scope |
| 2 | Dispatch 4 agents in parallel |
| 3 | Wait for all results |
| 4 | Synthesize: tally, de-duplicate, prioritize |
| 5 | Present summary + recommendations |
| 6 | Record memory with findings |
