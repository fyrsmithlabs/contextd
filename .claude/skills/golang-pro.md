# Golang Pro Skill

**Type**: Project Skill (gitignored)
**Purpose**: Expert Go development with enforced TDD, test coverage, and quality standards

## Overview

This skill implements expert-level Go development following strict TDD methodology. It is the **REQUIRED** skill for all Go code implementation in this repository.

## When to Use

**ALWAYS use this skill for**:
- Implementing new Go features
- Fixing Go bugs or issues
- Refactoring Go code
- Adding new packages or modules
- ANY Go code changes

**Usage Pattern**:
```
Use the golang-pro skill to [implement/fix/refactor] [description]
```

**Examples**:
- "Use the golang-pro skill to implement user authentication service"
- "Use the golang-pro skill to fix race condition in cache package"
- "Use the golang-pro skill to refactor database connection pool"

## What This Skill Does

### 1. Test-Driven Development (TDD)
- **RED**: Writes failing tests first based on requirements
- **GREEN**: Implements minimal code to pass tests
- **REFACTOR**: Improves code while maintaining passing tests

### 2. Quality Standards
- **Coverage**: Ensures ≥70% test coverage (≥80% preferred)
- **Tests**: Writes comprehensive tests including failure cases
- **Race Detection**: Runs `go test -race ./...`
- **Linting**: Runs gofmt, golint, go vet, staticcheck
- **Build Verification**: Ensures code compiles without errors

### 3. Code Standards
Follows all standards from:
- `docs/standards/coding-standards.md` - Go coding conventions
- `docs/standards/testing-standards.md` - Test requirements
- `docs/standards/architecture.md` - Architecture patterns

### 4. Documentation
- Updates CHANGELOG.md with changes
- Adds/updates code comments where needed
- Creates conventional commits

### 5. Verification
Before completing, runs:
```bash
go build ./...                          # Verify builds
go test ./... -coverprofile=coverage.out  # Run tests with coverage
go test -race ./...                     # Check race conditions
gofmt -w .                              # Format code
golint ./...                            # Lint
go vet ./...                            # Vet
staticcheck ./...                       # Static analysis
```

## Workflow

### Step 1: Understand Requirements
- Read specification from `docs/specs/` if exists
- Understand acceptance criteria
- Review related issues/PRs

### Step 2: Write Tests (RED Phase)
```go
// Example: pkg/auth/service_test.go
package auth

import (
    "testing"
)

func TestAuthenticateUser_ValidCredentials_ReturnsToken(t *testing.T) {
    // Arrange
    service := NewService()
    username := "testuser"
    password := "validpass"

    // Act
    token, err := service.Authenticate(username, password)

    // Assert
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
    if token == "" {
        t.Error("Expected token, got empty string")
    }
}

func TestAuthenticateUser_InvalidCredentials_ReturnsError(t *testing.T) {
    // Arrange
    service := NewService()
    username := "testuser"
    password := "wrongpass"

    // Act
    token, err := service.Authenticate(username, password)

    // Assert
    if err == nil {
        t.Error("Expected error for invalid credentials")
    }
    if token != "" {
        t.Error("Expected empty token for invalid credentials")
    }
}
```

### Step 3: Implement Code (GREEN Phase)
```go
// Example: pkg/auth/service.go
package auth

import (
    "errors"
    "fmt"
)

type Service struct {
    // fields
}

func NewService() *Service {
    return &Service{}
}

func (s *Service) Authenticate(username, password string) (string, error) {
    // Minimal implementation to pass tests
    if username == "" || password == "" {
        return "", errors.New("username and password required")
    }

    // Validate credentials
    if !s.validateCredentials(username, password) {
        return "", fmt.Errorf("invalid credentials for user: %s", username)
    }

    // Generate token
    token, err := s.generateToken(username)
    if err != nil {
        return "", fmt.Errorf("failed to generate token: %w", err)
    }

    return token, nil
}
```

### Step 4: Refactor (REFACTOR Phase)
- Extract common logic
- Improve naming
- Remove duplication
- Add comments for complex logic
- Ensure tests still pass

### Step 5: Verify Quality
```bash
# Run all quality checks
go build ./...
go test ./... -coverprofile=coverage.out -covermode=atomic
go test -race ./...
gofmt -w .
golint ./...
go vet ./...
staticcheck ./...

# Check coverage
go tool cover -func=coverage.out
# Ensure ≥70% coverage
```

### Step 6: Update Documentation
```markdown
# CHANGELOG.md
## [Unreleased]
### Added
- User authentication service with JWT tokens (#42)
  - Validates credentials against user repository
  - Generates secure JWT tokens
  - Includes comprehensive test suite (94.8% coverage)
```

### Step 7: Create Commit
```bash
git add .
git commit -m "feat(auth): implement user authentication service

Implements JWT-based authentication following TDD methodology.

Features:
- Credential validation
- JWT token generation
- Secure password hashing
- Comprehensive error handling

Tests:
- 94.8% coverage
- All edge cases covered
- Race conditions tested

Closes #42

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

## Best Practices

### Naming Conventions
**AVOID redundant package names**:
```go
// GOOD
package slack
type Client struct {}
func (c *Client) SendMessage() {}

// BAD - redundant
package slack
type SlackClient struct {}  // "Slack" is redundant
func (c *SlackClient) SendSlackMessage() {}  // "Slack" redundant
```

### Error Handling
**ALWAYS wrap errors with context**:
```go
// GOOD
if err != nil {
    return fmt.Errorf("failed to connect to database: %w", err)
}

// BAD - no context
if err != nil {
    return err
}
```

### Test Structure (AAA Pattern)
```go
func TestFunction_Condition_ExpectedBehavior(t *testing.T) {
    // Arrange - Setup test data
    input := "test"
    expected := "TEST"

    // Act - Execute function
    result := ToUpper(input)

    // Assert - Verify result
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

## Coverage Requirements

### Minimum Coverage
- **Overall**: ≥70% (enforced)
- **Preferred**: ≥80%
- **Critical Paths**: 100%

### What to Test
✅ **Must Test**:
- All public functions
- Error cases
- Edge cases (nil, empty, boundary values)
- Concurrent access (if applicable)

❌ **Skip Testing**:
- Generated code
- Simple getters/setters
- Third-party library wrappers (test integration instead)

## Common Patterns

### Table-Driven Tests
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -2, -3, -5},
        {"mixed signs", 2, -3, -1},
        {"zero", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Expected %d, got %d", tt.expected, result)
            }
        })
    }
}
```

### Mocking Interfaces
```go
type MockUserRepository struct {
    FindByUsernameFunc func(string) (*User, error)
}

func (m *MockUserRepository) FindByUsername(username string) (*User, error) {
    if m.FindByUsernameFunc != nil {
        return m.FindByUsernameFunc(username)
    }
    return nil, errors.New("not implemented")
}
```

## Troubleshooting

### Issue: Tests Failing
1. Check test setup (Arrange phase)
2. Verify implementation logic (Act phase)
3. Review assertions (Assert phase)
4. Run single test: `go test -v -run TestName`

### Issue: Low Coverage
1. Identify uncovered lines: `go tool cover -html=coverage.out`
2. Add tests for uncovered branches
3. Test error cases
4. Test edge cases

### Issue: Race Conditions
1. Run with race detector: `go test -race ./...`
2. Review concurrent access to shared state
3. Add proper synchronization (mutexes, channels)
4. Re-test with race detector

## Integration with Workflow

This skill integrates with:
- `/start-task` - Creates test template, then use this skill
- `/run-quality-gates` - Runs same checks this skill performs
- `/debug-issue` - Use this skill to implement fixes
- Code review workflow - Ensures standards are met

## Output Format

When complete, provide:
```
✅ Implementation Complete

Package: pkg/auth
Files Changed:
- pkg/auth/service.go (new)
- pkg/auth/service_test.go (new)
- CHANGELOG.md (updated)

Tests:
- 15 tests written
- 94.8% coverage
- All tests passing
- No race conditions

Quality Checks:
✅ Build successful
✅ Tests pass
✅ Coverage ≥70%
✅ Race detector clean
✅ Formatted (gofmt)
✅ Linted (golint)
✅ Vetted (go vet)
✅ Static analysis clean (staticcheck)

Commit: feat(auth): implement user authentication service
Ready for: /run-quality-gates quick
```

## Notes

- **NEVER skip tests** - TDD is mandatory
- **NEVER commit failing tests** - All tests must pass
- **NEVER skip quality checks** - Run all verifications
- **ALWAYS update CHANGELOG.md** - Document changes
- **ALWAYS use conventional commits** - Follow format

## Related Documentation

- `docs/standards/coding-standards.md` - Go coding standards
- `docs/standards/testing-standards.md` - Test requirements
- `docs/standards/architecture.md` - Architecture patterns
- `.scripts/pre-task-complete.sh` - Automated quality checks
