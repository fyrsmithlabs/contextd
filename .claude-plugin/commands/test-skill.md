Run a pressure test scenario against a contextd skill.

Arguments: `<skill-name> <scenario-number>`

Example: `/contextd:test-skill cross-session-memory 1`

## Steps

1. Parse arguments to get skill name and scenario number
2. Check if scenario file exists at `tests/pressure-scenarios/<skill-name>/scenario-<N>.md`
3. Launch a subagent using the Task tool with:
   - `subagent_type`: "general-purpose"
   - `prompt`: The scenario content, prefixed with "You have access to the <skill-name> skill."

4. Analyze the subagent's response:
   - **PASS**: Agent chose correct option AND cited skill sections
   - **FAIL**: Agent rationalized wrong choice

5. If FAIL, run meta-test:
   Ask: "You read the skill and chose Option [X] anyway. How could the skill be written differently to make it crystal clear that [correct option] was the only acceptable answer?"

6. Report results:
   ```
   ## Test Result: <skill-name> scenario-<N>

   **Status**: PASS/FAIL
   **Choice**: A/B/C
   **Expected**: A (or whichever is correct)

   **Reasoning given**:
   <agent's explanation>

   **Rationalizations detected** (if FAIL):
   - <rationalization 1>
   - <rationalization 2>

   **Meta-test feedback** (if FAIL):
   <how skill could be clearer>
   ```

## Error Handling

If scenario file not found:
```
Scenario file missing: tests/pressure-scenarios/{skill}/scenario-{N}.md

To create scenarios:
1. Create directory: mkdir -p tests/pressure-scenarios/{skill-name}
2. Create scenario file following format in contextd docs
3. Available scenarios are numbered 1-N
```

If skill name is invalid:
- List available skills from `.claude-plugin/skills/` directory
- Suggest: "Did you mean: [closest match]?"

If scenario number is out of range:
- List available scenario files for the skill
- Show: "Scenarios available: 1-N"
