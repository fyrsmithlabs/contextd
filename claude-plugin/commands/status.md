Show contextd status for current session and project.

Gather and display:

1. **Session Info**
   - Tenant ID (from git remote)
   - Project path
   - Current session identifier

2. **Memories** (call `mcp__contextd__memory_search` with broad query)
   - Count of memories for this project
   - Recent memories recorded this session
   - Top memories by confidence

3. **Checkpoints** (call `mcp__contextd__checkpoint_list`)
   - Available checkpoints for this project
   - Most recent checkpoint summary

4. **Remediations**
   - Recent remediations used or recorded
   - Count by category

Format as a clean status report showing what contextd knows about this project.

## Error Handling

If any MCP call fails, show partial results with status indicators:

```
## contextd Status

**Session Info**
- Tenant: fyrsmithlabs
- Project: /home/user/projects/contextd
- Session: sess_abc123

**Memories**: ✅ 12 found (3 high confidence)
**Checkpoints**: ✅ 4 available
**Remediations**: ❌ Could not fetch (server error)
```

If all calls fail:
1. Check server: `curl -s http://localhost:9090/health`
2. Display: "contextd server not responding. Status unavailable."
3. Suggest: "Start contextd with `contextd serve` or verify MCP configuration."
