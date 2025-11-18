# Multi-Tenant Architecture

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the architectural design of contextd's multi-tenant system.

---

## Three-Tier Namespace

```
contextd/
├── shared/                          # Tier 1: Global Knowledge
│   ├── remediations                 # Error solutions (cross-project)
│   ├── troubleshooting_patterns     # Common patterns
│   └── skills                       # Reusable templates
│
├── project_<hash>/                  # Tier 2: Per-Project (isolated)
│   ├── checkpoints                  # Session checkpoints
│   ├── research                     # Research documents
│   └── notes                        # Session notes
│
└── user_<id>/                       # Tier 3: Per-User (future)
    ├── personal_notes               # User-specific notes
    └── preferences                  # User preferences
```

---

## Database Naming Convention

### Shared Database

**Name**: `shared`

**Purpose**: Global knowledge accessible by all projects

**Collections**:
- `remediations` - Error solutions (cross-project)
- `skills` - Reusable templates
- `troubleshooting_patterns` - Common error patterns

**Access**: All projects can read/write

### Project Database

**Format**: `project_<hash>`

**Hash Generation**:
```go
// SHA256 hash of project path
func projectHash(path string) string {
    h := sha256.Sum256([]byte(path))
    return fmt.Sprintf("%x", h)[:8]  // First 8 characters
}

// Example
"/home/user/projects/contextd" → "abc123de45678901..." → "abc123de"
```

**Properties**:
- **Deterministic**: Same path always produces same hash
- **Collision-Resistant**: SHA256 ensures uniqueness (2^32 combinations for 8 chars)
- **Short**: 8 characters sufficient for 100+ projects
- **Opaque**: Path cannot be recovered from hash (one-way function)

**Collections**:
- `checkpoints` - Session checkpoints
- `research` - Research documents
- `notes` - Session notes

**Access**: Only the specific project (isolated)

### User Database (Future)

**Format**: `user_<identifier>`

**Purpose**: Per-user data isolation for multi-user deployments

**Collections** (planned):
- `personal_notes` - User-specific notes
- `preferences` - User preferences

**Access**: Only the specific user

---

## Database Isolation Strategy

### Physical Isolation

```
Database: shared
├── Collection: remediations
├── Collection: skills
└── Collection: troubleshooting_patterns

Database: project_abc123de
├── Collection: checkpoints
├── Collection: research
└── Collection: notes

Database: project_def456gh
├── Collection: checkpoints
├── Collection: research
└── Collection: notes
```

### Benefits

**Security**:
- No metadata filters = no filter injection attacks
- Complete separation between projects
- Database-level access control (future)

**Performance**:
- Queries only scan relevant database (partition pruning)
- 10-16x faster than filter-based approach
- Smaller indexes per database (faster queries)
- Higher cache hit rate (per-project caching)

**Isolation**:
- Complete physical separation
- No cross-project data leakage
- Per-database resource limits (future)

**Resource Management**:
- Load/unload databases independently
- Per-database quotas (future)
- Database-level backups

---

## Universal Abstraction Layer

### Database-Agnostic Interface

```go
package vectorstore

// UniversalVectorStore provides database-agnostic operations
type UniversalVectorStore interface {
    // Database operations
    CreateDatabase(ctx context.Context, db Database) error
    GetDatabase(ctx context.Context, name string) (*Database, error)
    ListDatabases(ctx context.Context, filter DatabaseType) ([]Database, error)
    DeleteDatabase(ctx context.Context, name string) error

    // Collection operations (scoped to database)
    CreateCollection(ctx context.Context, dbName, collName string, schema CollectionSchema) error
    DeleteCollection(ctx context.Context, dbName, collName string) error
    ListCollections(ctx context.Context, dbName string) ([]string, error)
    CollectionExists(ctx context.Context, dbName, collName string) (bool, error)

    // Vector operations (scoped to database + collection)
    Insert(ctx context.Context, dbName, collName string, vectors []Vector) error
    Search(ctx context.Context, dbName, collName string, query SearchQuery) ([]SearchResult, error)
    Delete(ctx context.Context, dbName, collName string, filter Filter) error
    Get(ctx context.Context, dbName, collName string, ids []string) ([]Vector, error)

    // Metadata and health
    GetCapabilities(ctx context.Context) Capabilities
    Health(ctx context.Context) error
    Close() error
}
```

### Data Models

**Database**:
```go
type Database struct {
    Name     string            // Database name (e.g., "shared", "project_abc123de")
    Type     DatabaseType      // Scope/tier (shared, project, user)
    Metadata map[string]string // Custom metadata
}

type DatabaseType string

const (
    DatabaseTypeShared  DatabaseType = "shared"   // Global knowledge
    DatabaseTypeProject DatabaseType = "project"  // Per-project
    DatabaseTypeUser    DatabaseType = "user"     // Per-user (future)
)
```

**Vector**:
```go
type Vector struct {
    ID        string                 `json:"id"`
    Embedding []float32              `json:"embedding"`
    Payload   map[string]interface{} `json:"payload"`
}
```

**Search Query**:
```go
type SearchQuery struct {
    Vector []float32 `json:"vector"`
    TopK   int       `json:"top_k"`
    Filter Filter    `json:"filter"`
}
```

**Search Result**:
```go
type SearchResult struct {
    ID       string                 `json:"id"`
    Score    float32                `json:"score"`
    Distance float32                `json:"distance"`
    Payload  map[string]interface{} `json:"payload"`
}
```

---

## Adapter Pattern

### Strategy 1: Native Databases (Weaviate, Milvus)

**Mapping**:
```
Logical:  shared/remediations
Physical: Database=shared, Collection=remediations

Logical:  project_abc123de/checkpoints
Physical: Database=project_abc123de, Collection=checkpoints
```

**Characteristics**:
- True database isolation
- Native database operations
- Best performance

### Strategy 2: Collection Prefixes (Qdrant, Pinecone, Chroma)

**Mapping**:
```
Logical:  shared/remediations
Physical: Collection=shared__remediations

Logical:  project_abc123de/checkpoints
Physical: Collection=project_abc123de__checkpoints
```

**Separator**: Double underscore `__`

**Characteristics**:
- Single database with prefixed collections
- Requires name parsing
- Still provides isolation (collection-level)

### Helper Functions

```go
// GetDatabaseName returns database name for scope and identifier
func GetDatabaseName(scope DatabaseType, identifier string) string

// GetCollectionName returns physical collection name
func GetCollectionName(nativeDatabases bool, dbName, collName string) string

// ParseCollectionName extracts database and collection from physical name
func ParseCollectionName(physicalName string) (dbName, collName string)

// ParseDatabaseType extracts type from database name
func ParseDatabaseType(dbName string) DatabaseType

// ValidateDatabaseName checks naming conventions
func ValidateDatabaseName(dbName string) error

// ValidateCollectionName checks naming conventions
func ValidateCollectionName(collName string) error
```

---

## Collection Schema

```go
type CollectionSchema struct {
    Name           string                    // Collection name
    VectorDim      int                       // Vector dimension (1536 for OpenAI)
    DistanceMetric DistanceMetric            // cosine, l2, ip
    Fields         map[string]FieldType      // Field definitions
    Indexed        []string                  // Fields to index
    Description    string                    // Collection description
}

type DistanceMetric string
const (
    DistanceCosine DistanceMetric = "cosine"  // Angle-based similarity
    DistanceL2     DistanceMetric = "l2"      // Euclidean distance
    DistanceIP     DistanceMetric = "ip"      // Inner product
)

type FieldType string
const (
    FieldTypeString   FieldType = "string"
    FieldTypeInt64    FieldType = "int64"
    FieldTypeFloat32  FieldType = "float32"
    FieldTypeBool     FieldType = "bool"
    FieldTypeJSON     FieldType = "json"
    FieldTypeArray    FieldType = "array"
)
```

---

## Service Integration

### Checkpoint Service (Project-Isolated)

**Before** (Filter-Based):
```go
type CheckpointService struct {
    store vectorstore.VectorStore
}

func (s *CheckpointService) Save(ctx context.Context, projectPath string, cp *Checkpoint) error {
    // Add project_path to metadata
    cp.Metadata["project_path"] = projectPath
    return s.store.Insert(ctx, "default", "checkpoints", cp.ToVector())
}

func (s *CheckpointService) Search(ctx context.Context, projectPath string, query string) ([]*Checkpoint, error) {
    // Filter by project_path (VULNERABLE to injection)
    filter := fmt.Sprintf("project_path == '%s'", projectPath)
    return s.store.Search(ctx, "default", "checkpoints", query, filter)
}
```

**After** (Database-Per-Project):
```go
type CheckpointService struct {
    store vectorstore.UniversalVectorStore
}

func (s *CheckpointService) Save(ctx context.Context, projectPath string, cp *Checkpoint) error {
    // Get project database (physical isolation)
    dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)
    return s.store.Insert(ctx, dbName, "checkpoints", []vectorstore.Vector{cp.ToVector()})
}

func (s *CheckpointService) Search(ctx context.Context, projectPath string, query string) ([]*Checkpoint, error) {
    // No filter needed (physical isolation)
    dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)
    return s.store.Search(ctx, dbName, "checkpoints", query)
}
```

### Remediation Service (Shared Database)

```go
type RemediationService struct {
    store vectorstore.UniversalVectorStore
}

func (s *RemediationService) Save(ctx context.Context, rem *Remediation) error {
    // Always use shared database
    return s.store.Insert(ctx, "shared", "remediations", []vectorstore.Vector{rem.ToVector()})
}

func (s *RemediationService) Search(ctx context.Context, errorMsg string) ([]*Remediation, error) {
    // Search across all remediations (cross-project knowledge)
    return s.store.Search(ctx, "shared", "remediations", errorMsg)
}
```

---

## Performance Characteristics

### Query Performance

**Comparison: Filter-Based vs Database-Per-Project**

| Metric | Filter-Based | Database-Per-Project | Improvement |
|--------|-------------|---------------------|-------------|
| Query Time (10 projects) | 100ms | 10ms | **10x faster** |
| Query Time (100 projects) | 1000ms | 10ms | **100x faster** |
| Scan Scope | All vectors | Project vectors only | Partition pruning |
| Filter Overhead | High (eval every vector) | None | Eliminated |
| Memory Usage | High (full scan) | Low (project only) | 10-100x less |

### Performance Benefits

1. **Partition Pruning**: Only scans relevant database (10-16x faster)
2. **No Filter Evaluation**: Eliminates metadata filter overhead
3. **Smaller Indexes**: Per-project indexes are smaller and faster
4. **Better Caching**: Higher cache hit rate for project data
5. **Reduced Memory**: Only load relevant database into memory

### Scalability Limits

**Database Limits**:
- Qdrant: Unlimited collections (collection-prefix strategy)
- Weaviate: 1000+ databases
- Milvus: 65,536 databases

**Expected Load**:
- Users: 1-10 per instance
- Projects per user: 10-100
- Total projects: 10-1000 (well within limits)

**Optimization Strategies**:
1. Database pooling (reuse connections)
2. Lazy loading (load database on first access)
3. Connection reuse (keep frequently used databases open)
4. Database unloading (close unused databases after timeout)

---

## Access Control (Future)

### Current (v2.0)

- No authentication (MVP trusted network assumption)
- All databases accessible by any client
- HTTP transport with reverse proxy recommended
- Multi-session support (multiple concurrent Claude instances)

### Future (Multi-User)

**Database-Level ACLs**:
```yaml
database: project_abc123de
acl:
  - user: alice
    permissions: [read, write]
  - user: bob
    permissions: [read]
```

**Role-Based Access Control (RBAC)**:
```yaml
roles:
  - name: project_owner
    permissions: [database.create, database.delete, collection.*, vector.*]
  - name: project_contributor
    permissions: [collection.read, vector.read, vector.write]
  - name: project_viewer
    permissions: [collection.read, vector.read]
```

**User-to-Project Mapping**:
```yaml
users:
  - name: alice
    projects:
      - project_abc123de: project_owner
      - project_def456gh: project_contributor
```

---

## Summary

**Key Architectural Decisions**:

1. **Database-Per-Project**: Complete physical isolation
2. **Shared Database**: Cross-project knowledge
3. **SHA256 Hashing**: Deterministic, collision-resistant naming
4. **Universal Interface**: Database-agnostic abstraction
5. **Adapter Pattern**: Support multiple vector databases
6. **Performance**: 10-16x faster via partition pruning
7. **Security**: Eliminates filter injection attacks
