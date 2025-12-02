// Package embeddings provides embedding generation via multiple providers.
package embeddings

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	fastembed "github.com/anush008/fastembed-go"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// FastEmbedConfig holds configuration for the FastEmbed provider.
type FastEmbedConfig struct {
	// Model is the embedding model to use.
	// Supported: BAAI/bge-small-en-v1.5 (default), BAAI/bge-base-en-v1.5,
	// sentence-transformers/all-MiniLM-L6-v2, etc.
	Model string

	// CacheDir is the directory to cache model files.
	// Defaults to ~/.cache/contextd/models
	CacheDir string

	// MaxLength is the maximum input sequence length.
	// Defaults to 512.
	MaxLength int
}

// FastEmbedProvider provides embedding generation using local ONNX models.
type FastEmbedProvider struct {
	model     *fastembed.FlagEmbedding
	modelName string
	dimension int
	mu        sync.RWMutex
}

// modelMapping maps friendly model names to fastembed model constants.
var modelMapping = map[string]fastembed.EmbeddingModel{
	"BAAI/bge-small-en-v1.5":                fastembed.BGESmallENV15,
	"BAAI/bge-small-en":                     fastembed.BGESmallEN,
	"BAAI/bge-base-en-v1.5":                 fastembed.BGEBaseENV15,
	"BAAI/bge-base-en":                      fastembed.BGEBaseEN,
	"BAAI/bge-small-zh-v1.5":                fastembed.BGESmallZH,
	"sentence-transformers/all-MiniLM-L6-v2": fastembed.AllMiniLML6V2,
	// Also accept the fastembed model names directly
	"fast-bge-small-en-v1.5":  fastembed.BGESmallENV15,
	"fast-bge-small-en":       fastembed.BGESmallEN,
	"fast-bge-base-en-v1.5":   fastembed.BGEBaseENV15,
	"fast-bge-base-en":        fastembed.BGEBaseEN,
	"fast-bge-small-zh-v1.5":  fastembed.BGESmallZH,
	"fast-all-MiniLM-L6-v2":   fastembed.AllMiniLML6V2,
}

// modelDimensions maps fastembed models to their embedding dimensions.
var modelDimensions = map[fastembed.EmbeddingModel]int{
	fastembed.BGESmallENV15: 384,
	fastembed.BGESmallEN:    384,
	fastembed.BGEBaseENV15:  768,
	fastembed.BGEBaseEN:     768,
	fastembed.BGESmallZH:    512,
	fastembed.AllMiniLML6V2: 384,
}

// NewFastEmbedProvider creates a new FastEmbed embedding provider.
func NewFastEmbedProvider(cfg FastEmbedConfig) (*FastEmbedProvider, error) {
	// Map model name to fastembed constant
	model, ok := modelMapping[cfg.Model]
	if !ok {
		// Check if it's a direct fastembed model name
		model = fastembed.EmbeddingModel(cfg.Model)
		// Validate it's a known model
		if _, known := modelDimensions[model]; !known {
			return nil, fmt.Errorf("%w: unsupported model %q (supported: BAAI/bge-small-en-v1.5, BAAI/bge-base-en-v1.5, sentence-transformers/all-MiniLM-L6-v2)", ErrInvalidConfig, cfg.Model)
		}
	}

	// Get dimension for this model
	dimension := modelDimensions[model]

	// Set defaults
	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join(".", "local_cache")
	}

	maxLength := cfg.MaxLength
	if maxLength == 0 {
		maxLength = 512
	}

	// Disable progress bar for server use
	showProgress := false

	opts := &fastembed.InitOptions{
		Model:                model,
		CacheDir:             cacheDir,
		MaxLength:            maxLength,
		ShowDownloadProgress: &showProgress,
	}

	flagEmbed, err := fastembed.NewFlagEmbedding(opts)
	if err != nil {
		return nil, fmt.Errorf("initializing FastEmbed: %w", err)
	}

	return &FastEmbedProvider{
		model:     flagEmbed,
		modelName: cfg.Model,
		dimension: dimension,
	}, nil
}

// Embedder returns an Embedder interface implementation.
func (p *FastEmbedProvider) Embedder() vectorstore.Embedder {
	return p
}

// EmbedDocuments generates embeddings for multiple texts.
// Uses "passage: " prefix for document embeddings as recommended by BGE models.
func (p *FastEmbedProvider) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("%w: texts cannot be empty", ErrEmptyInput)
	}

	// Check context before proceeding
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Use PassageEmbed which adds "passage: " prefix for documents
	embeddings, err := p.model.PassageEmbed(texts, 256)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}

	return embeddings, nil
}

// EmbedQuery generates an embedding for a single query.
// Uses "query: " prefix for query embeddings as recommended by BGE models.
func (p *FastEmbedProvider) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("%w: text cannot be empty", ErrEmptyInput)
	}

	// Check context before proceeding
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// QueryEmbed adds "query: " prefix automatically
	embedding, err := p.model.QueryEmbed(text)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}

	return embedding, nil
}

// Dimension returns the embedding dimension for the current model.
func (p *FastEmbedProvider) Dimension() int {
	return p.dimension
}

// Close releases resources held by the FastEmbed provider.
func (p *FastEmbedProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.model != nil {
		return p.model.Destroy()
	}
	return nil
}
