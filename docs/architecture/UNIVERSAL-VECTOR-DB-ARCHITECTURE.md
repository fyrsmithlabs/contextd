# Universal Vector Database Architecture

## Overview


## Core Principle

**The logical structure is independent of the underlying vector database implementation.**

```
┌─────────────────────────────────────┐
│   Application Layer                 │
│   (Checkpoints, Remediations, etc)  │
└─────────────────┬───────────────────┘
                  │
┌─────────────────▼───────────────────┐
│   Universal Abstraction Layer       │
│   (Database + Collection concept)   │
└─────────────────┬───────────────────┘
                  │
        ┌─────────┴─────────┐
        │                   │
┌───────▼──────┐   ┌────────▼────────┐
│  Adapter     │   │   Adapter       │
└──────────────┘   └─────────────────┘
        │                   │
┌───────▼──────┐   ┌────────▼────────┐
│   Weaviate   │   │   Pinecone      │
│   Adapter    │   │   Adapter       │
└──────────────┘   └─────────────────┘
```

## Logical Structure

### Three-Tier Namespace

```
contextd/
├── shared/                          # Tier 1: Shared/Global
│   ├── remediations
│   ├── troubleshooting_patterns
│   └── skills
│
├── project:<project-id>/            # Tier 2: Per-Project
│   ├── checkpoints
│   ├── research
│   └── notes
│
└── user:<user-id>/                  # Tier 3: Per-User (future)
    ├── personal_notes
    └── preferences
```

This logical structure is **universal** and maps to different vector databases differently.

## Universal Abstraction Layer

### Database Concept

**Definition**: A namespace containing related collections with isolation guarantees.

**Properties**:
- **Isolation**: Data in DB A cannot leak to DB B
- **Independence**: DBs can be created/deleted independently
- **Access Control**: Permissions scoped per database
- **Backup/Restore**: Can backup/restore individual databases

### Collection Concept

**Definition**: A set of vectors with the same schema within a database.

**Properties**:
- **Schema**: All vectors have consistent fields/dimensions
- **Indexing**: Optimized for similarity search
- **Filtering**: Support for metadata filtering

### Universal Interface

```go
package vectorstore

// Database represents a namespace for collections
type Database struct {
    Name       string
    Type       DatabaseType  // shared, project, user
    Metadata   map[string]string
}

// DatabaseType defines the scope/tier
type DatabaseType string

const (
    DatabaseTypeShared  DatabaseType = "shared"
    DatabaseTypeProject DatabaseType = "project"
    DatabaseTypeUser    DatabaseType = "user"
)

// UniversalVectorStore provides database-agnostic operations
type UniversalVectorStore interface {
    // Database operations
    CreateDatabase(ctx context.Context, db Database) error
    GetDatabase(ctx context.Context, name string) (*Database, error)
    ListDatabases(ctx context.Context, filter DatabaseType) ([]Database, error)
    DeleteDatabase(ctx context.Context, name string) error

    // Collection operations (all scoped to a database)
    CreateCollection(ctx context.Context, dbName, collName string, schema CollectionSchema) error
    DeleteCollection(ctx context.Context, dbName, collName string) error
    ListCollections(ctx context.Context, dbName string) ([]string, error)

    // Vector operations (all scoped to database + collection)
    Insert(ctx context.Context, dbName, collName string, vectors []Vector) error
    Search(ctx context.Context, dbName, collName string, query SearchQuery) ([]SearchResult, error)
    Delete(ctx context.Context, dbName, collName string, filter Filter) error

    // Bulk operations
    BulkInsert(ctx context.Context, ops []BulkOperation) error

    // Metadata
    GetCapabilities(ctx context.Context) Capabilities
}

// Capabilities describes what the underlying DB supports
type Capabilities struct {
    NativeDatabases     bool   // True if DB has native database concept
    MaxDatabases        int    // 0 = unlimited
    MaxCollections      int    // 0 = unlimited
    MaxVectorDimension  int
    SupportsHNSW        bool
    SupportsQuantization bool
    SupportsFiltering   bool
    SupportsTLS         bool
}
```

## Database Mapping Strategies

Different vector databases implement the logical structure differently:


**Vector DBs with native database support**:
- Weaviate 1.14+

**Mapping**:
```
Logical: shared/remediations
Physical: Database=shared, Collection=remediations

Logical: project_abc123/checkpoints
Physical: Database=project_abc123, Collection=checkpoints
```

**Implementation**:
```go
}

    return a.client.CreateDatabase(ctx, db.Name)
}

    // Switch to database
    a.client.UsingDatabase(dbName)
    // Insert into collection
    return a.client.Insert(ctx, collName, vectors)
}
```

### Strategy 2: Collection Prefixes (Qdrant, Pinecone)

**Vector DBs without native database support**:
- Qdrant
- Pinecone
- Chroma

**Mapping**:
```
Logical: shared/remediations
Physical: Collection=shared__remediations

Logical: project_abc123/checkpoints
Physical: Collection=project_abc123__checkpoints
```

**Separator**: Double underscore `__` (unlikely to appear in names)

**Implementation**:
```go
type QdrantAdapter struct {
    client *qdrant.Client
}

func (a *QdrantAdapter) CreateDatabase(ctx context.Context, db Database) error {
    // No-op: Qdrant doesn't have databases
    // Just store metadata
    return a.storeMetadata(db)
}

func (a *QdrantAdapter) Insert(ctx context.Context, dbName, collName string, vectors []Vector) error {
    physicalName := fmt.Sprintf("%s__%s", dbName, collName)
    return a.client.Upsert(ctx, physicalName, vectors)
}

func (a *QdrantAdapter) ListDatabases(ctx context.Context, filter DatabaseType) ([]Database, error) {
    // Parse collection names to extract database names
    collections, _ := a.client.ListCollections(ctx)

    dbMap := make(map[string]*Database)
    for _, coll := range collections {
        parts := strings.Split(coll, "__")
        if len(parts) == 2 {
            dbName := parts[0]
            if _, exists := dbMap[dbName]; !exists {
                dbMap[dbName] = &Database{Name: dbName}
            }
        }
    }

    return values(dbMap), nil
}
```

### Strategy 3: Namespaces (Pinecone, future)

**Vector DBs with namespace support**:
- Pinecone (namespaces within indexes)

**Mapping**:
```
Logical: shared/remediations
Physical: Index=contextd, Namespace=shared__remediations

Logical: project_abc123/checkpoints
Physical: Index=contextd, Namespace=project_abc123__checkpoints
```

**Implementation**:
```go
type PineconeAdapter struct {
    client *pinecone.Client
    indexName string
}

func (a *PineconeAdapter) Insert(ctx context.Context, dbName, collName string, vectors []Vector) error {
    namespace := fmt.Sprintf("%s__%s", dbName, collName)
    return a.client.Upsert(ctx, a.indexName, namespace, vectors)
}
```

### Strategy 4: Key Prefixes (Redis, future)

**Key-value stores with vector support**:
- Redis with RediSearch
- Valkey

**Mapping**:
```
Logical: shared/remediations
Physical: Keys=shared:remediations:*

Logical: project_abc123/checkpoints
Physical: Keys=project_abc123:checkpoints:*
```

## Adapter Pattern

Each vector database has an adapter implementing the `UniversalVectorStore` interface:

```go
// pkg/vectorstore/adapter/adapter.go

type Adapter interface {
    UniversalVectorStore

    // Adapter-specific initialization
    Initialize(ctx context.Context, config Config) error
    Close() error

    // Health check
    Ping(ctx context.Context) error
}

// Factory pattern
func NewAdapter(dbType string, config Config) (Adapter, error) {
    switch dbType {
    case "qdrant":
        return qdrant.NewAdapter(config)
    case "weaviate":
        return weaviate.NewAdapter(config)
    case "pinecone":
        return pinecone.NewAdapter(config)
    case "chroma":
        return chroma.NewAdapter(config)
    default:
        return nil, fmt.Errorf("unsupported vector db: %s", dbType)
    }
}
```

## Database Naming Convention

### Shared Database

**Format**: `shared`

**Purpose**: Global knowledge accessible across all projects

**Collections**:
- `remediations`
- `troubleshooting_patterns`
- `skills`

### Project Database

**Format**: `project_<hash>`

**Hash Algorithm**: SHA256(project_path)[:8]

**Examples**:
```
/home/user/projects/contextd
→ SHA256 → abc123de45678901...
→ Database: project_abc123de

/var/www/myapp
→ SHA256 → def456ab78901234...
→ Database: project_def456ab
```

**Collections**:
- `checkpoints`
- `research`
- `notes`

### User Database (Future)

**Format**: `user_<username>` or `user_<hash>`

**Examples**:
```
Username: john
→ Database: user_john

User ID: 12345
→ Database: user_12345
```

**Collections**:
- `personal_notes`
- `preferences`

## Configuration

### Environment Variables

```bash
# Vector DB type (adapter selection)

# Structure mode
export VECTOR_DB_STRUCTURE=multi       # multi or flat (legacy)

# Naming
export VECTOR_DB_SHARED_NAME=shared
export VECTOR_DB_PROJECT_HASH_ALGO=sha256
export VECTOR_DB_PROJECT_HASH_LENGTH=8

# Database-specific
export QDRANT_HOST=localhost
export QDRANT_PORT=6334
export WEAVIATE_URL=http://localhost:8080
export PINECONE_API_KEY=xxx
export PINECONE_ENVIRONMENT=us-west1-gcp
```

### Config File

```yaml
vector_db:
  # Adapter selection

  # Structure
  structure: multi  # multi or flat

  # Naming
  shared_database: shared
  project_hash:
    algorithm: sha256
    length: 8

  # Database-specific config
  qdrant:
    host: localhost
    port: 6334
    api_key: ${QDRANT_API_KEY}

    uri: localhost:19530

  weaviate:
    url: http://localhost:8080
    api_key: ${WEAVIATE_API_KEY}
```

## Collection Schemas (Universal)

### Shared Collections

#### remediations
```yaml
Collection: shared/remediations
Purpose: Error solutions usable across projects
Schema:
  vector:
    dimension: 384 or 1536
    distance: cosine
  payload:
    id: uuid
    error_message: text (indexed)
    solution: text
    stack_trace: text
    tags: keyword[] (indexed)
    language: keyword (indexed)
    framework: keyword
    severity: keyword (indexed)
    usage_count: integer
    success_rate: float
    created_at: timestamp (indexed)
    updated_at: timestamp
```

#### troubleshooting_patterns
```yaml
Collection: shared/troubleshooting_patterns
Purpose: Common troubleshooting workflows
Schema:
  vector:
    dimension: 384 or 1536
    distance: cosine
  payload:
    id: uuid
    pattern_name: text
    description: text
    diagnostic_steps: text[]
    solutions: text[]
    tags: keyword[] (indexed)
    category: keyword (indexed)
    success_rate: float
    usage_count: integer
    created_at: timestamp (indexed)
```

#### skills
```yaml
Collection: shared/skills
Purpose: Reusable code/workflow templates
Schema:
  vector:
    dimension: 384 or 1536
    distance: cosine
  payload:
    id: uuid
    name: text (indexed)
    description: text
    content: text
    version: keyword
    author: keyword (indexed)
    category: keyword (indexed)
    prerequisites: text[]
    expected_outcome: text
    tags: keyword[] (indexed)
    usage_count: integer
    success_rate: float
    created_at: timestamp (indexed)
    updated_at: timestamp
```

### Project Collections

#### checkpoints
```yaml
Collection: project_<hash>/checkpoints
Purpose: Project-specific session checkpoints
Schema:
  vector:
    dimension: 384 or 1536
    distance: cosine
  payload:
    id: uuid
    summary: text
    content: text
    project_path: keyword (indexed)
    timestamp: timestamp (indexed)
    token_count: integer
    tags: keyword[] (indexed)
    git_branch: keyword
    git_commit: keyword
    files_changed: keyword[]
```

#### research
```yaml
Collection: project_<hash>/research
Purpose: Project-specific documentation
Schema:
  vector:
    dimension: 384 or 1536
    distance: cosine
  payload:
    id: uuid
    title: text
    document_section: text
    category: keyword (indexed)
    key_findings: text
    recommendations: text
    source_url: keyword
    project_path: keyword (indexed)
    tags: keyword[] (indexed)
    date_added: timestamp (indexed)
```

#### notes
```yaml
Collection: project_<hash>/notes
Purpose: Session notes and observations
Schema:
  vector:
    dimension: 384 or 1536
    distance: cosine
  payload:
    id: uuid
    session_id: keyword (indexed)
    note_type: keyword (indexed)
    title: text
    content: text
    metadata: json
    tags: keyword[] (indexed)
    project_path: keyword (indexed)
    timestamp: timestamp (indexed)
```

## Implementation

### Directory Structure

```
pkg/vectorstore/
├── interface.go              # UniversalVectorStore interface
├── types.go                  # Common types (Vector, Database, etc)
├── database.go               # Database naming and hashing logic
├── adapter/
│   ├── adapter.go           # Adapter interface and factory
│   ├── qdrant/
│   │   ├── adapter.go       # Qdrant adapter (Strategy 2)
│   │   └── mapping.go       # Collection prefix logic
│   ├── weaviate/
│   │   ├── adapter.go       # Weaviate adapter (Strategy 1)
│   │   └── mapping.go
│   ├── pinecone/
│   │   ├── adapter.go       # Pinecone adapter (Strategy 3)
│   │   └── mapping.go
│   └── chroma/
│       ├── adapter.go       # Chroma adapter (Strategy 2)
│       └── mapping.go
├── migration/
│   ├── flat_to_multi.go     # Migrate flat → multi-DB
│   └── validator.go         # Validate migration
└── testing/
    ├── mock.go              # Mock adapter for tests
    └── suite.go             # Universal test suite
```

### Key Implementation Files

#### pkg/vectorstore/database.go
```go
package vectorstore

import (
    "crypto/sha256"
    "fmt"
)

// GetDatabaseName returns the database name for a given scope
func GetDatabaseName(scope DatabaseType, identifier string) string {
    switch scope {
    case DatabaseTypeShared:
        return "shared"
    case DatabaseTypeProject:
        hash := projectHash(identifier)
        return fmt.Sprintf("project_%s", hash[:8])
    case DatabaseTypeUser:
        return fmt.Sprintf("user_%s", identifier)
    default:
        return "shared"
    }
}

// projectHash generates a hash for a project path
func projectHash(path string) string {
    h := sha256.Sum256([]byte(path))
    return fmt.Sprintf("%x", h)
}

// GetCollectionName returns the full collection name
func GetCollectionName(adapter Adapter, dbName, collName string) string {
    caps := adapter.GetCapabilities(context.Background())

    if caps.NativeDatabases {
        // Strategy 1: Native databases
        return collName
    } else {
        // Strategy 2: Collection prefixes
        return fmt.Sprintf("%s__%s", dbName, collName)
    }
}
```

#### pkg/vectorstore/adapter/qdrant/adapter.go
```go
package qdrant

type Adapter struct {
    client *qdrant.Client
    config Config
}

func (a *Adapter) GetCapabilities(ctx context.Context) Capabilities {
    return Capabilities{
        NativeDatabases:    false,  // Qdrant uses collection prefixes
        MaxDatabases:       0,      // Unlimited
        MaxCollections:     0,      // Unlimited
        MaxVectorDimension: 65536,
        SupportsHNSW:       true,
        SupportsQuantization: true,
        SupportsFiltering:  true,
        SupportsTLS:        true,
    }
}

func (a *Adapter) CreateDatabase(ctx context.Context, db Database) error {
    // Qdrant doesn't have databases - this is a logical operation
    // Store metadata in internal collection or config
    return a.storeMetadata(ctx, db)
}

func (a *Adapter) Insert(ctx context.Context, dbName, collName string, vectors []Vector) error {
    // Use collection prefix strategy
    physicalName := fmt.Sprintf("%s__%s", dbName, collName)
    return a.client.Upsert(ctx, physicalName, vectors)
}
```

```go

type Adapter struct {
    config Config
}

func (a *Adapter) GetCapabilities(ctx context.Context) Capabilities {
    return Capabilities{
        MaxDatabases:       100,
        MaxCollections:     1000,
        MaxVectorDimension: 32768,
        SupportsHNSW:       true,
        SupportsQuantization: true,
        SupportsFiltering:  true,
        SupportsTLS:        true,
    }
}

func (a *Adapter) CreateDatabase(ctx context.Context, db Database) error {
    return a.client.CreateDatabase(ctx, db.Name)
}

func (a *Adapter) Insert(ctx context.Context, dbName, collName string, vectors []Vector) error {
    // Switch to database first
    if err := a.client.UsingDatabase(dbName); err != nil {
        return err
    }
    // Insert into collection
    return a.client.Insert(ctx, collName, vectors)
}
```

## Service Layer Updates

Services should be database-aware:

```go
// pkg/checkpoint/service.go

type Service struct {
    store      vectorstore.UniversalVectorStore
    embedding  *embedding.Service
    projectDB  string  // project_abc123de
}

// NewService creates a checkpoint service for a specific project
func NewService(
    store vectorstore.UniversalVectorStore,
    embedding *embedding.Service,
    projectPath string,
) *Service {
    // Calculate database name from project path
    dbName := vectorstore.GetDatabaseName(
        vectorstore.DatabaseTypeProject,
        projectPath,
    )

    return &Service{
        store:     store,
        embedding: embedding,
        projectDB: dbName,
    }
}

func (s *Service) Save(ctx context.Context, cp *Checkpoint) error {
    // Generate embedding
    emb, err := s.embedding.Generate(ctx, cp.Summary+"\n"+cp.Content)
    if err != nil {
        return err
    }

    // Insert into project database
    return s.store.Insert(ctx, s.projectDB, "checkpoints", []Vector{
        {
            ID:        cp.ID,
            Embedding: emb,
            Payload:   cp.ToMap(),
        },
    })
}
```

## Migration Strategy

### Flat to Multi-DB Migration

```bash
# Command
ctxd migrate-structure --from flat --to multi --dry-run

# Process
1. Scan all existing collections
2. Group vectors by project_path field
3. Calculate project database names
4. Create new databases
5. Copy vectors to appropriate databases
6. Verify data integrity
7. Backup old structure
8. Switch to multi-DB mode
```

### Migration Script

```go
// cmd/ctxd/migrate_structure.go

func migrateStructure(from, to string, dryRun bool) error {
    // 1. Connect to vector store
    store := connectVectorStore()

    // 2. Read all checkpoints from flat structure
    checkpoints, _ := store.ListCollections(ctx, "")

    // 3. Group by project
    projectGroups := make(map[string][]Checkpoint)
    for _, cp := range checkpoints {
        projectGroups[cp.ProjectPath] = append(projectGroups[cp.ProjectPath], cp)
    }

    // 4. Create project databases
    for projectPath, cps := range projectGroups {
        dbName := vectorstore.GetDatabaseName(
            vectorstore.DatabaseTypeProject,
            projectPath,
        )

        if !dryRun {
            store.CreateDatabase(ctx, Database{Name: dbName, Type: DatabaseTypeProject})
            store.CreateCollection(ctx, dbName, "checkpoints", checkpointSchema)
            store.BulkInsert(ctx, dbName, "checkpoints", cps)
        }

        fmt.Printf("Would migrate %d checkpoints to %s\n", len(cps), dbName)
    }

    // 5. Move remediations to shared DB
    // ... similar process

    return nil
}
```

## Testing

### Universal Test Suite

All adapters must pass the same test suite:

```go
// pkg/vectorstore/testing/suite.go

type AdapterTestSuite struct {
    suite.Suite
    adapter Adapter
}

func (s *AdapterTestSuite) TestDatabaseOperations() {
    ctx := context.Background()

    // Create database
    db := Database{Name: "test_db", Type: DatabaseTypeProject}
    err := s.adapter.CreateDatabase(ctx, db)
    s.NoError(err)

    // List databases
    dbs, err := s.adapter.ListDatabases(ctx, DatabaseTypeProject)
    s.NoError(err)
    s.Contains(dbs, db)

    // Delete database
    err = s.adapter.DeleteDatabase(ctx, "test_db")
    s.NoError(err)
}

func (s *AdapterTestSuite) TestVectorOperations() {
    // Insert, search, delete tests
    // Must work identically across all adapters
}

// Run suite for each adapter
func TestQdrantAdapter(t *testing.T) {
    suite.Run(t, &AdapterTestSuite{
        adapter: qdrant.NewAdapter(config),
    })
}

    suite.Run(t, &AdapterTestSuite{
    })
}
```

## Rollout Plan

### Phase 1: Abstraction Layer (Week 1-2)
- ✅ Define `UniversalVectorStore` interface
- ✅ Implement database naming logic
- ✅ Create adapter factory
- ✅ Universal test suite

### Phase 2: Adapters (Week 3-4)
- ✅ Qdrant adapter (Strategy 2: Prefixes)
- ⏳ Weaviate adapter (optional)
- ⏳ Pinecone adapter (optional)

### Phase 3: Service Updates (Week 5)
- ✅ Update checkpoint service
- ✅ Update remediation service
- ✅ Update research service
- ✅ Update skills service

### Phase 4: Migration (Week 6)
- ✅ Migration script
- ✅ Data validation
- ✅ Rollback capability

### Phase 5: Deployment (Week 7+)
- ✅ Feature flag
- ✅ Gradual rollout
- ✅ Monitor performance
- ✅ Documentation

## Benefits

### 1. Database Independence

✅ **Switch databases easily**
```bash
# Today: Qdrant
export VECTOR_DB_TYPE=qdrant


# Next week: Weaviate
export VECTOR_DB_TYPE=weaviate
```

✅ **Multi-database support**
```bash

# Use Qdrant for shared (better filtering)
export VECTOR_DB_SHARED_TYPE=qdrant
```

### 2. Future-Proof

✅ **New databases easily added**
- Implement adapter
- Pass test suite
- Deploy

✅ **Adapt to database evolution**
- Qdrant adds native DBs → Switch to Strategy 1
- Pinecone adds features → Enhance adapter

### 3. Testing

✅ **Mock adapter for unit tests**
```go
mockStore := testing.NewMockAdapter()
svc := checkpoint.NewService(mockStore, ...)
```

✅ **Integration tests work with any DB**
```bash
# Test with Qdrant
VECTOR_DB_TYPE=qdrant go test ./...

```

### 4. Hybrid Deployments

✅ **Best tool for each job**
```yaml
shared:

projects:
  type: qdrant        # Better performance and filtering
  host: qdrant-cluster.svc
```

## Open Questions

1. **Should we support hybrid deployments?** (Different DBs for different tiers)
3. **Metadata storage?** Where to store database metadata for DBs without native databases?
4. **Migration backwards compatibility?** Support old flat structure indefinitely?

## Next Steps

1. ✅ Review this universal architecture
2. ⏳ Approve design and naming
3. ⏳ Implement abstraction layer
4. ⏳ Update existing Qdrant code to use adapter
6. ⏳ Create migration tooling

---

**Status**: 🔍 Proposal (Awaiting Feedback)
**Author**: Claude Code
**Date**: 2025-01-01
**Version**: 2.0 (Universal Architecture)
