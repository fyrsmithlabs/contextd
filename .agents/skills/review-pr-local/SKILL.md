---
name: review-pr-local
specializes: review-pr
description: Repo-specific PR review guidance for contextd. Only the categories declared overridable by the core review-pr skill may be specialized here.
---

# Repo-specific PR review guidance for `contextd`

This file is a companion to the core `review-pr` skill. It does not
redefine the review output schema, severity labels, safety rules, or
evidence rules. It only specializes the override categories the core
skill marks as overridable.

## Tenant context propagation

- Any code path that touches the vectorstore must accept `context.Context` and rely on `vectorstore.TenantFromContext(ctx)` for filtering. Flag any new public function that takes a vectorstore call but does not thread `ctx` through.
- Flag any user-supplied filter map that is passed to a vectorstore search without first running through `vectorstore.ApplyTenantFilters` — filter injection is the documented threat model.
- Calls that explicitly use `NewNoIsolation()` outside of `_test.go` files are a P0 bug.

## Secret scrubbing

- All MCP tool responses, branch returns, and HTTP `/api/v1/scrub` outputs must pass through the gitleaks scrubber (`internal/secrets`). Flag any new response path that bypasses it.
- New string fields surfaced to the user that could contain command output, environment values, or file contents need scrubbing before they leave the process.

## Test coverage expectations

The repo has documented coverage floors in `CLAUDE.md`: `secrets` 97%, `project` 97%, `reasoningbank` 82%, `remediation` 82%. PRs that materially reduce coverage in those packages need to either restore it or explain why the regression is acceptable.

## Graceful degradation

- chromem is the default vectorstore; Qdrant is optional. New features must work with chromem alone — flag any code path that hard-depends on Qdrant without a chromem fallback.
- Embedding provider failures should degrade to grep fallback (the `semantic_search` MCP tool pattern), not return an empty result silently.

## Debugging and observability

- Do not suggest removing structured log fields (zap) or OTEL spans from error paths. These are load-bearing for the institutional-knowledge replay.
- Flag log statements that print raw user input, vectorstore payloads, or memory contents — those must go through the scrubber if they are emitted at INFO or above.

## Backwards compatibility

- The MCP tool surface in `internal/mcp/tools.go` is consumed by external agents. Renames or signature changes need a deprecation note in the PR description or they should be flagged.
- HTTP API shape (`/api/v1/scrub`, `/api/v1/threshold`, `/api/v1/status`) is consumed by the `ctxd` CLI and the marketplace plugin. Breaking changes here require the plugin to be updated in the same PR or a follow-up issue.
