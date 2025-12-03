//go:build !cgo

// Package embeddings provides embedding generation via multiple providers.
package embeddings

import (
	"context"
	"errors"
)

// ErrFastEmbedNotAvailable is returned when FastEmbed is not available (requires CGO).
var ErrFastEmbedNotAvailable = errors.New("fastembed: not available (binary built without CGO support, use TEI provider instead)")

// FastEmbedConfig holds configuration for the FastEmbed provider.
type FastEmbedConfig struct {
	Model     string
	CacheDir  string
	MaxLength int
}

// FastEmbedProvider provides embedding generation using local ONNX models.
// This is a stub for non-CGO builds.
type FastEmbedProvider struct{}

// NewFastEmbedProvider returns an error when CGO is not available.
func NewFastEmbedProvider(_ FastEmbedConfig) (*FastEmbedProvider, error) {
	return nil, ErrFastEmbedNotAvailable
}

// EmbedDocuments returns an error when CGO is not available.
func (p *FastEmbedProvider) EmbedDocuments(_ context.Context, _ []string) ([][]float32, error) {
	return nil, ErrFastEmbedNotAvailable
}

// EmbedQuery returns an error when CGO is not available.
func (p *FastEmbedProvider) EmbedQuery(_ context.Context, _ string) ([]float32, error) {
	return nil, ErrFastEmbedNotAvailable
}

// Dimension returns 0 when CGO is not available.
func (p *FastEmbedProvider) Dimension() int {
	return 0
}

// Close is a no-op when CGO is not available.
func (p *FastEmbedProvider) Close() error {
	return nil
}

// fastEmbedModelDimension returns dimensions for known models.
// This is a fallback when CGO is not available.
func fastEmbedModelDimension(model string) (int, bool) {
	dims := map[string]int{
		"BAAI/bge-small-en-v1.5":                384,
		"BAAI/bge-small-en":                     384,
		"BAAI/bge-base-en-v1.5":                 768,
		"BAAI/bge-base-en":                      768,
		"BAAI/bge-small-zh-v1.5":                512,
		"sentence-transformers/all-MiniLM-L6-v2": 384,
		"fast-bge-small-en-v1.5":                384,
		"fast-bge-small-en":                     384,
		"fast-bge-base-en-v1.5":                 768,
		"fast-bge-base-en":                      768,
		"fast-bge-small-zh-v1.5":                512,
		"fast-all-MiniLM-L6-v2":                 384,
	}
	dim, ok := dims[model]
	return dim, ok
}
