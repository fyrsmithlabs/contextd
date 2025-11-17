# Development Standards

This directory contains the development standards for contextd. These standards apply to ALL code in the project and must be followed rigorously.

## Standards Files

### [architecture.md](./architecture.md)
**Architectural patterns and design principles for contextd**

Key topics:
- Security-first design (Unix sockets, bearer tokens, credential management)
- Context efficiency (local-first, checkpoint+clear at 70%)
- Multi-tenant isolation (database-per-project)
- Component architecture (communication, security, configuration, observability)
- Vector store abstraction (universal interface)
- Service layer patterns
- Dual-mode operation (API + MCP)
- Embedding service architecture (TEI vs OpenAI)
- Server lifecycle and graceful shutdown
- Route structure and middleware stack
- Key design decisions and rationale

**When to read**: Before implementing any new feature or making architectural changes

### [coding-standards.md](./coding-standards.md)
**Go coding standards for contextd**

Key topics:
- Security-first coding (credential handling, constant-time comparison, input validation)
- Context efficiency (golden rule: 1 message = all operations, file organization)
- TDD requirements (write tests first, minimum 80% coverage)
- Naming conventions (avoid redundant package names)
- Error handling (always wrap with context, use errors.Is/As)
- Context propagation (always pass as first parameter)
- Concurrency patterns (goroutines, mutexes, race detection)
- Struct and function design
- Documentation requirements
- OpenTelemetry instrumentation
- Security patterns

**When to read**: Before writing ANY Go code

### [testing-standards.md](./testing-standards.md)
**Test-driven development and coverage requirements**

Key topics:
- TDD workflow (red → green → refactor)
- Coverage requirements (80% overall, 100% critical paths)
- Test organization and naming
- Test types (unit, integration, benchmark)
- Testing patterns (fixtures, helpers, mocking, table-driven)
- Race condition testing
- Error path testing
- Skill-first testing (contextd-specific)
- Persona agent testing
- Bug tracking with regression tests
- Pre-commit testing script
- CI/CD integration with Codecov

**When to read**: Before writing ANY tests (which should be BEFORE implementation)

### [package-guidelines.md](./package-guidelines.md)
**Template for package-specific CLAUDE.md files**

Key topics:
- Structure for package CLAUDE.md files
- How to reference root standards
- Package-specific rules and constraints
- Testing requirements per package
- Examples for different package types (http, repository, service)

**When to read**: When creating a new package or working in an existing package with its own CLAUDE.md

## Quick Start

### For New Contributors

1. **Read ALL standards** in this directory (estimated 30-45 minutes)
2. **Understand the workflow**: Standards → Specs → TDD → Implementation
3. **Reference frequently**: These standards are the source of truth
4. **Ask questions**: If unclear, ask before proceeding

### For Specific Tasks

| Task Type | Read These Standards (In Order) |
|-----------|--------------------------------|
| Architecture decisions | architecture.md → coding-standards.md |
| Any code changes | coding-standards.md → testing-standards.md → architecture.md |
| New packages | package-guidelines.md → architecture.md → coding-standards.md |
| Writing tests | testing-standards.md → [relevant feature spec] |
| Security changes | architecture.md (Security section) → coding-standards.md (Security section) |

## Standards vs. Specs

**Standards** (this directory):
- **Scope**: Template-wide rules that apply to ALL projects using this template
- **Purpose**: Define HOW to develop (coding style, architecture patterns, testing approach)
- **Examples**: How to write Go code, test requirements, architecture patterns
- **Changes**: Infrequent, requires careful consideration

**Specs** (`docs/specs/`):
- **Scope**: Project-specific feature specifications
- **Purpose**: Define WHAT to build (features, APIs, business logic)
- **Examples**: Authentication spec, Payment API spec, Database schema
- **Changes**: Frequent, as features are added/modified

## Critical Rules

### 1. Security First
- Unix socket only (0600 permissions)
- Bearer token authentication (constant-time comparison)
- No credentials in code or logs
- Validate all inputs
- See: `architecture.md` and `coding-standards.md` security sections

### 2. Context Efficiency
- Golden rule: "1 MESSAGE = ALL RELATED OPERATIONS"
- Local-first operations (<50ms response)
- Checkpoint+clear at 70% (never /compact)
- See: `architecture.md` and `coding-standards.md` context sections

### 3. Test-Driven Development
- Write tests FIRST (red → green → refactor)
- Minimum 80% coverage (100% for critical paths)
- All code must pass `go test -race ./...`
- See: `testing-standards.md`

### 4. Multi-Tenant Isolation
- Database-per-project physical isolation
- No cross-project data access
- See: `architecture.md` multi-tenant section

## Enforcement

These standards are enforced through:

1. **Code Review**: All PRs reviewed against these standards
2. **CI/CD**: Automated checks for coverage, linting, formatting
3. **Golang-Pro Skill**: Delegates all Go coding to enforce TDD and standards
4. **Pre-Commit Hooks**: `.scripts/pre-task-complete.sh` runs quality gates

## Updating Standards

**CRITICAL**: Changes to standards affect all current and future development.

**Process for updating standards**:
1. Create issue describing proposed change and rationale
2. Discuss with team/maintainers
3. Create PR with standard update
4. Update related code to comply (if needed)
5. Announce changes to team
6. Update this README if new standard added

**Minor fixes** (typos, clarifications, examples):
- Can be done via PR without issue
- Still requires review

## Related Documentation

- **Root CLAUDE.md**: `/home/dahendel/projects/contextd/CLAUDE.md`
- **Specs**: `docs/specs/` - Feature specifications
- **ADRs**: `docs/adr/` - Architecture decision records
- **Research**: `docs/research/` - Investigation and analysis

## Questions?

If you have questions about these standards:

1. Check if your question is answered in the standard itself
2. Check related standards (cross-references at bottom of each file)
3. Check specs in `docs/specs/` for feature-specific guidance
4. Ask in PR comments or issue discussion
5. Create issue for standard clarification if needed

## Summary

**Before doing ANY work**:

1. ✅ Read relevant standards from this directory
2. ✅ Read feature specs from `docs/specs/` (if applicable)
3. ✅ Check package CLAUDE.md (if working in specific package)
4. ✅ Follow TDD (tests first!)
5. ✅ Run quality gates before commit (`.scripts/pre-task-complete.sh`)
6. ✅ Verify coverage ≥ 80%
7. ✅ Pass all tests including race detector

**Remember**: Standards are the foundation. Specs define features. Tests prove correctness. Code implements solutions.
