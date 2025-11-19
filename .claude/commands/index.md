---
name: index
description: Index repository for semantic search
---

# Index Repository

Call the contextd MCP `index_repository` tool.

**Workflow:**
1. Get current directory: `pwd`
2. Call MCP tool: `index_repository`
   - `project_path`: current directory
   - `force`: false (optional)
3. Report: operation_id, status

**Note**: Indexing runs async in background.

ARGUMENTS: {path}
