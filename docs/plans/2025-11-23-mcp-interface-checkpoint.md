# MCP Interface Brainstorm Checkpoint

**Date**: 2025-11-23
**Status**: Ready for Spec Writing

## Final Proposal Summary

contextd is a tool proxy server that AI agents use instead of built-in tools (Bash, Read, Write), providing secret scrubbing, context efficiency, ReasoningBank, and context folding.

## Architecture

```
Agent (Claude Code, OpenCode)
    │
    │ Calls tool in .claude/tools/
    ▼
contextd-proxy (single binary, symlinked per tool)
    │
    │ gRPC
    ▼
contextd server (Go + Echo)
    │
    ├── hashicorp/go-plugin → spawns tool binary
    ├── Tool executes (seccomp + namespace isolated)
    ├── gitleaks scrub (tool + server level)
    ├── Truncate to token budget
    ├── Qdrant (memories, checkpoints)
    └── Return minimal response
```

## Core Tools

| Tool | Purpose | Returns |
|------|---------|---------|
| safe_bash | Execute commands | Scrubbed output (truncated) |
| safe_read | Read files | Scrubbed content (truncated) |
| safe_write | Write files | Confirmation |
| memory_search | Search ReasoningBank | References (IDs + titles) |
| memory_get | Get full memory | Full content |
| memory_store | Store reasoning pattern | Memory ID |
| checkpoint_save | Save session state | Checkpoint ID |
| checkpoint_restore | Restore session | Summary (not full state) |
| branch | Start isolated subtask | Branch ID |
| return | Complete subtask | Summary to parent context |

## Tool Interface

```go
type Tool interface {
    Execute(ctx context.Context, input json.RawMessage) (*Response, error)
}

type Response struct {
    Data       any      `json:"data"`
    Refs       []string `json:"refs,omitempty"`
    TokensUsed int      `json:"tokens_used"`
    HasMore    bool     `json:"has_more,omitempty"`
    Truncated  bool     `json:"truncated,omitempty"`
}

type ToolHost interface {
    Scrub(content string) (string, error)
    Log(entry AuditEntry) error
    MemorySearch(query string, limit int) ([]MemoryRef, error)
    MemoryStore(memory Memory) (string, error)
    CheckpointSave(state State) (string, error)
    CheckpointRestore(id string, level string) (Context, error)
}
```

## Tool Installation Structure

```
.claude/tools/safe_bash/
├── schema.json       # MCP schema (~50 tokens)
├── TOOL.md          # Minimal docs (~95 tokens)
└── safe_bash        # Symlink → contextd-proxy
```

## Context Efficiency Rules

1. Default to references (IDs, not content)
2. Token budgets per tool (max 500 search, 2000 per tool)
3. Lazy TOOL.md loading (on first use)
4. Truncate large output with continuation
5. Checkpoint restore levels: summary/context/full (default: context)
6. Branch isolation (subtask context doesn't pollute parent)

## Technology Stack

| Component | Technology |
|-----------|------------|
| Server | Go + Echo |
| Proxy | Go (single binary) |
| Plugins | hashicorp/go-plugin |
| Fetcher | hashicorp/go-getter |
| Scrubber | gitleaks |
| Storage | Qdrant |
| Isolation | seccomp + Linux namespaces |

## Security Layers

1. Process isolation (seccomp + namespaces)
2. mTLS (proxy ↔ server)
3. Dual gitleaks scrubbing (tool + server)
4. Audit logging

## Key Decisions Made

| Decision | Rationale |
|----------|-----------|
| Native Go (not WASM) | User maintainability, WASM backburnered to Phase 2 |
| go-plugin for tools | Battle-tested (Terraform), process isolation built-in |
| go-getter for install | Multi-source (git, http, s3), used by Terraform |
| Single proxy binary | Avoids N binary distribution, symlink pattern |
| gRPC (not HTTP) for tools | Fast, typed, streaming support |
| HTTP for automations | External integrations, webhooks |
| Default to refs | Context efficiency - 87% token reduction |
| Minimal TOOL.md | ~95 tokens vs ~800 tokens (88% reduction) |

## Research Findings

### Claude Code Enforcement (Broken)
- permissions.deny: BROKEN (#6631)
- PreToolUse hooks: BROKEN (#4669, #3514)
- --disallowedTools: BROKEN (#2625)
- Exit code 2: Works (most reliable)

### Context Efficiency Estimates
- Tool discovery: 18,000 → 1,200 tokens (93% savings)
- Search responses: 1,500 → 200 tokens (87% savings)
- Checkpoint resume: 50,000 → 2,000 tokens (96% savings)

## Open Items for Spec

1. Exact gRPC proto definitions
2. Seccomp profile specification
3. TOOL.md final format
4. schema.json standard
5. Error code catalog
6. Audit log schema
7. Qdrant collection schemas (reference existing)

## References

- [Full brainstorm](./2025-11-23-mcp-interface-brainstorm.md)
- [Context folding spec](../spec/context-folding/)
- [ReasoningBank spec](../spec/reasoning-bank/)
- [Collection architecture](../spec/collection-architecture/)
