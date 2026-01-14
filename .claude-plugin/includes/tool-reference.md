## Contextd MCP Tools Reference

### Memory Tools (ReasoningBank)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `memory_search` | Find past learnings | `project_id`, `query`, `limit` |
| `memory_record` | Save new learning | `project_id`, `title`, `content`, `outcome`, `tags` |
| `memory_feedback` | Rate helpfulness | `memory_id`, `helpful` |
| `memory_outcome` | Report task result | `memory_id`, `succeeded` |
| `memory_consolidate` | Merge similar memories | `project_id`, `similarity_threshold` |

### Remediation Tools

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `remediation_search` | Find past fixes | `query`, `project_path`, `scope`, `include_hierarchy` |
| `remediation_record` | Save fix pattern | `title`, `problem`, `root_cause`, `solution`, `category`, `scope` |
| `troubleshoot_diagnose` | AI error diagnosis | `error_message`, `error_context` |

### Search Tools

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `semantic_search` | Smart code search (auto-fallback) | `query`, `project_path` |
| `repository_index` | Index repo for search | `path`, `branch`, `include_patterns` |
| `repository_search` | Direct semantic search | `query`, `project_path` or `collection_name` |

### Checkpoint Tools

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `checkpoint_save` | Save context state | `session_id`, `project_path`, `name`, `description`, `summary` |
| `checkpoint_list` | View checkpoints | `project_path`, `session_id`, `limit` |
| `checkpoint_resume` | Restore state | `checkpoint_id`, `tenant_id`, `level` |

### Context Folding Tools

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `branch_create` | Isolated sub-task | `session_id`, `description`, `prompt`, `budget` |
| `branch_return` | Return with results | `branch_id`, `message` |
| `branch_status` | Check budget/state | `branch_id` or `session_id` |

### Reflection Tools

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `reflect_analyze` | Pattern analysis | `project_id`, `min_confidence`, `max_patterns` |
| `reflect_report` | Self-reflection report | `project_id`, `period_days`, `format` |

---

### When to Use Each Tool

| Situation | Tools to Use |
|-----------|--------------|
| Starting any task | `semantic_search` then `memory_search` |
| Encountering error | `troubleshoot_diagnose` then `remediation_search` |
| After fixing error | `remediation_record` |
| After completing work | `memory_record` |
| Context at 70%+ | `checkpoint_save` |
| Complex sub-task | `branch_create` -> work -> `branch_return` |
| Resuming work | `checkpoint_list` then `checkpoint_resume` |
| Improving over time | `reflect_analyze` and `reflect_report` |

### Key Identifiers

| ID | Source | Example |
|----|--------|---------|
| `tenant_id` | Git remote org/owner | `fyrsmithlabs` |
| `project_id` | Repository name | `contextd` |
| `session_id` | Claude session | Auto-provided by hooks |

### Quick Reference

**Pre-flight:**
```
semantic_search(query, project_path)
memory_search(project_id, query)
```

**On errors:**
```
troubleshoot_diagnose(error_message, error_context)
remediation_search(query, project_path)
```

**Post-flight:**
```
memory_record(project_id, title, content, outcome, tags)
remediation_record(...)  # if error fixed
```

**For sub-tasks:**
```
branch_create(session_id, description, prompt, budget)
branch_return(branch_id, message)
```
