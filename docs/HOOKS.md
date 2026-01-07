# Claude Code Hook Setup Guide

This guide covers integrating contextd with Claude Code using lifecycle hooks for automatic memory search, checkpoints, and session management.

---

## Quick Start

### 1. Add contextd to Claude Code MCP Configuration

**Claude Code CLI** (`~/.claude/settings.json`):

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-e", "CONTEXTD_EMBEDDINGS_PROVIDER=fastembed",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

**Claude Desktop** (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-e", "CONTEXTD_EMBEDDINGS_PROVIDER=fastembed",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

### 2. Verify Connection

After restarting Claude Code, the following tools should be available:
- `memory_search`, `memory_record`, `memory_feedback`
- `checkpoint_save`, `checkpoint_list`, `checkpoint_resume`
- `remediation_search`, `remediation_record`
- `repository_index`, `repository_search`
- `troubleshoot_diagnose`

---

## Available Hooks

| Hook | Trigger | Purpose |
|------|---------|---------|
| `session_start` | New session begins | Resume from checkpoint, search memories |
| `session_end` | Session ends | Record learnings, save checkpoint |
| `before_clear` | Before `/clear` command | Auto-checkpoint before clearing |
| `after_clear` | After `/clear` command | Resume prompt |
| `context_threshold` | Context usage reaches threshold | Auto-checkpoint warning |

---

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR` | `false` | Auto-save checkpoint before `/clear` |
| `CONTEXTD_AUTO_RESUME_ON_START` | `true` | Offer to resume from last checkpoint |
| `CONTEXTD_CHECKPOINT_THRESHOLD` | `70` | Context % that triggers threshold hook (1-99) |
| `CONTEXTD_VERIFY_BEFORE_CLEAR` | `true` | Prompt before clearing context |

### Config File

Create `~/.config/contextd/config.yaml`:

```yaml
hooks:
  auto_checkpoint_on_clear: true
  auto_resume_on_start: true
  checkpoint_threshold_percent: 70
  verify_before_clear: true
```

Environment variables override config file values.

---

## Common Use Cases

### Use Case 1: Auto-checkpoint on High Context

Save checkpoints automatically when context reaches 70%:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-e", "CONTEXTD_CHECKPOINT_THRESHOLD=70",
        "-e", "CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=true",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

**Behavior**: At 70% context usage, contextd triggers the `context_threshold` hook. Configure Claude Code to respond by saving a checkpoint.

### Use Case 2: Session Lifecycle Management

Enable auto-resume and auto-checkpoint:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-e", "CONTEXTD_AUTO_RESUME_ON_START=true",
        "-e", "CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=true",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

**Behavior**:
- Session start: Offers to resume from last checkpoint
- Before `/clear`: Automatically saves checkpoint
- After clear: Offers to resume

### Use Case 3: Conservative Mode

Prompt before all actions (safer for new users):

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-e", "CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=false",
        "-e", "CONTEXTD_VERIFY_BEFORE_CLEAR=true",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

**Behavior**: Always prompts before clearing, never auto-checkpoints.

### Use Case 4: Aggressive Auto-checkpoint

For long debugging sessions (50% threshold):

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-e", "CONTEXTD_CHECKPOINT_THRESHOLD=50",
        "-e", "CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR=true",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

---

## HTTP API Integration

contextd also exposes an HTTP API for threshold triggers and status checks.

### Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/status` | GET | Service health and status |
| `/api/v1/threshold` | POST | Trigger context threshold hook |
| `/api/v1/scrub` | POST | Scrub secrets from text |

### Example: Trigger Context Threshold

```bash
curl -X POST http://localhost:9090/api/v1/threshold \
  -H "Content-Type: application/json" \
  -d '{"percentage": 75, "session_id": "abc123"}'
```

### Example: Check Status

```bash
curl http://localhost:9090/api/v1/status
```

---

## Running Without Docker

### Build from Source

```bash
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd
go build -o contextd ./cmd/contextd
```

### Configure Claude Code for Local Binary

```json
{
  "mcpServers": {
    "contextd": {
      "command": "/path/to/contextd",
      "args": ["--mcp", "--no-http"],
      "env": {
        "CONTEXTD_EMBEDDINGS_PROVIDER": "fastembed",
        "CONTEXTD_VECTORSTORE_CHROMEM_PATH": "~/.local/share/contextd"
      }
    }
  }
}
```

---

## Troubleshooting

### Hooks Not Triggering

1. **Verify MCP connection**: Check that contextd tools appear in Claude Code
2. **Check logs**: Run contextd with `-v` for verbose logging
3. **Verify env vars**: Ensure environment variables are set correctly

### Checkpoints Not Resuming

1. **Check project path**: Ensure `project_path` matches between sessions
2. **Verify tenant ID**: Use consistent `tenant_id` across sessions
3. **List checkpoints**: Use `checkpoint_list` to verify checkpoints exist

### Embedding Errors (401 Unauthorized)

This indicates the embedding provider is misconfigured:

1. **Use FastEmbed (local)**: Set `CONTEXTD_EMBEDDINGS_PROVIDER=fastembed`
2. **Check TEI endpoint**: If using TEI, verify `CONTEXTD_EMBEDDINGS_BASE_URL`

### Docker Volume Permissions

```bash
# Fix permissions on data volume
docker run --rm -v contextd-data:/data alpine chown -R 1000:1000 /data
```

---

## Best Practices

1. **Start conservative**: Begin with prompts enabled, then automate as you gain confidence
2. **Set appropriate thresholds**: Use 70% for normal work, 50-60% for complex debugging
3. **Write descriptive summaries**: When saving checkpoints, include context about current task
4. **Combine with memory**: Record learnings (`memory_record`) alongside checkpoints

---

## MCP Tools Reference

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant past strategies/learnings |
| `memory_record` | Save new learning from current session |
| `memory_feedback` | Rate if a memory was helpful (adjusts confidence) |
| `checkpoint_save` | Save session state for later resumption |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from a checkpoint (summary/context/full) |
| `remediation_search` | Find fixes for error patterns |
| `remediation_record` | Record a new error fix |
| `repository_index` | Index repository for semantic search |
| `repository_search` | Semantic search over indexed code |
| `troubleshoot_diagnose` | AI-powered error diagnosis |

---

## Related Documentation

- [Architecture Overview](./architecture.md)
- [Configuration Reference](./configuration.md)
- [Troubleshooting Guide](./troubleshooting.md)
