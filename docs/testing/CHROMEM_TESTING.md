# Chromem Testing Best Practices

## Critical Requirement: Valid Filesystem Paths

**⚠️ NEVER use empty string `""` for chromem Path configuration in tests.**

### Problem

Chromem's `NewPersistentDB()` requires a valid filesystem path for persistent storage. Using an empty string causes:

```
Error: collection metadata file not found: /Users/user/.config/contextd/vectorstore/xxx
```

### Solution

Always use `t.TempDir()` for test chromem stores:

```go
// ✅ CORRECT
store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
    Path: t.TempDir(),  // Auto-cleanup temp directory
    DefaultCollection: "test",
    VectorSize: 384,
}, embedder, logger)

// ❌ WRONG
store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
    Path: "",  // Will fail with "collection metadata file not found"
    DefaultCollection: "test",
    VectorSize: 384,
}, embedder, logger)
```

### Benefits of `t.TempDir()`

1. **Automatic cleanup** - Test harness removes directory after test
2. **Test isolation** - Each test gets unique storage
3. **No state leakage** - Tests don't interfere with each other
4. **No manual cleanup** - No need for `defer os.RemoveAll()`

### Test Helper Pattern

For repeated chromem store creation, use test helpers:

```go
// testhelpers_test.go (package internal tests)
func createTestChromemStore(t *testing.T, name string) (*ChromemStore, *MockEmbedder) {
    t.Helper()

    embedder := &MockEmbedder{
        embedding: make([]float32, 384),
    }

    config := ChromemConfig{
        Path:              t.TempDir(),
        DefaultCollection: "test_" + name,
        VectorSize:        384,
    }

    store, err := NewChromemStore(config, embedder, zap.NewNop())
    require.NoError(t, err)

    t.Cleanup(func() {
        store.Close()
    })

    return store, embedder
}
```

### Common Patterns

#### Integration Tests

```go
func TestIntegration(t *testing.T) {
    embedder := newTestEmbedder(384)
    store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
        Path: t.TempDir(),  // ✅ Always use t.TempDir()
    }, embedder, logger)
    require.NoError(t, err)
    defer store.Close()

    // Test code...
}
```

#### Developer Simulator

```go
type Developer struct {
    tempDir     string  // Store temp path for cleanup
    vectorStore vectorstore.Store
}

func (d *Developer) StartContextd(ctx context.Context) error {
    tempDir, err := os.MkdirTemp("", "contextd-test-*")
    if err != nil {
        return err
    }
    d.tempDir = tempDir

    store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
        Path: tempDir,  // ✅ Valid filesystem path
    }, embedder, logger)
    if err != nil {
        os.RemoveAll(tempDir)
        return err
    }

    d.vectorStore = store
    return nil
}

func (d *Developer) StopContextd(ctx context.Context) error {
    if d.vectorStore != nil {
        d.vectorStore.Close()
    }

    if d.tempDir != "" {
        os.RemoveAll(d.tempDir)  // Cleanup
        d.tempDir = ""
    }

    return nil
}
```

## Historical Context

This pattern emerged from fixing integration test failures where chromem was initialized with `Path: ""`:

- **Checkpoint tests**: TokenCount values were 0 instead of expected values
- **Confidence calibration tests**: Collection metadata file not found
- **Developer simulator tests**: Collection metadata file not found
- **Debug tests**: Collection metadata file not found

All failures traced to the same root cause: chromem requires valid filesystem paths.

## See Also

- `/Users/dahendel/projects/fyrsmithlabs/contextd/internal/vectorstore/testhelpers_test.go` - Test helper implementations
- `/Users/dahendel/projects/fyrsmithlabs/contextd/test/integration/framework/developer.go` - Developer simulator pattern
