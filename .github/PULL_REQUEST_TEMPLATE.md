## Summary

<!-- Provide a brief description of your changes -->

## Type of Change

<!-- Check all that apply -->

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Code refactoring (no functional changes)
- [ ] Security fix
- [ ] Test improvement
- [ ] Infrastructure/tooling change

## Related Issues

<!-- Link related issues using keywords: Closes #123, Fixes #456, Resolves #789 -->

Closes #

## Changes Made

<!-- Provide a bulleted list of changes -->

-
-
-

## Research & Design

<!-- Required for new features and significant changes -->

- [ ] Research completed and documented in `docs/research/`
- [ ] Architecture decision recorded (ADR) if applicable
- [ ] SDK/library research completed (if applicable)
- [ ] Design reviewed and approved

**Research Document**: <!-- Link to research doc if applicable -->

## Testing Performed

<!-- Describe the testing you've done -->

### Test Coverage

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Regression tests added (for bug fixes)
- [ ] Test skill created (for new features)
- [ ] Manual testing completed
- [ ] Test coverage maintained or improved (≥80%)

### Test Results

```bash
# Paste test output showing coverage
go test -v -cover -race ./...
```

### Persona Agent Testing

<!-- If applicable, which persona agents tested this? -->

- [ ] @qa-engineer - Comprehensive testing
- [ ] @developer-user - Workflow testing
- [ ] @security-tester - Security audit
- [ ] @performance-tester - Performance benchmarks

## Code Quality Checklist

- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] No new compiler warnings
- [ ] Error handling comprehensive
- [ ] OpenTelemetry instrumentation added (if applicable)
- [ ] Security best practices followed
- [ ] Performance considerations addressed

## Documentation

- [ ] Code comments updated
- [ ] README.md updated (if applicable)
- [ ] CLAUDE.md updated (if applicable)
- [ ] API documentation updated
- [ ] Examples added/updated
- [ ] CHANGELOG.md updated

## Breaking Changes

<!-- List any breaking changes and migration steps -->

**Breaking Changes**: None / Yes (describe below)

<!-- If yes, provide:
- What breaks
- Why it's necessary
- Migration guide
- Deprecation notice (if applicable)
-->

## Screenshots/Recordings

<!-- Add screenshots or recordings for UI changes, new features, or bug fixes -->

<!-- Optional: Add before/after screenshots for bug fixes -->

## Performance Impact

<!-- Describe any performance implications -->

- [ ] No performance impact
- [ ] Performance improved (provide benchmarks)
- [ ] Performance impact acceptable (justify)

## Security Considerations

<!-- Describe security implications, if any -->

- [ ] No security impact
- [ ] Security improved (describe)
- [ ] Security implications reviewed and documented

## Deployment Notes

<!-- Any special deployment considerations? -->

- [ ] No special deployment steps required
- [ ] Database migration required
- [ ] Configuration changes required
- [ ] Dependencies updated (run `go mod tidy`)

## Rollback Plan

<!-- How to rollback if issues are found? -->

## Additional Context

<!-- Any additional information that reviewers should know -->

## Reviewer Notes

<!-- What should reviewers focus on? Any specific concerns? -->

---

**Checklist before requesting review:**

- [ ] All tests pass locally
- [ ] Code coverage maintained or improved
- [ ] Documentation updated
- [ ] Self-reviewed the code
- [ ] No merge conflicts
- [ ] Commit messages are clear and follow conventions
- [ ] Branch is up to date with base branch

---

<!-- Optional: Add co-authors if applicable
Co-Authored-By: Name <email@example.com>
-->
