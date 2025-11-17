# Qdrant Vector Database Specification

## Official References

- **Qdrant Docs**: https://qdrant.tech/documentation/
- **Go Client**: https://github.com/qdrant/go-client
- **API Reference**: https://qdrant.github.io/qdrant/redoc/index.html
- **Best Practices**: https://qdrant.tech/documentation/guides/

## Core Concepts

### Collections
- Container for vectors and payloads
- Schema-free (flexible payloads)
- Support for multiple vector fields per point
- HNSW index by default

### Points
- **ID**: Unique identifier (uint64 or UUID)
- **Vector**: Dense or sparse vector (single or multiple)
- **Payload**: JSON-like metadata (flexible schema)

### Indexes
- **HNSW**: Default, balanced speed/accuracy
- **Plain**: Exact search, no index overhead
- Automatic index optimization

### Distance Metrics
- **Cosine**: Cosine similarity (normalized vectors)
- **Euclid**: Euclidean distance (L2)
- **Dot**: Dot product (for normalized vectors)
- **Manhattan**: L1 distance

## Go Client Patterns

### Connection

```go
import (
    "context"
    "github.com/qdrant/go-client/qdrant"
)

// Connect to Qdrant
client, err := qdrant.NewClient(&qdrant.Config{
    Host: "localhost",
    Port: 6334,
    APIKey: "", // Optional
    UseTLS: false,
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Check connection
ctx := context.Background()
collections, err := client.ListCollections(ctx)
```

### Collection Creation

```go
// Create collection with vector config
err := client.CreateCollection(ctx, &qdrant.CreateCollection{
    CollectionName: "checkpoints",
    VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
        Size:     1536,              // Dimension (OpenAI ada-002)
        Distance: qdrant.Distance_Cosine,
        OnDisk:   qdrant.PtrOf(false), // Keep in memory
    }),
    HnswConfig: &qdrant.HnswConfigDiff{
        M:              qdrant.PtrOf(uint64(16)),  // Connections per layer
        EfConstruct:    qdrant.PtrOf(uint64(100)), // Build quality
        FullScanThreshold: qdrant.PtrOf(uint64(10000)),
    },
    OptimizersConfig: &qdrant.OptimizersConfigDiff{
        IndexingThreshold: qdrant.PtrOf(uint64(20000)),
    },
})
```

### Multiple Vectors per Point

```go
// For multi-vector scenarios (e.g., title + content embeddings)
vectorsConfig := qdrant.NewVectorsConfigMap(map[string]*qdrant.VectorParams{
    "title": {
        Size:     384,  // Smaller model for titles
        Distance: qdrant.Distance_Cosine,
    },
    "content": {
        Size:     1536, // Full model for content
        Distance: qdrant.Distance_Cosine,
    },
})

err := client.CreateCollection(ctx, &qdrant.CreateCollection{
    CollectionName: "documents",
    VectorsConfig:  vectorsConfig,
})
```

### Insert Points

```go
// Prepare points
points := []*qdrant.PointStruct{
    {
        Id: qdrant.NewIDNum(1),
        Vectors: qdrant.NewVectors([]float32{0.1, 0.2, 0.3}), // Your embedding
        Payload: qdrant.NewValueMap(map[string]any{
            "summary":   "Checkpoint summary",
            "project":   "contextd",
            "timestamp": time.Now().Unix(),
            "tags":      []string{"feature", "auth"},
        }),
    },
}

// Upsert (insert or update)
operation, err := client.Upsert(ctx, &qdrant.UpsertPoints{
    CollectionName: "checkpoints",
    Points:         points,
    Wait:           qdrant.PtrOf(true), // Wait for operation to complete
})
```

### Batch Insert

```go
const batchSize = 100

for i := 0; i < len(allPoints); i += batchSize {
    end := min(i+batchSize, len(allPoints))
    batch := allPoints[i:end]

    _, err := client.Upsert(ctx, &qdrant.UpsertPoints{
        CollectionName: "checkpoints",
        Points:         batch,
        Wait:           qdrant.PtrOf(false), // Async for performance
    })
    if err != nil {
        return err
    }
}

// Wait for indexing
client.WaitForCollection(ctx, "checkpoints")
```

### Search

```go
// Basic semantic search
results, err := client.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "checkpoints",
    Query:          qdrant.NewQuery(queryVector...),
    Limit:          qdrant.PtrOf(uint64(10)),
    WithPayload:    qdrant.NewWithPayload(true),
})

// Process results
for _, point := range results {
    id := point.Id
    score := point.Score
    payload := point.Payload

    summary := payload["summary"].GetStringValue()
    project := payload["project"].GetStringValue()
}
```

### Filtered Search

```go
// Search with filters
results, err := client.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "checkpoints",
    Query:          qdrant.NewQuery(queryVector...),
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            qdrant.NewMatch("project", "contextd"),
            qdrant.NewRange("timestamp", &qdrant.Range{
                Gte: qdrant.PtrOf(float64(timestampAfter)),
            }),
        },
    },
    Limit:       qdrant.PtrOf(uint64(10)),
    WithPayload: qdrant.NewWithPayload(true),
})
```

### Scroll (Pagination)

```go
// Scroll through all points
var offset *qdrant.PointId
const limit = 100

for {
    results, err := client.Scroll(ctx, &qdrant.ScrollPoints{
        CollectionName: "checkpoints",
        Limit:          qdrant.PtrOf(uint32(limit)),
        Offset:         offset,
        WithPayload:    qdrant.NewWithPayload(true),
    })
    if err != nil {
        return err
    }

    // Process results
    for _, point := range results.Result {
        process(point)
    }

    // Check if done
    if results.NextPageOffset == nil {
        break
    }
    offset = results.NextPageOffset
}
```

## Performance Optimization

### HNSW Parameters

```go
// Development: Balanced
M = 16
efConstruct = 100

// Production (speed): Lower accuracy, faster
M = 8
efConstruct = 64

// Production (accuracy): Higher accuracy, slower
M = 32
efConstruct = 200
```

### Search Parameters

```go
// Query with custom search params
results, err := client.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "checkpoints",
    Query:          qdrant.NewQuery(queryVector...),
    Params: &qdrant.SearchParams{
        HnswEf:           qdrant.PtrOf(uint64(128)), // Search quality
        Exact:            qdrant.PtrOf(false),       // Use HNSW
        IndexedOnly:      qdrant.PtrOf(false),       // Include non-indexed
    },
    Limit: qdrant.PtrOf(uint64(10)),
})
```

### Quantization (Memory Optimization)

```go
// Scalar quantization for memory savings
err := client.UpdateCollection(ctx, &qdrant.UpdateCollection{
    CollectionName: "checkpoints",
    QuantizationConfig: &qdrant.QuantizationConfig{
        Quantization: &qdrant.QuantizationConfig_Scalar{
            Scalar: &qdrant.ScalarQuantization{
                Type:      qdrant.QuantizationType_Int8,
                Quantile:  qdrant.PtrOf(float32(0.99)),
                AlwaysRam: qdrant.PtrOf(true),
            },
        },
    },
})
```

### On-Disk Storage

```go
// Move vectors to disk for large datasets
err := client.CreateCollection(ctx, &qdrant.CreateCollection{
    CollectionName: "large_collection",
    VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
        Size:     1536,
        Distance: qdrant.Distance_Cosine,
        OnDisk:   qdrant.PtrOf(true), // Store on disk
    }),
})
```

## Filter Expressions

### Basic Filters

```go
// Match exact value
qdrant.NewMatch("project", "contextd")

// Match keyword (for arrays)
qdrant.NewMatch("tags", "feature")

// Range filter
qdrant.NewRange("timestamp", &qdrant.Range{
    Gte: qdrant.PtrOf(float64(start)),
    Lt:  qdrant.PtrOf(float64(end)),
})

// Geo radius
qdrant.NewGeoRadius("location", &qdrant.GeoRadius{
    Center: &qdrant.GeoPoint{Lat: 52.52, Lon: 13.405},
    Radius: 1000.0, // meters
})
```

### Compound Filters

```go
filter := &qdrant.Filter{
    Must: []*qdrant.Condition{
        // AND conditions
        qdrant.NewMatch("project", "contextd"),
        qdrant.NewRange("timestamp", &qdrant.Range{
            Gte: qdrant.PtrOf(float64(timestampAfter)),
        }),
    },
    Should: []*qdrant.Condition{
        // OR conditions
        qdrant.NewMatch("priority", "high"),
        qdrant.NewMatch("priority", "critical"),
    },
    MustNot: []*qdrant.Condition{
        // NOT conditions
        qdrant.NewMatch("status", "archived"),
    },
}
```

### Nested Fields

```go
// Access nested payload fields
qdrant.NewMatch("metadata.category", "bug")
qdrant.NewRange("metadata.score", &qdrant.Range{
    Gte: qdrant.PtrOf(0.8),
})
```

## Payload Management

### Update Payload

```go
// Update specific fields
err := client.SetPayload(ctx, &qdrant.SetPayload{
    CollectionName: "checkpoints",
    Payload: qdrant.NewValueMap(map[string]any{
        "status": "completed",
        "updated_at": time.Now().Unix(),
    }),
    PointsSelector: &qdrant.PointsSelector{
        Points: &qdrant.PointsIdsList{
            Ids: []*qdrant.PointId{qdrant.NewIDNum(1)},
        },
    },
})
```

### Delete Payload Fields

```go
// Remove specific fields
err := client.DeletePayload(ctx, &qdrant.DeletePayload{
    CollectionName: "checkpoints",
    Keys:           []string{"temporary_field"},
    PointsSelector: &qdrant.PointsSelector{
        Points: &qdrant.PointsIdsList{
            Ids: []*qdrant.PointId{qdrant.NewIDNum(1)},
        },
    },
})
```

### Clear Payload

```go
// Remove all payload fields
err := client.ClearPayload(ctx, &qdrant.ClearPayload{
    CollectionName: "checkpoints",
    PointsSelector: &qdrant.PointsSelector{
        Points: &qdrant.PointsIdsList{
            Ids: []*qdrant.PointId{qdrant.NewIDNum(1)},
        },
    },
})
```

## Collection Management

### Get Collection Info

```go
info, err := client.GetCollection(ctx, "checkpoints")
fmt.Printf("Points count: %d\n", info.PointsCount)
fmt.Printf("Segments: %d\n", info.SegmentsCount)
fmt.Printf("Status: %s\n", info.Status)
```

### Update Collection

```go
// Update HNSW params
err := client.UpdateCollection(ctx, &qdrant.UpdateCollection{
    CollectionName: "checkpoints",
    HnswConfig: &qdrant.HnswConfigDiff{
        EfConstruct: qdrant.PtrOf(uint64(200)),
    },
})
```

### Delete Collection

```go
err := client.DeleteCollection(ctx, "checkpoints")
```

### Create Snapshot

```go
// Create backup snapshot
snapshot, err := client.CreateSnapshot(ctx, "checkpoints")
fmt.Printf("Snapshot: %s\n", snapshot.Name)
```

## Troubleshooting

### "collection not found"
```
Error: Collection 'checkpoints' not found
Fix: Create collection first with CreateCollection
Check: List collections to verify
```

### "dimension mismatch"
```
Error: Vector dimension 384 doesn't match collection (1536)
Fix: Ensure embedding model matches collection config
Note: Cannot change dimension after creation
```

### "slow search queries"
```
Check:
1. HNSW parameters (increase ef for accuracy)
2. Use filters to reduce search space
3. Consider quantization for memory
4. Check if vectors on disk (slower)
```

### "high memory usage"
```
Solutions:
1. Enable quantization (scalar or product)
2. Move vectors to disk (OnDisk: true)
3. Reduce HNSW M parameter
4. Use smaller batch sizes
```

### "indexing too slow"
```
Optimize:
1. Increase indexing_threshold
2. Reduce efConstruct during bulk import
3. Use async upserts (Wait: false)
4. Disable indexing during import, rebuild after
```


### When to Use Qdrant

**Advantages:**
- Simpler setup (single binary, no dependencies)
- Flexible schema (no pre-defined payload structure)
- Better filtering capabilities
- Built-in HTTP API
- Easier local development
- Automatic index optimization

**Use for:**
- Prototyping and development
- Simple deployments
- Flexible payload schemas
- Strong filtering requirements


**Advantages:**
- Better for massive scale (>10M vectors)
- More mature ecosystem
- Advanced partitioning
- Better for multi-tenancy

**Use for:**
- Production at scale
- Multi-tenant architectures
- Complex partitioning needs

## Best Practices

### ✅ DO

1. **Use HNSW** for >10k vectors (default)
2. **Batch inserts** (100-1000 points)
3. **Filter before search** to reduce search space
4. **Use quantization** for memory savings
5. **Wait for indexing** after bulk imports
6. **Monitor collection stats** (segments, points)
7. **Create snapshots** before major changes
8. **Use appropriate distance** metric (Cosine for normalized)
9. **Leverage flexible payloads** (no schema migrations)
10. **Test locally** (easy Docker setup)

### ❌ DON'T

1. **Insert one point at a time** (slow)
2. **Use Plain index** for large datasets
3. **Skip waiting** after bulk imports
4. **Ignore memory usage** (use quantization)
5. **Query without filters** on large collections
6. **Use wrong distance** metric (check embedding model)
7. **Forget to close client** (resource leak)
8. **Store large binary data** in payload (use external storage)
9. **Over-tune HNSW** without testing
10. **Skip backups** (use snapshots)

## Example: Contextd Integration

### Checkpoint Collection

```go
// Create collection
err := client.CreateCollection(ctx, &qdrant.CreateCollection{
    CollectionName: "checkpoints",
    VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
        Size:     1536,
        Distance: qdrant.Distance_Cosine,
    }),
})

// Insert checkpoint
point := &qdrant.PointStruct{
    Id: qdrant.NewIDUUID(uuid.New().String()),
    Vectors: qdrant.NewVectors(embedding),
    Payload: qdrant.NewValueMap(map[string]any{
        "summary":   summary,
        "content":   content,
        "project":   project,
        "timestamp": time.Now().Unix(),
        "tags":      tags,
    }),
}

client.Upsert(ctx, &qdrant.UpsertPoints{
    CollectionName: "checkpoints",
    Points:         []*qdrant.PointStruct{point},
})

// Search checkpoints
results, err := client.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "checkpoints",
    Query:          qdrant.NewQuery(queryEmbedding...),
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            qdrant.NewMatch("project", project),
            qdrant.NewRange("timestamp", &qdrant.Range{
                Gte: qdrant.PtrOf(float64(timestampAfter)),
            }),
        },
    },
    Limit:       qdrant.PtrOf(uint64(10)),
    WithPayload: qdrant.NewWithPayload(true),
})
```

---

**Reference**: Qdrant Documentation - https://qdrant.tech/documentation/
