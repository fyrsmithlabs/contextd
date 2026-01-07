# Running contextd in Docker

This guide covers running contextd in a Docker container for users who prefer container isolation over the native binary.

> **Note:** The Claude Code plugin installs the native binary. Use this guide only if you specifically want Docker.

## Quick Start

```bash
# Pull the image
docker pull ghcr.io/fyrsmithlabs/contextd:latest

# Run contextd
docker run -i --rm \
  -v contextd-data:/data \
  -v "${PWD}:${PWD}" \
  -v "${HOME}/.config/contextd:${HOME}/.config/contextd" \
  -w "${PWD}" \
  --user "$(id -u):$(id -g)" \
  ghcr.io/fyrsmithlabs/contextd:latest \
  --mcp --no-http
```

Add to your Claude Code MCP config (`~/.claude/settings.json`):

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-v", "${PWD}:${PWD}",
        "-v", "${HOME}/.config/contextd:${HOME}/.config/contextd",
        "-w", "${PWD}",
        "--user", "1000:1000",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ],
      "env": {}
    }
  }
}
```

Replace `1000:1000` with your user and group IDs. Find them with `id -u` and `id -g`.

## Volume Mounts

contextd requires three volume mounts:

| Mount | Purpose |
|-------|---------|
| `contextd-data:/data` | Persistent storage for vector database and embeddings cache |
| `${PWD}:${PWD}` | Current project directory for `repository_index` |
| `${HOME}/.config/contextd:...` | User configuration (optional but recommended) |

### Why ${PWD}:${PWD}?

The mount path must match inside and outside the container. When you call `repository_index("/home/user/myproject")`, contextd inside the container must find files at that exact path.

### Project Directory Access

The `${PWD}:${PWD}` mount gives contextd access only to your current working directory. If you need to index multiple projects, you have two options:

**Option A: Run Claude Code from the project root**

This is the simplest approach. Start Claude Code from the directory you want to index.

**Option B: Mount your projects directory**

```bash
-v "${HOME}/projects:${HOME}/projects"
```

This gives contextd access to all projects under `~/projects`.

## User ID Mapping

Run the container as your user to avoid permission issues:

```bash
--user "$(id -u):$(id -g)"
```

Files created in mounted volumes will have correct ownership. Without this flag, files are created as root.

In the MCP config, you must hardcode the IDs since shell expansion does not work in JSON:

```json
"--user", "1000:1000"
```

## Docker Desktop for Mac

Docker Desktop on macOS requires explicit permission to access directories outside `/Users`.

1. Open Docker Desktop
2. Go to Settings → Resources → File sharing
3. Add any directories you want to mount (e.g., `/Users/yourname/projects`)
4. Click "Apply & Restart"

Without this step, mounts outside your home directory will fail silently.

## Resource Limits

For large repositories, consider adding resource limits:

```bash
--memory=2g --cpus=2
```

The embedding model (FastEmbed) typically uses 500MB-1GB of RAM. The vector database (chromem) is lightweight.

## Persistent Container (Advanced)

Running a persistent container eliminates startup overhead (~500ms per call). This approach suits users who make many MCP calls per session.

**Create the container:**

```bash
docker run -d \
  --name contextd-server \
  --restart unless-stopped \
  --memory=2g \
  --cpus=2 \
  --user "$(id -u):$(id -g)" \
  -v contextd-data:/data \
  -v "${HOME}:${HOME}:ro" \
  ghcr.io/fyrsmithlabs/contextd:latest \
  tail -f /dev/null
```

**MCP config using docker exec:**

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "docker",
      "args": [
        "exec", "-i", "-w", "${PWD}",
        "contextd-server",
        "contextd", "--mcp", "--no-http"
      ],
      "env": {}
    }
  }
}
```

**Trade-offs:**

- Faster tool calls (no container startup)
- Mounts `${HOME}` read-only instead of just `${PWD}`
- Requires manual container management

**Container commands:**

```bash
docker stop contextd-server    # Stop
docker start contextd-server   # Start
docker logs contextd-server    # View logs
docker rm -f contextd-server   # Remove
```

## Troubleshooting

### "directory not found" errors

The mounted path inside the container does not match the path you requested. Verify:

1. You started Claude Code from the correct directory
2. The `${PWD}:${PWD}` mount is present
3. On macOS, Docker Desktop has permission to access the directory

### Permission denied

Add `--user "$(id -u):$(id -g)"` to run as your user instead of root.

### Container exits immediately

Check logs with `docker logs <container-id>`. Common causes:

- Missing `/data` volume mount
- ONNX runtime issues (rare, usually on ARM)

### Slow tool calls

Each `docker run` incurs ~500ms overhead. Use the persistent container approach if this affects your workflow.

## Data Backup

Back up the contextd data volume:

```bash
docker run --rm \
  -v contextd-data:/data \
  -v "$(pwd):/backup" \
  alpine tar czf /backup/contextd-backup.tar.gz /data
```

Restore:

```bash
docker run --rm \
  -v contextd-data:/data \
  -v "$(pwd):/backup" \
  alpine tar xzf /backup/contextd-backup.tar.gz -C /
```
