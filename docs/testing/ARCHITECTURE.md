# Framework Architecture

**Status**: Active Development
**Last Updated**: 2025-12-11

---

## Overview

The integration test framework simulates developers using contextd without requiring external services. All storage uses in-memory mocks.

```
┌─────────────────────────────────────────────────────────────┐
│                        Test Suite                           │
│  (suite_a_policy_test.go, suite_c_bugfix_test.go, etc.)    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Developer Simulator                      │
│  RecordMemory() SearchMemory() SaveCheckpoint() Resume()    │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
┌──────────────────┐ ┌──────────────┐ ┌──────────────────┐
│  ReasoningBank   │ │  Checkpoint  │ │     Scrubber     │
│    Service       │ │   Service    │ │   (gitleaks)     │
└──────────────────┘ └──────────────┘ └──────────────────┘
              │               │
              └───────┬───────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    SharedStore (Mock)                        │
│              In-memory vector store mock                     │
└─────────────────────────────────────────────────────────────┘
```

---

## Components

| Component | File | Purpose |
|-----------|------|---------|
| Developer | developer.go | Simulates a developer using contextd |
| SharedStore | developer.go | In-memory mock vector store |
| ReasoningBank | internal/reasoningbank/ | Memory storage and search |
| Checkpoint | internal/checkpoint/ | Session state persistence |
| Scrubber | internal/secrets/ | Secret removal (gitleaks) |
| TestMetrics | metrics.go | OpenTelemetry observability |

---

## Developer Simulator

The `Developer` struct wraps contextd services and exposes a simple API:

```go
type Developer struct {
    // Configuration
    id        string
    tenantID  string
    projectID string

    // Services
    reasoningBank     *reasoningbank.Service
    checkpointService checkpoint.Service
    scrubber          secrets.Scrubber

    // State
    vectorStore   vectorstore.Store
    sessionID     string
    stats         SessionStats
}
```

### Key Methods

| Method | What It Does |
|--------|--------------|
| `StartContextd(ctx)` | Initialize services, generate session ID |
| `StopContextd(ctx)` | Close services, clean up |
| `RecordMemory(ctx, record)` | Store memory (scrubs secrets first) |
| `SearchMemory(ctx, query, limit)` | Find memories (scrubs results) |
| `SaveCheckpoint(ctx, req)` | Save session state |
| `ResumeCheckpoint(ctx, id)` | Restore from checkpoint |
| `GiveFeedback(ctx, id, helpful, reason)` | Rate memory helpfulness |

### Automatic Secret Scrubbing

The Developer scrubs secrets at two points (defense-in-depth):

```go
// On storage
func (d *Developer) RecordMemory(ctx context.Context, record MemoryRecord) (string, error) {
    scrubbedTitle := d.scrubber.Scrub(record.Title).Scrubbed
    scrubbedContent := d.scrubber.Scrub(record.Content).Scrubbed
    // ... store scrubbed content
}

// On search (defense-in-depth)
func (d *Developer) SearchMemory(ctx context.Context, query string, limit int) ([]MemoryResult, error) {
    results, _ := d.reasoningBank.Search(ctx, ...)
    for _, r := range results {
        scrubbedTitle := d.scrubber.Scrub(r.Title).Scrubbed
        scrubbedContent := d.scrubber.Scrub(r.Content).Scrubbed
        // ...
    }
}
```

---

## SharedStore (Mock Vector Store)

The mock implements `vectorstore.Store` with in-memory storage:

```go
type mockVectorStore struct {
    collections map[string][]vectorstore.Document
    mu          sync.RWMutex
}
```

### Filter Support

The mock supports these filters in `SearchInCollection`:

| Filter | Type | Purpose |
|--------|------|---------|
| `id` | string | Exact match by document ID |
| `session_id` | string | Filter by session |
| `confidence.$gte` | float64 | Minimum confidence threshold |

```go
// Filter implementation
if idFilter, ok := filters["id"].(string); ok {
    if doc.Metadata["id"] != idFilter {
        shouldInclude = false
    }
}

if confFilter, ok := filters["confidence"].(map[string]interface{}); ok {
    if minConf, ok := confFilter["$gte"].(float64); ok {
        if doc.Metadata["confidence"].(float64) < minConf {
            shouldInclude = false
        }
    }
}
```

### Collection Isolation

Each `SharedStore` creates collections namespaced by ProjectID:

```go
sharedStore, _ := NewSharedStore(SharedStoreConfig{
    ProjectID: "test_project_a",  // Creates "test_project_a_memories"
})
```

This prevents cross-contamination between tests.

---

## TestMetrics (Observability)

The framework tracks operations via OpenTelemetry:

```go
type TestMetrics struct {
    // Counters
    testPassCounter metric.Int64Counter
    testFailCounter metric.Int64Counter

    // Histograms
    suiteDuration         metric.Float64Histogram
    memorySearchLatency   metric.Float64Histogram
    checkpointSaveLatency metric.Float64Histogram

    // Gauges (via callbacks)
    memoryHitRate       float64
    checkpointSuccessRate float64
    crossDevSearchRate    float64
}
```

### Usage in Tests

```go
metrics, _ := NewTestMetrics()

// Record search
start := time.Now()
results, _ := dev.SearchMemory(ctx, query, 5)
metrics.RecordMemorySearch(ctx, time.Since(start), len(results) > 0, isCrossDev)

// Get stats
stats := metrics.GetStats()
fmt.Printf("Hit rate: %.2f%%\n", stats.MemoryHitRate * 100)
```

---

## Extension Points

### Adding New Tests

1. Create `suite_X_name_test.go`
2. Follow the established pattern:

```go
func TestSuiteX_Name_Feature(t *testing.T) {
    t.Run("description of what is tested", func(t *testing.T) {
        // 1. Create SharedStore with unique ProjectID
        sharedStore, err := NewSharedStore(SharedStoreConfig{
            ProjectID: "test_project_x_feature",
        })
        require.NoError(t, err)
        defer sharedStore.Close()

        // 2. Create Developer
        dev, err := NewDeveloperWithStore(DeveloperConfig{...}, sharedStore)
        require.NoError(t, err)

        // 3. Start services
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        err = dev.StartContextd(ctx)
        require.NoError(t, err)
        defer dev.StopContextd(ctx)

        // 4. Test logic + assertions
    })
}
```

### Adding New Filter Types

Extend `mockVectorStore.SearchInCollection`:

```go
// Add new filter in SearchInCollection
if customFilter, ok := filters["custom_field"].(string); ok {
    docValue, _ := doc.Metadata["custom_field"].(string)
    if docValue != customFilter {
        shouldInclude = false
    }
}
```

### Adding New Developer Methods

1. Add method to `Developer` struct in `developer.go`
2. Update `SessionStats` if tracking is needed
3. Add corresponding activity if Temporal workflow integration is desired

---

## Anti-Patterns

### Sharing Store Without Unique ProjectIDs

```go
// BAD: Same ProjectID causes test interference
store1, _ := NewSharedStore(SharedStoreConfig{ProjectID: "shared"})
store2, _ := NewSharedStore(SharedStoreConfig{ProjectID: "shared"})
// Tests see each other's data
```

### Forgetting to Close Resources

```go
// BAD: Leaks resources
sharedStore, _ := NewSharedStore(...)
dev, _ := NewDeveloperWithStore(..., sharedStore)
dev.StartContextd(ctx)
// Missing: defer sharedStore.Close()
// Missing: defer dev.StopContextd(ctx)
```

### Skipping Threshold Assertions

```go
// BAD: Only checks existence
assert.GreaterOrEqual(t, len(results), 1)

// GOOD: Also validates quality
assert.GreaterOrEqual(t, len(results), 1)
if len(results) > 0 {
    assert.GreaterOrEqual(t, results[0].Confidence, 0.7)
}
```
