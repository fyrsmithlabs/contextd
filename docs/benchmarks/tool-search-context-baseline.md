# Tool Search Context Usage Baseline

**Date**: 2026-01-16
**Purpose**: Measure context usage before/after tool_search implementation

---

## Baseline Measurements (Before Tool Search)

### Test 1: Existing Project (contextd)

**Project**: `/Users/dahendel/projects/contextd`
**Memories in contextd**: 108

#### Current MCP Tools Loaded (ALL tools sent to context):

| Category | Tools | Count |
|----------|-------|-------|
| Memory | memory_search, memory_record, memory_feedback, memory_outcome, memory_consolidate | 5 |
| Checkpoint | checkpoint_save, checkpoint_list, checkpoint_resume | 3 |
| Remediation | remediation_search, remediation_record | 2 |
| Repository | semantic_search, repository_search, repository_index | 3 |
| Troubleshoot | troubleshoot_diagnose | 1 |
| Folding | branch_create, branch_return, branch_status | 3 |
| Conversation | conversation_index, conversation_search | 2 |
| Reflection | reflect_report, reflect_analyze | 2 |
| **TOTAL** | | **21** |

**Estimated Token Usage (all tools loaded)**:
- ~500-800 tokens per tool definition (name, description, schema)
- 21 tools × ~650 tokens = **~13,650 tokens** for tool definitions alone

---

### Test 2: Fresh Project (Empty)

**Project**: `/tmp/test-fresh-project`

**Observation**: Fresh project initialized at `/tmp/test-fresh-project`
**Background agent**: Testing context in fresh project environment

---

## Expected Impact of Tool Search

### Before Tool Search (Current State)
- **ALL 21 contextd tools** loaded into context on every session
- Estimated **~13,650 tokens** consumed by tool definitions
- No dynamic loading - same overhead for simple and complex tasks

### After Tool Search (Target State)
- **3 core tools** always loaded: `tool_search`, `semantic_search`, `memory_search`
- **18 tools deferred** - only loaded when discovered via search
- Estimated **~1,950 tokens** for core tools (3 × 650)
- **~85% reduction** in baseline tool token usage

### Measurement Plan

| Metric | Before | After | Method |
|--------|--------|-------|--------|
| Tools in initial context | 21 | 3 | Count non-deferred tools |
| Tool definition tokens | ~13,650 | ~1,950 | Estimate from schemas |
| Context available for work | Lower | Higher | Token budget math |
| Tool discovery latency | N/A | +1 call | Measure search overhead |

---

## Test Sessions

### Session 1: Main contextd project
- **Path**: `/Users/dahendel/projects/contextd`
- **Status**: Active (this session)
- **contextd memories**: 108

### Session 2: Fresh project
- **Path**: `/tmp/test-fresh-project`
- **Status**: Background agent running
- **contextd memories**: 0 (new project)

---

## Notes

- contextd HTTP server running on localhost:9090
- All tools currently registered without defer_loading
- tool_search implementation exists in worktree (needs merge)
- After implementation, re-run these tests and compare


---

## Fresh Project Test Results

**Path**: `/tmp/test-fresh-project`
**Contents**: Single README.md, fresh git init

### Tools Loaded (Even for Empty Project)

| Category | Count | Notes |
|----------|-------|-------|
| Claude Code Built-in | 11 | Bash, Read, Edit, Write, Glob, Grep, etc. |
| contextd MCP | 20 | ALL tools loaded regardless of project |
| GitHub MCP | ~30 | ALL GitHub operations available |
| **TOTAL** | **~61** | Loaded into every session |

### Key Finding: 100% Tool Loading Waste

For a fresh project that needs ZERO MCP tools:
- **20 contextd tools loaded** - none needed
- **~30 GitHub tools loaded** - none needed
- **Estimated wasted tokens**: 5,000-25,000

### Baseline Token Budget Analysis

```
Total Context Window:    200,000 tokens (Claude)
Tool Definitions:        -25,000 tokens (estimate)
System Prompt:            -5,000 tokens (estimate)
Conversation History:    -10,000 tokens (grows)
─────────────────────────────────────────────
Available for Work:      160,000 tokens

With tool_search (3 core tools):
Tool Definitions:         -1,950 tokens
Available for Work:      183,050 tokens (+23,050 gained)
```

### After tool_search Implementation

| Metric | Before | After | Savings |
|--------|--------|-------|---------|
| contextd tools loaded | 20 | 3 | 85% |
| Tool definition tokens | ~13,000 | ~1,950 | 85% |
| Fresh project overhead | 100% waste | ~0% waste | 100% |

