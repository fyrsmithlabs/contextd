// Package embeddings provides embedding generation via multiple providers.
package embeddings

import (
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// Provider is the interface for embedding providers.
type Provider interface {
	vectorstore.Embedder
	// Dimension returns the embedding dimension for the current model.
	Dimension() int
	// Close releases resources held by the provider.
	Close() error
}

// ProviderConfig holds configuration for creating an embedding provider.
type ProviderConfig struct {
	// Provider is the provider type: "fastembed" or "tei"
	Provider string
	// Model is the embedding model name
	Model string
	// BaseURL is the TEI URL (only used for TEI provider)
	BaseURL string
	// CacheDir is the model cache directory (only used for FastEmbed)
	CacheDir string
	// ShowProgress enables progress bars for downloads
	ShowProgress bool
}

// detectDimensionFromModel returns the embedding dimension for a model name.
// Falls back to 384 if model is unknown.
func detectDimensionFromModel(model string) int {
	// Check FastEmbed model mapping first
	if dim, ok := fastEmbedModelDimension(model); ok {
		return dim
	}
	// Common model dimension patterns
	switch {
	case contains(model, "base"):
		return 768
	case contains(model, "large"):
		return 1024
	case contains(model, "small"), contains(model, "mini"):
		return 384
	default:
		return 384 // Safe default for bge-small
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// NewProvider creates an embedding provider based on the configuration.
func NewProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "fastembed", "":
		return NewFastEmbedProvider(FastEmbedConfig{
			Model:        cfg.Model,
			CacheDir:     cfg.CacheDir,
			ShowProgress: cfg.ShowProgress,
		})
	case "tei":
		svc, err := NewService(Config{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
		if err != nil {
			return nil, err
		}
		dim := detectDimensionFromModel(cfg.Model)
		return &teiProvider{Service: svc, dimension: dim}, nil
	default:
		return nil, fmt.Errorf("%w: unknown provider %q", ErrInvalidConfig, cfg.Provider)
	}
}

// teiProvider wraps Service to implement Provider interface.
type teiProvider struct {
	*Service
	dimension int
}

// Dimension returns the embedding dimension based on the configured model.
func (t *teiProvider) Dimension() int {
	return t.dimension
}

// Close is a no-op for TEI since it uses HTTP.
func (t *teiProvider) Close() error {
	return nil
}
