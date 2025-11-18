# Checkpoint Requirements

**Parent**: [../SPEC.md](../SPEC.md)

## Functional Requirements

### Core Capabilities

1. **Checkpoint Creation** - Save session snapshots with automatic embedding generation
2. **Semantic Search** - Find relevant checkpoints using natural language queries
3. **Paginated Listing** - Browse recent checkpoints with filtering
4. **CRUD Operations** - Get, update, and delete checkpoints by ID
5. **Token Tracking** - Automatic token counting for cost tracking
6. **Health Monitoring** - OpenTelemetry instrumentation for all operations

### Use Cases

- **Context Recovery**: Resume work after extended breaks by searching past session summaries
- **Knowledge Reuse**: Find similar solutions from previous sessions ("how did I solve X?")
- **Session Boundaries**: Mark completion of significant work phases for portfolio tracking
- **Team Collaboration**: Share session knowledge across team members (future: shared checkpoints)
- **Workflow Automation**: Trigger actions based on checkpoint events (future: webhooks)

## Non-Functional Requirements

### Performance

**Latency Targets**:
- Create operation: 55-160ms typical
- Search operation: 60-270ms typical
- List operation: <50ms (small), 100-500ms (large)
- Get by ID: <15ms typical

**Throughput Targets**:
- Create: 100-200 req/s (TEI), 20-50 req/s (OpenAI)
- Search: 100-200 req/s (cached), 20-50 req/s (uncached)
- List: 50-100 req/s (small datasets)
- Get: 500-1000 req/s

**Memory Constraints**:
- Per checkpoint: ~7KB (OpenAI), ~2.5KB (TEI)
- Dataset size: 1K checkpoints = ~7MB (OpenAI)

### Security

**Multi-Tenant Isolation**:
- Database-per-project physical isolation (mandatory)
- No cross-project data leakage
- Filter injection attacks eliminated by design

**Input Validation**:
- Summary: 1-500 characters, required
- Description: 0-10,000 characters, optional
- Project path: Absolute path, must exist, no path traversal
- Context: Max 50 key-value pairs
- Tags: Max 20 tags, each max 50 characters

**Data Security**:
- Sensitive data redaction (API keys, passwords, tokens)
- No encryption at rest (rely on filesystem encryption)
- No encryption in transit (MVP, add TLS via reverse proxy for production)

### Reliability

**Error Handling**:
- Validation errors: Return error (do not retry)
- Timeout errors: Retry with increased timeout
- Rate limit errors: Retry with exponential backoff
- Network errors: Retry up to 3 times

**Observability**:
- All operations instrumented with OpenTelemetry
- Error metrics by type and operation
- Latency histograms for performance monitoring

### Scalability

**Current Architecture** (single-user):
- Single-user localhost service
- Concurrent requests via goroutines
- Local Qdrant storage

**Future Multi-User**:
- JWT authentication with user claims
- User-specific databases
- TLS via reverse proxy
- Per-user rate limits

## Data Model Requirements

### Checkpoint Entity

**Fields**:
- `ID`: UUID v4 (36 chars with hyphens)
- `Summary`: Required, 1-500 characters
- `Description`: Optional, max 10,000 characters
- `ProjectPath`: Required, absolute path, must exist
- `Context`: Optional, max 50 key-value pairs
- `Tags`: Optional, max 20 tags, each max 50 characters
- `TokenCount`: Auto-calculated, read-only
- `CreatedAt`: Auto-generated, read-only
- `UpdatedAt`: Auto-generated, read-only

### Vector Storage

**Collection**: `checkpoints`
**Vector Dimension**: 1536 (OpenAI) or 384 (TEI)
**Distance Metric**: Cosine similarity

**Indexes**:
- Primary: `id` (unique)
- Vector: HNSW index on `embedding` field
- Metadata: No secondary indexes

## Rate Limiting Requirements

**Default Limits**:
- `checkpoint_save`: 10 req/min, burst 20
- `checkpoint_search`: 20 req/min, burst 40
- `checkpoint_list`: 20 req/min, burst 40

**Enforcement**:
- Token bucket algorithm
- Per-project rate limiting
- HTTP 429 (Too Many Requests) on limit exceeded
- Retry-After header indicates wait time

## Testing Requirements

### Coverage Targets

- **Overall**: ≥80% line coverage
- **Critical Paths**: 100% coverage
  - Create workflow (embedding + database)
  - Search workflow (embedding + vector search)
  - Multi-tenant database scoping

### Test Types

1. **Unit Tests** - Service logic with mocks
2. **Integration Tests** - End-to-end workflows with real dependencies
3. **Performance Tests** - Latency benchmarks, throughput tests
4. **Regression Tests** - Bug fixes with automated tests
