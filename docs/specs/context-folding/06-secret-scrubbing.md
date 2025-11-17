# 6. Secret Scrubbing Integration

[← Back to Branch-Aware Features](05-branch-aware-features.md) | [Next: Git Pre-Fetch →](07-git-prefetch.md)

---

## Overview

Extends existing 5-layer secret defense with **double-scan strategy**: scan during execution + scan on fold.

**Existing Infrastructure**: `pkg/secrets` with Gitleaks SDK, redaction format `[REDACTED:rule-id:preview]`, hierarchical allowlists.

---

## Double-Scan Strategy

**Execution-Time Scanning** (Layer 2):
- Every file operation in branch: file reads, git diffs, search results
- Process: Operation Result → Gitleaks Scan → Redact → Store in Archive

**Fold-Time Scanning** (Defense-in-Depth):
- When `context_return(message)` called
- Scan LLM-generated summary for leaked secrets
- Process: Summary → Gitleaks Scan → Redact → Store in Main Collection

**Why Both?**
- Execution catches secrets in raw data
- Fold catches secrets in LLM summaries
- Example: LLM writes "Found issue: API key sk_live_abc123" → fold scan catches it

---

## NATS Secret Events

**Subject**: `secrets.detected.{owner_id}.{project_hash}.{branch_id}`

**Payload**:
```json
{
  "branch_id": "br_abc123",
  "operation": "file_read",
  "file_path": "config/.env",
  "redactions": [
    {
      "rule_id": "github-pat",
      "match_preview": "ghp_abc1...",
      "redacted_as": "[REDACTED:github-pat:ghp_]"
    }
  ]
}
```

---

## Archive vs Main Storage

**Archive** (encrypted, full detail):
```json
{
  "branch_id": "br_abc123",
  "operations": [
    {
      "content_unredacted": "API_KEY=sk_live_abc123",  // Encrypted AES-256-GCM
      "content_redacted": "API_KEY=[REDACTED:api-key:sk_l]",
      "secrets_found": 1
    }
  ]
}
```

**Main** (clean, folded):
```json
{
  "checkpoint_id": "ckpt_abc123",
  "summary": "Fixed auth: API key [REDACTED:api-key:sk_l] was expired",
  "secrets_scrubbed": 1,
  "clean": true
}
```

---

## MCP Response Format

All tools return scan metadata:
```json
{
  "result": {...},
  "security": {
    "secrets_scanned": true,
    "secrets_found": 2,
    "redactions_applied": 2,
    "clean": true
  }
}
```

---

## Configuration

```yaml
security:
  secret_scrubbing:
    # Always enabled, no toggle
    
    gitleaks:
      config_path": ".gitleaks.toml"
      user_allowlist: "~/.config/contextd/allowlist.toml"
      
    double_scan:
      execution_time: true
      fold_time: true
      
    archive:
      store_unredacted: true
      encryption: "aes-256-gcm"
```

---

[← Back: Branch-Aware Features](05-branch-aware-features.md) | [Next: Git Pre-Fetch →](07-git-prefetch.md)
