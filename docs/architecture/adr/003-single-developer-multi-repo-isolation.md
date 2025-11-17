# ADR 003: Single-Developer Multi-Repo Isolation

**Status**: Proposed
**Date**: 2025-01-07
**Supersedes**: Org/team features in CRITICAL-MULTI-TENANT-REDESIGN.md

---

## Context

**Problem**: Current "shared" database causes cross-project confusion and potential data leakage.

**User Report**: "cross session confusion between projects"

**Original Plan**: Full GitHub org/team/RBAC architecture (CRITICAL-MULTI-TENANT-REDESIGN.md)

**Reality Check**:
- OAuth/SSO/RBAC is months of work
- 90% of users are **single developers** with multiple repos
- Team features can wait - fix the immediate UX/security issue first

---

## Decision

**Implement single-developer multi-repo isolation in v2.1:**

Focus on **owner-scoped databases** without org/team complexity:

```
Database Structure (v2.1):

owner_<identifier>/              # User's shared knowledge
├── remediations                 # Shared across user's repos
├── skills                       # User's templates
└── troubleshooting_patterns     # User's patterns

project_<hash>/                  # Per-project isolation
├── checkpoints                  # Project-specific
└── research                     # Project-specific

Future (0.9.0-rc-1+):
org_<identifier>/                # Deferred to 0.9.0-rc-1
├── remediations
└── ...
```

**Defer to future versions:**
- Organization databases
- Team permissions
- RBAC
- OAuth/SSO
- Multi-user authentication

---

## Rationale

### Why Single-Developer First?

**User Demographics:**
- 90% single developers (personal projects, freelance, side projects)
- 10% teams/enterprises (can wait for 0.9.0-rc-1)

**Developer Workflows:**
```
Typical Developer:
├── personal-blog (public, GitHub)
├── side-project (private, GitHub)
├── freelance-client-a (private, GitLab)
├── freelance-client-b (private, GitHub)
└── learning-rust (public, GitHub)

Problem: ALL these share knowledge in current "shared" db
Solution: Scope by git repository owner/origin
```

**Benefits of Simple Approach:**
- ✅ Ships in weeks, not months
- ✅ Solves 90% of user pain (cross-session confusion)
- ✅ No authentication complexity
- ✅ No OAuth/SSO integration
- ✅ Backward compatible migration
- ✅ Foundation for future team features

---

## Design

### Repository Identity Detection

**Use git remote origin to determine scope:**

```go
type RepositoryIdentity struct {
    Path       string  // /home/user/projects/my-app
    RemoteURL  string  // git@github.com:johndoe/my-app.git
    Owner      string  // johndoe (extracted from remote)
    RepoName   string  // my-app
    OwnerType  string  // "personal" (v2.1 only supports this)
}

// DetectRepositoryIdentity reads .git/config and extracts owner
func DetectRepositoryIdentity(projectPath string) (*RepositoryIdentity, error) {
    // Read .git/config
    // Parse [remote "origin"] url
    // Extract owner from URL:
    //   - git@github.com:owner/repo.git → owner
    //   - https://github.com/owner/repo.git → owner
    //   - git@gitlab.com:owner/repo.git → owner

    return &RepositoryIdentity{
        Path:      projectPath,
        RemoteURL: remoteURL,
        Owner:     extractOwner(remoteURL),
        RepoName:  extractRepoName(remoteURL),
        OwnerType: "personal", // v2.1: Always personal
    }
}
```

**Supported Git Hosts:**
- GitHub (git@github.com:owner/repo.git)
- GitLab (git@gitlab.com:owner/repo.git)
- Bitbucket (git@bitbucket.org:owner/repo.git)
- Self-hosted (git@git.company.com:owner/repo.git)
- Generic (any git@ or https:// URL with owner/repo pattern)

**Fallback for Non-Git Repositories:**
```go
// If no .git directory found:
// Use hash of full path as "owner" identifier
// Database: owner_<hash_of_user_home_dir>
// This ensures all non-git projects share one scope
```

### Database Scoping Strategy

**Simple Two-Tier Model:**

```
Tier 1: Project-Specific (private to project)
  - Database: project_<hash_of_project_path>
  - Contains: checkpoints, research, notes
  - Visibility: This project only
  - Search Priority: HIGHEST

Tier 2: Owner-Scoped (shared across owner's repos)
  - Database: owner_<git_owner_or_user_hash>
  - Contains: remediations, skills, troubleshooting_patterns
  - Visibility: All repos with same owner
  - Search Priority: MEDIUM

Tier 3: Public Knowledge (opt-in, future)
  - Database: public_knowledge
  - Contains: User-contributed public knowledge
  - Visibility: Everyone (opt-in only)
  - Search Priority: LOWEST
  - Status: Deferred to v2.2+
```

**CRITICAL: Project vs Owner Scope Separation**

These tiers are **completely orthogonal** and do NOT interfere:

| Aspect | Project Scope | Owner Scope |
|--------|--------------|-------------|
| **Database Name** | `project_<hash>` | `owner_<hash>` |
| **Hash Source** | SHA256(project_path) | SHA256(git_owner) |
| **Services** | Checkpoint only | Remediation, Skills, Troubleshooting |
| **Collections** | checkpoints, research, notes | remediations, skills, troubleshooting_patterns |
| **Isolation** | Per project path | Per git repository owner |
| **Search Scope** | Single project only | All owner's repos |

**No Collision Risk**:
- Different hash sources (path vs owner) → different values
- Different prefixes (`project_` vs `owner_`) → different names
- Different collections → no data mixing
- Type-safe routing (`DatabaseTypeProject` vs `DatabaseTypeOwner`) → no misrouting

**Example**:
```
Project A (/home/user/my-app, owner: johndoe):
  project_abc123de/checkpoints       ← Project-specific (ISOLATED)
  owner_johndoe45/remediations       ← Owner-shared (across johndoe's repos)

Project B (/home/user/client-work, owner: client-org):
  project_def456gh/checkpoints       ← Project-specific (ISOLATED)
  owner_client78/remediations        ← Owner-shared (DIFFERENT owner, NO overlap)
```

**Guarantee**:
- Checkpoints ALWAYS stay private to project
- Remediations/skills shared ONLY across same owner's repos
- ZERO interference between project and owner scopes
- Physical database boundaries enforce isolation

### Search Behavior (Fixes Cross-Session Confusion)

**Before (v2.0 - BROKEN):**
```
User in project "personal-blog":
  Searches: "database connection error"

  Search: shared/remediations
  Results:
    1. PostgreSQL error (personal-blog)
    2. MySQL error (client-project-a) ← WRONG CONTEXT
    3. MongoDB error (client-project-b) ← WRONG CONTEXT
    4. Redis error (side-project) ← COULD BE RELEVANT

  Problem: Can't tell which is which!
```

**After (v2.1 - FIXED):**
```
User in project "personal-blog":
  Searches: "database connection error"

  Step 1: Search project_abc123/remediations (personal-blog specific)
    Results: (none in project-specific)

  Step 2: Search owner_johndoe/remediations (all johndoe's repos)
    Results:
      1. PostgreSQL error (from personal-blog) ← SAME OWNER ✓
      2. Redis error (from side-project) ← SAME OWNER ✓

    NOT returned:
      ✗ MySQL error (from client-project-a) ← DIFFERENT OWNER
      ✗ MongoDB error (from client-project-b) ← DIFFERENT OWNER

  Context is clear: All results are from johndoe's personal projects
```

### Configuration

**Simple Config (v2.1):**

```yaml
# ~/.config/contextd/config.yaml
knowledge_sharing:
  # Scope for shared knowledge (remediations, skills, patterns)
  # Options: "owner" (default), "project", "disabled"
  scope: owner

  # Which collections to share at owner scope
  # Set to false to keep project-specific only
  share_remediations: true
  share_skills: true
  share_patterns: true
```

**Examples:**

```yaml
# Default: Share across all my repos
knowledge_sharing:
  scope: owner

# Paranoid: Keep everything project-specific
knowledge_sharing:
  scope: project

# Custom: Share patterns but not remediations
knowledge_sharing:
  scope: owner
  share_remediations: false  # Keep errors private to project
  share_skills: true          # Share templates across my repos
  share_patterns: true        # Share troubleshooting patterns
```

---

## Implementation Plan

### Phase 1: Repository Detection (Week 1)

```
Tasks:
- [ ] Implement DetectRepositoryIdentity()
- [ ] Extract owner from git remote URL
- [ ] Support GitHub, GitLab, Bitbucket, generic
- [ ] Fallback for non-git repositories
- [ ] Unit tests for URL parsing
- [ ] Integration tests
```

**Acceptance Criteria:**
- Correctly extracts owner from all git URL formats
- Handles missing .git directory gracefully
- Deterministic owner for non-git repos

### Phase 2: Database Restructuring (Week 2)

```
Tasks:
- [ ] Add owner_<identifier>/ database creation
- [ ] Update remediation service to use owner scope
- [ ] Update skills service to use owner scope
- [ ] Update troubleshooting service to use owner scope
- [ ] Keep checkpoint service project-scoped (no change)
- [ ] Add configuration for sharing preferences
```

**Acceptance Criteria:**
- Remediations saved to owner_<owner>/remediations
- Search checks project_<hash> first, then owner_<owner>
- Configuration respected (can disable sharing)

### Phase 3: Migration Tool (Week 3)

```
Tasks:
- [ ] Analyze existing shared/ database
- [ ] Group by detected repository owner
- [ ] Migrate to owner_<owner>/ databases
- [ ] Preserve metadata and embeddings
- [ ] Validation: No data loss
- [ ] Backup before migration
- [ ] Rollback capability
```

**Acceptance Criteria:**
- All existing remediations migrated
- Grouped by repository owner
- Backward compatibility maintained
- Clean migration path

### Phase 4: Testing & Documentation (Week 4)

```
Tasks:
- [ ] End-to-end testing
- [ ] Cross-session confusion resolved
- [ ] Performance testing (search across multiple owner DBs)
- [ ] Update documentation
- [ ] Migration guide
- [ ] Release notes
```

**Acceptance Criteria:**
- No cross-owner data leakage
- Search returns contextually relevant results
- Performance < 100ms for search
- Clear documentation for users

---

## Migration Strategy

### Existing Users (v2.0 → v2.1)

```bash
# Step 1: Backup
contextd backup create --output ~/contextd-v2.0-backup.tar.gz

# Step 2: Analyze current data
contextd migrate analyze-owners
# Output:
# Found 47 remediations in shared/
# Detected 3 unique owners:
#   - johndoe (32 remediations across 5 repos)
#   - unknown (10 remediations, non-git repos)
#   - clienta (5 remediations, client project)

# Step 3: Preview migration
contextd migrate to-owner-scope --dry-run
# Output:
# Will create:
#   - owner_johndoe/ (32 remediations)
#   - owner_<hash>/ (10 remediations, fallback)
#   - owner_clienta/ (5 remediations)

# Step 4: Execute migration
contextd migrate to-owner-scope --confirm

# Step 5: Validate
contextd validate-migration
# Output:
# ✓ All 47 remediations migrated
# ✓ No data loss
# ✓ Embeddings preserved
# ✓ Search functional
```

### New Users (v2.1+)

**Zero configuration** - automatic owner detection:

```bash
# User clones repo
cd ~/projects/my-app

# User runs contextd
contextd

# Automatic:
# 1. Detects git remote: git@github.com:johndoe/my-app.git
# 2. Extracts owner: johndoe
# 3. Creates databases:
#    - project_<hash>/ (for checkpoints)
#    - owner_johndoe/ (for shared knowledge, if doesn't exist)
# 4. Ready to use - no config needed
```

---

## Edge Cases

### 1. Multiple Git Remotes

```bash
# Repo has multiple remotes
git remote -v
  origin    git@github.com:johndoe/my-app.git (fetch)
  upstream  git@github.com:upstream/my-app.git (fetch)
```

**Solution**: Use `origin` remote (convention), fallback to first remote

### 2. No Git Repository

```bash
# Working directory has no .git
cd ~/projects/scratch-code
```

**Solution**: Hash user's home directory → `owner_<hash_of_home_dir>`
- All non-git projects share one owner scope
- Still isolated from git-based projects

### 3. Forked Repositories

```bash
# User forks someone else's repo
git remote -v
  origin  git@github.com:johndoe/forked-repo.git (fetch)
  upstream git@github.com:original/forked-repo.git (fetch)
```

**Solution**: Use `origin` owner (johndoe), not upstream
- Fork is treated as johndoe's repo
- Shares knowledge with johndoe's other repos

### 4. Organization Repos (Single Developer)

```bash
# Developer works alone on org repo
git remote -v
  origin  git@github.com:my-company/backend.git (fetch)
```

**Solution (v2.1)**: Treat as owner `my-company`
- All my-company repos share knowledge
- 0.9.0-rc-1: Upgrade to org features with RBAC

### 5. Changed Git Remote

```bash
# User changes remote URL
git remote set-url origin git@gitlab.com:newuser/repo.git
```

**Solution**: Re-detect on each contextd startup
- Owner may change → knowledge scope changes
- Old owner's knowledge still accessible (database persists)
- Could provide migration prompt

---

## Security Model (v2.1)

### Threat Model

**In Scope:**
- ✅ Cross-project confusion (FIXED)
- ✅ Unintended knowledge leakage between unrelated repos (FIXED)
- ✅ Single-user data isolation

**Out of Scope (Future):**
- ❌ Multi-user access control (0.9.0-rc-1)
- ❌ Organization RBAC (0.9.0-rc-1)
- ❌ Team permissions (0.9.0-rc-1)
- ❌ OAuth/SSO (0.9.0-rc-1)

### Guarantees

**v2.1 Provides:**
1. **Owner Isolation**: Repos with different owners NEVER share knowledge
2. **Project Privacy**: Checkpoints NEVER leave project scope
3. **Configurable Sharing**: User can disable owner-level sharing
4. **Deterministic**: Same git remote = same owner = same scope

**v2.1 Does NOT Provide:**
1. Multi-user authentication (still single bearer token)
2. Granular ACLs (all or nothing at owner level)
3. Encryption at rest (planned for v2.2)

---

## Future Evolution Path

### v2.1 (This ADR) - Immediate

- ✅ Owner-scoped databases
- ✅ Git-based repository detection
- ✅ Configurable sharing preferences
- ✅ Migration from v2.0

### v2.2 - Short Term (3 months)

- [ ] Encryption at rest
- [ ] Public knowledge opt-in
- [ ] Enhanced search ranking by scope
- [ ] CLI improvements

### 0.9.0-rc-1 - Long Term (6+ months)

- [ ] Organization databases (when demand exists)
- [ ] OAuth/SSO integration
- [ ] RBAC (owner, admin, developer, viewer)
- [ ] Team permissions
- [ ] Multi-user deployment
- [ ] Audit logging

---

## Success Criteria

**v2.1 Release Gates:**

1. **Functionality**:
   - [ ] Automatic owner detection from git remote
   - [ ] Owner-scoped databases working
   - [ ] Search checks project → owner hierarchy
   - [ ] Configuration respected

2. **User Experience**:
   - [ ] Zero configuration for git repos
   - [ ] Clear search result context
   - [ ] No cross-owner confusion
   - [ ] Clean migration path

3. **Performance**:
   - [ ] Search < 100ms (p95)
   - [ ] Database creation < 1s
   - [ ] Migration < 5min for 1000 remediations

4. **Testing**:
   - [ ] >80% code coverage
   - [ ] Integration tests for all git hosts
   - [ ] Migration tests (v2.0 → v2.1)
   - [ ] Edge case tests

---

## Alternatives Considered

### Alternative 1: Full Org/Team/RBAC (Original Plan)

**Pros**: Complete solution for enterprises
**Cons**: 6+ months, OAuth complexity, 90% of users don't need it
**Decision**: Defer to 0.9.0-rc-1

### Alternative 2: Keep "shared" Database, Add Filters

**Pros**: No migration needed
**Cons**: Doesn't fix UX issue, still confusing results, filter complexity
**Decision**: Rejected - doesn't solve root cause

### Alternative 3: Project-Only (No Sharing)

**Pros**: Maximum isolation, simplest implementation
**Cons**: Loses value of shared knowledge, users want cross-repo learning
**Decision**: Rejected - users need cross-repo knowledge

### Alternative 4: Manual Owner Configuration

**Pros**: User has full control
**Cons**: Poor UX, users won't configure it, defeats purpose of automation
**Decision**: Rejected - auto-detection is better UX

---

## Conclusion

**Implement owner-scoped isolation in v2.1:**

- Solves immediate problem (cross-session confusion)
- Simple implementation (weeks, not months)
- Serves 90% of users (single developers)
- Foundation for future team features (0.9.0-rc-1)
- No authentication complexity
- Backward compatible migration

**Defer to 0.9.0-rc-1:**
- Organization databases
- RBAC
- OAuth/SSO
- Multi-user deployment

**Next Steps:**
1. Approve this ADR
2. Create v2.1 milestone
3. Begin Phase 1 implementation (repository detection)
4. Target release: 4 weeks
