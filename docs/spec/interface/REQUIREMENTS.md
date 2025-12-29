# MCP Interface Requirements

**Version**: 1.0.0
**Status**: Draft
**Date**: 2025-11-23

## Overview

Requirements for contextd MCP interface - a tool proxy system enabling AI agents to use secure, context-efficient tools with ReasoningBank and Context Folding support.

---

## Functional Requirements

### FR-1: Tool Proxy Architecture

- FR-1.1: Single `contextd-proxy` binary serves all tools via symlink pattern
- FR-1.2: Proxy determines tool from `argv[0]` (symlink name)
- FR-1.3: Proxy communicates with contextd server via gRPC
- FR-1.4: Server executes tools via hashicorp/go-plugin (subprocess isolation)

### FR-2: Tool Discovery

- FR-2.1: Tools installed to `.claude/tools/<tool_name>/` directory
- FR-2.2: Each tool directory contains: `schema.json`, `TOOL.md`, symlink to proxy
- FR-2.3: `schema.json` follows JSON Schema draft 2020-12
- FR-2.4: `TOOL.md` is minimal (~100 tokens max)
- FR-2.5: Extended docs in `TOOL_EXTENDED.md` (loaded on-demand only)

### FR-3: Tool Interface

- FR-3.1: All tools implement `Execute(ctx, input) → Response` interface
- FR-3.2: Every response includes `tokens_used` field
- FR-3.3: Large responses include `has_more` and continuation mechanism
- FR-3.4: Truncated responses include `truncated: true`
- FR-3.5: Reference-based responses include `refs` array for deferred loading

### FR-4: Core Tools - Safe Execution

- FR-4.1: `safe_bash` - Execute shell commands with scrubbing
- FR-4.2: `safe_read` - Read files with path restrictions and scrubbing
- FR-4.3: `safe_write` - Write files with path restrictions and audit

### FR-5: Core Tools - ReasoningBank

- FR-5.1: `memory_search` - Search memories, return references by default
- FR-5.2: `memory_get` - Get full memory content by ID
- FR-5.3: `memory_store` - Store new reasoning pattern
- FR-5.4: `memory_feedback` - Provide outcome feedback on memory

### FR-6: Core Tools - Context Folding

- FR-6.1: `branch` - Create isolated subtask context
- FR-6.2: `return` - Complete branch, return summary to parent
- FR-6.3: Branch context isolated from parent (no pollution)
- FR-6.4: Return supports `extract_memory: true` for async distillation

### FR-7: Core Tools - Checkpoints

- FR-7.1: `checkpoint_save` - Save session state, return ID
- FR-7.2: `checkpoint_restore` - Restore with levels: summary/context/full
- FR-7.3: Default restore level is `context` (not full)
- FR-7.4: `checkpoint_list` - List available checkpoints with summaries

### FR-8: Tool Host Capabilities

- FR-8.1: Tools receive `ToolHost` interface for host services
- FR-8.2: ToolHost provides: Scrub, Log, MemorySearch, MemoryStore
- FR-8.3: ToolHost provides: CheckpointSave, CheckpointRestore
- FR-8.4: Tools cannot access host resources without ToolHost

### FR-9: CLI (ctxd)

- FR-9.1: `ctxd server start/stop` - Manage contextd server
- FR-9.2: `ctxd tool install <source>` - Install tool via go-getter
- FR-9.3: `ctxd tool list/remove/update` - Manage installed tools
- FR-9.4: `ctxd tool run <tool> --input '{}'` - Test tool directly
- FR-9.5: `ctxd doctor` - Health check (server, qdrant, tools)
- FR-9.6: `ctxd config set/get` - Configuration management

### FR-10: Tool Installation

- FR-10.1: Support git sources: `github.com/org/tool@version`
- FR-10.2: Support HTTP sources: `https://example.com/tool.tar.gz`
- FR-10.3: Support S3 sources: `s3::bucket/tool.zip`
- FR-10.4: Verify tool binary hash on install
- FR-10.5: Create symlink to contextd-proxy on install

---

## Context Efficiency Requirements

### CE-1: Token Budgets

- CE-1.1: Tool discovery (all schemas) < 1,500 tokens
- CE-1.2: Individual TOOL.md < 100 tokens
- CE-1.3: Search responses default < 500 tokens
- CE-1.4: Any single tool response < 2,000 tokens (unless explicit)
- CE-1.5: Checkpoint restore (context level) < 3,000 tokens

### CE-2: Response Tiering

- CE-2.1: `memory_search` defaults to references (IDs + titles only)
- CE-2.2: Full content requires explicit `memory_get` call
- CE-2.3: `checkpoint_restore` defaults to `context` level
- CE-2.4: Large bash/read output truncated with continuation token

### CE-3: Lazy Loading

- CE-3.1: TOOL.md loaded on first tool use, not at discovery
- CE-3.2: TOOL_EXTENDED.md never auto-loaded
- CE-3.3: Memory content loaded only via explicit `memory_get`

### CE-4: Session Efficiency

- CE-4.1: `session_init` provides warm start (~800 tokens)
- CE-4.2: Warm start includes: recent checkpoints, relevant memories, blockers
- CE-4.3: Branch isolation prevents context pollution to parent

---

## Security Requirements

### SEC-1: Secret Scrubbing

- SEC-1.1: All tool output scrubbed via gitleaks before return
- SEC-1.2: Dual scrubbing: tool-level + server-level
- SEC-1.3: Secrets tokenized (e.g., `[SECRET_1]`), not redacted
- SEC-1.4: Token registry stores mappings for tool-to-tool flow

### SEC-2: Process Isolation

- SEC-2.1: Tools execute in separate process (go-plugin)
- SEC-2.2: Seccomp profiles restrict syscalls per tool
- SEC-2.3: Linux namespaces isolate: pid, net, mount, user
- SEC-2.4: Network namespace blocks all network by default
- SEC-2.5: Resource limits via cgroups (memory, CPU)

### SEC-3: Communication Security

- SEC-3.1: mTLS between proxy and server
- SEC-3.2: gRPC with certificate validation
- SEC-3.3: Unix socket with restrictive permissions (0600)

### SEC-4: Binary Integrity

- SEC-4.1: Tool binaries verified by hash on load
- SEC-4.2: Symlink targets verified at startup
- SEC-4.3: Proxy verifies it's running from expected location

### SEC-5: Audit Logging

- SEC-5.1: All tool invocations logged
- SEC-5.2: Log includes: tool, input (scrubbed), output hash, duration
- SEC-5.3: Log includes: session ID, timestamp, secrets found
- SEC-5.4: Audit log is append-only

---

## Non-Functional Requirements

### NFR-1: Performance

- NFR-1.1: Tool invocation latency < 50ms (excluding execution)
- NFR-1.2: Proxy → server round-trip < 10ms
- NFR-1.3: Scrubbing adds < 20ms for typical output
- NFR-1.4: Support concurrent tool executions

### NFR-2: Reliability

- NFR-2.1: Graceful degradation if Qdrant unavailable
- NFR-2.2: Tool timeout configurable (default 30s, max 300s)
- NFR-2.3: Server restart doesn't lose in-flight requests

### NFR-3: Compatibility

- NFR-3.1: Works with Claude Code (primary)
- NFR-3.2: Works with OpenCode
- NFR-3.3: Works with any agent reading .claude/tools/
- NFR-3.4: Linux required for seccomp/namespaces
- NFR-3.5: macOS supported with reduced isolation

### NFR-4: Extensibility

- NFR-4.1: Third-party tools can implement Tool interface
- NFR-4.2: Custom tools installable via ctxd
- NFR-4.3: Tool SDK published as Go package

---

## Interface Requirements

### API-1: gRPC (Tool-facing)

```protobuf
service ToolService {
    rpc Execute(ExecuteRequest) returns (ExecuteResponse);
    rpc GetSchema(SchemaRequest) returns (SchemaResponse);
}
```

### API-2: gRPC (CLI-facing)

```protobuf
service ManagementService {
    rpc InstallTool(InstallRequest) returns (InstallResponse);
    rpc ListTools(ListRequest) returns (ListResponse);
    rpc RemoveTool(RemoveRequest) returns (RemoveResponse);
    rpc ServerStatus(StatusRequest) returns (StatusResponse);
}
```

### API-3: HTTP (Automation-facing)

```
POST /v1/tools/{tool}/execute
GET  /v1/tools
GET  /v1/health
POST /v1/memory/search
POST /v1/checkpoint/save
```

---

## Data Requirements

### DATA-1: Tool Definition

```
.claude/tools/<name>/
├── schema.json      # JSON Schema 2020-12
├── TOOL.md         # Minimal docs (<100 tokens)
├── TOOL_EXTENDED.md # Extended docs (optional)
└── <name>          # Symlink → contextd-proxy
```

### DATA-2: Response Format

```json
{
  "data": {},
  "refs": ["id1", "id2"],
  "tokens_used": 150,
  "has_more": true,
  "truncated": false
}
```

### DATA-3: TOOL.md Format

```markdown
# {name}
> {one-line purpose}

**Use**: {trigger conditions}
**Not**: {operation} → `{preferred_tool}`
**Limits**: {constraints}
**Ref**: `TOOL_EXTENDED.md`
```

---

## Acceptance Criteria

- [ ] contextd-proxy binary works via symlink pattern
- [ ] All core tools implemented and functional
- [ ] Token budgets enforced on all responses
- [ ] Dual gitleaks scrubbing verified (no secret leakage)
- [ ] Process isolation verified (seccomp blocks syscalls)
- [ ] mTLS working between proxy and server
- [ ] ReasoningBank tools return references by default
- [ ] Context folding branch/return working
- [ ] Checkpoint restore levels working
- [ ] ctxd CLI functional
- [ ] Audit log captures all operations
- [ ] Test coverage ≥ 80%
