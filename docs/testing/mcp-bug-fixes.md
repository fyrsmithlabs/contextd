# MCP Bug Fixes

**Parent**: [MCP_E2E_TEST_RESULTS.md](../../MCP_E2E_TEST_RESULTS.md)

## Issue #1: Checkpoint Qdrant Filter Syntax ✅

**GitHub**: [#1](https://github.com/fyrsmithlabs/contextd/issues/1)

**Error Message**:
```
unknown field 'project_hash', expected one of 'should', 'min_should', 'must', 'must_not'
```

**Affected Tools**:
- checkpoint_search
- checkpoint_list

**Root Cause**: Invalid Qdrant filter structure (flat map instead of nested)

**Fix Applied**:
```go
// BEFORE (pkg/checkpoint/service.go:148-152)
filters := map[string]interface{}{
    "project_hash": projectHash(opts.ProjectPath),
}

// AFTER (pkg/checkpoint/service.go:148-160)
filters := map[string]interface{}{
    "must": []map[string]interface{}{
        {
            "key": "project_hash",
            "match": map[string]interface{}{
                "value": projectHash(opts.ProjectPath),
            },
        },
    },
}
```

**Files Modified**:
- `pkg/checkpoint/service.go` - Filter syntax fix
- `pkg/checkpoint/service_test.go` - Filter validation tests

**Verification**:
- ✅ Tests pass: 87.2% coverage (target: 80%)
- ✅ Manual testing: checkpoint_search returns results
- ✅ Manual testing: checkpoint_list returns results

---

## Issue #2: Test Script Missing project_path ✅

**GitHub**: [#2](https://github.com/fyrsmithlabs/contextd/issues/2)

**Error Message**:
```
project_path is required
```

**Affected Tests**:
- remediation_save
- remediation_search

**Root Cause**: Test arguments didn't include required parameter

**Fix Applied**:
```bash
# BEFORE (line 120)
curl ... '{"name":"remediation_save","arguments":{"error_msg":"test error","solution":"test solution"}}'

# AFTER (line 120)
curl ... '{"name":"remediation_save","arguments":{"error_msg":"test error","solution":"test solution","project_path":"'$PROJECT_PATH'"}}'
```

**Files Modified**:
- `/tmp/test_all_mcp_tools.sh` - Lines 120, 127

**Verification**:
- ✅ Test script includes project_path
- ✅ remediation_save test passes
- ✅ remediation_search test passes

---

## Issue #3: Missing Qdrant Collection ✅

**GitHub**: [#3](https://github.com/fyrsmithlabs/contextd/issues/3)

**Error Message**:
```
Collection 'contextd' doesn't exist!
```

**Affected Tools**:
- checkpoint_search
- checkpoint_list
- remediation_save

**Root Cause**: Collection not created automatically on startup

**Fix Applied**:

**1. Added EnsureCollection() method** (`pkg/vectorstore/collections.go`):
```go
func (s *Service) EnsureCollection(ctx context.Context, collectionName string, vectorSize int) error {
    // Validate inputs
    if collectionName == "" {
        return fmt.Errorf("%w: collection name required", ErrInvalidConfig)
    }
    if vectorSize <= 0 {
        return fmt.Errorf("%w: got %d", ErrInvalidVectorSize, vectorSize)
    }

    // Check if exists (idempotent)
    exists, err := s.CollectionExists(ctx, collectionName)
    if err != nil {
        return fmt.Errorf("checking collection existence: %w", err)
    }
    if exists {
        return nil  // Already exists
    }

    // Create collection
    url := fmt.Sprintf("%s/collections/%s", s.config.URL, collectionName)
    body := map[string]interface{}{
        "vectors": map[string]interface{}{
            "size":     vectorSize,
            "distance": "Cosine",
        },
    }
    // ... HTTP PUT request
}
```

**2. Added startup initialization** (`cmd/contextd/main.go`):
```go
// Get vector size from embedding model
func getVectorSizeForModel(modelName string) int {
    switch {
    case strings.Contains(modelName, "bge-small"):
        return 384
    case strings.Contains(modelName, "bge-base"):
        return 768
    case strings.Contains(modelName, "openai"):
        return 1536
    default:
        return 384  // Safe default
    }
}

// Initialize collection
collectionName := "contextd"
vectorSize := getVectorSizeForModel(cfg.EmbeddingModel)
if err := vectorStore.EnsureCollection(ctx, collectionName, vectorSize); err != nil {
    logger.Fatal("Failed to ensure collection exists", zap.Error(err))
}
logger.Info("Collection verified", zap.String("collection", collectionName), zap.Int("vector_size", vectorSize))
```

**Files Modified**:
- `pkg/vectorstore/collections.go` - 81 lines added (EnsureCollection method)
- `pkg/vectorstore/collections_test.go` - 45 lines added (tests)
- `cmd/contextd/main.go` - Collection initialization on startup

**Verification**:
- ✅ Tests pass: vectorstore package
- ✅ Logs show "Collection verified" on startup
- ✅ Idempotent: Works on first run and subsequent restarts
- ✅ All affected tools now pass

---

## Issue #4: Remediation Qdrant Filter Syntax ✅

**GitHub**: [#4](https://github.com/fyrsmithlabs/contextd/issues/4)

**Error Message**:
```
unknown field 'project_path', expected one of 'should', 'min_should', 'must', 'must_not'
```

**Affected Tools**:
- remediation_search

**Root Cause**: Same invalid Qdrant filter structure as Issue #1

**Fix Applied**:
```go
// BEFORE (pkg/remediation/service.go:120-130)
filters := map[string]interface{}{
    "project_path": opts.ProjectPath,
}

// AFTER (pkg/remediation/service.go:120-130)
filters := map[string]interface{}{
    "must": []map[string]interface{}{
        {
            "key": "project_path",
            "match": map[string]interface{}{
                "value": opts.ProjectPath,
            },
        },
    },
}
```

**Files Modified**:
- `pkg/remediation/service.go` - Filter syntax fix (Search and List methods)
- `pkg/remediation/service_test.go` - Filter validation tests

**Verification**:
- ✅ Tests pass: 88.5% coverage (target: 80%)
- ✅ Manual testing: remediation_search returns results with scores
- ✅ Manual testing: Returns semantic_score, string_score, combined score

---

## Common Patterns Identified

### 1. Qdrant Filter Syntax
**Pattern**: Both checkpoint and remediation had identical filter bugs

**Lesson**: Search for similar code patterns when fixing bugs

**Fix Template**:
```go
filters := map[string]interface{}{
    "must": []map[string]interface{}{
        {
            "key": "field_name",
            "match": map[string]interface{}{
                "value": field_value,
            },
        },
    },
}
```

### 2. Infrastructure Initialization
**Pattern**: Application assumed infrastructure existed

**Lesson**: Always verify and auto-create infrastructure dependencies

**Fix Template**:
```go
// Check if exists
exists, err := checkExists(ctx, resource)
if err != nil {
    return fmt.Errorf("checking existence: %w", err)
}
if exists {
    return nil  // Idempotent
}

// Create if missing
if err := create(ctx, resource); err != nil {
    return fmt.Errorf("creating resource: %w", err)
}
```

### 3. Test Coverage Gaps
**Pattern**: No E2E tests caught these bugs before

**Lesson**: Comprehensive E2E testing is essential

**Action**: Created `/tmp/test_all_mcp_tools.sh` for systematic testing
