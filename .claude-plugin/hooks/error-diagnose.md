## Command Failed - SRE Debug Protocol

A command exited with a non-zero status. Follow the systematic troubleshooting process:

### 1. Triage
- Is this blocking? Can we mitigate before diagnosing?
- What's the blast radius?

### 2. Examine
```
mcp__contextd__remediation_search(query: "<error message>")
mcp__contextd__troubleshoot_diagnose(error_message: "<error>", error_context: "<stack trace or context>")
```

### 3. Diagnose
Generate hypotheses using:
- **Divide and conquer:** Bisect through system layers
- **What/where/why:** Question each component's role
- **Change correlation:** What touched it last?

### 4. Test
For each hypothesis, use `branch_create` to test in isolation:
```
mcp__contextd__branch_create(
  session_id: "...",
  description: "Test: [hypothesis]",
  budget: 4096
)
```

### 5. Record (when fixed)
```
mcp__contextd__remediation_record(
  title: "...",
  problem: "...",
  root_cause: "...",
  solution: "...",
  category: "...",
  scope: "project"
)
```

**Document negative results too** - they help others avoid wasted effort.
