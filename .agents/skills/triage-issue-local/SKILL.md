---
name: triage-issue-local
specializes: triage-issue
description: Repo-specific triage guidance for contextd. Only the categories declared overridable by the core triage-issue skill may be specialized here.
---

# Repo-specific triage guidance for `contextd`

This file is a companion to the core `triage-issue` skill. It does not
redefine the triage output schema, safety rules, or follow-up-question
contract. It only specializes the override categories the core skill
marks as overridable.

## Heuristics

- Before asking any follow-up question, try to answer it via `mcp__contextd__semantic_search` against the repo, the package READMEs in `internal/`, or `CLAUDE.md`. Only ask the reporter for details only they would know.
- Reproducibility for this repo depends heavily on **environment** (chromem vs Qdrant, FastEmbed model availability, MCP transport in use, OS). Default to asking which vectorstore backend and which embedding provider are configured when a search/index/memory bug is reported.
- Distinguish symptoms in the **MCP layer** (tool call errors, schema mismatches) from symptoms in the **service layer** (vectorstore filter errors, embedding failures). The fix lives in different packages.

## Label taxonomy

The label taxonomy for this repository is managed in `.github/issue-triage/config.json`. Prefer labels from that configuration. Heuristic mappings:

- Mentions of `chromem`, `qdrant`, `payload isolation`, `tenant` → `vectorstore`
- Mentions of `memory_*`, `reasoning`, `confidence scoring` → `reasoningbank`
- Mentions of `ctxd` (CLI), `cmd/ctxd` → `cli`
- Mentions of `git status capture`, `workstate` → `workstate`
- Mentions of `Temporal`, `workflow worker`, `webhook` → `temporal`
- Mentions of `gitleaks`, `secret leak`, `scrub`, `tenant injection` → `security`
- Failed tests or coverage drops → `testing`
- Latency / memory / throughput regressions → `performance`
- Phase rollouts, roadmap items → `phase-a` / `roadmap` / `epic`

Avoid inventing new labels; if a needed area is missing, propose it in the triage rather than auto-creating.

## Recurring follow-up patterns

When triaging vectorstore-related reports:

- Ask whether the user has `Isolation: NewNoIsolation()` set (only valid for tests).
- Ask whether the call passes `ctx` produced by `vectorstore.ContextWithTenant`.
- Ask which collection was searched and which tenant/team/project IDs were in context.

When triaging embedding-related reports:

- Ask which embedding provider is active (FastEmbed local ONNX, TEI, etc.) and whether the ONNX model has been auto-downloaded (`docs/spec/onnx-auto-download/`).

When triaging memory/recall reports:

- Ask whether the issue is "memory not found" (search side) or "memory not recorded" (record side), since the fixes diverge.

## Owner-inference hints

Solo-maintained by `@dahendel`; `.github/STAKEHOLDERS` carries the per-directory map. Do not suggest assignees outside the STAKEHOLDERS file without explicit maintainer input.
