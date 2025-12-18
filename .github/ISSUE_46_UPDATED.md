# Updated Issue #46: Conversation Indexing & Policies

**Updated Title:** `feat: Conversation Indexing & Policies`

---

## Summary

Index Claude Code conversation history from `~/.claude/projects/` to pre-warm contextd with:

- **Remediations** - Error → fix patterns extracted from past sessions
- **Violations** - Skill bypasses and bad patterns detected
- **Decisions** - Architectural choices with rationale
- **Policies** - Searchable compliance rules for self-reflection

## Implementation Philosophy

**Skills/Commands First → Backend Later**

1. **Prove it works** with skills/commands using existing MCP tools
2. **Validate UX** through real usage
3. **Implement in Go** once the design is battle-tested

**Benefits:**

- Faster iteration (no recompile)
- User feedback before heavy investment
- Skills define the contract that backend will implement
- Avoids over-engineering before understanding real needs

## Conversation JSONL Format

```
~/.claude/projects/{project-path-encoded}/
├── {uuid}.jsonl        # Session conversations
└── agent-{id}.jsonl    # Agent sub-conversations
```

**Message Types**: `user`, `assistant`, `file-history-snapshot`, `summary`, `system`

## Architecture

The proposed system includes three main components:

1. **ONBOARD** - Processes conversation files through scrubbing, extraction, and storage
2. **REFLECTS** - Periodic evaluation of compliance against policies
3. **SKILL LOAD** - Runtime injection of policy context

## Policy Schema

```go
type Policy struct {
    ID          string    `json:"id"`
    SkillName   string    `json:"skill_name"`
    Statement   string    `json:"statement"`      // renamed from "enforcement"
    Summary     string    `json:"summary"`
    Source      string    `json:"source"`
    Category    string    `json:"category"`       // verification|process|security
    Violations  int       `json:"violations"`
    Successes   int       `json:"successes"`
    Occurrences int       `json:"occurrences"`
    LastChecked time.Time `json:"last_checked"`
    CreatedAt   time.Time `json:"created_at"`
}
```

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Deduplication | Merge + boost confidence | Single policy with higher confidence; track both violations and successes |
| Index tracking | SHA256 file hash | Most reliable method; re-indexes on any change |
| Reflect output | Summary + actionable + full report + vectordb | Display summary, save timestamped markdown, store searchable |
| Extraction quality | 0.6 confidence threshold | Low-confidence extractions marked for optional review |

## Implementation Tasks

### PHASE A: Skills/Commands (Prove It Works)

- [ ] **A1**: Create `.claude-plugin/skills/conversation-indexing/SKILL.md`
- [ ] **A2**: Update `.claude-plugin/commands/onboard.md`
- [ ] **A3**: Policy Storage (Using Memories)
- [ ] **A4**: Update `.claude-plugin/commands/reflect.md`
- [ ] **A5**: Reflection Reports
- [ ] **A6**: Skill Load Policy Injection

### PHASE B: Backend Implementation (Once Validated)

- [ ] **B1**: `internal/conversation/` package
- [ ] **B2**: `internal/extraction/` package
- [ ] **B3**: `internal/policy/` package (renamed from enforcement)
- [ ] **B4**: `internal/reflection/` package
- [ ] **B5**: CLI integration

## User Warnings

**Online Mode** warning includes estimated token usage (~50k tokens per conversation) with options for context folding, batch mode, or cancellation.

## Security Considerations

1. **Secret Scrubbing** - All content scrubbed before indexing using gitleaks SDK
2. **Path Validation** - Prevent path traversal vulnerabilities
3. **Consent** - Explicit user opt-in required
4. **Scope** - Index only project-specific conversations

## Files to Create/Modify

**Phase A**: Five files (mostly modifications, one new skill file)
**Phase B**: Four new internal packages plus CLI modifications

## Related Issues

- Depends on: Behavioral Taxonomy
- Related to: Unified Payload Filtering (#40)

---

## Terminology Changes Summary

| Old Term | New Term |
|----------|----------|
| Skill Enforcements | Policies |
| SkillEnforcement (struct) | Policy |
| enforcement (field) | statement |
| internal/enforcement/ | internal/policy/ |
| Enforcement Storage | Policy Storage |
| Enforcement Injection | Policy Injection |
