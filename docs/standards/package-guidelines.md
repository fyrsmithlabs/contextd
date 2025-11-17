# Package-Specific CLAUDE.md Guidelines

## Overview

Each package in this codebase should have its own CLAUDE.md file that:
1. References the root CLAUDE.md
2. Provides package-specific instructions
3. Defines package-specific patterns and constraints
4. Lists package-specific test requirements

## Template

Create a CLAUDE.md file in each package directory following this template:

```markdown
# Package: [Package Name]

## Quick Reference

**IMPORTANT**: Always refer to the root [CLAUDE.md](../../CLAUDE.md) for project-wide standards.

## Package Overview

[Brief description of this package's purpose and responsibilities]

## Applicable Specifications

Before working on this package, review these specifications:

- [docs/specs/architecture.md](../../docs/specs/architecture.md) - [Specific sections relevant to this package]
- [docs/specs/coding-standards.md](../../docs/specs/coding-standards.md) - All standards apply
- [docs/specs/testing-standards.md](../../docs/specs/testing-standards.md) - All standards apply
- [docs/specs/[specific-spec].md] - [Why this spec is relevant]

## Package-Specific Rules

### Naming Conventions

[Any package-specific naming rules beyond the global standards]

### Design Patterns

[Required design patterns for this package]

### Dependencies

**Allowed Dependencies:**
- [List of approved internal and external dependencies]

**Forbidden Dependencies:**
- [List of dependencies that should NOT be used in this package]

### Interface Requirements

[List the key interfaces this package must implement or consume]

### Error Handling

[Package-specific error handling patterns]

## Testing Requirements

### Test Coverage

- Minimum Coverage: [X%] (may be higher than project default for critical packages)
- Critical Paths: [List critical paths requiring 100% coverage]

### Test Types Required

- [ ] Unit tests for all public functions
- [ ] Integration tests for [specific scenarios]
- [ ] [Package-specific test types]

### Test Fixtures

[Location and usage of test fixtures specific to this package]

## Configuration

[Package-specific configuration requirements]

## Performance Requirements

[Package-specific performance constraints]

## Examples

### Good Example
```go
[Code example showing correct usage]
```

### Bad Example
```go
[Code example showing what NOT to do]
```

## Common Pitfalls

1. [Common mistake #1]
2. [Common mistake #2]

## Checklist

Before submitting PR for this package:

- [ ] Read root CLAUDE.md
- [ ] Read applicable specs from docs/specs/
- [ ] [Package-specific checklist items]
- [ ] All tests pass
- [ ] Coverage meets requirements
```

## Example Package CLAUDE.md Files

### Example 1: pkg/http/CLAUDE.md

```markdown
# Package: http

## Quick Reference

**IMPORTANT**: Always refer to the root [CLAUDE.md](../../CLAUDE.md) for project-wide standards.

## Package Overview

The http package provides HTTP server implementation and handler utilities for the application's REST API. It handles request routing, middleware, and response formatting.

## Applicable Specifications

Before working on this package, review these specifications:

- [docs/specs/architecture.md](../../docs/specs/architecture.md) - Presentation Layer architecture
- [docs/specs/coding-standards.md](../../docs/specs/coding-standards.md) - All standards apply
- [docs/specs/testing-standards.md](../../docs/specs/testing-standards.md) - All standards apply

## Package-Specific Rules

### Naming Conventions

- Handlers should be named `Handle[Resource][Action]` (e.g., `HandleUserCreate`)
- Middleware should be named descriptively (e.g., `AuthMiddleware`, `LoggingMiddleware`)

### Design Patterns

- Use dependency injection for all handler dependencies
- Middleware should follow the standard http.Handler pattern
- All handlers should accept context.Context

### Dependencies

**Allowed Dependencies:**
- `net/http` (standard library)
- `github.com/gorilla/mux` or similar router
- `internal/service` (for business logic)
- `pkg/logger` (for logging)

**Forbidden Dependencies:**
- Direct database access (use service layer)
- Business logic in handlers (belongs in service layer)

### Interface Requirements

All handlers must:
- Accept `context.Context` as first parameter
- Return structured errors via error response type
- Use consistent JSON response format

### Error Handling

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    int    `json:"code"`
    Details string `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, err error, code int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error:   err.Error(),
        Code:    code,
    })
}
```

## Testing Requirements

### Test Coverage

- Minimum Coverage: 80% (project default)
- Critical Paths: 100% coverage for authentication and authorization handlers

### Test Types Required

- [ ] Unit tests for all handlers using httptest
- [ ] Integration tests for full request/response cycles
- [ ] Middleware tests for each middleware function

### Test Fixtures

Example handler test:

```go
func TestHandleUserCreate(t *testing.T) {
    req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"email":"test@example.com"}`))
    w := httptest.NewRecorder()

    handler := NewUserHandler(mockService)
    handler.HandleUserCreate(w, req)

    res := w.Result()
    if res.StatusCode != http.StatusCreated {
        t.Errorf("expected status 201, got %d", res.StatusCode)
    }
}
```

## Performance Requirements

- Request timeout: 30 seconds maximum
- Response time: < 500ms (p95)
- Maximum request size: 10MB

## Examples

### Good Example: Handler with Dependency Injection

```go
type UserHandler struct {
    service UserService
    logger  Logger
}

func NewUserHandler(service UserService, logger Logger) *UserHandler {
    return &UserHandler{
        service: service,
        logger:  logger,
    }
}

func (h *UserHandler) HandleUserCreate(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, err, http.StatusBadRequest)
        return
    }

    user, err := h.service.CreateUser(ctx, req)
    if err != nil {
        h.logger.Error("failed to create user", "error", err)
        writeError(w, err, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(user)
}
```

### Bad Example: Handler with Business Logic

```go
// BAD - Don't put business logic in handlers
func (h *UserHandler) HandleUserCreate(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req)

    // BAD - Direct database access in handler
    user := &User{Email: req.Email}
    h.db.Save(user)

    // BAD - No error handling
    json.NewEncoder(w).Encode(user)
}
```

## Common Pitfalls

1. **Putting business logic in handlers** - Handlers should only handle HTTP concerns
2. **Ignoring errors** - Always check and handle errors appropriately
3. **Not using context** - Always propagate context through handler calls
4. **Direct database access** - Use service layer instead
5. **Inconsistent response formats** - Use standard response types

## Checklist

Before submitting PR for this package:

- [ ] Read root CLAUDE.md
- [ ] Read docs/specs/architecture.md (Presentation Layer)
- [ ] Handlers use dependency injection
- [ ] No business logic in handlers
- [ ] All errors are handled and logged
- [ ] Consistent response format used
- [ ] All tests pass
- [ ] httptest used for handler testing
- [ ] 80% coverage achieved
```

### Example 2: pkg/repository/CLAUDE.md

```markdown
# Package: repository

## Quick Reference

**IMPORTANT**: Always refer to the root [CLAUDE.md](../../CLAUDE.md) for project-wide standards.

## Package Overview

The repository package provides data access layer implementations for persistent storage. It abstracts database operations and provides a clean interface for the service layer.

## Applicable Specifications

Before working on this package, review these specifications:

- [docs/specs/architecture.md](../../docs/specs/architecture.md) - Repository Layer architecture
- [docs/specs/coding-standards.md](../../docs/specs/coding-standards.md) - All standards apply
- [docs/specs/testing-standards.md](../../docs/specs/testing-standards.md) - All standards apply

## Package-Specific Rules

### Naming Conventions

- Repository interfaces: `[Entity]Repository`
- Repository implementations: `[database][Entity]Repository` (e.g., `postgresUserRepository`)
- Methods: Use standard CRUD names (Create, FindByID, Update, Delete)

### Design Patterns

- Use Repository pattern for all data access
- Implement interfaces in service package, not repository package
- Use transaction objects for multi-operation transactions

### Dependencies

**Allowed Dependencies:**
- Database drivers (sql, postgres, etc.)
- Context package
- Internal domain types

**Forbidden Dependencies:**
- HTTP packages
- Service layer packages (repositories should not depend on services)
- Business logic

### Interface Requirements

Standard repository interface:

```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}
```

### Error Handling

Define sentinel errors:

```go
var (
    ErrNotFound      = errors.New("not found")
    ErrDuplicate     = errors.New("duplicate entry")
    ErrConstraint    = errors.New("constraint violation")
)
```

## Testing Requirements

### Test Coverage

- Minimum Coverage: 100% (critical path)
- All database operations must be tested

### Test Types Required

- [ ] Unit tests with mocked database
- [ ] Integration tests with test database
- [ ] Transaction rollback tests
- [ ] Error handling tests

### Test Fixtures

Use testcontainers for integration tests:

```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:15",
            ExposedPorts: []string{"5432/tcp"},
        },
        Started: true,
    })
    require.NoError(t, err)

    t.Cleanup(func() {
        container.Terminate(ctx)
    })

    // Return configured DB connection
}
```

## Performance Requirements

- Query timeout: 10 seconds maximum
- Use connection pooling
- Implement retry logic for transient errors

## Examples

### Good Example: Repository Implementation

```go
type postgresUserRepository struct {
    db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *postgresUserRepository {
    return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    err := r.db.QueryRowContext(ctx,
        "SELECT id, email, created_at FROM users WHERE id = $1",
        id,
    ).Scan(&user.ID, &user.Email, &user.CreatedAt)

    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("querying user: %w", err)
    }

    return &user, nil
}
```

### Bad Example: Repository with Business Logic

```go
// BAD - Don't put business logic in repository
func (r *postgresUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    r.db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = $1", id).Scan(&user)

    // BAD - Business logic doesn't belong here
    if user.IsActive {
        user.LastAccessedAt = time.Now()
        r.Update(ctx, &user)
    }

    return &user, nil
}
```

## Common Pitfalls

1. **Not using parameterized queries** - Always use placeholders to prevent SQL injection
2. **Putting business logic in repositories** - Keep repositories focused on data access
3. **Not handling sql.ErrNoRows** - Convert to domain-specific error
4. **Ignoring context** - Always respect context cancellation
5. **Not using transactions** - Use transactions for multi-step operations

## Checklist

Before submitting PR for this package:

- [ ] Read root CLAUDE.md
- [ ] Read docs/specs/architecture.md (Repository Layer)
- [ ] All queries use parameterized statements
- [ ] No business logic in repository methods
- [ ] Errors are wrapped with context
- [ ] Context is propagated correctly
- [ ] Integration tests with test database
- [ ] 100% test coverage
- [ ] Transaction handling tested
```

## Creating Package CLAUDE.md Files

When creating a new package:

1. Copy the template above
2. Fill in package-specific information
3. Reference relevant specifications
4. Define package rules and constraints
5. Create example tests
6. Document common pitfalls

## Package CLAUDE.md Locations

Each major package should have its own CLAUDE.md:

```
pkg/
├── http/
│   └── CLAUDE.md
├── service/
│   └── CLAUDE.md
├── repository/
│   └── CLAUDE.md
└── [package-name]/
    └── CLAUDE.md
```

## Benefits of Package-Specific CLAUDE.md

- **Focused guidance**: Package-specific instructions right where they're needed
- **Lower token count**: Root CLAUDE.md stays small by delegating to package files
- **Better context**: AI gets precise instructions for the code being modified
- **Maintainable**: Easier to update package-specific rules without affecting root file
- **Discoverable**: Developers find guidance in the package they're working on
