# Semantic Similarity Testing - Embedding Options

The semantic similarity integration tests support two modes:

## Default Mode: Fake Semantic Embedder (Fast)

By default, tests use a deterministic fake embedder for fast, reproducible testing:

```bash
go test -v ./test/integration/framework -run SemanticSimilarity
```

This mode:
- ✅ Runs in CI/CD without external dependencies
- ✅ Fast execution (no model downloads)
- ✅ Deterministic results
- ✅ Works without ONNX runtime

## Optional Mode: Real FastEmbed Embeddings

Set `USE_REAL_EMBEDDINGS=1` to test with real FastEmbed models:

```bash
USE_REAL_EMBEDDINGS=1 go test -v ./test/integration/framework -run SemanticSimilarity
```

This mode:
- Requires ONNX runtime (`/usr/lib/libonnxruntime.so` or `ONNX_PATH` env var)
- Downloads embedding models on first run
- Provides more realistic semantic similarity testing
- Skipped automatically in short mode: `go test -short`

## Implementation

The `createEmbedder()` helper function in `semantic_similarity_test.go`:
1. Checks `USE_REAL_EMBEDDINGS` environment variable
2. Returns fake embedder by default
3. Returns real FastEmbed provider when enabled
4. Handles cleanup properly via `embedderCloser` wrapper

This pattern follows `internal/embeddings/fastembed_test.go`.
