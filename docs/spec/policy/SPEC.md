# Policy Specification

**Feature**: Policy Management
**Status**: ⏸️ FUTURE - Not Implemented
**Created**: 2025-12-20
**Updated**: 2026-01-06

**⚠️ NOT IMPLEMENTED**: This feature is planned for a future release. Policy management tools and hierarchical scoping are not yet implemented in contextd.

## Overview

Policy provides organization-wide, team-level, and project-level rules that govern agent behavior. Unlike memories (learned strategies), policies are prescriptive rules defined by administrators that agents MUST follow. Policies are stored in vectorstore for semantic retrieval and injected into agent context during relevant operations.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Policy Hierarchy                          │
├─────────────────────────────────────────────────────────────┤
│  org_policies        → Base rules for all teams/projects    │
│       ↓ inherits                                             │
│  {team}_policies     → Team overrides/additions              │
│       ↓ inherits                                             │
│  {team}_{proj}_policies → Project-specific rules            │
└─────────────────────────────────────────────────────────────┘
```

Policies cascade down: project inherits team, team inherits org. Lower scopes can:
- **Override**: Replace a policy (same ID, different content)
- **Extend**: Add new policies
- **Disable**: Mark inherited policy as inactive at this scope

## User Scenarios

### P1: Security Policy Enforcement

**Story**: As a security lead, I want to define "no hardcoded secrets" as an org-wide policy, so that all agents across all projects enforce this rule.

**Acceptance Criteria**:
```gherkin
Given an org policy "SEC-001: No Hardcoded Secrets" with severity="error"
And the policy content describes detection patterns and remediation
When an agent generates code with a hardcoded API key
Then the policy is surfaced in pre-commit check context
And agent is instructed to use environment variables instead
```

**Edge Cases**:
- Policy disabled at project level for test fixtures
- Conflicting policies at different scopes
- Policy with regex patterns that don't compile

### P2: Coding Standard Policies

**Story**: As a tech lead, I want to define team coding standards as policies, so that agents follow consistent patterns.

**Acceptance Criteria**:
```gherkin
Given a team policy "Go Error Handling" with severity="warning"
And content specifies: "Always wrap errors with fmt.Errorf, never return bare errors"
When agent writes Go error handling code
Then the policy is injected into relevant context
And agent follows the specified pattern
```

### P3: Policy Search During Task

**Story**: As an agent starting a code review task, I want relevant policies automatically surfaced, so that I apply required rules.

**Acceptance Criteria**:
```gherkin
Given policies exist for security, testing, and documentation
When agent starts task "review authentication module"
Then security policies are retrieved (high relevance)
And testing policies are retrieved (medium relevance)
And documentation policies are filtered by confidence threshold
And total injection is <1000 tokens
```

### P4: Policy Override at Project Level

**Story**: As a project lead, I want to override an org policy for my experimental project, so that agents have appropriate flexibility.

**Acceptance Criteria**:
```gherkin
Given org policy "PERF-001: No synchronous I/O" with severity="error"
When project lead creates override with severity="warning" and reason="Prototype phase"
Then agents in this project see the warning-level version
And override includes expiration date
And audit log captures the override
```

### P5: Explicit Policy Creation

**Story**: As a team lead, I want to create a new policy via MCP tool, so that it's immediately available to agents.

**Acceptance Criteria**:
```gherkin
Given team lead calls policy_record with title, content, category, severity
Then policy is created with unique ID
And policy is embedded and stored in vectorstore
And policy is immediately searchable
And policy scope is set based on context (team or project)
```

### P6: Policy Metrics Dashboard

**Story**: As a compliance officer, I want to view policy hit rates and compliance metrics, so that I can assess organizational adherence.

**Acceptance Criteria**:
```gherkin
Given policies have been in use for 30 days
When compliance officer requests policy_metrics
Then system returns:
  - Hit rate per policy (searches that retrieved this policy)
  - Violation rate per policy (check failures)
  - Compliance rate per scope (team/project)
  - Trend data over time
And metrics are filterable by category, severity, scope
```

## Functional Requirements

### FR-001: Policy Storage
The system MUST store policies in vectorstore collections:
- `org_policies` for organization-level
- `{team}_policies` for team-level
- `{team}_{project}_policies` for project-level

### FR-002: Policy Schema
Policies MUST include:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | Yes | Unique identifier (e.g., "SEC-001") |
| tenant_id | string | Yes | Organization identifier |
| team_id | string | No | Team scope (empty = org-level) |
| project_id | string | No | Project scope (empty = team-level) |
| title | string | Yes | Human-readable title |
| content | string | Yes | Full policy text (embedded for search) |
| category | string | Yes | security, compliance, coding, testing, documentation |
| severity | string | Yes | error, warning, info |
| tags | []string | No | Additional categorization |
| enabled | bool | Yes | Whether policy is active |
| overrides | string | No | ID of policy this overrides |
| expires_at | timestamp | No | Optional expiration |
| created_at | timestamp | Yes | Creation time |
| updated_at | timestamp | Yes | Last update time |
| created_by | string | Yes | Creator identity |

### FR-003: Semantic Search
The system MUST retrieve policies by semantic similarity to query context.
- Search MUST cascade through scope hierarchy (project → team → org)
- Results MUST be deduplicated (lower scope overrides higher)
- Results MUST exclude disabled policies

### FR-004: Scope Inheritance
Policies MUST cascade from org → team → project:
- Org policies apply to all teams and projects unless overridden
- Team policies apply to all projects in team unless overridden
- Project policies are most specific

### FR-005: Override Mechanism
The system MUST support policy overrides:
- Override MUST reference parent policy ID in `overrides` field
- Override MUST include reason for override
- Override MAY have expiration date
- System MUST log all overrides for audit

### FR-006: Severity Levels
Policies MUST have severity that guides agent behavior:
| Severity | Agent Behavior |
|----------|----------------|
| error | MUST NOT proceed if policy violated |
| warning | SHOULD flag but MAY proceed with justification |
| info | Surface as guidance, no enforcement |

### FR-007: Category Classification
Policies MUST belong to categories for filtering:
- `security`: Authentication, authorization, secrets, vulnerabilities
- `compliance`: Regulatory requirements, data handling
- `coding`: Style, patterns, conventions
- `testing`: Test coverage, test patterns
- `documentation`: Documentation requirements
- `architecture`: System design rules

### FR-008: MCP Tools
The system MUST expose MCP tools:

| Tool | Purpose |
|------|---------|
| `policy_search` | Find policies by semantic query and scope |
| `policy_record` | Create new policy |
| `policy_update` | Modify existing policy |
| `policy_disable` | Disable policy at scope |
| `policy_list` | List policies by category/scope |
| `policy_check` | Validate action against applicable policies |
| `policy_metrics` | Retrieve policy usage and compliance metrics |

### FR-009: Context Injection
Policies MUST be injectable into agent context:
- Policies retrieved via `policy_search` format as `InjectedItem{Type: "policy"}`
- Injection respects token budget
- Higher severity policies prioritized

### FR-010: Audit Trail
The system MUST maintain audit trail for:
- Policy creation, updates, deletions
- Policy overrides and disabling
- Policy violations detected

### FR-011: Metrics Collection (MANDATORY)
The system MUST collect and expose policy metrics:

| Metric | Description | Storage |
|--------|-------------|---------|
| `policy_search_count` | Times policy appeared in search results | Counter per policy |
| `policy_hit_count` | Times policy was injected into context | Counter per policy |
| `policy_check_count` | Times policy was checked | Counter per policy |
| `policy_violation_count` | Times policy check failed | Counter per policy |
| `policy_compliance_rate` | (checks - violations) / checks | Computed |
| `scope_compliance_rate` | Aggregate compliance per team/project | Computed |

### FR-012: Metrics Aggregation
The system MUST aggregate metrics at multiple levels:
- Per-policy metrics (individual policy performance)
- Per-category metrics (security compliance, coding compliance, etc.)
- Per-scope metrics (team compliance, project compliance)
- Per-severity metrics (error-level compliance vs warning-level)
- Time-series data (daily, weekly, monthly trends)

### FR-013: Metrics Exposure
The system MUST expose metrics via:
- `policy_metrics` MCP tool for agent queries
- HTTP endpoint `/api/v1/policy/metrics` for dashboards
- OpenTelemetry metrics for observability stack integration
- Prometheus-compatible endpoint `/metrics`

## MCP Tool Specifications

### policy_search

**Input**:
```json
{
  "query": "string (required) - semantic search query",
  "tenant_id": "string (required) - organization",
  "team_id": "string (optional) - team scope",
  "project_id": "string (optional) - project scope",
  "category": "string (optional) - filter by category",
  "severity": "string (optional) - minimum severity",
  "limit": "int (optional, default: 10)"
}
```

**Output**:
```json
{
  "policies": [
    {
      "id": "SEC-001",
      "title": "No Hardcoded Secrets",
      "content": "...",
      "category": "security",
      "severity": "error",
      "scope": "org",
      "relevance": 0.92
    }
  ],
  "count": 1
}
```

### policy_record

**Input**:
```json
{
  "tenant_id": "string (required)",
  "team_id": "string (optional)",
  "project_id": "string (optional)",
  "id": "string (optional) - custom ID or auto-generated",
  "title": "string (required)",
  "content": "string (required)",
  "category": "string (required)",
  "severity": "string (required)",
  "tags": ["string"],
  "overrides": "string (optional) - policy ID to override",
  "expires_at": "timestamp (optional)"
}
```

**Output**:
```json
{
  "id": "SEC-002",
  "scope": "team",
  "created": true
}
```

### policy_check

**Input**:
```json
{
  "tenant_id": "string (required)",
  "team_id": "string (optional)",
  "project_id": "string (optional)",
  "action": "string (required) - description of action to check",
  "context": "string (optional) - additional context"
}
```

**Output**:
```json
{
  "allowed": false,
  "violations": [
    {
      "policy_id": "SEC-001",
      "title": "No Hardcoded Secrets",
      "severity": "error",
      "message": "Action contains potential hardcoded secret pattern"
    }
  ],
  "warnings": [],
  "guidance": "Use environment variables or secret manager"
}
```

### policy_metrics

**Input**:
```json
{
  "tenant_id": "string (required)",
  "team_id": "string (optional) - filter by team",
  "project_id": "string (optional) - filter by project",
  "policy_id": "string (optional) - specific policy metrics",
  "category": "string (optional) - filter by category",
  "severity": "string (optional) - filter by severity",
  "start_time": "timestamp (optional) - default: 30 days ago",
  "end_time": "timestamp (optional) - default: now",
  "granularity": "string (optional) - daily|weekly|monthly, default: daily"
}
```

**Output**:
```json
{
  "summary": {
    "total_policies": 45,
    "total_searches": 1250,
    "total_hits": 890,
    "total_checks": 2340,
    "total_violations": 127,
    "overall_compliance_rate": 0.946
  },
  "by_category": [
    {
      "category": "security",
      "policy_count": 12,
      "searches": 450,
      "hits": 380,
      "checks": 890,
      "violations": 45,
      "compliance_rate": 0.949
    }
  ],
  "by_severity": [
    {
      "severity": "error",
      "policy_count": 8,
      "checks": 560,
      "violations": 12,
      "compliance_rate": 0.979
    }
  ],
  "by_scope": [
    {
      "scope": "team:platform",
      "checks": 340,
      "violations": 23,
      "compliance_rate": 0.932
    }
  ],
  "top_violated": [
    {
      "policy_id": "CODE-003",
      "title": "Error wrapping required",
      "violations": 34,
      "compliance_rate": 0.891
    }
  ],
  "trends": [
    {
      "date": "2025-12-19",
      "checks": 89,
      "violations": 4,
      "compliance_rate": 0.955
    }
  ]
}
```

## Success Criteria

### SC-001: Retrieval Relevance
>85% of retrieved policies should be rated "applicable" to the current task context.

### SC-002: Override Accuracy
100% of policy overrides MUST be properly resolved (lower scope wins).

### SC-003: Enforcement Rate
>95% of error-severity policy violations should be caught before commit.

### SC-004: Search Performance
Policy search MUST complete in <100ms for <1000 policies per scope.

### SC-005: Cascade Performance
Full hierarchy search (project → team → org) MUST complete in <200ms.

### SC-006: Audit Completeness
100% of policy lifecycle events MUST be captured in audit trail.

### SC-007: Metrics Availability
Policy metrics MUST be queryable within 1 minute of event occurrence.

### SC-008: Metrics Accuracy
Computed compliance rates MUST match raw counts with <0.1% variance.

### SC-009: Dashboard Latency
`policy_metrics` queries MUST complete in <500ms for 90-day windows.

## Metrics Schema

### Event Storage (Recent)
```json
{
  "event_id": "uuid",
  "event_type": "search|hit|check|violation",
  "policy_id": "SEC-001",
  "tenant_id": "org-123",
  "team_id": "platform",
  "project_id": "contextd",
  "session_id": "session-456",
  "timestamp": "2025-12-20T10:30:00Z",
  "metadata": {}
}
```

### Aggregate Storage (Historical)
```json
{
  "policy_id": "SEC-001",
  "tenant_id": "org-123",
  "date": "2025-12-19",
  "search_count": 45,
  "hit_count": 38,
  "check_count": 120,
  "violation_count": 5
}
```

### Rollup Schedule
- Events older than 30 days → aggregate to daily counts
- Daily aggregates older than 1 year → aggregate to monthly counts

## Security Considerations

### SEC-001: Authorization
- Only authorized users (leads, admins) can create/modify policies
- Read access follows scope hierarchy
- Override requires elevated permissions
- Metrics access requires at least read access to scope

### SEC-002: Tenant Isolation
- Policies MUST be isolated by tenant
- Cross-tenant policy access MUST be blocked
- Payload filtering enforces isolation
- Metrics MUST be isolated by tenant

### SEC-003: Content Validation
- Policy content MUST be scrubbed for secrets before storage
- Regex patterns in policies MUST be validated for safety
- Size limits prevent abuse (max 10KB per policy)

## Implementation Notes

### Package Structure
```
internal/policy/
├── service.go       # PolicyService implementation
├── types.go         # Policy, PolicyRequest, etc.
├── store.go         # Vectorstore operations
├── cascade.go       # Scope hierarchy resolution
├── check.go         # Policy violation checking
├── metrics.go       # Metrics collection and aggregation
├── metrics_store.go # Metrics storage backend
└── service_test.go  # Tests
```

### Dependencies
- `internal/vectorstore` - Storage backend
- `internal/embeddings` - Text embedding
- `internal/secrets` - Content scrubbing
- `internal/tenant` - Tenant context
- `internal/telemetry` - OpenTelemetry integration

### Collection Schema
```json
{
  "name": "org_policies",
  "vectors": {
    "size": 384,
    "distance": "Cosine"
  },
  "payload_indexes": [
    {"field": "tenant_id", "type": "keyword"},
    {"field": "category", "type": "keyword"},
    {"field": "severity", "type": "keyword"},
    {"field": "enabled", "type": "bool"},
    {"field": "created_at", "type": "datetime"}
  ]
}
```

### Metrics Collection
```json
{
  "name": "policy_events",
  "payload_indexes": [
    {"field": "tenant_id", "type": "keyword"},
    {"field": "policy_id", "type": "keyword"},
    {"field": "event_type", "type": "keyword"},
    {"field": "timestamp", "type": "datetime"}
  ]
}
```

## OpenTelemetry Metrics

The following metrics MUST be exported:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `contextd_policy_searches_total` | Counter | tenant_id, category | Total policy searches |
| `contextd_policy_hits_total` | Counter | tenant_id, policy_id, category | Total policy injections |
| `contextd_policy_checks_total` | Counter | tenant_id, policy_id, severity | Total policy checks |
| `contextd_policy_violations_total` | Counter | tenant_id, policy_id, severity | Total violations |
| `contextd_policy_compliance_ratio` | Gauge | tenant_id, scope, category | Current compliance rate |
| `contextd_policy_search_latency_seconds` | Histogram | tenant_id | Search latency |
| `contextd_policy_check_latency_seconds` | Histogram | tenant_id | Check latency |

## Real-World Use Case: Preventive Security Patterns

This use case demonstrates why policies are needed - memories alone are insufficient for enforcing engineering standards.

### The Problem

During development of contextd's metadata CLI commands (2026-01-26), the following pattern emerged:

```
1. Developer writes code accepting user input (CLI args, file paths)
2. Code does NOT include input validation (path traversal, hash format)
3. Consensus review catches issues (CWE-22 path traversal, missing regex validation)
4. Developer fixes issues
5. Memory recorded: "We fixed path traversal"
6. NEXT TIME: Same issues appear because memory is descriptive, not prescriptive
```

**Evidence**: Path traversal prevention has appeared in 3+ separate consensus reviews despite memories existing about prior fixes.

### Why Memories Failed

Memories in `engineering-practices` project contained:
```
Title: "Consensus review caught critical security issues before merge"
Content: "Path traversal risk in quarantine operations - Missing hash validation"
         "hash validation added (^[a-f0-9]{8}$)"
```

**Problem**: This memory records WHAT was fixed, not WHAT TO DO before writing code.

Even with a preventive memory added:
```
Title: "PREVENTIVE: Input validation checklist for CLI/API code"
Content: [checklist with code patterns]
Tags: ["security", "input-validation", "preventive"]
```

**The gap**: Memories are opt-in retrieval. Developer must remember to search before coding.

### How Policies Solve This

With the Policy system, the security lead defines:

```json
{
  "id": "SEC-007",
  "title": "Input Validation Required",
  "category": "security",
  "severity": "error",
  "scope": "org",
  "content": "All code accepting user input MUST validate:\n\n## Path Inputs\n- Check for '..' in raw input\n- Use filepath.Clean()\n- Validate within expected boundaries\n- For hashes: regex ^[a-fA-F0-9]{8}$\n\n## Code Pattern\n```go\nif strings.Contains(userPath, \"..\") {\n    return fmt.Errorf(\"path traversal not allowed\")\n}\n```"
}
```

**Automatic Enforcement**:
1. Agent starts task: "Add metadata CLI commands"
2. `policy_search("CLI user input file paths")` → SEC-007 injected
3. Agent sees policy BEFORE writing code
4. Code includes validation from the start
5. Consensus review catches edge cases, not known patterns

### Policy vs Memory Comparison

| Aspect | Memory | Policy |
|--------|--------|--------|
| Retrieval | Opt-in (developer must search) | Automatic (injected on relevant tasks) |
| Enforcement | None (informational) | Severity-based (error blocks, warning flags) |
| Authority | Learned suggestion | Prescriptive rule from lead |
| Scope | Project-specific | Hierarchical (org → team → project) |
| Audit | Usage count only | Compliance rate, violation tracking |

### Implementation Priority

This use case demonstrates Policy system should be **P1 priority** because:
1. Security issues recurring despite memory records
2. Consensus review time wasted on preventable issues
3. "Fix later" mentality enabled without enforcement
4. No compliance metrics without policy_check integration

### Proposed Policy Set from Use Case

| Policy ID | Title | Severity | Category |
|-----------|-------|----------|----------|
| SEC-007 | Input Validation Required | error | security |
| SEC-008 | Path Traversal Prevention | error | security |
| SEC-009 | Hash/ID Format Validation | warning | security |
| CODE-010 | No Silent Error Handling | warning | coding |
| CODE-011 | Restrictive File Permissions | warning | coding |

These policies would be created at org level in `org_policies` collection and automatically surfaced when agents work on CLI/API code.

---

## Future Considerations

### Phase 2: Policy Templates
- Pre-built policy templates (OWASP, SOC2, etc.)
- One-click adoption of template sets
- Template versioning and updates

### Phase 3: Automated Enforcement
- Pre-commit hook integration
- CI/CD pipeline checks
- Real-time violation alerts

### Phase 4: Policy Analytics Dashboard
- Grafana dashboard templates
- Compliance trend visualization
- Violation hotspot identification
- Team comparison reports
