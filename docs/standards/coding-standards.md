# Go Coding Standards

This document defines the Go coding standards for contextd, focusing on security-first development and context efficiency.

## Core Principles

### 1. Security-First Coding

**Every line of code MUST prioritize security:**

- **Never cat credentials to context** - Use indirect file reads with 0600 permissions
- **No secrets in code** - Use environment variables and config files
- **Constant-time comparisons** - For authentication tokens and sensitive data
- **Wrap all errors** - Provide context without leaking sensitive information
- **Validate all inputs** - At service boundary, before processing

### 2. Context Efficiency

**Minimize context bloat in all coding decisions:**

- **Golden Rule**: "1 MESSAGE = ALL RELATED OPERATIONS"
  - Batch ALL todos in ONE TodoWrite call (5-10+ minimum)
  - Batch ALL file operations in ONE message
  - Batch ALL bash commands in ONE message
  - Use concurrent/parallel execution wherever possible

- **File Organization**: NEVER save files to root folder
  - `/docs` - Documentation and markdown files
  - `/config` - Configuration files
  - `/scripts` - Utility scripts
  - `/examples` - Example code
  - Follow Go project best practices for repo structure

### 3. Test-Driven Development (TDD)

**ALL code MUST be test-driven. No exceptions.**

- Write tests FIRST (red phase)
- Implement minimal code to pass (green phase)
- Refactor while maintaining passing tests (refactor phase)
- Minimum 80% coverage (100% for critical paths)
- See: `docs/standards/testing-standards.md`

## Naming Conventions

### Avoid Redundant Package Names

**CRITICAL**: Package names appear at call sites, so don't repeat them.

```go
// GOOD - Clean, no redundancy
package slack
type Client struct {}
func (c *Client) SendMessage() {}
func NewClient() *Client { return &Client{} }

// Usage:
client := slack.NewClient()
client.SendMessage()

// BAD - Redundant "Slack" prefix
package slack
type SlackClient struct {}  // "Slack" is redundant
func (c *SlackClient) SendSlackMessage() {}  // "Slack" redundant
func NewSlackClient() *SlackClient { return &SlackClient{} }

// Usage (awkward):
client := slack.NewSlackClient()
client.SendSlackMessage()
```

### Interface Naming

```go
// GOOD - Single-method interfaces end with "-er"
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Searcher interface {
    Search(query string) ([]Result, error)
}

// GOOD - Multi-method interfaces describe behavior
type VectorStore interface {
    CreateDatabase(ctx context.Context, database string) error
    UpsertPoints(ctx context.Context, database, collection string, points []Point) error
    Search(ctx context.Context, database, collection string, vector []float32, limit int) ([]SearchResult, error)
}

// BAD - Don't use "I" prefix
type IVectorStore interface { ... }  // Not idiomatic Go
```

### Variable Naming

```go
// GOOD - Short names for short scopes
for i, v := range items {
    // i and v are clear in this small scope
}

// GOOD - Descriptive names for larger scopes
func ProcessCheckpoints(ctx context.Context, projectPath string) error {
    checkpointService := checkpoint.NewService(store)
    projectHash := hashProjectPath(projectPath)
    // Descriptive names in larger scope
}

// GOOD - Receiver names (single letter or acronym)
func (c *Client) Connect() error { ... }
func (vs *VectorStore) Search() error { ... }

// BAD - Receiver named "this" or "self"
func (this *Client) Connect() error { ... }  // Not idiomatic Go
```

### Package Naming

```go
// GOOD - Short, lowercase, single word
package auth
package checkpoint
package vectorstore

// BAD - Underscores, mixed case, multiple words
package auth_service  // Use package auth, type Service
package CheckPoint    // Use lowercase: checkpoint
package vector_store  // Use single word: vectorstore
```

## Error Handling

### Always Wrap Errors with Context

```go
// GOOD - Wrap with context using %w
func (s *Service) Authenticate(username, password string) (string, error) {
    user, err := s.repo.FindByUsername(username)
    if err != nil {
        return "", fmt.Errorf("failed to find user %q: %w", username, err)
    }

    if !user.ValidatePassword(password) {
        return "", fmt.Errorf("invalid password for user %q", username)
    }

    token, err := s.generateToken(user)
    if err != nil {
        return "", fmt.Errorf("failed to generate token for user %q: %w", username, err)
    }

    return token, nil
}

// BAD - No context, no wrapping
func (s *Service) Authenticate(username, password string) (string, error) {
    user, err := s.repo.FindByUsername(username)
    if err != nil {
        return "", err  // Lost context!
    }
    // ...
}
```

### Never Panic for Runtime Errors

```go
// GOOD - Return errors for runtime issues
func LoadConfig() (*Config, error) {
    data, err := os.ReadFile("config.json")
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    // ...
}

// BAD - Panic for runtime errors
func LoadConfig() *Config {
    data, err := os.ReadFile("config.json")
    if err != nil {
        panic(err)  // Don't panic for runtime errors!
    }
    // ...
}

// ACCEPTABLE - Panic for programmer errors (rare)
func NewService(store VectorStore) *Service {
    if store == nil {
        panic("store cannot be nil")  // Programmer error, not runtime
    }
    return &Service{store: store}
}
```

### Use errors.Is() and errors.As()

```go
// GOOD - Check error types properly
import "errors"

var ErrNotFound = errors.New("not found")

func (s *Service) GetCheckpoint(id string) (*Checkpoint, error) {
    checkpoint, err := s.store.Get(id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get checkpoint: %w", err)
    }
    return checkpoint, nil
}

// GOOD - Extract error details
var notFoundErr *NotFoundError
if errors.As(err, &notFoundErr) {
    log.Printf("Resource not found: %s", notFoundErr.Resource)
}

// BAD - String comparison (fragile)
if err.Error() == "not found" {  // Breaks if error message changes!
    // ...
}
```

## Context Propagation

### Always Pass Context as First Parameter

```go
// GOOD - Context first, propagate through call chain
func (s *Service) CreateCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
    ctx, span := tracer.Start(ctx, "CreateCheckpoint")
    defer span.End()

    return s.store.Upsert(ctx, checkpoint)
}

func (s *Store) Upsert(ctx context.Context, checkpoint *Checkpoint) error {
    ctx, span := tracer.Start(ctx, "Store.Upsert")
    defer span.End()

    // Use ctx for cancellation, timeouts, tracing
    return s.client.Insert(ctx, checkpoint)
}

// BAD - No context propagation
func (s *Service) CreateCheckpoint(checkpoint *Checkpoint) error {
    return s.store.Upsert(checkpoint)  // Lost tracing, cancellation!
}
```

### Never Store Context in Structs

```go
// GOOD - Pass context as parameter
type Service struct {
    store VectorStore
}

func (s *Service) Operation(ctx context.Context) error {
    return s.store.DoSomething(ctx)
}

// BAD - Context in struct (breaks cancellation)
type Service struct {
    ctx   context.Context  // Don't do this!
    store VectorStore
}
```

## Concurrency

### Use Goroutines for Independent Operations

```go
// GOOD - Concurrent independent operations
func (s *Service) ProcessBatch(items []Item) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(items))

    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            if err := s.processItem(item); err != nil {
                errChan <- err
            }
        }(item)
    }

    wg.Wait()
    close(errChan)

    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("processing failed: %v", errs)
    }

    return nil
}
```

### Protect Shared State with Mutexes

```go
// GOOD - Protect shared state
type Cache struct {
    mu    sync.RWMutex
    items map[string]Item
}

func (c *Cache) Get(key string) (Item, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, ok := c.items[key]
    return item, ok
}

func (c *Cache) Set(key string, item Item) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.items[key] = item
}

// BAD - Race condition
type Cache struct {
    items map[string]Item  // No mutex!
}

func (c *Cache) Get(key string) (Item, bool) {
    return c.items[key]  // Race condition!
}
```

### Always Use go test -race

```bash
# REQUIRED - Check for race conditions
go test -race ./...
```

## Struct Design

### Use Composition Over Inheritance

```go
// GOOD - Composition
type BaseClient struct {
    http *http.Client
    url  string
}

type AuthenticatedClient struct {
    BaseClient
    token string
}

// GOOD - Interface satisfaction through composition
type LoggedStore struct {
    store  VectorStore
    logger *log.Logger
}

func (ls *LoggedStore) Search(ctx context.Context, query string) ([]Result, error) {
    ls.logger.Printf("Searching: %s", query)
    return ls.store.Search(ctx, query)
}
```

### Prefer Small Interfaces

```go
// GOOD - Small, focused interfaces
type Searcher interface {
    Search(ctx context.Context, query string) ([]Result, error)
}

type Indexer interface {
    Index(ctx context.Context, doc Document) error
}

// Compose when needed
type SearchAndIndex interface {
    Searcher
    Indexer
}

// BAD - Monolithic interface
type VectorDatabase interface {
    Search(ctx context.Context, query string) ([]Result, error)
    Index(ctx context.Context, doc Document) error
    Delete(ctx context.Context, id string) error
    Update(ctx context.Context, doc Document) error
    Backup(ctx context.Context, path string) error
    Restore(ctx context.Context, path string) error
    // Too many responsibilities!
}
```

## Function Design

### Return Errors, Don't Panic

```go
// GOOD - Return errors
func ParseConfig(data []byte) (*Config, error) {
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    return &config, nil
}

// BAD - Panic (caller can't handle)
func ParseConfig(data []byte) *Config {
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        panic(err)
    }
    return &config
}
```

### Keep Functions Small and Focused

```go
// GOOD - Small, single-responsibility functions
func (s *Service) Authenticate(username, password string) (string, error) {
    user, err := s.findUser(username)
    if err != nil {
        return "", err
    }

    if err := s.validatePassword(user, password); err != nil {
        return "", err
    }

    return s.generateToken(user)
}

func (s *Service) findUser(username string) (*User, error) { ... }
func (s *Service) validatePassword(user *User, password string) error { ... }
func (s *Service) generateToken(user *User) (string, error) { ... }

// BAD - Giant function doing everything
func (s *Service) Authenticate(username, password string) (string, error) {
    // 200 lines of code doing everything
    // Hard to test, hard to understand, hard to maintain
}
```

## Documentation

### Document Exported Types and Functions

```go
// GOOD - Package doc, exported types documented
// Package checkpoint provides session checkpoint management.
//
// Checkpoints are snapshots of Claude Code sessions that can be
// saved, searched, and restored. They support semantic search
// via vector embeddings.
package checkpoint

// Service provides checkpoint management operations.
type Service struct {
    store VectorStore
}

// NewService creates a new checkpoint service.
func NewService(store VectorStore) *Service {
    return &Service{store: store}
}

// Save saves a checkpoint to the vector store.
//
// The checkpoint is embedded using the configured embedding service
// and stored in the project-specific database for isolation.
func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    // ...
}

// BAD - No documentation
package checkpoint

type Service struct {
    store VectorStore
}

func NewService(store VectorStore) *Service {
    return &Service{store: store}
}

func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    // ...
}
```

### Add Comments for Complex Logic

```go
// GOOD - Explain "why", not "what"
func (s *Service) HybridSearch(ctx context.Context, query string) ([]Result, error) {
    // Use 70% semantic + 30% string matching for better recall
    // Semantic alone misses exact matches, string alone misses similar concepts
    semantic, err := s.semanticSearch(ctx, query, 0.7)
    if err != nil {
        return nil, fmt.Errorf("semantic search failed: %w", err)
    }

    string, err := s.stringSearch(ctx, query, 0.3)
    if err != nil {
        return nil, fmt.Errorf("string search failed: %w", err)
    }

    return s.merge(semantic, string), nil
}

// BAD - Obvious comments
func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    // Call the store's Upsert method
    return s.store.Upsert(ctx, checkpoint)  // The code is self-explanatory
}
```

## Testing Integration

### Write Tests First (TDD)

```go
// Step 1: Write failing test (RED)
func TestAuthenticate_ValidCredentials_ReturnsToken(t *testing.T) {
    service := NewService(mockStore)

    token, err := service.Authenticate("user", "pass")

    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if token == "" {
        t.Fatal("Expected token, got empty string")
    }
}

// Step 2: Implement to pass (GREEN)
func (s *Service) Authenticate(username, password string) (string, error) {
    // Minimal implementation
    return "dummy-token", nil
}

// Step 3: Refactor (maintain passing tests)
func (s *Service) Authenticate(username, password string) (string, error) {
    user, err := s.findUser(username)
    if err != nil {
        return "", err
    }
    // ... proper implementation
}
```

**See:** `docs/standards/testing-standards.md` for complete TDD requirements

## OpenTelemetry Instrumentation

### Add Spans for Important Operations

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("contextd")

func (s *Service) CreateCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
    // Create span for tracing
    ctx, span := tracer.Start(ctx, "Service.CreateCheckpoint")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("checkpoint.id", checkpoint.ID),
        attribute.String("checkpoint.project", checkpoint.Project),
    )

    // Perform operation
    if err := s.store.Upsert(ctx, checkpoint); err != nil {
        span.RecordError(err)
        return fmt.Errorf("failed to upsert: %w", err)
    }

    return nil
}
```

## Security Patterns

### Constant-Time Comparison for Secrets

```go
import "crypto/subtle"

// GOOD - Constant-time comparison
func (m *Middleware) validateToken(provided string) bool {
    expected := m.expectedToken
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// BAD - Timing attack vulnerable
func (m *Middleware) validateToken(provided string) bool {
    return provided == m.expectedToken  // Timing attack possible!
}
```

### Never Log Sensitive Data

```go
// GOOD - Don't log tokens, passwords, API keys
log.Printf("Authentication attempt for user: %s", username)

if !validateToken(token) {
    log.Printf("Invalid token for user: %s", username)
    return ErrUnauthorized
}

// BAD - Logs sensitive data
log.Printf("Validating token: %s", token)  // Token in logs!
log.Printf("User password: %s", password)  // Password in logs!
```

### Validate All Inputs

```go
// GOOD - Validate at service boundary
func (s *Service) CreateCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
    if checkpoint == nil {
        return errors.New("checkpoint cannot be nil")
    }

    if checkpoint.Summary == "" {
        return errors.New("checkpoint summary required")
    }

    if checkpoint.Project == "" {
        return errors.New("checkpoint project required")
    }

    // Proceed with validated input
    return s.store.Upsert(ctx, checkpoint)
}
```

## Import Organization

```go
// GOOD - Organized imports
import (
    // Standard library
    "context"
    "fmt"
    "log"

    // Third-party
    "github.com/labstack/echo/v4"
    "go.opentelemetry.io/otel"

    // Local
    "github.com/axyzlabs/contextd/pkg/auth"
    "github.com/axyzlabs/contextd/pkg/config"
)

// BAD - Unorganized
import (
    "github.com/axyzlabs/contextd/pkg/auth"
    "context"
    "github.com/labstack/echo/v4"
    "fmt"
)
```

## Code Formatting

### Always Run gofmt

```bash
# Format all Go files
gofmt -w .

# Or use goimports (adds/removes imports)
goimports -w .
```

### Use golangci-lint

```bash
# Run all linters
golangci-lint run

# Auto-fix issues
golangci-lint run --fix
```

## Summary Checklist

Before committing code, verify:

- [ ] Code formatted with `gofmt -w .`
- [ ] Tests written first (TDD red → green → refactor)
- [ ] Test coverage ≥80%
- [ ] No race conditions: `go test -race ./...`
- [ ] Errors wrapped with context
- [ ] No panics for runtime errors
- [ ] Context propagated through call chain
- [ ] Exported types/functions documented
- [ ] No credentials in code or logs
- [ ] OTEL spans for important operations
- [ ] Input validation at service boundary
- [ ] Security-first coding principles followed

## Related Standards

- **Architecture**: `docs/standards/architecture.md`
- **Testing**: `docs/standards/testing-standards.md`
- **Package Guidelines**: `docs/standards/package-guidelines.md`

## Go Resources

- **Effective Go**: https://golang.org/doc/effective_go
- **Go Code Review Comments**: https://github.com/golang/go/wiki/CodeReviewComments
- **Go Proverbs**: https://go-proverbs.github.io/
