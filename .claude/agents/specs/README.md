# Agent Reference Specifications

This directory contains authoritative documentation references for each agent. Agents MUST consult these specs before troubleshooting or making recommendations.

## Purpose

- **Ground agents in actual documentation** (reduce hallucinations)
- **Provide authoritative sources** (SDK docs, provider docs, best practices)
- **Enable consistent recommendations** (same patterns across sessions)
- **Faster troubleshooting** (reference known solutions)

## Structure

```
specs/
├── README.md                    # This file
├── golang-spec.md              # Go best practices, stdlib, tools
├── opentelemetry-spec.md       # OTEL instrumentation, best practices
├── testing-spec.md             # Go testing patterns, coverage tools
├── echo-framework-spec.md      # Echo v4 API, middleware, patterns
├── security-spec.md            # OWASP, Go security, contextd patterns
└── contextd-architecture.md    # contextd-specific patterns and decisions
```

## Usage in Agents

### Agent Definition Pattern

```yaml
---
name: agent-name
description: Agent description
tools: Read, Grep, Bash
specs:
  - /specs/relevant-spec.md
  - /specs/another-spec.md
---

## Reference Documentation

Before troubleshooting, ALWAYS consult:
1. `/specs/primary-spec.md` - Primary technology reference
2. `/specs/contextd-architecture.md` - contextd-specific patterns
3. Project docs in `/docs/` - Implementation details
```

### Troubleshooting Protocol

```
1. Identify the issue
2. Consult relevant spec files
3. Search project docs for similar issues
4. Apply spec-documented patterns
5. Verify against contextd architecture
6. Provide solution with spec references
```

## Spec File Guidelines

### Required Sections

Each spec file should include:
- **Overview** - Technology summary
- **Key Concepts** - Core principles
- **Common Patterns** - Recommended approaches
- **Anti-Patterns** - What to avoid
- **Troubleshooting** - Common issues and solutions
- **References** - External documentation links

### Format

- Use markdown with clear headings
- Include code examples
- Link to official documentation
- Keep contextd-specific notes separate
- Update when docs/patterns change

## Maintenance

### When to Update

- New SDK version released
- Best practices evolve
- Common issues discovered
- contextd architecture changes
- Agent feedback indicates gaps

### Who Updates

- Agent developers
- Code reviewers
- Project maintainers
- After incident post-mortems

## Spec Coverage

| Agent | Primary Specs | Status |
|-------|--------------|--------|
| golang-reviewer | golang-spec.md, security-spec.md | ✅ |
| observability-architect | opentelemetry-spec.md | ✅ |
| test-strategist | testing-spec.md, golang-spec.md | ✅ |
| security-auditor | security-spec.md, contextd-architecture.md | ✅ |
| mcp-developer | contextd-architecture.md | 🚧 |
| cli-developer | golang-spec.md | 🚧 |

## Examples

### Good Spec Reference
```

Agent analysis:
2. Identified missing HNSW index (spec recommends for >500k vectors)
3. Verified against /specs/contextd-architecture.md collection sizes
4. Applied spec pattern: switched from IVF_FLAT to HNSW
5. Result: p95 latency reduced from 450ms to 80ms

```

### Bad Approach (No Spec Reference)
```
Issue: Slow search queries

Agent: "Try adding more memory" ❌
- No spec consultation
- Generic advice
- No contextd-specific analysis
- No documentation reference
```

---

**Principle**: Trust the specs. They contain proven patterns and solutions. Always reference them.
