---
name: dedupe-issue-local
specializes: dedupe-issue
description: Repo-specific dedupe guidance for contextd. Only the categories declared overridable by the core dedupe-issue skill may be specialized here.
---

# Repo-specific dedupe guidance for `contextd`

This file is a companion to the core `dedupe-issue` skill. It does not
redefine the duplicate-detection algorithm, the similarity thresholds,
or the output contract. It only specializes the override categories the
core skill marks as overridable.

## Known-duplicate clusters

The `update-dedupe` loop will populate this section as recurring duplicates emerge. The following clusters are seeded from existing repo history and `CLAUDE.md` patterns:

- **Qdrant filter syntax errors** — historical issues #1, #3, #4 collapsed into a single root cause (Qdrant payload filter shape). Future reports with phrases like "Qdrant filter syntax error", "collection not found after index", or "filter must contain at least one condition" should be checked against these closed issues first.
- **ONNX / embedding download failures** — multiple variants of "embedding provider failed", "empty or nil input texts", or "model file missing" usually trace back to the same auto-download path (`docs/spec/onnx-auto-download/`). Surface the canonical reproduction issue rather than each new instance.

## Repo-specific surface terms

When computing title/body similarity, treat these phrases as equivalent surface forms of the same concept:

- `chromem` / `chromem-go` / `embedded vectorstore`
- `Qdrant` / `qdrant-client` / `external vectorstore`
- `FastEmbed` / `ONNX embeddings` / `local embeddings`
- `ReasoningBank` / `memory layer` / `cross-session memory`
- `MCP tool` / `MCP handler` / `tool call`
- `payload isolation` / `tenant filtering` / `multi-tenant`
- `gitleaks scrubbing` / `secret scrubbing` / `secret redaction`
