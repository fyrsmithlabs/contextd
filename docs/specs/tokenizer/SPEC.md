# Tokenizer Interface and BPE Implementation Specification

## Overview

This specification defines a tokenizer interface for the embedding package to replace the current token estimation approximation (~4 chars per token) with accurate Byte-Pair Encoding (BPE) tokenization. This will improve token count accuracy for chunking operations and cost estimation.

## Problem Statement

**Current State:**
- Token counting uses rough approximation: `~4 characters per token`
- Inaccurate for chunking long texts (especially for TEI with 512 token limit)
- Cost estimation is approximate
- No model-specific tokenization strategies

**Issues:**
1. Over/under-chunking due to inaccurate token counts
2. Potential API failures when chunks exceed actual token limits
3. Inaccurate cost estimates for OpenAI API usage
4. Cannot accurately predict when texts need chunking

## Requirements

### Functional Requirements

1. **Tokenizer Interface**
   - Define Go interface following best practices
   - Support token counting (Encode operation)
   - Support text generation from tokens (Decode operation)
   - Model-specific implementations

2. **BPE Implementation**
   - Proper Byte-Pair Encoding algorithm
   - Support for vocabulary and merge rules
   - Compatible with OpenAI's BPE (GPT tokenizer)
   - Compatible with TEI models (BERT-style tokenization)

3. **Integration with Embedding Service**
   - Replace `estimateTokenCount` with accurate tokenization
   - Update `chunkText` to use real token counts
   - Maintain backward compatibility during transition

4. **Performance**
   - Token counting: <1ms for typical texts (<10K chars)
   - Minimal memory overhead (<10MB for vocabulary)
   - Thread-safe for concurrent use

### Non-Functional Requirements

1. **Testing**
   - ≥80% test coverage (TDD approach)
   - Unit tests for tokenizer interface
   - Integration tests with embedding service
   - Performance benchmarks

2. **Documentation**
   - Package-level documentation
   - Example usage
   - Migration guide from estimation to BPE

3. **Security**
   - No external API calls for tokenization (local only)
   - Safe handling of arbitrary text inputs
   - No exposure of sensitive data in tokens

## Architecture

### Package Structure

```
pkg/embedding/
├── tokenizer/
│   ├── tokenizer.go          # Interface definition
│   ├── bpe.go                # BPE implementation
│   ├── bpe_test.go           # BPE tests
│   ├── openai.go             # OpenAI BPE tokenizer
│   ├── openai_test.go
│   ├── tei.go                # TEI BERT tokenizer
│   ├── tei_test.go
│   ├── vocab.go              # Vocabulary management
│   └── CLAUDE.md             # Package documentation
├── embedding.go              # Updated to use tokenizer
└── embedding_test.go         # Updated tests
```

### Interface Design

```go
package tokenizer

// Tokenizer provides text tokenization capabilities
type Tokenizer interface {
    // Encode converts text to token IDs
    Encode(text string) ([]int, error)

    // Decode converts token IDs back to text
    Decode(tokens []int) (string, error)

    // CountTokens returns the number of tokens for text
    // This is optimized for the common use case of just counting
    CountTokens(text string) (int, error)

    // Vocabulary returns the tokenizer's vocabulary size
    Vocabulary() int

    // MaxTokens returns the maximum tokens this tokenizer supports
    MaxTokens() int

    // Name returns the tokenizer identifier
    Name() string
}

// Config holds tokenizer configuration
type Config struct {
    Type      TokenizerType // openai or tei
    Model     string        // Model identifier
    VocabFile string        // Optional vocabulary file path
    MergeFile string        // Optional merge rules file path
    MaxTokens int           // Maximum tokens (512 for TEI, 8191 for OpenAI)
}

// TokenizerType identifies the tokenizer implementation
type TokenizerType string

const (
    TokenizerTypeOpenAI TokenizerType = "openai"
    TokenizerTypeTEI    TokenizerType = "tei"
)

// New creates a tokenizer based on configuration
func New(cfg *Config) (Tokenizer, error)

// OpenAITokenizer implements BPE for OpenAI models
type OpenAITokenizer struct {
    vocab      map[string]int
    merges     []Merge
    maxTokens  int
    model      string
}

// TEITokenizer implements WordPiece tokenization for TEI models
type TEITokenizer struct {
    vocab      map[string]int
    maxTokens  int
    model      string
}

// Merge represents a BPE merge operation
type Merge struct {
    Left  string
    Right string
    Rank  int
}
```

## Implementation Plan

### Phase 1: Interface and Test Infrastructure (TDD Start)

**Tasks:**
1. Create `pkg/embedding/tokenizer/tokenizer.go` with interface definitions
2. Create `pkg/embedding/tokenizer/tokenizer_test.go` with comprehensive test cases
3. Define test fixtures (sample texts, expected token counts)
4. Create mock tokenizer for testing

**Test Cases:**
- Empty text handling
- Single word tokenization
- Multi-word tokenization
- Special characters and punctuation
- Unicode and emoji handling
- Large text handling (>10K chars)
- Thread safety

**Acceptance Criteria:**
- All tests written (failing initially - RED)
- Interface clearly defined
- Test fixtures prepared

### Phase 2: BPE Core Implementation (TDD Green)

**Tasks:**
1. Implement basic BPE algorithm
2. Implement vocabulary loading
3. Implement merge rules loading
4. Pass basic tokenization tests

**BPE Algorithm Steps:**
```
1. Convert text to bytes
2. Apply UTF-8 byte encoding
3. Split into initial tokens (characters or bytes)
4. Apply merge rules iteratively:
   - Find most frequent adjacent pair
   - Merge pair according to rules
   - Repeat until no more merges
5. Convert tokens to IDs using vocabulary
```

**Acceptance Criteria:**
- Basic BPE algorithm working
- Tests passing (GREEN)
- Can tokenize simple English text

### Phase 3: OpenAI Tokenizer (Specific Implementation)

**Tasks:**
1. Implement OpenAI BPE tokenizer
2. Load GPT tokenizer vocabulary
3. Load GPT merge rules
4. Test with GPT-3 examples

**Resources:**
- Use `github.com/tiktoken-go/tokenizer` for reference
- Or implement from OpenAI's BPE specification

**Test Cases:**
- Match OpenAI's token counts for known texts
- Handle OpenAI's special tokens
- Verify 8191 token limit

**Acceptance Criteria:**
- OpenAI tokenizer passes all tests
- Token counts match OpenAI API
- Integration tests with embedding service

### Phase 4: TEI Tokenizer (WordPiece)

**Tasks:**
1. Implement WordPiece tokenization for BERT models
2. Load TEI vocabulary
3. Test with TEI models (bge-small, bge-large)

**Algorithm:**
```
WordPiece (different from BPE):
1. Split text into words
2. For each word:
   - Try to match longest subword in vocabulary
   - If not found, split into smaller subwords
   - Use "##" prefix for continuation tokens
3. Convert subwords to IDs
```

**Test Cases:**
- Match TEI's token counts
- Handle TEI's special tokens
- Verify 512 token limit

**Acceptance Criteria:**
- TEI tokenizer passes all tests
- Token counts match TEI service
- Integration tests with embedding service

### Phase 5: Integration with Embedding Service

**Tasks:**
1. Add tokenizer to embedding service configuration
2. Update `estimateTokenCount` to use tokenizer
3. Update `chunkText` to use accurate counts
4. Add tokenizer selection based on provider

**Code Changes:**
```go
// In embedding.go Service struct
type Service struct {
    // ... existing fields ...
    tokenizer tokenizer.Tokenizer
}

// In NewService
func NewService(cfg *Config, meters *metrics.Meters) (*Service, error) {
    // Create tokenizer based on provider
    tokenizerCfg := &tokenizer.Config{
        Type:      getTokenizerType(cfg.Providers[0].Type),
        Model:     cfg.Model,
        MaxTokens: cfg.Providers[0].MaxTokens,
    }
    tok, err := tokenizer.New(tokenizerCfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create tokenizer: %w", err)
    }

    return &Service{
        // ... existing fields ...
        tokenizer: tok,
    }, nil
}

// Replace estimateTokenCount
func (s *Service) countTokens(text string) (int, error) {
    return s.tokenizer.CountTokens(text)
}
```

**Acceptance Criteria:**
- Embedding service uses tokenizer
- All existing tests still pass
- Token counts are accurate

### Phase 6: Performance Optimization and Caching

**Tasks:**
1. Add token count caching for repeated texts
2. Optimize vocabulary lookup (trie or hash map)
3. Add performance benchmarks
4. Profile and optimize hot paths

**Performance Targets:**
- Token counting: <1ms for 1K chars
- Token counting: <10ms for 10K chars
- Memory: <10MB for vocabulary

**Acceptance Criteria:**
- Performance benchmarks meet targets
- No memory leaks
- Thread-safe implementation

## Testing Strategy

### Unit Tests

**Coverage Targets:**
- `tokenizer.go`: 100% (interface definitions)
- `bpe.go`: ≥90% (core algorithm)
- `openai.go`: ≥85% (OpenAI implementation)
- `tei.go`: ≥85% (TEI implementation)
- Overall: ≥80%

**Test Categories:**
1. **Interface Tests**: Mock implementations
2. **Algorithm Tests**: BPE/WordPiece correctness
3. **Integration Tests**: With embedding service
4. **Performance Tests**: Benchmarks
5. **Regression Tests**: Known issues

### Integration Tests

```go
func TestTokenizerIntegration(t *testing.T) {
    // Test with actual embedding service
    svc := createTestEmbeddingService()

    text := "This is a test of the tokenizer integration"

    // Count tokens
    count, err := svc.tokenizer.CountTokens(text)
    require.NoError(t, err)

    // Should be accurate (not ~4 chars per token)
    assert.Greater(t, count, 0)
    assert.Less(t, count, len(text)/2)
}
```

### Performance Tests

```go
func BenchmarkTokenization(b *testing.B) {
    tok := createTestTokenizer()
    text := loadTestText() // 1K, 10K, 100K chars

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = tok.CountTokens(text)
    }
}
```

## Migration Strategy

### Phase 1: Parallel Mode (Default: Estimation)

- Add tokenizer but keep estimation as default
- Add flag to enable tokenizer: `EMBEDDING_USE_TOKENIZER=true`
- Log differences between estimation and tokenization
- Monitor in production

### Phase 2: Tokenizer Default (Fallback: Estimation)

- Make tokenizer default
- Keep estimation as fallback if tokenizer fails
- Monitor error rates

### Phase 3: Tokenizer Only

- Remove estimation code
- Tokenizer is only method

## Dependencies

### External Libraries

**Option 1: Implement from scratch (Recommended)**
- Full control over implementation
- No external dependencies
- Better for learning and maintenance

**Option 2: Use existing library**
- `github.com/tiktoken-go/tokenizer` - Go port of tiktoken
- Pros: Tested, compatible with OpenAI
- Cons: External dependency, may not support TEI

**Decision:** Start with Option 1 (implement from scratch) for learning and control. Can add Option 2 later if needed.

## Security Considerations

1. **Input Validation**
   - Validate text length before tokenization
   - Handle malformed UTF-8 gracefully
   - Prevent excessive memory allocation

2. **Resource Limits**
   - Set maximum text length (e.g., 1MB)
   - Limit vocabulary size in memory
   - Timeout for tokenization operations

3. **No External Calls**
   - All tokenization happens locally
   - No API calls to external services
   - Vocabulary loaded from local files

## Performance Notes

**Expected Performance:**
- Token counting: O(n) where n = text length
- Vocabulary lookup: O(1) with hash map
- Memory: O(v) where v = vocabulary size

**Optimizations:**
- Cache token counts for repeated texts
- Use efficient data structures (tries for vocabulary)
- Pre-compile merge rules
- Lazy load vocabularies

## Related Documentation

- BPE Algorithm: [Neural Machine Translation of Rare Words with Subword Units](https://arxiv.org/abs/1508.07909)
- OpenAI Tokenizer: [tiktoken](https://github.com/openai/tiktoken)
- BERT Tokenization: [BERT Paper](https://arxiv.org/abs/1810.04805)
- WordPiece: [Google's Neural Machine Translation System](https://arxiv.org/abs/1609.08144)

## Success Criteria

1. **Accuracy**
   - Token counts match OpenAI API (±1 token)
   - Token counts match TEI service (±1 token)
   - No chunking failures due to incorrect counts

2. **Performance**
   - Token counting <1ms for typical texts
   - No performance regression in embedding service
   - Memory usage <10MB additional

3. **Testing**
   - ≥80% test coverage
   - All tests passing
   - Performance benchmarks documented

4. **Integration**
   - Seamless integration with embedding service
   - No breaking changes to public API
   - Backward compatible migration path

## Future Enhancements

1. **Support for More Models**
   - Llama tokenizer
   - Mistral tokenizer
   - Custom vocabulary support

2. **Advanced Features**
   - Token-aware text splitting (preserve sentence boundaries)
   - Token budgeting for LLM prompts
   - Token visualization tools

3. **Performance**
   - Parallel tokenization for batches
   - GPU acceleration for large texts
   - Streaming tokenization for very large texts

## Appendix

### Sample BPE Merge Rules

```
# Example merge rules (simplified)
# Format: left right rank
e s 1
es t 2
est space 3
# ... thousands more rules
```

### Sample Vocabulary

```
# Example vocabulary (simplified)
{
  "a": 0,
  "b": 1,
  "c": 2,
  "##ing": 3,
  "##ed": 4,
  # ... thousands more entries
}
```

### Example Usage

```go
// Create tokenizer
cfg := &tokenizer.Config{
    Type:      tokenizer.TokenizerTypeOpenAI,
    Model:     "text-embedding-3-small",
    MaxTokens: 8191,
}
tok, err := tokenizer.New(cfg)
if err != nil {
    return err
}

// Count tokens
text := "This is a sample text for tokenization"
count, err := tok.CountTokens(text)
fmt.Printf("Token count: %d\n", count) // Accurate count, not estimation

// Encode text
tokens, err := tok.Encode(text)
fmt.Printf("Tokens: %v\n", tokens) // [1212, 318, 257, ...]

// Decode tokens
decoded, err := tok.Decode(tokens)
fmt.Printf("Decoded: %s\n", decoded) // "This is a sample text for tokenization"
```
