Initialize contextd for a NEW project repository.

**Use `/onboard` instead for existing projects with code but no CLAUDE.md.**

## Detection Phase

Check project status:
1. Does CLAUDE.md exist in project root?
2. Does `mcp__contextd__checkpoint_list` return any data for this project?

**If CLAUDE.md exists or has contextd data:** Inform user to use `/onboard` instead.

---

## New Project Flow

### Mini Brainstorm (1-2 questions only)

Ask the user:
1. **"What does this project do?"** (one sentence description)
2. **"Any critical conventions I should know?"** (optional - skip if they say no)

### Generate Starter CLAUDE.md

Use the `writing-claude-md` skill to create a scaffolded CLAUDE.md:

```markdown
# CLAUDE.md - [Project Name]

**Status**: Active Development
**Last Updated**: [Today's Date]

---

## Critical Rules

**ALWAYS** [placeholder - user fills in]
**NEVER** [placeholder - user fills in]

---

## Project Overview
[User's one-sentence description]

## Architecture
<!-- Add key components and their relationships -->

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| | | |

## Commands

| Command | Purpose |
|---------|---------|
| | |

## Code Standards
[User's conventions if provided, otherwise placeholder]

## Known Pitfalls
<!-- Document gotchas as you discover them -->

## ADRs (Architectural Decisions)
<!-- Format: ADR-NNN: Title, Status, Context, Decision, Consequences -->
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
     project_id: "<derived from git remote or directory>",
     title: "Project initialized",
     content: "Initialized new project with starter CLAUDE.md",
     outcome: "success",
     tags: ["init", "new-project"]
   )
   ```

3. **Confirm:** "Project initialized. CLAUDE.md created and repository indexed."

---

## Error Handling

If contextd unavailable:
1. Check server: `curl -s http://localhost:9090/health`
   Expected: `{"status":"ok"}`
   If different or no response: contextd is not running
2. Show: "contextd server not responding. Start with `contextd serve`."
