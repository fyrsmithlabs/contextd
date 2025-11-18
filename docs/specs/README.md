# Project Specifications

This directory contains project-specific feature specifications for contextd.

## Overview

**Specs define WHAT to build** (features, APIs, business logic). They are different from Standards, which define HOW to build (coding patterns, architecture principles).

## Specs vs. Standards

| Type | Location | Purpose | Examples | Changes |
|------|----------|---------|----------|---------|
| **Standards** | `docs/standards/` | HOW to develop | Coding style, test requirements, architecture patterns | Infrequent, template-wide |
| **Specs** | `docs/specs/` | WHAT to build | Feature specs, API designs, business logic | Frequent, project-specific |

## Spec-Driven Development

### Workflow

1. **Feature request** → Create or update spec
2. **Review spec** → Architect, stakeholders approve
3. **Break down** → `/spec-to-issue` creates GitHub issues
4. **Implement** → Follow TDD using golang-pro skill
5. **Verify** → Implementation matches spec
6. **Update spec** → If implementation reveals needed changes

### When to Create a Spec

**Always create a spec for:**
- ✅ New features or major enhancements
- ✅ New API endpoints or MCP tools
- ✅ Complex business logic
- ✅ Integration with external systems
- ✅ Multi-tenant isolation changes
- ✅ Security-sensitive operations

**Don't need a spec for:**
- ❌ Bug fixes (unless they expose design gaps)
- ❌ Simple refactoring
- ❌ Documentation updates
- ❌ Test improvements
- ❌ Minor tweaks to existing features

### Spec Structure

Each spec should include:

```markdown
# Feature: [Name]

## Overview
Brief description of the feature.

## Motivation
Why we're building this.

## Requirements
- Functional requirements
- Non-functional requirements (performance, security, etc.)

## Architecture
High-level design and component interactions.

## API Design
Endpoints, request/response formats, error handling.

## Data Model
Database schema, vector collections, payloads.

## Security Considerations
Auth, permissions, input validation, multi-tenant isolation.

## Testing Strategy
Test scenarios, coverage requirements, edge cases.

## Implementation Plan
Phases, dependencies, estimated effort.

## Open Questions
Unresolved decisions, areas needing discussion.

## References
Related ADRs, research docs, standards.
```

## Current Specs

### Core Features

**To be created as needed. Examples:**
- `checkpoint-tagging.md` - Add tags to checkpoints for better organization
- `remediation-v2.md` - Enhanced remediation with confidence scoring
- `skills-management.md` - CRUD operations for skills
- `agent-templates.md` - Reusable agent configuration templates
- `repository-indexing-v2.md` - Enhanced code indexing with AST parsing

### Security

**MVP Security Posture** (Trusted Network):
- No authentication required
- HTTP transport on port 8080
- Deploy on trusted network (VPN, internal, or localhost)
- Use SSH tunnel for remote access: `ssh -L 8080:localhost:8080 user@server`

**POST-MVP Security** (Production):
- Bearer token authentication
- TLS via reverse proxy
- Rate limiting and OAuth/SSO

### Resolutions

Error resolution specifications go in `resolutions/` subdirectory:

```
docs/specs/resolutions/
├── error-[brief-name].md
└── ...
```

These document:
- Problem description
- Root cause analysis
- Proposed solution
- Testing approach
- Prevention measures

## Creating a Spec

### Option 1: Manual Creation

1. Copy template above to `docs/specs/[feature-name].md`
2. Fill in each section
3. Get review from architect/team
4. Create GitHub issue: `/create-spec-issue [feature-name]`
5. Convert to tasks: `/spec-to-issue [feature-name]`

### Option 2: Agent-Assisted

```
Have the spec-writer agent create a specification for [feature description]
```

The spec-writer agent will:
- Ask clarifying questions
- Research similar features
- Draft comprehensive spec
- Include security considerations
- Suggest testing approach

### Option 3: Slash Command

```
/create-spec-issue [feature-name]
```

This creates a GitHub issue requesting the spec be written.

## Spec Review Process

1. **Draft**: Author creates initial spec
2. **Technical Review**: Architect reviews design
3. **Security Review**: Security auditor checks considerations
4. **Approval**: Stakeholders sign off
5. **Implementation**: Convert to issues, start work
6. **Updates**: Refine spec as implementation progresses

## Spec Quality Checklist

Before considering a spec complete:

- [ ] Overview clearly explains the feature
- [ ] Motivation justifies why we're building it
- [ ] Requirements are specific and testable
- [ ] Architecture diagram included (if complex)
- [ ] API design follows REST conventions
- [ ] Data model includes multi-tenant considerations
- [ ] Security section addresses threats
- [ ] Testing strategy covers edge cases
- [ ] Implementation plan is realistic
- [ ] Open questions documented
- [ ] References link to relevant standards/ADRs

## Integration with Development Workflow

### Starting a New Feature

```bash
# 1. Check if spec exists
ls docs/specs/my-feature.md

# 2. If not, create spec issue
/create-spec-issue my-feature

# 3. Once spec is written and approved
/spec-to-issue my-feature

# 4. Select an issue from the created tasks
/start-task <issue-number>

# 5. Implement following TDD
Use the golang-pro skill to implement [feature]

# 6. Reference spec during implementation
# Keep docs/specs/my-feature.md open
```

### During Implementation

- **Question arises**: Document in spec's "Open Questions"
- **Design change needed**: Update spec, get approval
- **Better approach found**: Update spec, explain rationale
- **Edge case discovered**: Add to spec's requirements

## Spec Maintenance

### Keeping Specs Current

- Update specs when requirements change
- Mark deprecated features in spec
- Link to implementation (PRs, commits)
- Document lessons learned

### Archiving Specs

When a feature is deprecated:
1. Move spec to `docs/specs/archive/`
2. Add deprecation notice at top
3. Link to replacement spec (if any)

## Examples

### Good Spec Example

See `docs/research/WEB-SCRAPING-COLLY-MIGRATION.md` for an example of thorough research and specification.

### Minimal Spec Example

For simple features:

```markdown
# Feature: Checkpoint Export

## Overview
Add ability to export checkpoints as JSON.

## Requirements
- GET /api/v1/checkpoints/:id/export returns JSON
- Include all checkpoint fields
- Multi-tenant isolation enforced

## Security
- Bearer token required
- Only owner's checkpoints accessible

## Testing
- Unit test export function
- Integration test API endpoint
- Test multi-tenant isolation
```

## Related Documentation

- **Standards**: `docs/standards/` - HOW to develop
- **ADRs**: `docs/adr/` - Architecture decisions
- **Research**: `docs/research/` - Investigation and analysis
- **Root CLAUDE.md**: Project instructions

## Questions?

If you have questions about specs:
1. Check this README
2. Review existing specs in this directory
3. Check standards in `docs/standards/`
4. Ask in issue or PR discussion

## Summary

**Specs are essential for**:
- Clear communication of requirements
- Alignment between stakeholders
- Guidance during implementation
- Validation that implementation matches intent
- Documentation for future maintenance

**Remember**:
- Specs define WHAT, standards define HOW
- Write specs before implementation
- Keep specs updated as you build
- Reference specs in PR descriptions
- Use specs to validate completion
