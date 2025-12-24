# Conversation Indexing Schema

**Related Documents:**
- [SPEC.md](SPEC.md) - Requirements and success criteria
- [DESIGN.md](DESIGN.md) - Architecture and components
- [CONFIG.md](CONFIG.md) - Configuration reference

## Collection Naming

```
{tenant}_{project}_conversations
```

**Examples:**
- `dahendel_contextd_conversations`
- `dahendel_gitops_conversations`

Each project maintains a separate collection. Conversations from different projects never mix.

## Document Types

The collection stores three document types:

| Type | Purpose |
|------|---------|
| `message` | Individual user or assistant message |
| `decision` | Extracted decision from conversation |
| `summary` | Session summary (future enhancement) |

## Document Schema

### Base Fields (All Types)

```go
type ConversationDocument struct {
    // Identity
    ID        string    `json:"id"`         // UUID
    SessionID string    `json:"session_id"` // Claude Code session UUID
    Type      string    `json:"type"`       // "message", "decision", "summary"
    Timestamp time.Time `json:"timestamp"`

    // Content (embedded for search)
    Content string `json:"content"` // Scrubbed text

    // Context tags
    Tags   []string `json:"tags"`             // ["kubernetes", "debugging", "golang"]
    Domain string   `json:"domain,omitempty"` // "infrastructure", "backend", "frontend"

    // Cross-references
    FilesDiscussed []FileReference   `json:"files_discussed,omitempty"`
    CommitsMade    []CommitReference `json:"commits_made,omitempty"`

    // Metadata
    IndexedAt        time.Time `json:"indexed_at"`
    ExtractionMethod string    `json:"extraction_method"` // "heuristic", "llm"
}
```

### Message Fields

```go
type MessageDocument struct {
    ConversationDocument

    // Message-specific
    Role         string `json:"role"`          // "user" or "assistant"
    MessageUUID  string `json:"message_uuid"`  // Original message UUID
    MessageIndex int    `json:"message_index"` // Position in session
}
```

### Decision Fields

```go
type DecisionDocument struct {
    ConversationDocument

    // Decision-specific
    Summary      string   `json:"summary"`                 // One-line decision summary
    Alternatives []string `json:"alternatives,omitempty"`  // Options considered
    Reasoning    string   `json:"reasoning,omitempty"`     // Why this was chosen
    Confidence   float64  `json:"confidence"`              // Extraction confidence

    // Source reference
    SourceMessageUUID string `json:"source_message_uuid"` // Message containing decision
}
```

## Reference Types

### FileReference

```go
type FileReference struct {
    Path       string   `json:"path"`                  // File path
    LineRanges []string `json:"line_ranges,omitempty"` // ["10-25", "100"]
    Action     string   `json:"action"`                // "read", "edited", "created", "deleted"
}
```

### CommitReference

```go
type CommitReference struct {
    SHA     string `json:"sha"`               // Short or full SHA
    Message string `json:"message,omitempty"` // Commit message if available
}
```

## Index Metadata

Store index state in a separate metadata document:

```go
type ConversationIndexMetadata struct {
    ProjectPath      string    `json:"project_path"`
    TenantID         string    `json:"tenant_id"`
    LastIndexedAt    time.Time `json:"last_indexed_at"`
    SessionsIndexed  int       `json:"sessions_indexed"`
    MessagesIndexed  int       `json:"messages_indexed"`
    DecisionsIndexed int       `json:"decisions_indexed"`

    // Track indexed sessions to detect new ones
    IndexedSessionIDs []string `json:"indexed_session_ids"`
}
```

## Vector Embedding

Embed the `Content` field for semantic search. Additional fields serve as metadata filters.

**Searchable by vector similarity:**
- Message content
- Decision summaries

**Filterable by metadata:**
- `type` (message, decision, summary)
- `tags` (array contains)
- `domain` (exact match)
- `session_id` (exact match)
- `files_discussed.path` (exact match)
- `timestamp` (range)

## Example Documents

### Message Document

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "session_id": "10b1b621-d586-4a94-985b-e673f931439b",
  "type": "message",
  "timestamp": "2025-12-04T16:58:17Z",
  "content": "Let's use chromem instead of Qdrant for the embedded vectorstore. It simplifies deployment since there's no external dependency.",
  "role": "assistant",
  "message_uuid": "ed240f6b-6648-40fe-bb13-2957b88f7a68",
  "message_index": 42,
  "tags": ["architecture", "vectorstore", "golang"],
  "domain": "backend",
  "files_discussed": [
    {"path": "internal/vectorstore/chromem.go", "action": "created"},
    {"path": "internal/vectorstore/factory.go", "action": "edited"}
  ],
  "commits_made": [],
  "indexed_at": "2025-12-09T10:00:00Z",
  "extraction_method": "heuristic"
}
```

### Decision Document

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440002",
  "session_id": "10b1b621-d586-4a94-985b-e673f931439b",
  "type": "decision",
  "timestamp": "2025-12-04T16:58:17Z",
  "content": "Chose chromem over Qdrant for embedded deployment simplicity",
  "summary": "Use chromem as default vectorstore instead of Qdrant",
  "alternatives": ["Qdrant", "Milvus", "Weaviate"],
  "reasoning": "Chromem embeds directly in the Go binary with no external service dependency. Simplifies deployment for single-user Claude Code scenarios.",
  "confidence": 0.9,
  "source_message_uuid": "ed240f6b-6648-40fe-bb13-2957b88f7a68",
  "tags": ["architecture", "vectorstore", "deployment"],
  "domain": "backend",
  "files_discussed": [
    {"path": "internal/vectorstore/chromem.go", "action": "created"}
  ],
  "commits_made": [
    {"sha": "abc123", "message": "feat: add chromem vectorstore implementation"}
  ],
  "indexed_at": "2025-12-09T10:00:00Z",
  "extraction_method": "llm"
}
```

## Migration Considerations

### Adding New Fields

New fields default to empty/zero values. Queries handle missing fields gracefully.

### Reindexing

Use `force: true` in IndexOptions to reindex existing sessions. The system deletes old documents for a session before reindexing.

### Retention

Future enhancement: configurable retention policy to remove old conversation documents. Decisions may persist longer than raw messages.
