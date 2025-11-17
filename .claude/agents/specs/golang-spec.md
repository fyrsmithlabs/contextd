# Go Language Specification for contextd

## Official References

- **Effective Go**: https://go.dev/doc/effective_go
- **Go Security**: https://go.dev/doc/security/best-practices
- **Go Blog**: https://go.dev/blog/
- **Standard Library**: https://pkg.go.dev/std

## Key Principles

### Effective Go Core Concepts

1. **Formatting**: Use `gofmt`, no exceptions
2. **Commentary**: Godoc format, explain "why" not "what"
3. **Names**: Short, concise, lowercase, no underscores
4. **Semicolons**: Automatic insertion, place `{` on same line
5. **Control Structures**: No parentheses, `if` with init statement
6. **Functions**: Multiple return values, named returns for docs
7. **Data**: Composite literals, new vs make, arrays vs slices
8. **Initialization**: init functions, const order
9. **Methods**: Pointer vs value receivers
10. **Interfaces**: Accept interfaces, return structs
11. **Errors**: errors.Is/As, wrap with %w
12. **Panic**: Only for programming errors
13. **Concurrency**: Share memory by communicating

### Error Handling

**✅ CORRECT Pattern:**
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to open file %s: %w", path, err)
}

// Check error types
if errors.Is(err, fs.ErrNotExist) {
    // Handle missing file
}

// Check error values
var pathErr *fs.PathError
if errors.As(err, &pathErr) {
    // Handle path error
}
```

**❌ INCORRECT Pattern:**
```go
// Don't ignore errors
data, _ := os.ReadFile(path)

// Don't lose error context
if err != nil {
    return err  // Where did this come from?
}

// Don't use panic for normal errors
if err != nil {
    panic(err)  // Only for programming errors!
}
```

### Concurrency Patterns

**Channel Communication:**
```go
// Buffered channels for non-blocking sends
ch := make(chan Result, 100)

// Range over channel
for result := range ch {
    process(result)
}

// Select for multiple channels
select {
case msg := <-ch1:
    handle(msg)
case msg := <-ch2:
    handle(msg)
case <-ctx.Done():
    return ctx.Err()
}
```

**Context Propagation:**
```go
func Process(ctx context.Context, data Data) error {
    // Always accept context as first parameter
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Pass context to child operations
    return processChild(ctx, data)
}
```

**Goroutine Lifecycle:**
```go
// ✅ GOOD: Clear termination
func startWorker(ctx context.Context) {
    go func() {
        defer cleanup()

        for {
            select {
            case <-ctx.Done():
                return  // Clean exit
            case work := <-workCh:
                process(work)
            }
        }
    }()
}

// ❌ BAD: Goroutine leak
func startWorker() {
    go func() {
        for {
            work := <-workCh  // Blocks forever if channel closed
            process(work)
        }
    }()  // No way to stop this!
}
```

### Memory Management

**Slices:**
```go
// Pre-allocate known capacity
results := make([]Result, 0, len(items))

// Copy slices properly
dst := make([]byte, len(src))
copy(dst, src)

// Avoid slice memory leaks
func processLargeSlice(data []byte) []byte {
    // If returning small portion, copy it
    result := make([]byte, 10)
    copy(result, data[:10])
    return result  // Original data can be GC'd
}
```

**Maps:**
```go
// Pre-allocate known size
m := make(map[string]int, 1000)

// Delete to free memory
delete(m, key)

// Range over map (unordered)
for k, v := range m {
    process(k, v)
}
```

**sync.Pool:**
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()

    // Use buf
}
```

## Standard Library Patterns

### File Operations

```go
// Safe file read with proper cleanup
func readFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("open %s: %w", path, err)
    }
    defer f.Close()

    data, err := io.ReadAll(f)
    if err != nil {
        return nil, fmt.Errorf("read %s: %w", path, err)
    }

    return data, nil
}
```

### HTTP Clients

```go
// Configure timeout
client := &http.Client{
    Timeout: 30 * time.Second,
}

// Close response body
resp, err := client.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()

// Read with limit
body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
```

### Testing

```go
// Table-driven tests
func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    error
    }{
        {"valid", "test", nil},
        {"empty", "", ErrEmpty},
        {"too long", strings.Repeat("a", 1001), ErrTooLong},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Validate(tt.input)
            if got != tt.want {
                t.Errorf("Validate(%q) = %v, want %v", tt.input, got, tt.want)
            }
        })
    }
}
```

## Security Patterns

### Path Validation (Go 1.20+)

```go
import "path/filepath"

// Prevent directory traversal
func safePath(userPath string) (string, error) {
    if !filepath.IsLocal(userPath) {
        return "", fmt.Errorf("path must be local: %s", userPath)
    }

    fullPath := filepath.Join(baseDir, userPath)

    // Double-check with EvalSymlinks
    realPath, err := filepath.EvalSymlinks(fullPath)
    if err != nil {
        return "", err
    }

    if !strings.HasPrefix(realPath, baseDir) {
        return "", fmt.Errorf("path outside base: %s", userPath)
    }

    return realPath, nil
}
```

### Constant-Time Comparison

```go
import "crypto/subtle"

// Prevent timing attacks
func validateToken(token, expected string) bool {
    return subtle.ConstantTimeCompare(
        []byte(token),
        []byte(expected),
    ) == 1
}
```

### Command Execution

```go
import (
    "context"
    "os/exec"
    "time"
)

// Safe command execution
func runCommand(ctx context.Context, name string, args ...string) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    // No shell interpolation
    cmd := exec.CommandContext(ctx, name, args...)

    // Restrict environment
    cmd.Env = []string{
        "PATH=/usr/local/bin:/usr/bin",
        "HOME=/var/lib/app",
    }

    // Set working directory
    cmd.Dir = workDir

    return cmd.Run()
}
```

## Common Anti-Patterns

### ❌ Don't Use

1. **Global mutable state** (except config loaded once)
2. **init() for complex logic** (use explicit initialization)
3. **Naked returns in long functions** (confusing)
4. **Type assertions without checking** (use comma-ok idiom)
5. **Closing channels multiple times** (causes panic)
6. **Writing to closed channels** (causes panic)
7. **Modifying slice during range** (undefined behavior)
8. **Empty interface{}** (use generics in Go 1.18+)
9. **Panic in library code** (return errors)
10. **Goroutines without cleanup** (use context)

## Tools

### Required Tools

```bash
# Format code
gofmt -w .

# Organize imports
goimports -w .

# Static analysis
go vet ./...

# Race detector
go test -race ./...

# Coverage
go test -cover ./...

# Benchmarks
go test -bench=. -benchmem
```

### Recommended Linters

- **staticcheck**: https://staticcheck.io/
- **golangci-lint**: https://golangci-lint.run/
- **gosec**: https://github.com/securego/gosec
- **govulncheck**: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck

## Troubleshooting

### "panic: send on closed channel"
```
Root cause: Writing to closed channel
Fix: Ensure sender controls channel lifecycle
Pattern: Use context.Done() to signal shutdown
```

### "fatal error: concurrent map write"
```
Root cause: Concurrent map access without sync
Fix: Use sync.RWMutex or sync.Map
Pattern: Protect all map access with locks
```

### "goroutine leak"
```
Root cause: Goroutine waiting forever
Fix: Use context for cancellation
Pattern: Always provide exit path for goroutines
```

### "race condition detected"
```
Root cause: Concurrent access to shared data
Fix: Use sync.Mutex, sync.RWMutex, or channels
Pattern: Share memory by communicating
```

## contextd-Specific Patterns

See `/specs/contextd-architecture.md` for:
- Error handling conventions
- Logging patterns
- Configuration management
- Testing standards

---

**References:**
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://go.dev/wiki/CodeReviewComments
- Go Security: https://go.dev/doc/security/best-practices
