# Package Guidelines - CLAUDE.md

See [../CLAUDE.md](../CLAUDE.md) for project overview and architecture.

## Package Philosophy

The `pkg/` directory contains **reusable, public-facing packages** that:
- Have NO dependencies on `internal/` or `cmd/`
- Can be imported by external projects
- Provide stable, documented APIs
- Include comprehensive tests
- Follow Go standard library conventions

**CRITICAL**: If a package is only used by this project, consider moving it to `internal/`.

## Package Structure

```
pkg/
├── analytics/      - Usage analytics and metrics collection
├── api/            - API models and client utilities
├── auth/           - Authentication (Bearer token, middleware)
├── backup/         - Backup and restore functionality
├── checkpoint/     - Session checkpoint management
├── config/         - Configuration loading and validation
├── embedding/      - Embedding generation (OpenAI/TEI)
├── mcp/            - Model Context Protocol server
├── remediation/    - Error remediation and matching
├── security/       - Security utilities (redaction, validation)
├── skills/         - Skills management system
├── telemetry/      - OpenTelemetry initialization
├── troubleshooting/ - AI-powered error diagnosis
├── validation/     - Request validation utilities
└── vectorstore/    - Vector database interface (abstraction)
```

## Package Design Patterns

### Standard Package Layout

Every package MUST follow this structure:

```
pkg/yourpackage/
├── yourpackage.go       # Main implementation, public API
├── models.go            # Data structures (if complex)
├── service.go           # Service layer (if applicable)
├── yourpackage_test.go  # Tests
├── README.md            # Package documentation (optional)
└── internal.go          # Internal helpers (unexported)
```

### Package-Level Documentation

Every package MUST have package-level documentation in the main file:

```go
// Package yourpackage provides functionality for X.
//
// This package is designed to be used by Y for Z purpose.
//
// Example usage:
//
//	svc := yourpackage.New(config)
//	result, err := svc.DoSomething(ctx, input)
//	if err != nil {
//	    return err
//	}
//
package yourpackage
```

### Public API Design

Public functions MUST:
- Accept `context.Context` as first parameter (if they do I/O)
- Return error as last return value
- Use clear, self-documenting names
- Have comprehensive documentation
- Handle edge cases gracefully

```go
// Good: Clear, context-aware, error-handling
func (s *Service) GetCheckpoint(ctx context.Context, id string) (*Checkpoint, error) {
    if id == "" {
        return nil, fmt.Errorf("checkpoint ID is required")
    }

    // Implementation with context support
    return s.fetch(ctx, id)
}

// Bad: No context, unclear name, poor error handling
func (s *Service) Get(id string) *Checkpoint {
    // Missing error handling
    // Missing context support
    // Unclear what is being "gotten"
    return s.fetch(id)
}
```

## Key Packages

### pkg/auth

**Purpose**: Bearer token authentication and middleware

**Files**:
- `auth.go` - Token generation, loading, middleware

**Key Functions**:
```go
// Generate or load token from filesystem
func GetOrCreateToken(path string) (string, error)

// Echo middleware for Bearer token authentication
func BearerAuthMiddleware(validToken string) echo.MiddlewareFunc
```

**Security Requirements**:
- Token files MUST have 0600 permissions
- Token generation uses crypto/rand (32 bytes → hex)
- Token comparison uses constant-time comparison
- Tokens NEVER logged or included in error messages

**Usage**:
```go
token, err := auth.GetOrCreateToken("/path/to/token")
if err != nil {
    return err
}

middleware := auth.BearerAuthMiddleware(token)
api.Use(middleware)
```

### pkg/config

**Purpose**: Configuration loading and environment variable management

**Files**:
- `config.go` - Config struct, Load function

**Key Types**:
```go
type Config struct {
    SocketPath    string
    TokenPath     string
    EmbeddingURL  string
    EmbeddingModel string
    OTELEndpoint  string
    OTELEnvironment string
}
```

**Configuration Priority**:
1. Environment variables (highest)
2. Hardcoded defaults (lowest)

**Environment Variables**:
```bash
CONTEXTD_SOCKET            - Unix socket path
CONTEXTD_TOKEN_PATH        - Token file path
EMBEDDING_BASE_URL         - Embedding service URL
EMBEDDING_MODEL            - Embedding model name
OTEL_EXPORTER_OTLP_ENDPOINT - OpenTelemetry collector
OTEL_SERVICE_NAME          - Service name for tracing
OTEL_ENVIRONMENT           - Environment (dev/prod)
```

**Usage**:
```go
cfg := config.Load()
fmt.Println(cfg.SocketPath)
```

### pkg/telemetry

**Purpose**: OpenTelemetry initialization (traces + metrics)

**Files**:
- `telemetry.go` - Init function, shutdown function

**Key Functions**:
```go
// Initialize OpenTelemetry with context
func Init(ctx context.Context, serviceName, environment, version string) (func(context.Context) error, error)
```

**Configuration**:
- Trace export: OTLP/HTTP (batch: 5s timeout, 512 spans)
- Metric export: OTLP/HTTP (60s interval)
- Propagation: W3C Trace Context + Baggage
- Resource attributes: service.name, service.version, deployment.environment

**Usage**:
```go
shutdown, err := telemetry.Init(ctx, "contextd", "production", "1.0.0")
if err != nil {
    return err
}
defer shutdown(context.Background())
```

**CRITICAL**: ALWAYS defer shutdown to flush traces/metrics before exit.

### pkg/checkpoint

**Purpose**: Session checkpoint management and search

**Files**:
- `models.go` - Checkpoint data structures
- `service.go` - Service implementation
- `checkpoint.go` - Core logic

**Key Types**:
```go
type Checkpoint struct {
    ID          string    `json:"id"`
    Timestamp   time.Time `json:"timestamp"`
    Summary     string    `json:"summary"`
    Context     string    `json:"context"`
    Metadata    map[string]string `json:"metadata"`
}

type Service struct {
    store     vectorstore.VectorStore
    embedding *embedding.Service
}
```

**Key Functions**:
```go
// Save checkpoint with automatic embedding generation
func (s *Service) Save(ctx context.Context, cp *Checkpoint) error

// Semantic search across checkpoints
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Checkpoint, error)

// List recent checkpoints
func (s *Service) List(ctx context.Context, limit int) ([]*Checkpoint, error)
```

**Design Decisions**:
- Embeddings generated automatically on save
- Search uses cosine similarity (threshold: 0.7)

**Usage**:
```go
cp := &checkpoint.Checkpoint{
    Summary: "Completed feature X",
    Context: "Added files A, B, C...",
}

if err := svc.Save(ctx, cp); err != nil {
    return err
}

results, err := svc.Search(ctx, "feature X", 10)
```

### pkg/remediation

**Purpose**: Error remediation storage and hybrid matching

**Files**:
- `models.go` - Remediation data structures
- `service.go` - Service implementation
- `matcher.go` - Hybrid matching algorithm
- `remediation.go` - Core logic

**Key Types**:
```go
type Remediation struct {
    ID          string    `json:"id"`
    ErrorMsg    string    `json:"error_msg"`
    Solution    string    `json:"solution"`
    Context     string    `json:"context"`
    Patterns    []string  `json:"patterns"`
}

type Service struct {
    store     vectorstore.VectorStore
    embedding *embedding.Service
    matcher   *Matcher
}
```

**Hybrid Matching Algorithm**:
- 70% semantic similarity (embedding cosine distance)
- 30% string matching (Levenshtein distance)
- Combined score threshold: 0.6
- Prioritizes exact error message matches

**Key Functions**:
```go
// Save remediation with pattern extraction
func (s *Service) Save(ctx context.Context, rem *Remediation) error

// Hybrid search (semantic + string matching)
func (s *Service) Search(ctx context.Context, errorMsg string, limit int) ([]*Remediation, error)

// Extract common error patterns
func ExtractPatterns(errorMsg string) []string
```

**Pattern Extraction**:
- File paths normalized (e.g., `/path/to/file.go:123` → `*.go:*`)
- Numbers replaced with placeholders (e.g., `port 8080` → `port *`)
- UUIDs/hashes replaced with placeholders
- Common error message templates identified

**Usage**:
```go
rem := &remediation.Remediation{
    ErrorMsg: "dial tcp 127.0.0.1:8080: connect: connection refused",
    Solution: "Start the server first: ./server",
}

if err := svc.Save(ctx, rem); err != nil {
    return err
}

matches, err := svc.Search(ctx, "connection refused", 5)
```

### pkg/embedding

**Purpose**: Embedding generation (supports OpenAI and TEI)

**Files**:
- `embedding.go` - Service implementation, multi-provider support

**Key Types**:
```go
type Service struct {
    client      *http.Client
    baseURL     string
    model       string
    apiKey      string // Optional (not needed for TEI)
}
```

**Supported Providers**:
1. **OpenAI** - `https://api.openai.com/v1` (requires API key)
2. **TEI** - `http://localhost:8080/v1` (local, no API key)

**Key Functions**:
```go
// Generate embedding for single text
func (s *Service) Generate(ctx context.Context, text string) ([]float32, error)

// Generate embeddings for multiple texts (batched)
func (s *Service) GenerateBatch(ctx context.Context, texts []string) ([][]float32, error)
```

**Configuration**:
```bash
# OpenAI (default)
EMBEDDING_BASE_URL=https://api.openai.com/v1
EMBEDDING_MODEL=text-embedding-3-small
OPENAI_API_KEY=sk-xxx

# TEI (recommended - no quotas)
EMBEDDING_BASE_URL=http://localhost:8080/v1
EMBEDDING_MODEL=BAAI/bge-small-en-v1.5
# No API key needed
```

**CRITICAL**:
- ALWAYS use batch generation for multiple texts (reduces API calls)
- ALWAYS respect context deadlines (set client timeout)
- TEI requires Docker: `docker-compose up -d tei`

**Usage**:
```go
svc := embedding.NewService(
    "http://localhost:8080/v1",
    "BAAI/bge-small-en-v1.5",
    "", // No API key for TEI
)

vec, err := svc.Generate(ctx, "Hello, world!")
```



**Files**:

**Key Types**:
```go
type Client struct {
    localFirst  bool
    clusterURI  string
}
```

**Collections**:
- `checkpoints` - Session checkpoints (dim: 1536)
- `remediations` - Error solutions (dim: 1536)
- `skills` - Skills/templates (dim: 1536)
- `documents` - Indexed repository files (dim: 1536)

**Key Functions**:
```go
// Create collection with vector index
func (c *Client) CreateCollection(ctx context.Context, name string, dim int) error

// Insert vectors with metadata
func (c *Client) Insert(ctx context.Context, collection string, vectors [][]float32, data []map[string]interface{}) error

// Vector similarity search
func (c *Client) Search(ctx context.Context, collection string, vector []float32, limit int) ([]map[string]interface{}, error)
```

**Local-First Mode**:
- Background goroutine syncs to cluster (when configured)
- Fallback to local if cluster unavailable

**Usage**:
```go
if err != nil {
    return err
}
defer client.Close()

err = client.Insert(ctx, "checkpoints", vectors, metadata)
```

### pkg/mcp

**Purpose**: Model Context Protocol server implementation

**Files**:
- `server.go` - MCP server, tool handlers

**Key Types**:
```go
type Server struct {
    services *Services // Reference to application services
    reader   *bufio.Reader
    writer   *bufio.Writer
}
```

**MCP Tools** (12 total):
1. `checkpoint_save` - Save session checkpoint
2. `checkpoint_search` - Semantic search checkpoints
3. `checkpoint_list` - List recent checkpoints
4. `remediation_save` - Store error solution
5. `remediation_search` - Find similar fixes
6. `skill_save` - Save reusable skill
7. `skill_search` - Search skills
8. `collection_create` - Create vector collection
9. `collection_delete` - Delete collection
10. `collection_list` - List collections
11. `index_repository` - Index repository
12. `status` - Service health

**Protocol**:
- Transport: stdio (stdin/stdout)
- Format: JSON-RPC 2.0
- Schema: MCP specification

**Key Functions**:
```go
// Start MCP server (blocking)
func (s *Server) Run(ctx context.Context) error

// Handle tool execution
func (s *Server) handleToolCall(ctx context.Context, tool Tool) (interface{}, error)
```

**Adding New Tools**:
```go
// 1. Add to tool list
{
    Name:        "new_tool",
    Description: "Imperative description",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param": map[string]interface{}{
                "type": "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param"},
    },
}

// 2. Add handler case
case "new_tool":
    var params NewToolParams
    if err := json.Unmarshal(tool.Arguments, &params); err != nil {
        return nil, fmt.Errorf("invalid arguments: %w", err)
    }
    return s.services.YourService.DoSomething(ctx, params)
```

### pkg/validation

**Purpose**: Request validation utilities

**Files**:
- `validation.go` - Validation functions

**Key Functions**:
```go
// Validate and bind request body to struct
func ValidateRequest(c echo.Context, v interface{}) error

// Validate struct using tags
func Validate(v interface{}) error
```

**Validation Tags**:
```go
type CreateCheckpointRequest struct {
    Summary  string            `json:"summary" validate:"required,min=1,max=500"`
    Context  string            `json:"context" validate:"max=10000"`
    Metadata map[string]string `json:"metadata" validate:"dive,keys,min=1,max=50,endkeys,min=1,max=500"`
}
```

**Usage**:
```go
var req CreateCheckpointRequest
if err := validation.ValidateRequest(c, &req); err != nil {
    return err // Echo handles error response
}
```

### pkg/security

**Purpose**: Security utilities (redaction, sanitization)

**Files**:
- `redact.go` - Redact sensitive data
- `redact_test.go` - Tests

**Key Functions**:
```go
// Redact sensitive patterns (API keys, tokens, passwords)
func Redact(text string) string

// Redact file paths containing usernames
func RedactPaths(text string) string
```

**Redacted Patterns**:
- API keys: `sk-...`, `key-...`
- Tokens: `Bearer ...`, `token: ...`
- Passwords: `password=...`
- Environment variables: `API_KEY=...`
- File paths: `/home/username/...` → `/home/***/...`

**Usage**:
```go
log.Info(security.Redact(errorMsg))
```

## Package Dependencies

**CRITICAL**: Packages MUST respect this dependency order:

```
Level 1 (no dependencies):
├── config
├── auth
└── security

Level 2 (depends on Level 1):
├── telemetry
├── embedding
└── validation

Level 3 (depends on Level 1-2):
└── vectorstore

Level 4 (depends on Level 1-3):
├── checkpoint
├── remediation
├── skills
└── backup

Level 5 (depends on Level 1-4):
├── troubleshooting
├── mcp
└── analytics
```

**Dependency Rules**:
- NEVER create circular dependencies
- NEVER import from `internal/` or `cmd/`
- Prefer interfaces over concrete types for dependencies
- Use dependency injection (constructor parameters)

## Testing Guidelines

### Test File Structure

Every package MUST have tests:

```go
// yourpackage_test.go
package yourpackage

import (
    "context"
    "testing"
)

func TestYourFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"empty input", "", "", true},
        {"invalid input", "bad", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := YourFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Coverage

Packages MUST maintain:
- >80% test coverage for public functions
- >60% overall coverage including error paths
- Example tests for documentation

### Running Tests

```bash
# All packages
go test ./pkg/...

# Specific package
go test ./pkg/checkpoint/

# With coverage
go test -cover ./pkg/...

# With race detection
go test -race ./pkg/...

```

## Adding New Packages

### 1. Create Package Directory

```bash
mkdir -p pkg/yourpackage
```

### 2. Create Main File

Create `pkg/yourpackage/yourpackage.go`:

```go
// Package yourpackage provides functionality for X.
//
// This package is designed to be used by Y for Z purpose.
//
// Example usage:
//
//	svc := yourpackage.New(config)
//	result, err := svc.DoSomething(ctx, input)
//
package yourpackage

import (
    "context"
    "fmt"
)

// Service provides X functionality.
type Service struct {
    // Dependencies
}

// New creates a new Service.
func New(deps ...interface{}) *Service {
    return &Service{}
}

// DoSomething performs the primary function.
func (s *Service) DoSomething(ctx context.Context, input string) (string, error) {
    if input == "" {
        return "", fmt.Errorf("input is required")
    }

    // Implementation
    return result, nil
}
```

### 3. Add Tests

Create `pkg/yourpackage/yourpackage_test.go`:

```go
package yourpackage

import (
    "context"
    "testing"
)

func TestDoSomething(t *testing.T) {
    svc := New()

    result, err := svc.DoSomething(context.Background(), "test")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result == "" {
        t.Error("expected non-empty result")
    }
}
```

### 4. Add Documentation

Add package documentation to main file and consider creating `pkg/yourpackage/README.md` for complex packages.

### 5. Update Dependencies

If the package depends on other packages, ensure they're at the correct dependency level.

## Common Patterns

### Service Pattern

Most packages follow the service pattern:

```go
type Service struct {
    // Dependencies injected via constructor
    store     Store
    logger    Logger
}

func NewService(store Store, logger Logger) *Service {
    return &Service{
        store:  store,
        logger: logger,
    }
}

func (s *Service) DoSomething(ctx context.Context, input Input) (Output, error) {
    // Implementation with context support
}
```

### Interface-Based Design

Define interfaces for dependencies:

```go
// Define interface in your package
type Store interface {
    Save(ctx context.Context, data Data) error
    Load(ctx context.Context, id string) (Data, error)
}

// Accept interface in constructor
func NewService(store Store) *Service {
    return &Service{store: store}
}
```

### Error Handling

Use wrapped errors with context:

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Context Support

ALL I/O operations MUST respect context:

```go
func (s *Service) Fetch(ctx context.Context, id string) (*Data, error) {
    // Check context before expensive operation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Pass context to dependencies
    return s.store.Load(ctx, id)
}
```

## Performance Considerations

- **Connection Pooling**: Reuse HTTP clients and database connections
- **Batch Operations**: Group multiple operations when possible
- **Caching**: Cache frequently accessed data (with TTL)
- **Context Timeouts**: Set reasonable timeouts for operations
- **Memory**: Avoid holding large objects in memory unnecessarily

## Related Documentation

- **Server**: See [../cmd/contextd/CLAUDE.md](../cmd/contextd/CLAUDE.md)
- **Client**: See [../cmd/ctxd/CLAUDE.md](../cmd/ctxd/CLAUDE.md)
- **Architecture**: See [../docs/ARCHITECTURE-RECOMMENDATIONS.md](../docs/ARCHITECTURE-RECOMMENDATIONS.md)
