Install the contextd MCP server and ctxd CLI.

## What Gets Installed

- **contextd** - MCP server for Claude Code integration
- **ctxd** - CLI tool for manual operations (scrub, init, health checks)

## Step 1: Choose Installation Method

Use AskUserQuestion to ask:

**Question:** "How would you like to install contextd?"

**Options:**
1. **Homebrew (Recommended)** - Quick install via brew, auto-updates
2. **Binary** - Manual download from GitHub releases
3. **Docker** - Container with all dependencies included

---

## Step 2: Installation Instructions

### If Homebrew selected:

```bash
brew install fyrsmithlabs/tap/contextd
```

Verify installation:
```bash
contextd --version
ctxd --version
```

### If Binary selected:

1. Download from GitHub releases:
   ```
   https://github.com/fyrsmithlabs/contextd/releases/latest
   ```

2. Extract and move to PATH:
   ```bash
   tar -xzf contextd_*.tar.gz
   chmod +x contextd ctxd
   mv contextd ctxd ~/.local/bin/  # or /usr/local/bin/
   ```

3. Verify:
   ```bash
   contextd --version
   ctxd --version
   ```

### If Docker selected:

```bash
docker pull ghcr.io/fyrsmithlabs/contextd:latest
```

Note: Docker only includes `contextd`. For `ctxd` CLI, use Homebrew or Binary install.

---

## Step 3: Configure Claude Code

Add to your Claude Code settings (`~/.claude/settings.json`):

### For Homebrew/Binary:

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

### For Docker:

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "${HOME}/.config/contextd:/root/.config/contextd", "ghcr.io/fyrsmithlabs/contextd:latest", "--mcp", "--no-http"]
    }
  }
}
```

---

## Step 4: Verify

After adding to settings.json:

1. Restart Claude Code (or run `/mcp`)
2. Check contextd appears in MCP servers list
3. Test with: `mcp__contextd__memory_search(project_id: "test", query: "hello")`

---

## Troubleshooting

**"contextd not found"**
- Ensure binary is in PATH: `which contextd`
- For Homebrew: `brew link fyrsmithlabs/tap/contextd`

**Docker "permission denied"**
- Add user to docker group: `sudo usermod -aG docker $USER`
- Log out and back in

**MCP server not appearing**
- Check settings.json syntax (valid JSON?)
- Ensure `"type": "stdio"` is present
- Restart Claude Code completely
