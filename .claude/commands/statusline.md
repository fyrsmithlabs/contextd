---
description: Configure contextd statusline for Claude Code
---

# Statusline Configuration

Configure the contextd statusline integration for Claude Code.

## Available Actions

1. **Install** - Install the statusline script into Claude Code settings
2. **Uninstall** - Remove the statusline configuration
3. **Test** - Test the statusline output without installing
4. **Status** - Show current configuration status

## Quick Install

To install the statusline, run:

```bash
ctxd statusline install --server http://localhost:9090
```

## Test Output

To test the statusline output without installing:

```bash
ctxd statusline test --server http://localhost:9090
```

## What It Shows

The statusline displays contextd metrics in Claude Code's status bar:

| Icon | Meaning |
|------|---------|
| 🟢/🟡/🔴 | Service health (all ok / some unavailable / error) |
| 🧠N | Memory count |
| 💾N | Checkpoint count |
| 📊N% | Context usage percentage |
| C:0.XX | Last confidence score |
| F:X.Xx | Compression ratio |

Example output: `🟢 │ 🧠12 │ 💾3 │ 📊68% │ C:.85 │ F:2.1x`

## Configuration

The statusline configuration is stored in `~/.config/contextd/config.yaml`:

```yaml
statusline:
  enabled: true
  endpoint: http://localhost:9090
  show:
    service: true
    memories: true
    checkpoints: true
    context: true
    confidence: true
    compression: true
  thresholds:
    context_warning: 70
    context_critical: 85
```

## Manual Installation

If automatic installation doesn't work, manually add to Claude Code settings:

```json
{
  "statusLine": "/path/to/ctxd statusline run --server http://localhost:9090"
}
```

## Troubleshooting

### Statusline not updating
- Ensure contextd server is running: `ctxd health`
- Check server URL is correct: `ctxd statusline test`

### Server not responding
- Start the contextd server: `contextd` (MCP mode) or check your configuration

### Wrong metrics displayed
- Verify the endpoint returns expected data: `curl http://localhost:9090/api/v1/status`
