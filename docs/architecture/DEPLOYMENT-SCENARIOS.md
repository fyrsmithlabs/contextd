# Deployment Scenarios & Architecture Guidance

**Date**: 2025-01-07
**Purpose**: Guide deployment architecture selection based on use case

---

## Deployment Scenarios

### Scenario 1: Single Developer (Personal Use)

**Profile**:
- One developer
- Multiple personal projects
- Local development only

**Architecture**: **v2.1 (Owner-Scoped)**

```
owner_johndoe/
├── remediations      # Shared across all johndoe's repos
├── skills
└── troubleshooting

project_blog/
├── checkpoints       # blog-specific

project_sideproject/
├── checkpoints       # sideproject-specific
```

**Configuration**: Zero config (auto-detect git owner)

**Timeline**: Available in 4 weeks

---

### Scenario 2: Single Team (Startup)

**Profile**:
- One organization (Acme Corp)
- One development team (10 developers)
- All work on shared repositories

**Architecture**: **v2.2 (Team-Aware, Single Team)**

```
org_acme/
├── remediations      # Org-wide (whole team shares)
├── skills
└── troubleshooting

project_api/
├── checkpoints       # API-specific

project_web/
├── checkpoints       # Web-specific
```

**Configuration**:
```yaml
# .contextd/org.yaml
organization: acme
teams:
  engineering:  # Single team
    repos: [api, web, mobile]
```

**Timeline**: Available in 3-4 months (after v2.1)

---

### Scenario 3: Single Org, Multiple Teams

**Profile**:
- One organization (Acme Corp)
- Multiple teams (Backend, Frontend, Platform)
- Teams need isolation + org-level sharing

**Architecture**: **v2.2 (Team-Aware, Multi-Team)** ⭐ **YOUR USE CASE**

```
org_acme/
├── remediations      # Org-wide (infrastructure, security)
├── skills
└── troubleshooting

team_backend/
├── remediations      # Backend team only
├── skills
└── troubleshooting

team_frontend/
├── remediations      # Frontend team only
├── skills
└── troubleshooting

project_api/
├── checkpoints       # API-specific
```

**Search Hierarchy**:
1. Project-specific (highest priority)
2. Team-level (Backend sees Backend patterns)
3. Org-level (Everyone sees org standards)
4. Public (opt-in)

**Configuration**:
```yaml
# .contextd/org.yaml
organization: acme

teams:
  backend:
    repos: [api, services, database]
  frontend:
    repos: [web, mobile]
  platform:
    repos: [k8s, monitoring, ci-cd]

sharing:
  org_level:
    remediations: true    # Org-wide security/compliance
  team_level:
    remediations: true    # Team-specific patterns
```

**RBAC**:
```yaml
# .contextd/users.yaml
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
```

**Timeline**: Available in 3-4 months (after v2.1)

---

### Scenario 4: Multiple Organizations (MSP/SaaS)

**Profile**:
- Managed Service Provider
- Multiple client organizations
- Complete isolation required

**Architecture**: **0.9.0-rc-1 (Multi-Org)**

```
org_client_a/
├── team_dev/
└── team_ops/

org_client_b/
├── team_engineering/

org_msp_internal/
├── team_platform/
```

**Requirements**:
- OAuth/SSO per organization
- Complete org isolation
- Cross-org data leakage IMPOSSIBLE

**Timeline**: 6+ months (future)

---

## Decision Matrix

| Use Case | Recommended Version | Key Features | Timeline |
|----------|-------------------|--------------|----------|
| **Personal (1 dev)** | v2.1 | Owner-scoped | 4 weeks |
| **Startup (1 team)** | v2.2 | Org-level sharing | 3-4 months |
| **Enterprise (multi-team)** | v2.2 | Team isolation + org sharing | 3-4 months |
| **MSP (multi-org)** | 0.9.0-rc-1 | Full org isolation | 6+ months |

---

## Your Deployment: Single Org, Multi-Team

**Recommendation**: **v2.2 (Team-Aware)**

### Why v2.2 Fits Your Need

✅ **Team Isolation**: Backend team doesn't see Frontend team's solutions
✅ **Org-Level Sharing**: Platform team's infrastructure patterns visible to all
✅ **Flexible Sharing**: Choose what's team-only vs org-wide
✅ **Auto-Detection**: Reads CODEOWNERS or GitHub API for team assignment
✅ **RBAC**: Admins can publish to org, developers to team

### Implementation Path

**Phase 1: v2.1 (4 weeks)**
- Deploy owner-scoped isolation
- Each developer's personal repos isolated
- Foundation for team features

**Phase 2: v2.2 (3-4 months)**
- Add team detection (CODEOWNERS/GitHub API)
- Implement org-level database
- Add RBAC (admin/maintainer/developer)
- Enable 4-tier search (project → team → org → public)

**Phase 3: Production (1-2 weeks after v2.2)**
- Configure teams in `.contextd/org.yaml`
- Assign users to teams
- Migrate existing data (v2.1 → v2.2)
- Deploy to organization

---

## Configuration Examples

### For Your Org (Acme Corp)

```yaml
# .contextd/org.yaml (checked into shared config repo)
organization: acme-corp

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
      - ci-cd

# Knowledge sharing
sharing:
  org_level:
    remediations: true     # Platform patterns, security fixes
    skills: true           # Org best practices
    troubleshooting: true  # Infrastructure troubleshooting

  team_level:
    remediations: true     # Team-specific solutions
    skills: true           # Team processes
    troubleshooting: true  # Team tools

# Default team for repos not in above list
default_team: platform
```

```yaml
# .contextd/users.yaml (managed by admins)
users:
  # Platform team (admins)
  alice@acme.com:
    role: admin
    teams: [backend, frontend, platform]

  # Backend team
  bob@acme.com:
    role: maintainer
    teams: [backend]

  charlie@acme.com:
    role: developer
    teams: [backend]

  # Frontend team
  diana@acme.com:
    role: maintainer
    teams: [frontend]

  eve@acme.com:
    role: developer
    teams: [frontend]
```

---

## Key Benefits for Your Use Case

### Team Isolation
```
Backend Developer searches "authentication error":
  ✓ Sees: Backend team's OAuth patterns
  ✓ Sees: Org-wide security standards
  ✗ Does NOT see: Frontend team's JWT implementation
```

### Org-Level Sharing
```
Platform Team publishes "Kubernetes best practices":
  → Saved to: org_acme/skills
  → Visible to: All teams (backend, frontend, platform)
  → Use case: Infrastructure knowledge benefits everyone
```

### Flexible Permissions
```
Admin (alice):
  - Can publish to org_acme/ (whole org sees it)
  - Can manage teams and users

Maintainer (bob - backend lead):
  - Can publish to team_backend/ (backend team sees it)
  - Can publish to org_acme/ (with admin approval)

Developer (charlie):
  - Can publish to team_backend/ (backend team sees it)
  - Can search org_acme/ (read-only)
```

---

## Migration Strategy

### Current State (v2.0)
```
shared/
├── remediations (ALL teams mixed - SECURITY ISSUE)
```

### Intermediate (v2.1) - 4 Weeks
```
owner_alice/remediations
owner_bob/remediations
owner_charlie/remediations
```

### Final (v2.2) - 3-4 Months
```
org_acme/remediations      # Platform patterns
team_backend/remediations  # Backend team
team_frontend/remediations # Frontend team
```

**Data Migration**:
1. v2.0 → v2.1: Group by git owner
2. v2.1 → v2.2: Group owners by team, promote valuable patterns to org

---

## Deployment Recommendations

**For Single Org, Multi-Team** (YOUR CASE):

1. **Deploy v2.1 immediately** (4 weeks):
   - Fixes cross-session confusion
   - Establishes isolation patterns
   - No team complexity yet

2. **Plan v2.2** (during v2.1):
   - Define teams
   - Assign users
   - Decide org-level vs team-level sharing policies

3. **Deploy v2.2** (3-4 months):
   - Enable team detection
   - Add RBAC
   - Migrate to team databases

4. **Iterate**:
   - Adjust team assignments
   - Refine sharing policies
   - Promote high-value patterns to org level

---

## Success Metrics

### v2.1 Success Criteria
- [ ] No cross-owner data leakage
- [ ] Cross-session confusion eliminated
- [ ] Auto-detection from git remote works
- [ ] <100ms search performance

### v2.2 Success Criteria
- [ ] No cross-team data leakage
- [ ] Org-level sharing works
- [ ] RBAC enforced
- [ ] Team auto-detection works (CODEOWNERS/GitHub API)
- [ ] 4-tier search (project → team → org → public)

---

## Summary

**For Your Deployment** (single org, multiple teams):

✅ **Use v2.2 (Team-Aware)** when available
✅ **Start with v2.1** to fix immediate issues
✅ **Configure teams** via org.yaml
✅ **Auto-detect** from CODEOWNERS
✅ **RBAC** for org-level publishing

**Timeline**:
- Week 4: v2.1 deployed (owner-scoped)
- Month 4: v2.2 deployed (team-aware)
- Your org is secure, collaborative, and scalable

---

**Questions? See**:
- [TEAM-AWARE-ARCHITECTURE-V2.2.md](TEAM-AWARE-ARCHITECTURE-V2.2.md) - Full v2.2 design
- [ADR-003](adr/003-single-developer-multi-repo-isolation.md) - v2.1 foundation
