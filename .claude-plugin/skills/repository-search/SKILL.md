---
name: repository-search
description: Use when searching codebase semantically - indexes repositories for meaning-based search that finds code by concept rather than exact keywords
---

# Repository Search

## Overview

Semantic code search finds code by meaning, not just keywords. Index once, search by concept.

## When to Use

**ALWAYS use semantic_search FIRST for code lookups:**
- "Where is authentication handled?"
- "Find error handling patterns"
- "How does the API validate input?"

**Use Read/Grep instead when:**
- You know the exact file path
- You need exact string matching

## Tools

### semantic_search (PREFERRED)

Smart search that auto-selects the best method:
```json
{
  "query": "user authentication validation",
  "project_path": "/path/to/repo",
  "limit": 10
}
```
- Uses semantic search if repository is indexed
- Falls back to grep automatically if not indexed
- No manual indexing required

### repository_index

Index a repository for semantic search (optional - enhances semantic_search):
```json
{
  "path": "/path/to/repo",
  "include_patterns": ["*.go", "*.ts", "*.py"],
  "exclude_patterns": ["vendor/**", "node_modules/**"],
  "max_file_size": 1048576
}
```

### repository_search

Direct semantic search (requires prior indexing):
```json
{
  "query": "user authentication validation",
  "project_path": "/path/to/repo",
  "limit": 10
}
```

## When to Re-index

- After `git commit` - captures code changes
- After `git checkout` - updates branch context
- After pulling changes - ensures search is current

## Query Writing Tips

| Instead of | Write |
|------------|-------|
| "auth" | "user authentication and login handling" |
| "err" | "error handling and validation" |
| "func main" | "application entry point and initialization" |

Semantic search understands concepts - be descriptive, not literal.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Searching before indexing | Run `repository_index` first |
| Using grep-style patterns | Use natural language descriptions |
| Not re-indexing after commits | Index after each commit |
| Vague single-word queries | Be specific: "database connection pooling" |
