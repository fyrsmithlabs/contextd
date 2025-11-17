---
name: golang-pro
description: Expert Go development following Effective Go, TDD, and interface-driven development. Enforces 80%+ test coverage, security-first patterns, and proper error handling.
---

# Golang Pro Skill

Use this skill for **ALL Go code development** in the contextd project.

## MANDATORY: Check Security First

Before writing ANY Go code, review the security checklist from project CLAUDE.md Section 1:
- [ ] Does this expose data across project/owner boundaries?
- [ ] Are all user inputs validated and sanitized?
- [ ] Is sensitive data encrypted/redacted?
- [ ] Are there access control checks?
- [ ] Does this maintain multi-tenant isolation?
- [ ] Could this cause compliance violations (GDPR, HIPAA, SOC 2)?

**For contextd**: ALWAYS check if secrets could leak into context. Apply scrubbing where needed.

## Workflow: RED-GREEN-REFACTOR

### 1. RED Phase (Write Failing Test)

**Before ANY implementation:**

1. **Read the spec** (if it exists in `docs/specs/<feature>/SPEC.md`)
2. **Write a test that fails** (test-driven approach)
3. **Run the test and verify it fails**
4. **Commit the failing test**

Example test structure:
```go
func TestCheckpointSave(t *testing.T) {
    tests := []struct {
        name    string
        input   SaveRequest
        want    SaveResponse
        wantErr bool
    }{
        {
            name: "valid checkpoint",
            input: SaveRequest{
                Summary:     "test checkpoint",
                ProjectPath: "/tmp/test",
            },
            want: SaveResponse{ID: "ckpt_123"},
            wantErr: false,
        },
        {
            name: "missing project path",
            input: SaveRequest{Summary: "test"},
            want: SaveResponse{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := SaveCheckpoint(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("SaveCheckpoint() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("SaveCheckpoint() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Command**: `go test ./... -v`

### 2. GREEN Phase (Make Test Pass)

**Implement minimal code to pass the test:**

1. **Follow Effective Go principles** (see checklist below)
2. **Use interfaces for dependencies** (enable testing/mocking)
3. **Handle errors properly** (never ignore errors)
4. **Add input validation** (sanitize all user inputs)
5. **Run tests and verify they pass**
6. **Check coverage**: `go test ./... -cover` (must be ≥80%)

**Effective Go Checklist:**
- [ ] Use clear, descriptive names (exportedFunc, privateFunc)
- [ ] Accept interfaces, return structs (flexible design)
- [ ] Handle errors explicitly (no `_` for errors)
- [ ] Use `defer` for cleanup (close files, unlock mutexes)
- [ ] Avoid global state (pass dependencies explicitly)
- [ ] Use `context.Context` for cancellation/deadlines
- [ ] Document exported functions with godoc comments
- [ ] Use `gofmt` (or `gofumpt`) for formatting

**Interface-Driven Example:**
```go
// Define interface for dependency
type VectorStore interface {
    Search(ctx context.Context, query string) ([]Result, error)
    Insert(ctx context.Context, doc Document) error
}

// Accept interface, return struct
func NewCheckpointService(store VectorStore) *CheckpointService {
    return &CheckpointService{store: store}
}

// Implementation uses interface
func (s *CheckpointService) Save(ctx context.Context, req SaveRequest) (SaveResponse, error) {
    // Validate input
    if err := req.Validate(); err != nil {
        return SaveResponse{}, fmt.Errorf("invalid request: %w", err)
    }

    // Use interface
    if err := s.store.Insert(ctx, req.ToDocument()); err != nil {
        return SaveResponse{}, fmt.Errorf("store insert: %w", err)
    }

    return SaveResponse{ID: "ckpt_123"}, nil
}
```

**Security Patterns:**
```go
// Input validation
func (r *SaveRequest) Validate() error {
    if r.ProjectPath == "" {
        return errors.New("project_path required")
    }
    // Sanitize file paths
    clean := filepath.Clean(r.ProjectPath)
    if !filepath.IsAbs(clean) {
        return errors.New("project_path must be absolute")
    }
    return nil
}

// Secret scrubbing (when feature is implemented)
func ScrubSecrets(content string) string {
    // Regex patterns for common secrets
    patterns := []struct {
        regex *regexp.Regexp
        label string
    }{
        {regexp.MustCompile(`(?i)(api[_-]?key|token|password)\s*[:=]\s*['"]\s*([^'"]+)`), "API_KEY"},
        {regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`), "OPENAI_KEY"},
    }

    result := content
    for _, p := range patterns {
        result = p.regex.ReplaceAllString(result, fmt.Sprintf("[REDACTED:%s]", p.label))
    }
    return result
}
```

**Command**: `go test ./... -cover`

### 3. REFACTOR Phase (Improve Code)

**After tests pass, improve the code:**

1. **Extract common patterns** (DRY principle)
2. **Simplify complex functions** (split into smaller functions)
3. **Add documentation** (godoc for exported symbols)
4. **Check for error handling** (wrap errors with context)
5. **Run tests again** (verify refactor didn't break anything)
6. **Run linters**: `golangci-lint run`

**Refactoring Checklist:**
- [ ] Functions < 50 lines (split if longer)
- [ ] Cyclomatic complexity < 10 (simplify if higher)
- [ ] No duplicated code (extract to functions)
- [ ] Clear variable names (no single letters except i, j, k in loops)
- [ ] Errors wrapped with context: `fmt.Errorf("operation: %w", err)`
- [ ] Godoc comments for all exported symbols

**Command**:
```bash
go test ./... -cover
golangci-lint run
go build ./...
```

### 4. COMMIT Phase

**Create a proper commit:**

1. **Verify all tests pass**: `go test ./...`
2. **Verify coverage ≥80%**: `go test ./... -cover`
3. **Verify build succeeds**: `go build ./...`
4. **Run pre-commit hooks**: `pre-commit run --all-files`
5. **Stage changes**: `git add <files>`
6. **Commit with message**:
   ```
   feat(pkg/checkpoint): implement checkpoint save with validation

   - Add CheckpointService with Save method
   - Implement input validation for SaveRequest
   - Add VectorStore interface for testability
   - Achieve 85% test coverage

   Implements: docs/specs/checkpoint/SPEC.md
   ```

**Commit Message Format:**
- `feat(scope): description` - New feature
- `fix(scope): description` - Bug fix
- `refactor(scope): description` - Code refactor
- `test(scope): description` - Test changes
- `docs(scope): description` - Documentation

## Testing Requirements

### Coverage Target: ≥80%

**Run coverage report:**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Focus coverage on:**
- [ ] Happy path (success cases)
- [ ] Error cases (all error branches)
- [ ] Edge cases (empty inputs, nil values, boundary conditions)
- [ ] Concurrency (if applicable, use race detector: `go test -race`)

### Test Types

**Unit Tests** (test individual functions):
```go
func TestValidateRequest(t *testing.T) {
    // Table-driven tests
    tests := []struct {
        name    string
        req     SaveRequest
        wantErr bool
    }{
        {"valid", SaveRequest{ProjectPath: "/tmp/test"}, false},
        {"empty path", SaveRequest{}, true},
        {"relative path", SaveRequest{ProjectPath: "relative"}, true},
    }
    // ...
}
```

**Integration Tests** (test component interactions):
```go
func TestCheckpointServiceIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup real dependencies
    store := qdrant.NewClient(testConfig)
    service := NewCheckpointService(store)

    // Test actual operations
    ctx := context.Background()
    resp, err := service.Save(ctx, testRequest)
    // ...
}
```

**Run integration tests:** `go test ./... -v` (run short tests with `-short` flag)

### Mock Interfaces

**Use gomock for interface mocking:**
```bash
go install github.com/golang/mock/mockgen@latest
mockgen -source=vectorstore.go -destination=mock_vectorstore.go -package=checkpoint
```

**Example mock usage:**
```go
func TestCheckpointServiceMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockStore := NewMockVectorStore(ctrl)
    mockStore.EXPECT().
        Insert(gomock.Any(), gomock.Any()).
        Return(nil)

    service := NewCheckpointService(mockStore)
    // Test with mock
}
```

## Error Handling

### Wrap Errors with Context

**Always wrap errors with operation context:**
```go
if err := store.Insert(ctx, doc); err != nil {
    return fmt.Errorf("inserting checkpoint: %w", err)
}
```

### Define Sentinel Errors

**For expected errors, use sentinel values:**
```go
var (
    ErrNotFound = errors.New("checkpoint not found")
    ErrInvalidInput = errors.New("invalid input")
)

// Usage
if err := validate(); errors.Is(err, ErrInvalidInput) {
    // Handle validation error
}
```

### Custom Error Types

**For rich error information:**
```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// Usage
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    // Handle validation error
}
```

## Performance Considerations

### Avoid Premature Optimization

**Optimize only when:**
1. Profiling shows a bottleneck
2. Performance requirement is not met
3. User experience is affected

**Profile first:**
```bash
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof
```

### Common Performance Patterns

**Use sync.Pool for frequently allocated objects:**
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processData() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()
    // Use buffer
}
```

**Use buffered channels for async operations:**
```go
ch := make(chan Result, 100) // Buffered channel
go func() {
    for res := range ch {
        process(res)
    }
}()
```

## Documentation

### Godoc Comments

**All exported symbols must have godoc comments:**
```go
// CheckpointService manages checkpoint storage and retrieval.
// It provides methods for saving, searching, and deleting checkpoints
// with support for multi-tenant isolation.
type CheckpointService struct {
    store VectorStore
}

// Save stores a checkpoint with the given request parameters.
// It validates the input, generates a unique ID, and persists
// the checkpoint to the vector store.
//
// Returns ErrInvalidInput if the request fails validation.
func (s *CheckpointService) Save(ctx context.Context, req SaveRequest) (SaveResponse, error) {
    // Implementation
}
```

### Package Documentation

**Add package-level documentation in doc.go:**
```go
// Package checkpoint provides checkpoint management for contextd.
//
// Checkpoints allow users to save and resume session state efficiently.
// Each checkpoint is stored with vector embeddings for semantic search.
//
// Example usage:
//
//     service := checkpoint.NewService(store)
//     resp, err := service.Save(ctx, checkpoint.SaveRequest{
//         Summary:     "Working on feature X",
//         ProjectPath: "/path/to/project",
//     })
//
package checkpoint
```

## Checklist: Before Requesting Code Review

- [ ] All tests pass: `go test ./...`
- [ ] Coverage ≥80%: `go test ./... -cover`
- [ ] No race conditions: `go test -race ./...`
- [ ] Build succeeds: `go build ./...`
- [ ] Linters pass: `golangci-lint run`
- [ ] Pre-commit hooks pass: `pre-commit run --all-files`
- [ ] Security checklist completed (Section 1 of project CLAUDE.md)
- [ ] Documentation complete (godoc for all exported symbols)
- [ ] Error handling proper (no ignored errors, wrapped with context)
- [ ] Interfaces used for dependencies (testability)
- [ ] CHANGELOG.md updated

## Common Mistakes to Avoid

1. **Ignoring errors**: Always check `err != nil`
2. **Naked returns**: Don't use naked returns in functions >5 lines
3. **Global state**: Pass dependencies explicitly, avoid `init()` with side effects
4. **Missing context**: Always accept `context.Context` for I/O operations
5. **Panic in libraries**: Return errors, don't panic (except in `init()` or impossible states)
6. **Unbounded goroutines**: Use worker pools or limit concurrency
7. **Not closing resources**: Always `defer` cleanup (Close, Unlock, etc.)
8. **Ignoring race detector**: Run `go test -race` to catch data races

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

---

**Remember**: This skill enforces TDD. **Write the test first.** Always.
