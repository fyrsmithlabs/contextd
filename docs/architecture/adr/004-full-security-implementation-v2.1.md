# ADR 004: Full Security Implementation - v2.1 Accelerated

**Status**: Proposed
**Date**: 2025-01-07
**Supersedes**: ADR-003 incremental approach
**Context**: Single production user enables complete security implementation without phasing

---

## Context

**Original Plan** (ADR-003):
- v2.1: Owner-scoping only (4 weeks)
- v2.2: Add team awareness (3-4 months later)
- 0.9.0-rc-1: Add multi-org (6+ months later)

**New Reality**:
- Only ONE production user (you)
- Can test/iterate rapidly
- No migration compatibility needed (fresh start)
- Can implement complete solution immediately

**Decision**: Implement FULL security architecture in v2.1 (6-8 weeks)

---

## Decision

**Ship v2.1 with complete multi-tenant security**:

✅ Git-based owner detection
✅ Team detection (CODEOWNERS/GitHub API)
✅ Org-level sharing
✅ RBAC (admin/maintainer/developer roles)
✅ 4-tier search hierarchy (project → team → org → public)
✅ Complete isolation (no cross-team leakage)

**Benefits**:
- Single user can dogfood complete solution
- Find security issues before multi-user deployment
- No incremental migration complexity
- Faster time to production-ready

---

## Architecture

### Complete Database Hierarchy (v2.1)

```
org_<org_name>/               # Organization-wide knowledge
├── remediations              # Shared: Platform, security, compliance
├── skills                    # Shared: Org best practices
└── troubleshooting_patterns  # Shared: Infrastructure patterns

team_<team_name>/             # Team-specific knowledge
├── remediations              # Team-only: Team patterns
├── skills                    # Team-only: Team processes
└── troubleshooting_patterns  # Team-only: Team tools

project_<hash>/               # Project-specific (unchanged)
├── checkpoints               # Private: Session state
└── research                  # Private: Research notes
```

### Detection Strategy

```go
type RepositoryContext struct {
    // Filesystem
    ProjectPath  string  // /home/user/projects/backend-api

    // Git detection
    RemoteURL    string  // git@github.com:acme-corp/backend-api.git
    Organization string  // acme-corp (from URL)
    Repository   string  // backend-api (from URL)

    // Team detection (priority order)
    Team         string  // backend (from CODEOWNERS → GitHub API → config)
    TeamSource   string  // "codeowners", "github_api", "config", "default"

    // User detection
    GitUser      string  // From git config user.email
    GitName      string  // From git config user.name
}

func DetectContext(projectPath string) (*RepositoryContext, error) {
    ctx := &RepositoryContext{ProjectPath: projectPath}

    // 1. Parse git remote URL
    ctx.RemoteURL = readGitRemote(projectPath)
    ctx.Organization, ctx.Repository = parseRemoteURL(ctx.RemoteURL)

    // 2. Detect team (priority order)
    if team := parseCodeowners(projectPath); team != "" {
        ctx.Team = team
        ctx.TeamSource = "codeowners"
    } else if team := queryGitHubAPI(ctx.Organization, ctx.Repository); team != "" {
        ctx.Team = team
        ctx.TeamSource = "github_api"
    } else if team := readLocalConfig(projectPath); team != "" {
        ctx.Team = team
        ctx.TeamSource = "config"
    } else {
        ctx.Team = readOrgConfig(ctx.Organization).DefaultTeam
        ctx.TeamSource = "default"
    }

    // 3. Detect user
    ctx.GitUser = readGitConfig("user.email")
    ctx.GitName = readGitConfig("user.name")

    return ctx, nil
}
```

### Search Hierarchy (Complete)

```go
func (s *RemediationService) Search(ctx context.Context, req *SearchRequest) ([]Remediation, error) {
    repoCtx := DetectContext(req.ProjectPath)
    user := GetUserFromContext(ctx)  // From authentication

    results := []Remediation{}

    // 1. Project-specific (HIGHEST priority)
    projectDB := fmt.Sprintf("project_%s", hashPath(req.ProjectPath))
    results = append(results, s.searchDB(ctx, projectDB, req.Query)...)

    // 2. Team-level (HIGH priority)
    // SECURITY: Verify user is member of team
    if user.IsMemberOf(repoCtx.Team) {
        teamDB := fmt.Sprintf("team_%s", repoCtx.Team)
        results = append(results, s.searchDB(ctx, teamDB, req.Query)...)
    }

    // 3. Org-level (MEDIUM priority)
    // SECURITY: Verify user is member of org
    if user.IsMemberOf(repoCtx.Organization) {
        orgDB := fmt.Sprintf("org_%s", repoCtx.Organization)
        results = append(results, s.searchDB(ctx, orgDB, req.Query)...)
    }

    // 4. Public (LOW priority, opt-in)
    if s.config.AllowPublicSearch {
        results = append(results, s.searchDB(ctx, "public", req.Query)...)
    }

    return rankAndDedupe(results), nil
}
```

### RBAC Implementation

```go
type User struct {
    ID           string
    Email        string   // From git config or auth
    Name         string   // From git config
    Organizations []string // Orgs user belongs to
    Teams        []string // Teams user belongs to
    Roles        map[string]Role // Role per org/team
}

type Role string
const (
    RoleAdmin      Role = "admin"       // Full control
    RoleMaintainer Role = "maintainer"  // Team lead
    RoleDeveloper  Role = "developer"   // Team member
    RoleViewer     Role = "viewer"      // Read-only
)

func (s *RemediationService) Create(ctx context.Context, req *CreateRequest) error {
    user := GetUserFromContext(ctx)
    repoCtx := DetectContext(req.ProjectPath)

    // Determine target scope
    scope := req.Scope  // "project", "team", or "org"

    // RBAC check
    switch scope {
    case "org":
        // Requires admin or maintainer role
        if !user.HasRole(repoCtx.Organization, RoleAdmin, RoleMaintainer) {
            return ErrUnauthorized
        }
        dbName = fmt.Sprintf("org_%s", repoCtx.Organization)

    case "team":
        // Requires team membership
        if !user.IsMemberOf(repoCtx.Team) {
            return ErrUnauthorized
        }
        dbName = fmt.Sprintf("team_%s", repoCtx.Team)

    case "project":
        // Anyone can write to their own project
        dbName = fmt.Sprintf("project_%s", hashPath(req.ProjectPath))
    }

    return s.store.Insert(ctx, dbName, "remediations", vectors)
}
```

---

## Configuration

### Organization Config (Git-tracked)

**File**: `.contextd/org.yaml` (in org's config repo)

```yaml
organization: acme-corp

# Team definitions
teams:
  backend:
    description: Backend services team
    repos:
      - backend-api
      - payment-service
      - user-service
    default_role: developer

  frontend:
    description: Frontend applications team
    repos:
      - web-app
      - mobile-app
    default_role: developer

  platform:
    description: Platform and infrastructure team
    repos:
      - kubernetes-infra
      - monitoring
      - ci-cd
    default_role: developer

# Default team for repos not in above lists
default_team: platform

# Knowledge sharing policies
sharing:
  org_level:
    enabled: true
    remediations: true
    skills: true
    troubleshooting: true

  team_level:
    enabled: true
    remediations: true
    skills: true
    troubleshooting: true

  public_level:
    enabled: false  # Disable until ready
```

### User Configuration (Git-tracked, admin-managed)

**File**: `.contextd/users.yaml`

```yaml
users:
  # Your user (admin)
  you@company.com:
    name: Your Name
    role: admin
    organizations: [acme-corp]
    teams: [backend, frontend, platform]
    permissions:
      can_publish_org: true
      can_manage_users: true
      can_configure_teams: true

  # Future team members (when you add them)
  # teammate@company.com:
  #   name: Teammate Name
  #   role: developer
  #   organizations: [acme-corp]
  #   teams: [backend]

# Single-user mode (current)
single_user_mode: true
default_user: you@company.com
```

### Project-Level Override (Optional)

**File**: `.contextd/project.yaml` (in specific repo)

```yaml
# Override team assignment
team: backend  # Force this repo to backend team

# Override sharing for this project
sharing:
  project_scope_only: false  # Allow team/org search
  contribute_to_team: true   # Save to team level
  contribute_to_org: false   # Don't save to org (project-specific solutions)
```

---

## Implementation Plan (6-8 Weeks)

### Week 1-2: Core Detection & Database Layer

**Tasks**:
- [ ] Implement `DetectRepositoryContext()`
  - Git remote URL parsing
  - Org extraction
  - Repo name extraction
- [ ] Implement team detection
  - CODEOWNERS parser
  - GitHub API client
  - Local config fallback
- [ ] Add database types (org, team, project)
- [ ] Update vectorstore adapter (create org/team databases)
- [ ] Unit tests (detection logic)

**Deliverable**: Can detect org/team from any git repo

---

### Week 3-4: Service Layer Updates

**Tasks**:
- [ ] Update `pkg/remediation` service
  - Add scope parameter (project/team/org)
  - Implement 4-tier search
  - Add RBAC checks
- [ ] Update `pkg/skills` service
  - Same changes as remediation
- [ ] Update `pkg/troubleshooting` service
  - Same changes as remediation
- [ ] Keep `pkg/checkpoint` unchanged (project-only)
- [ ] Integration tests (multi-scope)

**Deliverable**: Services support org/team/project scopes

---

### Week 5: RBAC & User Management

**Tasks**:
- [ ] Implement User model
- [ ] Implement Role model
- [ ] Add permission checking middleware
- [ ] Load users.yaml config
- [ ] Single-user mode (auto-auth as default user)
- [ ] Unit tests (RBAC logic)

**Deliverable**: RBAC enforced, single-user auto-auth works

---

### Week 6: Configuration & CLI

**Tasks**:
- [ ] Implement org.yaml loader
- [ ] Implement users.yaml loader
- [ ] Implement project.yaml loader (optional)
- [ ] Add CLI commands:
  - `contextd teams list`
  - `contextd teams show <team>`
  - `contextd users list`
  - `contextd config validate`
- [ ] Config validation

**Deliverable**: Configuration system complete

---

### Week 7: Migration & Testing

**Tasks**:
- [ ] Migration tool (v2.0 → v2.1)
  - Analyze existing shared/ database
  - Detect teams from project paths
  - Migrate to team databases
  - Promote high-value patterns to org
- [ ] End-to-end testing
  - Multi-team scenarios
  - RBAC enforcement
  - Search hierarchy
- [ ] Performance testing
- [ ] Security testing (no cross-team leakage)

**Deliverable**: Migration tool works, all tests pass

---

### Week 8: Documentation & Release

**Tasks**:
- [ ] Update all documentation
- [ ] Write deployment guide
- [ ] Create example configs
- [ ] Migration guide (v2.0 → v2.1)
- [ ] Release notes
- [ ] CHANGELOG update
- [ ] v2.1.0 release

**Deliverable**: Production-ready release

---

## Dogfooding Strategy

**You as the Test User**:

### Phase 1: Single Team (Week 3-4)
```yaml
# Your initial config
organization: your-org
teams:
  engineering:  # Single team
    repos: [contextd, other-projects]

users:
  you@email.com:
    role: admin
    teams: [engineering]
```

**Test**:
- Create remediations in different scopes
- Verify search finds correct scope
- Check no cross-team leakage (only one team exists)

---

### Phase 2: Multi-Team Simulation (Week 5-6)
```yaml
# Simulate multi-team org
organization: your-org
teams:
  backend:
    repos: [contextd, api-service]
  frontend:
    repos: [web-app]

users:
  you@email.com:
    role: admin
    teams: [backend, frontend]  # You're on both (for testing)
```

**Test**:
- Create backend-specific remediation
- Switch to frontend repo
- Verify search doesn't show backend remediation
- Create org-level skill
- Verify both teams see it

---

### Phase 3: RBAC Testing (Week 6-7)
```yaml
# Test different roles (simulate by changing config)
users:
  you@email.com:
    role: developer  # Temporarily demote yourself
    teams: [backend]
```

**Test**:
- Try to publish to org level → should fail
- Publish to team level → should work
- Search org level → should work (read-only)

---

### Phase 4: Real Usage (Week 8+)

Use contextd for real work across your projects:
- Personal projects
- Work client projects (if applicable)
- Open source contributions

**Monitor**:
- Cross-session confusion gone?
- Search results relevant?
- Scope detection accurate?
- Any security issues?

---

## Success Criteria

**Must Have** (v2.1 Release):

1. ✅ Org/team/project detection works
2. ✅ 4-tier search hierarchy functional
3. ✅ RBAC enforced (no unauthorized writes)
4. ✅ No cross-team data leakage
5. ✅ Configuration system works
6. ✅ Migration tool (v2.0 → v2.1)
7. ✅ Single-user mode (your use case)
8. ✅ >80% test coverage
9. ✅ <100ms search performance
10. ✅ Complete documentation

**Nice to Have** (Can defer):

1. ⏸ GitHub API caching (fallback to local config OK)
2. ⏸ Public knowledge marketplace (disable for now)
3. ⏸ Multi-org support (single org only)
4. ⏸ OAuth/SSO (single-user bearer token OK)

---

## Risk Assessment

### Risk 1: Complexity

**Concern**: Implementing too much at once

**Mitigation**:
- You're the only user (no migration compatibility needed)
- Can iterate rapidly
- Incremental testing (phase 1-4)
- Rollback to v2.0 if needed

### Risk 2: Team Detection Failures

**Concern**: CODEOWNERS parsing or GitHub API issues

**Mitigation**:
- Local config fallback always works
- Explicit org.yaml mapping
- Manual override in project.yaml
- Fail gracefully (default team assignment)

### Risk 3: RBAC Bugs

**Concern**: Permission checks have holes

**Mitigation**:
- Comprehensive security tests
- You'll find issues during dogfooding
- Single user = lower blast radius
- Fix before multi-user deployment

### Risk 4: Performance

**Concern**: 4-tier search too slow

**Mitigation**:
- Parallel database queries
- Early termination (stop when enough results)
- Caching (vector store handles this)
- Benchmark before release

---

## Migration Path

### Your Current State (v2.0)

```
shared/
├── remediations (YOUR data + any test data)
```

### After v2.1 Migration

**Automatic Detection**:
```bash
# Migration analyzes your repos
contextd migrate analyze
# Output:
# Found repos:
#   - /home/user/contextd → org=axyzlabs, team=platform (from CODEOWNERS)
#   - /home/user/client-work → org=client, team=engineering (from remote)
#   - /home/user/personal → org=johndoe, team=personal (fallback)

# Migration creates databases
contextd migrate execute
# Creates:
#   - org_axyzlabs/
#   - team_platform/
#   - org_client/
#   - team_engineering/
#   - org_johndoe/
#   - team_personal/
#   - project_<hash>/ for each repo
```

**Manual Promotion** (optional):
```bash
# After migration, review and promote valuable patterns to org
contextd remediation promote <id> --from team_platform --to org_axyzlabs
```

---

## Advantages of Full Implementation

**vs Incremental Approach**:

| Aspect | Incremental (ADR-003) | Full (ADR-004) |
|--------|---------------------|----------------|
| **Timeline** | 4 weeks + 3-4 months = 7 months | 6-8 weeks total |
| **Migrations** | 2 (v2.0→v2.1, v2.1→v2.2) | 1 (v2.0→v2.1) |
| **Testing** | Partial per phase | Complete from start |
| **Security** | Delayed team isolation | Immediate team isolation |
| **Dogfooding** | Owner-only first | Full feature set |
| **Production Ready** | After v2.2 (7 months) | After v2.1 (8 weeks) |

**Benefits for You**:
- ✅ Faster to production-ready (8 weeks vs 7 months)
- ✅ Dogfood complete solution immediately
- ✅ Find security issues early
- ✅ No incremental migration pain
- ✅ Ready for team deployment when needed

---

## Alternatives Considered

### Alternative 1: Stick to ADR-003 (Incremental)

**Pros**: Lower risk, simpler each phase
**Cons**: 7 months to production-ready, multiple migrations
**Decision**: Rejected - you're the only user, no need for caution

### Alternative 2: Skip to 0.9.0-rc-1 (Multi-Org)

**Pros**: Complete enterprise solution
**Cons**: OAuth/SSO overhead, too complex for single user
**Decision**: Rejected - over-engineering for current need

### Alternative 3: Minimal v2.1 (Project-Only)

**Pros**: Simplest possible
**Cons**: No team/org features, not production-ready
**Decision**: Rejected - doesn't solve your multi-team use case

---

## Conclusion

**Implement complete security architecture in v2.1** (6-8 weeks):

✅ Org/team/project hierarchy
✅ RBAC (admin/maintainer/developer)
✅ 4-tier search
✅ Complete isolation
✅ Single migration (v2.0 → v2.1)
✅ Production-ready in 2 months

**Rationale**:
- You're the only user (can iterate rapidly)
- Dogfood complete solution
- Find issues before multi-user
- Faster to production (8 weeks vs 7 months)

**Next Steps**:
1. Approve this ADR
2. Create v2.1-full milestone
3. Begin Week 1 implementation
4. Dogfood throughout development
5. Ship production-ready v2.1

---

**Status**: Awaiting approval
**Supersedes**: ADR-003 (incremental approach)
**Related**: TEAM-AWARE-ARCHITECTURE-V2.2.md (now merged into v2.1)
