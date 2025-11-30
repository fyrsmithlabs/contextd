# contextd: ReasoningBank-Based Context Management System

**Version**: 1.0.0-draft
**Date**: 2025-11-22
**Status**: Design Document

## Executive Summary

contextd is an MCP-compatible context management system that reduces AI agent context usage over time while building institutional developer knowledge. It combines three key innovations:

1. **Context-Folding** (intra-session): Active context management via `branch()`/`return()` tools
2. **ReasoningBank** (cross-session): Distilled strategies from successes AND failures
3. **Institutional Knowledge** (cross-org): Consolidated patterns across teams and projects

The system uses Qdrant as the vector store with a hierarchical collection architecture supporting multi-tenant SaaS deployments.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Three-Layer Model](#three-layer-model)
3. [Collection Architecture](#collection-architecture)
4. [Data Flow](#data-flow)
5. [MCP Interface](#mcp-interface)
6. [Multi-Tenancy & RBAC](#multi-tenancy--rbac)
7. [Consolidation & Deduplication](#consolidation--deduplication)
8. [Deployment Models](#deployment-models)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              AI Agents                                  │
│         (Claude, GPT, Gemini, or any MCP-compatible agent)              │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   │ MCP Protocol
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         contextd MCP Server                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   Context   │  │  Memory     │  │  Standards  │  │   Session   │    │
│  │   Folding   │  │  Manager    │  │  Engine     │  │   Manager   │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   RBAC      │  │  Distiller  │  │ Consolidator│  │   Indexer   │    │
│  │   Layer     │  │  (async)    │  │  (async)    │  │   (async)   │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Qdrant Vector Store                             │
│                                                                         │
│   Per-Org Collections:                                                  │
│   ├── {org}_memories, {org}_{team}_memories, {org}_{team}_{proj}_*      │
│   ├── {org}_remediations, {org}_{team}_remediations, ...                │
│   ├── {org}_policies, {org}_coding_standards, {org}_repo_standards      │
│   ├── {org}_skills, {org}_agents                                        │
│   ├── {org}_{team}_{proj}_codebase, _sessions, _checkpoints             │
│   └── {org}_feedback, {org}_anti_patterns                               │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Three-Layer Model

### Layer 1: Context-Folding (Intra-Session)

Active context management within a single agent session using `branch()` and `return()` MCP tools.

**MCP Tools**:
```
branch(description: string, prompt: string) → subtask_id
  - Forks current context into isolated subtask
  - Passes only description + prompt (not full context)
  - Subtask sees: system prompt + branch description + its own accumulating context

return(message: string) → void
  - Collapses subtask, returns summary to parent
  - Parent receives only the return message, not subtask's full trace
```

**When to Branch**:
- File exploration (reading multiple files to find something)
- Research tasks (web searches, documentation lookup)
- Trial-and-error debugging
- Any "throwaway" reasoning that won't affect final output

**Budget Enforcement**:
- Parent context has hard budget (e.g., 32K tokens)
- Subtasks inherit budget slice or get separate allocation
- `return()` is mandatory before subtask exceeds its budget

---

### Layer 2: ReasoningBank (Cross-Session Memory)

Distill strategies from session outcomes into structured memory items. Learn from both successes AND failures.

**Memory Item Schema**:
```json
{
  "id": "uuid",
  "title": "Short strategy name (for retrieval)",
  "description": "When/why to apply this strategy",
  "content": "Detailed steps or approach",
  "metadata": {
    "project": "contextd",
    "source_session": "session-abc123",
    "outcome": "success|failure|mixed",
    "confidence": 0.85,
    "usage_count": 12,
    "last_used": "2025-01-15T...",
    "tags": ["error-handling", "go", "testing"]
  }
}
```

**Consensus-Based Confidence**:
```
Signals (weighted):
- Explicit feedback (+user marked helpful)     → +0.3
- Implicit success (task completed, no retry)  → +0.1
- Agent self-judgment ("this worked well")     → +0.1
- Code stability (no reverts within 7 days)    → +0.2
- Time-decay (older = less confident)          → -0.05/month
```

---

### Layer 3: Institutional Knowledge (Cross-Project/Org)

Consolidate proven patterns from multiple projects into org-wide knowledge.

**Hierarchical Scoping**:
```
Organization: Acme Corp
├── Team: Platform
│   ├── Project: contextd
│   └── Project: auth-service
└── Team: Frontend
    ├── Project: dashboard
    └── Project: mobile-app

Retrieval cascade:
1. Project bank (most specific)
2. Team bank
3. Org bank (most general)

Promotion cascade:
Project → Team (if 2+ projects share) → Org (if 2+ teams share)
```

---

## Collection Architecture

### Database Structure (per Organization)

```
Qdrant Cluster
│
└── Database: {org_id}
    │
    ├── ══════════════════════════════════════════════════════════
    │   KNOWLEDGE COLLECTIONS (ReasoningBank + Institutional)
    ├── ══════════════════════════════════════════════════════════
    │
    ├── {org}_memories                    # org-wide strategies
    ├── {org}_{team}_memories             # team-level strategies
    ├── {org}_{team}_{project}_memories   # project-specific
    │
    ├── ══════════════════════════════════════════════════════════
    │   REMEDIATION COLLECTIONS
    ├── ══════════════════════════════════════════════════════════
    │
    ├── {org}_remediations                # org-wide fixes
    ├── {org}_{team}_remediations         # team-level fixes
    ├── {org}_{team}_{project}_remediations
    │
    ├── ══════════════════════════════════════════════════════════
    │   STANDARDS & GOVERNANCE
    ├── ══════════════════════════════════════════════════════════
    │
    ├── {org}_policies                    # security/compliance policies
    ├── {org}_coding_standards            # org coding conventions
    ├── {org}_{team}_coding_standards     # team overrides/extensions
    ├── {org}_repo_standards              # repo structure, CI/CD patterns
    │
    ├── ══════════════════════════════════════════════════════════
    │   AGENT & SKILL REGISTRY
    ├── ══════════════════════════════════════════════════════════
    │
    ├── {org}_skills                      # reusable skill definitions
    ├── {org}_agents                      # agent configurations/personas
    │
    ├── ══════════════════════════════════════════════════════════
    │   CODEBASE CONTEXT
    ├── ══════════════════════════════════════════════════════════
    │
    ├── {org}_{team}_{project}_codebase   # file/function embeddings
    │
    ├── ══════════════════════════════════════════════════════════
    │   SESSION & OPERATIONAL
    ├── ══════════════════════════════════════════════════════════
    │
    ├── {org}_{team}_{project}_sessions   # session traces
    ├── {org}_{team}_{project}_checkpoints # saved context snapshots
    │
    ├── ══════════════════════════════════════════════════════════
    │   FEEDBACK & LEARNING
    ├── ══════════════════════════════════════════════════════════
    │
    ├── {org}_feedback                    # explicit user ratings
    └── {org}_anti_patterns               # failure-derived warnings
```

### Collection Schemas

#### memories
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "Go Error Wrapping",
    "description": "When handling errors in Go services",
    "content": "Always wrap with %w, include context...",
    "outcome": "success",
    "confidence": 0.92,
    "usage_count": 15,
    "source_session": "session-abc",
    "tags": ["go", "errors"],
    "created_at": "2025-01-15T...",
    "last_used": "2025-01-20T..."
  }
}
```

#### remediations
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "Fix: PostgreSQL Connection Pool Exhaustion",
    "problem": "Connection pool exhausted under load",
    "symptoms": ["timeout errors", "503 responses", "pgx pool full"],
    "root_cause": "Connections not returned due to missing defer",
    "solution": "Add defer conn.Release() after Acquire()",
    "code_diff": "...",
    "affected_files": ["pkg/db/pool.go"],
    "status": "canonical",
    "canonical_id": null,
    "occurrence_count": 3,
    "source_sessions": ["a", "b", "c"],
    "merged_from": ["rem_001", "rem_047"],
    "confidence": 0.95,
    "verified": true
  }
}
```

#### policies
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "Secret Management Policy",
    "category": "security",
    "severity": "critical",
    "requirement": "Never commit secrets to version control",
    "enforcement": "pre-commit hook + CI scan",
    "tools": ["gitleaks", "trufflehog"],
    "exceptions_process": "Security team approval required",
    "last_reviewed": "2025-01-01"
  }
}
```

#### coding_standards
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "Error Handling Standard",
    "language": "go",
    "rule": "Always wrap errors with context using fmt.Errorf and %w",
    "good_example": "return fmt.Errorf(\"failed to connect: %w\", err)",
    "bad_example": "return err",
    "rationale": "Preserves error chain for debugging",
    "linter": "errorlint",
    "severity": "error"
  }
}
```

#### repo_standards
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "Go Project Structure",
    "type": "directory_structure",
    "standard": "golang-standards/project-layout",
    "required_dirs": ["cmd/", "pkg/", "internal/"],
    "forbidden_dirs": ["src/", "lib/", "models/"],
    "rationale": "Consistency across Go projects",
    "template_repo": "github.com/acme/go-template"
  }
}
```

#### skills
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "name": "tdd-workflow",
    "description": "Test-driven development workflow",
    "trigger": "When implementing new features",
    "steps": ["Write failing test", "Implement minimal code", "Refactor"],
    "prompt_template": "...",
    "required_tools": ["test_runner", "file_edit"],
    "success_rate": 0.89,
    "avg_tokens_saved": 2500
  }
}
```

#### agents
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "name": "code-reviewer",
    "description": "Reviews PRs against coding standards",
    "system_prompt": "You are a code reviewer...",
    "skills": ["tdd-workflow", "security-review"],
    "collections_access": ["coding_standards", "policies", "repo_standards"],
    "model_preference": "claude-sonnet-4-5-20250929",
    "temperature": 0.3
  }
}
```

#### anti_patterns
```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "AVOID: Raw SQL in HTTP Handlers",
    "category": "security",
    "what_happened": "SQL injection in /api/users endpoint",
    "source_incident": "PR #234, security audit 2025-01",
    "why_bad": "User input concatenated into SQL query",
    "instead": "Use parameterized queries via sqlc",
    "detection": "grep for 'fmt.Sprintf.*SELECT'",
    "severity": "critical"
  }
}
```

---

## Data Flow

### Session Lifecycle

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         SESSION START                                   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 1. CONTEXT INJECTION (budget-aware, tiered)                             │
│                                                                         │
│    TIER 0 (always, ~500 tokens):                                        │
│    ├── policies (critical/security)                                     │
│    └── coding_standards (active project's language)                     │
│                                                                         │
│    TIER 1 (high confidence, ~1000 tokens):                              │
│    ├── memories (project → team → org, top-k by relevance)              │
│    ├── anti_patterns (if task matches known failure patterns)           │
│    └── skills (if task matches skill triggers)                          │
│                                                                         │
│    TIER 2 (on-demand via tools):                                        │
│    ├── remediations (when errors encountered)                           │
│    ├── codebase context (when exploring code)                           │
│    └── repo_standards (when creating files/projects)                    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 2. ACTIVE SESSION (Layer 1: Context-Folding)                            │
│                                                                         │
│    ┌─────────────────────────────────────────────────────────────┐     │
│    │ Main Context (32K budget)                                    │     │
│    │ ├── System prompt + injected knowledge                       │     │
│    │ ├── User request                                             │     │
│    │ │                                                            │     │
│    │ ├── branch("explore auth code") ──────┐                      │     │
│    │ │                                      ▼                     │     │
│    │ │                              ┌─────────────────┐           │     │
│    │ │                              │ Subtask Context │           │     │
│    │ │                              │ (isolated 8K)   │           │     │
│    │ │                              └────────┬────────┘           │     │
│    │ │                                       │                    │     │
│    │ ◄── return("auth in pkg/auth/...")◄─────┘                    │     │
│    │ │                                                            │     │
│    │ └── Continues with summary only                              │     │
│    └─────────────────────────────────────────────────────────────┘     │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 3. SESSION END                                                          │
│    ├── Save session trace → sessions collection                         │
│    └── Optional: checkpoint → checkpoints collection                    │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 4. ASYNC DISTILLATION                                                   │
│    ├── Successful strategies → memories                                 │
│    ├── Failed approaches → anti_patterns                                │
│    └── Bug fixes applied → remediations                                 │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 5. PERIODIC CONSOLIDATION                                               │
│    ├── Merge similar memories (SemDeDup clustering)                     │
│    ├── Deduplicate remediations by resolution                           │
│    ├── Promote cross-project patterns to team/org                       │
│    └── Decay/prune low-confidence items                                 │
└─────────────────────────────────────────────────────────────────────────┘
```

### Retrieval Priority Algorithm

```python
def retrieve_context(task_embedding, project, team, org, budget=2000):
    results = []
    remaining_budget = budget

    # TIER 0: Always inject (critical governance)
    tier0 = parallel_query([
        (f"{org}_policies", {"severity": "critical"}, 200),
        (f"{org}_coding_standards", {"language": detect_language(task)}, 200),
    ])
    results.extend(tier0)
    remaining_budget -= token_count(tier0)

    # TIER 1: High-relevance knowledge (cascading scope)
    for scope in [f"{org}_{team}_{project}", f"{org}_{team}", org]:
        if remaining_budget < 200:
            break

        memories = query(
            f"{scope}_memories",
            task_embedding,
            filter={"confidence": {"$gte": 0.7}},
            limit=5
        )
        memories = dedupe(memories, results)
        results.extend(fit_budget(memories, remaining_budget * 0.6))
        remaining_budget -= token_count(memories)

    # Anti-patterns (if task matches failure signatures)
    anti = query(f"{org}_anti_patterns", task_embedding, limit=3)
    if anti[0].score > 0.8:
        results.extend(anti[:2])
        remaining_budget -= token_count(anti[:2])

    return results
```

---

## MCP Interface

### Server Identification
```json
{
  "name": "contextd",
  "version": "1.0.0",
  "description": "Context management with ReasoningBank for AI agents",
  "protocol_version": "2024-11-05"
}
```

### Tools

#### Context-Folding (Layer 1)
```yaml
branch:
  description: Fork context into isolated subtask
  parameters:
    description: string
    prompt: string
    budget?: number
    inject_memories?: bool
  returns:
    subtask_id: string
    injected_context: string[]

return:
  description: Complete subtask and return summary to parent
  parameters:
    message: string
    extract_memory?: bool
  returns:
    success: boolean
```

#### Memory (Layer 2)
```yaml
memory_search:
  description: Search for relevant strategies/patterns
  parameters:
    query: string
    scope?: "project" | "team" | "org" | "all"
    outcome?: "success" | "failure" | "all"
    limit?: number
    min_confidence?: float
  returns:
    memories: Memory[]

memory_record:
  description: Explicitly capture a strategy or pattern
  parameters:
    title: string
    description: string
    content: string
    outcome: "success" | "failure"
    tags?: string[]
  returns:
    id: string

memory_feedback:
  description: Provide feedback on retrieved memory
  parameters:
    memory_id: string
    helpful: boolean
    comment?: string
  returns:
    success: boolean
    new_confidence: float
```

#### Remediation
```yaml
remediation_search:
  description: Find fixes for errors/problems
  parameters:
    error: string
    context?: string
    limit?: number
  returns:
    remediations: Remediation[]

remediation_record:
  description: Record a fix that worked
  parameters:
    title: string
    problem: string
    symptoms: string[]
    root_cause: string
    solution: string
    code_diff?: string
    affected_files?: string[]
  returns:
    id: string
```

#### Standards & Governance
```yaml
policy_check:
  description: Check if action complies with policies
  parameters:
    action: string
    context?: string
  returns:
    compliant: boolean
    violations: PolicyViolation[]
    suggestions: string[]

standards_get:
  description: Get relevant coding/repo standards
  parameters:
    type: "coding" | "repo" | "both"
    language?: string
    category?: string
  returns:
    standards: Standard[]
```

#### Skills & Agents
```yaml
skill_search:
  description: Find skills matching current task
  parameters:
    task: string
    limit?: number
  returns:
    skills: Skill[]

skill_get:
  description: Get full skill definition
  parameters:
    name: string
  returns:
    skill: Skill

agent_invoke:
  description: Invoke a configured agent for subtask
  parameters:
    agent_name: string
    task: string
    context?: object
  returns:
    subtask_id: string
```

#### Codebase Context
```yaml
codebase_search:
  description: Semantic search over codebase
  parameters:
    query: string
    file_pattern?: string
    limit?: number
  returns:
    results: CodeResult[]

codebase_index:
  description: Trigger re-indexing of codebase
  parameters:
    path?: string
    incremental?: boolean
  returns:
    job_id: string
```

#### Session Management
```yaml
session_start:
  description: Initialize session with context injection
  parameters:
    project: string
    task_description?: string
    budget?: number
  returns:
    session_id: string
    injected: InjectedContext
    remaining_budget: number

session_checkpoint:
  description: Save current context for later resumption
  parameters:
    summary: string
    tags?: string[]
  returns:
    checkpoint_id: string

session_resume:
  description: Resume from checkpoint
  parameters:
    checkpoint_id: string
  returns:
    session_id: string
    restored_context: string

session_end:
  description: End session and trigger distillation
  parameters:
    outcome?: "success" | "failure" | "partial"
    distill?: boolean
  returns:
    session_id: string
    memories_extracted: number
```

#### Knowledge Briefing
```yaml
briefing_get:
  description: Get onboarding/context briefing for project
  parameters:
    project: string
    depth?: "minimal" | "standard" | "comprehensive"
  returns:
    briefing: Briefing
    token_count: number

knowledge_promote:
  description: Promote knowledge to higher scope
  parameters:
    item_id: string
    item_type: "memory" | "remediation" | "standard"
    target_scope: "team" | "org"
    generalize?: boolean
  returns:
    new_id: string
```

#### Consolidation
```yaml
consolidation_status:
  description: Get status of collection optimization
  parameters:
    collection: "remediations" | "anti_patterns" | "memories"
  returns:
    last_run: timestamp
    items_before: number
    items_after: number
    clusters_merged: number

consolidation_trigger:
  description: Manually trigger consolidation (admin only)
  parameters:
    collection: string
    dry_run?: boolean
  returns:
    proposed_merges: MergeProposal[]
```

### Resources
```yaml
resources:
  - uri: contextd://project/{project}/briefing
    name: Project Briefing
    mimeType: text/markdown

  - uri: contextd://project/{project}/standards
    name: Project Standards
    mimeType: application/json

  - uri: contextd://org/{org}/policies
    name: Organization Policies
    mimeType: application/json
```

### Notifications
```yaml
notifications:
  contextd/memory_match:
    description: High-relevance memory found
    params:
      memory: Memory
      relevance: float

  contextd/policy_warning:
    description: Action may violate policy
    params:
      policy: Policy
      violation: string

  contextd/budget_warning:
    description: Context budget running low
    params:
      used: number
      total: number
```

---

## Multi-Tenancy & RBAC

### SaaS Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     contextd SaaS Platform                      │
├─────────────────────────────────────────────────────────────────┤
│  Control Plane (shared)                                         │
│  ├── Tenant Management API                                      │
│  ├── Billing & Usage Tracking                                   │
│  ├── Provisioning Service                                       │
│  └── Global Config                                              │
├─────────────────────────────────────────────────────────────────┤
│  Data Plane (isolated per tenant)                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Tenant A    │  │ Tenant B    │  │ Tenant C    │             │
│  │ Collection: │  │ Collection: │  │ Collection: │             │
│  │ acme_corp_* │  │ globex_*    │  │ initech_*   │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### Collection-per-Tenant Isolation

For SaaS deployments, each tenant gets isolated collections:

```
Tenant: acme-corp
├── acme_corp_memories
├── acme_corp_platform_memories      (team: platform)
├── acme_corp_platform_contextd_memories  (project)
├── acme_corp_remediations
├── acme_corp_policies
└── ...
```

**Why collection-per-tenant**:
- Physical isolation (impossible to leak data)
- Per-tenant performance guarantees
- Easy backup/restore/export per tenant
- GDPR: drop collection = complete deletion
- Compliance-friendly

### RBAC Model (contextd layer)

Qdrant provides collection-level RBAC. contextd implements finer-grained access:

```
Roles:
├── org_admin    → all collections in org
├── team_admin   → team + project collections
├── team_member  → team read + project read/write
├── project_dev  → single project read/write
└── viewer       → read-only specified collections

Permissions Matrix:
┌──────────────┬───────┬───────┬───────┬───────┬───────┐
│ Collection   │OrgAdm │TeamAdm│TeamMbr│ProjDev│Viewer │
├──────────────┼───────┼───────┼───────┼───────┼───────┤
│ org_*        │  RW   │  R    │  R    │  -    │  R    │
│ team_*       │  RW   │  RW   │  R    │  -    │  R    │
│ project_*    │  RW   │  RW   │  RW   │  RW   │  R    │
│ policies     │  RW   │  R    │  R    │  R    │  R    │
│ skills/agents│  RW   │  RW   │  R    │  R    │  R    │
└──────────────┴───────┴───────┴───────┴───────┴───────┘
```

### JWT Token Structure

```json
{
  "sub": "user-123",
  "org_id": "acme-corp",
  "team_ids": ["platform", "devops"],
  "role": "team_member",
  "projects": ["contextd", "auth-service"],
  "exp": 1737500000
}
```

---

## Consolidation & Deduplication

### Resolution-Based Deduplication

For remediations and similar collections, deduplicate by matching solutions:

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. CLUSTER BY RESOLUTION                                        │
│    embed(solution + code_diff), cluster where similarity > 0.85 │
├─────────────────────────────────────────────────────────────────┤
│ 2. MERGE SYMPTOMS (keep all - valuable signals)                 │
│    Union all symptoms from clustered items                      │
├─────────────────────────────────────────────────────────────────┤
│ 3. ARCHIVE ORIGINALS                                            │
│    Mark as "merged", link to canonical                          │
└─────────────────────────────────────────────────────────────────┘
```

### Consolidation Config

```json
{
  "remediations": {
    "enabled": true,
    "schedule": "0 3 * * *",
    "similarity_threshold": 0.85,
    "min_cluster_size": 2,
    "merge_strategy": "resolution_match",
    "archive_originals": true,
    "boost_confidence_per_occurrence": 0.05
  },
  "anti_patterns": {
    "enabled": true,
    "schedule": "0 4 * * *",
    "similarity_threshold": 0.80,
    "merge_strategy": "pattern_match"
  },
  "memories": {
    "enabled": true,
    "schedule": "0 5 * * *",
    "similarity_threshold": 0.90,
    "min_cluster_size": 3,
    "merge_strategy": "strategy_generalization"
  }
}
```

---

## Deployment Models

### Local-First (Individual/Small Team)
```
Developer Machine
├── contextd (single binary)
├── Qdrant (embedded or local container)
└── SQLite (metadata, if needed)
```

### On-Premise (Enterprise)
```
Kubernetes Cluster
├── contextd (Deployment, replicated)
├── Qdrant (StatefulSet, clustered)
├── PostgreSQL (metadata)
└── Redis (caching, optional)
```

### SaaS (Multi-Tenant)
```
Cloud Infrastructure
├── contextd API (auto-scaling)
├── Qdrant Cloud (managed)
├── PostgreSQL (tenant metadata)
├── Redis (session cache)
└── Background Workers (distillation, consolidation)
```

---

## References

1. **ReasoningBank**: Google DeepMind, arXiv:2509.25140 - Memory framework for agent self-evolution
2. **Context-Folding**: ByteDance, arXiv:2510.11967 - Active context management via branch/return
3. **12-Factor Agents**: HumanLayer - Architecture principles including "Own Your Context Window"
4. **Mem0**: Memory consolidation operations (ADD/MERGE/INVALIDATE/SKIP)
5. **SemDeDup**: Semantic clustering for deduplication

---

## Next Steps

1. [ ] Define detailed API specifications (OpenAPI/gRPC)
2. [ ] Design database schema for metadata (PostgreSQL)
3. [ ] Implement core MCP server
4. [ ] Build distillation pipeline
5. [ ] Create consolidation workers
6. [ ] Develop CLI and SDK
7. [ ] Write integration tests
8. [ ] Deploy proof-of-concept
