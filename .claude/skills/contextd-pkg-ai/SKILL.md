---
name: contextd-pkg-ai
description: Use when working with AI/embedding packages (pkg/embedding, pkg/search, pkg/semantic) or implementing vector operations, semantic search, or embedding generation - enforces provider abstraction, L2 normalization, hybrid search, privacy protection, and proper error handling for AI integrations
---

# AI Package Development (contextd:pkg-ai)

## Overview

Enforces patterns for embedding generation, semantic search, and AI integrations in contextd. Core principle: **AI operations must be abstracted, normalized, privacy-safe, and production-ready.**

## When to Use This Skill

Use when:
- Creating or modifying pkg/embedding (embedding generation)
- Creating or modifying pkg/search (semantic search operations)
- Creating or modifying pkg/semantic (semantic analysis/ranking)
- Implementing vector operations (normalization, similarity)
- Integrating external AI APIs (OpenAI, TEI)
- Adding hybrid search (semantic + keyword)

Do NOT use for:
- Non-AI packages (use appropriate pkg-* skill)
- Application logic (use golang-pro skill)

## Mandatory Architecture Patterns

### 1. Provider Abstraction (REQUIRED)

**ALWAYS create provider interface**, even for single implementation.

```go
// ✅ GOOD - Provider abstraction
type EmbeddingProvider interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
    IsExternal() bool
}

type OpenAIProvider struct {
    apiKey  string
    client  *http.Client
    model   string
}

type TEIProvider struct {
    baseURL string
    client  *http.Client
    model   string
}

// ❌ WRONG - Direct implementation, no abstraction
func Embed(text string) ([]float32, error) {
    // Hardcoded OpenAI API call
}
```

**Interface Alone Is Not Enough**

Services MUST accept interface parameter, NOT hardcode concrete type:

```go
// ✅ GOOD - Service accepts interface
func NewCheckpointService(embedder EmbeddingProvider) *Service {
    return &Service{embedder: embedder}
}

// ❌ WRONG - Service hardcodes concrete type
func NewCheckpointService() *Service {
    embedder := embedding.NewOpenAIProvider(...) // Hardcoded!
    return &Service{embedder: embedder}
}
```

**Why abstraction is REQUIRED:**
- contextd supports both TEI (local) and OpenAI (API)
- Privacy: sensitive data uses TEI, non-sensitive can use OpenAI
- Testing: mockable interface
- Future: easy to add new providers

**Rationalization Counter:**

| Excuse | Reality |
|--------|---------|
| "YAGNI - we only use OpenAI" | contextd requires TEI AND OpenAI support (architecture decision) |
| "Abstraction adds complexity" | Interface is 4 methods, provides massive testing/flexibility benefit |
| "We can refactor later" | Later = breaking change across codebase. Do it now. |
| "Interface exists, good enough" | Interface must be USED. Services must accept it as parameter. |

### 2. L2 Normalization (REQUIRED)

**ALWAYS normalize vectors** before storage AND before search. Use L2 (Euclidean) normalization.

```go
// ✅ GOOD - Correct L2 normalization
func normalize(vector []float32) []float32 {
    var sumSquares float32
    for _, v := range vector {
        sumSquares += v * v
    }
    magnitude := float32(math.Sqrt(float64(sumSquares)))

    if magnitude == 0 {
        return vector // Zero vector, return as-is
    }

    normalized := make([]float32, len(vector))
    for i, v := range vector {
        normalized[i] = v / magnitude
    }
    return normalized
}

// ❌ WRONG - Dividing by sum (NOT L2 normalization)
func normalize(vector []float32) []float32 {
    var sum float32
    for _, v := range vector {
        sum += v // WRONG: should be v * v
    }
    for i := range vector {
        vector[i] = vector[i] / sum
    }
    return vector
}
```

**Normalize ALL Embeddings**

Normalize BOTH document embeddings (at index time) AND query embeddings (at search time):

```go
// ✅ GOOD - Normalize both
func (s *IndexService) Index(doc Document) error {
    vector, _ := s.embedder.Embed(ctx, doc.Text)
    normalized := normalize(vector) // Normalize document
    return s.store.Upsert(ctx, normalized)
}

func (s *SearchService) Search(query string) ([]Result, error) {
    vector, _ := s.embedder.Embed(ctx, query)
    normalized := normalize(vector) // Normalize query
    return s.store.Search(ctx, normalized)
}

// ❌ WRONG - Only normalize documents, not queries
func (s *SearchService) Search(query string) ([]Result, error) {
    vector, _ := s.embedder.Embed(ctx, query)
    return s.store.Search(ctx, vector) // Missing normalization!
}
```

If query is not normalized but documents are, cosine similarity breaks.

**Test normalization produces unit vector:**

```go
func TestNormalize_UnitVector(t *testing.T) {
    vector := []float32{3.0, 4.0}
    normalized := normalize(vector)

    // Calculate magnitude (REQUIRED - must verify magnitude ≈ 1.0)
    var sumSquares float32
    for _, v := range normalized {
        sumSquares += v * v
    }
    magnitude := math.Sqrt(float64(sumSquares))

    // Must be 1.0 (unit vector)
    if math.Abs(magnitude-1.0) > 0.0001 {
        t.Errorf("Magnitude = %f, want 1.0", magnitude)
    }
}
```

**Tests WITHOUT magnitude validation do not satisfy requirement.**

**Rationalization Counter:**

| Excuse | Reality |
|--------|---------|
| "OpenAI already normalizes" | Not guaranteed, embeddings may change. Always normalize locally. |
| "Qdrant handles normalization" | Qdrant can normalize, but doing it client-side ensures consistency |
| "Stack Overflow code works" | Many SO examples use WRONG normalization (sum vs magnitude) |
| "Tests pass without it" | Tests without unit vector check don't validate correctness |
| "Normalize once at import" | Must normalize BOTH documents AND queries for cosine similarity |

### 3. Hybrid Search (REQUIRED)

**ALWAYS use hybrid search** (semantic + keyword) for production search.

**REQUIRED: 70% semantic, 30% keyword (EXACTLY)**

Not 90/10, not 50/50, not 99/1. The 70/30 ratio is empirically validated.

```go
// ✅ GOOD - Hybrid search (EXACTLY 70% semantic, 30% keyword)
func (s *SearchService) HybridSearch(ctx context.Context, query string, limit int) ([]Result, error) {
    // 70% semantic (EXACTLY)
    semanticLimit := int(float64(limit) * 0.7)
    semantic, err := s.semanticSearch(ctx, query, semanticLimit)
    if err != nil {
        return nil, fmt.Errorf("semantic search failed: %w", err)
    }

    // 30% keyword (EXACTLY)
    keywordLimit := int(float64(limit) * 0.3)
    keyword, err := s.keywordSearch(ctx, query, keywordLimit)
    if err != nil {
        return nil, fmt.Errorf("keyword search failed: %w", err)
    }

    // Merge, deduplicate, and rank
    return s.mergeResults(semantic, keyword, limit), nil
}

func (s *SearchService) mergeResults(semantic, keyword []Result, limit int) []Result {
    seen := make(map[string]bool)
    merged := []Result{}

    // Add semantic results first (higher weight)
    for _, r := range semantic {
        if !seen[r.ID] {
            seen[r.ID] = true
            merged = append(merged, r)
        }
    }

    // Add keyword results (deduplicated)
    for _, r := range keyword {
        if !seen[r.ID] {
            seen[r.ID] = true
            merged = append(merged, r)
        }
    }

    // Limit to requested count
    if len(merged) > limit {
        merged = merged[:limit]
    }

    return merged
}

// ❌ WRONG - Pure semantic search
func (s *SearchService) Search(ctx context.Context, query string, limit int) ([]Result, error) {
    vector, _ := s.embedder.Embed(ctx, query)
    return s.vectorStore.Search(ctx, vector, limit) // Missing keyword component
}

// ❌ WRONG - "Hybrid" with 99/1 split (effectively pure semantic)
semanticLimit := int(float64(limit) * 0.99) // Too high!
keywordLimit := int(float64(limit) * 0.01)  // Too low!
```

**Rationalization Counter:**

| Excuse | Reality |
|--------|---------|
| "Pure semantic is fast enough" | Recall matters more than speed. Hybrid improves recall 20-40%. |
| "Tests pass with semantic-only" | Tests don't measure recall. User experience degrades. |
| "Hybrid adds complexity" | 30 lines of code, massive quality improvement |
| "Keyword search is old tech" | Hybrid leverages both: semantic for concepts, keyword for exact matches |
| "90/10 is close enough" | No. 70/30 is validated ratio. Use exactly 70/30. |

### 4. Privacy Protection (CRITICAL)

**NEVER send sensitive data to external APIs** without sanitization.

**REQUIRED Sanitization Patterns (ALL must be implemented):**

```go
// ✅ GOOD - Comprehensive sanitization
func sanitizeError(msg string) string {
    // Redact emails (REQUIRED)
    msg = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`).
        ReplaceAllString(msg, "[EMAIL]")

    // Redact file paths (REQUIRED)
    msg = regexp.MustCompile(`/[a-zA-Z0-9_\-./]+`).ReplaceAllString(msg, "[PATH]")

    // Redact API keys - sk-, pk- patterns (REQUIRED)
    msg = regexp.MustCompile(`\b(sk|pk)[-_][a-zA-Z0-9]{20,}\b`).ReplaceAllString(msg, "[API_KEY]")

    // Redact IP addresses (REQUIRED)
    msg = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`).ReplaceAllString(msg, "[IP]")

    // Redact common username patterns (REQUIRED)
    msg = regexp.MustCompile(`user=\S+`).ReplaceAllString(msg, "user=[REDACTED]")
    msg = regexp.MustCompile(`@\w+`).ReplaceAllString(msg, "@[USER]")

    return msg
}

// ❌ WRONG - Only sanitizes email (incomplete)
func sanitizeError(msg string) string {
    return regexp.MustCompile(`[email pattern]`).ReplaceAllString(msg, "[EMAIL]")
    // Missing: paths, API keys, IPs, usernames
}
```

**Usage Pattern:**

```go
// ✅ GOOD - Sanitize before external API
func (s *TroubleshootService) EmbedError(ctx context.Context, errorMsg string) ([]float32, error) {
    // Sanitize ALWAYS for external providers
    if s.embedder.IsExternal() {
        sanitized := sanitizeError(errorMsg)
        return s.embedder.Embed(ctx, sanitized)
    }

    // Local provider (TEI): can use original
    return s.embedder.Embed(ctx, errorMsg)
}

// ❌ WRONG - Send raw error to external API
func (s *TroubleshootService) EmbedError(ctx context.Context, errorMsg string) ([]float32, error) {
    return s.embedder.Embed(ctx, errorMsg)
    // errorMsg might contain: "Auth failed for user john@acme.com with key sk-abc123..."
}
```

**Rationalization Counter:**

| Excuse | Reality |
|--------|---------|
| "Error messages don't contain PII" | They often do: emails, usernames, file paths with client names |
| "OpenAI doesn't store embeddings" | OpenAI's data policy can change. Sanitize anyway. |
| "It's just for internal use" | Compliance violations (GDPR, HIPAA) apply to internal data too |
| "Sanitizing email is enough" | Must sanitize ALL: emails, paths, API keys, IPs, usernames |

### 5. Timeouts and Error Handling (REQUIRED)

**ALWAYS set timeouts** for external API calls. **ALWAYS propagate context.**

```go
// ✅ GOOD - Context propagation, timeout, retry
func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    // Set 30s timeout for embedding call
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Retry with exponential backoff (3 attempts)
    var vector []float32
    err := retry.Do(
        func() error {
            var apiErr error
            vector, apiErr = p.callAPI(ctx, text)
            return apiErr
        },
        retry.Attempts(3),
        retry.Context(ctx),
        retry.Delay(500*time.Millisecond),
        retry.DelayType(retry.BackOffDelay),
    )

    if err != nil {
        return nil, fmt.Errorf("embedding failed after retries: %w", err)
    }

    // Normalize before return
    return normalize(vector), nil
}

// ❌ WRONG - No timeout, no context, no retry
func Embed(text string) ([]float32, error) {
    client := &http.Client{} // No timeout!
    resp, err := client.Post(url, body)
    if err != nil {
        return nil, err // Single attempt, no retry
    }
    // ...
}
```

**Rationalization Counter:**

| Excuse | Reality |
|--------|---------|
| "OpenAI API is fast" | Fast 99% of time. Timeout prevents 1% causing 60s hangs. |
| "Timeouts slow things down" | Timeouts PREVENT slowdowns. They're maximum wait, not added delay. |
| "Demo doesn't need context" | Production code needs proper patterns from day one. |
| "Retry adds complexity" | 5 lines with retry library. Prevents transient failure escalation. |

## Quick Reference: AI Package Checklist

**Before completing any AI package work:**

- [ ] **Provider abstraction**: Interface defined, multiple implementations
- [ ] **Dimensions validation**: Check expected dimensions (384 for BGE, 1536 for OpenAI)
- [ ] **L2 normalization**: Applied before storage, unit vector test exists
- [ ] **Hybrid search**: 70% semantic + 30% keyword + deduplication
- [ ] **Privacy protection**: Sanitize before external APIs, document what gets sent
- [ ] **Context propagation**: All functions take `ctx context.Context` as first param
- [ ] **Timeouts**: 30s for embeddings, appropriate for other operations
- [ ] **Retry logic**: Exponential backoff for transient failures
- [ ] **Error wrapping**: All errors wrapped with context
- [ ] **Godoc comments**: All exported functions documented

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Hardcoded OpenAI calls | Create EmbeddingProvider interface |
| Missing normalization | Add normalize() before return, test unit vector |
| Pure semantic search | Implement hybrid (70/30 semantic/keyword) |
| No dimension check | Validate vector length matches expected |
| No timeout | Use context.WithTimeout (30s for embeddings) |
| Sending PII to external API | Sanitize before external calls, use TEI for sensitive data |
| No deduplication | Merge results with seen map |
| Ignoring errors silently | Wrap and return all errors |

## Testing Requirements

**All AI packages MUST have:**

1. **Unit vector test** (normalization):
   ```go
   func TestNormalize_UnitVector(t *testing.T)
   ```

2. **Dimension consistency test**:
   ```go
   func TestEmbed_ConsistentDimensions(t *testing.T)
   ```

3. **Hybrid search quality test**:
   ```go
   func TestHybridSearch_BetterRecall(t *testing.T)
   ```

4. **Privacy sanitization test**:
   ```go
   func TestSanitize_RedactsPII(t *testing.T)
   ```

5. **Timeout test**:
   ```go
   func TestEmbed_RespectsTimeout(t *testing.T)
   ```

6. **Mock provider test** (validates abstraction):
   ```go
   func TestService_WithMockProvider(t *testing.T)
   ```

**Mock MUST Implement Full Interface:**

```go
// ✅ GOOD - Full interface implementation
type MockProvider struct {
    EmbedFunc      func(ctx context.Context, text string) ([]float32, error)
    EmbedBatchFunc func(ctx context.Context, texts []string) ([][]float32, error)
    DimensionsFunc func() int
    IsExternalFunc func() bool
}

func (m *MockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    if m.EmbedFunc != nil {
        return m.EmbedFunc(ctx, text)
    }
    return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    if m.EmbedBatchFunc != nil {
        return m.EmbedBatchFunc(ctx, texts)
    }
    return nil, nil
}

func (m *MockProvider) Dimensions() int {
    if m.DimensionsFunc != nil {
        return m.DimensionsFunc()
    }
    return 384
}

func (m *MockProvider) IsExternal() bool {
    if m.IsExternalFunc != nil {
        return m.IsExternalFunc()
    }
    return false
}

// ❌ WRONG - Partial mock (doesn't compile as EmbeddingProvider)
type MockProvider struct {
    EmbedFunc func(ctx context.Context, text string) ([]float32, error)
    // Missing: EmbedBatch, Dimensions, IsExternal
}
```

## Integration with Other Skills

**Before marking AI package work complete:**

Use `contextd:completing-major-task` with evidence:
- Build: `go build ./pkg/embedding/...`
- Tests: `go test -v ./pkg/embedding/...` (show coverage ≥80%)
- Security: Verify no PII sent to external APIs
- Functionality: Show embedding dimensions, normalized vectors, hybrid search results

**Before creating PR:**

Use `contextd:code-review` - reviewer validates:
- Provider abstraction exists
- Normalization implemented correctly
- Hybrid search (not pure semantic)
- Privacy protection for external APIs
- Timeouts and retries present

## Red Flags - STOP and Fix

If you catch yourself saying:

- "We're only using OpenAI, abstraction violates YAGNI"
- "OpenAI embeddings are already normalized"
- "Pure semantic is fast enough"
- "Error messages don't contain sensitive data"
- "Demo doesn't need context/timeouts"
- "We can refactor the interface later"
- "Dimension checking is redundant"
- "This Stack Overflow normalization works"

**All of these are WRONG. Follow the patterns in this skill.**

## The Bottom Line

AI packages in contextd require:

1. **Abstraction**: EmbeddingProvider interface (TEI + OpenAI)
2. **Normalization**: L2 normalization with unit vector test
3. **Hybrid search**: 70/30 semantic/keyword split
4. **Privacy**: Sanitize before external APIs
5. **Production patterns**: Context, timeouts, retries, error wrapping

**No shortcuts. These are architectural requirements, not suggestions.**
