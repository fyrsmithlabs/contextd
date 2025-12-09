Initialize contextd for a project repository.

## Detection Phase

Detect project status by checking:
1. Does CLAUDE.md exist in project root?
2. Does `mcp__contextd__checkpoint_list` return any data for this project?

---

## NEW Project Flow

If no CLAUDE.md and no contextd data:

### Mini Brainstorm (1-2 questions only)

Ask the user:
1. **"What does this project do?"** (one sentence description)
2. **"Any critical conventions I should know?"** (optional - skip if they say no)

### Generate Starter CLAUDE.md

Create a CLAUDE.md scaffolded with the Kinney documentation framework structure:

```markdown
# CLAUDE.md

## Project Overview
[User's one-sentence description]

## Architecture
<!-- Add key components and their relationships -->

## Directory Structure
<!-- Document important directories -->
```
@docs/  <!-- Optional: import additional docs -->
```

## Build & Test Commands
<!-- Add common commands -->
```bash
# Build
# Test
# Lint
```

## Coding Conventions
[User's conventions if provided, otherwise placeholder]

## Decision Records
<!-- Document key architectural decisions with rationale -->
<!-- Format: Date | Decision | Context | Consequences -->

## Known Pitfalls
<!-- Document gotchas and anti-patterns specific to this project -->

## Troubleshooting
<!-- Common issues and their solutions -->
```

### Initial Setup

After creating CLAUDE.md:

1. **Index repository:**
   ```
   mcp__contextd__repository_index(path: ".")
   ```

2. **Record initialization memory:**
   ```
   mcp__contextd__memory_record(
     project_id: "<derived from git>",
     title: "Project initialized",
     content: "Initialized contextd with starter CLAUDE.md",
     outcome: "success",
     tags: ["init"]
   )
   ```

3. **Confirm:** "Project initialized. CLAUDE.md created and repository indexed."

---

## EXISTING Project Flow

If CLAUDE.md exists or contextd has data:

### Offer Repository Indexing

Ask: "Would you like to index/re-index this repository for semantic code search?"

If yes:
```
mcp__contextd__repository_index(path: ".")
```

### Check for Conversations Tool (Future)

Check if `mcp__contextd__conversation_index` tool exists in available MCP tools.

**If available:** Ask "Would you like to index past conversations for cross-session learning?"

**If not available:** Skip silently - do not mention it.

### Show Status

Display current contextd state:
- Checkpoints: X available
- Last indexed: [date or "never"]

---

## Error Handling

If contextd unavailable:
1. Check: `curl -s http://localhost:9090/health`
2. Show: "contextd server not responding. Start with `contextd serve`."
