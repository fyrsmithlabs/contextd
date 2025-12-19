Manage agent policies - STRICT guardrails that MUST be followed at all times.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| (none) | List all enabled policies with stats |
| `add` | Interactively add a new policy |
| `remove {name}` | Disable a policy (preserves history) |
| `stats` | Detailed violation/success statistics |
| `init` | Initialize recommended built-in policies |

## Examples

```bash
/policies              # List all policies
/policies add          # Add new policy interactively
/policies remove test-before-fix  # Disable a policy
/policies stats        # Show detailed statistics
/policies init         # Add recommended policies
```

## Flow

### List Policies (Default)

Search for all enabled policies:

```
memory_search(project_id: "global", query: "type:policy enabled:true", limit: 50)
```

Display as table:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ AGENT POLICIES                                                              │
├──────────────────────┬───────────┬──────────┬──────────┬────────────────────┤
│ Name                 │ Category  │ Severity │ V/S Ratio│ Rule (truncated)   │
├──────────────────────┼───────────┼──────────┼──────────┼────────────────────┤
│ test-before-fix      │ verify    │ high     │ 3/47     │ Run tests before...│
│ contextd-first       │ process   │ high     │ 2/89     │ Search contextd... │
│ no-secrets-in-context│ security  │ critical │ 0/156    │ Never read secr... │
│ consensus-binary     │ quality   │ medium   │ 1/12     │ APPROVE or REQUE...│
└──────────────────────┴───────────┴──────────┴──────────┴────────────────────┘

4 policies enabled. Run `/policies stats` for detailed statistics.
```

### Add Policy

Interactive prompts:

1. **Name**: Short identifier (lowercase, hyphens)
   ```
   Enter policy name (e.g., test-before-fix): _
   ```

2. **Rule**: The MUST statement
   ```
   Enter the rule (what MUST happen): _
   Example: "Always run tests before claiming a fix is complete"
   ```

3. **Description**: Why this policy exists
   ```
   Why does this policy exist? _
   Example: "Prevents false claims of completion. Tests verify the fix works."
   ```

4. **Category**: Select from list
   ```
   Select category:
   [1] verification - Testing, validation, confirmation
   [2] process      - Workflow steps, ordering
   [3] security     - Safety, permissions, credentials
   [4] quality      - Code standards, best practices
   [5] communication- User interaction
   ```

5. **Severity**: Select from list
   ```
   Select severity:
   [1] critical - Security, data loss potential
   [2] high     - Process integrity, correctness
   [3] medium   - Quality, maintainability
   ```

6. **Scope**: Select from list
   ```
   Select scope:
   [1] global              - Applies to all sessions
   [2] skill:{skill_name}  - Applies when skill is active
   [3] project:{path}      - Applies to specific project
   ```

After confirmation:
```
✓ Policy 'test-before-fix' created.
```

See `policies` skill for storage implementation details.

### Remove Policy

Disable rather than delete (preserves violation history):

1. Search for policy by name
2. Update memory with `enabled:false` tag
3. Confirm removal

```
Policy 'test-before-fix' disabled. History preserved.
Use `/policies stats --all` to see disabled policies.
```

### Stats

Detailed statistics for all policies:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ POLICY STATISTICS                                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ SUMMARY                                                                     │
│ ───────                                                                     │
│ Total Policies: 4 enabled, 1 disabled                                       │
│ Overall Compliance: 96.7% (295 successes / 305 evaluations)                 │
│ Total Violations: 10                                                        │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ BY CATEGORY                                                                 │
│ ───────────                                                                 │
│ verification: 2 policies, 98.0% compliance                                  │
│ process:      1 policy,  97.8% compliance                                   │
│ security:     1 policy,  100% compliance                                    │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ TOP VIOLATORS                                                               │
│ ─────────────                                                               │
│ 1. test-before-fix     (3 violations) - Last: 2 days ago                    │
│ 2. contextd-first      (2 violations) - Last: 5 days ago                    │
│ 3. consensus-binary    (1 violation)  - Last: 7 days ago                    │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ PERFECT COMPLIANCE (0 violations)                                           │
│ ─────────────────────────────────                                           │
│ • no-secrets-in-context (156 successes)                                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Init

Initialize recommended built-in policies:

```
Initializing recommended policies...

[1/6] no-secrets-in-context (critical/security)
      Rule: Never read secrets (.env, credentials) into context
      ✓ Created

[2/6] no-force-push-main (critical/security)
      Rule: Never force push to main/master branch
      ✓ Created

[3/6] test-before-fix (high/verification)
      Rule: Always run tests before claiming a fix is complete
      ✓ Created

[4/6] contextd-first (high/process)
      Rule: Search contextd before filesystem search
      ✓ Created

[5/6] verify-before-complete (high/verification)
      Rule: Run verification commands before marking task complete
      ✓ Created

[6/6] consensus-binary (medium/quality)
      Rule: Consensus reviews must be APPROVE or REQUEST CHANGES only
      ✓ Created

6 policies initialized. Run `/policies` to view.
```

If policies already exist, skip them:
```
[1/6] no-secrets-in-context (critical/security)
      ⏭ Already exists, skipping
```

## Error Handling

@_error-handling.md

**Policy-specific errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| Policy not found | Name doesn't match any policy | Check `/policies` for correct name |
| Policy already exists | Trying to add duplicate name | Use different name or remove existing |
| Invalid category | Category not in allowed list | Use: verification, process, security, quality, communication |
| Invalid severity | Severity not in allowed list | Use: critical, high, medium |

## Integration

Policies are checked:
- **At skill load**: Applicable policies injected into context
- **During /reflect**: Compliance evaluated against recent actions
- **By /onboard**: New policies extracted from conversation patterns

## Related

- `policies` skill - Full documentation on policy system
- `/reflect` - Evaluates policy compliance
- `/onboard` - Extracts policies from conversations
