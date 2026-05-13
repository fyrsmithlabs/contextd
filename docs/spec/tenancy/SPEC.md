# Tenancy Model

**Status**: Active
**Owner**: contextd core
**Last updated**: 2026-05-13

## Goal

contextd is multi-tenant by design, but most users are solo developers. The
tenancy model needs to be:

1. **Invisible** for a solo dev running `contextd` against a single repo.
2. **Explicit** for teams who want to share memories across projects.
3. **Configurable** for deployments that need physical (filesystem) isolation.

This spec defines the data model, defaulting rules, and isolation modes.

## ID hierarchy

contextd uses three identifiers, scoped from coarsest to finest:

| ID          | Required? | Default                                         | Purpose                                    |
|-------------|-----------|--------------------------------------------------|--------------------------------------------|
| `TenantID`  | Optional in MCP handlers; required on `TenantInfo` once resolved | Git remote owner → `git user.name` → `$USER` → `"local"` | Organization or individual identity        |
| `TeamID`    | Optional  | None                                            | Team scope within a tenant (no auto-default) |
| `ProjectID` | **Required floor** when resolving from defaults; optional when an explicit `TenantInfo` is supplied for team/org scope | Sanitized basename of the project path / CWD | Per-repository isolation — the *floor* of contextd's isolation model |

### Why ProjectID is the floor

If a single developer works on `~/repos/contextd` and `~/repos/other`,
memories from one repo must not surface in the other. ProjectID enforces that.
Without it, a single solo-dev tenant (e.g. `dahendel`) would conflate every
project they touch.

The floor is enforced in two places:

- **Implicit defaulting**: `vectorstore.TenantFromContext` returns
  `ErrMissingProject` if the registered default resolver produces a TenantInfo
  with empty ProjectID.
- **MCP handlers**: The `tenantCtx` helper derives ProjectID from
  `project_path` and surfaces it on the `TenantInfo`. Handlers that
  intentionally operate at team or org scope (e.g. `remediation_search` with
  team scope) pass `projectID=""` explicitly — an opt-in deviation.

### Why TenantID is *not* required

Solo devs shouldn't have to invent a tenant name. The default resolver derives
one deterministically:

1. GitHub username from the project's `origin` remote (best — survives moves).
2. Git global `user.name` (good — stable across repos for one user).
3. `$USER` environment variable.
4. Literal `"local"` (last resort).

Multi-tenant SaaS callers that already plumb `TenantInfo` continue to work
unchanged; explicit values always win.

## Isolation modes

`IsolationMode` is selected via `CONTEXTD_ISOLATION_MODE` /
`vectorstore.isolation_mode` config:

| Mode         | Strategy                                              | When to use                                                                 |
|--------------|-------------------------------------------------------|------------------------------------------------------------------------------|
| `payload`    | Single collection; tenant fields stored on each document and injected into every filter | **Default.** Solo devs, most teams. Lowest operational cost.                |
| `filesystem` | Separate database directory per tenant/project        | Regulated deployments needing physical isolation. Each tenant gets its own data root. |
| `none`       | No isolation enforcement                              | **Tests only.** Provides zero security guarantees.                          |

The runtime default remains `payload` — this PR does not change that. It only
makes the choice explicit and configurable.

Mode is wired into the store at factory time
(`vectorstore.NewStore`). Tests may override via the `WithIsolation` option.

## Default resolution rules

```
TenantFromContext(ctx)
├── ctx has *TenantInfo with TenantID != ""
│       └── return as-is (caller wins)
├── no TenantInfo, default resolver registered
│       ├── resolver() returns nil       → ErrMissingTenant
│       ├── derived.TenantID  == ""      → ErrMissingTenant
│       ├── derived.ProjectID == ""      → ErrMissingProject
│       └── otherwise                    → return derived
└── no TenantInfo, no resolver registered → ErrMissingTenant
```

The resolver is registered once in `cmd/contextd/main.go` and only there.
Library tests that instantiate the vector store directly retain the historical
fail-closed contract.

## Use-case matrix

| Scenario                           | Recommended mode | Tenant            | Team   | Project        |
|------------------------------------|------------------|--------------------|--------|----------------|
| Solo dev, single repo              | `payload`        | auto (`$USER`)     | —      | auto (basename) |
| Solo dev, many repos               | `payload`        | auto               | —      | auto per repo   |
| Small team, shared remediations    | `payload`        | org name           | team   | per repo        |
| Org with regulated data            | `filesystem`     | tenant per regulator boundary | optional | optional |
| Integration tests in CI            | `payload` or `none` | fixture           | fixture | fixture       |
| Unit tests inside vectorstore pkg  | `payload` (default) — no resolver | explicit | explicit | explicit |

## Surface area introduced by this change

| Component                              | Where                                       |
|----------------------------------------|---------------------------------------------|
| `ErrMissingProject`                    | `internal/vectorstore/tenant.go`            |
| `SetDefaultTenantResolver`             | `internal/vectorstore/tenant.go`            |
| `tenant.GetDefaultProjectID`, `tenant.DefaultsForPath` | `internal/tenant/defaults.go`     |
| `VectorStoreConfig.IsolationMode` (`CONTEXTD_ISOLATION_MODE`) | `internal/config/config.go` |
| `Server.tenantCtx` MCP helper          | `internal/mcp/tools.go`                     |

## Migration notes

- Existing `TenantInfo{TenantID: "org-123"}` callers continue to work — they
  carry their own context and never hit the resolver.
- Handlers using the legacy `validateAndDeriveProjectPath` /
  `withTenantContext` pair still compile; only three were migrated in the
  introducing PR. A follow-up will move the remaining ~20 call sites onto
  `tenantCtx`.
- No on-disk format changes. PayloadIsolation already encodes tenant fields
  on each document; nothing needs re-indexing.
