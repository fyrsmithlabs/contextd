# Multi-Tenant Implementation

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the implementation details of contextd's multi-tenant architecture.

---

## Implementation Status

**Version**: 2.0.0
**Status**: Implemented
**Date**: 2025-11-04

---

## Package Structure

```
pkg/vectorstore/
├── vectorstore.go           # UniversalVectorStore interface
├── database.go              # Database operations
├── collection.go            # Collection operations
├── vector.go                # Vector operations
├── naming.go                # Database/collection naming helpers
├── adapter/
│   ├── qdrant/
│   │   ├── adapter.go      # Qdrant adapter implementation
│   │   ├── database.go     # Database operations (collection prefixes)
│   │   └── collection.go   # Collection operations
│   └── weaviate/           # Future: Weaviate adapter
└── vectorstore_test.go     # Integration tests
```

---

## Core Implementation

### UniversalVectorStore Interface

**Location**: `pkg/vectorstore/vectorstore.go`

```go
package vectorstore

type UniversalVectorStore interface {
    // Database operations
    CreateDatabase(ctx context.Context, db Database) error
    GetDatabase(ctx context.Context, name string) (*Database, error)
    ListDatabases(ctx context.Context, filter DatabaseType) ([]Database, error)
    DeleteDatabase(ctx context.Context, name string) error

    // Collection operations
    CreateCollection(ctx context.Context, dbName, collName string, schema CollectionSchema) error
    DeleteCollection(ctx context.Context, dbName, collName string) error
    ListCollections(ctx context.Context, dbName string) ([]string, error)
    CollectionExists(ctx context.Context, dbName, collName string) (bool, error)

    // Vector operations
    Insert(ctx context.Context, dbName, collName string, vectors []Vector) error
    Search(ctx context.Context, dbName, collName string, query SearchQuery) ([]SearchResult, error)
    Delete(ctx context.Context, dbName, collName string, filter Filter) error
    Get(ctx context.Context, dbName, collName string, ids []string) ([]Vector, error)

    // Metadata
    GetCapabilities(ctx context.Context) Capabilities
    Health(ctx context.Context) error
    Close() error
}
```

### Naming Helpers

**Location**: `pkg/vectorstore/naming.go`

```go
// GetDatabaseName returns database name for scope and identifier
func GetDatabaseName(scope DatabaseType, identifier string) string {
    switch scope {
    case DatabaseTypeShared:
        return "shared"
    case DatabaseTypeProject:
        return fmt.Sprintf("project_%s", hashIdentifier(identifier))
    case DatabaseTypeUser:
        return fmt.Sprintf("user_%s", identifier)
    default:
        return "shared"
    }
}

// hashIdentifier generates SHA256 hash (first 8 chars)
func hashIdentifier(identifier string) string {
    h := sha256.Sum256([]byte(identifier))
    return fmt.Sprintf("%x", h)[:8]
}

// ValidateDatabaseName checks naming conventions
func ValidateDatabaseName(dbName string) error {
    if len(dbName) == 0 {
        return ErrInvalidDatabaseName
    }
    if len(dbName) > 64 {
        return ErrInvalidDatabaseName
    }
    // Check format: shared, project_*, user_*
    if dbName == "shared" {
        return nil
    }
    if strings.HasPrefix(dbName, "project_") {
        return nil
    }
    if strings.HasPrefix(dbName, "user_") {
        return nil
    }
    return ErrInvalidDatabaseName
}
```

---

## Qdrant Adapter Implementation

### Database Operations (Collection Prefix Strategy)

**Location**: `pkg/vectorstore/adapter/qdrant/database.go`

**Strategy**: Qdrant doesn't have native databases, so we use collection prefixes (`<database>__<collection>`)

```go
func (a *QdrantAdapter) CreateDatabase(ctx context.Context, db vectorstore.Database) error {
    // Qdrant: Database creation is implicit (no-op)
    // Collections will be created with prefix: <db.Name>__<collection>
    return nil
}

func (a *QdrantAdapter) ListDatabases(ctx context.Context, filter vectorstore.DatabaseType) ([]vectorstore.Database, error) {
    // List all collections
    collections, err := a.client.ListCollections(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list collections: %w", err)
    }

    // Extract unique database names from collection prefixes
    dbMap := make(map[string]bool)
    for _, coll := range collections {
        dbName, _ := parseCollectionName(coll.Name)
        if filter == "" || matchesDatabaseType(dbName, filter) {
            dbMap[dbName] = true
        }
    }

    // Convert to Database slice
    databases := make([]vectorstore.Database, 0, len(dbMap))
    for dbName := range dbMap {
        databases = append(databases, vectorstore.Database{
            Name: dbName,
            Type: vectorstore.ParseDatabaseType(dbName),
        })
    }

    return databases, nil
}

// parseCollectionName extracts database and collection from physical name
// Format: <database>__<collection>
func parseCollectionName(physicalName string) (dbName, collName string) {
    parts := strings.SplitN(physicalName, "__", 2)
    if len(parts) == 2 {
        return parts[0], parts[1]
    }
    return "shared", physicalName
}

// buildCollectionName creates physical collection name
func buildCollectionName(dbName, collName string) string {
    return fmt.Sprintf("%s__%s", dbName, collName)
}
```

### Collection Operations

**Location**: `pkg/vectorstore/adapter/qdrant/collection.go`

```go
func (a *QdrantAdapter) CreateCollection(ctx context.Context, dbName, collName string, schema vectorstore.CollectionSchema) error {
    // Build physical collection name with database prefix
    physicalName := buildCollectionName(dbName, collName)

    // Create Qdrant collection
    err := a.client.CreateCollection(ctx, &qdrant.CreateCollection{
        CollectionName: physicalName,
        VectorsConfig: qdrant.VectorsConfig{
            Params: &qdrant.VectorParams{
                Size:     uint64(schema.VectorDim),
                Distance: convertDistanceMetric(schema.DistanceMetric),
            },
        },
    })

    if err != nil {
        return fmt.Errorf("failed to create collection: %w", err)
    }

    return nil
}

func (a *QdrantAdapter) ListCollections(ctx context.Context, dbName string) ([]string, error) {
    // List all collections
    collections, err := a.client.ListCollections(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list collections: %w", err)
    }

    // Filter by database prefix
    prefix := dbName + "__"
    result := make([]string, 0)
    for _, coll := range collections {
        if strings.HasPrefix(coll.Name, prefix) {
            // Extract logical collection name (remove prefix)
            _, collName := parseCollectionName(coll.Name)
            result = append(result, collName)
        }
    }

    return result, nil
}
```

### Vector Operations

**Location**: `pkg/vectorstore/adapter/qdrant/adapter.go`

```go
func (a *QdrantAdapter) Insert(ctx context.Context, dbName, collName string, vectors []vectorstore.Vector) error {
    // Build physical collection name
    physicalName := buildCollectionName(dbName, collName)

    // Convert to Qdrant points
    points := make([]*qdrant.PointStruct, len(vectors))
    for i, v := range vectors {
        points[i] = &qdrant.PointStruct{
            Id:      qdrant.NewIDString(v.ID),
            Vectors: qdrant.NewVectors(v.Embedding...),
            Payload: convertPayload(v.Payload),
        }
    }

    // Upsert to Qdrant
    _, err := a.client.Upsert(ctx, &qdrant.UpsertPoints{
        CollectionName: physicalName,
        Points:         points,
    })

    if err != nil {
        return fmt.Errorf("failed to insert vectors: %w", err)
    }

    return nil
}

func (a *QdrantAdapter) Search(ctx context.Context, dbName, collName string, query vectorstore.SearchQuery) ([]vectorstore.SearchResult, error) {
    // Build physical collection name
    physicalName := buildCollectionName(dbName, collName)

    // Search in Qdrant
    results, err := a.client.Search(ctx, &qdrant.SearchPoints{
        CollectionName: physicalName,
        Vector:         query.Vector,
        Limit:          uint64(query.TopK),
        Filter:         convertFilter(query.Filter),
    })

    if err != nil {
        return nil, fmt.Errorf("failed to search: %w", err)
    }

    // Convert to SearchResult
    searchResults := make([]vectorstore.SearchResult, len(results))
    for i, r := range results {
        searchResults[i] = vectorstore.SearchResult{
            ID:      r.Id.GetString(),
            Score:   r.Score,
            Payload: convertPayloadToMap(r.Payload),
        }
    }

    return searchResults, nil
}
```

---

## Migration Implementation

### Migration Tool

**Location**: `cmd/contextd/migrate.go`

```go
type MigrationService struct {
    store  vectorstore.UniversalVectorStore
    legacy vectorstore.VectorStore  // Legacy flat structure
}

func (m *MigrationService) Analyze(ctx context.Context) (*AnalysisReport, error) {
    // Query legacy database for unique project_path values
    results, err := m.legacy.Search(ctx, "default", "checkpoints", vectorstore.SearchQuery{
        TopK: 10000,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to query legacy: %w", err)
    }

    // Count vectors per project
    projectCounts := make(map[string]int)
    for _, r := range results {
        projectPath := r.Payload["project_path"].(string)
        projectCounts[projectPath]++
    }

    return &AnalysisReport{
        TotalVectors:   len(results),
        UniqueProjects: len(projectCounts),
        ProjectCounts:  projectCounts,
    }, nil
}

func (m *MigrationService) CreateDatabases(ctx context.Context, projectPaths []string) error {
    // Create shared database
    err := m.store.CreateDatabase(ctx, vectorstore.Database{
        Name: "shared",
        Type: vectorstore.DatabaseTypeShared,
    })
    if err != nil {
        return fmt.Errorf("failed to create shared database: %w", err)
    }

    // Create project databases
    for _, path := range projectPaths {
        dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, path)
        err := m.store.CreateDatabase(ctx, vectorstore.Database{
            Name: dbName,
            Type: vectorstore.DatabaseTypeProject,
            Metadata: map[string]string{
                "project_path": path,
                "created_at":   time.Now().Format(time.RFC3339),
            },
        })
        if err != nil {
            return fmt.Errorf("failed to create database %s: %w", dbName, err)
        }
    }

    return nil
}

func (m *MigrationService) MigrateData(ctx context.Context) error {
    // Query all vectors from legacy database
    results, err := m.legacy.Search(ctx, "default", "checkpoints", vectorstore.SearchQuery{
        TopK: 10000,
    })
    if err != nil {
        return fmt.Errorf("failed to query legacy: %w", err)
    }

    // Group vectors by project_path
    projectVectors := make(map[string][]vectorstore.Vector)
    for _, r := range results {
        projectPath := r.Payload["project_path"].(string)
        projectVectors[projectPath] = append(projectVectors[projectPath], vectorstore.Vector{
            ID:        r.ID,
            Embedding: r.Embedding,
            Payload:   r.Payload,
        })
    }

    // Insert vectors to project databases
    for projectPath, vectors := range projectVectors {
        dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)
        err := m.store.Insert(ctx, dbName, "checkpoints", vectors)
        if err != nil {
            return fmt.Errorf("failed to migrate project %s: %w", projectPath, err)
        }
    }

    return nil
}
```

---

## Service Integration

### Checkpoint Service

**Location**: `pkg/checkpoint/service.go`

```go
type Service struct {
    store    vectorstore.UniversalVectorStore
    embedder embedding.Service
}

func (s *Service) Save(ctx context.Context, projectPath string, cp *Checkpoint) error {
    // Generate embedding
    embedding, err := s.embedder.Embed(ctx, cp.Summary)
    if err != nil {
        return fmt.Errorf("failed to embed: %w", err)
    }

    // Get project database
    dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

    // Insert vector
    vector := vectorstore.Vector{
        ID:        cp.ID,
        Embedding: embedding,
        Payload: map[string]interface{}{
            "summary":    cp.Summary,
            "content":    cp.Content,
            "created_at": cp.CreatedAt.Unix(),
        },
    }

    err = s.store.Insert(ctx, dbName, "checkpoints", []vectorstore.Vector{vector})
    if err != nil {
        return fmt.Errorf("failed to insert: %w", err)
    }

    return nil
}

func (s *Service) Search(ctx context.Context, projectPath, query string, topK int) ([]*Checkpoint, error) {
    // Generate query embedding
    embedding, err := s.embedder.Embed(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to embed: %w", err)
    }

    // Get project database
    dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

    // Search (no filter needed - physical isolation!)
    results, err := s.store.Search(ctx, dbName, "checkpoints", vectorstore.SearchQuery{
        Vector: embedding,
        TopK:   topK,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to search: %w", err)
    }

    // Convert to Checkpoint
    checkpoints := make([]*Checkpoint, len(results))
    for i, r := range results {
        checkpoints[i] = &Checkpoint{
            ID:      r.ID,
            Summary: r.Payload["summary"].(string),
            Content: r.Payload["content"].(string),
            Score:   r.Score,
        }
    }

    return checkpoints, nil
}
```

---

## Error Handling

### Error Definitions

**Location**: `pkg/vectorstore/errors.go`

```go
var (
    // Database errors
    ErrDatabaseNotFound      = errors.New("database not found")
    ErrDatabaseAlreadyExists = errors.New("database already exists")
    ErrDatabaseLimit         = errors.New("database limit reached")
    ErrInvalidDatabaseName   = errors.New("invalid database name")

    // Collection errors
    ErrCollectionNotFound      = errors.New("collection not found")
    ErrCollectionAlreadyExists = errors.New("collection already exists")
    ErrInvalidCollectionName   = errors.New("invalid collection name")

    // Migration errors
    ErrMigrationInProgress = errors.New("migration already in progress")
    ErrMigrationFailed     = errors.New("migration failed")
    ErrRollbackRequired    = errors.New("migration failed, rollback required")
    ErrDataLoss            = errors.New("potential data loss detected")
)
```

---

## Configuration

### Multi-Tenant Settings

**Location**: `pkg/config/config.go`

```go
type Config struct {
    Database DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
    MultiTenant MultiTenantConfig `yaml:"multi_tenant"`
    Limits      LimitsConfig      `yaml:"limits"`
}

type MultiTenantConfig struct {
    Enabled          bool   `yaml:"enabled"`           // v2.0+: ALWAYS true
    ProjectHashAlgo  string `yaml:"project_hash_algo"` // sha256, sha512
    DatabasePrefix   string `yaml:"database_prefix"`   // "project_"
}

type LimitsConfig struct {
    MaxDatabases              int `yaml:"max_databases"`                // 100
    MaxCollectionsPerDB       int `yaml:"max_collections_per_db"`       // 100
    MaxVectorsPerCollection   int `yaml:"max_vectors_per_collection"`   // 1000000
}
```

---

## Summary

**Implementation Highlights**:

1. **UniversalVectorStore Interface**: Database-agnostic abstraction
2. **Qdrant Adapter**: Collection prefix strategy (`<db>__<collection>`)
3. **Naming Helpers**: SHA256 hash-based database naming
4. **Migration Tool**: Analyze, create, migrate, validate, cleanup
5. **Service Integration**: Checkpoint, remediation services use new API
6. **Error Handling**: Comprehensive error types and wrapping
7. **Configuration**: Multi-tenant settings and limits

**Files Modified**:
- `pkg/vectorstore/` - Core abstraction layer
- `pkg/vectorstore/adapter/qdrant/` - Qdrant adapter
- `pkg/checkpoint/` - Checkpoint service
- `pkg/remediation/` - Remediation service
- `cmd/contextd/migrate.go` - Migration tool
- `pkg/config/` - Configuration

**Test Coverage**: ≥80% overall, 100% for core operations
