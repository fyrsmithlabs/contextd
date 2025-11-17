# Team-Aware Architecture (v2.2)

**Status**: Proposed for v2.2 (after v2.1 owner-scoping)
**Date**: 2025-01-07
**Use Case**: Single org deployment with team isolation

---

## Problem Statement

**Realistic Enterprise Deployment**:
- One organization (e.g., Acme Corp) deploys contextd
- Multiple development teams use shared instance
- Teams need isolation BUT some knowledge should be org-wide

**Scenarios**:

### Scenario 1: Single Org, Single Team (Simplest)
```
Acme Corp
└── Backend Team (10 developers)
    ├── Dev 1 → personal repos
    ├── Dev 2 → personal repos
    └── ...
```

**Need**:
- Developers share knowledge within team
- No cross-team concerns (only one team)

### Scenario 2: Single Org, Multiple Teams (Most Common)
```
Acme Corp
├── Backend Team (10 developers)
│   ├── microservices-api
│   └── payment-service
│
├── Frontend Team (8 developers)
│   ├── web-app
│   └── mobile-app
│
└── Platform Team (5 developers)
    ├── kubernetes-infra
    └── monitoring
```

**Need**:
- Teams share knowledge within their team
- Some knowledge shared org-wide (platform patterns, security fixes)
- Teams do NOT see each other's proprietary solutions

### Scenario 3: Single Org, Cross-Team Collaboration
```
Acme Corp (backend and frontend teams collaborate on project X)
├── Backend Team → works on API
└── Frontend Team → works on UI
    └── Both need to share project X knowledge
```

**Need**:
- Project-level sharing across teams
- Team-level sharing within teams
- Org-level sharing for general patterns

---

## Proposed Architecture: Three-Level Hierarchy

### Database Structure

```
Organization Level (org-wide knowledge):
org_acme/
├── remediations          # Org-wide error solutions
├── skills                # Org-wide best practices
└── troubleshooting       # Org-wide patterns

Team Level (team-specific knowledge):
team_backend/
├── remediations          # Backend team solutions
├── skills                # Backend team practices
└── troubleshooting       # Backend team patterns

team_frontend/
├── remediations          # Frontend team solutions
├── skills                # Frontend team practices
└── troubleshooting       # Frontend team patterns

Project Level (project-specific, unchanged):
project_abc123de/
├── checkpoints           # Project-specific sessions
└── research              # Project-specific research
```

### GitHub-Based Detection

**Leverage GitHub's Built-in Org/Team Model**:

```go
type RepositoryContext struct {
    // Git repository metadata
    Path         string  // /home/user/projects/backend-api
    RemoteURL    string  // git@github.com:acme-corp/backend-api.git

    // Automatically detected from GitHub
    Organization string  // acme-corp (from remote URL)
    Repository   string  // backend-api

    // Configured or detected
    Team         string  // backend (from CODEOWNERS, GitHub API, or config)
}

// Detection strategy:
// 1. Parse git remote → extract org (acme-corp)
// 2. Check CODEOWNERS file → extract team (@acme-corp/backend-team)
// 3. Fallback: GitHub API → get repo's team assignments
// 4. Fallback: Local config → .contextd/team.yaml
```

---

## Search Hierarchy (4-Tier)

When a developer searches for remediations:

```
Priority 1: PROJECT-SPECIFIC (highest relevance)
  Search: project_<hash>/remediations
  Example: Solutions specific to backend-api project

Priority 2: TEAM-SCOPED (high relevance)
  Search: team_backend/remediations
  Example: Backend team's patterns (microservices, databases, APIs)

Priority 3: ORG-SCOPED (medium relevance)
  Search: org_acme/remediations
  Example: Org-wide patterns (security, compliance, tooling)

Priority 4: PUBLIC KNOWLEDGE (low relevance, opt-in)
  Search: public_knowledge/remediations
  Example: Open source community solutions
```

**Key Principle**: Search from specific → general, stop at team boundary unless org-level

---

## Configuration Model

### Organization Configuration

**File**: `.contextd/org.yaml` (checked into org repos)

```yaml
organization: acme-corp

# Knowledge sharing settings
sharing:
  # Org-wide knowledge (accessible to all teams)
  org_level:
    remediations: true     # Security fixes, compliance patterns
    skills: true           # Org-wide best practices
    troubleshooting: true  # Infrastructure patterns

  # Team-level knowledge (accessible within team only)
  team_level:
    remediations: true     # Team-specific solutions
    skills: true           # Team processes
    troubleshooting: true  # Team tools

# Team definitions
teams:
  backend:
    repos:
      - backend-api
      - payment-service
      - user-service

  frontend:
    repos:
      - web-app
      - mobile-app

  platform:
    repos:
      - kubernetes-infra
      - monitoring
      - ci-cd-pipeline

# Default team assignment (fallback)
default_team: platform

# Cross-team projects (optional)
shared_projects:
  - name: project-x
    teams: [backend, frontend]
    repos:
      - project-x-api
      - project-x-ui
```

### Team Configuration (Optional Override)

**File**: `.contextd/team.yaml` (in team repos)

```yaml
team: backend

# Override org-level sharing settings
sharing:
  allow_org_search: true      # Can search org_acme/
  contribute_to_org: false    # Cannot publish to org level (admins only)

  allow_cross_team: false     # Cannot see other teams
```

---

## Access Control Matrix

| Knowledge Type | Project Scope | Team Scope | Org Scope | Public Scope |
|----------------|---------------|------------|-----------|--------------|
| **Checkpoints** | ✅ Own project only | ❌ No | ❌ No | ❌ No |
| **Research** | ✅ Own project only | ❌ No | ❌ No | ❌ No |
| **Remediations** | ✅ Search/Write | ✅ Search/Write | ✅ Search only* | ✅ Search only (opt-in) |
| **Skills** | ✅ Search/Write | ✅ Search/Write | ✅ Search only* | ✅ Search only (opt-in) |
| **Troubleshooting** | ✅ Search/Write | ✅ Search/Write | ✅ Search only* | ✅ Search only (opt-in) |

**\*Org-level write requires permission** (admin/maintainer role)

---

## Role-Based Access Control (RBAC)

### Roles

```go
type Role string

const (
    RoleAdmin      Role = "admin"       // Full control (org-wide)
    RoleMaintainer Role = "maintainer"  // Team lead (can publish to org)
    RoleDeveloper  Role = "developer"   // Team member (read/write team)
    RoleViewer     Role = "viewer"      // Read-only access
)
```

### Permission Matrix

| Action | Admin | Maintainer | Developer | Viewer |
|--------|-------|------------|-----------|--------|
| **Write to org/** | ✅ | ✅ | ❌ | ❌ |
| **Write to team/** | ✅ | ✅ | ✅ | ❌ |
| **Write to project/** | ✅ | ✅ | ✅ | ❌ |
| **Search org/** | ✅ | ✅ | ✅ | ✅ |
| **Search team/** | ✅ | ✅ | ✅ | ✅ |
| **Manage users** | ✅ | ❌ | ❌ | ❌ |
| **Configure teams** | ✅ | ❌ | ❌ | ❌ |

### User Assignment

**File**: `.contextd/users.yaml` (admin-managed)

```yaml
users:
  alice@acme.com:
    role: admin
    teams: [backend, frontend, platform]

  bob@acme.com:
    role: maintainer
    teams: [backend]

  charlie@acme.com:
    role: developer
    teams: [backend]

  diana@acme.com:
    role: developer
    teams: [frontend]
```

---

## Implementation Strategy

### v2.1 (Foundation) - 4 Weeks

**Scope**: Owner-based isolation (NO team support yet)

```
owner_alice/          # Alice's personal repos
owner_bob/            # Bob's personal repos
project_<hash>/       # Project-specific
```

**Benefits**:
- Fixes immediate cross-session confusion
- Establishes database scoping pattern
- No authentication complexity

**Migration**: v2.0 → v2.1 (owner-scoped)

---

### v2.2 (Team-Aware) - 8-12 Weeks After v2.1

**Scope**: Add team and org scoping

```
org_acme/             # NEW: Org-level knowledge
team_backend/         # NEW: Team-level knowledge
team_frontend/        # NEW: Team-level knowledge
project_<hash>/       # Unchanged
```

**New Features**:
1. **Team Detection**:
   - Parse CODEOWNERS
   - GitHub API integration
   - Manual config fallback

2. **Org Detection**:
   - Extract from git remote URL
   - Config file override

3. **RBAC**:
   - User-to-team mapping
   - Role-based permissions
   - Write access control

4. **Search Hierarchy**:
   - 4-tier search (project → team → org → public)
   - Configurable priorities

**Migration**: v2.1 → v2.2 (add team layer)

---

### 0.9.0-rc-1 (Enterprise) - 6+ Months

**Scope**: Full enterprise features

- OAuth/SSO integration
- Fine-grained ACLs
- Audit logging
- Multi-org support
- Public knowledge marketplace

---

## Example Workflows

### Workflow 1: Backend Developer Searches

```
Developer: charlie@acme.com (Backend Team)
Project: backend-api
Search: "database connection timeout"

Search Hierarchy:
1. project_abc123de/remediations
   → Result: "Postgres pool exhausted in backend-api" ✓

2. team_backend/remediations
   → Result: "Connection retry logic for microservices" ✓

3. org_acme/remediations
   → Result: "Standard DB connection settings for Acme" ✓

4. public_knowledge/remediations (if enabled)
   → Result: "Generic PostgreSQL tuning tips" ✓

Charlie sees: 4 results, all contextually relevant
  - Most relevant: backend-api specific
  - Useful: Backend team patterns
  - General: Org-wide standards
  - FYI: Public knowledge
```

### Workflow 2: Frontend Developer Searches (Different Team)

```
Developer: diana@acme.com (Frontend Team)
Project: web-app
Search: "database connection timeout"

Search Hierarchy:
1. project_def456gh/remediations
   → Result: (none)

2. team_frontend/remediations
   → Result: (none - frontend doesn't do DB directly)

3. org_acme/remediations
   → Result: "Standard DB connection settings for Acme" ✓

4. public_knowledge/remediations
   → Result: "Generic PostgreSQL tuning tips" ✓

Diana sees: 2 results
  - Does NOT see Backend team's microservices solutions
  - Sees org-wide standards (helpful)
  - Sees public knowledge
```

**Result**: Team isolation maintained, org knowledge shared appropriately.

---

### Workflow 3: Platform Team Publishes Org-Wide

```
Developer: alice@acme.com (Admin, Platform Team)
Action: Save remediation for "Kubernetes pod crash loop"

Decision: Should this be team-level or org-level?

# Alice's choice (via CLI or API):
contextd remediation create \
  --error "Kubernetes pod crash loop" \
  --solution "Check resource limits and liveness probes" \
  --scope org  # ← Publish to org_acme/ (accessible to all teams)

# Result:
# - Saved to: org_acme/remediations
# - Visible to: All teams (backend, frontend, platform)
# - Use case: Infrastructure knowledge benefits everyone
```

---

## Team Detection Methods (Priority Order)

### Method 1: CODEOWNERS File (Preferred)

```bash
# .github/CODEOWNERS in backend-api repo
* @acme-corp/backend-team

# contextd parses this:
# Org: acme-corp
# Team: backend-team → normalize to "backend"
```

**Pros**: Already exists in most orgs, authoritative
**Cons**: Requires GitHub-specific file

---

### Method 2: GitHub API (Automatic)

```bash
# Query GitHub API for repo metadata
GET /repos/acme-corp/backend-api/teams

# Response:
[
  {
    "name": "backend-team",
    "slug": "backend-team"
  }
]

# contextd uses: team = "backend"
```

**Pros**: No config needed, always up-to-date
**Cons**: Requires GitHub API access, rate limits

---

### Method 3: Local Config (Fallback)

```yaml
# .contextd/team.yaml in repo
team: backend
org: acme-corp
```

**Pros**: Works without GitHub, explicit control
**Cons**: Manual maintenance, can be out of sync

---

### Method 4: Org Config Mapping (Centralized)

```yaml
# .contextd/org.yaml (org-wide, checked into repos)
teams:
  backend:
    repos:
      - backend-api
      - payment-service
```

**Pros**: Centralized management, single source of truth
**Cons**: Requires org-level config repo

---

## Security Considerations

### Threat Model

**Threats Addressed**:
1. ✅ Cross-team data leakage (teams isolated)
2. ✅ Unauthorized org-level publishing (RBAC)
3. ✅ Team boundary bypass (search hierarchy enforcement)

**Threats Deferred to 0.9.0-rc-1**:
1. ⏸ Cross-org isolation (single org only in v2.2)
2. ⏸ Fine-grained ACLs (team-level only)
3. ⏸ Audit logging (0.9.0-rc-1)

### Defense Layers

1. **Database Boundaries**: Physical isolation (org/team/project DBs)
2. **Search Filters**: Enforce team membership in queries
3. **Write Permissions**: RBAC checks before org-level writes
4. **Config Validation**: Strict team assignment rules

---

## Migration Path

### v2.0 → v2.1 (Owner Scoping)

```bash
# Current v2.0
shared/
├── remediations (ALL users mixed)

# Migrate to v2.1
owner_alice/remediations
owner_bob/remediations
owner_charlie/remediations

# Strategy: Group by git owner from project_path metadata
```

---

### v2.1 → v2.2 (Team Awareness)

```bash
# Current v2.1
owner_alice/remediations  # Alice is on backend team
owner_bob/remediations    # Bob is on backend team
owner_charlie/remediations # Charlie is on frontend team

# Migrate to v2.2
team_backend/remediations   # Merge alice + bob
team_frontend/remediations  # Charlie's data
org_acme/remediations       # Admin-promoted patterns

# Strategy:
# 1. Detect teams from CODEOWNERS/GitHub API
# 2. Group owner databases by team
# 3. Promote high-value patterns to org level (manual/AI)
```

---

## Decision Points

### Question 1: Should v2.2 Support Multi-Org?

**Option A**: Single org only (simpler)
- Deployment: One contextd instance per organization
- Database: `org_acme/`, `team_*/`, `project_*/`
- Use case: 90% of enterprise deployments

**Option B**: Multi-org support (complex)
- Deployment: One contextd instance for multiple orgs
- Database: `org_acme/`, `org_competitor/`, etc.
- Use case: SaaS providers, MSPs

**Recommendation**: **Option A** for v2.2
- Defer multi-org to 0.9.0-rc-1
- Reduces complexity (no cross-org isolation needed)
- Matches realistic deployment (companies run their own instance)

---

### Question 2: How to Handle Cross-Team Projects?

**Option A**: Explicit project-level sharing
```yaml
# org.yaml
shared_projects:
  - name: project-x
    teams: [backend, frontend]
```

**Option B**: Create virtual team
```yaml
teams:
  project-x:  # Virtual team for cross-team project
    repos: [project-x-api, project-x-ui]
```

**Option C**: Use org-level for cross-team
```
# Backend and frontend publish to org_acme/
# Both teams can search org level
```

**Recommendation**: **Option A** (explicit sharing)
- Clear intent (project marked as cross-team)
- Bounded scope (only listed teams)
- Audit trail (who has access)

---

### Question 3: Default Search Scope?

**Option A**: Team-scoped (most private)
- Default: Search only project + team
- Opt-in: Search org level

**Option B**: Org-scoped (most collaborative)
- Default: Search project + team + org
- Opt-out: Restrict to team only

**Recommendation**: **Option B** (org-scoped default)
- Better collaboration (org knowledge visible)
- Still team-isolated (no cross-team leakage)
- Matches expected behavior (developers expect org standards)

---

## Implementation Checklist (v2.2)

### Phase 1: Team Detection (Weeks 1-2)
- [ ] Implement CODEOWNERS parser
- [ ] Integrate GitHub API for team lookup
- [ ] Add local config fallback
- [ ] Add org config mapping
- [ ] Unit tests for all detection methods

### Phase 2: Database Layer (Weeks 3-4)
- [ ] Add `DatabaseTypeOrg` and `DatabaseTypeTeam`
- [ ] Update database naming (`org_*`, `team_*`)
- [ ] Modify services to support team scope
- [ ] Update search hierarchy (4-tier)

### Phase 3: RBAC (Weeks 5-6)
- [ ] Implement role definitions
- [ ] Add permission checking middleware
- [ ] User-to-team mapping
- [ ] Org-level write restrictions

### Phase 4: Migration (Weeks 7-8)
- [ ] Migration tool (v2.1 → v2.2)
- [ ] Group owners by team
- [ ] Org-level promotion (manual selection)
- [ ] Validation and testing

### Phase 5: Configuration (Weeks 9-10)
- [ ] Org config schema
- [ ] Team config schema
- [ ] User config schema
- [ ] Config validation

### Phase 6: Testing & Release (Weeks 11-12)
- [ ] E2E tests (multi-team scenarios)
- [ ] Security tests (team isolation)
- [ ] Performance tests
- [ ] Documentation
- [ ] v2.2.0 release

---

## Success Criteria

**v2.2 Must Have**:
1. ✅ Team-level isolation (no cross-team leakage)
2. ✅ Org-level sharing (visible to all teams)
3. ✅ RBAC (admin/maintainer/developer roles)
4. ✅ Team auto-detection (CODEOWNERS/GitHub API)
5. ✅ 4-tier search (project → team → org → public)
6. ✅ Backward compatible (v2.1 → v2.2 migration)

**Nice to Have**:
1. ⏸ Cross-team projects (shared projects config)
2. ⏸ Audit logging (track access)
3. ⏸ Analytics (usage metrics per team)

---

## Summary

**Recommended Approach**:

1. **v2.1 (NOW)**: Owner-scoped isolation
   - Simple, ships in 4 weeks
   - Fixes cross-session confusion
   - Foundation for team features

2. **v2.2 (3-4 MONTHS)**: Team-aware architecture
   - Single org, multiple teams
   - Team detection (CODEOWNERS/GitHub API)
   - RBAC (admin/maintainer/developer)
   - 4-tier search (project → team → org → public)

3. **0.9.0-rc-1 (6+ MONTHS)**: Enterprise features
   - Multi-org support
   - OAuth/SSO
   - Fine-grained ACLs
   - Audit logging

**For Your Deployment** (single org, possibly multi-team):
- Use v2.2 architecture
- Start with single team, add more as needed
- Org-level sharing for infrastructure/security patterns
- Team-level sharing for team-specific knowledge

---

**Status**: Proposed for discussion and approval
**Next Steps**: Approve v2.1 first, then plan v2.2 based on user feedback
