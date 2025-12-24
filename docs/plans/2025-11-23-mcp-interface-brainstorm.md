# MCP Interface Brainstorm Session

**Date**: 2025-11-23
**Status**: In Progress

## Summary

Brainstorming session for contextd MCP interface architecture, focusing on:
- Secret scrubbing with gitleaks (dual-layer: tool + server)
- MCP server (stdio + Streamable HTTP)
- Process isolation (seccomp + namespaces) for tool execution
- Capability-based security model (deny-by-default)
- gRPC API for ctxd CLI and integrations
- Audit logging for all operations

**Key Decision**: Native Go with process isolation (Phase 1). WASM sandboxing backburnered to Phase 2 if stronger isolation needed.

## Key Decisions Made

### 1. Transport Architecture
- **Client**: stdio (for Claude Code compatibility)
- **Server**: Streamable HTTP (MCP 2025-03-26 spec)
- **Communication**: NATS lattice (wasmCloud pattern) for distributed mode

### 2. Secret Scrubbing Strategy
- **Primary**: WASM tools with mandatory scrubber capability
- **Backup**: PreToolUse hooks to block + scrub if raw tools used
- **Pattern**: Tokenization (secrets → [SECRET_N]) not redaction
- **SDK**: gitleaks detect package (or Rust equivalent)

### 3. Tool Execution Model
- **NOT arbitrary code execution** - security risk
- **WASM components** as sandboxed tool implementations
- **Capability providers** grant access to host resources
- **Deny-by-default** - tools can only access granted capabilities

### 4. WASM Runtime Candidates
| Runtime | Notes |
|---------|-------|
| WasmEdge | Fastest, WASI++, JS support |
| Wasmtime | Pure Rust, most mature |
| Wasmer | Rust-native, multiple backends |

### 5. Language Choice
**Go** selected for:
- User familiarity (can take over codebase)
- Good WASM runtime support (wazero, wasmtime-go)
- Excellent tooling
- Memory safety via GC

### 6. Claude Code Enforcement Reality (CRITICAL RESEARCH)

**Source Code Research Results** (2025-11-23):

| Mechanism | Status | Issue |
|-----------|--------|-------|
| `permissions.deny` (Read/Write) | **BROKEN** | #6631 |
| PreToolUse `permissionDecision: deny` | **BROKEN** | #4669 (open) |
| PreToolUse `preventContinuation: true` | **BROKEN** | #3514 |
| `--disallowedTools` flag | **BROKEN** | #2625 |
| Exit code 2 from hooks | Documented | Uses stderr |
| OS-level shell wrapper | **WORKS** | #2695 solution |

**Key Insight**: Claude Code's permission system is fundamentally unreliable.
The only trustworthy enforcement is at the **execution layer** (OS/WASM).

**This validates our architecture**: contextd server becomes the single enforcement point.

### 7. Enforcement Strategy (Updated)

Given broken Claude Code enforcement, our strategy is:

**Layer 1: Best-Effort Agent Configuration** (advisory only)
- `.claude/skills/contextd/SKILL.md` with `allowed-tools` whitelist
- `permissions.deny` for built-in tools (may not work)
- `AGENTS.md` instructions (can be ignored)

**Layer 2: Exit Code 2 Hooks** (most reliable Claude Code mechanism)
```bash
# PreToolUse hook - exit 2 to block
if [[ "$TOOL_NAME" =~ ^(Bash|Read|Write|Edit)$ ]]; then
  echo "Use contextd tools instead" >&2
  exit 2
fi
```

**Layer 3: WASM Sandbox** (TRUE enforcement - our architecture)
- Agent can ONLY access contextd MCP tools
- contextd tools execute in WASM sandbox
- Capabilities enforced at runtime level
- Secret scrubbing mandatory before output

**Defense-in-Depth**: Layers 1-2 are advisory. Layer 3 is the real security boundary.

## Architecture Overview (Finalized)

### Native Go with Process Isolation

```
┌─────────────────────────────────────────────────────────────┐
│ contextd server (native Go)                                 │
│                                                             │
│  ├── MCP Server (stdio / Streamable HTTP)                   │
│  ├── gRPC API (for ctxd CLI, integrations)                  │
│  │                                                          │
│  ├── Core Services                                          │
│  │   ├── Tool Router (capability-based dispatch)            │
│  │   ├── Token Registry (secret tokenization)               │
│  │   ├── Audit Logger (all operations)                      │
│  │   └── Qdrant Client (vectors, state)                     │
│  │                                                          │
│  ├── Scrubbing Pipeline                                     │
│  │   ├── Tool-level scrub (gitleaks)                        │
│  │   └── Server-level scrub (gitleaks - catches all)        │
│  │                                                          │
│  └── Tool Executors (process-isolated)                      │
│      ├── safe_bash (seccomp + namespace)                    │
│      ├── safe_read (restricted paths)                       │
│      ├── safe_write (restricted paths)                      │
│      └── user_tools/* (sandboxed processes)                 │
│                                                             │
├── Process Isolation Layer                                   │
│   ├── seccomp profiles (syscall filtering)                  │
│   ├── Linux namespaces (pid, net, mount, user)              │
│   ├── cgroups (memory, CPU limits)                          │
│   └── Capability drops (no CAP_SYS_ADMIN, etc.)             │
│                                                             │
└── Protocol Enforcement                                      │
    ├── gRPC schema validation                                │
    ├── Capability-based method access                        │
    └── Mandatory scrubbing before response                   │
└─────────────────────────────────────────────────────────────┘
```

### Security Layers (Defense in Depth)

| Layer | Protects Against | Mechanism |
|-------|------------------|-----------|
| D1: Process Isolation | Code escape | seccomp + namespaces |
| D2: Capability Model | Unauthorized access | Deny-by-default config |
| D3: gRPC Schema | Protocol abuse | Type validation |
| D4: Dual Scrubbing | Secret leakage | gitleaks (tool + server) |
| D5: Resource Limits | DoS/abuse | cgroups |

### External Interfaces

```
Agents (Claude, OpenCode)
    │
    │ MCP (stdio or Streamable HTTP)
    ▼
contextd server
    │
    │ gRPC API
    ▼
ctxd CLI
├── ctxd tool install <name>
├── ctxd config set <key> <val>
├── ctxd server start/stop
├── ctxd index <path>
└── ctxd query "search"
```

### Phase 2: wasmCloud for SaaS (Designed For)

Local architecture mirrors future SaaS deployment:

| Component | Phase 1 (Local) | Phase 2 (SaaS) |
|-----------|-----------------|----------------|
| Isolation | seccomp/namespace | WASM sandbox |
| Runtime | Native Go | wasmCloud + wazero |
| Providers | Go interfaces | wasmCloud capability providers |
| Tools | Go binaries | WASM components |
| Messaging | gRPC (local) | NATS lattice (distributed) |
| Multi-tenant | Single user | Full isolation per tenant |

**Design Principle**: Capability interfaces designed to be compatible with both models.

```go
// Same interface, different implementations
type ShellProvider interface {
    Exec(ctx context.Context, cmd string, caps Capabilities) (Output, error)
}

// Phase 1: Native process with seccomp
type SeccompShellProvider struct { ... }

// Phase 2: wasmCloud capability provider
type WasmCloudShellProvider struct { ... }
```

**Migration Path**:
1. Tools implement capability interfaces (not direct syscalls)
2. Providers are swappable (seccomp → WASM)
3. External APIs (MCP, gRPC) stay identical
4. Business logic unchanged, only isolation layer swaps

## Capability-Based Security Model

```toml
# Example: safe_bash.toml
[component]
name = "safe_bash"
wasm = "safe_bash.wasm"

[capabilities]
required = ["shell:exec", "scrubber:scan"]
optional = ["fs:read"]

[limits]
max_execution_time = "30s"
max_memory = "64MB"
max_output_size = "1MB"
```

## Data Flow with Scrubbing

1. Claude calls MCP tool: `safe_bash({ command: "cat .env" })`
2. contextd routes to `safe_bash.wasm` component
3. Component calls `shell_exec()` via capability provider
4. Raw output captured in WASM sandbox
5. Component calls `scrubber_scan()` - MANDATORY capability
6. Scrubber tokenizes secrets: `API_KEY=sk-xxx` → `API_KEY=[SECRET_1]`
7. Scrubbed output returned to Claude
8. Token registry stores mapping for tool-to-tool flow

## Research References

- [Anthropic Code Execution with MCP](https://www.anthropic.com/engineering/code-execution-with-mcp)
- [MCP Spec 2025-03-26](https://modelcontextprotocol.io/specification/2025-03-26)
- [WasmEdge](https://github.com/WasmEdge/WasmEdge)
- [wasmCloud](https://github.com/wasmCloud/wasmCloud)
- [gitleaks](https://github.com/gitleaks/gitleaks)
- [Claude Code Hooks](https://code.claude.com/docs/en/hooks)

## Multi-Layer Defense Architecture

### Layer A: Agent-Level Enforcement (Best-Effort, Advisory)

```
┌─────────────────────────────────────────────────────────────┐
│ Agent Configuration (may be bypassed/broken)                │
│  ├── .claude/skills/contextd/SKILL.md (allowed-tools)       │
│  ├── permissions.deny (BROKEN per #6631)                    │
│  ├── AGENTS.md instructions (can be ignored)                │
│  └── Exit code 2 hooks (most reliable CC mechanism)         │
└─────────────────────────────────────────────────────────────┘
```

### Layer B: WASM Sandbox Enforcement (TRUE Security Boundary)

```
┌─────────────────────────────────────────────────────────────┐
│ contextd WASM Runtime                                       │
│  ├── Capability-based security (deny-by-default)            │
│  ├── Tools can ONLY access granted capabilities             │
│  ├── No raw shell/filesystem access without contextd        │
│  └── Resource limits (memory, time, output size)            │
└─────────────────────────────────────────────────────────────┘
```

### Layer C: Multi-Layer Scrubbing (Defense-in-Depth)

```
Agent Request
    │
    ▼
contextd MCP Server
    │
    ├── Audit Log ← [LOGGED: tool, input, timestamp, session]
    │
    ▼
WASM Tool (safe_bash.wasm)
    │
    ├── Execute via capability provider
    ├── gitleaks scrub (Layer C1) ← Tool-level scrubbing
    │
    ▼
contextd Output Handler
    │
    ├── gitleaks scrub (Layer C2) ← Server-level scrubbing
    ├── Audit Log ← [LOGGED: output, secrets_found, scrub_applied]
    │
    ▼
Scrubbed Response to Agent
```

**Scrubbing Guarantees:**
- C1 (Tool-level): WASM tools use scrubber capability
- C2 (Server-level): contextd scrubs ALL output (catches tool mistakes)
- Audit log flags tools that leak secrets pre-C2 (security review)

### Layer D: Audit Trail

All operations logged with:
- Tool name, input parameters
- Raw output (stored securely, not in context)
- Scrubbed output (what agent sees)
- Secrets detected and tokenized
- Timestamp, session ID, user/org context

## User-Defined Tool Support

### Adding Custom Tools

```
.contextd/tools/
├── my_tool/
│   ├── TOOL.md           ← Human-readable docs (agent reads)
│   ├── schema.json       ← MCP input schema (agent reads)
│   ├── my_tool.wasm      ← WASM binary (contextd executes)
│   └── capabilities.toml ← Required/optional capabilities
```

### Capability Declaration

```toml
# capabilities.toml
[component]
name = "my_tool"

[capabilities]
required = ["scrubber:scan"]  # Mandatory for all tools
optional = ["http:fetch", "fs:read"]

[limits]
max_execution_time = "30s"
max_memory = "64MB"
```

### Tool Security Model

1. **Tools WITH scrubber capability**: Scrub their own output (Layer C1)
2. **Tools WITHOUT scrubber**: Server-level scrub catches secrets (Layer C2)
3. **Audit log**: Flags tools that leak → triggers security review
4. **Capability approval**: New capabilities require user approval

## Cross-Platform Compatibility

### Claude Code
```json
{
  "permissions": {
    "deny": ["Bash(*)", "Read(*)", "Write(*)", "Edit(*)"]
  },
  "hooks": {
    "PreToolUse": [{
      "matcher": "Bash|Read|Write|Edit",
      "hooks": [{"type": "command", "command": "contextd hook block-builtin"}]
    }]
  }
}
```

### OpenCode
```json
{
  "tools": { "bash": false, "read": false, "write": false },
  "mcpServers": {
    "contextd": { "type": "stdio", "command": "contextd", "args": ["mcp", "serve"] }
  }
}
```

### Auto-Configuration
```bash
contextd init --platform=claude-code  # Creates .claude/settings.json
contextd init --platform=opencode     # Creates opencode.json
contextd init --platform=all          # All configs + AGENTS.md
```

## Open Questions

1. ~~Which WASM runtime?~~ → Research Go-native (wazero, wasmtime-go)
2. NATS vs HTTP for client-server? → Start HTTP, add NATS later
3. How to handle tool-to-tool secret untokenization?
4. User-defined tool approval workflow?
5. Audit log retention and access policies?

## Resolved Questions

| Question | Answer |
|----------|--------|
| Trust Claude Code permissions? | NO - multiple critical bugs |
| Enforcement strategy? | Process isolation (seccomp/namespace) + scrubbing |
| Language? | Native Go (no TinyGo/WASM in Phase 1) |
| Single vs double scrub? | Double: tool-level + server-level |
| WASM? | Backburnered to Phase 2 if needed |
| Runtime? | wazero (if Phase 2 needed) |

## Next Steps

1. Design capability/tool configuration schema
2. Implement proof-of-concept:
   - contextd server (native Go)
   - MCP interface (stdio + HTTP)
   - gRPC API for ctxd
   - gitleaks integration (both scrubbing layers)
   - Audit logging
   - Process isolation (seccomp profiles)
3. Implement ctxd CLI (tool management, config)
4. Write formal specs:
   - `docs/spec/mcp-interface/`
   - `docs/spec/multi-tenancy/`
   - `docs/spec/consolidation/`
5. Test exit code 2 hooks as best-effort agent blocking
