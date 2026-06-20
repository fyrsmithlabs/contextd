# MCP Tool Handler Style Guide

Audience: contributors writing or modifying MCP tools in `internal/mcp/`.
Status: normative. Deviations require an explicit comment justifying why.

SDK: `github.com/modelcontextprotocol/go-sdk v1.1.0` (see `go.mod`).
Spec baseline: MCP `2025-06-18` (annotations + structured content GA).

This guide is opinionated. It tells you what to do, not what to consider.

---

## 1. Tool registration

**1.1 — Name format.** Every tool is `<noun>_<verb>` in `snake_case`. The
noun is the domain object (`checkpoint`, `memory`, `remediation`, `branch`,
`conversation`, `repository`, `reflect`, `troubleshoot`). The verb is one of
`search`, `list`, `save`, `record`, `resume`, `feedback`, `outcome`,
`consolidate`, `index`, `diagnose`, `create`, `return`, `status`, `report`,
`analyze`. Do not invent new verbs without updating this list.

**1.2 — Description budget.** `Description` is ≤200 characters, starts
with an imperative verb, names the user-visible effect, and does not
restate the tool name. Long-form behavioral docs belong in `docs/spec/`.

**1.3 — One registration site per category.** Each domain has a
`register<Domain>Tools()` method called from `registerTools()`. New tools
go into the existing category or get their own `register…Tools` function.

```go
mcp.AddTool(s.mcp, &mcp.Tool{
    Name:        "checkpoint_save",
    Description: "Save a session checkpoint for later resumption",
}, func(ctx context.Context, req *mcp.CallToolRequest, args checkpointSaveInput) (*mcp.CallToolResult, checkpointSaveOutput, error) {
    ...
})
```

Counter-example: `Description: "Tool that does memory consolidation and …"`
restates the name and exceeds the budget. Prefer
`"Consolidate similar memories to reduce redundancy"`.

---

## 2. Tool annotations

Every `mcp.Tool` MUST set `Annotations: &mcp.ToolAnnotations{…}`. The four
hints below come from the MCP 2025-06-18 spec and are honored by
`go-sdk@v1.1.0`.

| Hint              | Type    | Default | Meaning                                            |
|-------------------|---------|---------|----------------------------------------------------|
| `ReadOnlyHint`    | `bool`  | `false` | Tool does not mutate environment.                  |
| `DestructiveHint` | `*bool` | `true`  | Tool can destroy/modify existing data.             |
| `IdempotentHint`  | `bool`  | `false` | Repeated identical calls have no extra effect.     |
| `OpenWorldHint`   | `*bool` | `true`  | Tool reaches an unbounded external world.          |

**2.1 — Classification matrix for contextd tools:**

- **Pure reads** (`*_list`, `*_search`, `*_status`, `checkpoint_resume`,
  `reflect_analyze`, `reflect_report`, `conversation_search`,
  `branch_status`, `memory_search`):
  `ReadOnlyHint=true`, `OpenWorldHint=ptrFalse()`.
- **Append-only writes** (`checkpoint_save`, `remediation_record`,
  `memory_record`, `conversation_index`, `repository_index`,
  `branch_create`): `ReadOnlyHint=false`, `DestructiveHint=ptrFalse()`,
  `IdempotentHint=false`, `OpenWorldHint=ptrFalse()`.
- **Mutating writes** (`memory_feedback`, `memory_outcome`,
  `remediation_feedback`, `memory_consolidate`,
  `memory_consolidate_session`, `branch_return`): same as above but
  `DestructiveHint=ptrTrue()` because confidence scores and consolidations
  overwrite prior state.
- **External-world reads** (`troubleshoot_diagnose`): pure read but may
  call an LLM — `ReadOnlyHint=true`, `OpenWorldHint=ptrTrue()`.

**2.2 — Pointer hints are explicit.** Allocate `ptrTo(true)` / `ptrTo(false)`
for `DestructiveHint` and `OpenWorldHint` so the JSON includes the field.
Omission is treated as "unknown default" by clients.

```go
mcp.AddTool(s.mcp, &mcp.Tool{
    Name:        "memory_search",
    Description: "Search for relevant memories from past sessions",
    Annotations: &mcp.ToolAnnotations{
        ReadOnlyHint:  true,
        OpenWorldHint: ptrFalse(),
    },
}, ...)
```

---

## 3. Input schema

**3.1 — Always declare input as a named struct.** No anonymous inline
struct args. Named structs make the schema reviewable.

**3.2 — Every field has a `jsonschema` tag.** The tag carries the
description shown to LLMs. Mark required fields with `required,`.

```go
type checkpointSaveInput struct {
    SessionID   string `json:"session_id"   jsonschema:"required,Session identifier"`
    ProjectPath string `json:"project_path" jsonschema:"required,Project path"`
    TenantID    string `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived from project_path via git remote if not provided)"`
}
```

**3.3 — Enums are validated server-side, not just typed.**
`semantic_search.content_mode` (`minimal`, `preview`, `full`) is the
canonical example. The handler rejects unknown values with a descriptive
error. Do not rely solely on the client honoring the schema.

**3.4 — `omitempty` for optional fields.** Use `,omitempty` so the client
schema reflects that the field is optional. Pair with sensible defaults
inside the handler (e.g. `limit := 10` if `<= 0`).

**3.5 — Validate every identifier.** All tenant/team/project IDs and
paths run through `sanitize.Validate*` and `sanitize.ValidateProjectPath`.
This is non-negotiable (CWE-22, CWE-287 hardening).

---

## 4. Output / structured content

`go-sdk@v1.1.0` populates `CallToolResult.StructuredContent` automatically
when the handler signature returns `(*mcp.CallToolResult, OUT, error)`.

**4.1 — Always return a typed output struct.** The third return value
must be a named struct with `jsonschema` tags on every field. This is
what populates `structuredContent` over the wire.

**4.2 — Always return one human-readable line of `TextContent`.** Use a
single line summarizing the outcome. Keep it under ~200 chars.

```go
return &mcp.CallToolResult{
    Content: []mcp.Content{
        &mcp.TextContent{Text: fmt.Sprintf("Found %d checkpoints", output.Count)},
    },
}, output, nil
```

**4.3 — Never embed JSON in `TextContent`.** Use the structured output
struct. The text channel is for humans glancing at logs.

**4.4 — Do not return `map[string]interface{}` for the structured
output.** Use typed row structs. The SDK derives the output schema from
them.

---

## 5. Tenant plumbing — the `tenantCtx` helper

The canonical pattern was introduced in commit `3192901`
(`internal/mcp/tools.go`). It replaces the older
`validateAndDeriveProjectPath` + `withTenantContext` pair.

**5.1 — Every handler that touches the vectorstore calls
`s.tenantCtx(ctx, projectPath, tenantID, teamID, projectID)` exactly
once, near the top of the handler, immediately after `startMetrics`.**
Pass `""` for IDs you want auto-derived.

Before (deprecated):

```go
validPath, tenantID, projectID, err := s.validateAndDeriveProjectPath(args.ProjectPath, args.TenantID)
if err != nil { toolErr = err; return ..., err }

ctx, err = withTenantContext(ctx, tenantID, args.TeamID, projectID)
if err != nil { toolErr = err; return ..., err }
```

After:

```go
ctx, rt, err := s.tenantCtx(ctx, args.ProjectPath, args.TenantID, "", "")
if err != nil {
    toolErr = err
    return nil, checkpointSaveOutput{}, err
}
// rt.ValidPath, rt.TenantID, rt.TeamID, rt.ProjectID are sanitized.
```

**5.2 — Use `resolvedTenant` (`rt`) values when populating service
request structs.** Do not re-read `args.TenantID` — the helper may have
derived a different value.

**5.3 — For team/org-scoped operations, pass `projectID=""`.** The helper
preserves an empty project floor for cross-project queries.

**5.4 — The legacy helpers are deprecated.** `withTenantContext` and
`validateAndDeriveProjectPath` exist only to support not-yet-migrated
handlers. Do not add new calls to them.

---

## 6. Error handling

**6.1 — Two error channels: `error` return and `toolErr` metric.**

```go
var toolErr error
defer s.startMetrics(ctx, "checkpoint_save", &toolErr)()
```

When you return a non-nil Go error, also assign it to `toolErr` so the
deferred metrics recorder captures the failure.

**6.2 — Wrap with `fmt.Errorf("<op> failed: %w", err)`.** Contextual
message for the client; preserves the chain for logs.

**6.3 — Validation errors return a Go error, not `IsError: true`.** The
SDK turns the Go error into a JSON-RPC error which clients render as a
tool failure. Reserve `mcp.CallToolResult.IsError` for cases where the
tool itself succeeded but the *result* is a "negative."

**6.4 — Never leak internal paths, stack traces, or service plumbing.**
Use sanitized names and human-friendly phrases.

---

## 7. Secret scrubbing

`s.scrubber` is the configured `secrets.Scrubber`. Every string field
that came from user content, git history, or a service response MUST
pass through it before being returned in `TextContent` or in the
structured output struct.

**7.1 — Scrub every free-form text field.** That includes `summary`,
`description`, `content`, `solution`, `code_diff`, `root_cause`, and any
field that may contain a quoted error message, file contents, or LLM
output.

```go
scrubbedSummary := s.scrubber.Scrub(cp.Summary).Scrubbed
scrubbedDesc    := s.scrubber.Scrub(cp.Description).Scrubbed
```

**7.2 — Do not scrub IDs, counts, scores, or enums.** Wasted work,
ugly logs.

**7.3 — `s.scrubber` is required for production but optional in tests.**
Use `if s.scrubber != nil` guards; never `panic` if missing.

---

## 8. Pagination

contextd uses limit-based pagination today (every `*_list` / `*_search`
takes a `Limit` field). No tool currently uses cursors.

**8.1 — Default limits.** `Limit int `json:"limit,omitempty"`` with an
in-handler default. `5` for memories, `10` for searches, `20` for lists.
Cap at 100 server-side regardless of caller value.

**8.2 — Cursors only when result sets routinely exceed 100 items.** If
needed, add `Cursor string` to input and `NextCursor string` to output,
opaque base64 over the underlying store's offset/keyset. Do not expose
raw scroll IDs.

**8.3 — Cap result counts.** The MCP layer must enforce its own cap
(`limit > 100 => limit = 100`).

---

## 9. Tests

**9.1 — Each handler has at least one direct test.** See
`tools_checkpoint_test.go`, `tools_folding_test.go`,
`tools_repository_test.go`, `tools_tenant_context_test.go`.

**9.2 — Test the tenant context plumbing explicitly.** Verify the
handler sets `vectorstore.TenantInfo` on `ctx` with the expected
`TenantID`/`TeamID`/`ProjectID` derived from inputs.

**9.3 — Test rejection of malformed identifiers.** At least one test per
handler must call with a malformed `tenant_id`, `project_path`, or
`project_id` and assert an error.

**9.4 — Cover enum / mode branches.** One test per enum value plus an
"invalid mode" rejection test.

**Coverage target: 80% line coverage on `internal/mcp/`.** New handlers
must not lower the package's coverage.

---

## 10. Anti-patterns — do not do this

- **Do not** use anonymous struct args.
- **Do not** add new callers of `withTenantContext` or
  `validateAndDeriveProjectPath`. Use `tenantCtx`.
- **Do not** call `filepath.Base()` on untrusted input. Use
  `sanitize.SafeBasename` (already wrapped by `deriveProjectID`).
- **Do not** return `map[string]interface{}` in new tools. Use typed
  row structs.
- **Do not** skip the `defer s.startMetrics(...)` line.
- **Do not** write tool descriptions longer than 200 characters or that
  begin with "This tool ...".
- **Do not** return raw service errors. Wrap with `%w`.
- **Do not** scrub IDs, scores, or counts.
- **Do not** register a tool without annotations.
- **Do not** introduce a new verb suffix without updating §1.1.
- **Do not** embed JSON in `TextContent`.
- **Do not** rely on the client to enforce `required` — re-validate
  inside the handler.

---

## Quick template for a new tool

```go
type fooSearchInput struct {
    ProjectPath string `json:"project_path" jsonschema:"required,Project path"`
    TenantID    string `json:"tenant_id,omitempty" jsonschema:"Tenant identifier (auto-derived)"`
    Query       string `json:"query"        jsonschema:"required,Search query"`
    Limit       int    `json:"limit,omitempty" jsonschema:"Max results (default 10, max 100)"`
}

type fooSearchRow struct {
    ID    string  `json:"id"    jsonschema:"Item identifier"`
    Score float64 `json:"score" jsonschema:"Relevance score (0-1)"`
}

type fooSearchOutput struct {
    Results []fooSearchRow `json:"results" jsonschema:"Matching items"`
    Count   int            `json:"count"   jsonschema:"Number of items returned"`
}

mcp.AddTool(s.mcp, &mcp.Tool{
    Name:        "foo_search",
    Description: "Search foos by semantic match against the query.",
    Annotations: &mcp.ToolAnnotations{
        ReadOnlyHint:  true,
        OpenWorldHint: ptrFalse(),
    },
}, func(ctx context.Context, req *mcp.CallToolRequest, args fooSearchInput) (*mcp.CallToolResult, fooSearchOutput, error) {
    var toolErr error
    defer s.startMetrics(ctx, "foo_search", &toolErr)()

    ctx, rt, err := s.tenantCtx(ctx, args.ProjectPath, args.TenantID, "", "")
    if err != nil {
        toolErr = err
        return nil, fooSearchOutput{}, err
    }

    limit := args.Limit
    if limit <= 0 { limit = 10 }
    if limit > 100 { limit = 100 }

    raw, err := s.fooSvc.Search(ctx, rt.TenantID, args.Query, limit)
    if err != nil {
        toolErr = fmt.Errorf("foo search failed: %w", err)
        return nil, fooSearchOutput{}, toolErr
    }

    rows := make([]fooSearchRow, 0, len(raw))
    for _, r := range raw {
        rows = append(rows, fooSearchRow{ID: r.ID, Score: r.Score})
    }

    out := fooSearchOutput{Results: rows, Count: len(rows)}
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: fmt.Sprintf("Found %d foos", out.Count)},
        },
    }, out, nil
})
```
