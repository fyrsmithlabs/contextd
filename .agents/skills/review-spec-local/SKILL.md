---
name: review-spec-local
specializes: review-spec
description: Repo-specific spec-review guidance for contextd. Only the categories declared overridable by the core review-spec skill may be specialized here.
---

# Repo-specific spec-review guidance for `contextd`

This file is a companion to the core `review-spec` skill. It does not
redefine the review output schema, severity labels, safety rules, or
evidence rules. It only specializes the override categories the core
skill marks as overridable.

## Spec location and file conventions

Specs in this repo live under `docs/spec/<topic>/` (not `specs/GH<issue-number>/`).
A topic folder typically contains a subset of:

- `SPEC.md` — the canonical specification of behavior
- `ARCH.md` — architectural diagrams and component layout
- `DESIGN.md` — design rationale and alternatives considered
- `CONSENSUS-REVIEW.md` — captured review notes from prior consensus passes

When reviewing a spec PR:

- The originating GitHub issue (if any) should be linked at the top of the relevant file.
- New top-level topics should have at minimum a `SPEC.md`; `ARCH.md` is required when the change introduces new components or interfaces.
- Code references should use the `path/file:line` convention so they resolve in editors.

## Tenant and security invariants

Any spec that introduces a new vectorstore-touching subsystem must explicitly state how it handles tenant context (`TenantInfo`, `ContextWithTenant`, `ApplyTenantFilters`). Flag specs that gloss over isolation, since "fail-closed on missing tenant" is a non-negotiable invariant.

Any spec that introduces new outbound responses (MCP tool output, HTTP responses, log lines that include user payloads) must state where gitleaks scrubbing happens.

## Linking conventions

- Prefer repo-root-relative links over absolute filesystem paths in spec prose.
- When a spec references another spec in the same repository, link to it via its relative path under `docs/spec/`.
- Cross-link related specs in a "See Also" section near the end of `SPEC.md`.

## Migration and rollout

Specs that change wire formats, the MCP tool surface, or storage layouts must include a "Migration" section that names the affected callers (CLI, marketplace plugin, downstream agents). Flag specs that change shape without acknowledging the dependent ecosystem.
