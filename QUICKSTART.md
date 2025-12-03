# contextd Quick Start

AI agent memory and context management for Claude Code.

## Option 1: Build Locally

```bash
# Clone the repo
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd

# Build the container
docker build -t contextd:latest .

# Add to Claude Code config
```

## Option 2: Docker Hub (when available)

```bash
# Pull the image (once pushed to Docker Hub)
docker pull fyrsmithlabs/contextd:latest
```

## Configure Claude Code

Add this to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "contextd:latest"
      ]
    }
  }
}
```

If using Docker Hub image, replace `contextd:latest` with `fyrsmithlabs/contextd:latest`.

## Restart Claude Code

After adding the config, restart Claude Code. The MCP tools will be available:

- `memory_search` - Find relevant past strategies
- `memory_record` - Save new memories
- `memory_feedback` - Rate memory helpfulness
- `checkpoint_save` - Save context snapshot
- `checkpoint_list` - List available checkpoints
- `checkpoint_resume` - Resume from checkpoint
- `remediation_search` - Find error fix patterns
- `remediation_record` - Record new fixes
- `troubleshoot_diagnose` - Diagnose errors with AI

## Data Persistence

All data persists in the `contextd-data` Docker volume:

```bash
# View data
docker volume inspect contextd-data

# Backup
docker run --rm -v contextd-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/contextd-backup.tar.gz /data

# Reset (WARNING: deletes all data)
docker volume rm contextd-data
```

## Architecture

The container bundles:
- **contextd** - MCP server for Claude Code
- **Qdrant** - Vector database for semantic search
- **FastEmbed** - Local embeddings (ONNX, no API calls)

Everything runs locally. No external API calls required.
