# Contextd Statusline Design

**Date:** 2025-12-16
**Status:** Approved
**Issue:** #17 (Context Folding)

## Overview

Statusline integration that displays contextd state in Claude Code's status bar - memory counts, checkpoint counts, context health warnings, service status, and compression stats.

## Goals

1. **Awareness** - Passive display showing contextd is active with basic stats
2. **Monitoring** - Health indicators, warnings when context is high or services degraded

## Output Format

Compact with `│` separators, configurable items:

```
🟢 │ 🧠12 │ 💾3 │ 📊68% │ C:.85 │ F:2.1x
```

| Symbol | Meaning |
|--------|---------|
| `🟢/🟡/🔴` | Service health (healthy/degraded/down) |
| `🧠12` | 12 memories for this project |
| `💾3` | 3 checkpoints saved |
| `📊68%` | Context at 68% (yellow >70%, red >85%) |
| `C:.85` | Last memory confidence 0.85 |
| `F:2.1x` | Last compression ratio 2.1x |

## Data Flow

```
Claude Code updates conversation
        ↓
Calls: ctxd statusline run
        ↓
ctxd reads JSON from stdin (session data from Claude)
        ↓
ctxd calls GET http://localhost:9090/api/v1/status
        ↓
ctxd formats output with ANSI colors
        ↓
Claude Code displays first line of stdout
```

## HTTP Endpoint Enhancement

Enhanced `/api/v1/status` response:

```json
{
  "status": "healthy",
  "services": {"checkpoint": "ok", "memory": "ok", "compression": "ok"},
  "counts": {
    "checkpoints": 3,
    "memories": 12
  },
  "context": {
    "usage_percent": 68,
    "threshold_warning": false
  },
  "compression": {
    "last_ratio": 2.1,
    "last_quality": 0.92,
    "operations_total": 5
  },
  "memory": {
    "last_confidence": 0.85
  }
}
```

## CLI Commands

```bash
# Run mode - called by Claude Code
ctxd statusline run

# Installation helpers
ctxd statusline install    # Adds statusLine to .claude/settings.json
ctxd statusline uninstall  # Removes statusLine from settings.json
ctxd statusline test       # Runs once with mock data, shows output
```

**Install modifies `.claude/settings.json`:**
```json
{
  "statusLine": "ctxd statusline run"
}
```

## Slash Command

`/contextd:statusline` command:

```
/contextd:statusline           # Show current status + config
/contextd:statusline install   # Install statusline
/contextd:statusline uninstall # Remove statusline
/contextd:statusline test      # Preview output
```

## Configuration

In `~/.config/contextd/config.yaml`:

```yaml
statusline:
  enabled: true
  endpoint: "http://localhost:9090"  # For remote MCP support

  # Items to display (all enabled by default)
  show:
    service: true      # 🟢/🟡/🔴
    memories: true     # 🧠12
    checkpoints: true  # 💾3
    context: true      # 📊68%
    confidence: true   # C:.85
    compression: true  # F:2.1x

  # Thresholds for warnings
  thresholds:
    context_warning: 70   # Yellow at 70%
    context_critical: 85  # Red at 85%
```

## Plugin Integration

**Session-start hook auto-configures statusline:**

```bash
# Auto-configure statusline if not present
if ! grep -q '"statusLine"' ~/.claude/settings.json 2>/dev/null; then
  ctxd statusline install --quiet
fi
```

**Plugin file additions:**
- `.claude-plugin/commands/statusline.md` - Slash command
- `.claude-plugin/hooks/session-start.sh` - Enhanced with auto-install

## Implementation Components

| File | Change |
|------|--------|
| `cmd/ctxd/statusline.go` | New - statusline subcommands |
| `internal/http/server.go` | Enhanced `/api/v1/status` response |
| `internal/http/status.go` | New - status response builder |
| `internal/reasoningbank/service.go` | Add `Count()`, track last confidence |
| `internal/compression/service.go` | Expose last ratio/quality |
| `internal/config/statusline.go` | New - statusline config struct |

**Claude plugin:**
- `.claude-plugin/commands/statusline.md`
- `.claude-plugin/hooks/session-start.sh` (enhanced)

## Testing

- Unit tests for statusline output formatting
- Unit tests for config parsing
- Integration test for HTTP endpoint
- `ctxd statusline test` for manual verification
