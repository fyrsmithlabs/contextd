# Repository Indexing Implementation

**Parent**: [../SPEC.md](../SPEC.md)

## API Specification

### MCP Tool: `index_repository`

#### Description

Index an existing repository or directory for semantic search. Creates searchable checkpoints from files matching include patterns while respecting exclude patterns and file size limits.

#### Input Schema

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Path to repository or directory to index (required)",
      "minLength": 1
    },
    "include_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "File patterns to include (e.g., ['*.md', '*.txt'])",
      "default": []
    },
    "exclude_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "File patterns to exclude (e.g., ['*.log', 'node_modules/**'])",
      "default": []
    },
    "max_file_size": {
      "type": "integer",
      "description": "Maximum file size in bytes (default: 1MB, max: 10MB)",
      "default": 1048576,
      "minimum": 0,
      "maximum": 10485760
    }
  },
  "required": ["path"]
}
```

#### Output Schema

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Repository path that was indexed"
    },
    "files_indexed": {
      "type": "integer",
      "description": "Number of files successfully indexed"
    },
    "include_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Include patterns used"
    },
    "exclude_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Exclude patterns used"
    },
    "max_file_size": {
      "type": "integer",
      "description": "Maximum file size applied (bytes)"
    },
    "indexed_at": {
      "type": "string",
      "format": "date-time",
      "description": "Timestamp when indexing completed"
    }
  }
}
```

#### Error Responses

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `validation_error` | 400 | Invalid input (path doesn't exist, invalid pattern, size too large) |
| `timeout_error` | 504 | Indexing exceeded 5-minute timeout |
| `internal_error` | 500 | Unexpected error during indexing |

---

## File Pattern Matching

### Pattern Syntax

Uses Go's `filepath.Match` syntax (glob-style):

| Pattern | Matches |
|---------|---------|
| `*.go` | All Go files in current directory |
| `**/*.go` | All Go files recursively (requires custom implementation) |
| `test_*.go` | Go files starting with "test_" |
| `main.go` | Exact filename match |
| `pkg/*/service.go` | service.go in any immediate subdirectory of pkg/ |

**Note**: Go's `filepath.Match` does NOT support `**` (recursive). The implementation matches patterns against both basename and full relative path to achieve recursive behavior.

### Include Patterns

**Behavior**:
- If **empty**: Include ALL files (subject to exclude patterns and size limit)
- If **specified**: Include ONLY files matching at least one pattern

**Matching logic**:
```go
if len(includePatterns) > 0 {
    included := false
    for _, pattern := range includePatterns {
        if matched, _ := filepath.Match(pattern, basename); matched {
            included = true
            break
        }
    }
    if !included {
        return nil // Skip file
    }
}
```

**Examples**:
```javascript
// Include only markdown files
include_patterns: ["*.md"]

// Include multiple types
include_patterns: ["*.go", "*.py", "*.js"]

// Include all (default)
include_patterns: []
```

### Exclude Patterns

**Behavior**:
- Files matching ANY exclude pattern are skipped
- Matched against both basename and full path
- Takes precedence over include patterns

**Matching logic**:
```go
for _, pattern := range excludePatterns {
    if matched, _ := filepath.Match(pattern, basename); matched {
        return nil // Skip file
    }
    if matched, _ := filepath.Match(pattern, fullPath); matched {
        return nil // Skip file
    }
}
```

**Examples**:
```javascript
// Exclude vendor and test files
exclude_patterns: ["vendor/**", "*_test.go"]

// Exclude binaries and logs
exclude_patterns: ["*.exe", "*.bin", "*.log", "*.tmp"]

// Exclude common directories
exclude_patterns: ["node_modules/**", ".git/**", "__pycache__/**"]
```

### Pattern Validation

Patterns are validated before indexing:

```go
for _, pattern := range includePatterns {
    if _, err := filepath.Match(pattern, "test"); err != nil {
        return ValidationError("invalid include pattern")
    }
}
```

Invalid patterns cause immediate error (before any indexing).

### Common Patterns

#### Exclude Sensitive Files
```javascript
exclude_patterns: [
  "*.env",
  "*.key",
  "*.pem",
  "*credentials*",
  "*secret*",
  ".aws/**",
  ".ssh/**"
]
```

#### Index Documentation Only
```javascript
include_patterns: ["*.md", "*.txt", "*.rst"],
exclude_patterns: ["node_modules/**", "vendor/**"]
```

#### Index Source Code Only
```javascript
include_patterns: ["*.go", "*.js", "*.py", "*.java", "*.c", "*.cpp"],
exclude_patterns: ["*_test.*", "*.min.js", "vendor/**", "node_modules/**"]
```

---

## File Size Handling

### Size Limits

- **Default**: 1MB (1,048,576 bytes)
- **Minimum**: 0 bytes (no limit, not recommended)
- **Maximum**: 10MB (10,485,760 bytes)

### Rationale

**1MB default**:
- Balances indexing coverage and embedding costs
- Most source files are < 1MB
- Prevents indexing large generated files (minified JS, compiled binaries)

**10MB maximum**:
- Protects against runaway embedding costs
- OpenAI API has 8192 token limit (~32KB text)
- Large files would be truncated or fail anyway

### Size Filtering Logic

```go
if info.Size() > maxFileSize {
    return nil // Skip file silently
}
```

**Behavior**:
- Files larger than limit are silently skipped
- No error is returned (not considered failure)
- Skipped files NOT counted in `files_indexed`
- Logged at debug level (not user-visible)

### Size-Based Strategy

**Small repositories (< 100 files)**:
```javascript
max_file_size: 10485760  // 10MB - index everything
```

**Medium repositories (100-1000 files)**:
```javascript
max_file_size: 1048576  // 1MB - skip large generated files
```

**Large repositories (> 1000 files)**:
```javascript
max_file_size: 524288  // 512KB - aggressively filter
include_patterns: ["*.md", "*.txt"]  // Documentation only
```

---

## Error Handling

### Error Categories

#### 1. Validation Errors (User-facing)

**Path doesn't exist**:
```json
{
  "error": "validation_error",
  "message": "path does not exist",
  "details": {"field": "path", "path": "/invalid/path"}
}
```

**Invalid pattern**:
```json
{
  "error": "validation_error",
  "message": "invalid include pattern",
  "details": {"field": "include_patterns", "pattern": "[invalid"}
}
```

**Max file size too large**:
```json
{
  "error": "validation_error",
  "message": "max_file_size too large",
  "details": {"field": "max_file_size", "max_allowed": 10485760, "provided": 20971520}
}
```

#### 2. Timeout Errors

**Indexing timeout** (5 minutes exceeded):
```json
{
  "error": "timeout_error",
  "message": "repository indexing timed out",
  "details": {"timeout": "5m0s"}
}
```

#### 3. Internal Errors

**Embedding generation failure**:
```json
{
  "error": "internal_error",
  "message": "failed to generate embedding",
  "details": {"cause": "OpenAI API rate limit exceeded"}
}
```

**Vector store failure**:
```json
{
  "error": "internal_error",
  "message": "failed to insert checkpoint"
}
```

**Path traversal detected**:
```json
{
  "error": "internal_error",
  "message": "path traversal detected",
  "details": {"path": "../../etc/passwd"}
}
```

### Error Recovery

#### Continue on File Errors

Current behavior: **FAIL FAST** - first error aborts indexing

```go
if err := processFile(path); err != nil {
    return err // Abort entire indexing
}
```

**Future improvement**: Continue indexing, collect errors

```go
var errors []error
if err := processFile(path); err != nil {
    errors = append(errors, err)
    continue // Process next file
}
```

Return partial success with error details:
```json
{
  "files_indexed": 42,
  "files_failed": 3,
  "errors": [
    {"file": "path/to/file1.txt", "error": "permission denied"},
    {"file": "path/to/file2.md", "error": "encoding error"}
  ]
}
```

#### Retry Strategy

**Transient errors** (network, rate limits):
- Retry with exponential backoff
- Max 3 retries per file
- Skip file after max retries

**Permanent errors** (invalid UTF-8, permission denied):
- Log error and skip file
- Continue indexing

---

## Testing Requirements

### Unit Tests

#### 1. Pattern Matching Tests

```go
func TestPatternMatching(t *testing.T) {
    tests := []struct {
        name            string
        filename        string
        includePatterns []string
        excludePatterns []string
        want            bool
    }{
        {"include match", "file.md", []string{"*.md"}, nil, true},
        {"exclude match", "test.log", nil, []string{"*.log"}, false},
        {"include + exclude", "test.md", []string{"*.md"}, []string{"test.*"}, false},
        {"no patterns", "file.txt", nil, nil, true},
    }
    // ...
}
```

#### 2. File Size Filtering Tests

```go
func TestFileSizeFiltering(t *testing.T) {
    tests := []struct {
        name        string
        fileSize    int64
        maxFileSize int64
        want        bool
    }{
        {"within limit", 500000, 1048576, true},
        {"exceeds limit", 2000000, 1048576, false},
        {"exact limit", 1048576, 1048576, true},
    }
    // ...
}
```

#### 3. Path Traversal Prevention Tests

```go
func TestPathTraversalPrevention(t *testing.T) {
    tests := []struct {
        name     string
        repoPath string
        filePath string
        wantErr  bool
    }{
        {"valid path", "/repo", "/repo/file.txt", false},
        {"traversal up", "/repo", "/repo/../etc/passwd", true},
        {"traversal sideways", "/repo", "/other/file.txt", true},
    }
    // ...
}
```

### Integration Tests

#### 1. End-to-End Indexing Test

```go
func TestIndexRepository_EndToEnd(t *testing.T) {
    // Setup: Create temp directory with test files
    tmpDir := t.TempDir()
    createTestFiles(tmpDir, map[string]string{
        "file1.md":  "# Documentation",
        "file2.txt": "Plain text",
        "large.bin": string(make([]byte, 2*1024*1024)), // 2MB
    })

    // Index repository
    output, err := indexRepository(ctx, tmpDir, []string{"*.md", "*.txt"}, nil, 1048576)

    // Verify: 2 files indexed (large.bin skipped)
    assert.NoError(t, err)
    assert.Equal(t, 2, output.FilesIndexed)

    // Verify checkpoints created
    checkpoints, err := checkpointService.Search(ctx, "documentation", 10, tmpDir, []string{"indexed"})
    assert.NoError(t, err)
    assert.Len(t, checkpoints.Results, 1)
}
```

#### 2. Pattern Exclusion Test

```go
func TestIndexRepository_ExcludePatterns(t *testing.T) {
    tmpDir := t.TempDir()
    createTestFiles(tmpDir, map[string]string{
        "main.go":      "package main",
        "main_test.go": "package main",
        "vendor/pkg.go": "package vendor",
    })

    output, err := indexRepository(ctx, tmpDir, []string{"*.go"}, []string{"*_test.go", "vendor/**"}, 1048576)

    assert.NoError(t, err)
    assert.Equal(t, 1, output.FilesIndexed) // Only main.go
}
```

#### 3. Large Repository Test

```go
func TestIndexRepository_LargeRepo(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping large repository test")
    }

    tmpDir := t.TempDir()
    // Create 1000 files
    for i := 0; i < 1000; i++ {
        createFile(tmpDir, fmt.Sprintf("file%d.txt", i), "content")
    }

    start := time.Now()
    output, err := indexRepository(ctx, tmpDir, nil, nil, 1048576)
    duration := time.Since(start)

    assert.NoError(t, err)
    assert.Equal(t, 1000, output.FilesIndexed)
    assert.Less(t, duration, 5*time.Minute) // Must complete within timeout
}
```

### Manual Testing Checklist

- [ ] Index small repository (< 10 files)
- [ ] Index medium repository (100-1000 files)
- [ ] Index large repository (> 1000 files)
- [ ] Verify checkpoint creation in database
- [ ] Search indexed files semantically
- [ ] Test with various include patterns
- [ ] Test with various exclude patterns
- [ ] Test with different max file sizes
- [ ] Verify path traversal prevention
- [ ] Test with non-existent path (validation error)
- [ ] Test with invalid patterns (validation error)
- [ ] Test timeout with very large repository
- [ ] Test OpenTelemetry traces and metrics
- [ ] Test CLI command (`ctxd index`)

---

## Implementation Plan

### Phase 1: Core Implementation (Complete)

**Status**: Implemented in v0.x

**Components**:
- [x] MCP tool handler (`handleIndexRepository`)
- [x] File tree traversal (`indexRepositoryFiles`)
- [x] Pattern matching (include/exclude)
- [x] File size filtering
- [x] Path traversal prevention
- [x] Checkpoint creation integration
- [x] CLI command (`ctxd index`)

**Commits**:
- Initial implementation with basic pattern matching
- Path traversal security fix (CVE-007)

### Phase 2: Optimizations (Future)

**Priority**: Medium

**Features**:
- [ ] Batch embedding generation (10x speedup)
- [ ] Parallel processing with worker pool (20x speedup)
- [ ] Incremental indexing (skip unchanged files)
- [ ] Progress reporting (callback for every N files)

**Estimated Effort**: 1-2 weeks

### Phase 3: Enhanced Error Handling (Future)

**Priority**: Low

**Features**:
- [ ] Continue on file errors (partial success)
- [ ] Retry with exponential backoff
- [ ] Detailed error reporting per file
- [ ] Binary file detection and skip

**Estimated Effort**: 1 week

### Phase 4: Advanced Features (Future)

**Priority**: Low

**Features**:
- [ ] AST-based code indexing (extract functions, classes)
- [ ] Chunking for large files (split into sections)
- [ ] Metadata extraction (author, license, etc.)
- [ ] Custom tag strategies (language-specific)
- [ ] De-duplication (avoid re-indexing identical files)

**Estimated Effort**: 2-3 weeks

---

## Open Questions

### Q1: Should indexing be idempotent?

**Current behavior**: Re-indexing creates duplicate checkpoints

**Options**:
1. **Keep current behavior** (simplest)
   - Pros: Simple implementation, no state tracking
   - Cons: Duplicate checkpoints clutter search results
2. **Delete old checkpoints before re-indexing**
   - Pros: Clean state, no duplicates
   - Cons: Loses historical checkpoints (e.g., if file was edited)
3. **Update existing checkpoints**
   - Pros: Preserves checkpoint IDs, clean state

**Recommendation**: Option 1 for MVP, Option 3 for future enhancement

### Q2: How to handle binary files?

**Current behavior**: Attempts to read as UTF-8, may fail

**Options**:
1. **Skip binary files automatically** (check MIME type or magic bytes)
2. **Extract metadata only** (filename, size, type)
3. **Convert to text** (e.g., PDF → text, DOCX → text)

**Recommendation**: Option 1 (skip) for MVP, Option 3 (convert) for future

### Q3: Should we support symbolic links?

**Current behavior**: `filepath.Walk` follows symlinks

**Risk**: Circular symlinks cause infinite loop

**Options**:
1. **Continue following symlinks** (current)
2. **Skip symlinks entirely**
3. **Follow with cycle detection**

**Recommendation**: Option 3 for security and robustness

### Q4: What about very large repositories (> 10,000 files)?

**Current limitation**: 5-minute timeout may be insufficient

**Options**:
1. **Increase timeout** (e.g., 30 minutes)
2. **Batch indexing** (index 1000 files at a time, multiple calls)
3. **Background job** (return immediately, index asynchronously)

**Recommendation**: Option 1 (increase timeout) for now, Option 3 (background job) for future

### Q5: How to exclude sensitive files automatically?

**Current behavior**: User must specify exclude patterns

**Options**:
1. **Keep current behavior** (explicit exclusions)
2. **Built-in exclusion list** (*.env, *.key, etc.)
3. **AI-powered detection** (classify files as sensitive)

**Recommendation**: Option 2 (built-in list) as default, allow override
