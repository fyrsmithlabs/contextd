# Plugin Streamlining Design

**Date:** 2026-01-14
**Status:** Ready for Implementation
**Author:** Claude + User collaboration

---

## Overview

Streamline the contextd Claude plugin to reduce maintenance burden, simplify onboarding, and enforce workflows through hooks rather than documentation alone.

## Goals

1. **Reduce maintenance burden** - De-duplicate tool documentation across files
2. **Simplify onboarding** - Fewer skills/agents to understand
3. **Reduce agent sprawl** - Consolidate 5 agents into 2
4. **Enforce workflows** - Use hooks to remind/enforce rather than rely on documentation

## Changes Summary

| Component | Before | After | Reduction |
|-----------|--------|-------|-----------|
| Agents | 5 (1,927 lines) | 2 (~250 lines) | -87% |
| Skills | 12 (2,860 lines) | 6 (~670 lines) | -77% |
| Commands | 16 | 9 | -44% |
| Hooks | 3 | 6 | +3 (enforcement) |

---

## Agents

### Before (5 agents)
- `contextd-task-executor.md` (93 lines)
- `task-orchestrator.md` (342 lines)
- `systematic-debugging.md` (446 lines)
- `refactoring-agent.md` (522 lines)
- `architecture-analyzer.md` (529 lines)

### After (2 agents)

#### 1. contextd-task-agent.md (~150 lines)

**Purpose:** Unified agent for debugging, refactoring, architecture analysis, and general contextd-enforced work.

**Key features:**
- Mode detection based on task type (Debug, Refactor, Analyze, Execute)
- SRE-aligned debug flow (Triage → Examine → Diagnose → Test → Cure)
- References shared includes instead of duplicating tool docs

**Modes:**
| Task Type | Mode | Focus |
|-----------|------|-------|
| Fix bug, error, failure | **Debug** | SRE troubleshooting process |
| Restructure, rename, extract | **Refactor** | checkpoint → incremental changes |
| Understand codebase, onboard | **Analyze** | repository_index → semantic_search |
| General development | **Execute** | memory_search → work → memory_record |

#### 2. contextd-orchestrator.md (~100 lines)

**Purpose:** Power-user agent for multi-agent workflows with context-folding.

Keeps orchestration patterns but references shared includes.

---

## Skills

### Before (12 skills)
`checkpoint-workflow`, `context-folding`, `conversation-indexing`, `cross-session-memory`, `error-remediation`, `policies`, `project-onboarding`, `repository-search`, `secret-scrubbing`, `self-reflection`, `session-lifecycle`, `using-contextd`, `consensus-review`, `writing-claude-md`

### After (6 skills)

| Skill | Merges | Purpose | Lines |
|-------|--------|---------|-------|
| **using-contextd** | `using-contextd` + `cross-session-memory` + `repository-search` | Canonical tool reference | ~120 |
| **contextd-workflow** | `checkpoint-workflow` + `session-lifecycle` + `error-remediation` | Pre/work/post-flight flow | ~100 |
| **context-folding** | Keep (trimmed) | Isolated sub-task execution | ~80 |
| **project-setup** | `project-onboarding` + `writing-claude-md` + `policies` | Onboarding + CLAUDE.md | ~100 |
| **consensus-review** | Keep (trimmed) | Multi-agent code/PR validation | ~150 |
| **self-reflection** | Keep (trimmed) | Improving skills/agents/docs | ~120 |

### Dropped
- `policies` → Fold into project-setup
- `secret-scrubbing` → Implementation detail, not user-facing
- `conversation-indexing` → Document in using-contextd as optional

---

## Commands

### Before (16 commands)
`checkpoint`, `consensus-review`, `diagnose`, `help`, `init`, `install`, `onboard`, `policies`, `reflect`, `remember`, `resume`, `search`, `status`, `statusline`, `test-skill`

### After (9 commands)

| Keep | Reason |
|------|--------|
| `/contextd:search` | Core - semantic search |
| `/contextd:remember` | Core - quick memory record |
| `/contextd:checkpoint` | Core - save/list/resume |
| `/contextd:diagnose` | Core - error diagnosis |
| `/contextd:status` | Core - check contextd health |
| `/contextd:init` | Setup - index new repo (merge onboard) |
| `/contextd:reflect` | Meta - self-reflection |
| `/contextd:consensus-review` | Review - multi-agent code review |
| `/contextd:help` | Discovery - list commands |

| Drop | Reason |
|------|--------|
| `install` | One-time, document in README |
| `onboard` | Merge into `init --full` |
| `policies` | Fold into project-setup skill |
| `statusline` | Niche, document in help |
| `test-skill` | Developer tool, not user-facing |
| `resume` | **Replaced by SessionStart hook** |

---

## Hooks

### New Hook Architecture

| Hook | Event | Purpose |
|------|-------|---------|
| `session-start.md` | SessionStart | Check checkpoints, offer resume |
| `prompt-reminder.md` | UserPromptSubmit | Remind: memory_search before work |
| `context-monitor.sh` | UserPromptSubmit | Warn at 50/75/90% context |
| `precompact.sh` | PreCompact | **Force** checkpoint before compaction |
| `stop-reminder.md` | Stop | Remind: memory_record after work |
| `error-diagnose.md` | PostToolUse (Bash fail) | Trigger SRE debug flow on errors |

### Context Monitoring (Three-Tier Approach)

Based on research into Claude Code internals:

#### Tier 1: `context_window` from Claude Code (Most Accurate)
Since v2.0.65, Claude Code provides context data in statusline JSON:
```json
{
  "context_window": {
    "total_input_tokens": 176000,
    "total_output_tokens": 5000,
    "context_window_size": 200000
  }
}
```

#### Tier 2: JSONL Transcript Parsing (Fallback)
Parse `~/.claude/projects/**/*.jsonl` for most recent assistant message:
```
total = input_tokens + cache_read_input_tokens + cache_creation_input_tokens
adjusted = total * 1.30  # 30% fudge for hidden context
```

#### Tier 3: PreCompact Hook (Safety Net)
Fires reliably at ~78% context utilization.

#### Thresholds
| % of 160k usable | Action |
|------------------|--------|
| < 50% | No output |
| 50-74% | Optional checkpoint suggestion |
| 75-89% | Recommend checkpoint |
| 90%+ | Force checkpoint with template |

### Sources
- [How to Calculate Claude Code Context](https://codelynx.dev/posts/calculate-claude-code-context)
- [ccstatusline context_window feature](https://github.com/sirmalloc/ccstatusline/issues/126)
- [Claude Code Status Line Docs](https://code.claude.com/docs/en/statusline)

---

## File Structure

```
.claude-plugin/
├── plugin.json
├── agents/
│   ├── contextd-task-agent.md      # Unified (debug/refactor/analyze/execute)
│   └── contextd-orchestrator.md    # Multi-agent workflows
├── skills/
│   ├── using-contextd/SKILL.md     # Canonical tool reference
│   ├── contextd-workflow/SKILL.md  # Pre/work/post-flight
│   ├── context-folding/SKILL.md    # Isolated sub-tasks
│   ├── project-setup/SKILL.md      # Onboarding + CLAUDE.md
│   ├── consensus-review/SKILL.md   # Multi-agent code review
│   └── self-reflection/SKILL.md    # Plugin/skill improvement
├── includes/
│   ├── contextd-protocol.md        # 27-line canonical protocol
│   ├── tool-reference.md           # Single source for tool docs
│   └── common-patterns.md          # Shared patterns
├── hooks/
│   ├── hooks.json                  # ✅ Updated
│   ├── session-start.md            # ✅ Created
│   ├── prompt-reminder.md          # ✅ Created
│   ├── context-monitor.sh          # ✅ Created (three-tier)
│   ├── precompact.sh               # ✅ Updated
│   ├── stop-reminder.md            # ✅ Created
│   └── error-diagnose.md           # ✅ Created
└── commands/
    └── (9 commands)
```

---

## Implementation Plan

### Phase 1: Hooks (Complete)
- [x] Create `context-monitor.sh` with three-tier approach
- [x] Update `hooks.json` with all hooks
- [x] Create `session-start.md`
- [x] Create `prompt-reminder.md`
- [x] Create `stop-reminder.md`
- [x] Create `error-diagnose.md`
- [x] Update `precompact.sh`

### Phase 2: Includes
- [ ] Create `includes/tool-reference.md` - single source for all MCP tools
- [ ] Create `includes/common-patterns.md` - shared patterns (SRE debug, etc.)
- [ ] Keep `includes/contextd-protocol.md` (already exists, 27 lines)

### Phase 3: Agents
- [ ] Create `contextd-task-agent.md` (consolidate 4 agents)
- [ ] Slim `contextd-orchestrator.md` (reference includes)
- [ ] Delete old agents after validation

### Phase 4: Skills
- [ ] Create/update `using-contextd/SKILL.md`
- [ ] Create/update `contextd-workflow/SKILL.md`
- [ ] Trim `context-folding/SKILL.md`
- [ ] Create `project-setup/SKILL.md`
- [ ] Trim `consensus-review/SKILL.md`
- [ ] Trim `self-reflection/SKILL.md`
- [ ] Delete merged/dropped skills

### Phase 5: Commands
- [ ] Update `init.md` to include onboard functionality
- [ ] Delete dropped commands
- [ ] Update `help.md` with new command list

### Phase 6: Validation
- [ ] Test all hooks fire correctly
- [ ] Verify context monitoring accuracy
- [ ] Test agent mode detection
- [ ] Validate skill workflows

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Users rely on old agent names | Add deprecation notice, redirect |
| Context monitor inaccuracy | Three-tier fallback + PreCompact safety net |
| Hook fatigue (too many reminders) | Hooks are gentle, non-blocking |
| Lost functionality in consolidation | Skills exist for reinforcement, not new features |

---

## Success Criteria

1. **Maintenance:** Single place to update tool documentation
2. **Onboarding:** New users understand 2 agents + 6 skills
3. **Enforcement:** Hooks fire at correct events
4. **Context:** Checkpoints saved before context loss
5. **Debugging:** SRE flow triggered automatically on errors
