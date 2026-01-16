// Package embeddings provides embedding generation via multiple providers.
//
// Supports FastEmbed (local ONNX) and TEI (external service) providers.
// Factory pattern enables provider selection at runtime with automatic
// dimension detection for common models.
//
// See CLAUDE.md for provider configuration and model selection.
package embeddings
