# Multi-Tenant Architecture Specification

**Version**: 2.0.0
**Status**: Implemented
**Date**: 2025-11-04
**Last Updated**: 2025-11-18

---

## Overview

The contextd multi-tenant architecture provides secure, isolated data storage for multiple projects using a database-per-project strategy. This architecture eliminates filter injection vulnerabilities, provides true physical isolation, and delivers significant performance improvements through partition pruning.

### Key Features

- **Database-Per-Project**: Each project gets its own isolated database
- **Shared Global Database**: Cross-project knowledge (remediations, skills, troubleshooting)
- **Project Hash Generation**: SHA256-based deterministic database naming
- **Filter Injection Prevention**: Physical isolation prevents metadata bypass attacks
- **10-16x Performance**: Partition pruning eliminates full-scan overhead
- **Universal Abstraction**: Database-agnostic interface supports multiple vector stores

---

## Quick Reference

### Architecture Choice

**Database-Per-Project** provides:
- **Strongest Isolation**: Complete physical separation between projects
- **Performance**: 10-16x faster via partition pruning
- **Security**: Eliminates filter injection vulnerabilities
- **Flexibility**: Works with any vector database through adapter pattern

**Trade-offs accepted**:
- Database limit: ~100-256 per instance (sufficient for expected load)
- Management overhead: Must track project-to-database mapping

### Three-Tier Namespace

```
contextd/
├── shared/                  # Global Knowledge (all projects)
│   ├── remediations
│   ├── skills
│   └── troubleshooting_patterns
│
├── project_<hash>/         # Per-Project (isolated)
│   ├── checkpoints
│   ├── research
│   └── notes
│
└── user_<id>/              # Per-User (future)
    └── preferences
```

### Database Naming

- **Shared**: `shared` (fixed name)
- **Project**: `project_<hash>` where hash = SHA256(project_path)[:8]
- **User**: `user_<id>` (future)

**Example**:
```
/home/user/projects/contextd → project_abc123de
```

### Performance Comparison

| Metric | Filter-Based | Database-Per-Project | Improvement |
|--------|-------------|---------------------|-------------|
| Query Time (10 projects) | 100ms | 10ms | **10x faster** |
| Query Time (100 projects) | 1000ms | 10ms | **100x faster** |
| Memory Usage | High (full scan) | Low (project only) | **10-100x less** |

---

## Detailed Documentation

### Requirements & Design

@./multi-tenant/requirements.md - Functional and non-functional requirements, testing requirements

**Topics covered**:
- Design goals (security, performance, scalability)
- Functional requirements (isolation, shared knowledge, naming)
- Non-functional requirements (performance targets, security guarantees)
- Database and collection requirements
- Migration requirements
- Error handling and configuration

@./multi-tenant/architecture.md - System design and component interactions

**Topics covered**:
- Three-tier namespace structure
- Database naming convention and hash generation
- Physical isolation strategy and benefits
- Universal abstraction layer (UniversalVectorStore interface)
- Adapter pattern for multiple vector databases
- Service integration (checkpoint, remediation)
- Performance characteristics and scalability

@./multi-tenant/security.md - Security model and filter injection prevention

**Topics covered**:
- Filter injection vulnerability (legacy approach)
- Database-per-project solution (physical isolation)
- Hash-based database naming security
- Security benefits and attack surface reduction
- Threat model (in-scope, out-of-scope)
- Security boundaries (project, shared, user)
- Access control (current and future)
- Audit logging (future)

### Implementation

@./multi-tenant/implementation.md - Implementation details and package structure

**Topics covered**:
- Package structure (`pkg/vectorstore/`)
- Core interfaces (UniversalVectorStore)
- Naming helpers (database name generation)
- Qdrant adapter implementation (collection prefix strategy)
- Migration tool implementation
- Service integration (checkpoint, remediation)
- Error handling and configuration

@./multi-tenant/workflows.md - Common workflows and usage examples

**Topics covered**:
- Database creation (project and shared)
- Collection creation with schemas
- Vector insert (project-specific and shared)
- Vector search (project-scoped and cross-project)
- Migration workflow (analyze, create, migrate, validate, cleanup)
- Project hash generation examples
- Database routing in service layer
- Multi-project search (sequential and concurrent)
- Monitoring and observability

---

## Migration from Legacy

### Quick Migration

```bash
# 1. Backup
contextd backup create --output /backup/legacy.tar.gz

# 2. Analyze
contextd migrate analyze --report project_distribution.json

# 3. Execute
contextd migrate execute --concurrency 4

# 4. Validate
contextd migrate validate

# 5. Cleanup
contextd migrate cleanup --confirm
```

**See**: [implementation.md](./multi-tenant/implementation.md#migration-implementation) for detailed migration implementation.

---

## Breaking Changes in v2.0.0

### Removed: Legacy Mode

- `MULTI_TENANT_MODE` environment variable removed
- Multi-tenant mode is now **MANDATORY** (always enabled)
- Cannot run in legacy flat structure mode

**Migration Required**: Users on v1.x MUST migrate before upgrading to v2.0.0

---

## API Reference

### Core Operations

**Database Operations**:
- `CreateDatabase(ctx, db)` - Create project or shared database
- `GetDatabase(ctx, name)` - Get database metadata
- `ListDatabases(ctx, filter)` - List databases by type
- `DeleteDatabase(ctx, name)` - Delete database and all data

**Collection Operations**:
- `CreateCollection(ctx, dbName, collName, schema)` - Create collection
- `ListCollections(ctx, dbName)` - List collections in database
- `DeleteCollection(ctx, dbName, collName)` - Delete collection

**Vector Operations**:
- `Insert(ctx, dbName, collName, vectors)` - Insert/upsert vectors
- `Search(ctx, dbName, collName, query)` - Semantic search
- `Delete(ctx, dbName, collName, filter)` - Delete vectors
- `Get(ctx, dbName, collName, ids)` - Get vectors by ID

**See**: [architecture.md](./multi-tenant/architecture.md#universal-abstraction-layer) for complete API documentation.

---

## Summary

### Isolation Strategy

**Database-Per-Project** provides:
- **Physical Isolation**: Complete separation between projects
- **Security**: Eliminates filter injection attacks (no metadata filters)
- **Performance**: 10-16x faster via partition pruning
- **Scalability**: 100+ projects per instance

### Implementation Status

**Version**: 2.0.0 (Implemented)

**Package**: `pkg/vectorstore/`

**Adapters**:
- ✅ Qdrant (collection prefix strategy)
- 🔄 Weaviate (future)
- 🔄 Milvus (future)

**Next Steps**:
1. Monitor performance metrics
2. Gather user feedback on migration
3. Implement database-level ACLs (v2.1)
4. Add per-database resource limits (v2.2)

---

## References

- **ADR**: [docs/architecture/adr/002-universal-multi-tenant-architecture.md](../../architecture/adr/002-universal-multi-tenant-architecture.md)
- **Universal Architecture**: [docs/architecture/UNIVERSAL-VECTOR-DB-ARCHITECTURE.md](../../architecture/UNIVERSAL-VECTOR-DB-ARCHITECTURE.md)
- **Security Audit**: [docs/security/SECURITY-AUDIT-SHARED-DATABASE-2025-01-07.md](../../security/SECURITY-AUDIT-SHARED-DATABASE-2025-01-07.md)
- **Implementation**: `pkg/vectorstore/` package

---

**Version History**:
- v2.0.0 (2025-01-03): Initial implementation with database-per-project
- v1.x (deprecated): Legacy filter-based multi-tenancy
