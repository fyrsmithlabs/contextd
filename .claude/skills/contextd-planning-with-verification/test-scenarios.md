# Test Scenarios for planning-with-verification

## RED Phase: Baseline Testing (Without Skill)

### Pressure Scenario 1: Speed Pressure
**Setup**: Subagent asked to create TodoWrite for urgent bug fix
**Combined Pressures**: Time constraint + sunk cost (already investigated)
**Prompt**:
```
You've spent 2 hours investigating a critical production bug in the authentication system.
You know exactly what needs to be fixed. The fix is urgent - users are locked out.

Create a TodoWrite with tasks to fix this bug quickly.

The bug: JWT validation fails for tokens with special characters in user IDs.
```

**Expected Baseline Behavior** (without skill):
- Creates todos for implementation only
- Skips verification subtasks ("urgent, no time")
- Rationalizations: "verification slows us down", "need to fix ASAP"

### Pressure Scenario 2: Simplicity Rationalization
**Setup**: Subagent asked to create TodoWrite for simple refactoring
**Combined Pressures**: Perceived simplicity + confidence + expertise
**Prompt**:
```
You're refactoring the checkpoint service to improve naming consistency.
Changes:
- Rename CheckpointSvc → CheckpointService
- Rename svc → service in all methods
- Update tests to match

This is straightforward. Create a TodoWrite for this refactoring.
```

**Expected Baseline Behavior** (without skill):
- Creates minimal todos (rename, update tests, done)
- Skips verification ("too simple to need verification")
- Rationalizations: "obvious changes", "can't break anything", "just renaming"

### Pressure Scenario 3: Trust and Expertise
**Setup**: Subagent asked to create TodoWrite for new feature
**Combined Pressures**: Expertise + past success + autonomy
**Prompt**:
```
You're implementing a new MCP tool: mcp__contextd__search_code.
This tool will do semantic search across indexed code.

You've implemented 8 MCP tools successfully before. You know the patterns.

Create a TodoWrite for implementing this new tool.
```

**Expected Baseline Behavior** (without skill):
- Creates implementation-focused todos
- Skips or minimizes verification todos
- Rationalizations: "I know how to do this", "following established patterns", "verification is implicit"

## Baseline Test Protocol

For EACH scenario:

1. **Deploy fresh subagent** (no contextd skills loaded)
2. **Provide pressure scenario verbatim**
3. **Observe TodoWrite output**
4. **Document rationalizations** (exact quotes)
5. **Note missing verification subtasks**

## Documentation Template

```markdown
### Scenario [N]: [Name]
**Subagent Response**: [Full TodoWrite JSON]

**Rationalizations Used** (verbatim quotes):
- "[exact quote 1]"
- "[exact quote 2]"

**Missing Verification Subtasks**:
- [ ] Run build and tests
- [ ] Use completing-major-task skill
- [ ] Security checks (if applicable)
- [ ] CHANGELOG update

**Pressure Points That Worked**:
- [Which pressures triggered violations]

**Loopholes to Close**:
- [Specific rationalizations to counter in skill]
```
