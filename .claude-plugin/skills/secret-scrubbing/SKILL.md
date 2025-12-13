---
name: secret-scrubbing
description: Use when setting up PostToolUse hooks for automatic secret scrubbing - configures ctxd CLI to scrub output from Read, Bash, Grep, and WebFetch tools before displaying to context
---

# Secret Scrubbing

## CRITICAL: Defense-in-Depth Only

**Secret scrubbing is a SAFETY NET, NOT permission to read secrets.**

Scrubbing catches mistakes but CANNOT guarantee complete protection:
- Pattern matching may miss novel secret formats
- Secrets briefly exist in memory before scrubbing
- High-entropy detection has false negatives
- Custom/internal secret formats may not be detected

**NEVER intentionally read credential files (`.env`, `credentials.json`, `*.pem`, etc.).**

Instead:
- Check file exists: `test -f .env && echo "exists"`
- Check variable set: `test -n "$API_KEY" && echo "set"`
- Use secret managers and environment injection

Treat scrubbing as the last line of defense, not the first.

---

## CRITICAL PREREQUISITE

**contextd HTTP server MUST be running BEFORE configuring hooks.**

Check: `ctxd health` (expect: "Server Status: ok")
If not running: `contextd &`

Without the server, ALL tool calls will fail after hook configuration.

---

## Overview

Automatic secret scrubbing via PostToolUse hooks. Tool outputs are piped through `ctxd scrub` before appearing in Claude's context, preventing accidental exposure of API keys, passwords, and credentials.

## How It Works

```
┌────────────────────────────────────────────────────────────────┐
│  1. Tool executes (Read, Bash, Grep, WebFetch)                │
├────────────────────────────────────────────────────────────────┤
│  2. PostToolUse hook triggers                                  │
│     echo "$TOOL_OUTPUT" | ctxd scrub -                        │
├────────────────────────────────────────────────────────────────┤
│  3. ctxd sends content to contextd HTTP server                │
│     POST http://localhost:9090/api/v1/scrub                   │
│     { "content": "..." }                                      │
├────────────────────────────────────────────────────────────────┤
│  4. Server scrubs secrets using gitleaks rules                │
│     Returns: { "content": "...", "findings_count": N }        │
├────────────────────────────────────────────────────────────────┤
│  5. Scrubbed content replaces original output                 │
│     Secrets replaced with [REDACTED]                          │
└────────────────────────────────────────────────────────────────┘
```

## Setup

### Step 1: Verify Server Is Running

```bash
ctxd health
```

Expected output:
```
Server Status: ok
Server URL: http://localhost:9090
```

If you get "connection refused", start the contextd server:
```bash
contextd &
```

### Step 2: Configure settings.json Hooks

Add PostToolUse hooks in your Claude Code `settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Read",
        "hooks": ["echo \"$TOOL_OUTPUT\" | ctxd scrub -"]
      },
      {
        "matcher": "Bash",
        "hooks": ["echo \"$TOOL_OUTPUT\" | ctxd scrub -"]
      },
      {
        "matcher": "Grep",
        "hooks": ["echo \"$TOOL_OUTPUT\" | ctxd scrub -"]
      },
      {
        "matcher": "WebFetch",
        "hooks": ["echo \"$TOOL_OUTPUT\" | ctxd scrub -"]
      }
    ]
  }
}
```

**Note:** `$TOOL_OUTPUT` is provided by Claude Code and contains the raw tool output.

### Step 3: Verify ctxd Is In PATH

```bash
which ctxd
```

If not found, add contextd bin directory to PATH or use absolute path in hooks.

## What Gets Scrubbed

The scrubber uses gitleaks detection rules:

| Pattern | Example |
|---------|---------|
| API Keys | `sk-...`, `AKIA...` |
| Passwords | `password=secret123` |
| Tokens | `ghp_...`, `Bearer ...` |
| Private Keys | `-----BEGIN RSA PRIVATE KEY-----` |
| Connection Strings | `postgres://user:pass@host` |
| AWS Credentials | `aws_secret_access_key = ...` |

## Troubleshooting

### "connection refused" Error

**Problem:** ctxd cannot connect to contextd server.

**Solution:**
1. Check server status: `ctxd health`
2. If not running, start it: `contextd &`
3. Verify port: `curl -s http://localhost:9090/health`
4. Check for port conflicts: `lsof -i :9090`

### Hooks Not Triggering

**Problem:** Tool output contains secrets but not scrubbed.

**Solution:**
1. Verify hooks are configured in settings.json
2. Restart Claude Code after settings change
3. Check Claude Code logs for hook errors
4. Verify ctxd is executable: `ctxd --help`

### Slow Tool Responses

**Problem:** Tools take longer after enabling scrub hooks.

**Cause:** Each tool output is sent to the server for scrubbing.

**Mitigations:**
- Only enable scrub hooks for tools that might expose secrets
- Ensure contextd server is local (not remote)
- Consider disabling for large file reads

### Custom Server Port

**Problem:** contextd running on non-default port.

**Solution:** Use `--server` flag:
```json
{
  "hooks": ["echo \"$TOOL_OUTPUT\" | ctxd scrub --server http://localhost:8080 -"]
}
```

## Alternative: Direct API Call

If ctxd CLI is unavailable, use curl with jq to wrap content as JSON:

```json
{
  "hooks": [
    "echo \"$TOOL_OUTPUT\" | jq -Rs '{content: .}' | curl -s -X POST http://localhost:9090/api/v1/scrub -H 'Content-Type: application/json' -d @- | jq -r '.content'"
  ]
}
```

**Note:** Requires jq for JSON encoding and parsing. The `jq -Rs '{content: .}'` wraps the raw output as a proper JSON request body.

## Quick Reference

| Command | Purpose |
|---------|---------|
| `ctxd health` | Check server is running |
| `ctxd scrub FILE` | Scrub a file |
| `ctxd scrub -` | Scrub from stdin |
| `curl localhost:9090/health` | Direct health check |

## Contextd Integration

**If secrets are accidentally exposed:**

1. Record as a remediation for future prevention:
```
remediation_record(
  title: "Secret exposure in [context]",
  problem: "Accidentally read credential file",
  root_cause: "Did not check file type before reading",
  solution: "Use existence checks, verify scrubbing is active",
  category: "security",
  scope: "org",
  tenant_id: "<tenant>"
)
```

2. Search for past incidents:
```
remediation_search(query: "secret exposure credential", tenant_id: "<tenant>")
```

## CRITICAL

**NEVER read .env files or credential files directly.**

Even with scrubbing, raw secrets briefly exist in tool output. Use existence checks instead:

```bash
# CORRECT: Check if file exists
test -f .env && echo "exists"

# CORRECT: Check if var is set
test -n "$API_KEY" && echo "set"

# WRONG: Reads secrets into context
cat .env  # Even with scrubbing, avoid this
```

Scrubbing is a safety net, not permission to read secrets.
