# REFACTOR Phase: New Rationalizations Found

## Meta-Testing: Finding Loopholes

**Test Method**: Look for ways to technically comply while violating spirit

### New Rationalization 1: "Interface Without Abstraction"

**Scenario**: Agent creates interface but only one implementation exists, never used polymorphically

```go
// Interface defined (✓)
type EmbeddingProvider interface { ... }

// Only OpenAI implemented
type OpenAIProvider struct { ... }

// But services hardcode OpenAI:
func NewCheckpointService() *Service {
    provider := embedding.NewOpenAIProvider(...) // Hardcoded!
    return &Service{embedder: provider}
}
```

**Letter compliance**: Interface exists
**Spirit violation**: Interface never used for abstraction

**Fix Required**: Add to skill:
```markdown
## Interface Alone Is Not Enough

**REQUIRED**: Services MUST accept interface, NOT concrete type:

✅ GOOD:
func NewService(embedder EmbeddingProvider) *Service

❌ WRONG:
func NewService() *Service {
    embedder := NewOpenAIProvider(...) // Hardcoded
}
```

---

### New Rationalization 2: "Normalize Once at Import"

**Scenario**: Agent normalizes during initial indexing but not during search query embedding

```go
// Normalizes document embeddings at index time
func (s *IndexService) Index(doc Document) error {
    vector, _ := s.embedder.Embed(doc.Text)
    normalized := normalize(vector)
    s.store.Upsert(normalized)
}

// But query embedding NOT normalized
func (s *SearchService) Search(query string) error {
    vector, _ := s.embedder.Embed(query)
    return s.store.Search(vector) // Missing normalization!
}
```

**Letter compliance**: normalize() called somewhere
**Spirit violation**: Inconsistent normalization breaks cosine similarity

**Fix Required**: Add to skill:
```markdown
## Normalize ALL Embeddings

**REQUIRED**: Normalize BOTH document embeddings AND query embeddings

If query is not normalized but documents are, cosine similarity breaks.
```

---

### New Rationalization 3: "Hybrid Search with 99/1 Split"

**Scenario**: Agent implements "hybrid" search but uses 99% semantic, 1% keyword

```go
// Technically hybrid...
semanticLimit := int(float64(limit) * 0.99)
keywordLimit := int(float64(limit) * 0.01)
```

**Letter compliance**: Has both semantic and keyword
**Spirit violation**: 99/1 is effectively pure semantic

**Fix Required**: Strengthen skill requirement:
```markdown
**REQUIRED**: 70% semantic, 30% keyword (EXACTLY)

Not 90/10, not 50/50. 70/30 is empirically validated ratio.
```

---

### New Rationalization 4: "Sanitize Only Email"

**Scenario**: Agent sanitizes emails but not paths, API keys, usernames

```go
func sanitizeError(msg string) string {
    // Only redacts emails
    return regexp.MustCompile(`[email pattern]`).ReplaceAllString(msg, "[EMAIL]")
    // Missing: paths, API keys, IP addresses, usernames
}
```

**Letter compliance**: Has sanitization
**Spirit violation**: Incomplete sanitization leaks other PII

**Fix Required**: Strengthen skill requirement:
```markdown
**REQUIRED Sanitization Patterns**:
- Emails: `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}`
- Paths: `/[a-zA-Z0-9_\-./]+`
- API keys: `\b(sk|pk)[-_][a-zA-Z0-9]{20,}\b`
- IP addresses: `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`
- Usernames: Common patterns like "user=" or "@username"

ALL must be implemented.
```

---

### New Rationalization 5: "Tests Pass Without Unit Vector Check"

**Scenario**: Agent writes normalization test but doesn't verify magnitude = 1.0

```go
func TestNormalize(t *testing.T) {
    vector := []float32{3.0, 4.0}
    normalized := normalize(vector)

    // Just checks it returns something
    if len(normalized) != 2 {
        t.Error("wrong length")
    }
    // Missing: magnitude check!
}
```

**Letter compliance**: Test exists for normalize()
**Spirit violation**: Test doesn't validate correctness

**Fix Required**: Strengthen skill requirement:
```markdown
**REQUIRED Test**:

func TestNormalize_UnitVector(t *testing.T) {
    // Must calculate magnitude and assert ≈ 1.0
    magnitude := math.Sqrt(sumSquares)
    if math.Abs(magnitude - 1.0) > 0.0001 {
        t.Errorf("Magnitude = %f, want 1.0", magnitude)
    }
}

Tests WITHOUT magnitude validation do not satisfy requirement.
```

---

### New Rationalization 6: "Mock Provider Doesn't Implement Interface"

**Scenario**: Agent creates mock for testing but doesn't implement full interface

```go
type MockProvider struct {
    EmbedFunc func(ctx context.Context, text string) ([]float32, error)
    // Missing: EmbedBatch, Dimensions, IsExternal
}

func (m *MockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    return m.EmbedFunc(ctx, text)
}
// Doesn't compile as EmbeddingProvider!
```

**Letter compliance**: Has mock
**Spirit violation**: Mock doesn't satisfy interface

**Fix Required**: Add to skill:
```markdown
## Mock MUST Implement Full Interface

❌ WRONG: Partial mock
type MockProvider struct {
    EmbedFunc func(...) // Only Embed
}

✅ GOOD: Full interface implementation
type MockProvider struct {
    EmbedFunc      func(ctx context.Context, text string) ([]float32, error)
    EmbedBatchFunc func(ctx context.Context, texts []string) ([][]float32, error)
    DimensionsFunc func() int
    IsExternalFunc func() bool
}

func (m *MockProvider) Embed(...) { return m.EmbedFunc(...) }
func (m *MockProvider) EmbedBatch(...) { return m.EmbedBatchFunc(...) }
func (m *MockProvider) Dimensions() int { return m.DimensionsFunc() }
func (m *MockProvider) IsExternal() bool { return m.IsExternalFunc() }
```

---

## Summary of Loopholes Found

| # | Loophole | How to Exploit | Fix |
|---|----------|----------------|-----|
| 1 | Interface without abstraction | Define interface but hardcode concrete type | Require services accept interface |
| 2 | Inconsistent normalization | Normalize documents but not queries | Require normalize ALL embeddings |
| 3 | Fake hybrid search | 99/1 semantic/keyword split | Require exactly 70/30 |
| 4 | Incomplete sanitization | Sanitize email only, skip paths/keys | Require ALL patterns |
| 5 | Weak normalization test | Test exists but no magnitude check | Require magnitude ≈ 1.0 assertion |
| 6 | Partial mock | Mock doesn't implement full interface | Require all interface methods |

**Next**: Update skill with these counters
