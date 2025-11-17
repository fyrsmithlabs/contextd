# BPE Tokenizer - Parallel Execution Plan

**Total Tasks**: 12 (IDs 17-28)
**Serial Time**: 27 hours
**Parallel Time**: 16 hours
**Time Savings**: 41%

---

## Dependency Graph

```
Phase 1 (2 hours parallel):
  Task 17 (Interface) ──┬──> Task 18 (Interface Tests)
                        │
  Task 19 (Types) ──────┤
                        │
  Task 22 (Vocab) ──────┘

Phase 2 (4 hours parallel):
  Task 18 (Interface Tests) ─────> [Complete]
  Task 20 (BPE Core) ────────────> [Needs Task 19]
  Task 23 (Vocab Tests) ─────────> [Needs Task 22]

Phase 3 (3 hours):
  Task 21 (BPE Tests) ───────────> [Needs Task 20]

Phase 4 (3 hours parallel):
  Task 24 (OpenAI Tokenizer) ────> [Needs Tasks 19, 20, 22]
  Task 26 (TEI Tokenizer) ───────> [Needs Tasks 19, 22]

Phase 5 (2 hours parallel):
  Task 25 (OpenAI Tests) ────────> [Needs Task 24]
  Task 27 (TEI Tests) ───────────> [Needs Task 26]

Phase 6 (2 hours):
  Task 28 (Documentation) ───────> [Needs ALL tasks]
```

---

## Phase 1: Foundation (Parallel - 2 hours)

**Run these 3 tasks in PARALLEL**:

### Task 17: Define Tokenizer Interface
- **File**: `pkg/embedding/tokenizer/tokenizer.go`
- **Time**: 1 hour
- **Dependencies**: None ✅ START NOW
- **What**: Define Tokenizer interface, TokenizerType constants
- **Command**: `task-master set-status --id=17 --status=in-progress`

### Task 19: Define Common Types
- **File**: `pkg/embedding/tokenizer/types.go`
- **Time**: 1 hour
- **Dependencies**: Task 17 ⚠️ NEEDS INTERFACE
- **What**: Config struct, Merge struct, New() factory
- **Command**: `task-master set-status --id=19 --status=in-progress`

### Task 22: Implement Vocabulary Management
- **File**: `pkg/embedding/tokenizer/vocab.go`
- **Time**: 2 hours
- **Dependencies**: None ✅ START NOW
- **What**: loadVocabulary, loadMergeRules, VocabularyCache
- **Command**: `task-master set-status --id=22 --status=in-progress`

**Parallelization**: Tasks 17 and 22 can start immediately. Task 19 starts after Task 17 (minimal delay).

---

## Phase 2: Core & Tests (Parallel - 4 hours)

**Run these 3 tasks in PARALLEL after Phase 1**:

### Task 18: Create Interface Test Suite
- **File**: `pkg/embedding/tokenizer/tokenizer_test.go`
- **Time**: 2 hours
- **Dependencies**: Task 17 ✅ READY
- **What**: Mock tokenizer, comprehensive test cases
- **Command**: `task-master set-status --id=18 --status=in-progress`

### Task 20: Implement BPE Core Algorithm
- **File**: `pkg/embedding/tokenizer/bpe.go`
- **Time**: 4 hours
- **Dependencies**: Task 19 ✅ READY
- **What**: applyBPE, chunkText, aggregateEmbeddings
- **Command**: `task-master set-status --id=20 --status=in-progress`

### Task 23: Create Vocabulary Tests
- **File**: `pkg/embedding/tokenizer/vocab_test.go`
- **Time**: 2 hours
- **Dependencies**: Task 22 ✅ READY
- **What**: Test vocabulary loading, caching, errors
- **Command**: `task-master set-status --id=23 --status=in-progress`

**Parallelization**: All 3 tasks can run simultaneously after Phase 1.

---

## Phase 3: BPE Tests (Serial - 3 hours)

**Single task after Phase 2**:

### Task 21: Create BPE Algorithm Tests
- **File**: `pkg/embedding/tokenizer/bpe_test.go`
- **Time**: 3 hours
- **Dependencies**: Task 20 ✅ READY
- **What**: Test BPE functions, UTF-8, benchmarks
- **Command**: `task-master set-status --id=21 --status=in-progress`

**Note**: Must wait for Task 20 (BPE implementation) to complete.

---

## Phase 4: Tokenizer Implementations (Parallel - 3 hours)

**Run these 2 tasks in PARALLEL after Phase 3**:

### Task 24: Implement OpenAI Tokenizer
- **File**: `pkg/embedding/tokenizer/openai.go`
- **Time**: 3 hours
- **Dependencies**: Tasks 19, 20, 22 ✅ ALL READY
- **What**: OpenAITokenizer struct, BPE-based, 8191 limit
- **Command**: `task-master set-status --id=24 --status=in-progress`

### Task 26: Implement TEI Tokenizer
- **File**: `pkg/embedding/tokenizer/tei.go`
- **Time**: 3 hours
- **Dependencies**: Tasks 19, 22 ✅ ALL READY (NOT BPE!)
- **What**: TEITokenizer struct, WordPiece, 512 limit
- **Command**: `task-master set-status --id=26 --status=in-progress`

**Parallelization**: Both can run simultaneously. TEI doesn't need BPE (Task 20).

---

## Phase 5: Implementation Tests (Parallel - 2 hours)

**Run these 2 tasks in PARALLEL after Phase 4**:

### Task 25: Create OpenAI Tokenizer Tests
- **File**: `pkg/embedding/tokenizer/openai_test.go`
- **Time**: 2 hours
- **Dependencies**: Task 24 ✅ READY
- **What**: Test OpenAI tokenizer, validate against API
- **Command**: `task-master set-status --id=25 --status=in-progress`

### Task 27: Create TEI Tokenizer Tests
- **File**: `pkg/embedding/tokenizer/tei_test.go`
- **Time**: 2 hours
- **Dependencies**: Task 26 ✅ READY
- **What**: Test TEI tokenizer, WordPiece validation
- **Command**: `task-master set-status --id=27 --status=in-progress`

**Parallelization**: Both can run simultaneously after their implementations.

---

## Phase 6: Documentation (Serial - 2 hours)

**Final task after ALL others complete**:

### Task 28: Create Package Documentation
- **File**: `pkg/embedding/tokenizer/CLAUDE.md`
- **Time**: 2 hours
- **Dependencies**: ALL TASKS (17-27) ✅ MUST COMPLETE FIRST
- **What**: Complete CLAUDE.md with all sections
- **Command**: `task-master set-status --id=28 --status=in-progress`

**Note**: Must wait for all 11 previous tasks to complete.

---

## Quick Start Guide

### Option 1: Single Developer (Serial)
```bash
# Phase 1
task-master set-status --id=17 --status=in-progress
# ... complete Task 17 ...
task-master set-status --id=17 --status=done

# Continue through phases sequentially
```

### Option 2: Multiple Developers/Agents (Parallel)
```bash
# Phase 1 - Start 3 agents in parallel
# Agent 1:
task-master set-status --id=17 --status=in-progress

# Agent 2 (immediately):
task-master set-status --id=22 --status=in-progress

# Agent 3 (after Agent 1 completes Task 17):
task-master set-status --id=19 --status=in-progress

# Phase 2 - Start 3 agents in parallel
# Agent 1: task-master set-status --id=18 --status=in-progress
# Agent 2: task-master set-status --id=20 --status=in-progress
# Agent 3: task-master set-status --id=23 --status=in-progress

# Continue pattern...
```

### Option 3: Git Worktrees (Recommended for Parallel)
```bash
# Create worktrees for parallel work
git worktree add ../contextd-task17 feature/tokenizer-interface
git worktree add ../contextd-task22 feature/tokenizer-vocab

# Terminal 1:
cd ../contextd-task17
claude  # Work on Task 17

# Terminal 2:
cd ../contextd-task22
claude  # Work on Task 22 in parallel

# Merge when ready
```

---

## Task Dependencies Matrix

| Task | Depends On | Can Start After | Parallel With |
|------|------------|----------------|---------------|
| 17   | None       | Immediately    | 22            |
| 18   | 17         | Phase 1        | 20, 23        |
| 19   | 17         | Phase 1        | 22            |
| 20   | 19         | Phase 2        | 18, 23        |
| 21   | 20         | Phase 3        | None (serial) |
| 22   | None       | Immediately    | 17, 19        |
| 23   | 22         | Phase 2        | 18, 20        |
| 24   | 19,20,22   | Phase 4        | 26            |
| 25   | 24         | Phase 5        | 27            |
| 26   | 19,22      | Phase 4        | 24            |
| 27   | 26         | Phase 5        | 25            |
| 28   | All        | Phase 6        | None (final)  |

---

## Critical Path Analysis

**Longest path determines minimum time**:
```
Task 17 (1h) → Task 19 (1h) → Task 20 (4h) → Task 21 (3h) → Task 24 (3h) → Task 25 (2h) → Task 28 (2h)
Total: 16 hours (critical path)
```

**Parallel paths can complete faster**:
- Task 22 → 23 (4 hours) - completes early
- Task 26 → 27 (5 hours) - runs parallel to OpenAI tasks

---

## Verification Commands

```bash
# Check next available task
task-master next

# View specific task
task-master show <id>

# Check dependencies
task-master validate-dependencies

# List pending tasks
task-master list --status=pending

# Show progress
task-master list | grep "BPE Task"
```

---

## Tips for Parallel Execution

1. **Use Git Worktrees**: Isolate work in separate directories
2. **Communication**: Coordinate which tasks each agent/developer takes
3. **Merge Frequently**: Integrate completed tasks regularly
4. **Test Integration**: After each phase, test combined code
5. **Phase Gates**: Don't start Phase N+1 until Phase N completes
6. **Document As You Go**: Update CLAUDE.md incrementally

---

## Success Criteria

- ✅ All 12 tasks completed
- ✅ All tests passing (>80% coverage)
- ✅ Token counting <1ms for typical texts
- ✅ Token counts match APIs (±1 token)
- ✅ Complete documentation
- ✅ Integration with embedding service works

---

**Ready to start? Begin with Tasks 17 and 22 in parallel!**

```bash
# Start now:
task-master set-status --id=17 --status=in-progress
task-master set-status --id=22 --status=in-progress
```
