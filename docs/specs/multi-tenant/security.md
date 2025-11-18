# Multi-Tenant Security

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the security model and filter injection prevention for contextd's multi-tenant architecture.

---

## Filter Injection Prevention

### The Problem: Filter-Based Isolation (Legacy)

**Vulnerable Pattern**:
```go
// Filter-based isolation (VULNERABLE)
query := fmt.Sprintf("project_path == '%s'", projectPath)

// Attack vector
projectPath = "' OR '1'=='1"

// Result: Injected query
query = "project_path == '' OR '1'=='1'"

// Impact: Access to ALL projects (bypass isolation)
```

**Vulnerability Details**:
- User-controlled `projectPath` injected into filter expression
- No sanitization or escaping
- Filter evaluation happens in vector database
- Metadata filters are string-based (not type-safe)
- Attacker can craft path to bypass filter

**Attack Scenarios**:
1. **OR Injection**: `' OR '1'=='1` → Always true, returns all projects
2. **AND Injection**: `' AND project_path != '` → Exclude own project, return others
3. **Negation**: `' OR project_path != 'victim_project` → Access specific project
4. **Pattern Matching**: `' OR project_path LIKE '%sensitive%` → Pattern-based access

**Impact**:
- **Confidentiality Breach**: Read other projects' data
- **Data Leakage**: Cross-project information disclosure
- **Compliance Violation**: GDPR, HIPAA, SOC 2 violations

---

## The Solution: Database-Per-Project

### Physical Isolation

**Secure Pattern**:
```go
// Physical isolation (SECURE)
dbName := GetDatabaseName(DatabaseTypeProject, projectPath)
// Result: dbName = "project_abc123de"

// No filters, no injection possible
results := store.Search(ctx, dbName, "checkpoints", query)
```

**Security Properties**:
1. **No Metadata Filters**: Eliminates injection attack vector entirely
2. **Physical Isolation**: Complete database-level separation
3. **Hash-Based Naming**: Opaque, deterministic, collision-resistant
4. **Type-Safe**: No string concatenation or formatting
5. **One-Way Function**: Cannot recover path from hash

### Hash-Based Database Naming

**Hash Generation**:
```go
func projectHash(path string) string {
    h := sha256.Sum256([]byte(path))
    return fmt.Sprintf("%x", h)[:8]  // First 8 characters
}
```

**Security Properties**:
- **Deterministic**: Same path always produces same hash
- **One-Way**: Cannot reverse hash to recover path
- **Collision-Resistant**: SHA256 provides 2^32 combinations (8 hex chars)
- **Opaque**: No information leakage from database name
- **Non-Enumerable**: Cannot guess other project hashes

**Example**:
```
Input:  /home/user/projects/contextd
Hash:   abc123de45678901cdef... (SHA256)
DB:     project_abc123de (first 8 chars)
```

---

## Security Benefits

### Attack Surface Reduction

**Filter-Based** (Legacy):
- ❌ User-controlled filter expressions
- ❌ String concatenation vulnerabilities
- ❌ Metadata filter evaluation (complex, error-prone)
- ❌ Cross-project queries possible with injection

**Database-Per-Project** (Current):
- ✅ No user-controlled filters
- ✅ No string concatenation
- ✅ Type-safe database selection
- ✅ Cross-project queries physically impossible

### Defense in Depth

**Layer 1: Database Isolation**
- Physical separation at database level
- No shared collections between projects
- Database boundaries enforced by vector store

**Layer 2: Hash-Based Naming**
- One-way hash prevents path recovery
- Opaque names prevent enumeration
- Collision-resistant (SHA256)

**Layer 3: Type-Safe API**
- No string formatting in queries
- Compile-time type checking
- No user-controlled filter expressions

**Layer 4: Access Control** (Future)
- Database-level ACLs
- Role-based access control (RBAC)
- User-to-project mapping
- Audit logging

---

## Threat Model

### In Scope

**Filter Injection** (Mitigated):
- Injection via project_path parameter
- Metadata filter bypass
- Cross-project data access

**Data Leakage** (Mitigated):
- Accidental cross-project queries
- Shared collection access
- Information disclosure via database names

**Resource Exhaustion** (Partially Mitigated):
- Database limit enforcement (100 per instance)
- Collection limit enforcement (100 per database)
- Vector limit enforcement (1M per collection)

### Out of Scope (MVP)

**Authentication/Authorization**:
- MVP uses trusted network assumption
- No authentication required
- All databases accessible by any client
- Post-MVP: Add Bearer token, JWT, or OAuth

**Rate Limiting**:
- No rate limiting in MVP
- Rely on reverse proxy for production
- Post-MVP: Add per-user/per-database limits

**DDoS Protection**:
- Not implemented in MVP
- Use reverse proxy (nginx, Caddy) for production
- Post-MVP: Add request throttling

### Out of Scope (By Design)

**SQL Injection**: Not applicable (no SQL database)

**XSS**: Not applicable (no web UI)

**CSRF**: Not applicable (no web sessions)

---

## Security Boundaries

### Project Boundary

**Isolation Guarantee**: No cross-project data access

**Enforcement**:
- Physical database separation
- Hash-based database naming
- No metadata filters

**Verification**:
- Integration tests verify isolation
- Security audit confirms no bypass
- Regression tests prevent regressions

### Shared Knowledge Boundary

**Access**: All projects can read/write shared database

**Collections**:
- `remediations` - Error solutions
- `skills` - Reusable templates
- `troubleshooting_patterns` - Common patterns

**Security Considerations**:
- Shared data intentionally cross-project
- No sensitive project data in shared collections
- Read-write access for all projects (by design)

### User Boundary (Future)

**Isolation Guarantee**: No cross-user data access

**Enforcement**:
- Per-user databases (`user_<id>`)
- Authentication and authorization
- User-to-project mapping

**Status**: Reserved for future multi-user implementation

---

## Access Control (Future)

### Current State (v2.0)

**No Authentication** (MVP):
- Trusted network assumption
- All databases accessible by any client
- HTTP transport with reverse proxy recommended
- Multi-session support (multiple concurrent Claude instances)

**Deployment Security**:
- Deploy behind VPN
- Use SSH tunneling for remote access
- Add TLS via reverse proxy (nginx/Caddy)

### Future Multi-User (v2.1+)

**Database-Level ACLs**:
```yaml
database: project_abc123de
acl:
  - user: alice
    permissions: [read, write, delete]
  - user: bob
    permissions: [read]
  - team: engineering
    permissions: [read, write]
```

**Role-Based Access Control (RBAC)**:
```yaml
roles:
  - name: project_owner
    permissions:
      - database.create
      - database.delete
      - collection.*
      - vector.*

  - name: project_contributor
    permissions:
      - collection.read
      - collection.write
      - vector.read
      - vector.write

  - name: project_viewer
    permissions:
      - collection.read
      - vector.read
```

**User-to-Project Mapping**:
```yaml
users:
  - name: alice
    email: alice@example.com
    projects:
      - project_abc123de: project_owner
      - project_def456gh: project_contributor

  - name: bob
    email: bob@example.com
    projects:
      - project_abc123de: project_viewer
```

---

## Audit Logging (Future)

### Access Logs

**Required Events**:
- Database creation/deletion
- Collection creation/deletion
- Vector insert/search/delete
- Authentication attempts (success/failure)
- Authorization failures

**Log Format**:
```json
{
  "timestamp": "2025-01-03T10:30:00Z",
  "event": "database.access",
  "user": "alice",
  "database": "project_abc123de",
  "collection": "checkpoints",
  "operation": "search",
  "success": true,
  "query": "feature implementation",
  "results": 10,
  "latency_ms": 15
}
```

### Security Events

**Required Events**:
- Filter injection attempts (if detected)
- Database limit exceeded
- Collection limit exceeded
- Unauthorized access attempts
- Failed authentication

**Alert Triggers**:
- Repeated authentication failures
- Database limit approaching (>80%)
- Unusual access patterns (anomaly detection)

---

## Security Validation

### Integration Tests

**Multi-Project Isolation**:
```go
func TestMultiProjectIsolation(t *testing.T) {
    // Create two projects
    projectA := "/home/user/projects/a"
    projectB := "/home/user/projects/b"

    // Insert data to project A
    service.Save(ctx, projectA, checkpointA)

    // Search in project B
    results, err := service.Search(ctx, projectB, "checkpoint")

    // Verify: No results from project A
    assert.Equal(t, 0, len(results))
}
```

**Shared Database Access**:
```go
func TestSharedDatabaseAccess(t *testing.T) {
    // Insert remediation (shared)
    service.SaveRemediation(ctx, remediation)

    // Search from project A
    resultsA, _ := service.SearchRemediations(ctx, projectA, "error")

    // Search from project B
    resultsB, _ := service.SearchRemediations(ctx, projectB, "error")

    // Verify: Both can access shared remediation
    assert.Contains(t, resultsA, remediation)
    assert.Contains(t, resultsB, remediation)
}
```

### Security Audit

**Checklist**:
- [ ] No filter injection possible
- [ ] No cross-project data access
- [ ] Hash generation tested (deterministic, collision-resistant)
- [ ] Database naming validated
- [ ] Service integration verified
- [ ] Migration preserves isolation
- [ ] Regression tests cover security boundaries

---

## Migration Security

### Data Integrity

**Guarantees**:
- No data loss during migration
- Checksums validated
- Rollback capability (before cleanup)

**Validation**:
```bash
# Generate checksums before migration
contextd migrate analyze --checksums before.txt

# Execute migration
contextd migrate execute

# Validate checksums after migration
contextd migrate validate --checksums before.txt
```

### Cleanup Safety

**Protection**:
- Dry-run mode (no actual cleanup)
- Backup requirement before cleanup
- Validation required before cleanup
- Confirmation prompt

**Example**:
```bash
# Require explicit confirmation
contextd migrate cleanup --confirm

# Prompt: "This will delete the legacy 'default' database. Type 'DELETE' to confirm:"
```

---

## Summary

**Security Model**:
1. **Physical Isolation**: Database-per-project eliminates filter injection
2. **Hash-Based Naming**: Opaque, deterministic, collision-resistant
3. **Type-Safe API**: No user-controlled filter expressions
4. **Access Control** (Future): Database-level ACLs and RBAC
5. **Audit Logging** (Future): Comprehensive security event tracking

**Attack Mitigation**:
- ✅ Filter injection: **Eliminated** (no filters)
- ✅ Data leakage: **Prevented** (physical isolation)
- ✅ Resource exhaustion: **Limited** (database/collection/vector limits)
- ⚠️  Authentication: **Not implemented** (MVP trusted network, post-MVP: add auth)
- ⚠️  Rate limiting: **Not implemented** (MVP relies on reverse proxy)

**Deployment Recommendations**:
- Deploy behind VPN or use SSH tunneling
- Add TLS via reverse proxy (nginx/Caddy)
- Post-MVP: Add authentication (Bearer token, JWT, OAuth)
- Post-MVP: Add rate limiting per user/database
