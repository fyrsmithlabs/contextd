# Qdrant Client

Production-ready Qdrant gRPC client implementation for contextd.

## Overview

The `internal/qdrant` package provides a robust, production-grade client for interacting with Qdrant vector database via gRPC. It implements the `Client` interface defined in `client.go` using the official Qdrant Go SDK.

## Features

- **gRPC Transport**: Native gRPC connection (port 6334) for better performance than HTTP REST
- **Automatic Retry**: Configurable retry logic with exponential backoff for transient failures
- **Connection Pooling**: Efficient connection management via gRPC channels
- **Type Safety**: Strongly typed operations with proper error handling
- **Flexible Configuration**: Extensive configuration options with sensible defaults
- **Health Checks**: Automatic health verification on startup
- **TLS Support**: Optional TLS encryption for production deployments

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/fyrsmithlabs/contextd/internal/qdrant"
)

func main() {
    // Create client with default configuration (localhost:6334)
    client, err := qdrant.NewGRPCClient(nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Create a collection
    err = client.CreateCollection(ctx, "my_collection", 384)
    if err != nil {
        log.Fatal(err)
    }

    // Upsert points
    points := []*qdrant.Point{
        {
            ID:     "doc1",
            Vector: []float32{0.1, 0.2, 0.3, /* ... 384 dimensions */},
            Payload: map[string]interface{}{
                "title": "Example document",
                "tags":  "important",
            },
        },
    }

    err = client.Upsert(ctx, "my_collection", points)
    if err != nil {
        log.Fatal(err)
    }

    // Search for similar vectors
    queryVector := []float32{0.15, 0.25, 0.35 /* ... */}
    results, err := client.Search(ctx, "my_collection", queryVector, 10, nil)
    if err != nil {
        log.Fatal(err)
    }

    for _, result := range results {
        log.Printf("ID: %s, Score: %.4f", result.ID, result.Score)
    }
}
```

## Configuration

### ClientConfig Options

```go
config := &qdrant.ClientConfig{
    // Server connection
    Host:   "localhost",  // Qdrant server hostname
    Port:   6334,         // gRPC port (NOT 6333 which is HTTP REST)
    UseTLS: false,        // Enable TLS for production
    APIKey: "",           // Optional API key for authentication

    // Performance tuning
    MaxMessageSize: 50 * 1024 * 1024,  // 50MB max message size
    DialTimeout:    5 * time.Second,    // Connection timeout
    RequestTimeout: 30 * time.Second,   // Request timeout

    // Reliability
    RetryAttempts: 3,  // Number of retry attempts for transient failures

    // Vector configuration
    Distance: qdrant.Distance_Cosine,  // Default distance metric
}

client, err := qdrant.NewGRPCClient(config)
```

### Default Configuration

```go
// Use defaults for local development
client, err := qdrant.NewGRPCClient(nil)

// Defaults:
// - Host: "localhost"
// - Port: 6334
// - UseTLS: false
// - MaxMessageSize: 50MB
// - DialTimeout: 5s
// - RequestTimeout: 30s
// - RetryAttempts: 3
// - Distance: Cosine
```

### Production Configuration

```go
config := &qdrant.ClientConfig{
    Host:           "qdrant.prod.example.com",
    Port:           6334,
    UseTLS:         true,
    APIKey:         os.Getenv("QDRANT_API_KEY"),
    MaxMessageSize: 100 * 1024 * 1024,  // 100MB for large batches
    DialTimeout:    10 * time.Second,
    RequestTimeout: 60 * time.Second,
    RetryAttempts:  5,
}

client, err := qdrant.NewGRPCClient(config)
```

## Client Operations

### Collection Management

```go
// Create collection
err := client.CreateCollection(ctx, "embeddings", 384)

// Check if collection exists
exists, err := client.CollectionExists(ctx, "embeddings")

// List all collections
collections, err := client.ListCollections(ctx)

// Delete collection
err := client.DeleteCollection(ctx, "embeddings")
```

### Point Operations

#### Upsert Points

```go
points := []*qdrant.Point{
    {
        ID:     "uuid-1",
        Vector: []float32{0.1, 0.2, 0.3},
        Payload: map[string]interface{}{
            "text":       "Example document",
            "category":   "documentation",
            "confidence": 0.95,
            "active":     true,
        },
    },
}

err := client.Upsert(ctx, "my_collection", points)
```

#### Search with Filters

```go
queryVector := []float32{0.15, 0.25, 0.35}

filter := &qdrant.Filter{
    Must: []qdrant.Condition{
        {
            Field: "category",
            Match: "documentation",
        },
        {
            Field: "confidence",
            Range: &qdrant.RangeCondition{
                Gte: ptrFloat64(0.8),
            },
        },
    },
}

results, err := client.Search(ctx, "my_collection", queryVector, 10, filter)
```

#### Get Points by ID

```go
ids := []string{"uuid-1", "uuid-2", "uuid-3"}
points, err := client.Get(ctx, "my_collection", ids)

for _, point := range points {
    log.Printf("ID: %s, Payload: %+v", point.ID, point.Payload)
}
```

#### Delete Points

```go
ids := []string{"uuid-1", "uuid-2"}
err := client.Delete(ctx, "my_collection", ids)
```

### Health Check

```go
err := client.Health(ctx)
if err != nil {
    log.Printf("Qdrant is unhealthy: %v", err)
}
```

## Error Handling

The client distinguishes between transient and permanent errors:

### Transient Errors (Automatically Retried)
- `codes.Unavailable` - Service temporarily unavailable
- `codes.DeadlineExceeded` - Request timeout
- `codes.Aborted` - Request aborted
- `codes.ResourceExhausted` - Rate limiting

### Permanent Errors (Not Retried)
- `codes.InvalidArgument` - Bad request
- `codes.NotFound` - Collection/point not found
- `codes.PermissionDenied` - Authorization failed
- `codes.AlreadyExists` - Resource already exists

```go
err := client.CreateCollection(ctx, "test", 384)
if err != nil {
    // Check for specific error types
    st, ok := status.FromError(err)
    if ok {
        switch st.Code() {
        case codes.AlreadyExists:
            log.Println("Collection already exists")
        case codes.InvalidArgument:
            log.Println("Invalid vector size")
        default:
            log.Printf("Unexpected error: %v", err)
        }
    }
}
```

## Integration with contextd Services

### Using with Checkpoint Service

```go
import (
    "github.com/fyrsmithlabs/contextd/internal/checkpoint"
    "github.com/fyrsmithlabs/contextd/internal/qdrant"
)

// Create Qdrant client
qdrantClient, err := qdrant.NewGRPCClient(nil)
if err != nil {
    log.Fatal(err)
}
defer qdrantClient.Close()

// Create checkpoint service
checkpointConfig := checkpoint.DefaultServiceConfig()
checkpointService, err := checkpoint.NewService(
    checkpointConfig,
    qdrantClient,
    logger,
)
```

### Using with Remediation Service

```go
import (
    "github.com/fyrsmithlabs/contextd/internal/remediation"
    "github.com/fyrsmithlabs/contextd/internal/qdrant"
)

// Create Qdrant client
qdrantClient, err := qdrant.NewGRPCClient(nil)
if err != nil {
    log.Fatal(err)
}
defer qdrantClient.Close()

// Create remediation service (needs embedder)
remediationService, err := remediation.NewService(
    remediationConfig,
    qdrantClient,
    embedder,
    logger,
)
```

## Connection Lifecycle

### Startup
1. Client validates configuration
2. Establishes gRPC connection
3. Performs health check
4. Returns ready-to-use client or error

### Runtime
- Connection is maintained via gRPC channel
- Automatic reconnection on transient failures
- Exponential backoff for retries

### Shutdown
```go
// Always close the client when done
defer client.Close()

// Or explicitly
err := client.Close()
if err != nil {
    log.Printf("Error closing client: %v", err)
}
```

## Testing

### Run Unit Tests

```bash
go test ./internal/qdrant/... -v
```

### Run with Coverage

```bash
go test ./internal/qdrant/... -cover
```

### Integration Testing

For integration tests with a real Qdrant instance:

```bash
# Start Qdrant with Docker
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant

# Run integration tests
go test ./internal/qdrant/... -tags=integration
```

## Performance Considerations

### Batch Operations
- Upsert points in batches of 100-1000 for best performance
- Larger batches reduce network overhead but increase latency

### Connection Pooling
- gRPC client automatically pools connections
- No need for manual connection management

### Message Size
- Default 50MB max message size
- Increase for large document batches
- Consider chunking very large operations

### Timeouts
- DialTimeout: Connection establishment (default 5s)
- RequestTimeout: Individual operations (default 30s)
- Adjust based on network latency and data size

## Troubleshooting

### Connection Refused
```
Error: connection refused
```
**Solution**: Ensure Qdrant is running and accessible on port 6334 (gRPC, not 6333 HTTP)

### Health Check Failed
```
Error: health check failed
```
**Solution**: Verify Qdrant is running and responsive. Check logs for startup errors.

### Collection Already Exists
```
Error: rpc error: code = AlreadyExists
```
**Solution**: Use `CollectionExists` before creating, or handle the error gracefully.

### Vector Dimension Mismatch
```
Error: rpc error: code = InvalidArgument desc = wrong vector dimension
```
**Solution**: Ensure all vectors match the collection's configured dimension.

## Migration from Stub

If upgrading from the stub implementation:

```go
// Old (stub)
// var client qdrant.Client  // was nil or mock

// New (production)
client, err := qdrant.NewGRPCClient(nil)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

All interface methods remain the same - just instantiate the real client instead of a stub.

## References

- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [Qdrant Go Client SDK](https://github.com/qdrant/go-client)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)

## License

Part of contextd project. See main repository LICENSE.
