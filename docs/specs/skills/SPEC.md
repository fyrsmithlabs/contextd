# Skills Management System Specification

## Overview

The Skills Management System provides a semantic search and storage platform for reusable workflow templates and knowledge artifacts. Skills are versioned, categorized, and enriched with usage analytics to enable AI agents and developers to discover and apply proven patterns across projects.

**📚 For skill authoring best practices**, see [SKILL-AUTHORING.md](SKILL-AUTHORING.md) which provides comprehensive TDD-based methodology.

**Package**: `pkg/skills`

**Status**: Production (Multi-Tenant Mode Only)

**Database**: `shared` (global knowledge accessible to all projects)

## Purpose

Skills serve as reusable knowledge artifacts that:
- Capture proven workflows and patterns
- Enable semantic discovery through vector embeddings
- Track effectiveness through usage analytics
- Support versioning and evolution
- Provide prerequisites and expected outcomes
- Facilitate knowledge sharing across projects

## Skill Creation Methodology

**📚 For comprehensive skill authoring best practices**, see [SKILL-AUTHORING.md](SKILL-AUTHORING.md) which provides TDD-based methodology.

**This section incorporates proven patterns from the [superpowers plugin](https://github.com/superpowers-labs/superpowers)** by @dmarx and contributors, adapted for contextd's skill management system.

### TDD for Skill Creation

**Skills ARE tested like code.** You write tests (pressure scenarios), watch them fail (baseline), write the skill (documentation), watch tests pass (compliance).

#### The Iron Law

```
NO SKILL WITHOUT A FAILING TEST FIRST
```

This applies to:
- ✅ New skills - test before writing
- ✅ Skill edits - test before changing
- ✅ "Simple additions" - test first
- ✅ "Just documentation" - test first

**No exceptions.** Untested skills have issues. Always.

#### RED-GREEN-REFACTOR Cycle

| Phase | Skill Testing | What You Do |
|-------|---------------|-------------|
| **RED** | Baseline test | Run scenario WITHOUT skill, watch agent fail |
| **Verify RED** | Capture rationalizations | Document exact failures verbatim |
| **GREEN** | Write skill | Address specific baseline failures |
| **Verify GREEN** | Pressure test | Run scenario WITH skill, verify compliance |
| **REFACTOR** | Plug holes | Find new rationalizations, add counters |
| **Stay GREEN** | Re-verify | Test again, ensure still compliant |

### Pressure Testing with Subagents

**Purpose**: Simulate realistic conditions where agents might skip/rationalize skill application.

**Pressure types:**
1. **Time pressure**: "Need this in 5 minutes"
2. **Sunk cost**: "Spent 4 hours already"
3. **Authority**: "Manager says skip tests"
4. **Exhaustion**: "End of day, dinner in 30 min"
5. **Overconfidence**: "Done this before, simple"

**Combine 3+ pressures** for discipline-enforcing skills.

#### Baseline Testing (RED Phase)

**Process:**
1. Create realistic scenario with pressures
2. Run WITHOUT the skill
3. Document agent's exact choices and rationalizations
4. Identify patterns in failures
5. Note which pressures trigger violations

#### Writing Skill (GREEN Phase)

Write skill that addresses **specific baseline failures**.

**Don't**:
- Write generic advice
- Add hypothetical counters
- Over-explain simple concepts

**Do**:
- Address exact rationalizations from baseline
- Use agent's own words in rationalization tables
- Make rules explicit and unambiguous

#### Closing Loopholes (REFACTOR Phase)

**For discipline-enforcing skills:**

1. **Explicit exceptions list:**
```markdown
## No Exceptions

- Not for "simple code"
- Not for "I already tested manually"
- Not for "tests after achieve same goal"
- Delete means delete
```

2. **Rationalization table:**
```markdown
| Excuse | Reality |
|--------|---------|
| "Too simple to test" | Simple code breaks. Test takes 30s. |
| "I'll test after" | Tests passing immediately prove nothing. |
```

3. **Red flags list:**
```markdown
## Red Flags - STOP

- Code before test
- "Already manually tested"
- "This is different because..."

All of these mean: Delete code. Start over.
```

### Claude Search Optimization (CSO)

Skills must be **discoverable** by AI agents searching for solutions.

#### 1. Rich Description Field

The description is read by AI to decide "Should I load this skill?"

**Include**:
- Concrete triggers and symptoms
- Problem descriptions (not language-specific unless skill is)
- Technology context if skill is specific
- Both when to use AND what it does

#### 2. Keyword Coverage

Use words AI would search for:
- **Error messages**: "ENOTEMPTY", "timeout", "race condition"
- **Symptoms**: "flaky", "inconsistent", "hanging"
- **Synonyms**: "timeout/hang/freeze", "cleanup/teardown"
- **Tools**: Actual commands, library names

#### 3. Descriptive Naming

Use active voice, verb-first:
- ✅ `github-actions-workflows` not `gha-workflow-docs`
- ✅ `creating-skills` not `skill-creation`
- ✅ `testing-skills-with-subagents` not `subagent-skill-test`

Gerunds (-ing) work well for processes:
- `creating-skills`, `testing-skills`, `debugging-workflows`

#### 4. Token Efficiency

Skills load into EVERY conversation where relevant. Every token counts.

**Target word counts**:
- Frequently-referenced skills: <200 words
- Other skills: <500 words
- Heavy reference: Separate file, link from skill

**Techniques**:

**Move details to tool help:**
```markdown
# ❌ Document all flags
tool supports --flag1, --flag2, --flag3...

# ✅ Reference help
tool supports multiple modes. Run --help for details.
```

**Use cross-references:**
```markdown
# ❌ Repeat workflow
When doing X, follow these 20 steps...

# ✅ Reference other skill
REQUIRED: Use other-skill-name for workflow.
```

**Compress examples:**
```markdown
# ❌ Verbose (42 words)
User: "How did we solve X?"
You: I'll search conversations.
[Dispatch agent with query...]

# ✅ Minimal (20 words)
User: "How did we solve X?"
You: Searching...
[Dispatch agent → synthesis]
```

### Skill Structure Guidelines

#### Frontmatter (YAML)

```yaml
---
name: skill-name-with-hyphens
description: Use when [specific triggers] - [what it does in third person]
---
```

**Rules**:
- Only two fields: `name` and `description`
- Max 1024 characters total
- Name: Letters, numbers, hyphens only (no special chars)
- Description: Start with "Use when..." then explain what it does

**Good Description Examples**:

```yaml
# ✅ Technology-agnostic with clear triggers
description: Use when tests have race conditions, timing dependencies, or pass/fail inconsistently - replaces arbitrary timeouts with condition polling for reliable async tests

# ✅ Technology-specific with explicit context
description: Use when creating or modifying GitHub Actions workflows - provides security patterns, common gotchas, performance optimizations, and debugging techniques

# ✅ Problem-first, then solution
description: Use when errors occur deep in execution and you need to trace back to find the original trigger - systematically traces bugs backward through call stack to identify source
```

**Bad Description Examples**:

```yaml
# ❌ Too abstract, no triggers
description: For async testing

# ❌ First person
description: I can help with async tests

# ❌ Doesn't include when to use
description: Provides async testing patterns
```

#### Content Structure

```markdown
# Skill Name

## Overview
What is this? Core principle in 1-2 sentences.

## When to Use
Bullet list with SYMPTOMS and use cases
When NOT to use

## Quick Reference
Table or bullets for scanning common operations

## Core Pattern (for techniques/patterns)
Before/after code comparison or step-by-step

## Implementation
Inline code for simple patterns
Link to file for heavy reference

## Common Mistakes
What goes wrong + fixes

## Real-World Impact (optional)
Concrete results from using this skill
```

### Credits and Attribution

This skill creation methodology is based on the excellent patterns developed by the **superpowers plugin** community:

- **Repository**: https://github.com/superpowers-labs/superpowers
- **Key Skills Referenced**:
  - `writing-skills` - TDD approach to skill creation
  - `testing-skills-with-subagents` - Pressure testing methodology
  - `using-superpowers` - Discovery and application patterns

**Thank you** to @dmarx and all superpowers contributors for developing these proven patterns.

**Adaptations for contextd**:
- Integration with contextd MCP API for skill storage
- Storage in Qdrant vector database with semantic search
- Usage tracking and analytics for skill effectiveness
- Category and tag organization for discovery

## Features and Capabilities

### Core Features

1. **Skill Creation** (`Create`)
   - Automatic embedding generation for semantic search
   - Version management (semantic versioning)
   - Author attribution and metadata
   - Category and tag-based organization
   - Prerequisites and expected outcomes documentation

2. **Semantic Search** (`Search`)
   - Vector similarity search using embeddings
   - Category and tag filtering
   - Configurable result limits (1-100)
   - Score and distance metrics
   - Multi-field embedding (name + description + content)

3. **Skill Listing** (`List`)
   - Paginated results (limit + offset)
   - Category and tag filtering
   - Multiple sort options (created_at, updated_at, usage_count, success_rate)
   - Total count for pagination

4. **Skill Updates** (`Update`)
   - Partial updates (nullable fields)
   - Automatic re-embedding when content changes
   - Version tracking
   - Metadata updates
   - Prerequisites and expected outcome updates

5. **Skill Deletion** (`Delete`)
   - Hard delete by ID
   - Filter-based deletion
   - Cascade considerations (usage stats)

6. **Skill Application** (`Apply`, `GetByID`)
   - Retrieve skill content for application
   - Optional usage tracking (success/failure)
   - Success rate calculation
   - Usage count tracking

### Usage Analytics

- **Usage Count**: Number of times skill has been applied
- **Success Rate**: Ratio of successful applications (0.0-1.0)
- **Success Tracking**: Optional success flag on apply
- **Last Used Tracking**: Automatic timestamp updates
- **Trend Analysis**: Support for usage analytics queries (future)

## Skill Lifecycle

### 1. Creation Phase

```
User/Agent Request → Validation → Embedding Generation → Database Storage
```

**Steps**:
1. Validate skill metadata (name, description, content, version, category)
2. Generate embedding vector from `name + description + content`
3. Serialize metadata to JSON
4. Store in shared database with vector and payload
5. Return skill with generated ID and token count

**Constraints**:
- Name: 1-200 characters (required)
- Description: 1-2000 characters (required)
- Content: 1-50000 characters (required)
- Version: Semantic version format (required)
- Author: Max 200 characters (required)
- Category: Predefined categories (required)
- Tags: 0-20 tags, max 50 characters each
- Metadata: Max 100 key-value pairs

### 2. Discovery Phase

```
Search Query → Embedding Generation → Vector Search → Filter Application → Results Ranking
```

**Search Methods**:
1. **Semantic Search**: Vector similarity using cosine distance
2. **Category Filter**: Exact match on category field
3. **Tag Filter**: Multiple tags with OR logic
4. **Combined**: Semantic + filters for precise discovery

**Ranking**:
- Primary: Vector similarity score (0.0-1.0)
- Secondary: Distance metric (lower is better)
- Optional: Boost by usage_count or success_rate (future)

### 3. Application Phase

```
Get by ID → Return Content → Optional Usage Tracking → Statistics Update
```

**Application Flow**:
1. Retrieve skill by ID
2. Return full content including prerequisites and expected outcome
3. Agent applies skill to current context
4. Optional: Record usage with success flag
5. Update usage statistics (count, success rate)

**Statistics Calculation**:
```
previous_success_count = usage_count * success_rate
new_usage_count = usage_count + 1
new_success_count = previous_success_count + (1 if success else 0)
new_success_rate = new_success_count / new_usage_count
```

### 4. Evolution Phase

```
Update Request → Delta Computation → Optional Re-embedding → Delete + Re-insert
```

**Update Strategy**:
- Read existing skill vector
- Apply partial updates to payload
- Re-embed if name, description, or content changed
- Preserve usage statistics and creation timestamp

**Version Management**:
- Semantic versioning (major.minor.patch)
- Version updates tracked in updated_at timestamp
- No automatic version incrementing (user responsibility)

### 5. Deletion Phase

```
Delete Request → Filter Construction → Vector Deletion → Confirmation
```

**Deletion Strategy**:
- Hard delete (no soft delete)
- Filter-based deletion (typically by ID)
- No cascade (usage stats lost)
- Idempotent (no error if already deleted)

## Architecture and Design

### Service Layer

**Service Structure**:
```go
type Service struct {
    vectorStore     VectorStore         // Universal vector store interface
    embedder        EmbeddingGenerator  // Embedding service
    tracer          trace.Tracer        // OpenTelemetry tracing
    meter           metric.Meter        // OpenTelemetry metrics
    createCounter   metric.Int64Counter
    searchCounter   metric.Int64Counter
    updateCounter   metric.Int64Counter
    deleteCounter   metric.Int64Counter
    applyCounter    metric.Int64Counter
    operationTime   metric.Float64Histogram
    embeddingTime   metric.Float64Histogram
    mu              sync.Mutex          // Protect concurrent updates
}
```

**Dependencies**:
- **VectorStore**: Universal interface supporting multi-tenant database isolation
- **EmbeddingGenerator**: Interface for OpenAI or TEI embedding generation
- **Tracer/Meter**: OpenTelemetry instrumentation for observability

**Concurrency**:
- Service is thread-safe
- Mutex protects Update and RecordUsage operations
- Read operations (Search, List, GetByID) are lock-free
- Create and Delete are inherently safe (no read-modify-write)

### Database Schema

**Database**: `shared` (global knowledge)

**Collection**: `skills`

**Vector Dimension**: 1536 (standard for text-embedding-3-small and BAAI/bge-small-en-v1.5)

**Payload Schema**:
```go
{
    "id":               string              // UUID
    "name":             string              // Skill name (max 200 chars)
    "description":      string              // Description (max 2000 chars)
    "content":          string              // Markdown content (max 50000 chars)
    "version":          string              // Semantic version (e.g., "1.0.0")
    "author":           string              // Author name (max 200 chars)
    "category":         string              // Category (debugging, deployment, etc.)
    "prerequisites":    []string            // Required tools/skills/conditions
    "expected_outcome": string              // What skill accomplishes
    "tags":             string              // Comma-separated tags
    "metadata":         string              // JSON-encoded key-value pairs
    "usage_count":      int64               // Times applied
    "success_rate":     float32             // Success ratio (0.0-1.0)
    "created_at":       int64               // Unix timestamp
    "updated_at":       int64               // Unix timestamp
}
```

**Index Strategy**:
- Vector index: HNSW (Hierarchical Navigable Small World) for fast ANN search
- Scalar index: category (exact match filter)
- Scalar index: tags (substring match filter)
- Scalar index: created_at, updated_at (sorting)
- Scalar index: usage_count, success_rate (future analytics)

### Multi-Tenant Isolation

**Storage Model**:
- Skills stored in `shared` database (global knowledge)
- All projects access same skill collection
- No project-specific filtering (skills are global by design)
- Physical isolation from project-specific data (checkpoints, research)

**Security**:
- Database-level isolation prevents cross-contamination
- No filter injection risk (skills have no project_id field)
- Better query performance (no filter overhead)

**Backward Compatibility**:
- Legacy mode removed in v2.0.0 (security fix)
- All deployments must use multi-tenant mode
- Migration required for v1.x users (see docs/MIGRATION-FROM-LEGACY.md)

## API Specifications

### MCP Tools

The skills system exposes 6 MCP tools for Claude Code integration.

#### 1. skill_create

Create a new reusable skill/workflow template with semantic search capability.

**Input**:
```json
{
  "name": "string (required, 1-200 chars)",
  "description": "string (required, 1-2000 chars)",
  "content": "string (required, 1-50000 chars, markdown format)",
  "version": "string (required, semantic version)",
  "author": "string (required, max 200 chars)",
  "category": "string (required, debugging|deployment|analysis|testing|etc)",
  "tags": ["string", "..."] (optional, max 20 tags),
  "prerequisites": ["string", "..."] (optional),
  "expected_outcome": "string (optional)",
  "metadata": {"key": "value", ...} (optional)
}
```

**Output**:
```json
{
  "id": "uuid",
  "name": "string",
  "version": "string",
  "token_count": 1234,
  "created_at": "2025-11-04T10:00:00Z"
}
```

**Timeout**: 30 seconds (embedding + database operations)

**Errors**:
- ValidationError: Invalid name, description, content, tags, or category
- TimeoutError: Operation exceeded 30 seconds
- InternalError: Embedding generation or database failure

#### 2. skill_search

Search for skills using semantic similarity with optional filtering by category and tags.

**Input**:
```json
{
  "query": "string (required, 1-1000 chars)",
  "top_k": 5 (optional, default 5, range 1-100),
  "category": "string (optional)",
  "tags": ["string", "..."] (optional)
}
```

**Output**:
```json
{
  "results": [
    {
      "id": "uuid",
      "name": "string",
      "description": "string",
      "content": "string",
      "version": "string",
      "author": "string",
      "category": "string",
      "tags": ["string", "..."],
      "prerequisites": ["string", "..."],
      "expected_outcome": "string",
      "usage_count": 42,
      "success_rate": 0.85,
      "score": 0.92,
      "distance": 0.15,
      "metadata": {"key": "value"},
      "created_at": "2025-11-04T10:00:00Z",
      "updated_at": "2025-11-04T10:00:00Z"
    }
  ],
  "query": "string",
  "top_k": 5
}
```

**Timeout**: 10 seconds

**Errors**:
- ValidationError: Invalid query or top_k
- TimeoutError: Search exceeded 10 seconds
- InternalError: Embedding or database failure

#### 3. skill_list

List all skills with pagination and filtering by category, tags, sorting by creation date, usage count, or success rate.

**Input**:
```json
{
  "limit": 10 (optional, default 10, range 1-100),
  "offset": 0 (optional, default 0, min 0),
  "category": "string (optional)",
  "tags": ["string", "..."] (optional),
  "sort_by": "created_at|updated_at|usage_count|success_rate (optional)"
}
```

**Output**:
```json
{
  "skills": [
    {
      "id": "uuid",
      "name": "string",
      "description": "string",
      "content": "string",
      "version": "string",
      "author": "string",
      "category": "string",
      "tags": ["string", "..."],
      "prerequisites": ["string", "..."],
      "expected_outcome": "string",
      "usage_count": 42,
      "success_rate": 0.85,
      "metadata": {"key": "value"},
      "created_at": "2025-11-04T10:00:00Z",
      "updated_at": "2025-11-04T10:00:00Z"
    }
  ],
  "total": 100,
  "limit": 10,
  "offset": 0
}
```

**Timeout**: 5 seconds


**Errors**:
- ValidationError: Invalid limit or offset
- InternalError: Database failure

#### 4. skill_update

Update an existing skill (name, description, content, version, tags, metadata).

**Input**:
```json
{
  "id": "string (required, uuid)",
  "name": "string (optional, 1-200 chars)",
  "description": "string (optional, 1-2000 chars)",
  "content": "string (optional, 1-50000 chars)",
  "version": "string (optional, semantic version)",
  "category": "string (optional)",
  "tags": ["string", "..."] (optional),
  "prerequisites": ["string", "..."] (optional),
  "expected_outcome": "string (optional)",
  "metadata": {"key": "value", ...} (optional)
}
```

**Output**:
```json
{
  "id": "uuid",
  "name": "string",
  "version": "string",
  "updated_at": "2025-11-04T10:00:00Z"
}
```

**Timeout**: 30 seconds (re-embedding if content changed)

**Behavior**:
- Partial updates (only specified fields changed)
- Re-embedding triggered if name, description, or content updated
- Usage statistics preserved
- Created timestamp preserved

**Errors**:
- ValidationError: Missing or invalid ID
- NotFoundError: Skill ID not found
- InternalError: Database or embedding failure

#### 5. skill_delete

Delete a skill by ID. This action cannot be undone.

**Input**:
```json
{
  "id": "string (required, uuid)"
}
```

**Output**:
```json
{
  "id": "uuid",
  "message": "Skill {id} deleted successfully"
}
```

**Timeout**: 5 seconds

**Errors**:
- ValidationError: Missing ID
- InternalError: Database failure (no error if skill doesn't exist)

#### 6. skill_apply

Apply a skill to the current context and track usage statistics.

**Input**:
```json
{
  "id": "string (required, uuid)",
  "success": true|false|null (optional, for tracking)
}
```

**Output**:
```json
{
  "id": "uuid",
  "name": "string",
  "content": "string (markdown)",
  "prerequisites": ["string", "..."],
  "expected_outcome": "string",
  "usage_count": 43,
  "success_rate": 0.86
}
```

**Timeout**: 5 seconds

**Behavior**:
- Returns full skill content for agent application
- If success flag provided, updates usage statistics
- Statistics update failure doesn't fail the operation

**Errors**:
- ValidationError: Missing ID
- NotFoundError: Skill ID not found
- InternalError: Database failure

### Internal Service API

**Service Interface**:
```go
type Service interface {
    Create(ctx context.Context, req *validation.CreateSkillRequest) (*Skill, error)
    Search(ctx context.Context, req *validation.SearchSkillsRequest) (*SearchResult, error)
    List(ctx context.Context, req *validation.ListSkillsRequest) (*ListResult, error)
    Update(ctx context.Context, id string, fields *UpdateFields) (*Skill, error)
    Delete(ctx context.Context, id string) error
    GetByID(ctx context.Context, id string) (*Skill, error)
    RecordUsage(ctx context.Context, id string, success bool) error
}
```

**Request Validation**:
- Implemented in `pkg/validation` package
- Input sanitization and bounds checking
- Required field validation
- Format validation (semantic versioning, UUIDs)

**Error Handling**:
- Wrapped errors with context (`fmt.Errorf("...: %w", err)`)
- Span error recording for OpenTelemetry
- Never leak internal details or stack traces
- Consistent error types (validation, not found, timeout, internal)

## Data Models and Schemas

### Core Types

**Skill**:
```go
type Skill struct {
    ID              string            `json:"id"`              // UUID
    Name            string            `json:"name"`            // Skill name
    Description     string            `json:"description"`     // Brief description
    Content         string            `json:"content"`         // Markdown content
    Version         string            `json:"version"`         // Semantic version
    Author          string            `json:"author"`          // Author identifier
    Tags            []string          `json:"tags"`            // Categorization tags
    Prerequisites   []string          `json:"prerequisites"`   // Required tools/skills
    ExpectedOutcome string            `json:"expected_outcome"` // What skill achieves
    Category        string            `json:"category"`        // Primary category
    UsageCount      int               `json:"usage_count"`     // Application count
    SuccessRate     float64           `json:"success_rate"`    // Success ratio
    TokenCount      int               `json:"token_count"`     // Embedding tokens
    CreatedAt       time.Time         `json:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at"`
    Metadata        map[string]string `json:"metadata"`        // Additional metadata
}
```

**SkillSearchResult**:
```go
type SkillSearchResult struct {
    Skill    Skill   `json:"skill"`
    Score    float32 `json:"score"`     // Similarity score (0.0-1.0)
    Distance float32 `json:"distance"`  // Vector distance (lower is better)
}
```

**UpdateFields**:
```go
type UpdateFields struct {
    Name            *string           `json:"name,omitempty"`
    Description     *string           `json:"description,omitempty"`
    Content         *string           `json:"content,omitempty"`
    Version         *string           `json:"version,omitempty"`
    Tags            []string          `json:"tags,omitempty"`
    Prerequisites   []string          `json:"prerequisites,omitempty"`
    ExpectedOutcome *string           `json:"expected_outcome,omitempty"`
    Category        *string           `json:"category,omitempty"`
    Metadata        map[string]string `json:"metadata,omitempty"`
}
```

**ListResult**:
```go
type ListResult struct {
    Skills []Skill `json:"skills"`
    Total  int     `json:"total"`
    Limit  int     `json:"limit"`
    Offset int     `json:"offset"`
}
```

**SearchResult**:
```go
type SearchResult struct {
    Results []SkillSearchResult `json:"results"`
    Query   string              `json:"query"`
    TopK    int                 `json:"top_k"`
}
```

### Validation Constraints

**Field Constraints**:
- **Name**: 1-200 characters, required
- **Description**: 1-2000 characters, required
- **Content**: 1-50000 characters, required, markdown format
- **Version**: Semantic version format (e.g., "1.0.0"), required
- **Author**: Max 200 characters, required
- **Category**: Predefined categories, required
- **Tags**: 0-20 tags, max 50 characters each
- **Prerequisites**: 0-100 items, max 200 characters each
- **ExpectedOutcome**: Max 1000 characters
- **Metadata**: Max 100 key-value pairs, keys max 50 chars, values max 500 chars

**Categories**:
- debugging
- deployment
- analysis
- testing
- development
- documentation
- infrastructure
- security
- performance
- monitoring

## Vector Embedding Strategy

### Embedding Generation

**Content Concatenation**:
```
embedding_content = name + "\n" + description + "\n" + content
```

**Rationale**:
- Name provides primary context and keywords
- Description adds semantic meaning
- Content provides detailed implementation knowledge
- Concatenation creates rich semantic representation

**Embedding Service**:
- OpenAI: `text-embedding-3-small` (1536 dimensions, $0.02/1M tokens)
- TEI: `BAAI/bge-small-en-v1.5` (1536 dimensions, local, no cost)

**Caching**:
- Embedding service handles caching internally
- Cache key: SHA256 of input text
- Cache TTL: 24 hours (configurable)

### Re-embedding Triggers

Re-embedding occurs when:
- Name is updated
- Description is updated
- Content is updated

No re-embedding when:
- Version changes
- Tags change
- Prerequisites change
- Expected outcome changes
- Metadata changes
- Usage statistics update

**Performance**:
- Average embedding time: 100-500ms (OpenAI), 50-100ms (TEI)
- Cached embeddings: <10ms
- Embedding dimension: 1536 floats (6144 bytes)

## Search and Discovery

### Semantic Search Algorithm

**Process**:
1. Generate query embedding (same model as skill embeddings)
2. Compute cosine similarity between query vector and all skill vectors
3. Apply category and tag filters (post-filtering)
4. Sort by similarity score (descending)
5. Return top K results

**Similarity Metric**:
- Cosine similarity: `dot(query, skill) / (norm(query) * norm(skill))`
- Range: -1.0 to 1.0 (typically 0.0 to 1.0 for text)
- Higher score = better match

**Distance Metric**:
- Euclidean distance: `sqrt(sum((query[i] - skill[i])^2))`
- Lower distance = better match
- Provided for advanced filtering

### Filtering

**Category Filter**:
```
Filter: category == "debugging"
```

**Tag Filter** (OR logic):
```
Filter: (tags like "%golang%" or tags like "%error-handling%")
```

**Combined Filter**:
```
Filter: category == "debugging" and (tags like "%golang%" or tags like "%error-handling%")
```

**Filter Sanitization**:
- Escape double quotes and backslashes
- Prevent filter injection attacks

### Performance Characteristics

**Search Performance**:
- Cold search: 200-500ms (embedding generation + vector search)
- Warm search (cached embedding): 50-100ms (vector search only)
- Index type: HNSW (sub-linear time complexity)
- Scalability: Handles 100K+ skills with <100ms search time

**List Performance**:
- Fetches extra results to handle pagination
- Performance: 100-200ms for typical pagination

**Update Performance**:
- Read-modify-write with mutex protection
- Re-embedding adds 100-500ms if content changed
- Performance: 200-700ms depending on re-embedding

## Error Handling

### Error Categories

**ValidationError**:
- Invalid input parameters
- Field constraint violations
- Format errors (semantic version, UUID)
- HTTP 400 Bad Request

**NotFoundError**:
- Skill ID not found
- No matching skills for filters
- HTTP 404 Not Found

**TimeoutError**:
- Operation exceeded timeout
- Context deadline exceeded
- HTTP 408 Request Timeout

**InternalError**:
- Database failures
- Embedding service failures
- Unexpected errors
- HTTP 500 Internal Server Error

### Error Wrapping

**Pattern**:
```go
if err := doSomething(); err != nil {
    span.RecordError(err)
    return fmt.Errorf("failed to do something: %w", err)
}
```

**Benefits**:
- Preserves error chain for debugging
- Adds context at each layer
- Enables error type checking with `errors.Is()`
- OpenTelemetry span recording

### Graceful Degradation

**Usage Statistics**:
- Apply operation succeeds even if RecordUsage fails
- Usage tracking is best-effort
- Errors logged but not propagated

**Metadata Deserialization**:
- Continue with nil metadata if JSON unmarshal fails
- Log error but don't fail operation
- Prevents data corruption from blocking reads

## Security Considerations

### Input Validation

**Sanitization**:
- Filter injection prevention (escape quotes and backslashes)
- Length limits on all text fields
- Bounds checking on numeric fields
- Format validation (semantic versioning, UUIDs)

**Validation Layer**:
- Service boundary validation (pkg/validation)
- MCP tool input validation (pkg/mcp/skills_tools.go)
- Consistent validation across all entry points

### Filter Injection Prevention

**Vulnerability**:
```go
// VULNERABLE (before fix)
filter := fmt.Sprintf(`category == "%s"`, userInput)
// userInput = `" or 1==1 or category == "`
// Result: category == "" or 1==1 or category == ""
```

**Mitigation**:
```go
func sanitizeFilterValue(value string) string {
    value = strings.ReplaceAll(value, "\\", "\\\\")
    value = strings.ReplaceAll(value, "\"", "\\\"")
    return value
}

filter := fmt.Sprintf(`category == "%s"`, sanitizeFilterValue(userInput))
```

### No Credential Exposure

**Rules**:
- No credentials in skill content
- No credentials in metadata
- Redaction of sensitive patterns in logs
- API keys stored separately (pkg/security/redact.go)

### Database Isolation

**Security Model**:
- Skills stored in `shared` database (isolated from project data)
- No cross-contamination with checkpoints or research
- Physical database boundary prevents unauthorized access
- No project_id field eliminates filter injection risk

## Testing Requirements

### Coverage Targets

- **Overall Package**: ≥ 80% code coverage
- **Service Layer**: ≥ 90% code coverage
- **Critical Paths**: 100% code coverage
  - Create with embedding generation
  - Search with filters
  - Update with re-embedding
  - RecordUsage statistics calculation
  - Filter sanitization

### Test Types

**Unit Tests**:
- Service methods with mocked dependencies
- Filter sanitization
- Validation logic
- Conversion functions (convertSearchResults)
- Statistics calculations

**Integration Tests**:
- End-to-end with real vector store (test Qdrant instance)
- Embedding generation with mock or test service
- Multi-step workflows (create → search → apply → update)
- Concurrent operations (Update and RecordUsage mutex protection)

**Edge Case Tests**:
- Empty results
- Maximum field lengths
- Concurrent updates
- Re-embedding triggers
- Filter injection attempts
- Invalid UUIDs
- Missing required fields

### Test Files

**Location**: `pkg/skills/`
- `service_test.go` - Service method unit tests
- `service_method_test.go` - Method-specific detailed tests
- `edge_cases_test.go` - Edge cases and error conditions
- `mocks_test.go` - Mock implementations for testing

**Test Naming**:
```go
func TestService_Create_ValidInput_Success(t *testing.T)
func TestService_Search_WithCategoryFilter_ReturnsFiltered(t *testing.T)
func TestService_Update_ContentChanged_ReembedsTriggers(t *testing.T)
func TestSanitizeFilterValue_InjectionAttempt_EscapesQuotes(t *testing.T)
```

## Usage Examples

### Example 1: Create a Skill

```go
import (
    "context"
    "github.com/axyzlabs/contextd/pkg/skills"
    "github.com/axyzlabs/contextd/pkg/validation"
)

// Create skills service
svc, err := skills.NewService(vectorStore, embedder)
if err != nil {
    return err
}

// Create skill
req := &validation.CreateSkillRequest{
    Name:        "Debugging Go Race Conditions",
    Description: "Systematic approach to finding and fixing Go race conditions",
    Content: `# Debugging Go Race Conditions

## Prerequisites
- Go toolchain installed
- Familiarity with goroutines
- Understanding of shared memory

## Steps
1. Run tests with -race flag
2. Identify data races in output
3. Add mutex or channel synchronization
4. Re-run tests to verify fix

## Common Patterns
- Shared map access without mutex
- Channel closes without synchronization
- WaitGroup counter races
`,
    Version:         "1.0.0",
    Author:          "Claude Code",
    Category:        "debugging",
    Tags:            []string{"golang", "concurrency", "race-conditions"},
    Prerequisites:   []string{"Go toolchain", "test suite"},
    ExpectedOutcome: "Identify and fix all race conditions in Go code",
    Metadata: map[string]string{
        "language": "go",
        "difficulty": "intermediate",
    },
}

skill, err := svc.Create(context.Background(), req)
if err != nil {
    return err
}

fmt.Printf("Created skill %s (ID: %s)\n", skill.Name, skill.ID)
```

### Example 2: Search for Skills

```go
// Search for debugging skills
searchReq := &validation.SearchSkillsRequest{
    Query:    "how to debug race conditions in golang",
    TopK:     5,
    Category: "debugging",
    Tags:     []string{"golang", "concurrency"},
}

results, err := svc.Search(context.Background(), searchReq)
if err != nil {
    return err
}

for _, result := range results.Results {
    fmt.Printf("Found: %s (score: %.2f)\n", result.Skill.Name, result.Score)
    fmt.Printf("  Prerequisites: %v\n", result.Skill.Prerequisites)
    fmt.Printf("  Expected: %s\n", result.Skill.ExpectedOutcome)
    fmt.Printf("  Usage: %d times, %.1f%% success\n",
        result.Skill.UsageCount,
        result.Skill.SuccessRate*100)
}
```

### Example 3: Apply a Skill

```go
// Get skill by ID
skill, err := svc.GetByID(context.Background(), "skill-uuid-here")
if err != nil {
    return err
}

// Display to agent
fmt.Printf("# %s\n\n", skill.Name)
fmt.Printf("%s\n\n", skill.Content)
fmt.Printf("Prerequisites: %v\n", skill.Prerequisites)
fmt.Printf("Expected: %s\n", skill.ExpectedOutcome)

// Agent applies skill...

// Record success
success := true // or false if skill didn't work
err = svc.RecordUsage(context.Background(), skill.ID, success)
if err != nil {
    // Log but don't fail - usage tracking is best-effort
    log.Warn("Failed to record usage", "error", err)
}
```

### Example 4: Update a Skill

```go
// Update skill version and content
newContent := "Updated content with additional examples..."
newVersion := "1.1.0"

fields := &skills.UpdateFields{
    Content: &newContent,
    Version: &newVersion,
}

updated, err := svc.Update(context.Background(), "skill-uuid-here", fields)
if err != nil {
    return err
}

fmt.Printf("Updated skill to version %s\n", updated.Version)
```

### Example 5: List Skills with Pagination

```go
// List debugging skills sorted by success rate
listReq := &validation.ListSkillsRequest{
    Limit:    20,
    Offset:   0,
    Category: "debugging",
    SortBy:   "success_rate",
}

result, err := svc.List(context.Background(), listReq)
if err != nil {
    return err
}

fmt.Printf("Found %d skills (showing %d-%d)\n",
    result.Total,
    result.Offset+1,
    result.Offset+len(result.Skills))

for _, skill := range result.Skills {
    fmt.Printf("- %s (v%s): %.1f%% success rate\n",
        skill.Name,
        skill.Version,
        skill.SuccessRate*100)
}
```

## Performance Benchmarks

### Target Metrics

| Operation | Target | Notes |
|-----------|--------|-------|
| Create | <500ms | Includes embedding generation |
| Search | <200ms | Cold (with embedding generation) |
| Search (cached) | <100ms | Warm (cached embedding) |
| List | <200ms | With pagination |
| Update (no re-embed) | <300ms | Read + delete + insert |
| Update (re-embed) | <700ms | Includes embedding generation |
| GetByID | <50ms | Direct ID lookup |
| RecordUsage | <100ms | Mutex-protected update |
| Delete | <100ms | Filter-based deletion |

### Scalability

| Metric | Value | Notes |
|--------|-------|-------|
| Max skills | 100,000+ | Vector database capacity |
| Search latency | <100ms | For 100K skills |
| Concurrent requests | 100+ | Service thread-safe |
| Embedding cache hit rate | 70-90% | For repeated searches |
| Update throughput | 10-20 ops/sec | Limited by mutex |

## Future Enhancements

### Phase 1 (MVP) - Complete
- ✅ Core CRUD operations
- ✅ Semantic search with filters
- ✅ Usage tracking and analytics
- ✅ Multi-tenant isolation (shared database)
- ✅ MCP tool integration

### Phase 2 (Planned)
- [ ] Skill versions (full version history)
- [ ] Skill dependencies (prerequisite skills)
- [ ] Skill ratings (user feedback)
- [ ] Advanced analytics (trend analysis, recommendation engine)
- [ ] Skill composition (combine multiple skills)

### Phase 3 (Future)
- [ ] Collaborative filtering (recommend based on similar users)
- [ ] Skill templates (parameterized skills)
- [ ] Skill marketplace (community sharing)
- [ ] Automated skill generation (from successful workflows)
- [ ] Skill testing framework (validate effectiveness)

## MCP Tool Implementation Details

### skill_create Tool

The `skill_create` MCP tool exposes contextd's reusable workflow skill creation through the Model Context Protocol. It validates an author-provided skill definition, generates an embedding, and persists the document in the vector store so it can later be searched and applied by other agents.

#### Current Implementation

**Entry point and lifecycle:**
1. `pkg/mcp/skills_tools.go:handleSkillCreate` is invoked by the MCP server when the `skill_create` tool is called.
2. The handler immediately wraps the request context in a 30s deadline using the shared `DefaultToolTimeout` constant.
3. Input validation is performed locally via helper functions for name, description, content, and tags. Validation errors are surfaced to the client as MCP validation errors and recorded in OpenTelemetry spans.
4. The handler builds a `validation.CreateSkillRequest` and delegates to the `pkg/skills.Service.Create` method.
5. Service responses are mapped back to the tool output (ID, name, version, token count, timestamps). All errors bubble up as MCP internal or timeout errors, and the span is finished with success/failure metrics.

**Service orchestration (pkg/skills/service.go):**
1. `Service.Create` starts a traced operation and records duration metrics.
2. A UUID is generated for the new skill; the ID is attached to the span.
3. Embedding input is assembled by concatenating the name, description, and content text blocks.
4. The service calls `EmbeddingGenerator.Embed(ctx, embeddingContent)`. The context carries the 30s tool deadline, so time spent inside the embedding API consumes the same budget.
5. Embedding metadata (tokens, cost, cache hit) is attached to the span and the latency histogram `skills.embedding.duration` is updated.
7. The service returns a domain `Skill` struct populated from the payload, including the embedding token usage.

**Skill provenance and local persistence:**
- Skill authors rely on their own context when crafting procedures. The system does not help capture references (links, scraped documents) that justify the skill's guidance.
- There is no verification workflow to confirm that a skill follows the "Demonstrate, practice, evaluate" loop recommended in the best-practices guide, nor is there a mechanism for reviewers to approve or reject a skill before it becomes searchable.

**Error propagation:**
- Validation failures are caught before the service call.
- Any error from the embedding generator, database setup, or insert operation is surfaced as a wrapped Go `error`, which the MCP layer converts into an MCP internal error.
- If the embedding call exceeds the 30s deadline, the context is cancelled, `Embed` returns with `context.DeadlineExceeded`, and the handler reports a tool timeout to the caller.

#### Known Issues

**Embedding Timeouts:**
- The single shared `DefaultToolTimeout` (30s) must cover validation, embedding, database setup, and persistence; long-running embedding requests or slow vector-store inserts can therefore exhaust the budget.
- Large skill payloads (long procedures) expand the prompt text, increasing embedding latency and token usage.
- When a timeout is hit, the caller receives an MCP timeout error but has no visibility into partial progress (e.g., whether the skill was stored without an embedding) or guidance on retry strategy. The system also lacks retry semantics.

**Knowledge Capture:**
- Research inputs, scraped references, and provenance are not recorded, so skill quality is difficult to audit or reuse across collaborators.

**Lifecycle Governance:**
- There is no enforced review, evaluation, or versioning discipline to ensure skills remain accurate and aligned with the target project.

**Operational Resilience:**
- Timeout behaviour, validation guardrails, and observability are insufficient for production-grade automation.

## Best-Practice Aligned Roadmap

The Claude [Agent Skill Best Practices](https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices) outline a lifecycle of **understand → research → structure → validate → share & maintain**. The `skill_create` experience must cover every phase so that each new skill is trustworthy, reviewable, and reusable by collaborators without vector-store access.

### Phase 1: Understand & scope the skill request

- Extend the MCP schema to require problem statement, success criteria, target persona, and environment prerequisites. Reject submissions that do not provide this framing.
- Capture "why this skill is needed" and any user constraints (SLAs, tools, permissions) so downstream automation understands applicability.

**Implementation**: Issue #113

### Phase 2: Research & gather authoritative knowledge

- Automatically invoke the Research Agent before embedding to obtain topical references (official docs, runbooks, incident reports). Store prompts, result rankings, and analyst notes to document provenance.
- Require at least one validated reference before the skill can proceed to the drafting step; flag knowledge gaps to the author within the MCP response.
- For each accepted reference, call the scrape tool to archive the source material under a deterministic repository path (see layout below). Include metadata (URL, retrieval timestamp, checksum, license) to support future audits and drift detection.

**Implementation**: Issue #114

### Phase 3: Draft & structure the procedure

- Provide a templated skill skeleton that enforces the best-practice sections: prerequisites, numbered steps, safety checks, failure handling, and follow-up verification. Validate conformance prior to embedding to avoid malformed content entering the vector store.
- Require explicit test or evaluation steps ("Demonstrate, practice, evaluate"), including expected outputs and rollback/abort instructions for risky actions.
- Support optional attachments (tables, command snippets) referenced by scraped documents so the skill remains self-contained.

**Implementation**: Issue #115

### Phase 4: Generate embeddings with resiliency guardrails

- Introduce per-tool timeout configuration and raise the create deadline (e.g., 90s) separate from read-oriented tools. Surface the defaults from configuration files and allow environment variables to override them at runtime so operators can tune behaviour without redeploying.
- Support multiple embedding providers with distinct SLAs by making the provider selection and timeout profile configurable. Document recommended settings for local (low-latency) vs. remote (higher-latency) providers and enforce provider-specific retries.
- Layer an internal embedding timeout so the MCP request can report partial progress or initiate background work if exceeded. Because asynchronous skill completion is acceptable for MVP, provide an opt-in background job that finishes embedding generation and repository writes after the initial request acknowledges receipt. Surface job status through follow-up queries or notifications so authors can track completion.
- Enforce content length limits and provide pre-embedding chunking for long procedures. Surface actionable validation feedback when limits are hit.
- Add retry with exponential backoff for transient provider failures while observing the configured deadline. Emit structured telemetry (latency, retries, payload size) for operational monitoring.

**Implementation**: Issue #116

### Phase 5: Persist knowledge for collaborators

- `skill.yaml` manifest capturing metadata, structured instructions, evaluation status, and embeddings summary (dimension, provider, token usage).
- `README.md` summarising the skill intent, review history, and maintenance owners.
- `references.json` listing Research Agent queries/responses, curated citations, and mapping to scraped files.
- `scraped/` directory containing sanitized documents from the scrape tool.

Ensure the write is transactional with the vector-store insert: if either fails, roll back and surface a recoverable error to avoid drift between storage locations.

**Repository layout:**
```
.claude/skills/
  <skill-id>/
    skill.yaml        # canonical manifest, versioned with repo
    README.md         # operator-facing summary, evaluation log, owners
    references.json   # Research Agent queries + result metadata
    scraped/
      <slug>.html     # sanitized scrape output
      <slug>.meta.json
```

Paths are deterministic to align with the best-practice emphasis on transparent, reviewable skills that teammates can reuse without privileged infrastructure access.

Adopt a retention policy that keeps the latest published skill version and the most recent two historical revisions per skill directory. Older revisions can be pruned during scheduled maintenance to prevent repository bloat while still supporting audits and rollbacks.

**Implementation**: Issue #117

### Phase 6: Review, evaluate, and publish

- Introduce workflow states (`draft`, `in_review`, `published`, `retired`). New skills start as `draft` and must pass reviewer approval before being searchable.
- Build a review checklist derived from the best practices (clear scope, up-to-date references, tested instructions, risk mitigations). Capture approver identity and comments in the manifest.
- Execute evaluation harnesses (automated smoke tests or recorded manual dry runs). Store results, transcripts, and follow-up actions in the manifest and README.
- Only transition to `published` after successful evaluation; optionally keep a `sandbox_only` flag for experimental skills.

**Implementation**: Issue #118

### Phase 7: Maintain & evolve shared knowledge

- Track ownership, SLAs, and review cadence within the manifest. Emit alerts when a skill approaches its review deadline or when scraped reference checksums change, signalling potential drift.
- Provide a lightweight `skill_update` companion flow to iterate on instructions while preserving version history. Store prior versions in the repository (e.g., `history/` folder) to satisfy the best-practice recommendation of learning from past changes.
- Record deprecation rationale when a skill is retired and ensure vector-store entries reflect the archival status. Persist retired versions in `.claude/skills/<skill-id>/history/` subject to the retention policy.

**Implementation**: Issue #119

### Phase 8: Evaluate with repeatable harnesses

Research outcome: recorded transcripts paired with scripted smoke tests best reinforce the "demonstrate-practice-evaluate" loop. Simulated runs provide deterministic regression coverage, while transcripts capture nuanced human judgement for complex procedures.

Implement a two-layer harness:
1. **Scenario simulator** – run deterministic, containerised workflows (e.g., mock API calls, CLI invocations with fixtures) defined in `tests/skills/<skill-id>/`. Use `go test` or `pytest` depending on the skill language footprint. Track success/failure in the manifest and block publication on failure.
2. **Transcript recorder** – store manual or semi-automated dry-run transcripts (console logs, screen captures, or agent chat history) under `.claude/skills/<skill-id>/evaluations/`. Include metadata linking the transcript to the scenario, operator, and date.

Automate regression checks by scheduling the simulator against the latest scraped artifacts. When reference checksums change, rerun the simulator and notify maintainers if failures occur. Use CI workflows to execute the harness on pull requests that modify skill manifests or scraped content.

Expose evaluation status in MCP responses so authors know whether additional practice runs or transcript uploads are required before publication.

**Implementation**: Issue #120

### Integration Testing

End-to-end testing of the complete skills lifecycle from creation through evaluation and publication.

**Implementation**: Issue #121

## Implementation Strategy

**Sequential Dependencies:**
1. **Foundation First**: Complete Phases 1-3 before starting Phase 4
2. **Core Features**: Complete Phases 4-6 before starting Phase 7
3. **Advanced Last**: Complete Phases 7-8 sequentially
4. **Integration Final**: All phases must complete before E2E testing

**Key Milestones:**
- Phase 3 Complete → Repository artifacts testable
- Phase 6 Complete → Complete lifecycle operational
- Phase 8 Complete → Production-ready
- Integration Complete → Release candidate

## Open Questions

1. How do we expose provider selection and SLA metadata to authors within the MCP schema so they can anticipate latency and cost trade-offs?

## Related Documentation

- **Claude Agent Skill Best Practices**: https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices
- **Architecture**: `docs/standards/architecture.md`
- **Coding Standards**: `docs/standards/coding-standards.md`
- **Testing Standards**: `docs/standards/testing-standards.md`
- **Multi-Tenant Architecture**: `docs/adr/002-universal-multi-tenant-architecture.md`
- **Package Guidelines**: `docs/standards/package-guidelines.md`
- **MCP Integration**: `pkg/mcp/README.md`
- **Vector Store**: `pkg/vectorstore/README.md`
- **Embedding Service**: `pkg/embedding/README.md`
- **Current Implementation**: `pkg/mcp/skills_tools.go`, `pkg/skills/service.go`

## GitHub Tracking

**GitHub Issue**: #112
**Feature Branch**: `feature/112-skills-tool-best-practices`
**Pull Request**: #129
**Status**: In Progress
**Created**: 2025-11-04

### Implementation Issues (8 Phases + Integration)

**Phase 1-3 (Foundation)**:
- #113 - Phase 1: Extend MCP Schema for Skill Context
- #114 - Phase 2: Integrate Research Agent
- #115 - Phase 3: Implement Skill Template Skeleton

**Phase 4-6 (Core Features)**:
- #116 - Phase 4: Implement Configurable Embedding Timeouts
- #117 - Phase 5: Implement Repository Artifact Persistence
- #118 - Phase 6: Implement Workflow State Machine

**Phase 7-8 (Advanced)**:
- #119 - Phase 7: Implement Skill Maintenance and Evolution
- #120 - Phase 8: Implement Evaluation Harnesses

**Integration**:
- #121 - Integration: End-to-End Skills Tool Testing

## Changelog

### 0.9.0-rc-1 (Planned - Issue #112)
- **Best Practice Alignment**: Complete 8-phase implementation
- Extended MCP schema with problem statement and success criteria
- Research Agent integration for knowledge gathering
- Repository artifacts as canonical source (.claude/skills/)
- Workflow state machine (draft → in_review → published → retired)
- Configurable embedding timeouts and retry logic
- Evaluation harnesses (scenario simulator + transcript recorder)
- Ownership tracking and version history
- Reference scraping and provenance tracking

### v2.1.0 (2025-11-05)
- **Skill Creation Methodology**: Added comprehensive TDD-based skill authoring patterns from superpowers plugin
- **RED-GREEN-REFACTOR Cycle**: Implemented testing methodology for skill creation
- **Pressure Testing**: Added subagent testing patterns for realistic scenario validation
- **Rationalization Tables**: Added loophole closing patterns with explicit counters
- **Claude Search Optimization**: Added CSO patterns for skill discoverability
- **Skill Structure Guidelines**: Added comprehensive frontmatter and content structure requirements
- **Credits and Attribution**: Added acknowledgment of superpowers plugin (@dmarx) for methodology patterns

### v2.0.0 (2025-11-04)
- Multi-tenant mode made mandatory (legacy mode removed)
- Skills stored in `shared` database (global knowledge)
- Filter injection vulnerability fixed
- Usage tracking enhanced with success rate calculation

### v1.0.0 (2025-10-01)
- Initial implementation
- Core CRUD operations
- Semantic search with filters
- Usage tracking
- MCP tool integration
