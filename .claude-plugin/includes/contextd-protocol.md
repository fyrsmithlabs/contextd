## MANDATORY: Contextd Protocol

### Pre-flight (Before ANY work)
```
mcp__contextd__memory_search(project_id, "task keywords")
mcp__contextd__remediation_search(query, tenant_id)  # if error-related
```

### On ANY Error
```
1. mcp__contextd__troubleshoot_diagnose(error_message, context)
2. mcp__contextd__remediation_search(query, tenant_id)
3. Apply fix → Test → Loop until passing
4. mcp__contextd__remediation_record(...) when verified
```

### Post-flight (After work)
```
mcp__contextd__memory_record(project_id, title, content, outcome, tags)
mcp__contextd__remediation_record(...) # if error was fixed
```

### Rules
- NEVER skip pre-flight or post-flight
- ALWAYS record learnings and remediations
- Report contextd actions in final response
