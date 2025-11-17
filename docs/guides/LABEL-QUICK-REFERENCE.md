# GitHub Labels - Quick Reference Guide

**Last Updated**: 2025-11-04
**Total Labels**: 49
**Full Documentation**: [Research Document](../research/GITHUB-REPOSITORY-CONFIGURATION-RESEARCH.md)

## Quick Label Lookup

### When Creating an Issue

**Always Add**:
1. **Type** (`type:*`) - What kind of issue?
2. **Priority** (`priority:*`) - How urgent?

**Often Add**:
3. **Area** (`area:*`) - Which part of codebase?
4. **Phase** (`phase-*`) - Which roadmap phase?

**Auto-Applied**:
- `status:needs-triage` - Automatically added to new issues

### When Creating a PR

**Always Add**:
1. **Type** (`type:*`) - What kind of change?

**Auto-Applied**:
- **Area labels** - Based on files changed
- **Size labels** - Based on lines changed
- `status:needs-review` - When ready

## Label Categories

### Type Labels (What is this?)

| Label | When to Use | Example |
|-------|-------------|---------|
| `type:bug` | Fixing a bug | "Fix: API returns 500 on empty request" |
| `type:feature` | New functionality | "Add checkpoint compression feature" |
| `type:enhancement` | Improving existing feature | "Improve search performance by 40%" |
| `type:documentation` | Docs only | "Update README with new install steps" |
| `type:refactor` | Code cleanup | "Refactor auth package structure" |
| `type:performance` | Optimization | "Reduce memory usage in vectorstore" |
| `type:security` | Security fix/improvement | "Fix token validation bypass" |
| `type:test` | Test improvements | "Add integration tests for MCP" |

### Priority Labels (How urgent?)

| Label | SLA | When to Use |
|-------|-----|-------------|
| `priority:critical` | 24 hours | Service crash, data loss, security breach |
| `priority:high` | 7 days | Major feature broken, workaround exists |
| `priority:medium` | 30 days | Minor feature broken, inconvenient |
| `priority:low` | Best effort | Nice to have, cosmetic issues |

**Default**: If unsure, use `priority:medium`

### Area Labels (Which part?)

| Label | Component | Files/Directories |
|-------|-----------|-------------------|
| `area:mcp` | MCP server | `pkg/mcp/`, `cmd/contextd/mcp*.go` |
| `area:api` | REST API | `pkg/server/`, `pkg/handlers/` |
| `area:vectorstore` | Qdrant integration | `pkg/vectorstore/` |
| `area:auth` | Authentication | `pkg/auth/` |
| `area:config` | Configuration | `pkg/config/` |
| `area:observability` | Monitoring | `pkg/observability/`, `pkg/telemetry/` |
| `area:multi-tenant` | Multi-tenancy | `pkg/multitenant/` |
| `area:migration` | Migrations | Migration-related code |

**Auto-Applied**: PRs automatically get area labels based on files changed

### Status Labels (What's the state?)

| Label | Meaning | Who Applies |
|-------|---------|-------------|
| `status:needs-triage` | Needs initial review | Auto (new issues) |
| `status:needs-spec` | Needs specification | Manual/Auto |
| `status:spec-review` | Spec under review | Spec PR creation |
| `status:in-progress` | Active development | Developer/Auto |
| `status:blocked` | Blocked by dependency | Developer |
| `status:needs-review` | Ready for review | Developer/Auto |
| `status:changes-requested` | Changes needed | Reviewer/Auto |
| `status:approved` | Approved for merge | Reviewer/Auto |

**Workflow**: needs-triage → needs-spec → spec-review → in-progress → needs-review → approved

### AI Workflow Labels (Agent states)

| Label | Triggers | Applied By |
|-------|----------|------------|
| `ai:needs-spec` | `spec-creation.yml` | Manual/Product Manager |
| `ai:spec-created` | - | Workflow (after spec PR) |
| `ai:in-development` | `auto-development.yml` | Workflow (after spec merge) |
| `ai:qa-testing` | QA agent workflow | QA Engineer/Workflow |
| `ai:code-review` | Code review workflow | Reviewer Agent/Workflow |

**Key**: These labels trigger automated workflows - use carefully!

### Size Labels (How big?)

| Label | Lines Changed | Auto-Applied |
|-------|---------------|--------------|
| `size:XS` | 0-10 | ✅ Yes |
| `size:S` | 11-50 | ✅ Yes |
| `size:M` | 51-200 | ✅ Yes |
| `size:L` | 201-500 | ✅ Yes |
| `size:XL` | 500+ | ✅ Yes |

**Note**: Automatically applied to PRs by workflow

### Phase Labels (When?)

| Label | Timeframe | Focus |
|-------|-----------|-------|
| `phase-1` | Months 1-2 | Foundation |
| `phase-2` | Months 3-4 | Context Folding |
| `phase-3` | Months 5-6 | ReasoningBank |
| `phase-4` | Months 7-8 | Integration |

**Usage**: Align issues with product roadmap

### Meta Labels (Special purposes)

| Label | Meaning | When to Use |
|-------|---------|-------------|
| `good-first-issue` | Good for newcomers | Easy, well-defined issues |
| `help-wanted` | Community help needed | Need external contributions |
| `question` | Question/help request | Not a bug or feature |
| `wontfix` | Will not be fixed | Not planned, out of scope |
| `duplicate` | Duplicate issue | Already reported |
| `invalid` | Invalid issue | Not a real issue |
| `stale` | No activity | Auto-applied after 60 days |

## Common Scenarios

### Scenario 1: Found a Bug

```bash
gh issue create \
  --label "type:bug,priority:high,area:api,status:needs-triage" \
  --title "[Bug]: API returns 500 on empty checkpoint" \
  --template bug_report.yml
```

### Scenario 2: Proposing a Feature

```bash
gh issue create \
  --label "type:feature,priority:medium,area:mcp,phase-2" \
  --title "[Feature]: Add checkpoint compression" \
  --template feature_request.yml
```

### Scenario 3: Need a Specification

```bash
gh issue create \
  --label "ai:needs-spec,type:documentation,priority:high" \
  --title "[Spec]: ReasoningBank Integration" \
  --template spec_request.yml
```

### Scenario 4: Creating a Feature PR

```bash
gh pr create \
  --title "feat(mcp): add checkpoint compression" \
  --template feature.md \
  --label "type:feature,area:mcp"
```

### Scenario 5: Bug Fix PR

```bash
gh pr create \
  --title "fix(api): handle empty checkpoint requests" \
  --template bugfix.md \
  --label "type:bug,area:api"
```

## Label Automation

### What Gets Auto-Labeled?

**On Issue Creation**:
- `status:needs-triage` - All new issues

**On PR Creation**:
- **Area labels** - Based on files changed (see `.github/labeler.yml`)
- **Size labels** - Based on lines changed (see `.github/workflows/pr-size.yml`)

**On Workflow Triggers**:
- `ai:spec-created` - After spec PR created
- `ai:in-development` - After spec merged
- `status:in-progress` - When PR linked to issue
- `status:approved` - After code review approval

**On Stale Detection**:
- `stale` - After 60 days of inactivity

### Triggering Workflows with Labels

| Apply Label | Triggers Workflow | Result |
|-------------|-------------------|--------|
| `ai:needs-spec` | `spec-creation.yml` | Spec PR created |
| (spec merge) | `auto-development.yml` | Implementation PR created |
| `priority:critical` | Alert workflow | Team notified |
| `help-wanted` | Community outreach | Publicized to contributors |

## Label Best Practices

### DO

✅ Apply type label to every issue/PR
✅ Apply priority label to bugs and features
✅ Use specific area labels when possible
✅ Update status labels as work progresses
✅ Use AI workflow labels to trigger automation
✅ Apply phase labels for roadmap alignment

### DON'T

❌ Apply multiple type labels (choose the primary one)
❌ Change label meanings arbitrarily
❌ Use labels as comments ("todo", "remind me")
❌ Skip labels entirely
❌ Apply status labels manually (prefer workflow automation)
❌ Use AI labels without understanding triggers

## Searching by Labels

### GitHub UI

**Filter by single label**:
```
label:"type:bug"
```

**Filter by multiple labels** (AND):
```
label:"type:bug" label:"priority:critical"
```

**Exclude label**:
```
label:"type:bug" -label:"status:blocked"
```

### GitHub CLI

**List critical bugs**:
```bash
gh issue list --label "type:bug,priority:critical"
```

**List phase-1 features**:
```bash
gh issue list --label "type:feature,phase-1"
```

**List issues needing specs**:
```bash
gh issue list --label "ai:needs-spec"
```

## Getting Help

### Label Questions

- Not sure which label? → Start with `type:*` and `priority:*`, others can be added later
- Multiple types apply? → Choose the primary purpose
- Wrong label applied? → Remove and add correct one
- New label needed? → Create issue with suggestion

### Documentation

- **Quick reference**: This guide
- **Full details**: [Research Document](../research/GITHUB-REPOSITORY-CONFIGURATION-RESEARCH.md)
- **Implementation**: [Implementation Guide](../research/GITHUB-REPOSITORY-CONFIGURATION-IMPLEMENTATION-GUIDE.md)
- **Templates**: [Templates Guide](../../.github/TEMPLATES_GUIDE.md)

### Support Channels

- GitHub Issues (label: `question`)
- GitHub Discussions
- Team chat
- Review documentation first

## Cheat Sheet

**Most Common Labels**:
```
type:bug + priority:high + area:[component]
type:feature + priority:medium + phase-[n]
type:documentation + area:[component]
ai:needs-spec + type:feature
status:needs-review (PR ready)
status:blocked (waiting on dependency)
```

**Quick Copy-Paste**:
```bash
# Critical bug
--label "type:bug,priority:critical,area:api"

# High priority feature
--label "type:feature,priority:high,area:mcp,phase-2"

# Documentation update
--label "type:documentation,priority:low"

# Need spec
--label "ai:needs-spec,type:feature,priority:high"
```

---

**Quick Reference Version**: 1.0
**Last Updated**: 2025-11-04
**Total Labels**: 49

For complete documentation, see:
- [Full Research](../research/GITHUB-REPOSITORY-CONFIGURATION-RESEARCH.md)
- [Implementation Guide](../research/GITHUB-REPOSITORY-CONFIGURATION-IMPLEMENTATION-GUIDE.md)
- [Executive Summary](../research/GITHUB-CONFIG-EXECUTIVE-SUMMARY.md)
