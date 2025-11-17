# Skill Authoring Guide

**Based on patterns from the [superpowers plugin](https://github.com/superpowers-labs/superpowers)** by @dmarx and contributors.

## Overview

This guide explains how to **author high-quality skills** for the contextd skills management system. While `SKILLS.md` covers the API and MCP tools for skill storage, this document focuses on **writing skills that work**.

**Core principle:** Skills are tested using TDD (Test-Driven Development) with AI agents as test subjects.

## Credits and Attribution

This authoring methodology is based on the excellent patterns developed by the **superpowers plugin** community:

- **Repository**: https://github.com/superpowers-labs/superpowers
- **Key Skills Referenced**:
  - `writing-skills` - TDD approach to skill creation
  - `testing-skills-with-subagents` - Pressure testing methodology
  - `using-superpowers` - Discovery and application patterns

**Thank you** to @dmarx and all superpowers contributors for developing these proven patterns.

## What is a Skill?

A **skill** is a reference guide for proven techniques, patterns, or tools that helps AI agents and developers apply effective approaches.

**Skills are:**
- Reusable techniques and patterns
- Reference guides and tools
- Proven workflows
- Knowledge artifacts

**Skills are NOT:**
- Narratives about solving one problem
- Project-specific procedures (use CLAUDE.md)
- Standard practices well-documented elsewhere

## Skill Types

### Technique Skills
Concrete methods with steps to follow.

**Examples**:
- Condition-based waiting for async tests
- Root cause tracing for debugging
- GitHub Actions workflow patterns

### Pattern Skills
Ways of thinking about problems.

**Examples**:
- Defense in depth for validation
- Test invariants for reliability
- Systematic debugging approach

### Reference Skills
API docs, syntax guides, tool documentation.

**Examples**:
- Docker command reference
- GitHub Actions syntax guide
- API endpoint documentation

## TDD for Skill Creation

**Skills ARE tested like code.** You write tests (pressure scenarios), watch them fail (baseline), write the skill (documentation), watch tests pass (compliance).

### The Iron Law

```
NO SKILL WITHOUT A FAILING TEST FIRST
```

This applies to:
- ✅ New skills - test before writing
- ✅ Skill edits - test before changing
- ✅ "Simple additions" - test first
- ✅ "Just documentation" - test first

**No exceptions.** Untested skills have issues. Always.

### RED-GREEN-REFACTOR Cycle

| Phase | Skill Testing | What You Do |
|-------|---------------|-------------|
| **RED** | Baseline test | Run scenario WITHOUT skill, watch agent fail |
| **Verify RED** | Capture rationalizations | Document exact failures verbatim |
| **GREEN** | Write skill | Address specific baseline failures |
| **Verify GREEN** | Pressure test | Run scenario WITH skill, verify compliance |
| **REFACTOR** | Plug holes | Find new rationalizations, add counters |
| **Stay GREEN** | Re-verify | Test again, ensure still compliant |

## Skill Structure

### Frontmatter (YAML)

```yaml
---
name: skill-name-with-hyphens
description: Use when [specific triggers] - [what it does in third person]
---
```

**Rules**:
- Only two fields: `name` and `description`
- Max 1024 characters total
- Name: Letters, numbers, hyphens only (no special chars)
- Description: Start with "Use when..." then explain what it does

**Good Description Examples**:

```yaml
# ✅ Technology-agnostic with clear triggers
description: Use when tests have race conditions, timing dependencies, or pass/fail inconsistently - replaces arbitrary timeouts with condition polling for reliable async tests

# ✅ Technology-specific with explicit context
description: Use when creating or modifying GitHub Actions workflows - provides security patterns, common gotchas, performance optimizations, and debugging techniques

# ✅ Problem-first, then solution
description: Use when errors occur deep in execution and you need to trace back to find the original trigger - systematically traces bugs backward through call stack to identify source
```

**Bad Description Examples**:

```yaml
# ❌ Too abstract, no triggers
description: For async testing

# ❌ First person
description: I can help with async tests

# ❌ Doesn't include when to use
description: Provides async testing patterns
```

### Content Structure

```markdown
# Skill Name

## Overview
What is this? Core principle in 1-2 sentences.

## When to Use
Bullet list with SYMPTOMS and use cases
When NOT to use

## Quick Reference
Table or bullets for scanning common operations

## Core Pattern (for techniques/patterns)
Before/after code comparison or step-by-step

## Implementation
Inline code for simple patterns
Link to file for heavy reference

## Common Mistakes
What goes wrong + fixes

## Real-World Impact (optional)
Concrete results from using this skill
```

## Claude Search Optimization (CSO)

Skills must be **discoverable** by AI agents searching for solutions.

### 1. Rich Description Field

The description is read by AI to decide "Should I load this skill?"

**Include**:
- Concrete triggers and symptoms
- Problem descriptions (not language-specific unless skill is)
- Technology context if skill is specific
- Both when to use AND what it does

### 2. Keyword Coverage

Use words AI would search for:
- **Error messages**: "ENOTEMPTY", "timeout", "race condition"
- **Symptoms**: "flaky", "inconsistent", "hanging"
- **Synonyms**: "timeout/hang/freeze", "cleanup/teardown"
- **Tools**: Actual commands, library names

### 3. Descriptive Naming

Use active voice, verb-first:
- ✅ `github-actions-workflows` not `gha-workflow-docs`
- ✅ `creating-skills` not `skill-creation`
- ✅ `testing-skills-with-subagents` not `subagent-skill-test`

Gerunds (-ing) work well for processes:
- `creating-skills`, `testing-skills`, `debugging-workflows`

### 4. Token Efficiency

Skills load into EVERY conversation where relevant. Every token counts.

**Target word counts**:
- Frequently-referenced skills: <200 words
- Other skills: <500 words
- Heavy reference: Separate file, link from skill

**Techniques**:

**Move details to tool help:**
```markdown
# ❌ Document all flags
tool supports --flag1, --flag2, --flag3...

# ✅ Reference help
tool supports multiple modes. Run --help for details.
```

**Use cross-references:**
```markdown
# ❌ Repeat workflow
When doing X, follow these 20 steps...

# ✅ Reference other skill
REQUIRED: Use other-skill-name for workflow.
```

**Compress examples:**
```markdown
# ❌ Verbose (42 words)
User: "How did we solve X?"
You: I'll search conversations.
[Dispatch agent with query...]

# ✅ Minimal (20 words)
User: "How did we solve X?"
You: Searching...
[Dispatch agent → synthesis]
```

## Testing Skills

### Pressure Scenarios

**Purpose**: Simulate realistic conditions where agents might skip/rationalize.

**Pressure types:**
1. **Time pressure**: "Need this in 5 minutes"
2. **Sunk cost**: "Spent 4 hours already"
3. **Authority**: "Manager says skip tests"
4. **Exhaustion**: "End of day, dinner in 30 min"
5. **Overconfidence**: "Done this before, simple"

**Combine 3+ pressures** for discipline-enforcing skills.

### Baseline Testing (RED Phase)

**Process:**
1. Create realistic scenario with pressures
2. Run WITHOUT the skill
3. Document agent's exact choices and rationalizations
4. Identify patterns in failures
5. Note which pressures trigger violations

**Example Scenario:**

```markdown
You spent 4 hours implementing a feature. It works perfectly.
You manually tested all edge cases. It's 6pm, dinner at 6:30pm.
Code review tomorrow at 9am. You didn't write tests yet.

Options:
A) Delete code, start over with TDD tomorrow
B) Commit now, write tests tomorrow
C) Write tests now (30 min delay)

Choose and explain.
```

**Document verbatim:**
- What choice did agent make?
- What rationalizations did they use?
- Which pressures influenced the decision?

### Writing Skill (GREEN Phase)

Write skill that addresses **specific baseline failures**.

**Don't**:
- Write generic advice
- Add hypothetical counters
- Over-explain simple concepts

**Do**:
- Address exact rationalizations from baseline
- Use agent's own words in rationalization tables
- Make rules explicit and unambiguous

### Closing Loopholes (REFACTOR Phase)

**For discipline-enforcing skills:**

1. **Explicit exceptions list:**
```markdown
## No Exceptions

- Not for "simple code"
- Not for "I already tested manually"
- Not for "tests after achieve same goal"
- Delete means delete
```

2. **Rationalization table:**
```markdown
| Excuse | Reality |
|--------|---------|
| "Too simple to test" | Simple code breaks. Test takes 30s. |
| "I'll test after" | Tests passing immediately prove nothing. |
```

3. **Red flags list:**
```markdown
## Red Flags - STOP

- Code before test
- "Already manually tested"
- "This is different because..."

All of these mean: Delete code. Start over.
```

### Re-testing

Run same scenarios WITH skill. Agent should now comply.

If new rationalizations appear:
1. Add explicit counters
2. Update rationalization table
3. Re-test until bulletproof

## Skill Content Guidelines

### Code Examples

**One excellent example beats many mediocre ones.**

Choose most relevant language:
- Testing: TypeScript/JavaScript
- System: Shell/Python
- Data: Python

**Good example**:
- Complete and runnable
- Well-commented (WHY not WHAT)
- From real scenario
- Shows pattern clearly

**Don't**:
- Implement in 5+ languages
- Create fill-in-blank templates
- Write contrived examples

### File Organization

**Self-contained** (everything inline):
```
skills/
  defense-in-depth/
    SKILL.md
```

**With reusable tool**:
```
skills/
  condition-based-waiting/
    SKILL.md
    example.ts
```

**With heavy reference**:
```
skills/
  github-actions-workflows/
    SKILL.md
    syntax-reference.md
    examples/
```

### Flowcharts

**Only use for**:
- Non-obvious decision points
- Process loops (might stop too early)
- "When to use A vs B" decisions

**Never use for**:
- Reference material → Use tables
- Code examples → Use markdown
- Linear instructions → Use numbered lists

## Common Anti-Patterns

### ❌ Narrative Example
```markdown
In session 2025-10-03, we found empty projectDir caused...
```
**Why bad**: Too specific, not reusable

### ❌ Multi-Language Dilution
```
skill/
  example.js
  example.py
  example.go
```
**Why bad**: Mediocre quality each, maintenance burden

### ❌ Code in Flowcharts
```dot
step1 [label="import fs"];
step2 [label="read file"];
```
**Why bad**: Can't copy-paste, hard to read

### ❌ Untested Skills
```markdown
# Just added this section because it seems useful
```
**Why bad**: Haven't verified agents actually need it

## Skill Creation Checklist

Use TodoWrite for EACH item:

**RED Phase:**
- [ ] Create pressure scenarios (3+ pressures for discipline skills)
- [ ] Run WITHOUT skill - document baseline verbatim
- [ ] Identify rationalization patterns

**GREEN Phase:**
- [ ] Name uses letters, numbers, hyphens only
- [ ] Description starts with "Use when..."
- [ ] Description in third person with triggers
- [ ] Keywords throughout for search
- [ ] Clear overview with core principle
- [ ] Address specific baseline failures
- [ ] Code inline OR separate file
- [ ] One excellent example
- [ ] Run WITH skill - verify compliance

**REFACTOR Phase:**
- [ ] Identify NEW rationalizations
- [ ] Add explicit counters (if discipline skill)
- [ ] Build rationalization table
- [ ] Create red flags list
- [ ] Re-test until bulletproof

**Quality Checks:**
- [ ] Flowchart only if non-obvious decision
- [ ] Quick reference table
- [ ] Common mistakes section
- [ ] No narrative storytelling
- [ ] Supporting files only for tools/heavy reference

**Storage in contextd:**
- [ ] Use `skill_create` MCP tool to store
- [ ] Include proper version, category, tags
- [ ] Add prerequisites and expected outcome

## Integration with contextd

Once skill is written and tested, store in contextd:

```bash
# Create skill in contextd
skill_create \
  name="Your Skill Name" \
  description="Use when [triggers] - [what it does]" \
  content="$(cat SKILL.md)" \
  version="1.0.0" \
  author="Your Name" \
  category="debugging|deployment|testing|etc" \
  prerequisites=["tool1", "tool2"] \
  expected_outcome="What success looks like" \
  tags=["tag1", "tag2", "tag3"]
```

**Categories**:
- `debugging` - Troubleshooting workflows
- `deployment` - Release and deployment
- `testing` - Test strategies and patterns
- `analysis` - Code review and analysis
- `monitoring` - Observability patterns
- `security` - Security procedures
- `performance` - Optimization techniques
- `refactoring` - Code improvement

## Related Documentation

- [SKILLS.md](SKILLS.md) - MCP API and tool usage
- [docs/specs/skills/SPEC.md](specs/skills/SPEC.md) - Skills system architecture

## Credits

This authoring methodology is based on patterns from the **superpowers plugin**:
- Repository: https://github.com/superpowers-labs/superpowers
- Primary contributors: @dmarx and community
- Skills referenced: `writing-skills`, `testing-skills-with-subagents`, `using-superpowers`

**Adaptations for contextd**:
- Integration with contextd MCP API
- Storage in Qdrant vector database
- Usage tracking and analytics
- Category and tag organization

Thank you to the superpowers community for developing and sharing these patterns!

## Summary

**Creating skills IS TDD for documentation:**

1. **RED**: Test WITHOUT skill - document failures
2. **GREEN**: Write skill addressing those failures
3. **REFACTOR**: Close loopholes, re-test

**Iron Law**: No skill without failing test first.

**Benefits**: Better quality, bulletproof against rationalization, proven effectiveness.

If you follow TDD for code, follow it for skills. Same discipline, same results.
