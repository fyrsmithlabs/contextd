# Troubleshooting Guide

This guide covers common issues you may encounter when installing, configuring, or using contextd. If your issue is not covered here, please [open an issue on GitHub](https://github.com/fyrsmithlabs/contextd/issues) or ask in the [Discord](https://discord.gg/Q6dNAUjgPH).

---

## Table of Contents

- [Checking contextd Status](#checking-contextd-status)
- [First-Run Issues](#first-run-issues)
- [MCP Server Connection Problems](#mcp-server-connection-problems)
- [Storage and Data Issues](#storage-and-data-issues)
- [Embedding Issues](#embedding-issues)
- [Secret Scrubbing Issues](#secret-scrubbing-issues)
- [Memory and Search Issues](#memory-and-search-issues)
- [Checkpoint Issues](#checkpoint-issues)
- [Context-Folding Issues](#context-folding-issues)
- [Repository Indexing Issues](#repository-indexing-issues)
- [Plugin Installation Issues](#plugin-installation-issues)
- [Docker-Specific Issues](#docker-specific-issues)
- [Port Conflicts](#port-conflicts)
- [Debug Logging](#debug-logging)
- [Resetting and Cleaning Data](#resetting-and-cleaning-data)
- [Error Codes Reference](#error-codes-reference)
- [Getting Help](#getting-help)

---

## Checking contextd Status

Before diving into specific issues, check the overall health of your contextd installation.

### Verify the binary is installed

```bash
which contextd
contextd --version
```

If `contextd` is not found, see ["contextd not found" after installation](#contextd-not-found-after-installation).

### Check MCP connection in Claude Code

In a Claude Code session, type:

```
/mcp
```

You should see:

```
contextd - connected
```

If it shows disconnected or missing, see [MCP Server Connection Problems](#mcp-server-connection-problems).

### Use the ctxd CLI for diagnostics

```bash
# Check MCP configuration
ctxd mcp status

# Check server health (requires HTTP mode)
ctxd health
```

**Note:** `ctxd health` checks the HTTP server on port 9090. If you are running with `--no-http` (the default for MCP mode), this command will report "connection refused" -- that is expected. Use `ctxd mcp status` instead for MCP-mode installations.

### Test contextd manually

```bash
# Start contextd in the foreground to check for startup errors
contextd --mcp --no-http

# You should see no errors. Press Ctrl+C to exit.
```

If you see errors at startup, the output will tell you what failed (ONNX download, config loading, vectorstore initialization, etc.).

---

## First-Run Issues

### ONNX runtime and model download

On first run, contextd automatically downloads two components:

1. **ONNX runtime** (~50MB) -- the inference engine for local embeddings
2. **Embedding model** (~50MB) -- the `BAAI/bge-small-en-v1.5` model

You will see output like:

```
ONNX runtime not found. Downloading v1.23.0...
Downloaded to ~/.config/contextd/lib/libonnxruntime.so
Downloading fast-bge-small-en-v1.5...
```

This is normal and only happens once. Subsequent runs start instantly.

**If the download fails:**

```bash
# Manually trigger the download
ctxd init

# Force re-download if files are corrupted
ctxd init --force
```

Common download failures:

| Cause | Fix |
|-------|-----|
| Network timeout | Check internet connection; corporate proxies may block the download |
| Disk space | Ensure at least 200MB free in `~/.config/contextd/` |
| Permission denied | Ensure write access to `~/.config/contextd/` (see [Permission denied errors](#permission-denied-errors)) |

### First tool call is slow

The very first MCP tool call in a session may take 2-5 seconds while contextd:

1. Loads the ONNX runtime into memory
2. Loads the embedding model
3. Initializes the vector store

This is a one-time cost per session. All subsequent tool calls are fast (typically under 100ms).

If **every** tool call is slow, check:

- Available system RAM (the embedding model needs ~500MB-1GB)
- Disk I/O speed (slow spinning drives can affect model loading)
- Whether you are running via Docker with `docker run` (each call starts a new container -- see [Docker-Specific Issues](#slow-tool-calls-with-docker) for the persistent container approach)

---

## MCP Server Connection Problems

### "contextd not found" after installation

The `contextd` binary must be in your shell's PATH.

```bash
# Check if contextd is found
which contextd

# If not found, check common install locations
ls ~/.local/bin/contextd
ls /usr/local/bin/contextd
ls /opt/homebrew/bin/contextd
```

**Fix:** Add the install directory to your PATH. Add one of these to your shell config (`~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`):

```bash
# For ~/.local/bin installs
export PATH="$HOME/.local/bin:$PATH"

# For Homebrew on Apple Silicon
export PATH="/opt/homebrew/bin:$PATH"
```

Then restart your shell **and** Claude Code.

**Important:** Claude Code inherits the PATH from your shell. If you installed contextd in a new terminal but Claude Code was already running, you must restart Claude Code to pick up the new PATH.

### MCP settings.json errors

The MCP configuration lives in `~/.claude/settings.json`. Common mistakes:

**Invalid JSON syntax:**

```bash
# Validate your settings.json
python3 -c "import json; json.load(open('$HOME/.claude/settings.json'))"
```

**Correct structure:**

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "contextd",
      "args": ["--mcp", "--no-http"]
    }
  }
}
```

Common mistakes:
- Missing outer `"mcpServers"` key
- Using `"server"` instead of `"command"`
- Trailing commas in JSON (not allowed)
- Wrong path to the binary (use an absolute path if `contextd` is not in PATH)

**Binary not at expected path:**

If contextd is not in your PATH, use an absolute path in settings.json:

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "/Users/yourname/.local/bin/contextd",
      "args": ["--mcp", "--no-http"]
    }
  }
}
```

### MCP server starts but immediately disconnects

Test manually to see error output:

```bash
contextd --mcp --no-http
```

If the process exits immediately, common causes:

- **Missing ONNX runtime**: Run `ctxd init` to download it.
- **Corrupted data directory**: See [Resetting and Cleaning Data](#resetting-and-cleaning-data).
- **Config file syntax error**: Check `~/.config/contextd/config.yaml` for YAML syntax errors.
- **Vectorstore provider invalid**: If `VECTORSTORE_PROVIDER` is set to an unrecognized value, contextd exits with an error. Valid values are `chromem` (default) and `qdrant`.

### Claude Code not showing contextd tools

After configuring MCP, restart Claude Code. If tools still do not appear:

1. Check `~/.claude/settings.json` for valid JSON
2. Verify the command path: `which contextd`
3. Test the binary manually: `contextd --version`
4. Verify Docker is running (if using Docker mode): `docker info`
5. Check for errors: `contextd --mcp --no-http 2>&1 | head -20`

### Auto-configuration

Use the built-in tooling instead of manual configuration:

```bash
# Auto-configure MCP settings
ctxd mcp install

# Verify configuration
ctxd mcp status

# Remove configuration
ctxd mcp uninstall
```

Or use the plugin command in Claude Code:

```
/contextd-init
```

---

## Storage and Data Issues

### Permission denied errors

contextd stores data in `~/.config/contextd/` by default. Ensure you own that directory:

```bash
# Check ownership
ls -la ~/.config/contextd/

# Fix ownership if needed
sudo chown -R $(whoami) ~/.config/contextd/

# Fix permissions (contextd uses 0700 for directories, 0600 for files)
chmod 700 ~/.config/contextd/
chmod 700 ~/.config/contextd/lib/
chmod 700 ~/.config/contextd/vectorstore/
```

If you installed contextd as root but run Claude Code as your user, the data directory may be owned by root. Fix with the `chown` command above.

### Binary permission denied

```bash
chmod +x ~/.local/bin/contextd
chmod +x ~/.local/bin/ctxd
```

### Data path confusion

contextd uses different default paths depending on how it is run:

| Context | Data Path |
|---------|-----------|
| Native binary | `~/.config/contextd/` |
| Docker | `/data/` (mapped to a Docker volume) |
| Environment override | `$CONTEXTD_DATA_PATH` or `$VECTORSTORE_PATH` |

If memories are "disappearing" between sessions, verify that the data path is consistent. Switching between Docker and native binary uses different storage locations unless you explicitly map the same path.

### Data directory structure

A healthy data directory looks like this:

```
~/.config/contextd/
  vectorstore/          # Memories, checkpoints, remediations
  lib/                  # ONNX runtime (libonnxruntime.so or .dylib)
  models/               # Embedding models (auto-downloaded)
  config.yaml           # Optional configuration file
```

If `vectorstore/` is missing or empty, contextd will create it on first use.

### chromem data issues

contextd uses chromem as its default embedded vector database. If you encounter data issues:

**"collection not found" errors:**

```
collection not found: memories_<tenant_id>
```

This usually means the tenant ID derived for this session does not match the one used when data was stored. See [Tenant ID mismatch](#tenant-id-mismatch).

**Disk full:**

```bash
# Check disk space
df -h ~/.config/contextd
```

Free up disk space or change the data directory via `CONTEXTD_VECTORSTORE_CHROMEM_PATH`.

**Corrupted vectorstore:**

If contextd fails to start with vectorstore errors, the data files may be corrupted:

```bash
# Back up current data
cp -r ~/.config/contextd/vectorstore ~/.config/contextd/vectorstore.bak

# Remove corrupted data
rm -rf ~/.config/contextd/vectorstore

# Restart contextd -- it will recreate empty collections
```

After resetting, you will need to re-record memories and re-index repositories.

---

## Embedding Issues

### ONNX runtime errors

**"ONNX runtime not found":**

```bash
# Download ONNX runtime
ctxd init

# Force re-download
ctxd init --force
```

**"unsupported platform" (ErrUnsupportedPlatform):**

Local ONNX embeddings require CGO and are supported on:
- macOS (Apple Silicon and Intel)
- Linux (x64 and arm64)

Windows builds use pure Go and do not support local ONNX embeddings. Use the TEI provider instead:

```bash
export EMBEDDINGS_PROVIDER=tei
export EMBEDDING_BASE_URL=http://localhost:8080
```

**"libonnxruntime.so: cannot open shared object file":**

The ONNX library was not found at the expected path:

```bash
# Check where contextd expects it
ls ~/.config/contextd/lib/

# Re-download
ctxd init --force

# Or set the path explicitly
export ONNX_PATH=$HOME/.config/contextd/lib/libonnxruntime.dylib  # macOS
export ONNX_PATH=$HOME/.config/contextd/lib/libonnxruntime.so      # Linux
```

**"API version mismatch":**

```
Error: API version [22] is not available, only API versions [1, 20] are supported
```

This means the ONNX runtime version does not match what contextd expects. Force re-download:

```bash
ctxd init --force
```

If building from source, ensure you use a compatible ONNX runtime version (v1.23.0 or as specified in the build).

**"FastEmbed not available" (ErrFastEmbedNotAvailable):**

The binary was built without CGO support. Either:

1. Install via Homebrew (builds with CGO): `brew install fyrsmithlabs/contextd/contextd`
2. Use the Docker image (includes ONNX runtime)
3. Use TEI as the embedding provider

### Model download failures

**"model not found" or "failed to load model":**

The embedding model may not have downloaded correctly:

```bash
# Check if models exist
ls ~/.config/contextd/models/

# Re-download by reinitializing
ctxd init --force
```

**Model loaded from wrong directory:**

Before v0.3.0, the model cache defaulted to `./local_cache` (relative to the working directory). This was fixed to use `~/.config/contextd/models/`. If you have models in the old location:

```bash
# Remove stale model caches from project directories
rm -rf ./local_cache

# Re-download to correct location
ctxd init --force
```

### Vector dimension mismatch

```
vector size X does not match configured size Y
```

This happens when the embedding model dimensions do not match `QDRANT_VECTOR_SIZE`. The default model (`BAAI/bge-small-en-v1.5`) produces 384-dimensional vectors.

| Model | Dimensions |
|-------|------------|
| `BAAI/bge-small-en-v1.5` | 384 |
| `BAAI/bge-base-en-v1.5` | 768 |
| `sentence-transformers/all-MiniLM-L6-v2` | 384 |

**Fix:** Ensure `QDRANT_VECTOR_SIZE` matches your model (or omit it to use the default of 384).

If you switched models, you must either reindex your data or reset the vectorstore.

### Using TEI instead of local ONNX

If local ONNX embeddings are not working, you can use HuggingFace Text Embeddings Inference (TEI):

```bash
# Start a TEI server
docker run -p 8080:80 ghcr.io/huggingface/text-embeddings-inference:latest \
  --model-id BAAI/bge-small-en-v1.5

# Configure contextd to use TEI
export EMBEDDINGS_PROVIDER=tei
export EMBEDDING_BASE_URL=http://localhost:8080
```

See [Configuration Reference](./configuration.md#embedding-models) for more embedding options.

---

## Secret Scrubbing Issues

### Secret detection issues

contextd uses [gitleaks](https://github.com/gitleaks/gitleaks) (embedded as a Go SDK) to scrub secrets from all MCP tool responses. There is no external `gitleaks` binary required.

### Scrubbing failures

If you see scrubbing-related errors:

1. **Check system resources**: Scrubbing failures are often caused by insufficient memory.
   ```bash
   free -m        # Linux
   vm_stat        # macOS
   ```

2. **Transient failure**: Scrubbing failures are typically transient. The operation is **blocked** (fail-safe) to prevent secret leakage. Retry the tool call.

3. **Check logs for details**: Run contextd with debug logging to see the underlying error:
   ```bash
   LOG_LEVEL=debug contextd --mcp --no-http
   ```

### False positives in secret detection

gitleaks may flag content that is not actually a secret (e.g., long hexadecimal strings, UUIDs, Base64-encoded content, example tokens in documentation). This is by design -- false positives are preferred over false negatives for security.

**Workarounds:**

- Avoid including example API keys or tokens in memory content
- If a tool response is being scrubbed incorrectly, the content is replaced with `[REDACTED]` -- the operation still succeeds, but some content may be lost
- Currently there is no user-facing configuration to add scrubbing exceptions

### Scrubbing and context-folding

When using `branch_return`, all returned content is automatically scrubbed for secrets before being passed back to the main context. If `branch_return` fails with a scrubbing error (FOLD014), this is fail-safe behavior -- the content is blocked to prevent potential secret leakage.

See [Error Codes: FOLD014](./api/error-codes.md#fold014-scrubbing-failed) for details.

---

## Memory and Search Issues

### memory_search returns no results

If `memory_search` returns empty results despite having recorded memories:

#### 1. Tenant ID mismatch

contextd derives tenant and project IDs from your git remote URL. If the derivation differs between sessions (e.g., HTTPS vs SSH remote, or no remote configured), memories are stored under one tenant but searched under another.

```bash
# Check your git remote
git remote -v

# The tenant ID is derived as: github.com/username (from the remote URL)
# The project ID is derived as: repository-name
```

**Fix:** Ensure your git remote URL is consistent across all sessions. If you cloned via HTTPS in one session and SSH in another, the tenant IDs may differ. See [Tenant ID mismatch](#tenant-id-mismatch) for more details.

#### 2. No memories recorded yet

If this is a new project, no memories exist. Record some first:

```
/contextd-remember
```

#### 3. Query too specific or too vague

Semantic search works best with natural language queries of moderate specificity:

```
# Too vague -- won't match well
memory_search(query="stuff")

# Too specific -- embedding similarity will be low
memory_search(query="the exact error from line 42 of auth.go on Tuesday")

# Good -- natural language with key terms
memory_search(query="authentication error fix JWT token")
```

#### 4. Non-git directory

In directories without a git repository, contextd uses a fallback identifier. Memories recorded in a git repo will not appear when searching from a non-git directory (and vice versa).

### memory_record fails

Common causes:

- **Empty title or content**: Both are required. The title should be concise (50-100 chars) and the content should explain the learning.
- **Missing project_id**: contextd derives this from git. If it cannot determine the project, the operation fails. Ensure you are in a git repository with a remote configured.
- **Invalid outcome**: Must be `"success"` or `"failure"`.

### Search relevance is poor

Semantic search quality depends on the embedding model. Tips for better results:

- Use descriptive titles when recording memories
- Include context in the content (language, framework, component)
- Use `memory_feedback` to rate results -- this adjusts confidence scores over time
- Run `memory_consolidate` periodically to merge related memories into refined summaries

---

## Checkpoint Issues

### Checkpoints not resuming

**1. Project path mismatch:**

Checkpoints are associated with a project path. If you run Claude Code from a different directory, the checkpoint may not be found.

```bash
# List available checkpoints to verify they exist
# In Claude Code: "List my checkpoints"
```

**2. Tenant ID mismatch:**

Same as memory search -- ensure consistent tenant IDs across sessions. See [Tenant ID mismatch](#tenant-id-mismatch).

**3. Checkpoint content too large:**

Checkpoints have a maximum content size (default: 1024 KB). If a save silently fails, the content may have exceeded this limit:

```bash
# Increase the limit if needed
export CHECKPOINT_MAX_CONTENT_SIZE_KB=2048
```

### Checkpoint not saving

- Verify contextd is connected: `/mcp` in Claude Code
- Ensure you are in a git repository (for tenant/project derivation)
- Check available disk space: `df -h ~/.config/contextd`

### Budget tracking issues

If context-folding branches report incorrect budget usage or budget tracking seems inconsistent:

1. **Check branch status**: Use `branch_status` to see current budget consumption.
2. **Budget was exhausted (FOLD008)**: Create a new branch with a larger budget.
3. **System error (FOLD009 / FOLD011)**: These indicate internal tracking issues. Create a new branch and [report the issue](https://github.com/fyrsmithlabs/contextd/issues).

See [Error Codes: Context-Folding](./api/error-codes.md#context-folding-errors-fold001-fold022) for all budget-related error codes.

---

## Context-Folding Issues

### branch_create fails

| Error | Cause | Fix |
|-------|-------|-----|
| FOLD004: Max depth exceeded | Too many nested branches | Return from child branches before creating new ones |
| FOLD013: Max concurrent branches | Too many active branches (default limit: 10) | Complete existing branches first |
| FOLD012: Rate limit exceeded | Too many operations per second | Wait a moment and retry |
| FOLD010: Invalid budget | Budget is zero, negative, or exceeds 100,000 | Use a positive integer between 1 and 100,000 |

### branch_return fails

| Error | Cause | Fix |
|-------|-------|-----|
| FOLD003: Branch not active | Branch was already completed or terminated | Check branch status; create a new branch |
| FOLD007: Active child branches | Child branches still running | Complete all child branches first |
| FOLD014: Scrubbing failed | Secret scrubbing failed on return content | Retry the operation; check system resources |
| FOLD021: Message too long | Return message exceeds 10,000 characters | Summarize findings instead of copying full content |
| FOLD006: Cannot return from root | Trying to return from the root session, not a branch | Only call `branch_return` from within an active branch |

### Session authorization errors (FOLD022)

This error means the caller is not authorized to access the branch or session. In single-user mode (default), this should not occur. If it does:

1. Ensure you are not mixing session IDs between different tenants
2. Check that the session_id matches what was returned by `branch_create`

See [Error Codes: FOLD022](./api/error-codes.md#fold022-session-unauthorized) for details.

---

## Repository Indexing Issues

### repository_index fails

**"git remote not found":**

Repository indexing uses git to determine the project identity. Ensure you are in a git repository with a remote configured:

```bash
git remote -v
```

If no remote exists, `repository_index` may fail or use a fallback identifier.

**Permission errors:**

contextd needs read access to the files it indexes. Check that the files are readable:

```bash
ls -la /path/to/your/project
```

### repository_search returns "collection not found"

This is a known issue (#19) when `repository_index` and `repository_search` derive different tenant IDs. The fix is to use the `collection_name` returned by `repository_index`:

```
# When you index, note the collection_name in the output
repository_index(project_path="/path/to/project")
# Output includes: collection_name: "repo_github.com_user_project_main"

# Use that collection_name for subsequent searches
repository_search(collection_name="repo_github.com_user_project_main", query="...")
```

### Large repositories are slow to index

Repository indexing reads and embeds every file that matches the include patterns. For large repositories:

- Create a `.contextdignore` file in the project root to exclude large or irrelevant directories
- The default excludes already skip `.git/`, `node_modules/`, `vendor/`, and `__pycache__/`
- Indexing is incremental -- subsequent runs only process changed files
- Configure exclusions via `REPOSITORY_IGNORE_FILES` and `REPOSITORY_FALLBACK_EXCLUDES` environment variables

---

## Plugin Installation Issues

### "claude plugins add" fails

```bash
claude plugins add fyrsmithlabs/marketplace
```

If this fails:

- Ensure you have the latest version of Claude Code: `claude update`
- Check your internet connection
- Verify the plugin repository exists: https://github.com/fyrsmithlabs/marketplace

### Plugin commands not appearing

After installing the plugin, restart Claude Code. Plugin commands (like `/contextd-init`) are loaded at startup.

If commands still do not appear:

1. Verify the plugin is installed: `claude plugins list`
2. Check for plugin errors in Claude Code output
3. Reinstall the plugin:
   ```bash
   claude plugins remove fyrsmithlabs/marketplace
   claude plugins add fyrsmithlabs/marketplace
   ```

### /contextd-init fails

The init command downloads the contextd binary and configures MCP. If it fails:

1. **Binary download failed**: Check your internet connection. The binary is downloaded from GitHub Releases.
2. **PATH not set**: If `~/.local/bin` is not in your PATH, the init command warns you. Add it to your shell config.
3. **Architecture detection failed**: The wrapper script detects your platform (darwin/linux, amd64/arm64). If detection fails, install manually from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases).

---

## Docker-Specific Issues

### Slow tool calls with Docker

Each `docker run` invocation starts a new container, incurring ~500ms overhead per tool call. This adds up quickly.

**Fix:** Use the persistent container approach:

```bash
# Create a persistent container
docker run -d \
  --name contextd-server \
  --restart unless-stopped \
  -v contextd-data:/data \
  -v "${HOME}:${HOME}:ro" \
  --user "$(id -u):$(id -g)" \
  ghcr.io/fyrsmithlabs/contextd:latest \
  tail -f /dev/null
```

Then configure MCP to use `docker exec` instead of `docker run`. See [Docker Guide: Persistent Container](./DOCKER.md#persistent-container-advanced) for full setup.

### "directory not found" in Docker

Docker containers cannot access host filesystem paths unless explicitly mounted. Ensure:

1. The `${PWD}:${PWD}` mount is present (maps your working directory into the container at the same path)
2. On macOS, Docker Desktop has permission to access the directory (Settings -> Resources -> File sharing)
3. You started Claude Code from the correct directory

### Docker volume permission errors

```bash
# Fix permissions on the data volume
docker run --rm -v contextd-data:/data alpine chown -R 1000:1000 /data
```

Replace `1000:1000` with your actual user/group IDs (`id -u` and `id -g`).

### Container exits immediately

```bash
# Check container logs for the error
docker logs <container-id>
```

Common causes:

- Missing `/data` volume mount
- ONNX runtime issues (rare, usually on ARM platforms)
- Insufficient memory allocated to Docker

### Data not persisting between container restarts

```bash
# Verify volume exists
docker volume ls | grep contextd

# Inspect volume
docker volume inspect contextd-data

# Check volume contents
docker run --rm -v contextd-data:/data alpine ls -la /data
```

If the volume does not exist, create it:

```bash
docker volume create contextd-data
```

---

## Port Conflicts

### "address already in use" on port 9090

contextd's HTTP server listens on port 9090 by default. If another process (or another contextd instance) is using that port:

**Option 1: Disable the HTTP server (recommended for MCP use)**

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "contextd",
      "args": ["--mcp", "--no-http"]
    }
  }
}
```

The `--no-http` flag disables the HTTP server entirely. This is the default for plugin configurations and allows multiple Claude Code sessions to run simultaneously.

**Option 2: Use a different port**

```bash
export SERVER_PORT=9091
```

**Option 3: Find and stop the conflicting process**

```bash
# Find what is using port 9090
lsof -i :9090

# Kill the process if it is a stale contextd instance
kill <PID>
```

### Qdrant port conflicts (external Qdrant only)

If using external Qdrant and ports 6333/6334 are in use:

```bash
lsof -i :6333
lsof -i :6334
```

Configure alternative ports via `QDRANT_PORT` and `QDRANT_HTTP_PORT` environment variables.

---

## Debug Logging

To get detailed diagnostic output from contextd, enable debug logging.

### Environment variable

```bash
export LOG_LEVEL=debug
```

### In MCP configuration

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "contextd",
      "args": ["--mcp", "--no-http"],
      "env": {
        "LOG_LEVEL": "debug"
      }
    }
  }
}
```

### Running manually with debug output

```bash
LOG_LEVEL=debug contextd --mcp --no-http 2>contextd-debug.log
```

Debug logging includes:

- Tenant ID derivation details
- Embedding model loading steps
- Vector store operations
- Secret scrubbing events
- MCP tool call parameters and timings
- Collection creation and search operations

**Warning:** Debug logs may contain sensitive information (file paths, query content). Do not share debug logs publicly without reviewing them first.

---

## Resetting and Cleaning Data

### Reset all contextd data

This deletes all memories, checkpoints, remediations, and indexed repositories:

```bash
rm -rf ~/.config/contextd/vectorstore
```

contextd will recreate empty collections on the next run.

### Reset only the ONNX runtime and models

```bash
rm -rf ~/.config/contextd/lib
rm -rf ~/.config/contextd/models

# Re-download
ctxd init
```

### Reset MCP configuration

```bash
ctxd mcp uninstall
```

This removes the contextd entry from `~/.claude/settings.json`.

### Reset Docker data

```bash
# Remove the Docker volume (WARNING: destroys all data)
docker volume rm contextd-data

# Recreate it
docker volume create contextd-data
```

### Full clean reinstall

```bash
# 1. Remove all contextd data and config
rm -rf ~/.config/contextd

# 2. Remove the binaries
rm -f ~/.local/bin/contextd
rm -f ~/.local/bin/ctxd

# 3. Remove MCP config
ctxd mcp uninstall 2>/dev/null

# 4. Reinstall
brew install fyrsmithlabs/contextd/contextd
# Or download from GitHub Releases

# 5. Re-initialize
ctxd init
ctxd mcp install
```

### Back up before resetting

Always back up before destructive operations:

```bash
# Back up all data
tar czf contextd-backup-$(date +%Y%m%d).tar.gz ~/.config/contextd/

# Restore later if needed
tar xzf contextd-backup-*.tar.gz -C ~/
```

---

## Error Codes Reference

contextd uses structured error codes for clear, actionable error messages. For the complete reference with detailed descriptions, causes, and resolutions, see [Error Codes](./api/error-codes.md).

### Quick reference for common errors

| Error | Meaning | Resolution |
|-------|---------|------------|
| FOLD001 | Branch not found | Check branch_id; use `branch_status` to list branches |
| FOLD004 | Max nesting depth exceeded | Return from child branches first |
| FOLD008 | Budget exhausted | Create new branch with larger budget |
| FOLD010 | Invalid budget | Use positive integer between 1 and 100,000 |
| FOLD012 | Rate limit exceeded | Wait and retry with backoff |
| FOLD013 | Max concurrent branches | Complete existing branches first |
| FOLD014 | Secret scrubbing failed | Retry; check system resources |
| FOLD022 | Session unauthorized | Verify session ownership and tenant context |
| ErrMissingTenant | No tenant context | Ensure you are in a git repository with a remote |
| ErrTenantFilterInUserFilters | Tenant filter injection blocked | Do not pass tenant_id in user filters |
| ErrMemoryNotFound | Memory does not exist | Verify memory ID; check tenant consistency |
| ErrUnsupportedPlatform | Platform not supported for ONNX | Use TEI embedding provider instead |
| ErrFastEmbedNotAvailable | Binary built without CGO | Install via Homebrew or use Docker |

### Tenant ID mismatch

Many "not found" errors are caused by inconsistent tenant ID derivation. contextd derives tenant and project IDs from your git remote URL:

| Remote URL | Tenant ID | Project ID |
|------------|-----------|------------|
| `git@github.com:user/repo.git` | `github.com/user` | `repo` |
| `https://github.com/user/repo.git` | `github.com/user` | `repo` |
| No remote configured | Fallback hash | Directory name |

If your remote URL changes or is inconsistent between sessions, data stored under one tenant will not be visible under another.

**Diagnose:**

```bash
# Check what git remote contextd will use
git remote get-url origin
```

**Fix:** Ensure all sessions use the same remote URL format. If you see inconsistencies, standardize on one format (e.g., always SSH or always HTTPS).

---

## Getting Help

If this guide does not resolve your issue:

1. **Search existing issues**: https://github.com/fyrsmithlabs/contextd/issues
2. **Enable debug logging** and capture the output (see [Debug Logging](#debug-logging))
3. **Open a new issue** with:
   - contextd version (`contextd --version`)
   - Operating system and architecture (`uname -a`)
   - How you installed contextd (Homebrew, binary, Docker, plugin)
   - The full error message or unexpected behavior
   - Debug log output (redact any sensitive information)
4. **Ask in Discord**: https://discord.gg/Q6dNAUjgPH

### Useful information for bug reports

```bash
# System info
uname -a
contextd --version

# Check MCP config
ctxd mcp status

# Check data directory
ls -la ~/.config/contextd/

# Capture debug logs
LOG_LEVEL=debug contextd --mcp --no-http 2>contextd-debug.log
```

---

## Related Documentation

- [README](../README.md) - Installation and quick start
- [Onboarding Guide](./ONBOARDING.md) - First use tutorial
- [Configuration Reference](./configuration.md) - All configuration options
- [Docker Guide](./DOCKER.md) - Running contextd in Docker
- [Hook Setup Guide](./HOOKS.md) - Claude Code lifecycle integration
- [Error Codes](./api/error-codes.md) - Complete error code reference
- [Architecture](./architecture.md) - Technical architecture overview
