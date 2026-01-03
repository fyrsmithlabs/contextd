# Ralph Wiggum Loop Prompt: Issue #53

## Task: Implement Conversation Indexing and Self-Reflection Features

Implement two major features for contextd with complete test coverage, linting compliance, and verified through consensus reviews.

---

## Context

**Issue:** #53 - Implement conversation indexing and self-reflection features

**Specifications:**
- Conversation Indexing: `docs/spec/conversation-indexing/SPEC.md`, `DESIGN.md`, `SCHEMA.md`, `CONFIG.md`
- Self-Reflection: `docs/plans/2025-12-12-self-reflection-design.md`

**This is a large feature.** Work incrementally through phases, running tests and linting after each significant change.

---

## Implementation Phases

### Phase 1: Conversation Indexing Core

**Package: `internal/conversation/`**

| File | Purpose |
|------|---------|
| `types.go` | RawMessage, ConversationDocument, MessageDocument, DecisionDocument, FileReference, CommitReference, IndexOptions, SearchOptions |
| `parser.go` | Parse Claude Code JSONL files from `~/.claude/projects/{project}/*.jsonl` |
| `extractor.go` | Extract messages, file references, commit SHAs from tool calls |
| `service.go` | ConversationService with Index() and Search() methods |

**Requirements:**
- Parse JSONL line-by-line, extract user/assistant messages and tool calls
- Integrate with existing `internal/secrets` scrubber for all content
- Use existing `internal/vectorstore` for storage
- Collection naming: `{tenant}_{project}_conversations`

### Phase 2: Decision Extraction

**Package: `internal/extraction/`**

| File | Purpose |
|------|---------|
| `types.go` | DecisionCandidate, Decision, Pattern, TagRule |
| `heuristic.go` | Keyword-based decision detection with confidence scoring |
| `tags.go` | Extract tags from content (languages, infrastructure, activities) |
| `provider.go` | Summarizer interface and factory |
| `llm.go` | LLM-based summarization via langchain-go (optional, can be stub initially) |

**Default Decision Patterns:** (from DESIGN.md)
```go
{Name: "lets_use",      Regex: `(?i)let's (go with|use|choose|pick)`, Weight: 0.9}
{Name: "decided_to",    Regex: `(?i)decided to`,                      Weight: 0.9}
{Name: "remember_this", Regex: `(?i)remember (this|that)`,            Weight: 1.0}
// ... see DESIGN.md for full list
```

**Confidence Thresholds:**
- >= 0.8: Index as decision directly
- 0.5 - 0.8: Index as candidate
- < 0.5: Skip

### Phase 3: MCP Tools

**File: `internal/mcp/tools_conversation.go`**

| Tool | Purpose |
|------|---------|
| `conversation_index` | Index conversations for a project |
| `conversation_search` | Semantic search over indexed conversations |

**Input/Output schemas must be added to:**
- `.claude-plugin/schemas/contextd-mcp-tools.schema.json`

### Phase 4: Self-Reflection Foundation

**Package: `internal/reflection/`**

| File | Purpose |
|------|---------|
| `types.go` | Finding, BehaviorType, Severity, ReflectionReport |
| `analyzer.go` | Mine ReasoningBank for behavioral patterns |
| `correlator.go` | Correlate behaviors to violated CLAUDE.md instructions |
| `report.go` | Generate findings reports |

**Behavioral Taxonomy:**
| Behavior | Description |
|----------|-------------|
| `rationalized-skip` | Agent justified skipping required step |
| `overclaimed` | Absolute language inappropriately |
| `ignored-instruction` | Didn't follow CLAUDE.md directive |
| `assumed-context` | Assumed without verification |
| `undocumented-decision` | Significant choice without rationale |

### Phase 5: Reflection Integration

**Files:**
- `internal/reflection/remediation.go` - Generate and pressure-test doc fixes
- `.claude-plugin/commands/reflect.md` - `/contextd:reflect` command

---

## Success Criteria Checklist

### Code Quality
- [ ] All new code in `internal/conversation/`, `internal/extraction/`, `internal/reflection/`
- [ ] Unit tests for all packages (target 80%+ coverage)
- [ ] Integration tests for MCP tools
- [ ] `go test ./...` passes
- [ ] `golangci-lint run --timeout=5m ./...` passes with zero errors

### Functional Requirements (from SPEC.md)
- [ ] FR-001: JSONL parsing extracts user/assistant messages and tool calls
- [ ] FR-002: All content scrubbed through gitleaks before storage
- [ ] FR-003: Separate `_conversations` collection per project
- [ ] FR-004: Semantic search with type/tag/file filters
- [ ] FR-005: Tag extraction (language, domain, activity)
- [ ] FR-006: File cross-references from tool calls
- [ ] FR-007: Commit cross-references from git operations
- [ ] FR-008: Heuristic decision detection with confidence scoring
- [ ] FR-011: MCP tools `conversation_index` and `conversation_search`

### Self-Reflection Requirements
- [ ] Behavioral pattern detection (5 types)
- [ ] Severity overlay (CRITICAL/HIGH/MEDIUM/LOW)
- [ ] Report generation with evidence
- [ ] `/contextd:reflect` command skeleton

### CI/CD Requirements
- [ ] Create feature branch `feature/issue-53-conversation-reflection`
- [ ] All commits pass pre-commit hooks
- [ ] PR created with proper description
- [ ] GitHub Actions CI passes
- [ ] claude-code-review action passes

### Consensus Review Requirements
- [ ] Consensus Review #1 passes (Security, Correctness, Architecture, UX)
- [ ] Remediate any findings from Review #1
- [ ] Consensus Review #2 passes
- [ ] Remediate any findings from Review #2
- [ ] Consensus Review #3 passes (final approval)

---

## TDD Workflow

For each component:
1. **RED**: Write failing tests first
2. **GREEN**: Implement minimum code to pass
3. **REFACTOR**: Clean up while keeping tests green
4. **LINT**: Run `golangci-lint run ./...` and fix issues
5. **COMMIT**: Atomic commits for each component

---

## File Structure to Create

```
internal/
├── conversation/
│   ├── parser.go
│   ├── parser_test.go
│   ├── extractor.go
│   ├── extractor_test.go
│   ├── service.go
│   ├── service_test.go
│   └── types.go
├── extraction/
│   ├── heuristic.go
│   ├── heuristic_test.go
│   ├── tags.go
│   ├── tags_test.go
│   ├── provider.go
│   ├── llm.go (stub)
│   └── types.go
├── reflection/
│   ├── analyzer.go
│   ├── analyzer_test.go
│   ├── correlator.go
│   ├── correlator_test.go
│   ├── report.go
│   ├── report_test.go
│   └── types.go
└── mcp/
    ├── tools_conversation.go
    └── tools_conversation_test.go

.claude-plugin/
├── commands/
│   └── reflect.md
└── schemas/
    └── contextd-mcp-tools.schema.json (update)
```

---

## Commands to Run

```bash
# After each component
go test ./internal/conversation/... -v
go test ./internal/extraction/... -v
go test ./internal/reflection/... -v
golangci-lint run --timeout=5m ./...

# Before PR
go test ./... -cover
golangci-lint run --timeout=5m ./...

# Create PR
git checkout -b feature/issue-53-conversation-reflection
git add -A
git commit -m "feat: implement conversation indexing and self-reflection (#53)"
git push -u origin feature/issue-53-conversation-reflection
gh pr create --title "feat: implement conversation indexing and self-reflection (#53)" --body "..."

# Run consensus review
/contextd:consensus-review "feature/issue-53-conversation-reflection changes"
```

---

## Iteration Guidelines

Each iteration should:

1. **Check current state** - What's implemented? What's failing?
2. **Run tests** - `go test ./... -short 2>&1 | tail -30`
3. **Run linter** - `golangci-lint run --timeout=5m ./... 2>&1`
4. **Fix issues** - Address test failures and lint errors
5. **Progress incrementally** - Complete one package at a time
6. **Commit frequently** - Small atomic commits

### Phase Progression

```
Phase 1 → Tests pass → Lint passes → Commit
Phase 2 → Tests pass → Lint passes → Commit
Phase 3 → Tests pass → Lint passes → Commit
Phase 4 → Tests pass → Lint passes → Commit
Phase 5 → Tests pass → Lint passes → Commit
Full test suite → PR created → CI passes
Consensus Review #1 → Remediate → Review #2 → Remediate → Review #3
```

---

## Completion Promise

When ALL of the following are TRUE, output:

```
<promise>ISSUE-53-COMPLETE</promise>
```

**Required conditions:**
1. All packages implemented (`conversation`, `extraction`, `reflection`)
2. All MCP tools registered (`conversation_index`, `conversation_search`)
3. `go test ./...` passes
4. `golangci-lint run --timeout=5m ./...` passes with zero errors
5. PR created and CI (GitHub Actions) passes
6. claude-code-review action passes
7. Three (3) consensus reviews pass with no CRITICAL/HIGH findings

---

## DO NOT emit the promise until:

- All tests pass
- All lint checks pass
- PR is created and CI passes
- 3 consensus reviews pass

**STRICT REQUIREMENTS:**
- Use `<promise>` XML tags EXACTLY as shown
- The statement MUST be completely TRUE
- Do NOT output false statements to exit the loop
- Do NOT lie even if stuck or running long

═══════════════════════════════════════════════════════════
CRITICAL - Ralph Loop Completion Promise
═══════════════════════════════════════════════════════════

To complete this loop, output this EXACT text:
  <promise>ISSUE-53-COMPLETE</promise>

This loop has max-iterations=30. The promise can ONLY be
output when ALL success criteria are genuinely met.
═══════════════════════════════════════════════════════════
