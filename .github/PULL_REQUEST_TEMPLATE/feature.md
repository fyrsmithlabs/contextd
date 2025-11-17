## Feature Summary

<!-- Brief description of the new feature -->

## Related Issues

<!-- Link related issues using keywords: Closes #123, Implements #456 -->

Closes #

## Specification

<!-- Link to feature specification -->

**Specification**: `docs/specs/[feature-name]/SPEC.md`

## Type of Feature

<!-- Check the primary type -->

- [ ] MCP tool (new MCP functionality)
- [ ] API endpoint (REST API addition)
- [ ] Core functionality (internal improvement)
- [ ] CLI command (ctxd command)
- [ ] Integration (third-party service)
- [ ] Infrastructure (deployment, monitoring)

## Research & Design Phase

**Required for all features**

- [ ] SDK research completed and documented
- [ ] Research document: `docs/research/[feature]-research.md`
- [ ] Architecture decision recorded (if significant)
- [ ] ADR document: `docs/architecture/adr/[number]-[title].md`
- [ ] Design reviewed and approved by maintainers
- [ ] Specification merged: PR #___

### SDK Evaluation

<!-- If applicable, summarize SDK research -->

**Chosen Approach**: Custom implementation / SDK: [name]

**Rationale**: <!-- Why this approach? -->

**Alternatives Considered**:
- Option 1: [pros/cons]
- Option 2: [pros/cons]

**Research Document**: [link]

## Implementation Details

### Changes Made

<!-- Detailed list of changes -->

**Packages Added/Modified**:
- `pkg/[package-name]/` - [description]
- `cmd/[command-name]/` - [description]

**Key Files**:
- `[file-path]` - [purpose]
- `[file-path]` - [purpose]

### Architecture

<!-- High-level architecture description -->

**Components**:
- Component 1: [purpose]
- Component 2: [purpose]

**Integration Points**:
- Integrates with: [existing systems]
- Dependencies: [new dependencies]

### Configuration

<!-- New configuration options -->

**Environment Variables** (if applicable):
```bash
NEW_CONFIG_VAR=value  # Description
```

**Config File Changes** (if applicable):
```yaml
# Example configuration
new_section:
  option: value
```

## Test-Driven Development (TDD)

**TDD workflow completed**

- [ ] **RED Phase**: Tests written first
- [ ] **GREEN Phase**: Implementation complete
- [ ] **REFACTOR Phase**: Code optimized

### Test Coverage

- [ ] Unit tests added (`*_test.go`)
- [ ] Integration tests added (if applicable)
- [ ] Table-driven tests used
- [ ] Edge cases covered
- [ ] Error paths tested
- [ ] Test coverage ≥80% overall
- [ ] Critical paths 100% coverage

### Test Results

```bash
# Paste comprehensive test output
go test -v -cover -race ./...

# Coverage details
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

**Coverage Summary**:
- Overall: ___%
- Package 1: ___%
- Package 2: ___%

### Test Skills Created

<!-- New test skills for this feature -->

- [ ] Test skill created in contextd skills database
- [ ] Skill name: `[feature-name]-testing`
- [ ] QA engineer executed skill successfully

## Code Quality

### Pre-Review Checklist

- [ ] Code builds successfully (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] No race conditions (`go test -race ./...`)
- [ ] Code formatted (`gofmt -w .`)
- [ ] Linting passes (`golint ./...`)
- [ ] Static analysis passes (`go vet ./...`, `staticcheck ./...`)
- [ ] No new compiler warnings

### Code Standards

- [ ] Follows `docs/standards/coding-standards.md`
- [ ] Package naming follows guidelines
- [ ] Proper error handling (wrapped with %w)
- [ ] Context as first parameter
- [ ] No redundant package names in identifiers
- [ ] Input validation implemented
- [ ] Comments added for exported types/functions

### Security Review

- [ ] Input sanitization implemented
- [ ] No credentials in code
- [ ] Unix socket permissions correct (if applicable)
- [ ] Bearer token validation (if applicable)
- [ ] No SQL injection vulnerabilities
- [ ] Security considerations documented

## Observability

**OpenTelemetry instrumentation**

- [ ] Spans added for key operations
- [ ] Metrics exported for monitoring
- [ ] Context propagation implemented
- [ ] Resource attributes set correctly
- [ ] Error tracking configured

**Instrumentation Details**:
- Span operations: [list key spans]
- Metrics collected: [list metrics]
- Trace IDs propagated: Yes/No

## Documentation

### Code Documentation

- [ ] Package documentation (`doc.go`)
- [ ] Godoc comments for exported types
- [ ] Godoc comments for exported functions
- [ ] Complex logic commented
- [ ] Examples provided (if applicable)

### User Documentation

- [ ] README.md updated (if user-facing)
- [ ] CLAUDE.md updated (if workflow changes)
- [ ] Getting Started guide updated (if applicable)
- [ ] API documentation updated (if API changes)
- [ ] Examples added to `examples/` (if applicable)
- [ ] CHANGELOG.md entry added

### Integration Documentation

- [ ] MCP tool documented (if MCP feature)
- [ ] CLI command documented (if ctxd command)
- [ ] Configuration documented
- [ ] Migration guide (if breaking changes)

## Performance Impact

**Performance analysis**

- [ ] Benchmarks created
- [ ] Performance acceptable (<50ms for local-first operations)
- [ ] Memory usage measured
- [ ] No performance regressions

**Benchmark Results**:
```bash
# Paste benchmark output
go test -bench=. -benchmem ./...
```

**Performance Summary**:
- Operation latency: ___ms
- Memory allocation: ___MB
- Compared to baseline: +/- ___%

## Breaking Changes

**Breaking Changes**: None / Yes (describe below)

<!-- If yes, provide detailed migration guide -->

### Migration Guide

<!-- Step-by-step migration instructions -->

**What Breaks**:
- [Specific API/behavior that changes]

**Why Necessary**:
- [Justification]

**Migration Steps**:
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Deprecation Notice**:
- Old API deprecated in: v__.__.___
- Removal planned for: v__.__.___

## AI/Agent Workflow

**Agent involvement** (if applicable)

- [ ] Spec-writer agent created specification: PR #___
- [ ] Golang-pro skill used for implementation
- [ ] QA-engineer agent executed tests
- [ ] Code-reviewer agent approved
- [ ] Product-manager agent reviewed alignment

**Agent Outputs**:
- Specification: [link]
- Implementation approach: [link]
- Test results: [link]
- Review comments: [link]

## Code Review Readiness

### Status

- [ ] Self-review completed (read own code line-by-line)
- [ ] Pre-PR script executed (`./scripts/pre-pr.sh`)
- [ ] All checklists completed
- [ ] Ready for maintainer review

### Review Focus Areas

<!-- What should reviewers focus on? -->

**Please review**:
- [ ] Architecture decisions
- [ ] Security implications
- [ ] Performance impact
- [ ] Error handling
- [ ] Test coverage

**Specific Concerns**:
<!-- Any areas you're uncertain about? -->

## Rollback Plan

**Rollback procedure** (if issues found post-merge):

1. [Step 1]
2. [Step 2]
3. [Step 3]

**Feature Flags**: Yes/No
- Flag name: `[flag-name]` (if applicable)
- Default: enabled/disabled

## Additional Context

<!-- Any additional information reviewers should know -->

### Related Features

<!-- Links to related features or dependencies -->

- Related to: #___
- Depends on: #___
- Blocks: #___

### Future Work

<!-- Planned follow-ups or known limitations -->

- [ ] Future enhancement 1
- [ ] Future enhancement 2

### Screenshots/Demos

<!-- Screenshots, videos, or terminal recordings -->

<!-- If CLI tool: -->
```bash
# Example usage
$ ctxd [command] [flags]
```

---

**Pre-Review Checklist** (Complete before requesting review):

- [ ] All tests pass locally
- [ ] Coverage ≥80% (100% for critical paths)
- [ ] Documentation complete
- [ ] Code self-reviewed
- [ ] No merge conflicts
- [ ] Branch up-to-date with main
- [ ] Commit messages clear and conventional
- [ ] Co-authors attributed (if applicable)

---

🤖 **Generated with [Claude Code](https://claude.com/claude-code)**

<!-- Optional: Co-author attribution -->
<!-- Co-Authored-By: Claude <noreply@anthropic.com> -->
