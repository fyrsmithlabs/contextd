# Agent Definitions

This directory contains agent definitions for multi-agent orchestration in contextd development.

## Available Agents

### From git-template (Generic Go Development)

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| **go-architect** | System design and architecture | Designing new features, refactoring architecture |
| **go-engineer** | Implementation and coding | Writing Go code (delegates to golang-pro skill) |

### Contextd-Specific Agents

#### Core Development

##### @golang-reviewer
Expert Go code reviewer specializing in security, performance, and idiomatic patterns.

**Use for:**
- Code review for security vulnerabilities
- Go best practices validation
- Concurrency safety review
- Performance optimization review
- Test coverage verification
- Pre-commit review

**Key skills:**
- Go security patterns (OWASP Go-SCP)
- Race condition detection
- Input validation review
- Error handling patterns
- golangci-lint/gosec expertise
- contextd architecture knowledge

##### @qdrant-specialist
Expert Qdrant vector database specialist for collection design and optimization.

**Use for:**
- Qdrant collection schema design
- HNSW parameter tuning
- Flexible payload strategies
- Quantization and memory optimization
- Search performance optimization

**Key skills:**
- Qdrant Go client expertise
- HNSW indexing strategies
- Payload filtering optimization
- Quantization techniques
- Schema-free design patterns
- Simpler deployment alternatives

##### @mcp-developer
Expert MCP developer for Model Context Protocol server implementation.

**Use for:**
- Implementing MCP server features
- Adding new MCP tools
- Protocol compliance and validation
- MCP integration patterns
- JSON-RPC implementation

**Key skills:**
- MCP server architecture
- Tool and resource implementation
- Protocol compliance testing
- Security controls
- Performance optimization

##### @cli-developer
Expert CLI developer for command-line tools and TUI applications.

**Use for:**
- Developing `ctxd` client commands
- TUI monitor implementation
- Shell completion scripts
- Command structure design
- Cross-platform CLI support

**Key skills:**
- CLI UX design
- Cobra/Viper frameworks
- Terminal UI (bubbletea, lipgloss)
- Shell completions
- Cross-platform compatibility

#### Infrastructure & Performance

##### @performance-engineer
Performance optimization specialist for API, embeddings, and database operations.

**Use for:**
- Optimizing embedding batch processing
- Vector database query performance
- API response time improvements
- Memory usage optimization
- Profiling and benchmarking

**Key skills:**
- Go profiling (pprof)
- Database query optimization
- Caching strategies
- Concurrent processing
- Performance testing

##### @security-auditor
Security specialist for authentication, authorization, and secure operations.

**Use for:**
- Unix socket security review
- Bearer token implementation
- Input validation
- Secrets management
- Security testing

**Key skills:**
- Authentication mechanisms
- Authorization patterns
- Secure coding practices
- Security testing
- Vulnerability assessment

#### Product & Project Management

##### @product-manager
Expert product manager for roadmap planning, issue grooming, and backlog management.

**Use for:**
- Issue triage and prioritization
- Roadmap alignment analysis
- Backlog grooming and organization
- Milestone planning
- Feature prioritization
- Technical debt management

**Key skills:**
- Issue grooming workflows
- Priority frameworks (RICE, MoSCoW)
- Roadmap planning and tracking
- Stakeholder communication
- Dependency management
- Strategic alignment

**Automated via:**
- `.github/workflows/issue-grooming.yml` (weekly automation)
- Manual invocation for specific grooming tasks
- Roadmap review and alignment checks

#### Documentation & Workflows

##### @documentation-engineer
Technical documentation specialist for API docs, user guides, and examples.

**Use for:**
- API documentation
- User guides and tutorials
- CLAUDE.md optimization
- Code documentation
- Example creation

**Key skills:**
- Technical writing
- API documentation
- Markdown/MDX
- Documentation testing
- User experience

##### @git-workflow-manager
Git workflow and release management specialist.

**Use for:**
- GitHub Actions workflows
- GoReleaser configuration
- Release automation
- Branch strategies
- CI/CD pipelines

**Key skills:**
- GitHub Actions
- GoReleaser
- Semantic versioning
- Release management
- CI/CD best practices

##### @tooling-engineer
Developer tooling specialist for build systems and development experience.

**Use for:**
- Makefile improvements
- Build scripts
- Development environment
- Testing infrastructure
- Linting and formatting

**Key skills:**
- Build systems (Make, Task)
- Go tooling
- Development automation
- Testing frameworks
- Code quality tools

### Testing Agents (Persona Agents)

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| **@qa-engineer** | Comprehensive testing | Execute test skills, edge cases, security |
| **@developer-user** | Workflow testing | Developer experience, realistic workflows |
| **@security-tester** | Security testing | Vulnerability testing, attack scenarios |
| **@performance-tester** | Performance testing | Load testing, benchmarks |

## Usage Pattern

### Standard Pattern
**"Have the [agent-type] agent [specific task]"**

Examples:
- "Have the go-architect agent design the authentication system"
- "Have the mcp-developer agent implement a new MCP tool for repository indexing"
- "Have the qa-engineer agent execute the MCP tool testing suite"
- "Have the security-tester agent audit the multi-tenant isolation"
- "Have the product-manager agent groom open issues"

### Agent-Specific Syntax
Some agents use `@` syntax:

```
@golang-reviewer review the authentication middleware changes
@mcp-developer help me add a new troubleshoot_v2 tool
@cli-developer improve the ctxd tui interface
@performance-engineer optimize the embedding batch processing
@product-manager groom issues for Phase 1 milestone
```

## Critical: Go Code Delegation

**ALL Go coding tasks MUST be delegated to the golang-pro skill:**

```
Use the golang-pro skill to [implement/fix/refactor] [description]
```

The golang-pro skill enforces:
- TDD (tests first, then implementation)
- Minimum 80% coverage
- Race detection
- Linting and formatting
- Conventional commits

**Do NOT write Go code directly** - always delegate to golang-pro.

## Agent Coordination

For complex tasks involving multiple agents:

1. **Architect first**: Design the solution
2. **Engineer implements**: Using golang-pro skill
3. **QA tests**: Execute test skills
4. **Security audits**: Review security implications
5. **Performance validates**: Benchmark critical paths

## Agent Selection Guide

| Task | Primary Agent | Supporting Agents |
|------|--------------|-------------------|
| Code review (Go) | @golang-reviewer | @security-auditor |
| Pre-commit review | @golang-reviewer | @performance-engineer |
| Security audit | @security-auditor | @golang-reviewer |
| Vector DB (Qdrant) | @qdrant-specialist | @performance-engineer |
| Add MCP tool | @mcp-developer | @documentation-engineer |
| Build ctxd feature | @cli-developer | @tooling-engineer |
| Optimize performance | @performance-engineer | @golang-reviewer |
| Write documentation | @documentation-engineer | - |
| Release automation | @git-workflow-manager | @tooling-engineer |
| Build improvements | @tooling-engineer | @cli-developer |
| Design architecture | go-architect | @golang-reviewer |
| Implement features | go-engineer (via golang-pro) | @golang-reviewer |
| Issue grooming | @product-manager | - |
| Roadmap alignment | @product-manager | go-architect |
| Backlog management | @product-manager | - |

## Creating New Agents

To add a new agent:

1. Create `.claude/agents/[agent-name].md`
2. Define:
   - Role and responsibilities
   - Activation criteria
   - Key specifications to reference
   - Decision frameworks
   - Examples (good/bad)
3. Update this README
4. Update root `CLAUDE.md` agent list

## Agent Templates

See existing agent files for structure:
- `go-architect.md` - Architecture agent template
- `go-engineer.md` - Implementation agent template
- `product-manager.md` - Product management agent template

## Best Practices

### Agent Specialization
- Keep agents focused on specific responsibilities
- Reference specs rather than duplicating content
- Document decision frameworks clearly
- Include activation criteria

### Agent Delegation
- Use specific, actionable language
- Provide context and constraints
- Reference relevant specs
- Set clear expectations

### Agent Coordination
- Plan multi-agent workflows upfront
- Establish clear handoff points
- Document dependencies between agents
- Verify output before next agent

## Related Documentation

- **Root CLAUDE.md**: Project-wide instructions
- **docs/standards/**: Development standards
- **docs/specs/**: Feature specifications
- **.claude/skills/**: Reusable skills (golang-pro, etc.)
- **.claude/commands/**: Slash commands

## Token Efficiency

Agents support token efficiency by:
- **On-demand loading**: Only load agent when needed
- **Reference specs**: Link to details, don't duplicate
- **Focused scope**: Each agent has clear boundaries
- **Progressive disclosure**: Load metadata first, details later

## Example Workflow

### Implementing New Feature

```
1. User: "Implement checkpoint tagging feature"

2. Have the go-architect agent design checkpoint tagging:
   - Database schema changes
   - API endpoints
   - Service layer design
   - See: docs/specs/checkpoint-tagging.md

3. Use the golang-pro skill to implement checkpoint tagging:
   - TDD: Write tests first
   - Implement service layer
   - Add API endpoints
   - Ensure 80% coverage

4. Have the qa-engineer agent test checkpoint tagging:
   - Execute checkpoint tagging test skill
   - Verify edge cases
   - Check error handling

5. Have the security-tester agent audit checkpoint tagging:
   - Verify multi-tenant isolation
   - Check input validation
   - Review permission model

6. Have the performance-tester agent benchmark checkpoint tagging:
   - Measure search performance
   - Test concurrent tag operations
   - Validate response times
```

### Issue Grooming Workflow

For maintaining healthy issue backlog:

```
1. Automated (Weekly):
   - GitHub Actions runs issue-grooming.yml every Monday
   - @product-manager agent analyzes all open issues
   - Compares issues against roadmap phases
   - Generates grooming report with recommendations

2. Manual Grooming:
   @product-manager groom issues for Phase 1, focusing on:
   - Missing labels and priorities
   - Milestone assignments
   - Roadmap alignment
   - Stale issue detection

3. Review and Apply:
   - Review grooming report recommendations
   - Apply label and milestone changes
   - Close or archive stale issues
   - Create missing issues for roadmap items
```

### Code Review Workflow

For comprehensive code review, use @golang-reviewer before committing:

```
# Before creating PR
@golang-reviewer review pkg/auth/middleware.go for security issues

# For new features
@golang-reviewer review pkg/checkpoint/* and verify test coverage

# For performance-critical changes
@golang-reviewer review pkg/embedding/batch.go focusing on performance
```

## Notes

- **Always use golang-pro** for Go code implementation
- **Reference standards** from `docs/standards/` before starting
- **Create test skills** for all new features
- **Document decisions** in ADRs when appropriate
- **Keep agents specialized** - avoid overlap
- Agents work best when given specific, focused tasks
- Provide relevant context (files, error messages, requirements)
- Agents can collaborate on complex tasks
- All agents follow contextd's security-first, context-optimization philosophy

## Questions?

If unclear about which agent to use:
1. Check this README for agent responsibilities
2. Review agent definition files
3. Check root CLAUDE.md for guidance
4. Ask before proceeding
