# Multi-Tenant Requirements

**Parent**: [../SPEC.md](../SPEC.md)

This document defines the requirements for contextd's multi-tenant architecture.

---

## Design Goals

1. **Security-First**: Physical isolation eliminates filter injection attacks
2. **High Performance**: 10-16x faster queries via partition pruning
3. **Scalability**: Support 100+ projects per user instance
4. **Clean Migration**: Smooth transition from legacy flat structure

---

## Functional Requirements

### FR1: Project Isolation

**Requirement**: Each project MUST have physically isolated data storage.

**Rationale**:
- Prevents filter injection vulnerabilities
- Eliminates cross-project data leakage
- Enables per-project access control

**Implementation**: Database-per-project strategy with unique database names

### FR2: Shared Knowledge

**Requirement**: Global knowledge (remediations, skills, troubleshooting patterns) MUST be accessible across all projects.

**Rationale**:
- Promotes knowledge reuse
- Reduces redundant storage
- Enables collaborative learning

**Implementation**: Shared database for cross-project collections

### FR3: Deterministic Naming

**Requirement**: Project databases MUST have deterministic, collision-resistant names.

**Rationale**:
- Ensures same project always maps to same database
- Prevents accidental collisions
- Enables offline hash computation

**Implementation**: SHA256 hash of project path (first 8 characters)

### FR4: Migration Support

**Requirement**: System MUST support migration from legacy flat structure to multi-tenant architecture.

**Rationale**:
- Existing users have data in legacy format
- Migration must be safe and verifiable
- No data loss acceptable

**Implementation**: Migration tools with analyze, execute, validate workflow

---

## Non-Functional Requirements

### NFR1: Performance

**Requirement**: Query performance MUST be 10x+ faster than filter-based approach.

**Target Metrics**:
- Search latency: <50ms (vs 500ms+ with filters)
- Memory usage: 10-100x less (load only project DB)
- Cache hit rate: >80% for project data

**Measurement**: Benchmark tests comparing filter-based vs database-per-project

### NFR2: Scalability

**Requirement**: System MUST support 100+ projects per user instance.

**Limits**:
- Maximum databases: 100 per instance (configurable)
- Maximum collections per database: 100
- Maximum vectors per collection: 1,000,000

**Rationale**: Expected load is 10-100 projects per user (well within limits)

### NFR3: Security

**Requirement**: System MUST eliminate filter injection attack vector.

**Attack Prevention**:
- No user-controlled metadata filters
- Physical database isolation
- Opaque database naming (hash-based)

**Validation**: Security audit confirms no filter injection possible

### NFR4: Backward Compatibility

**Requirement**: v1.x data MUST be migrated without data loss.

**Migration Guarantees**:
- All vectors migrated
- Checksums validated
- Rollback capability (before cleanup)

**Validation**: Migration tests verify data integrity

---

## Database Requirements

### Database Types

**Shared Database** (`shared`):
- Fixed name for global knowledge
- Contains: remediations, skills, troubleshooting_patterns
- Accessible by all projects

**Project Database** (`project_<hash>`):
- Unique per project based on path hash
- Contains: checkpoints, research, notes
- Isolated from other projects

**User Database** (`user_<id>`, future):
- Per-user data isolation
- Reserved for future multi-user implementation

### Database Naming Convention

**Format**: `<type>_<identifier>`

**Examples**:
- `shared` - Global knowledge database
- `project_abc123de` - Project with hash abc123de
- `user_john` - User john (future)

**Validation Rules**:
- Type prefix required (shared, project, user)
- Identifier must be alphanumeric (hash for project)
- Total length ≤64 characters

---

## Collection Requirements

### Collection Schema

**Required Fields**:
- Name: Collection identifier
- VectorDim: Embedding dimension (1536 for OpenAI)
- DistanceMetric: cosine, l2, or ip
- Fields: Field type definitions
- Indexed: Fields to index for filtering

**Supported Field Types**:
- string, int64, float32, bool, json, array

### Distance Metrics

**Cosine** (Recommended):
- Angle-based similarity
- Normalized vectors
- Range: [-1, 1]

**L2** (Euclidean):
- Distance-based similarity
- Unnormalized vectors
- Range: [0, ∞)

**IP** (Inner Product):
- Dot product similarity
- Unnormalized vectors
- Range: (-∞, ∞)

---

## Vector Operations Requirements

### Insert

**Requirement**: Insert vectors to project-specific or shared collections.

**Parameters**:
- Database name (project or shared)
- Collection name
- Vectors with ID, embedding, payload

**Guarantees**:
- Atomic insertion (all or nothing)
- Duplicate ID handling (upsert or error)

### Search

**Requirement**: Search within project database or shared database.

**Parameters**:
- Database name
- Collection name
- Query vector
- TopK (result limit)
- Filter (optional, within database only)

**Guarantees**:
- No cross-database search
- Results sorted by score
- Filter applied within database (no project filter needed)

### Delete

**Requirement**: Delete vectors from database.

**Parameters**:
- Database name
- Collection name
- Filter or IDs

**Guarantees**:
- Only affects specified database
- No cascading deletes

---

## Migration Requirements

### Migration Phases

**Phase 1: Analysis**
- Identify unique project paths
- Count vectors per project
- Estimate migration time
- Generate project distribution report

**Phase 2: Database Creation**
- Create shared database
- Create project databases (one per unique path)
- Verify database creation

**Phase 3: Data Migration**
- Copy vectors from default DB to project DBs
- Maintain checksums for validation
- Support dry-run mode

**Phase 4: Validation**
- Verify all vectors migrated
- Compare checksums
- Check data integrity

**Phase 5: Cleanup**
- Delete legacy default database
- Cleanup temporary migration data

### Migration Tools

**Required Commands**:
```bash
contextd migrate analyze        # Analyze current structure
contextd migrate plan           # Generate migration plan
contextd migrate execute        # Execute migration
contextd migrate validate       # Validate results
contextd migrate cleanup        # Remove legacy data
```

**Safety Features**:
- Dry-run mode (no actual changes)
- Backup requirement before migration
- Rollback capability (before cleanup)
- Validation checksums

---

## Error Handling Requirements

### Database Errors

**Required Error Types**:
- `ErrDatabaseNotFound` - Database does not exist
- `ErrDatabaseAlreadyExists` - Database creation conflict
- `ErrDatabaseLimit` - Maximum databases reached
- `ErrInvalidDatabaseName` - Name violates conventions

**Error Handling**:
- Clear error messages with context
- No sensitive data in errors
- Proper error wrapping with `%w`

### Collection Errors

**Required Error Types**:
- `ErrCollectionNotFound` - Collection does not exist
- `ErrCollectionAlreadyExists` - Collection creation conflict
- `ErrInvalidCollectionName` - Name violates conventions

### Migration Errors

**Required Error Types**:
- `ErrMigrationInProgress` - Migration already running
- `ErrMigrationFailed` - Migration failed
- `ErrRollbackRequired` - Rollback needed
- `ErrDataLoss` - Potential data loss detected

**Error Handling**:
- Automatic rollback on failure (before cleanup)
- Data loss prevention (validation before cleanup)
- Clear rollback instructions

---

## Testing Requirements

### Unit Test Coverage

**Required Coverage**:
- Overall: ≥80%
- Core functions: 100%
- Database operations: 100%
- Hash generation: 100%
- Naming validation: 100%

### Integration Tests

**Required Scenarios**:
1. Multi-project isolation (verify no cross-access)
2. Shared database access (verify cross-project read)
3. Migration flow (verify data integrity)
4. Performance comparison (verify 10x+ improvement)
5. Database limits (verify scalability)

### Performance Tests

**Required Benchmarks**:
- Search latency (project database)
- Search latency (shared database)
- Comparison: filter-based vs database-per-project
- Memory usage comparison
- Cache hit rate measurement

---

## Monitoring Requirements

### Metrics

**Database Metrics**:
- `contextd_databases_total{type}` - Count by type
- `contextd_database_operations_total{operation}` - Operation count
- `contextd_database_operations_duration_seconds{operation}` - Latency

**Collection Metrics**:
- `contextd_collections_total{database,type}` - Collection count
- `contextd_collection_operations_total{operation}` - Operation count
- `contextd_collection_operations_duration_seconds{operation}` - Latency

**Vector Metrics**:
- `contextd_vectors_total{database,collection}` - Vector count
- `contextd_vector_operations_total{operation}` - Operation count
- `contextd_search_latency_seconds{percentile}` - Search latency (p50, p95, p99)

**Migration Metrics**:
- `contextd_migration_status{phase}` - Migration phase
- `contextd_migration_vectors_migrated_total` - Migrated count
- `contextd_migration_duration_seconds` - Migration duration

### Traces

**Required Traces**:
- CreateDatabase (with database type)
- Insert vectors (with project_path, database)
- Search vectors (with database, collection)
- Migration operations (with phase)

**Trace Attributes**:
- `db.name` - Database name
- `db.operation` - Operation type
- `contextd.project_path` - Project path
- `contextd.collection` - Collection name
- `contextd.vector_count` - Vector count

---

## Configuration Requirements

### Multi-Tenant Settings

**Required Configuration**:
```yaml
database:
  multi_tenant:
    enabled: true  # v2.0+: ALWAYS true
    project_hash_algo: sha256  # sha256, sha512
    database_prefix: "project_"  # Prefix for project databases

  limits:
    max_databases: 100  # Per instance
    max_collections_per_db: 100  # Per database
    max_vectors_per_collection: 1000000  # Per collection
```

### Adapter Configuration

**Required Per-Adapter**:
- Connection URI
- Database strategy (native or prefix-based)
- Connection pool settings
- Timeout configuration

**Example (Qdrant)**:
```yaml
database:
  qdrant:
    host: "localhost"
    port: 6334
    use_collection_prefixes: true  # Required for Qdrant
```

---

## Summary

**Key Requirements**:
1. Physical database isolation per project
2. Shared database for cross-project knowledge
3. 10x+ performance improvement
4. Support 100+ projects per instance
5. Safe migration from legacy structure
6. Comprehensive monitoring and observability
7. 100% test coverage for core operations
