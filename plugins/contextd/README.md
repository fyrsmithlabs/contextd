# contextd plugin for Claude Code

Cross-session memory, context checkpoints, semantic code search, and error
remediation for Claude Code — backed by the [contextd](https://github.com/fyrsmithlabs/contextd)
MCP server.

## What you get

**Skills** (auto-activate based on context):

| Skill | Activates when |
|-------|----------------|
| `using-contextd` | Starting a session / non-trivial task |
| `cross-session-memory` | Searching for or recording reusable learnings |
| `checkpoint-workflow` | Preserving or resuming session state |
| `error-remediation` | An error, failed build, or failing test appears |

**Commands**:

| Command | Purpose |
|---------|---------|
| `/contextd:checkpoint` | Save a resumable context checkpoint |
| `/contextd:remember` | Record a learning into memory |
| `/contextd:diagnose` | Diagnose an error and find known fixes |
| `/contextd:resume` | List and resume from a checkpoint |
| `/contextd:status` | Show memories, checkpoints, and project context |
| `/contextd:search` | Search memories, remediations, and code |

**MCP server**: bundled `.mcp.json` launches `contextd --mcp`, exposing the full
tool set (memory, checkpoint, remediation, semantic search, context-folding,
conversation indexing, reflection).

**Hook**: a defensive `SessionStart` hook that reminds Claude to use contextd
when the binary is present, and no-ops silently when it isn't.

## Prerequisites

The `contextd` binary must be on `PATH` for the MCP server to start:

```bash
# Homebrew
brew tap fyrsmithlabs/tap && brew install contextd

# or build from source
go build -o contextd ./cmd/contextd
```

## Install

This repository is itself a plugin marketplace named `contextd`.

```
/plugin marketplace add fyrsmithlabs/contextd
/plugin install contextd@contextd
```

For local development against a checkout, point the marketplace at the repo root
(which contains `.claude-plugin/marketplace.json`).

## Configuration

The bundled MCP server defaults to the embedded `chromem` vector store. Override
via environment variables in `.mcp.json` or your contextd config — for example
`CONTEXTD_VECTORSTORE_PROVIDER=qdrant`. See the
[contextd docs](https://github.com/fyrsmithlabs/contextd) for the full list.
