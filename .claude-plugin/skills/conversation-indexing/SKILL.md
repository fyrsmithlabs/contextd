---
name: conversation-indexing
description: Use when onboarding a project to extract learnings from past Claude Code conversations. Indexes JSONL history to pre-warm contextd with remediations, memories, and policies.
---

# Conversation Indexing

## Overview

Index past Claude Code conversations from `~/.claude/projects/` to extract:
- **Remediations** - Error → fix patterns
- **Memories** - Learnings and decisions
- **Policies** - Behavioral rules extracted from corrections

## Conversation Format

Claude Code stores conversations as JSONL files:

```
~/.claude/projects/{project-path-encoded}/
├── {uuid}.jsonl           # Session conversations
└── agent-{id}.jsonl       # Agent sub-conversations
```

Each line is a JSON message with:
- `type`: `user`, `assistant`, `file-history-snapshot`, `summary`, `system`
- `message`: Content object with `role` and `content`
- `timestamp`: When the message was sent

## Extraction Flow

```
┌──────────────────────────────────────────────────────────────────┐
│  0. CONSENT (REQUIRED)                                           │
│     Show user exactly what will be indexed, get YES confirmation │
├──────────────────────────────────────────────────────────────────┤
│  1. SCAN                                                         │
│     Find JSONL files in ~/.claude/projects/{project}/            │
├──────────────────────────────────────────────────────────────────┤
│  2. VALIDATE PATHS (SECURITY GATE)                               │
│     Verify each file is under allowed base, no path traversal    │
│     ABORT if validation fails (see Path Validation section)      │
├──────────────────────────────────────────────────────────────────┤
│  3. SCRUB (SECURITY GATE - VERIFY SUCCESS)                       │
│     POST to /api/v1/scrub, ABORT if scrubbing fails              │
│     Set scrubbing_verified=true only on success                  │
├──────────────────────────────────────────────────────────────────┤
│  4. EXTRACT                                                      │
│     Identify patterns using heuristics + LLM                     │
│     ONLY proceed if scrubbing_verified=true                      │
├──────────────────────────────────────────────────────────────────┤
│  5. DEDUPLICATE                                                  │
│     Similarity > 0.85 = merge, 0.6-0.85 = review, <0.6 = new     │
├──────────────────────────────────────────────────────────────────┤
│  6. STORE                                                        │
│     Save to contextd (memory_record, remediation_record)         │
└──────────────────────────────────────────────────────────────────┘
```

**Security Gates**: Steps 2 and 3 are mandatory security gates. Failure at either gate ABORTS the entire operation for that file. No exceptions.

## Consent Protocol (REQUIRED)

Before indexing ANY conversations, display this prompt and wait for explicit YES:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ CONVERSATION INDEXING CONSENT                                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ I will index the following conversations for project: {project_name}        │
│                                                                             │
│ Files to be indexed:                                                        │
│   - abc123.jsonl (2024-12-15, ~50KB)                                        │
│   - def456.jsonl (2024-12-18, ~120KB)                                       │
│   - agent-xyz789.jsonl (2024-12-19, ~30KB)                                  │
│                                                                             │
│ What will be extracted:                                                     │
│   - Error → fix patterns (remediations)                                     │
│   - Learnings and insights (memories)                                       │
│   - User corrections (policies)                                             │
│                                                                             │
│ Security measures:                                                          │
│   - All content scrubbed for secrets before processing                      │
│   - Only project-specific conversations indexed                             │
│                                                                             │
│ Do you want to proceed? [YES/NO]: _                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

**CRITICAL**: Do NOT proceed without explicit "YES" from user.

## Extraction Patterns

### Remediations (Error → Fix)

Look for error-fix pairs:

```
# Indicators of error
"error", "Error:", "failed", "exception", "TypeError", "undefined"
"command failed", "exit code", "stack trace"

# Indicators of fix
"fixed by", "the fix was", "solved by", "resolved"
"changed X to Y", "needed to", "the issue was"
```

Extract:
- `problem`: The error message/symptom
- `root_cause`: What was wrong
- `solution`: How it was fixed
- `category`: Error type (build, runtime, config, etc.)

### Memories (Learnings)

Look for explicit learnings:

```
# User teaching agent
"remember that", "always do X", "never do Y", "from now on"
"you should have", "next time", "lesson learned"

# Agent realizing
"I learned", "I discovered", "key insight", "important note"
"this works because", "the trick is"
```

Extract:
- `title`: Brief description
- `content`: The learning with context
- `outcome`: success or failure
- `tags`: Categorization

### Policies (Behavioral Rules)

Look for correction patterns:

```
# User corrections
"why did you", "you should have", "don't ever", "always X first"
"I told you to", "that's wrong because", "you forgot to"

# Strong imperatives
"MUST", "NEVER", "ALWAYS", "REQUIRED"
```

Extract:
- `name`: Short identifier
- `rule`: The MUST statement
- `description`: Why this rule exists
- `category`: verification, process, security, quality
- `severity`: critical, high, medium

## Context Cost Warning

Indexing conversations uses significant context:

```
⚠️  WARNING: Online conversation indexing will use significant context.
    Estimated: ~50k tokens per conversation

    Found: 15 conversations to index
    Estimated total: ~750k tokens

    Options:
    [1] Continue with context folding (recommended for <10 conversations)
    [2] Switch to batch mode (process offline, no context cost)
    [3] Index specific conversations only
    [4] Cancel

    Choice: _
```

### Context Folding Mode

For small batches, use context folding to process in-session:

1. Create branch for conversation batch
2. Process conversations one at a time
3. Extract findings
4. Return branch with findings
5. Store in contextd

### Batch Mode (Recommended for Large Projects)

For large projects, process offline:

```bash
ctxd onboard --conversations --batch
```

This runs outside the agent context, no token cost.

## Storage Patterns

### Remediation

```json
{
  "title": "ENOENT when reading file",
  "problem": "Error: ENOENT: no such file or directory, open 'config.json'",
  "root_cause": "File path was relative, needed absolute",
  "solution": "Use path.resolve() to get absolute path before reading",
  "category": "runtime",
  "tenant_id": "user",
  "scope": "project",
  "tags": ["nodejs", "filesystem", "paths"]
}
```

### Memory

```json
{
  "project_id": "myproject",
  "title": "Use path.resolve for file operations",
  "content": "Always use path.resolve() for file operations to ensure absolute paths. Relative paths break when working directory changes.",
  "outcome": "success",
  "tags": ["nodejs", "best-practice", "extracted"]
}
```

### Policy

```json
{
  "project_id": "global",
  "title": "POLICY: verify-file-exists",
  "content": "RULE: Check if file exists before reading.\nDESCRIPTION: User corrected agent for assuming file existed.\nCATEGORY: verification\nSEVERITY: medium\nSCOPE: global\nSOURCE: conversation:{uuid}:turn:47",
  "outcome": "success",
  "tags": ["type:policy", "category:verification", "severity:medium", "scope:global", "enabled:true"]
}
```

## Secret Scrubbing (MANDATORY - FAIL CLOSED)

Before processing ANY conversation content, scrub secrets. **This is a security-critical step.**

### Scrubbing Protocol

```
# Step 1: Read raw content
raw_content = Read(conversation_file)

# Step 2: Call scrub API with error handling
try:
  response = POST http://localhost:9090/api/v1/scrub
    Body: {"content": raw_content}
    Timeout: 30s
catch NetworkError:
  ERROR: "Scrubbing service unreachable - ABORTING"
  LOG: "Cannot proceed without secret scrubbing. Ensure contextd HTTP server is running."
  return ABORT

# Step 3: VERIFY HTTP SUCCESS
if response.status != 200:
  ERROR: "Scrubbing failed with status {response.status} - ABORTING"
  LOG: "Response: {response.body}"
  return ABORT

# Step 4: VERIFY RESPONSE STRUCTURE
if response.body.scrubbed_content is undefined:
  ERROR: "Invalid scrub response - missing scrubbed_content field - ABORTING"
  return ABORT

# Step 5: SET VERIFICATION FLAG
scrubbing_verified = true
scrubbed_content = response.body.scrubbed_content

# Step 6: Log scrubbing results
if response.body.secrets_found > 0:
  LOG: "⚠️  Scrubbed {secrets_found} secrets from conversation"
  LOG: "Secret types: {response.body.secret_types}"
else:
  LOG: "✓ No secrets detected"
```

### Critical Verification Check

**Before ANY extraction or storage, verify scrubbing succeeded:**

```
# GATE: This check MUST pass before proceeding to extraction
if NOT scrubbing_verified:
  ERROR: "SECURITY VIOLATION: Attempting to process unscrubbed content"
  return ABORT

# ONLY after this gate can you proceed
proceed_with_extraction(scrubbed_content)
```

### Failure Modes

| Failure | Action | Recovery |
|---------|--------|----------|
| HTTP server not running | ABORT | Start contextd with `--http` flag |
| Network timeout | ABORT | Retry once, then abort |
| Non-200 response | ABORT | Check server logs |
| Missing response fields | ABORT | Update contextd version |
| Scrubbing exception | ABORT | Report bug |

**NEVER process unscrubbed content. NEVER catch and ignore scrubbing errors. ALWAYS fail closed.**

## Deduplication

When extracting, check for similar existing entries using these thresholds:

| Similarity Score | Action |
|------------------|--------|
| > 0.85 | **Merge**: Same pattern, boost confidence via `memory_feedback(helpful=true)` |
| 0.60 - 0.85 | **Review**: Show user, ask if duplicate or new |
| < 0.60 | **Create New**: Distinct enough to be separate entry |

```
# Step 1: Search for similar entries
existing = mcp__contextd__memory_search(
  project_id: "{project}",
  query: "{extracted learning summary}",
  limit: 3
)

# Step 2: Check similarity (use first result's score)
for result in existing:
  similarity = result.score  # 0.0 to 1.0

  if similarity > 0.85:
    # Same pattern - boost confidence
    mcp__contextd__memory_feedback(memory_id: result.id, helpful: true)
    skip_creation = true
    break

  elif similarity >= 0.60:
    # Ambiguous - ask user
    ask_user("Found similar entry: '{result.title}'. Is this a duplicate? [Y/N]")

  # else: < 0.60, create new entry
```

Similar check for remediations:
```
existing = mcp__contextd__remediation_search(
  query: "{error pattern}",
  tenant_id: "user",
  limit: 3
)
```

## Index Tracking

Track indexed files to avoid re-processing:

```json
{
  "file_path": "~/.claude/projects/-home-user-projects-myproject/{uuid}.jsonl",
  "sha256": "abc123...",
  "indexed_at": "2024-12-19T10:00:00Z",
  "extractions": {
    "remediations": 5,
    "memories": 12,
    "policies": 3
  }
}
```

Store as memory with tag `type:index-record`.

On re-run:
1. Hash file
2. Check if hash matches stored record
3. Skip if unchanged, re-index if modified

## Security Considerations

1. **Secret Scrubbing**: All content scrubbed with gitleaks before processing (see Scrubbing Protocol above)
2. **Path Validation**: Prevent path traversal attacks (see Path Validation below)
3. **Consent**: User must explicitly request conversation indexing with "YES"
4. **Scope**: Only index conversations for specific project (not all)

## Path Validation (REQUIRED)

Before processing ANY conversation file, validate the path to prevent path traversal attacks.

### Path Validation Protocol

```
# Step 1: Define allowed base directory
ALLOWED_BASE = os.path.expanduser("~/.claude/projects/")

# Step 2: Resolve the target path (follows symlinks, normalizes)
target_path = os.path.realpath(conversation_file)

# Step 3: Validate path is under allowed base
if NOT target_path.startswith(ALLOWED_BASE):
  ERROR: "SECURITY VIOLATION: Path traversal attempt detected"
  LOG: "Attempted path: {conversation_file}"
  LOG: "Resolved to: {target_path}"
  LOG: "Not under allowed base: {ALLOWED_BASE}"
  return ABORT

# Step 4: Additional checks
if ".." in conversation_file:
  ERROR: "SECURITY VIOLATION: Path contains '..' component"
  return ABORT

if conversation_file.startswith("/") and NOT conversation_file.startswith(ALLOWED_BASE):
  ERROR: "SECURITY VIOLATION: Absolute path outside allowed directory"
  return ABORT
```

### Valid Path Examples

```
# ✅ Valid
~/.claude/projects/-home-user-myproject/abc123.jsonl
~/.claude/projects/-home-user-myproject/agent-xyz.jsonl

# ❌ Invalid - Path traversal
~/.claude/projects/../../../etc/passwd
~/.claude/projects/-home-user-myproject/../../../secret.txt

# ❌ Invalid - Symlink escape
~/.claude/projects/symlink-to-root/etc/passwd

# ❌ Invalid - Absolute path outside base
/etc/passwd
/home/user/.ssh/id_rsa
```

### Implementation in Go

```go
import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

func validateConversationPath(conversationFile string) error {
    // Get home directory (more reliable than os.Getenv)
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return fmt.Errorf("cannot determine home directory: %w", err)
    }

    // Build and resolve allowed base path
    allowedBase := filepath.Join(homeDir, ".claude", "projects")
    allowedBase, err = filepath.Abs(allowedBase)
    if err != nil {
        return fmt.Errorf("cannot resolve allowed base: %w", err)
    }

    // Clean and make target path absolute
    targetPath, err := filepath.Abs(filepath.Clean(conversationFile))
    if err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }

    // Additional safety: reject paths with ".." in original input
    if strings.Contains(conversationFile, "..") {
        return fmt.Errorf("path contains '..' component: %s", conversationFile)
    }

    // Resolve symlinks if file exists, otherwise use cleaned path
    // NOTE: EvalSymlinks fails for non-existent files, which is OK
    realTarget := targetPath
    if resolved, err := filepath.EvalSymlinks(targetPath); err == nil {
        realTarget = resolved
    }

    // Resolve symlinks on base path too for accurate comparison
    realBase := allowedBase
    if resolved, err := filepath.EvalSymlinks(allowedBase); err == nil {
        realBase = resolved
    }

    // Check if under allowed base using filepath.Rel
    relPath, err := filepath.Rel(realBase, realTarget)
    if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
        return fmt.Errorf("path traversal: %s is outside %s", realTarget, realBase)
    }

    return nil
}
```

### Validation Gate

**This check MUST pass before reading any file:**

```
for conversation_file in files_to_index:
  # GATE: Validate path BEFORE reading
  if NOT validate_path(conversation_file):
    ERROR: "Skipping invalid path: {conversation_file}"
    continue

  # ONLY after validation can you read the file
  content = Read(conversation_file)
```

### Path Validation Failure Modes

| Failure | Action | Recovery |
|---------|--------|----------|
| Path contains `..` | Skip file | Check file path construction |
| Symlink outside allowed base | Skip file | Remove malicious symlink |
| Cannot determine home directory | ABORT all | Check environment |
| Absolute path outside base | Skip file | Use project-relative paths |
| Cannot resolve path | Skip file | Check file permissions |

## Usage

Conversation indexing is triggered via `/onboard --conversations`:

```bash
# Index current project's conversations
/onboard --conversations

# Index with batch mode (offline processing)
/onboard --conversations --batch

# Index specific conversation
/onboard --conversations --file={uuid}.jsonl
```

See `/contextd:onboard` for full onboarding workflow.

## Quick Reference

| Pattern | Extraction Type | Storage |
|---------|-----------------|---------|
| Error + Fix | Remediation | `remediation_record` |
| Explicit learning | Memory | `memory_record` |
| User correction | Policy | `memory_record` with policy tags |
| Decision with rationale | Memory (ADR) | `memory_record` with adr tag |
