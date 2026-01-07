# Collection Architecture Specification

**Feature**: Collection Architecture
**Status**: DEPRECATED
**Created**: 2025-11-22
**Updated**: 2026-01-06
**deprecated**: true
**deprecated_reason**: "contextd v2 uses payload-based tenant isolation with shared collections, not database-per-org or physical collection separation. See docs/spec/vector-storage/security.md for current multi-tenancy architecture."

**⚠️ DEPRECATED**: This specification describes a database-per-org architecture that was never implemented. contextd v2 uses **payload-based tenant isolation** where all tenants share collections with automatic metadata filtering. See `docs/spec/vector-storage/security.md` for the current architecture.

## Overview

Defines the Qdrant database and collection structure for contextd. Organizations get dedicated databases with collections organized by team and project scope.

## Database Structure

```
Qdrant Cluster
│
├── Database: {org_id}                    ← Physical isolation per org
│   │
│   ├── ═══════════════════════════════════════════════════
│   │   ORGANIZATION LEVEL (org_*)
│   ├── ═══════════════════════════════════════════════════
│   ├── org_memories                      # Org-wide strategies
│   ├── org_remediations                  # Org-wide fixes
│   ├── org_policies                      # Security/compliance
│   ├── org_coding_standards              # Org conventions
│   ├── org_repo_standards                # Repo structure
│   ├── org_skills                        # Skill definitions
│   ├── org_agents                        # Agent configs
│   ├── org_anti_patterns                 # Failure patterns
│   ├── org_feedback                      # User ratings
│   │
│   ├── ═══════════════════════════════════════════════════
│   │   TEAM LEVEL ({team}_*)
│   ├── ═══════════════════════════════════════════════════
│   ├── {team}_memories                   # Team strategies
│   ├── {team}_remediations               # Team fixes
│   ├── {team}_coding_standards           # Team overrides
│   │
│   ├── ═══════════════════════════════════════════════════
│   │   PROJECT LEVEL ({team}_{project}_*)
│   ├── ═══════════════════════════════════════════════════
│   ├── {team}_{project}_memories         # Project strategies
│   ├── {team}_{project}_remediations     # Project fixes
│   ├── {team}_{project}_codebase         # Code embeddings
│   ├── {team}_{project}_sessions         # Session traces
│   └── {team}_{project}_checkpoints      # Context snapshots
│
└── Database: {another_org_id}            ← Complete isolation
    └── ...
```

## Collection Naming Convention

| Scope | Pattern | Example |
|-------|---------|---------|
| Organization | `org_{type}` | `org_memories` |
| Team | `{team}_{type}` | `platform_memories` |
| Project | `{team}_{project}_{type}` | `platform_contextd_memories` |

## Collection Types by Scope

### Organization Level (`org_*`)

| Collection | Purpose |
|------------|---------|
| `org_memories` | Org-wide proven strategies |
| `org_remediations` | Org-wide bug fixes |
| `org_policies` | Security/compliance policies |
| `org_coding_standards` | Org coding conventions |
| `org_repo_standards` | Repository structure rules |
| `org_skills` | Reusable skill definitions |
| `org_agents` | Agent configurations |
| `org_anti_patterns` | Failure-derived warnings |
| `org_feedback` | User ratings across org |

### Team Level (`{team}_*`)

| Collection | Purpose |
|------------|---------|
| `{team}_memories` | Team-specific strategies |
| `{team}_remediations` | Team-specific fixes |
| `{team}_coding_standards` | Team convention overrides |

### Project Level (`{team}_{project}_*`)

| Collection | Purpose |
|------------|---------|
| `{team}_{project}_memories` | Project strategies |
| `{team}_{project}_remediations` | Project fixes |
| `{team}_{project}_codebase` | Code embeddings |
| `{team}_{project}_sessions` | Session traces |
| `{team}_{project}_checkpoints` | Context snapshots |

## Functional Requirements

### FR-001: Database Per Organization
Each organization MUST have a dedicated Qdrant database for physical isolation.

### FR-002: Collection Naming
Collections MUST follow the naming convention: `{scope_prefix}_{type}`.

### FR-003: Vector Configuration
All collections MUST use consistent embedding dimensions within an org (configurable: 1536 or 768).

### FR-004: Payload Indexing
Collections MUST index common filter fields: `confidence`, `outcome`, `tags`, `created_at`.

### FR-005: Database Lifecycle
System MUST support database creation, backup, restore, and deletion per organization.

### FR-006: Collection Lifecycle
System MUST create required collections when teams/projects are provisioned.

### FR-007: Cross-Collection Queries
System MUST support efficient queries across scope hierarchy within a database.

### FR-008: Schema Validation
Payloads MUST conform to defined schemas before insertion.

## User Scenarios

### P1: New Organization Onboarding

**Acceptance Criteria**:
```gherkin
Given a new organization "acme_corp" signs up
When the org is provisioned
Then a database "acme_corp" is created
And org-level collections are initialized (org_memories, org_policies, etc.)
And the database is empty but ready for use
```

### P2: New Team Creation

**Acceptance Criteria**:
```gherkin
Given organization "acme_corp" exists
When team "platform" is created
Then collections are created:
  - platform_memories
  - platform_remediations
  - platform_coding_standards
```

### P3: New Project Creation

**Acceptance Criteria**:
```gherkin
Given team "platform" exists in "acme_corp"
When project "contextd" is created
Then collections are created:
  - platform_contextd_memories
  - platform_contextd_remediations
  - platform_contextd_codebase
  - platform_contextd_sessions
  - platform_contextd_checkpoints
```

## Success Criteria

### SC-001: Query Performance
Single-collection queries MUST complete in <50ms for 100K points.

### SC-002: Cross-Scope Performance
Hierarchical queries (project→team→org) MUST complete in <150ms.

### SC-003: Isolation Guarantee
Queries in one org database MUST never access another org's data.

### SC-004: Provisioning Speed
New org database + collections MUST provision in <30 seconds.
