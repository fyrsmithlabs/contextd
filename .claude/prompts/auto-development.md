# Auto-Development Prompt

You are the golang-pro agent implementing a feature from a specification.

## Context
- Repository: {{ repository }}
- Spec Path: {{ spec_path }}
- Feature Name: {{ feature_name }}
- Related Issue: {{ issue_number }}
- Dry Run: {{ dry_run }}

## Your Role
Read and follow the instructions in `.claude/agents/golang-pro.md`.

## Tasks

### 1. Read Specification
Read the complete specification:
- Main spec: `{{ spec_path }}`
- Research docs: `docs/specs/{{ feature_name }}/research/`
- Decision docs: `docs/specs/{{ feature_name }}/decisions/`

Extract:
- Requirements (functional and non-functional)
- Architecture design
- API/Interface contracts
- Testing requirements (≥80% coverage)
- Performance targets
- Security considerations

### 2. Read Context Documents
- `CLAUDE.md` - Development philosophy, TDD requirements
- `pkg/CLAUDE.md` - Package guidelines and architecture
- `pkg/<feature>/CLAUDE.md` - Package-specific docs (if exists)
- `docs/standards/coding-standards.md` - Go coding standards
- `docs/standards/testing-standards.md` - Testing requirements
- `docs/standards/architecture.md` - Architecture patterns

### 3. Find or Create Implementation Issue
If issue number not provided:
- Search for related issues using GitHub MCP
- Create new issue if needed with title "Implement: {{ feature_name }}"
- Link issue to spec

### 4. Create Implementation Branch
- Branch name: `feature/<issue-number>-{{ feature_name }}`
- Base: main
- Ensure branch is up to date with main

### 5. Follow TDD Workflow (CRITICAL)

**Red Phase - Write Tests First**:
- Create test file: `pkg/<feature>/..._test.go`
- Write comprehensive tests covering all requirements
- Tests should fail initially (red phase)
- Target ≥80% coverage

**Green Phase - Implement**:
- Create implementation files in `pkg/<feature>/`
- Implement to make tests pass
- Follow Go best practices and coding standards
- Add OpenTelemetry instrumentation
- Add error handling and validation

**Refactor Phase**:
- Clean up code while keeping tests green
- Optimize performance
- Add documentation comments
- Ensure code quality

### 6. Package Structure
Create proper package structure:
```
pkg/{{ feature_name }}/
├── doc.go              # Package documentation
├── <feature>.go        # Main implementation
├── <feature>_test.go   # Test suite
├── interfaces.go       # Interface definitions
├── errors.go           # Custom errors
└── testdata/           # Test fixtures
```

### 7. Documentation
- Add package documentation (doc.go)
- Add godoc comments for exported functions
- Update README.md if needed
- Update IMPLEMENTATION-STATUS.md

### 8. Run Quality Checks
- `go build ./...` - Verify builds
- `go test ./...` - All tests pass
- `go test -race ./...` - No race conditions
- `go test -cover ./...` - Check coverage ≥80%
- `gofmt -w .` - Format code
- `golint ./...` - Linting
- `go vet ./...` - Vet checks
- `staticcheck ./...` - Static analysis

### 9. Commit Changes
Create clear, atomic commits following conventional commits:
- `feat(feature-name): add core implementation`
- `test(feature-name): add comprehensive test suite`
- `docs(feature-name): add package documentation`

### 10. Create Draft Pull Request
{{ dry_run_check }}
- Title: "feat: Implement {{ feature_name }}"
- Body with implementation checklist and coverage report
- Labels: `type:feature`, `ai:in-development`, `status:needs-review`
- Draft: true
- Link to related issue

**PR Description Template**:
```markdown
## Summary
Implements {{ feature_name }} as specified in {{ spec_path }}

## Implementation
- [x] Core functionality implemented
- [x] Tests written (≥80% coverage)
- [x] Documentation added
- [x] Quality checks passed

## Coverage Report
[Include go test -cover output]

## Related
Closes #{{ issue_number }}
```

### 11. Update Related Issue
{{ dry_run_check }}
- Add comment with PR link
- Include implementation summary
- Include test coverage report
- Update labels: Remove `ai:needs-dev`, Add `ai:in-development`
- **Do NOT close issue** - will auto-close when PR merges

## Output
- Implementation in `pkg/{{ feature_name }}/`
- Comprehensive tests with ≥80% coverage
- Draft pull request for review
- Updated issue with PR link

## Important Notes

**NEVER push directly to main**. Always create a feature branch and PR.
**NEVER use force-push** (`--force` or `--force-with-lease`).
**ALWAYS respect branch protection** rules and require human review.
