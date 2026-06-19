# MCP Protocol Expansion — Implementation Plan

**Status**: In progress
**Created**: 2026-06-19
**Branch**: `claude/mcp-full-protocol-v8of4b`

## Goal

Expand contextd's MCP server from a Tools-only/stdio surface to the full MCP
server surface — Resources, Prompts, and client primitives (logging,
elicitation) — plus remote hosting over Streamable HTTP and an agent-swarm
notification mechanism.

Evaluated against: https://modelcontextprotocol.io/docs/learn/architecture

## Decisions (locked 2026-06-19)

| # | Decision | Choice |
|---|----------|--------|
| Prompts | Richness | **Static instruction templates** mirroring the slash commands (no per-prompt service calls) |
| Auth | Remote HTTP auth timing | **Bearer token now** — shared-token middleware on the Echo MCP server |
| #6 | Swarm notifications depth | **Design doc + full impl** (subscribe + `ResourceUpdated` broadcast + tests) |
| HTTP | REST/MCP consolidation | **Later** — keep separate Echo servers for now |

## Completed

- Streamable HTTP transport as a **separate Echo server** (`internal/mcp/streamable.go`):
  `Server.StreamableHandler`, `Server.RunHTTP`. Stateful sessions (required for
  swarm subscriptions). `--mcp-http-port` / `--mcp-http-host` flags.
- serverInfo name `contextd-v2` → `contextd` (#5).

## Phases

### Phase 0 — Bearer-token auth (pull-forward of Phase E auth)
- `StreamableHTTPConfig.Token`; Echo middleware on the `/mcp` route requiring
  `Authorization: Bearer <token>` when a token is configured.
- `/health` stays unauthenticated.
- Token source: `--mcp-http-token` flag and/or `CONTEXTD_MCP_HTTP_TOKEN` env.
- If no token is configured: serve open but log a prominent warning (intended
  for localhost/testing only).
- Tests: 401 without/with-wrong token, 200 with correct token, open+warn when unset.

### Phase A — Resources (#1) · `internal/mcp/resources.go`
- URI scheme: `contextd://{project_id}/<kind>[/{id}]`.
- Collection templates: `…/memories`, `…/checkpoints`, `…/remediations`.
- By-id templates: `…/memory/{id}`, `…/checkpoint/{id}`, `…/remediation/{id}`.
- Static `contextd://help` documenting the scheme.
- Handlers: parse URI → `withTenantContext` → service call
  (`reasoningbank.GetByProjectID`/`ListMemories`, `checkpoint.Get`/`List`,
  `remediation.GetByScope`/`Search`) → **scrub** text fields → JSON
  `ReadResourceResult` (`mimeType: application/json`). `ResourceNotFoundError`
  on miss. Tenant isolation via project_id in URI (fail-closed).
- Tests: collection read, by-id read, not-found, scrubbing applied.

### Phase B — Prompts (#2) · `internal/mcp/prompts.go`
- Static prompts, one per feature: `contextd_checkpoint(summary?)`,
  `contextd_remember(content?)`, `contextd_diagnose(error)`,
  `contextd_resume(checkpoint_id?)`, `contextd_status`, `contextd_search(query)`.
- Each returns a `GetPromptResult` whose user message contains the workflow
  instructions (parity with the plugin slash commands), templating `{{args}}`.
- Tests: `prompts/list` count, `prompts/get` messages, argument substitution.

### Phase D — Agent-swarm notifications (#6)
- Design doc: `docs/spec/mcp-protocol/notifications-agent-swarm.md`.
- Mechanism: stateful HTTP sessions; agents `resources/subscribe` to a
  collection URI (`contextd://{project}/memories|checkpoints|remediations`).
  On record (`memory_record`, `remediation_record`, `checkpoint_save`) the
  server calls `s.mcp.ResourceUpdated(ctx, {URI: collection})`, pushing
  `notifications/resources/updated` to subscribed sessions; other agents
  re-read and inherit shared knowledge.
- Helper `s.notifyCollectionUpdated(ctx, projectID, kind)` invoked from the
  record handlers. Tenant-scoped (only the matching project's collection URI).
- Doc covers: topology, subscription model, fan-out, self-notify handling,
  ordering, failure modes, security (tenant isolation), limits.
- Tests: subscribe → record → assert `updated` delivered over in-memory transports.

### Phase C — Logging + Elicitation (#4)
- Logging: `req.Session.Log(...)` progress on `repository_index`,
  `repository_search`, `memory_consolidate`; no-op-safe when session is nil.
- Elicitation: `checkpoint_resume` with no id + multiple checkpoints →
  `req.Session.Elicit` a choice; graceful fallback to returning the list when
  the client lacks elicitation capability.
- Tests: handlers unaffected when logging; elicitation fallback path.

### Phase E — HTTP consolidation (later)
- Move REST routes (`/scrub`, `/threshold`, `/status`) onto the Echo MCP
  server; delete `internal/http`. (Auth already added in Phase 0.)

## Cross-cutting

- `registerResources()` and `registerPrompts()` wired into `NewServer` (served
  over both stdio and HTTP via the shared `*mcp.Server`).
- SDK auto-declares the `resources`, `prompts`, and `logging` capabilities when
  `Add*` is called.
- Docs: update `docs/CONTEXTD.md` (add Resources/Prompts tables), `README.md`,
  `CHANGELOG.md`. Update in-repo plugin per Priority #3 (new prompts can ship as
  plugin prompts). Consider `VERSION` bump on completion.

## Sequencing

Phase 0 → A → B → D → C → E. (#6 depends on the Phase A collection URIs.)
