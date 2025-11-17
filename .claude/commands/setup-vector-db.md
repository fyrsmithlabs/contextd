# Setup Vector Database Command

**Command**: `/setup-vector-db`
**Purpose**: Configure local or remote vector database for persistent AI agent memory
**Category**: Configuration

## Synopsis

```bash
/setup-vector-db <mode> [options]
```

## Description

Configures a vector database (Chroma) for storing persistent memory across AI agent sessions. Supports both local (privacy-first, zero-cost) and remote (team collaboration) modes.

## Arguments

### Required

- `<mode>` - Database mode:
  - `local` - Local Chroma database (default, recommended)
  - `remote` - Remote Chroma server
  - `status` - Check current configuration

### Optional Flags (for remote mode)

- `--endpoint <url>` - Remote Chroma server URL
- `--api-key <key>` - API key for authentication
- `--collection <name>` - Collection name (default: project name)
- `--test` - Test connection after setup

## Examples

### Setup Local Vector DB (Recommended)

```bash
/setup-vector-db local
```

**What this does**:
1. Creates `~/.claude/chroma-data/` directory
2. Configures Chroma MCP server in Claude Desktop config
3. Initializes SQLite-backed vector database
4. Verifies MCP server is running
5. Tests storage and retrieval

**Output**:
```
✓ Creating local vector database...
✓ Directory: ~/.claude/chroma-data/
✓ Configuring MCP server in ~/.config/Claude/claude_desktop_config.json
✓ Testing storage: OK
✓ Testing retrieval: OK

✓ Local vector database configured successfully!

Next steps:
1. Store information: "Store in memory: [content]"
2. Retrieve information: "What do you remember about [topic]?"
3. See usage patterns: docs/specs/vector-db-usage-patterns.md
```

### Setup Remote Vector DB

```bash
/setup-vector-db remote --endpoint https://chroma.example.com --api-key sk_xxx
```

**What this does**:
1. Validates remote endpoint accessibility
2. Tests authentication with API key
3. Creates or connects to collection
4. Configures MCP server with remote settings
5. Verifies storage and retrieval

**Output**:
```
✓ Connecting to remote Chroma server...
✓ Endpoint: https://chroma.example.com
✓ Testing authentication: OK
✓ Collection: git-template (auto-detected from project)
✓ Configuring MCP server
✓ Testing storage: OK
✓ Testing retrieval: OK

✓ Remote vector database configured successfully!

Next steps:
1. Store information: "Store in memory: [content]"
2. Retrieve information: "What do you remember about [topic]?"
3. Team members can connect to same collection for shared knowledge
```

### Check Current Configuration

```bash
/setup-vector-db status
```

**Output**:
```
Vector Database Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Mode: Local
Location: ~/.claude/chroma-data/
Size: 2.3 MB
Collections: 1
  - git-template: 127 entries
MCP Server: Running (PID: 12345)
Last Backup: 2025-10-24 15:30:00

Health: ✓ All systems operational
```

## Implementation

The command executes: `.scripts/setup-vector-db.sh`

## Prerequisites

- Claude Desktop with MCP support
- uvx (Python package runner) for local mode
- Network access for remote mode

## Local Mode Details

### Directory Structure

```
~/.claude/
├── chroma-data/           # Vector database storage
│   ├── chroma.sqlite3     # SQLite database
│   └── [collection-id]/   # Collection data
├── backups/               # Optional backups
│   └── chroma-backup-YYYYMMDD.tar.gz
└── data/
    └── checkpoints/       # Session checkpoints
```

### MCP Configuration

Updates `~/.config/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "chroma": {
      "command": "uvx",
      "args": [
        "chroma-mcp",
        "--client-type",
        "persistent",
        "--data-dir",
        "/home/user/.claude/chroma-data"
      ]
    }
  }
}
```

### Benefits of Local Mode

- ✅ **Privacy**: All data stays on your machine
- ✅ **Zero cost**: No cloud fees
- ✅ **Offline**: Works without internet
- ✅ **Fast**: No network latency
- ✅ **Simple**: One-time setup

### Tradeoffs of Local Mode

- ⚠️ **No sync**: Not shared across devices
- ⚠️ **Manual backup**: Must backup `~/.claude/chroma-data/`
- ⚠️ **Single user**: No team collaboration

## Remote Mode Details

### Supported Backends

- **Chroma Cloud**: Managed Chroma service
- **Self-hosted**: Your own Chroma server
- **Custom**: Any Chroma-compatible API

### Authentication

- API Key (recommended)
- Token-based
- No auth (development only)

### Collection Naming

**Auto-detected** from:
1. Git repository name (if in git repo)
2. Current directory name
3. Manual override with `--collection`

**Format**: `{org}/{project}` (e.g., `axyzlabs/git-template`)

### Benefits of Remote Mode

- ✅ **Team collaboration**: Shared knowledge base
- ✅ **Cross-device**: Access from anywhere
- ✅ **Automatic backup**: Cloud provider handles it
- ✅ **Scalable**: Handle large datasets

### Tradeoffs of Remote Mode

- ⚠️ **Cost**: Potential cloud fees
- ⚠️ **Privacy**: Data stored externally
- ⚠️ **Latency**: Network overhead
- ⚠️ **Availability**: Requires internet

## Backup & Restore

### Local Backup

```bash
# Automatic
/setup-vector-db local --enable-backups

# Manual
tar -czf ~/.claude/backups/chroma-backup-$(date +%Y%m%d).tar.gz ~/.claude/chroma-data/
```

### Restore from Backup

```bash
tar -xzf ~/.claude/backups/chroma-backup-YYYYMMDD.tar.gz -C ~/
```

### Remote Backup

Remote mode backups handled by cloud provider. Check provider documentation for:
- Snapshot schedules
- Point-in-time recovery
- Export capabilities

## Troubleshooting

### Issue: MCP server not starting

**Symptoms**: "Cannot connect to Chroma" errors

**Solutions**:
```bash
# Check MCP status
/mcp

# View logs
tail -f ~/.config/Claude/logs/mcp*.log

# Restart Claude Desktop
```

### Issue: Permission denied on ~/.claude/chroma-data/

**Symptoms**: "Permission denied" when storing data

**Solutions**:
```bash
# Fix permissions
chmod -R 755 ~/.claude/chroma-data/
chown -R $USER ~/.claude/chroma-data/
```

### Issue: Remote endpoint not accessible

**Symptoms**: "Connection refused" or "Timeout"

**Solutions**:
```bash
# Test endpoint
curl https://chroma.example.com/api/v1/heartbeat

# Check firewall/VPN
# Verify API key is correct
# Contact remote admin
```

### Issue: Collection not found

**Symptoms**: "Collection does not exist"

**Solutions**:
```bash
# List collections
/setup-vector-db status

# Create collection explicitly
/setup-vector-db remote --endpoint [url] --api-key [key] --collection [name]
```

## Migration Between Modes

### Local → Remote

```bash
# 1. Export local data
/setup-vector-db export --output local-export.json

# 2. Setup remote
/setup-vector-db remote --endpoint [url] --api-key [key]

# 3. Import to remote
/setup-vector-db import --input local-export.json
```

### Remote → Local

```bash
# 1. Export remote data
/setup-vector-db export --output remote-export.json

# 2. Setup local
/setup-vector-db local

# 3. Import to local
/setup-vector-db import --input remote-export.json
```

## Related Documentation

- **Usage Patterns**: `docs/specs/vector-db-usage-patterns.md`
- **Architecture**: `docs/specs/agent-architecture-evaluation.md`
- **Context Optimization**: `~/.claude/docs/context-optimization-guide.md`
- **Checkpoints**: Built into Claude Code (`/checkpoint` commands)

## See Also

- `/checkpoint save` - Save session checkpoint to vector DB
- `/checkpoint list` - List recent checkpoints
- `/checkpoint search` - Search checkpoints
- `docs/specs/vector-db-usage-patterns.md` - Usage patterns and best practices

---

**Version**: 1.0.0
**Last Updated**: 2025-10-25
**Status**: Active
