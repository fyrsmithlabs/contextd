# Tier 0: Session Start Injection

**Purpose**: Ultra-concise context for AI agent session initialization
**Token Budget**: ~100 tokens
**Targets**: Claude Code, Grok, Codex, OpenCode, any MCP client

---

## Injection Template

```
# contextd: Shared Knowledge Layer

## Tool Discovery
List `./servers/contextd/` for available tools. Fallback: MCP `tools/list`.
If contextd unavailable: WARN user, ask how to proceed.

## Behavior
1. BEFORE coding: `memory_search` for relevant strategies
2. PREFER `safe_*` tools (bash/read/write) — auto-scrubs secrets via gitleaks
3. On errors: `remediation_search` before retrying
4. LAZY-LOAD only — never pre-fill context

## Tool Priority
memory_* → checkpoint_* → policy_* → skill_* → safe_*

## Hierarchy
Project → Team → Org (search cascades upward)

## Secret Handling
All output SHOULD route through contextd scrubbing. Native tools work but unscrubbed.
```

**Token Count**: ~98

---

## Design Decisions

| Element | Choice | Rationale |
|---------|--------|-----------|
| Language | Imperative | Agents respond to directives, not descriptions |
| Structure | Numbered behaviors | Clear execution order |
| Tool list | Priority-ordered | Agents know what to try first |
| Fallback | Explicit | Handles unavailable contextd gracefully |

---

## Lazy-Load Tiers

| Tier | Location | Tokens | When Loaded |
|------|----------|--------|-------------|
| **0** | This injection | ~100 | Always (session_start) |
| **1** | @../CLAUDE.md | ~300 | Agent needs project context |
| **2** | @./CONTEXTD.md | ~1000+ | Agent needs full briefing |

---

## Maintenance

**Update when:**
- Tool categories change
- Tool priority changes
- Discovery mechanism changes

**After updates:**
1. Verify token count stays ~100
2. Re-optimize with `/lyra:lyra` if structure changes
3. Test across target agents (Claude, Grok, Codex, OpenCode)

**Do NOT update for:**
- Implementation details (update specs instead)
- Adding examples (create separate guide)
