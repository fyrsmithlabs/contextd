# Vector Database Restructure Proposal

## Current Problem

Currently, all data is stored in a single flat namespace with collections:
- `checkpoints` - Mixed: all projects in one collection
- `remediations` - Mixed: all projects/users
- `research_documents` - Mixed: all projects
- `session_notes` - Mixed: all sessions
- `skills` - Mixed: all users
- `troubleshooting_patterns` - Mixed: all projects

**Issues**:
1. ❌ No data isolation between projects
2. ❌ Can't easily delete all data for a project
3. ❌ Shared knowledge (troubleshooting) mixed with project-specific data
4. ❌ Filtering required for every query (`project == "x"`)
5. ❌ No multi-user support
6. ❌ Backup/restore is all-or-nothing

## Proposed Architecture

### Multi-Database Structure

```
contextd/
├── shared/                    # Global/shared database
│   ├── remediations          # Error solutions (cross-project)
│   ├── troubleshooting       # Troubleshooting patterns
│   └── skills                # Reusable skills/templates
│
├── project:<hash>/            # Per-project databases
│   ├── checkpoints           # Project-specific checkpoints
│   ├── research              # Project-specific research
│   └── notes                 # Project-specific notes
│
└── user:<username>/           # Per-user databases (future)
    ├── personal_notes        # User's personal notes
    └── preferences           # User preferences
```

### Database Naming Convention

**Qdrant**: Uses collection names (no native database concept)
```
shared_remediations
shared_troubleshooting
shared_skills

project_abc123_checkpoints        # Hash of /home/user/contextd
project_abc123_research
project_abc123_notes

user_john_personal_notes          # Future
user_john_preferences
```

```
Database: shared
  ├── remediations
  ├── troubleshooting
  └── skills

Database: project_abc123
  ├── checkpoints
  ├── research
  └── notes

Database: user_john              # Future
  ├── personal_notes
  └── preferences
```

## Detailed Design

### 1. Shared Database

**Purpose**: Cross-project knowledge that benefits all projects

**Collections**:

#### remediations
```yaml
Scope: Global
Purpose: Error solutions usable across all projects
Fields:
  - error_message (text)
  - solution (text)
  - stack_trace (text)
  - tags (keyword[])
  - language (keyword)        # NEW: programming language
  - framework (keyword)       # NEW: framework if applicable
  - severity (keyword)        # NEW: error severity
  - usage_count (integer)     # NEW: how often used
  - success_rate (float)      # NEW: solution success rate
  - created_at (timestamp)
  - updated_at (timestamp)
```

**Rationale**: Error solutions are universally applicable. A "connection refused" fix works across projects.

#### troubleshooting_patterns
```yaml
Scope: Global
Purpose: Common troubleshooting workflows
Fields:
  - pattern_name (text)
  - description (text)
  - diagnostic_steps (text[])
  - solutions (text[])
  - tags (keyword[])
  - category (keyword)        # network, filesystem, database, etc.
  - success_rate (float)
  - usage_count (integer)
  - created_at (timestamp)
```

**Rationale**: Troubleshooting patterns apply universally.

#### skills
```yaml
Scope: Global
Purpose: Reusable code/workflow templates
Fields:
  - name (text)
  - description (text)
  - content (text)
  - version (keyword)
  - author (keyword)
  - category (keyword)
  - prerequisites (text[])
  - expected_outcome (text)
  - tags (keyword[])
  - usage_count (integer)
  - success_rate (float)
  - created_at (timestamp)
```

**Rationale**: Skills are meant to be reusable across projects.

### 2. Project Database

**Purpose**: Project-specific knowledge and history

**Naming**: `project_<hash>` where hash = first 8 chars of SHA256(project_path)

**Example**:
```bash
/home/user/projects/contextd
→ SHA256(...) = abc123def456...
→ Database: project_abc123de
→ Collections:
   - project_abc123de_checkpoints
   - project_abc123de_research
   - project_abc123de_notes
```

**Collections**:

#### checkpoints
```yaml
Scope: Project-specific
Purpose: Session state and work history
Fields:
  - id (uuid)
  - summary (text)
  - content (text)
  - embedding (vector[384])
  - project_path (keyword)    # Full path for reference
  - timestamp (integer)
  - token_count (integer)
  - tags (keyword[])
  - git_branch (keyword)      # NEW: active branch
  - git_commit (keyword)      # NEW: commit hash
  - files_changed (keyword[]) # NEW: files in checkpoint
```

**Rationale**: Checkpoints are tightly coupled to a project's history.

#### research
```yaml
Scope: Project-specific
Purpose: Project-specific documentation and research
Fields:
  - id (uuid)
  - title (text)
  - document_section (text)
  - embedding (vector[384])
  - category (keyword)
  - key_findings (text)
  - recommendations (text)
  - source_url (keyword)      # NEW: where doc came from
  - project_path (keyword)
  - tags (keyword[])
  - date_added (timestamp)
```

**Rationale**: Research is specific to understanding a particular codebase.

#### notes
```yaml
Scope: Project-specific
Purpose: Session notes and observations
Fields:
  - id (uuid)
  - session_id (keyword)
  - note_type (keyword)       # observation, decision, todo, etc.
  - title (text)
  - content (text)
  - embedding (vector[384])
  - metadata (json)
  - tags (keyword[])
  - project_path (keyword)
  - timestamp (integer)
```

**Rationale**: Notes are session/project-specific.

### 3. User Database (Future Enhancement)

**Purpose**: Per-user preferences and private data

**Collections**:
- `personal_notes` - User's private notes
- `preferences` - UI preferences, shortcuts, etc.
- `workspaces` - Saved workspaces/layouts

**Rationale**: Enable multi-user support without data mixing.

## Implementation Strategy

### Phase 1: Core Restructure (This PR)

1. **Update Configuration**
   ```go
   type VectorDBConfig struct {
       ProjectHashFn  func(string) string
   }
   ```

2. **Update VectorStore Interface**
   ```go
   type VectorStore interface {
       // Database management
       CreateDatabase(ctx context.Context, name string) error
       ListDatabases(ctx context.Context) ([]string, error)
       DeleteDatabase(ctx context.Context, name string) error

       // Collection operations (existing)
       InsertCheckpoints(ctx context.Context, db string, data []CheckpointData) error
       SearchCheckpoints(ctx context.Context, db string, embedding []float32, ...) error
       // ... etc
   }
   ```

3. **Add Database Selection Logic**
   ```go
   func GetDatabaseName(scope string, projectPath string) string {
       switch scope {
       case "shared":
           return "shared"
       case "project":
           hash := projectHash(projectPath)
           return fmt.Sprintf("project_%s", hash[:8])
       case "user":
           return fmt.Sprintf("user_%s", username)
       default:
           return "shared"
       }
   }
   ```

4. **Update All Services**
   - Checkpoint service: Use project database
   - Remediation service: Use shared database
   - Skills service: Use shared database
   - Research service: Use project database

### Phase 2: Migration Script

Create `ctxd migrate-structure` command:

```bash
# Migrate existing flat structure to new multi-DB
ctxd migrate-structure --from flat --to multi-db --dry-run

# Actual migration
ctxd migrate-structure --from flat --to multi-db
```

**Process**:
1. Read all existing data
2. Group by project_path
3. Create project databases
4. Move checkpoints/research/notes to project DBs
5. Move remediations/skills/troubleshooting to shared DB
6. Verify data integrity
7. Create backup before and after

### Phase 3: Backward Compatibility

Support legacy flat structure with config flag:
```bash
export VECTOR_DB_STRUCTURE=flat    # Old behavior
export VECTOR_DB_STRUCTURE=multi   # New behavior (default)
```

## Benefits

### 1. Data Isolation

✅ **Project deletion is clean**
```bash
# Delete all data for a project
ctxd project delete /path/to/project
# Deletes entire project_abc123de database
```

✅ **No cross-project leakage**
- Searching in project A never returns project B's checkpoints
- No need for project filters in every query

### 2. Performance

✅ **Smaller collections = faster queries**
```
Before: Search 10,000 checkpoints across all projects (with filter)
After:  Search 500 checkpoints in one project (no filter)
```

✅ **Better caching**
- Project-specific indexes cached separately
- No cache pollution from other projects

✅ **Optimized per project**
- Can tune HNSW parameters per project
- Different quantization settings if needed

### 3. Multi-Tenancy Ready

✅ **User isolation** (future)
```
user_alice/ - Alice's data
user_bob/   - Bob's data
shared/     - Common knowledge
```

✅ **Access control**
- Grant user access to specific project DBs
- Shared DB is read-only for most users
- Admin can manage all DBs

### 4. Backup/Restore Granularity

✅ **Per-project backups**
```bash
# Backup just one project
ctxd backup --project /path/to/contextd

# Backup shared knowledge
ctxd backup --scope shared

# Backup everything
ctxd backup --all
```

✅ **Selective restore**
```bash
# Restore just one project
ctxd restore --project /path/to/contextd backup.json

# Restore shared knowledge to new instance
ctxd restore --scope shared shared-backup.json
```

### 5. Knowledge Sharing

✅ **Curated shared knowledge**
- Best remediations promoted to shared DB
- Verified troubleshooting patterns
- Community-contributed skills

✅ **Project-specific privacy**
- Proprietary code stays in project DB
- Sensitive data never in shared DB

## Migration Example

### Before (Flat Structure)
```
Collections:
├── checkpoints (569 vectors)
│   ├── project=/home/user/contextd (400)
│   ├── project=/home/user/other (150)
│   └── project=/home/user/test (19)
├── remediations (2 vectors)
└── skills (0 vectors)
```

### After (Multi-DB Structure)
```
Databases:
├── shared
│   ├── remediations (2 vectors)
│   └── skills (0 vectors)
├── project_abc123de (/home/user/contextd)
│   └── checkpoints (400 vectors)
├── project_def456ab (/home/user/other)
│   └── checkpoints (150 vectors)
└── project_789xyz12 (/home/user/test)
    └── checkpoints (19 vectors)
```

## Naming Alternatives

### For Shared Database

| Option | Pros | Cons |
|--------|------|------|
| `shared` | Clear intent | Generic |
| `global` | Common term | Can imply too broad |
| `common` | Descriptive | Less clear |
| `knowledge` | Semantic | Only fits some data |
| `universal` | Accurate | Verbose |

**Recommendation**: `shared` - Clear, concise, widely understood

### For Project Databases

| Option | Example | Pros | Cons |
|--------|---------|------|------|
| `project_<hash>` | `project_abc123de` | Collision-free | Not human-readable |
| `proj_<name>` | `proj_contextd` | Readable | Name collisions |
| `<hash>` | `abc123de` | Compact | No context |
| `workspace_<hash>` | `workspace_abc123de` | Descriptive | Verbose |

**Recommendation**: `project_<hash>` - Balance of clarity and uniqueness

## Database Support Matrix

### Qdrant

**Collections as Databases**:
```
✅ Collections: Yes (unlimited)
❌ Databases: No (use collection prefixes)
✅ Isolation: Collection-level
✅ Access control: Per-collection (Qdrant Cloud)
```

**Implementation**:
```go
// Qdrant uses collection name prefixes
func (c *QdrantClient) GetCollectionName(db, collection string) string {
    return fmt.Sprintf("%s_%s", db, collection)
}

// Example
shared_remediations
project_abc123_checkpoints
```


**Native Database Support** (2.4+):
```
✅ Collections: Yes (per database)
✅ Databases: Yes (native)
✅ Isolation: Database-level
✅ Access control: Per-database RBAC
```

**Implementation**:
```go
    return c.client.UsingDatabase(db)
}

// Example
Database: shared
  └── remediations
Database: project_abc123
  └── checkpoints
```

## API Changes

### Current API
```go
// No database concept
svc.checkpoint.Save(ctx, checkpoint)
svc.remediation.Search(ctx, query)
```

### Proposed API (Option 1: Explicit)
```go
// Explicit database parameter
svc.checkpoint.Save(ctx, "project_abc123", checkpoint)
svc.remediation.Search(ctx, "shared", query)
```

### Proposed API (Option 2: Context)
```go
// Database in context
ctx = vectorstore.WithDatabase(ctx, "project_abc123")
svc.checkpoint.Save(ctx, checkpoint)

ctx = vectorstore.WithDatabase(ctx, "shared")
svc.remediation.Search(ctx, query)
```

### Proposed API (Option 3: Service Factory)
```go
// Create service for specific database
projectSvc := checkpoint.NewService(store, "project_abc123")
projectSvc.Save(ctx, checkpoint)

sharedSvc := remediation.NewService(store, "shared")
sharedSvc.Search(ctx, query)
```

**Recommendation**: **Option 3 (Service Factory)**
- ✅ Type-safe
- ✅ Clear separation
- ✅ Easier testing
- ✅ No accidental cross-DB queries

## Configuration

### Environment Variables
```bash
# Enable multi-database structure
export VECTOR_DB_STRUCTURE=multi  # or "flat" for legacy

# Override shared database name
export VECTOR_DB_SHARED_NAME=shared

# Project hash algorithm
export VECTOR_DB_PROJECT_HASH=sha256  # or "md5"
```

### Config File
```yaml
vector_db:
  structure: multi  # or flat
  shared_database: shared
  project_hash:
    algorithm: sha256
    length: 8  # chars to use

  # Per-project overrides
  project_databases:
    "/home/user/special-project":
      database: custom_name
      collections:
        checkpoints:
          hnsw_m: 32  # Custom HNSW settings
```

## Rollout Plan

### Week 1: Design & Review
- ✅ Create proposal (this document)
- ⏳ Team review and feedback
- ⏳ Finalize design

### Week 2: Implementation
- Update VectorStore interface
- Implement database selection logic
- Update Qdrant client (collection prefixes)
- Add project hash function

### Week 3: Service Updates
- Update checkpoint service
- Update remediation service
- Update skills service
- Update research service
- Update session notes service

### Week 4: Migration & Testing
- Create migration script
- Test with sample data
- Performance benchmarks
- Documentation

### Week 5: Rollout
- Feature flag (default off)
- Gradual rollout
- Monitor performance
- Collect feedback

## Risks & Mitigations

### Risk 1: Data Migration Complexity
**Risk**: Migrating 569+ checkpoints could fail
**Mitigation**:
- Mandatory backup before migration
- Dry-run mode
- Rollback script
- Incremental migration (batch by project)

### Risk 2: Performance Regression
**Risk**: More databases = more overhead
**Mitigation**:
- Benchmark before/after
- Connection pooling per database
- Cache database metadata
- Lazy database creation

### Risk 3: Breaking Changes
**Risk**: Existing code expects flat structure
**Mitigation**:
- Feature flag for gradual rollout
- Backward compatibility mode
- Migration guide
- API versioning

### Risk 4: Database Limits
**Risk**: Too many databases/collections
**Mitigation**:
- Qdrant: No practical limit on collections
- Monitor and alert on limits
- Archival strategy for old projects

## Success Metrics

### Performance
- ✅ Search latency: <50ms (down from ~100ms)
- ✅ Insert throughput: 2x improvement
- ✅ Memory usage: 30% reduction per query

### Usability
- ✅ Clean project deletion in <1s
- ✅ No accidental cross-project queries
- ✅ Backup/restore per project

### Scalability
- ✅ Support 100+ concurrent projects
- ✅ 10,000+ checkpoints per project
- ✅ Shared DB with 10,000+ remediations

## Open Questions

1. **Should we migrate automatically or require manual migration?**
   - Auto: Easier for users, riskier
   - Manual: User control, requires action

2. **How to handle projects with same name in different directories?**
   - Hash prevents collisions, but UI display?
   - Store full path mapping?

3. **Should shared DB be read-only for project-level code?**
   - Prevents pollution
   - Requires admin workflow for contributions

4. **Database cleanup policy for deleted projects?**
   - Auto-delete after N days?
   - Archive to cold storage?
   - Keep forever?

5. **How to handle project renames/moves?**
   - Re-hash and migrate?
   - Keep old hash and add alias?
   - Manual intervention?

## Next Steps

1. **Review this proposal** - Get team/user feedback
2. **Create POC** - Prove concept with small dataset
3. **Performance test** - Benchmark multi-DB vs flat
4. **Update specs** - Detailed API specifications
5. **Implement** - Follow rollout plan above

## Feedback Welcome

Please provide feedback on:
- [ ] Overall architecture direction
- [ ] Naming conventions (shared, project_<hash>)
- [ ] API design (service factory vs context vs explicit)
- [ ] Migration strategy (auto vs manual)
- [ ] Rollout timeline
- [ ] Missing use cases or concerns

---

**Status**: 🔍 Proposal (Awaiting Feedback)
**Author**: Claude Code (with user guidance)
**Date**: 2025-01-01
**Related**: #54 (Qdrant Implementation)
